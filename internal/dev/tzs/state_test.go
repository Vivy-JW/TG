package tzs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestStateRoundTrip：写出去能原样读回来（含 Orphans 这个新增的可选字段）。
func TestStateRoundTrip(t *testing.T) {
	dir := t.TempDir()
	st := newState()
	st.SetEntry(`D:\ws\a`, &DaemonEntry{
		PID: 4242, Pipe: "tzs-cli-01a04826-11c46f9d", Workspace: `D:\ws\a`,
		Exe: `C:\tt\tzs-server.exe`, Log: LogPath(dir), Since: "2026-09-20T10:00:00+08:00",
		Orphans: []int{1234, 5678},
	})
	st.Last = &LastEntry{Workspace: `D:\ws\a`, Handle: "h1", OpenPath: `D:\pkg\x.tzs`}
	if err := SaveState(dir, st); err != nil {
		t.Fatal(err)
	}
	got, err := LoadState(dir)
	if err != nil {
		t.Fatal(err)
	}
	e := got.Entry(`D:\ws\a`)
	if e == nil {
		t.Fatal("读回来没有这条记录")
	}
	if e.PID != 4242 || e.Pipe != "tzs-cli-01a04826-11c46f9d" || len(e.Orphans) != 2 {
		t.Errorf("记录不对：%+v", e)
	}
	if got.Last == nil || got.Last.Handle != "h1" || got.Last.OpenPath != `D:\pkg\x.tzs` {
		t.Errorf("last 不对：%+v", got.Last)
	}
	if got.Version != StateVersion {
		t.Errorf("版本该是 %d，得 %d", StateVersion, got.Version)
	}
	// 原子写不留下临时文件。
	ents, _ := os.ReadDir(dir)
	for _, ent := range ents {
		if strings.Contains(ent.Name(), ".tmp") {
			t.Errorf("留下了临时文件：%s", ent.Name())
		}
	}
}

// TestStateMissingFileIsEmpty：状态文件不存在**不是错误**（第一次跑就是这样）。
func TestStateMissingFileIsEmpty(t *testing.T) {
	st, err := LoadState(t.TempDir())
	if err != nil {
		t.Fatalf("文件不存在不该报错：%v", err)
	}
	if len(st.Daemons) != 0 {
		t.Errorf("该是空状态，得 %+v", st)
	}
	if len(st.Keys()) != 0 {
		t.Errorf("Keys 该是空的")
	}
	if st.Entry("x") != nil {
		t.Errorf("空状态里查不到东西")
	}
}

// TestStateCorruptIsQuarantined：坏状态文件既**不**让命令失败，也**不**被静默吞掉。
//
// 这个文件只是记录，不是真相（真相是管道上有没有人应答）。让一个手改坏的、
// 或者写到一半断电的 JSON 把 `tt dev tzs` 全线挡住，是拿一条诊断信息否决整个工具；
// 而默默覆盖会让人永远不知道自己的记录丢过。
func TestStateCorruptIsQuarantined(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(StatePath(dir), []byte("{坏掉的 JSON"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadState(dir); err == nil {
		t.Fatal("严格版 LoadState 该报错")
	}
	st := loadState(dir) // 容错版：挪走 + 从空开始
	if len(st.Daemons) != 0 {
		t.Errorf("容错后该是空状态")
	}
	if _, err := os.Stat(StatePath(dir) + ".bad"); err != nil {
		t.Errorf("坏文件该被挪到 .bad 留证：%v", err)
	}
	if _, err := os.Stat(StatePath(dir)); err == nil {
		t.Errorf("坏文件不该留在原位（否则下一次读还是坏的）")
	}
}

// TestStateKeyIsLowercase：键按**小写的工作区原串**。
//
// Windows 路径大小写不敏感，而引擎自己也是先 ToLowerInvariant 再去 hash 管道名，
// 所以两种拼法必须是同一条记录 —— 不然会出现「同一个工作区两条守护进程记录」，
// 而 Reap/Stop 会各看到一半。
func TestStateKeyIsLowercase(t *testing.T) {
	st := newState()
	st.SetEntry(`D:\WS\AApp320`, &DaemonEntry{PID: 1})
	if st.Entry(`d:\ws\aapp320`) == nil {
		t.Fatal("大小写不同的拼法该命中同一条")
	}
	if st.Entry(`D:\WS\AApp320`) == nil {
		t.Fatal("原样拼法当然也该命中")
	}
	if len(st.Daemons) != 1 {
		t.Errorf("该只有一条记录，得 %d", len(st.Daemons))
	}
	// 尾斜杠**不**归一：引擎 hash 的是字面串，`D:\ws` 与 `D:\ws\` 是两个管道名、
	// 两个守护进程。归一会让第二个守护进程的记录覆盖第一个，Reap 从此看不到它。
	st.SetEntry(`D:\WS\AApp320\`, &DaemonEntry{PID: 2})
	if len(st.Daemons) != 2 {
		t.Errorf("尾斜杠是不同的身份，该是两条记录，得 %d", len(st.Daemons))
	}
	st.DropEntry(`D:\ws\aapp320`)
	if len(st.Daemons) != 1 {
		t.Errorf("DropEntry 该按同样的小写键删：%+v", st.Keys())
	}
}

// TestStateKeysSorted：清单顺序稳定（map 顺序随机会让同一份状态在两个进程里
// 报出不同的清单，看起来像状态在变）。
func TestStateKeysSorted(t *testing.T) {
	st := newState()
	for _, ws := range []string{"c", "a", "b"} {
		st.SetEntry(ws, &DaemonEntry{Workspace: ws})
	}
	got := strings.Join(st.Keys(), ",")
	if got != "a,b,c" {
		t.Errorf("Keys 该排序，得 %q", got)
	}
}

// TestStateSeparateFromServe：状态文件必须与 .tt-serve.json 分开。
//
// 合并会让 `tt serve` 的单实例判断与 `--stop` 指错人：那边要求 URL 非空，
// 而 tzs 的记录没有 URL —— 于是 serve 会认为自己的状态文件损坏（再拉一个服务抢端口），
// 或者拿着一个 tzs 守护进程的 pid 去杀。
func TestStateSeparateFromServe(t *testing.T) {
	dir := t.TempDir()
	if StatePath(dir) == filepath.Join(dir, ".tt-serve.json") {
		t.Fatal("不能与 .tt-serve.json 是同一个文件")
	}
	if filepath.Base(StatePath(dir)) != StateFileName {
		t.Errorf("状态文件名该是 %s", StateFileName)
	}
	if LogPath(dir) == filepath.Join(dir, ".tt-serve.log") {
		t.Fatal("日志也不能与 serve 的共用一个")
	}
	// 日志路径与状态文件同目录（一起进便携包/一起被清理）。
	if filepath.Dir(LogPath(dir)) != filepath.Dir(StatePath(dir)) {
		t.Errorf("日志该与状态文件同目录")
	}
}

// TestRememberAndLast：`last` 是可写可读的（`call` 省略 --handle 靠它）。
//
// 它**只是建议**：句柄只活在守护进程里，进程一死就没了。所以这里只测「记得住、
// 读得出」，不测「句柄还有效」—— 那是引擎的判断（E_NO_HANDLE，退 2）。
func TestRememberAndLast(t *testing.T) {
	dir := t.TempDir()
	o := Options{Workspace: `D:\ws\a`, WorkDir: dir}
	if Last(o) != nil {
		t.Fatal("还没记过，该是 nil")
	}
	if err := Remember(o, "h2", `D:\pkg\aapp320.tzs`); err != nil {
		t.Fatal(err)
	}
	got := Last(o)
	if got == nil || got.Handle != "h2" || got.OpenPath != `D:\pkg\aapp320.tzs` || got.Workspace != `D:\ws\a` {
		t.Fatalf("last 没记对：%+v", got)
	}
	// 记 last 不该动守护进程那一段（两个字段各管各的）。
	st, err := LoadState(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Daemons) != 0 {
		t.Errorf("记 last 不该写守护进程记录：%+v", st.Daemons)
	}
	// 没配工作区时不记（免得记下一条与真实调用无关的指引）。
	t.Setenv("TZSCLI_WS", "")
	if err := Remember(Options{WorkDir: dir}, "h3", "x"); err == nil {
		t.Error("没有工作区时不该记")
	}
	if Last(o).Handle != "h2" {
		t.Error("被拒的那次不该改掉已有的记录")
	}
}

// TestStateJSONShape 是一份**给人看的**形状断言：参考结构里的字段名不能漂。
func TestStateJSONShape(t *testing.T) {
	b, err := json.Marshal(&DaemonEntry{PID: 1, Pipe: "p", Workspace: "w", Exe: "e", Log: "l", Since: "s"})
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{`"pid"`, `"pipe"`, `"workspace"`, `"exe"`, `"log"`, `"since"`} {
		if !strings.Contains(string(b), key) {
			t.Errorf("记录里缺字段 %s：%s", key, b)
		}
	}
	// 没有孤儿时不写这个字段（老状态文件读进来也不该凭空多出来）。
	if strings.Contains(string(b), "orphans") {
		t.Errorf("没有孤儿时不该写 orphans：%s", b)
	}
}
