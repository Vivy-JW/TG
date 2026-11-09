package debug

// 源码本地镜像的单测。
//
// 镜像是给 AI 的便利副本:调试过程中读过的文件(含程序停过的文件)在本地留一份,
// 好让它整读、本地搜,不必为每个文件再走一次 SSH。这里钉的是它不会出事的那几处 ——
// 路径拼装不能越界、环境名要能当目录名、清空要真的清干净。

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// 镜像路径由服务器路径拼成,`..` 必须被消掉 —— 否则能从镜像根逃出去写任意位置
func TestSrcMirrorPathNoEscape(t *testing.T) {
	got := srcMirrorPath(`D:\data`, "env", "/u1/../../../etc/passwd")
	if strings.Contains(got, "..") {
		t.Errorf("路径含 .. 可以越界: %s", got)
	}
	if !strings.HasPrefix(got, filepath.Join(`D:\data`, "srccache", "env")) {
		t.Errorf("没有落在镜像根下: %s", got)
	}
	// 保留原始目录层次,方便人肉对照服务器路径
	want := filepath.Join(`D:\data`, "srccache", "env", "u1", "t35prd", "com", "wss", "4gl", "wssp01131.4gl")
	if p := srcMirrorPath(`D:\data`, "env", "/u1/t35prd/com/wss/4gl/wssp01131.4gl"); p != want {
		t.Errorf("路径拼接错误\n 得到 %s\n 期望 %s", p, want)
	}
	// 没有数据目录(未配置)时不落镜像,而不是落到进程当前目录
	if p := srcMirrorPath("", "env", "/u1/a.4gl"); p != "" {
		t.Errorf("dataDir 为空应返回空串,得到 %s", p)
	}
	if p := srcMirrorPath(`D:\data`, "env", ""); p != "" {
		t.Errorf("服务器路径为空应返回空串,得到 %s", p)
	}
}

func TestPathSafeSeg(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"asf", "asf"},
		{"示例测试区", "示例测试区"},
		{"a/b", "a_b"},
		{"..", "_"},
		{".", "_"},
		{"", "_"},
		{"a:b*c?", "a_b_c_"},
	} {
		if got := pathSafeSeg(c.in); got != c.want {
			t.Errorf("pathSafeSeg(%q) = %q,期望 %q", c.in, got, c.want)
		}
	}
}

// 环境名可能是"示例测试区"这类中文,也可能为空(退化成 主机-区域)
func TestMirrorEnvSeg(t *testing.T) {
	if got := mirrorEnvSeg("示例测试区", "10.1.2.3", "36"); got != "示例测试区" {
		t.Errorf("中文环境名应保留,得到 %q", got)
	}
	if got := mirrorEnvSeg("", "10.1.2.3", "36"); got != "10.1.2.3-36" {
		t.Errorf("空环境名应退化为 主机-区域,得到 %q", got)
	}
	// 环境名里的路径分隔符不能变成目录层级
	if got := mirrorEnvSeg("a/b:c", "h", "36"); strings.ContainsAny(got, `/\:`) {
		t.Errorf("环境名里的分隔符应被替换,得到 %q", got)
	}
	if got := mirrorEnvSeg("..", "h", "36"); got != "_" {
		t.Errorf("环境名 .. 应被消掉,得到 %q", got)
	}
}

func TestWriteAndClearMirror(t *testing.T) {
	dir := t.TempDir()
	src := []byte("MAIN\n  DISPLAY \"x\"\nEND MAIN\n")
	p := writeMirror(dir, "env", "/u1/erp/asf/4gl/asf_x.4gl", src)
	if p == "" {
		t.Fatal("应写出镜像并返回路径")
	}
	got, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("读回失败: %v", err)
	}
	if string(got) != string(src) {
		t.Errorf("内容不一致:\n得到 %q\n期望 %q", got, src)
	}
	// 不留 .tmp 残骸(原子落位)
	if _, err := os.Stat(p + ".tmp"); !os.IsNotExist(err) {
		t.Error("落位后不应留下 .tmp")
	}

	clearMirror(dir)
	if _, err := os.Stat(filepath.Join(dir, srcMirrorDir)); !os.IsNotExist(err) {
		t.Error("clearMirror 应清掉整个镜像目录")
	}
	// 没建过镜像时清理不该报错(每轮调试开始都会调一次)
	clearMirror(dir)
	clearMirror("")
}

// 会话级清理只删**自己那棵子树**（缺陷 X / T5.1 的判据）。
//
// 从前的 `clearMirror(cfg.DataDir)` 删的是全局 `<DataDir>/srccache`,
// 于是并发启动时 A 的清理会把 B 刚拉的镜像一起删掉;execlog 同病。
// 这里把"只删自己那一段"钉死,并且**把修前的行为也断言出来**
// （`clearMirror` 就是那个全局入口,它还留着给路径拼装的测试用）。
func TestClearMirrorForSessionIsScoped(t *testing.T) {
	dir := t.TempDir()
	aOther := writeMirrorScoped(dir, mirrorScopeOf("envB", "run2"), "/u1/b.4gl", []byte("B"))
	aMine := writeMirrorScoped(dir, mirrorScopeOf("envA", "run1"), "/u1/a.4gl", []byte("A"))
	if aOther == "" || aMine == "" {
		t.Fatal("两条会话都该能落镜像")
	}
	// 命令输出的落盘与源码镜像同一棵树、同一个作用域
	logOther := writeExecLog(dir, mirrorScopeOf("envB", "run2"), "b.log", []byte("B 的输出"))
	if logOther == "" {
		t.Fatal("应写出命令输出副本")
	}

	clearMirrorForSession(dir, mirrorScopeOf("envA", "run1"))

	if _, err := os.Stat(aMine); !os.IsNotExist(err) {
		t.Error("本会话新一轮该清空自己的镜像（「只活一轮」）")
	}
	if _, err := os.Stat(aOther); err != nil {
		t.Errorf("别的会话的镜像必须留着（并发启动互删就是缺陷 X）：%v", err)
	}
	if _, err := os.Stat(logOther); err != nil {
		t.Errorf("别的会话的命令输出副本也必须留着：%v", err)
	}
	// 没建过镜像时清理不该报错（每轮启动都会调一次）
	clearMirrorForSession(dir, mirrorScopeOf("envC", "run9"))
	clearMirrorForSession(dir, "")
	clearMirrorForSession("", "run1")
}

// 同一条会话第二轮启动：作用域跨轮不变，所以清的正是自己上一轮那一份。
func TestClearMirrorForSessionSameScopeAcrossRounds(t *testing.T) {
	dir := t.TempDir()
	s := &Session{ID: "s1", cfg: cfgFor("h1", "36")}
	s.cfg.DataDir = dir
	s.inheritMirrorState()
	scope := s.mirrorScope()
	if p := writeMirrorScoped(dir, scope, "/u1/a.4gl", []byte("第一轮")); p == "" {
		t.Fatal("应写出镜像")
	}
	// 新一轮：复用宿主会话（作用域继承下来），清理照旧清空自己那一份
	s.inheritMirrorState()
	if got := s.mirrorScope(); got != scope {
		t.Fatalf("复用的宿主会话作用域必须跨轮不变（否则上一轮的副本留在盘上，「只活一轮」就丢了）：%q → %q", scope, got)
	}
	clearMirrorForSession(dir, s.mirrorScope())
	if _, err := os.Stat(srcMirrorScopeRoot(dir, scope)); !os.IsNotExist(err) {
		t.Error("同一条会话新一轮启动时镜像照旧清空")
	}
}

// 运行作用域按会话唯一（缺陷 X 的路径判据）：两条会话不能共用同一个目录段。
func TestRunScopeIsUniquePerSession(t *testing.T) {
	a := &Session{ID: "s1", cfg: cfgFor("h1", "36")}
	b := &Session{ID: "s1", cfg: cfgFor("h1", "36")} // 连可见 ID 都相同（同毫秒）
	a.inheritMirrorState()
	b.inheritMirrorState()
	if a.runScope == "" || b.runScope == "" {
		t.Fatal("会话建立时该分配运行作用域")
	}
	if a.runScope == b.runScope {
		t.Fatalf("两条会话的运行作用域撞了（%q）—— 它们会互相清掉对方的镜像", a.runScope)
	}
	if a.mirrorScope() == b.mirrorScope() {
		t.Fatalf("镜像路径必须按会话分开，实得同一个 %q", a.mirrorScope())
	}
}

// 会话可见 ID 的唯一性（并发启动的连带项）：同一个毫秒里起两条不能撞 ID。
//
// 判据不能是"连续调 5000 次都不重复" —— 时钟要走到那么多次调用才推进一毫秒，
// 而"没撞"只说明时间戳真的没重复。所以这里造的是**时钟没走**的状态:
// 把"上一次发出去的值"预置成**未来**（= 刚刚发生过时钟回拨），连取两个。
func TestNextSessionIDIsUnique(t *testing.T) {
	old := sessionSeq.Load()
	defer sessionSeq.Store(old)

	// 病根就在这里:候选时间戳与上一次相同（甚至更小）时，唯一性只能来自那个计数器。
	sessionSeq.Store(time.Now().UnixMilli() + 3600_000)
	a := nextSessionID()
	b := nextSessionID()
	if a == b {
		t.Fatalf("时钟没走（或回拨）时连起两条会话拿到了同一个 ID（%q）—— 后一条会把前一条从池子里顶掉\n"+
			"    （`m.sessions[id] = sess` 直接覆盖，前一条就此消失）", a)
	}
	if a == "" || b == "" {
		t.Fatal("ID 不该为空")
	}
	// 既有形状不许变（H2）:会话 ID 一直是 `s<毫秒>`，用户在 --json 里照着它抄。
	if !strings.HasPrefix(a, "s") || !strings.HasPrefix(b, "s") {
		t.Fatalf("会话 ID 的既有形状是 `s<数字>`，实得 %q / %q", a, b)
	}
	// 单调:后发的一定不比先发的小（否则"取最小/最近"这类用法会翻车）
	na, _ := strconv.ParseInt(strings.TrimPrefix(a, "s"), 10, 64)
	nb, _ := strconv.ParseInt(strings.TrimPrefix(b, "s"), 10, 64)
	if nb <= na {
		t.Fatalf("会话 ID 该单调递增: %q → %q", a, b)
	}
}

// 同一服务器路径重复写要覆盖,不能留 .tmp 堆积
func TestWriteMirrorOverwrites(t *testing.T) {
	dir := t.TempDir()
	writeMirror(dir, "env", "/u1/a.4gl", []byte("第一版"))
	p := writeMirror(dir, "env", "/u1/a.4gl", []byte("第二版"))
	got, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("读回失败: %v", err)
	}
	if string(got) != "第二版" {
		t.Errorf("重复写应覆盖,得到 %q", got)
	}
}

// mirrorStopFile 的两道闸:没有数据目录、或停站没有文件名,都不该动手。
// 前者防的是"凭空在进程当前目录建 srccache",后者是入口停站的真实情况
// (fgldb 在程序跑起来之前给不出位置)。
func TestMirrorStopFileGuards(t *testing.T) {
	s := waitTestSession() // cfgFor 不带 DataDir
	s.mirrorStopFile(&StopInfo{File: "asf_x.4gl"})
	if s.mirrored != nil {
		t.Error("没有数据目录时不该记录已镜像,更不该开工")
	}
	s.mirrorStopFile(nil)
	s.mirrorStopFile(&StopInfo{})

	// 有数据目录但没有文件名:同样不动作
	s2 := waitTestSession()
	s2.cfg.DataDir = t.TempDir()
	s2.mirrorStopFile(&StopInfo{Reason: "entry"})
	if s2.mirrored != nil {
		t.Error("停站没有文件名时不该记录已镜像")
	}
	if _, err := os.Stat(filepath.Join(s2.cfg.DataDir, srcMirrorDir)); !os.IsNotExist(err) {
		t.Error("不该建出镜像目录")
	}
}

// 清目录与清记录必须成对:复用的宿主会话会把"已镜像"记录带进下一轮,
// 不同步清掉,新一轮就再也落不下任何副本。
func TestResetMirrored(t *testing.T) {
	s := waitTestSession()
	s.mirrored = map[string]bool{"asf_x.4gl": true}
	s.resetMirrored()
	if len(s.mirrored) != 0 {
		t.Errorf("应清空已镜像记录,实际 %v", s.mirrored)
	}
}
