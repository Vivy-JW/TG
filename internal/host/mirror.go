package host

// 本地源码镜像引擎:把某环境(T100 服务器)的源码拉到本地镜像目录。
//
// 两个 profile 的白名单：
//   - source（默认）：4gl 目录下后缀为 .4gl 的源码 + 42f（编译表单）+
//     42s/zh_CN（编译字符串）+ *.inc（4GL include）；
//     .4gl.src、.4gltest、.str、per、其它语言目录和备份都不取；
//   - interface：只取 42f 与 42s/zh_CN（供 tt interface 合成画面；只想省体积时
//     用它配一个独立目录，不要和 source 混在同一个目录里）。
//
// **默认 profile 必须是“所有读者的并集”**：一个环境只有一个镜像目录，而
// tt outline 只认 4gl、tt interface 只认 42f/42s，两者读的都是这个目录。少收一类
// 就等于让某个命令静默断粮（命令只会报“找不到文件”，看起来像文件不存在）。
// *.inc 没有命令直接读，它服务的是“人/AI 离线读码”（全局变量、共用参数），
// 缺了同样不报错 —— 这类静默缺口正是最该避免的。
//
// 4fd 是设计期字段描述，fieldId 会随保存重排，两个 profile 都不取。
// 备份/临时文件(x.4gl.bak、x.bck、x.bck1、*~ 等)不拉,并会清理本地残留。
//
// 传输:服务器侧 tar 打包(绝对路径,exec 通道不经登录 profile) → SFTP 流式
// 下载 → 本地标准库解压(gzip+tar),全程不进内存。
// 增量:服务器保留 marker 文件(.tdict-mirror-<env>.mark),pull 默认只打包
// find -newer 的变更文件;--full 全量打包并在本地整目录替换(含删除残留)。
// 白名单升级(新增文件类型)时,本地基线标记版本不一致 → 自动转全量补齐。

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha1"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

type MirrorProfile string

const (
	MirrorSource    MirrorProfile = "source"
	MirrorInterface MirrorProfile = "interface"
)

// ParseMirrorProfile 只认两个公开值；空串按 source。
func ParseMirrorProfile(s string) (MirrorProfile, error) {
	switch MirrorProfile(strings.ToLower(strings.TrimSpace(s))) {
	case "", MirrorSource:
		return MirrorSource, nil
	case MirrorInterface:
		return MirrorInterface, nil
	default:
		return "", fmt.Errorf("未知镜像 profile %q（可用：source | interface）", s)
	}
}

// Label 是给人看的 profile 名称。
func (p MirrorProfile) Label() string {
	if p == MirrorInterface {
		return "42f/42s"
	}
	return "4gl/42f/42s/inc"
}

// MirrorStats 一次镜像拉取的统计结果(CLI 打印用)。
type MirrorStats struct {
	Env     string        // 环境名
	Profile MirrorProfile // source | interface
	TopDir  string        // 服务器 TOP(打包根)
	Full    bool          // 是否全量
	Files   int           // 归档内文件数(无变更时为 0)
	Bytes   int64         // 下载字节数(压缩后)
	Pruned  int           // 本地清理掉的残留备份文件数(历史拉取留下)
	Elapsed string        // 耗时
	Note    string        // 附加说明(如 "无变更")
}

// localMarkName 本地镜像目录内的"完整基线"标记文件:存在表示该目录曾成功
// 建立/更新过完整镜像,增量更新以此为前提;缺失时强制全量(见 MirrorPull)。
const localMarkName = ".tdict-mirror.ok"

// mirrorTag 环境名 → 服务器 marker 文件名 ASCII 尾缀。
// 环境名可能是中文/含特殊字符(不能直接作文件名),且同一服务器可挂多个环境,
// 统一取 sha1 前 10 位十六进制:唯一、安全、与语言无关。
func mirrorTag(name string) string {
	sum := sha1.Sum([]byte(name))
	return fmt.Sprintf("%x", sum)[:10]
}

// mirrorBase 服务器侧 marker / 归档文件的基础路径(不带 .mark/.tar.gz 后缀)。
//
// **带 profile**：服务器上“上次打包到哪个时刻”的记录必须按 profile 分开。否则先用
// source 拉一次、再按 interface 拉时，`find -newer <marker>` 会把 42f/42s 全当成
// 老文件排除掉 —— 打包 0 个文件、还报“无变更”，用户在界面上看到的是成功。
//
// v4 及更早的版本用的是 `<TOP>/.tdict-mirror-<tag>`。升到带 profile 的命名后，旧
// marker 文件会留在服务器上不被引用（不自动删：那是别人的文件，且删了也没收益）。
func mirrorBase(top, envName string, profile MirrorProfile) string {
	if profile == "" {
		profile = MirrorSource
	}
	return top + "/.tdict-mirror-" + mirrorTag(envName) + "-" + string(profile)
}

// MirrorProgress 拉取过程的进度(Web 进度条用;CLI 传 nil 回调即无进度)。
type MirrorProgress struct {
	Phase   string // connect|probe|pack|download|done
	Message string // 人类可读说明
	Bytes   int64  // 已处理字节:pack=服务器归档当前大小,download=已读压缩字节
	Total   int64  // 总字节(仅 download 阶段已知;0=未知)
	Files   int    // 文件数(已知时)
}

// ProgressFunc 进度回调(可能被多次调用,实现需自行加锁)。
type ProgressFunc func(MirrorProgress)

// MirrorPull 拉取/更新指定环境(NamedSsh)的 source 镜像(4gl/**/*.4gl + 42f/42s/zh_CN + *.inc)到 mirrorDir/<环境名>/。
// full=true 全量重建;否则增量(服务器 marker 记录基线)。
// 增量只在本地已有完整基线(envDir 内存在 .tdict-mirror.ok 标记)时允许;
// 本地首次/目录被清/旧版无标记 → 自动转全量,防止"服务器 marker 存在而本地
// 无镜像"时只拉到变更文件造成镜像永久残缺。成功结束后写入/刷新本地基线标记。
// 镜像目录本身(mirrorDir)由调用方保证已配置;本函数只负责连服务器与落盘。
// ctx 见 MirrorPullProgress(它才是实现,这里是给不需要取消的调用方的薄封装)。
func MirrorPull(ctx context.Context, e *NamedSsh, mirrorDir string, full bool) (*MirrorStats, error) {
	return MirrorPullProgress(ctx, e, mirrorDir, full, nil)
}

// MirrorPullToDir 把镜像直接拉到一个指定目录（不再拼 <环境名>）。
// 用于设置页里“每个环境单独指定镜像目录”的场景；默认 source profile。
func MirrorPullToDir(ctx context.Context, e *NamedSsh, envDir string, full bool) (*MirrorStats, error) {
	return MirrorPullToDirProgress(ctx, e, envDir, full, nil)
}

// MirrorPullToDirProfile 按 profile 拉到指定目录。
func MirrorPullToDirProfile(ctx context.Context, e *NamedSsh, envDir string, full bool, profile MirrorProfile) (*MirrorStats, error) {
	return MirrorPullToDirProfileProgress(ctx, e, envDir, full, profile, nil)
}

// MirrorPullProgress 同 MirrorPull,并在各阶段回调 onProgress(供 Web 显示拉取进度)。
//
// 兼容旧调用：mirrorDir 是根目录，实际落点是 mirrorDir/<环境名>。
func MirrorPullProgress(ctx context.Context, e *NamedSsh, mirrorDir string, full bool, onProgress ProgressFunc) (*MirrorStats, error) {
	return MirrorPullToDirProgress(ctx, e, MirrorEnvDir(mirrorDir, e.Name), full, onProgress)
}

// MirrorPullToDirProgress 是 source profile 的薄封装。
func MirrorPullToDirProgress(ctx context.Context, e *NamedSsh, envDir string, full bool, onProgress ProgressFunc) (*MirrorStats, error) {
	return MirrorPullToDirProfileProgress(ctx, e, envDir, full, MirrorSource, onProgress)
}

// MirrorPullToDirProfileProgress 是镜像拉取的实现：envDir 就是本环境的最终目录。
//
// profile 只影响服务器侧 find 的白名单：source=4gl/**/*.4gl + 42f + 42s/zh_CN + *.inc，
// interface=42f/42s。落点、基线、增量、取消语义完全一致 —— 同一目录切 profile 不会
// 删除另一类的文件（全量解压也会把它们原样搬进新目录，见 extractFull）；想只要一类
// 的干净目录，给 interface profile 单独配一个目录。
//
// 服务器标记与本地基线都是**按 profile 分开记**的（mirrorBase）：否则 source 拉过一次
// 之后，再按 interface 拉会命中“没有比标记更新的文件”而报“无变更”——0 个文件、退正常，
// 用户以为拉到了，其实是空跑。
//
// ctx 用来**中途取消**:用户误点「全量重建」时界面上的「停止」按钮走的就是它。
// 取消点选在三处 —— 连接、服务器打包、下载解压循环(见 countingReader / extractTarGz)。
// 取消后返回 ctx.Err()(context.Canceled),调用方据此把任务标成"已停止"而不是"失败"。
//
// 取消不会留下"看着完整"的半个镜像:.tdict-mirror.ok 只在**解压全部成功之后**才写,
// 而 MirrorReady() 判的就是这个标记 —— 所以中途停下的目录会被下次拉取当作"缺基线",
// 自动转全量重建(见上面 needFull 的判定)。
func MirrorPullToDirProfileProgress(ctx context.Context, e *NamedSsh, envDir string, full bool, profile MirrorProfile, onProgress ProgressFunc) (*MirrorStats, error) {
	if profile == "" {
		profile = MirrorSource
	}
	if e == nil {
		return nil, fmt.Errorf("环境为空")
	}
	if strings.TrimSpace(envDir) == "" {
		return nil, fmt.Errorf("镜像目录为空")
	}
	envDir = filepath.Clean(envDir)
	report := func(p MirrorProgress) {
		if onProgress != nil {
			onProgress(p)
		}
	}
	start := time.Now()
	st := &MirrorStats{Env: e.Name, Profile: profile, Full: full}
	if e.Name == "" {
		return nil, fmt.Errorf("环境名为空")
	}

	report(MirrorProgress{Phase: "connect", Message: "连接 SSH " + e.SSHConfig.Addr()})
	conn, err := Dial(e.SSHConfig)
	if err != nil {
		return nil, fmt.Errorf("连接 %s 失败: %w", e.SSHConfig.Addr(), err)
	}
	defer conn.Close()
	if err := ctx.Err(); err != nil { // 连上就被取消:别再往下走
		return nil, err
	}

	report(MirrorProgress{Phase: "probe", Message: "探测 T100 目录(zone " + e.Zone + ")"})
	top, err := mirrorTopDir(conn, e)
	if err != nil {
		return nil, err
	}
	st.TopDir = top

	base := mirrorBase(top, e.Name, profile)

	// 本地无完整基线(首次拉取/目录被清/旧版本遗留),或白名单升级过(增量补不齐
	// 新增类型的历史文件)时,自动按全量拉取,避免镜像残缺
	needFull := full
	if !needFull {
		mark := readLocalMark(envDir)
		switch {
		case !MirrorReadyDir(envDir):
			needFull = true
			logfMirror("本地镜像缺少完整基线标记(%s),本次自动按全量拉取以保证完整", filepath.Join(envDir, localMarkName))
		case mark.Version != mirrorWhitelistVersion:
			needFull = true
			logfMirror("镜像白名单已升级(v%d -> v%d,文件类型有变),本次自动按全量拉取以补齐", mark.Version, mirrorWhitelistVersion)
		case !mark.Has(profile):
			// 这份目录还没拉过该 profile:服务器侧没有它的标记,mirrorPack 会按全量
			// find 打包,本地只需增量写入 —— 不动已有目录,另一类文件原样保留。
			logfMirror("该目录还没拉过 %s profile,本次只补这一类文件(已有的 %s 保留)",
				profile, mark.ProfilesLabel())
		}
	}
	st.Full = needFull

	// 服务器打包(输出 TDICT_MIRROR_OK <文件数> <字节> / TDICT_MIRROR_NONE)
	report(MirrorProgress{Phase: "pack", Message: "服务器打包中…"})
	n, size, err := mirrorPackProgress(ctx, conn, base, top, needFull, profile, report)
	if err != nil {
		return nil, fmt.Errorf("服务器打包失败: %w", err)
	}
	if n == 0 {
		st.Elapsed = time.Since(start).Round(100 * time.Millisecond).String()
		st.Note = "无变更(服务器 " + profile.Label() + " 自上次 pull 后未修改;--full 可强制全量重建)"
		// 服务器无变更但本地基线标记缺失时,依然补一个基线标记,
		// 否则下次 pull 仍会误判"无基线"而反复尝试全量
		if !needFull {
			if err := writeLocalMark(envDir, profile); err != nil {
				logfMirror("写入本地基线标记失败(忽略): %v", err)
			}
		}
		// 服务器虽无变更,本地仍可能残留旧版本拉下来的备份文件
		st.Pruned = pruneBackupFiles(envDir)
		if st.Pruned > 0 {
			st.Note += fmt.Sprintf(";已清理本地备份 %d 个", st.Pruned)
		}
		report(MirrorProgress{Phase: "done", Message: st.Note})
		return st, nil
	}
	st.Bytes = size

	// 下载并解压(流式):countingReader 按压缩字节回报进度
	if err := os.MkdirAll(filepath.Dir(envDir), 0o755); err != nil {
		return nil, fmt.Errorf("创建镜像父目录失败: %w", err)
	}
	sc, err := conn.SFTP()
	if err != nil {
		return nil, fmt.Errorf("打开 SFTP 失败: %w", err)
	}
	defer sc.Close()
	rf, err := sc.Open(base + ".tar.gz")
	if err != nil {
		return nil, fmt.Errorf("打开服务器归档失败: %w", err)
	}
	defer rf.Close()

	report(MirrorProgress{Phase: "download", Message: "下载并解压…", Total: size})
	cr := &countingReader{ctx: ctx, r: rf, total: size, report: report}
	var files int
	if needFull {
		files, err = extractFull(cr, envDir, profile)
	} else {
		files, err = extractIncremental(cr, envDir)
	}
	if err != nil {
		// 用户中途取消:原样返回 context.Canceled(调用方据此标"已停止",不当失败)。
		// 此时**不写基线标记** —— 半个镜像不会被下次拉取当成完整基线(见函数注释)。
		// 全量路径的 .staging 临时目录由 extractFull 自己清掉。
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("本地解压失败: %w", err)
	}
	// 解压完成、还没落任何"完成"痕迹之前再确认一次取消:
	// 否则用户刚点停止、这边刚好解压完,会留下一个标记完整但用户以为已停止的镜像。
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	st.Files = files
	if err := writeLocalMark(envDir, profile); err != nil {
		return nil, fmt.Errorf("写入本地基线标记失败: %w", err)
	}
	// 增量拉取不会删旧文件:清掉历史版本留在本地的备份文件(全量重建时本就没有)
	st.Pruned = pruneBackupFiles(envDir)

	// 清理服务器归档(marker 保留,供下次增量)
	if out, err := conn.Output(`rm -f "`+base+`.tar.gz"`, time.Minute); err != nil && !strings.Contains(out, "No such file") {
		// 清理失败不阻断:下次全量/增量都会重建归档
		logfMirror("清理服务器归档失败(忽略): %v", err)
	}
	st.Elapsed = time.Since(start).Round(100 * time.Millisecond).String()
	msg := fmt.Sprintf("完成:%d 个文件", files)
	if st.Pruned > 0 {
		msg += fmt.Sprintf(",清理本地备份 %d 个", st.Pruned)
	}
	report(MirrorProgress{Phase: "done", Message: msg, Bytes: size, Total: size, Files: files})
	return st, nil
}

// countingReader 统计已读字节并节流回调进度(下载/解压阶段共用;压缩字节即下载量)。
// 同时是**取消点**:带 ctx 时每次 Read 先看有没有被取消,取消了就返回 ctx.Err(),
// 让解压链路立刻断开(不必等整个归档下完)。
type countingReader struct {
	ctx    context.Context
	r      io.Reader
	n      int64
	total  int64
	last   time.Time
	report ProgressFunc
}

func (c *countingReader) Read(p []byte) (int, error) {
	if c.ctx != nil {
		if err := c.ctx.Err(); err != nil {
			return 0, err
		}
	}
	n, err := c.r.Read(p)
	if n > 0 {
		c.n += int64(n)
	}
	now := time.Now()
	// 节流:每 200ms 一次;流结束时(EOF)必报一次最终字节数。
	// report 可能为 nil —— MirrorPullProgress 的进度回调是可选参数,
	// 而 CLI(tt file mirror)就是不传的那个;这里不判空会直接空指针崩。
	if c.report != nil && ((n > 0 && now.Sub(c.last) >= 200*time.Millisecond) || err == io.EOF) {
		c.last = now
		c.report(MirrorProgress{Phase: "download", Message: "下载并解压…", Bytes: c.n, Total: c.total})
	}
	return n, err
}

// mirrorPackProgress 在服务器打包期间轮询归档大小回报进度(pack 阶段无法预知总量)。
// ctx 取消时主动返回:打包在服务器上是 `tar` 子进程,取消不会杀它,但至少本端立刻收手,
// 不再等它跑完(服务器归档是临时文件,下次拉取会重建)。
func mirrorPackProgress(ctx context.Context, conn *SSHConn, base, top string, full bool, profile MirrorProfile, report ProgressFunc) (int, int64, error) {
	var stopped atomic.Bool
	done := make(chan struct{})
	defer func() {
		stopped.Store(true)
		close(done)
	}()
	go func() {
		t := time.NewTicker(2 * time.Second)
		defer t.Stop()
		cmd := `stat -c %s ` + shdq(base+".tar.gz") + ` 2>/dev/null || echo 0`
		for {
			select {
			case <-done:
				return
			case <-ctx.Done():
				return
			case <-t.C:
				out, err := conn.Output(cmd, 8*time.Second)
				if err != nil {
					continue
				}
				var sz int64
				if _, err := fmt.Sscanf(strings.TrimSpace(out), "%d", &sz); err == nil && sz > 0 && !stopped.Load() {
					report(MirrorProgress{Phase: "pack", Message: "服务器打包中…", Bytes: sz})
				}
			}
		}
	}()
	// 打包本身也放进一个受 ctx 约束的等待:取消了就不再阻塞在后端调用上
	type packResult struct {
		n    int
		size int64
		err  error
	}
	ch := make(chan packResult, 1)
	go func() {
		n, size, err := mirrorPack(conn, base, top, full, profile)
		ch <- packResult{n, size, err}
	}()
	select {
	case <-ctx.Done():
		return 0, 0, ctx.Err()
	case r := <-ch:
		return r.n, r.size, r.err
	}
}

// localMark 是本地基线标记(.tdict-mirror.ok)的内容。
//
//	v5 起: "v5 source,interface 2026-10-26T10:00:00Z"
//	更早:   "v4 2026-10-26T10:00:00Z"(无 profile 列表)或纯时间戳
//
// profiles 记的是“这份目录拉过哪些 profile”,它是**并集**:第二次拉另一个 profile
// 不能把前一个从记录里抹掉,否则日志/判断会把“两类都在”误报成“只有一类”。
type localMark struct {
	Version  int
	Profiles map[MirrorProfile]bool
}

func (m localMark) Has(p MirrorProfile) bool { return m.Profiles[p] }

// ProfilesLabel 按固定顺序列出已拉过的 profile(日志与测试需要稳定输出)。
func (m localMark) ProfilesLabel() string {
	var out []string
	for _, p := range []MirrorProfile{MirrorSource, MirrorInterface} {
		if m.Profiles[p] {
			out = append(out, string(p))
		}
	}
	if len(out) == 0 {
		return "(无记录)"
	}
	return strings.Join(out, ",")
}

// readLocalMark 读本地基线标记;无标记、旧格式、字段缺失都返回零值(版本 0、空集合),
// 调用方据此当作“无基线”处理,不会凭空认为镜像完整。
func readLocalMark(envDir string) localMark {
	m := localMark{Profiles: map[MirrorProfile]bool{}}
	b, err := os.ReadFile(filepath.Join(envDir, localMarkName))
	if err != nil {
		return m
	}
	fields := strings.Fields(strings.TrimSpace(string(b)))
	if len(fields) == 0 || !strings.HasPrefix(fields[0], "v") {
		return m
	}
	if n, err := strconv.Atoi(strings.TrimPrefix(fields[0], "v")); err == nil {
		m.Version = n
	}
	for _, f := range fields[1:] {
		// profile 集合写成一个逗号分隔的字段（source,interface）；也容忍空格分隔的旧写法。
		for _, name := range strings.Split(f, ",") {
			switch p := MirrorProfile(strings.ToLower(strings.TrimSpace(name))); p {
			case MirrorSource, MirrorInterface:
				m.Profiles[p] = true
			}
		}
	}
	return m
}

// writeLocalMark 在镜像环境目录写入完整基线标记(白名单版本 + 已拉 profile 集合 + 时间)。
// profile 取并集:同目录先后拉两种 profile 后,记录里两种都在。
func writeLocalMark(envDir string, profile MirrorProfile) error {
	if err := os.MkdirAll(envDir, 0o755); err != nil {
		return err
	}
	mark := readLocalMark(envDir)
	if mark.Version != mirrorWhitelistVersion {
		// 版本升级:旧的 profile 记录不再作数(白名单变了,那一类可能得重新拉),
		// 但保留本次这个 —— 它刚刚被完整解压过。
		mark.Profiles = map[MirrorProfile]bool{}
	}
	if mark.Profiles == nil {
		mark.Profiles = map[MirrorProfile]bool{}
	}
	if profile != "" {
		mark.Profiles[profile] = true
	}
	return os.WriteFile(filepath.Join(envDir, localMarkName),
		[]byte(fmt.Sprintf("v%d %s %s\n", mirrorWhitelistVersion, mark.ProfilesLabel(),
			time.Now().UTC().Format(time.RFC3339))), 0o644)
}

// localMarkVersion 读本地基线标记里的白名单版本;无标记或旧格式(纯时间戳)返回 0。
func localMarkVersion(envDir string) int {
	return readLocalMark(envDir).Version
}

// MirrorEnvDir 返回某环境的本地镜像目录(mirrorDir/<环境名>)。
func MirrorEnvDir(mirrorDir, envName string) string {
	if mirrorDir == "" || envName == "" {
		return ""
	}
	return filepath.Join(mirrorDir, envName)
}

// MirrorReady 本地是否已有完整镜像基线(.tdict-mirror.ok 标记;增量更新的前提)。
// 无基线时 MirrorPull 会自动转全量。
func MirrorReady(mirrorDir, envName string) bool {
	return MirrorReadyDir(MirrorEnvDir(mirrorDir, envName))
}

// MirrorReadyDir 判一个“最终镜像目录”有没有完整基线。
func MirrorReadyDir(envDir string) bool {
	if strings.TrimSpace(envDir) == "" {
		return false
	}
	_, err := os.Stat(filepath.Join(envDir, localMarkName))
	return err == nil
}

func logfMirror(format string, args ...any) {
	// 轻量日志:镜像失败清理之类不致命信息走 stderr(无全局 logger 依赖)
	fmt.Fprintf(os.Stderr, "[mirror] "+format+"\n", args...)
}

// mirrorWhitelistFindExpr 服务器侧 find 的白名单。
//
// source 的四类各自限定：4gl 用后缀级限制 `-path '*/4gl/*' -iname '*.4gl'`（因此
// a.4gl.src、a.4gltest、a.4gl.20201110 都不会进来）；42f 与 42s 按目录限定（42s 只收
// zh_CN 语言目录）；*.inc 不限定目录（技能文档里的 inc 就在 4g1/、$ERP/cfg、$COM/cfg 多处）。
//
// **source 必须包含 4gl + 42f + 42s + inc 四类**：一个环境只有一个镜像目录，tt outline
// 只认 4gl、tt interface 只认 42f/42s，两者读同一个目录；少收一类就等于让某个命令
// 静默断粮（它只会报“找不到文件”，看起来像文件不存在）。
//
// 改这里必须同步 mirrorPathInProfile（本地分类器，全量解压时用它决定“哪些文件属于
// 本 profile、要以服务器为准”），mirror_test.go 里有交叉验证。
func mirrorWhitelistFindExpr(profile MirrorProfile) string {
	if profile == MirrorInterface {
		return `\( -path '*/42f/*' -o -path '*/42s/zh_CN/*' -o -path '*/42s/*/zh_CN/*' \)`
	}
	return `\( -path '*/4gl/*' -iname '*.4gl' ` +
		`-o -path '*/42f/*' ` +
		`-o -path '*/42s/zh_CN/*' -o -path '*/42s/*/zh_CN/*' ` +
		`-o -iname '*.inc' \)`
}

// mirrorPathInProfile 判断一个**本地**相对路径是否落在该 profile 的白名单内。
// 只给全量解压用：属于本 profile 的文件以服务器为准（新目录里没有就是服务器删了），
// 不属于本 profile 的文件原样保留（不因“给 A 做了全量”而删掉 B）。
//
// 语义必须与 mirrorWhitelistFindExpr 一致：/"/" 前缀归一化后，路径分段判断。
func mirrorPathInProfile(profile MirrorProfile, rel string) bool {
	p := strings.ToLower("/" + strings.TrimPrefix(filepath.ToSlash(rel), "./"))
	is42f := strings.Contains(p, "/42f/")
	// 42s 只收 zh_CN：浅层 <模块>/42s/zh_CN/* 与更深层 <模块>/42s/<...>/zh_CN/* 都算。
	is42s := strings.Contains(p, "/42s/") && strings.Contains(p, "/zh_cn/")
	if profile == MirrorInterface {
		return is42f || is42s
	}
	is4gl := strings.Contains(p, "/4gl/") && strings.HasSuffix(p, ".4gl")
	isInc := strings.HasSuffix(p, ".inc")
	return is4gl || is42f || is42s || isInc
}

// mirrorWhitelistVersion 白名单版本:新增文件类型时 +1。
// 本地基线标记(.tdict-mirror.ok)里记录该版本;版本不一致说明白名单升级过,
// 增量无法补齐新增类型的历史文件,于是下次拉取自动转全量(见 MirrorPullProgress)。
//
//	v1: 4gl/4fd
//	v2: +42s/zh_CN、*.inc
//	v3: 4fd → 42f（前端画面改为 42f+42s；4fd 不再镜像）
//	v4: source 收紧为 4gl/**/*.4gl；interface 保留 42f/42s
//	v5: source 恢复为 4gl + 42f + 42s/zh_CN + *.inc（一个镜像目录必须装齐所有读者的并集）
const mirrorWhitelistVersion = 5

// mirrorBackupSuffixes 备份/临时文件名特征(不含点全名包含即算备份)。
// 服务器侧打包排除与本地残留清理共用同一份列表,避免两边规则漂移:
//   - 框架/人工备份:x.4gl.bak、x.42f.bak、x.bck、x.bck1、x.bck2 …
//   - 编辑器临时文件:x.4gl~、x.swp、x.tmp 等
var mirrorBackupSuffixes = []string{".bak", ".bck", ".old", ".orig", ".tmp", ".swp", ".swo"}

// isBackupFile 判断文件名是否备份/临时文件(与服务器侧 find 排除规则一致)。
func isBackupFile(name string) bool {
	n := strings.ToLower(strings.TrimSpace(name))
	if n == "" {
		return false
	}
	if strings.HasSuffix(n, "~") {
		return true
	}
	for _, suf := range mirrorBackupSuffixes {
		if strings.Contains(n, suf) {
			return true
		}
	}
	return false
}

// mirrorBackupFindExpr 生成服务器侧 find 的备份排除条件(与 isBackupFile 同语义)。
func mirrorBackupFindExpr() string {
	parts := make([]string, 0, len(mirrorBackupSuffixes)+1)
	for _, suf := range mirrorBackupSuffixes {
		parts = append(parts, `-not -iname '*`+suf+`*'`)
	}
	parts = append(parts, `-not -iname '*~'`)
	return strings.Join(parts, " ")
}

// pruneBackupFiles 删除本地镜像目录内残留的备份文件(旧版本拉取过),返回删除数。
// 只删除文件,不动目录;root 不存在时静默返回 0。
func pruneBackupFiles(root string) int {
	removed := 0
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d == nil || d.IsDir() {
			return nil
		}
		if isBackupFile(d.Name()) {
			if os.Remove(path) == nil {
				removed++
			}
		}
		return nil
	})
	return removed
}

// mirrorTopDir 确定打包根:仅 SSH 登录探测(Runtime.TOP)——T100 路径无静态配置,
// 一律按登录区域动态获取;探测失败即报错。
func mirrorTopDir(conn *SSHConn, e *NamedSsh) (string, error) {
	if conn == nil || e.Zone == "" {
		return "", fmt.Errorf("环境 %q 缺少登录区域(zone),无法探测 T100 目录", e.Name)
	}
	env, err := ProbeTEnv(conn, e.Zone)
	if err != nil {
		return "", fmt.Errorf("环境 %s(zone %s)登录探测 T100 目录失败: %w", e.Name, e.Zone, err)
	}
	return strings.TrimSuffix(env.TOP, "/"), nil
}

// shdq 把值包成双引号 shell 字面量(路径含特殊字符安全)。
func shdq(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "$", `\$`)
	s = strings.ReplaceAll(s, "`", "\\`")
	return `"` + s + `"`
}

// mirrorPack 在服务器 TOP 下打包 profile 白名单文件。返回 (文件数, 归档字节);
// 无变更时返回 (0, 0) 且不产生归档。marker 保留在 TOP 下供下次增量。
func mirrorPack(conn *SSHConn, base, top string, full bool, profile MirrorProfile) (int, int64, error) {
	m := shdq(base + ".mark")
	o := shdq(base + ".tar.gz")
	l := shdq(base + ".list")
	t := shdq(top)
	// 注意:exec 通道不经过登录 profile,但这里只依赖绝对路径与标准工具
	// 白名单树(按 profile)+ 排除备份/临时文件(如 x.4gl.bak、x.bck、x.bck1)
	findExpr := mirrorWhitelistFindExpr(profile) + " " + mirrorBackupFindExpr()
	fullFlag := "0"
	if full {
		fullFlag = "1"
	}
	script := `set -e
cd ` + t + ` || exit 9
R=""
for d in erp com; do
  [ -d "$d" ] && R="$R $d"
done
if [ -z "$R" ]; then
  echo TDICT_MIRROR_NONE
  exit 0
fi
if [ "` + fullFlag + `" = "1" ] || [ ! -f ` + m + ` ]; then
  find $R -type f ` + findExpr + ` > ` + l + `
else
  find $R -type f ` + findExpr + ` -newer ` + m + ` > ` + l + `
fi
if [ -s "` + l + `" ]; then
  n=$(wc -l < "` + l + `")
  tar -czf ` + o + ` -T "` + l + `"
  touch ` + m + `
  s=$(wc -c < "` + o + `")
  rm -f "` + l + `"
  echo "TDICT_MIRROR_OK $n $s"
else
  rm -f "` + l + `"
  echo TDICT_MIRROR_NONE
fi`
	out, err := conn.Output(script, 20*time.Minute)
	if err != nil {
		return 0, 0, fmt.Errorf("%s(输出: %s)", err, strings.TrimSpace(out))
	}
	// 取最后一行标记(前面可能有 tar 警告)
	mark := ""
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "TDICT_MIRROR_OK") || line == "TDICT_MIRROR_NONE" {
			mark = line
		}
	}
	switch {
	case strings.HasPrefix(mark, "TDICT_MIRROR_OK"):
		var n int
		var s int64
		if _, err := fmt.Sscanf(mark, "TDICT_MIRROR_OK %d %d", &n, &s); err != nil {
			return 0, 0, fmt.Errorf("解析打包标记失败: %q", mark)
		}
		return n, s, nil
	case mark == "TDICT_MIRROR_NONE":
		return 0, 0, nil
	default:
		return 0, 0, fmt.Errorf("服务器脚本未输出预期标记(输出: %s)", strings.TrimSpace(out))
	}
}

// extractIncremental 增量解压:直接覆盖写入 <mirrorDir>/<环境名>/(文件级幂等)。
func extractIncremental(r io.Reader, envDir string) (int, error) {
	if err := os.MkdirAll(envDir, 0o755); err != nil {
		return 0, err
	}
	return extractTarGz(r, envDir)
}

// extractFull 全量解压:先写 .staging 临时目录,再把**本 profile 之外**的旧文件搬进去,
// 最后整目录替换。
//
// 为什么不能“旧目录整个删掉”:一个环境只有一个镜像目录,source(4gl/42f/42s/inc)与
// interface(42f/42s)可能先后拉进同一个目录。旧实现直接删旧目录,于是“给 profile A
// 做一次全量”会把 profile B 的文件一起删掉 —— 用户看到的是“我只拉了一类,另一类没了”。
// 全量的语义应当是“本 profile 的白名单内以服务器为准”,不是“清空别人的东西”。
func extractFull(r io.Reader, envDir string, profile MirrorProfile) (int, error) {
	ts := time.Now().Format("20060102-150405.000")
	staging := filepath.Join(filepath.Dir(envDir), ".staging-"+ts)
	if err := os.MkdirAll(staging, 0o755); err != nil {
		return 0, err
	}
	n, err := extractTarGz(r, staging)
	if err != nil {
		os.RemoveAll(staging)
		return 0, err
	}
	// 另一类的既有文件先搬进新目录:全量只对本 profile 的白名单负责。
	if kept, kerr := keepOtherProfileFiles(envDir, staging, profile); kerr != nil {
		os.RemoveAll(staging)
		return 0, kerr
	} else if kept > 0 {
		logfMirror("全量重建保留另一类文件 %d 个(不属于 %s 的白名单)", kept, profile)
	}
	old := envDir + ".old-" + ts
	if _, err := os.Stat(envDir); err == nil {
		if err := os.Rename(envDir, old); err != nil {
			os.RemoveAll(staging)
			return 0, fmt.Errorf("备份旧镜像目录失败: %w", err)
		}
	}
	if err := os.Rename(staging, envDir); err != nil {
		// 回滚:把旧目录放回去
		if _, serr := os.Stat(old); serr == nil {
			os.Rename(old, envDir)
		}
		os.RemoveAll(staging)
		return 0, fmt.Errorf("启用新镜像目录失败: %w", err)
	}
	if _, err := os.Stat(old); err == nil {
		if err := os.RemoveAll(old); err != nil {
			logfMirror("删除旧镜像备份失败(可手动清理 %s): %v", old, err)
		}
	}
	return n, nil
}

// keepOtherProfileFiles 把 oldDir 里**不属于当前 profile** 的普通文件搬进 staging。
//
// 几条规矩：
//   - staging 已有同名文件 = 服务器也给了它 → 以服务器为准，不覆盖；
//   - 备份/临时文件不搬（它们由 pruneBackupFiles 清理）；
//   - 基线标记 .tdict-mirror.ok 不搬（新目录会在解压完成后重新写，且要记上本 profile）；
//   - .staging-* / .old-* 这些临时目录跳过（历史失败留下时不当内容）。
func keepOtherProfileFiles(oldDir, staging string, profile MirrorProfile) (int, error) {
	if _, err := os.Stat(oldDir); err != nil {
		return 0, nil // 首次拉取:没有旧目录
	}
	kept := 0
	err := filepath.WalkDir(oldDir, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil // 单个文件读不到不该打断整个重建
		}
		if d.IsDir() {
			name := d.Name()
			if p != oldDir && (strings.HasPrefix(name, ".staging-") || strings.HasPrefix(name, ".old-")) {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() || isBackupFile(d.Name()) {
			return nil
		}
		rel, rerr := filepath.Rel(oldDir, p)
		if rerr != nil || rel == localMarkName {
			return nil
		}
		if mirrorPathInProfile(profile, rel) {
			return nil // 属于本 profile:以服务器为准(新目录里没有就是服务器删了)
		}
		dst := filepath.Join(staging, rel)
		if _, serr := os.Stat(dst); serr == nil {
			return nil
		}
		if err := copyRegularFile(p, dst); err != nil {
			return err
		}
		kept++
		return nil
	})
	return kept, err
}

// copyRegularFile 复制单个普通文件（保留 mtime，与原解压行为一致）。
func copyRegularFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	if st, err := in.Stat(); err == nil {
		_ = os.Chtimes(dst, st.ModTime(), st.ModTime())
	}
	return nil
}

// extractTarGz 从 tar.gz 流解压到 dest(标准库;不做任何 shell)。返回文件数。
// tar 头路径做穿越防护;回写 mtime 便于与服务器比对;文件 0644。
func extractTarGz(r io.Reader, dest string) (int, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return 0, fmt.Errorf("解压 gzip: %w", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	files := 0
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return files, err
		}
		name := filepath.Clean(filepath.FromSlash(hdr.Name))
		if name == "." || filepath.IsAbs(name) || name == ".." || strings.HasPrefix(name, ".."+string(filepath.Separator)) {
			return files, fmt.Errorf("归档含非法路径: %q", hdr.Name)
		}
		target := filepath.Join(dest, name)
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return files, err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return files, err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
			if err != nil {
				return files, err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return files, err
			}
			f.Close()
			_ = os.Chtimes(target, hdr.ModTime, hdr.ModTime)
			files++
		default:
			// 符号链接/设备等不落盘(白名单内不会出现)
		}
	}
	return files, nil
}
