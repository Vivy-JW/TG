package tzs

// corpus_test.go —— `.tzs` 语料关卡（旧实现：engine/batch.sh / batch-write.sh /
// batch-edit.sh / batch-action.sh，以及 gate-w3.py 的段 B）。
//
// 默认 `go test ./...` **一条都不跑**：语料回归要 67 个真实包、一份设计器、一个引擎产物目录，
// 而默认档整轮就要十几分钟 —— 塞进默认测试会让 `go test ./...` 变成随机器负载时好时坏的
// 假失败（cli/corpus_test.go 的 requireDeep 是同一个理由，那边的数是 9–11 分钟）。
//
// 开启方式（三样都得给；工作区**不由测试替你选** —— 每个包从自己的祖先目录里推）：
//
//	$env:TTZS_DEEP="1"
//	$env:TTZS_EXE="D:\...\engine\out\tzs-server.exe"          # 可省：仓库里 engine/out 的缺省
//	$env:TTZS_CORPUS="D:\t100_wrok_dir"                        # 可省：这就是缺省（TDEV_CORPUS 也认）
//	$env:TTZS_INSTALL="D:\APPS\T100设计器_1.0.0.251_免安装"      # 可省：引擎有内置缺省
//	go test ./internal/dev/tzs/ -run 'TestCorpus' -timeout 30m -v
//
// 另外两个旋钮：
//
//	TTZS_VALIDATE=sample|all   validate 跑在哪些包上（默认 sample：每轴一个，见 validateSample）
//	TTZS_CORPUS_LIMIT=N        只跑前 N 个包（冒烟用；旧实现的第二个位置参数就是它）
//
// 三条照抄的教训（旧关卡的 docstring 自己写的，逐条落在这里）：
//
//  1. **判据必须基线相对，不能硬编码零。** 实测三个语料包在**没人动过**的时候就报 stale
//     （capt111=3、cpmp530=2、cpmq001=2）。所以判据是「产出的 stale 等于 pristine 的 stale」
//     而不是「等于 0」；tsd/paths 的增删则必须是 0 —— 那三个包在 pristine 上也是 0。
//     pristine 自己先量一遍就是这条的落地方式：不需要存一张基线表。
//     旧实现的原话：「A gate that fails a file for a pre-existing property is worse than
//     no gate.」
//  2. **一包一进程。** 一个弹了模态对话框卡住的包只赔它自己那 120 s（perFileBudget），
//     而不是整轮。
//  3. **validate 要分层。** 它 1.3 s（114 元素）到 10.4 s（670 元素）一次，而每个包要调
//     **两次**（第一次就是 baseline，见 validate 的注释）。67 包 × 2 × 10 s ≈ 22 分钟光是
//     校验，所以默认只跑「每轴一个」的样本，全量留给 TTZS_VALIDATE=all。
//
// 两条**跑的时候**才知道的纪律（2026-09-26 加的，两条都踩过）：
//
//  4. **整轮在跑的时候不要重建引擎。** `engine/build.sh` 会覆盖 `out/*.exe`，而这一套
//     每一包都要 spawn `RoundTrip.exe`（`runRoundTrip`）—— 撞上重写的那一瞬间，得到的是
//     `fork/exec … RoundTrip.exe: The system cannot find the file specified`，表现为
//     两三条**假红**，而报告上看不出是并发构建造成的。
//  5. **用一份工作区副本跑**（`TTZS_CORPUS` 指到 `%TEMP%\ttws` 那样的拷贝）。缺省根是
//     `D:\t100_wrok_dir`，那是**真实客户目录**：这一套会在源包旁边写 `_tdev_*` 临时包
//     （跑完就删，但跑在客户目录上）。副本还顺带证明另一件事 —— 真语料一个字节都没被动过
//     （`TestCorpusPin` 对副本会报"数量对不上"，它是对着 pin 里那 67 条真路径算的；单跑
//     它、不设 `TTZS_CORPUS`，就能看到真语料的现状）。

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"tt/internal/dev/testutil"
	"tt/internal/testenv"
)

const (
	// deepEnv 是语料回归的总开关。
	deepEnv = "TTZS_DEEP"
	// validateEnv 决定 validate 跑在哪些包上。
	validateEnv = "TTZS_VALIDATE"
	// limitEnv 只跑前 N 个包（冒烟）。
	limitEnv = "TTZS_CORPUS_LIMIT"
	// perFileBudget 是一个包的预算，不是整轮的。照抄旧实现：batch.sh 的第二个位置参数
	// 缺省就是 120 s，理由是「a package that pops a modal dialog and hangs costs us its
	// timeout and nothing more」。
	perFileBudget = 120 * time.Second
)

// scratchPrefixes 是不算语料的文件名前缀：四个 batch 脚本与 gate-w3.py 各自排除集的**并集**
// （它们的注释解释了为什么必须取并集），外加 _tdev —— 本测试自己的临时产出。
// _tdev 必须在里面，否则一次跑崩留下的临时包会被下一轮当成语料。
//
// _tt_dry 与它们不同类：它是**引擎**自己产的临时包（dry-run 的回滚快照，见 DryRun.cs，
// 名字是 _tt_dry_<pid>_<n>.tzs，正常路径上写完就删）。放进这张表是因为"正常路径"之外还有
// 一条：守护进程在回滚中间死掉，包就留在源包旁边了 —— 而它落在**工作区里**，下一次语料遍历
// 会把它当成一份语料（实测留下过一个）。engine/make-manifest.sh 的排除集里也有它。
var scratchPrefixes = []string{"_ai", "_dw", "_ed", "_del", "_ac", "_tdev", "_tt_dry"}

// validateSample 是「每轴一个」的 validate 样本，逐字照搬 gate-w3.py 的 DEEP_SAMPLE。
//
// validate 是这一轮里唯一昂贵的断言（1.3–10.4 s 一次，每包两次），而这几个包各自开着
// 判定可能翻转的一根不同的轴 —— 剩下的包仍然跑便宜的那三条断言。全量留给 TTZS_VALIDATE=all。
//
// 与旧清单的一处差异：旧注释提到「最小字段表」那一轴随着 xiyuan/tst 模块被清掉而消失，
// 现在也没有对应的包，所以这里同样没有它 —— 记在这里，免得下一轮误以为漏抄了一条。
var validateSample = map[string]string{
	"aapp320(c).tzs":    "tpl=P，文档里每个例子都用它",
	"cpmp530(c).tzs":    "tpl=Q，且是三个自带 posX 漂移的包之一",
	"asft330(c).tzs":    "xiyuan/prd —— 跨工作区那一轴（另一个 TZSCLI_WS）",
	"aist310_wf(c).tzs": "Tree，577 元素，34 个 act",
	"axmt500_wf(c).tzs": "678 元素，动它之前就已经有 11 个 WARNING",
}

//---------------------------------------------------------------------------
// 开关与前置条件
//---------------------------------------------------------------------------

func requireDeep(t *testing.T) {
	t.Helper()
	if os.Getenv(deepEnv) == "" {
		t.Skip("语料深度回归未启用：设 TTZS_DEEP=1 并给足 -timeout 30m（见 corpus_test.go 顶部注释）")
	}
}

// corpusEnv 是一次语料回归的解析结果。
type corpusEnv struct {
	exe  string   // tzs-server.exe
	dir  string   // 引擎产物目录（RoundTrip.exe 在这个目录里）
	root string   // 语料根
	pkgs []string // 语料包，路径升序
}

// requireCorpus 把「真引擎 + 真语料」设成显式前置，缺哪样就说清缺哪样。
//
// 缺东西时**整条跳过**而不是逐包跳过：这些都是「这台机器上跑不了这批用例」的性质，
// 与某个包的性质无关，混成逐包跳过会让人以为跑过一次了。
func requireCorpus(t *testing.T) *corpusEnv {
	t.Helper()
	requireDeep(t)
	return requireCorpusEnv(t)
}

// engineExeAndDir 解析「引擎 exe 在哪、它的产物目录在哪」，缺了就说清缺什么、怎么办。
//
// 单独拆出来是因为它服务的**不只是**真语料那几条：mini_corpus_test.go 用的是仓库自带的
// 语料，同样需要一个引擎 —— 那条测试的前置只有引擎这一样，语料不需要。
func engineExeAndDir(t *testing.T) (exe, dir string) {
	t.Helper()
	exe = testenv.EngineExe()
	if exe == "" {
		// 从包目录出发的相对缺省（本仓库里引擎的产物就在那儿），只是省事，不是保证。
		exe = filepath.Join("..", "..", "..", "engine", "out", "tzs-server.exe")
	}
	if abs, err := filepath.Abs(exe); err == nil {
		exe = abs
	}
	if _, err := os.Stat(exe); err != nil {
		t.Skipf("找不到引擎 exe（TTZS_EXE=%s）：%v；先在 engine/ 里跑 build.sh", exe, err)
	}
	dir = filepath.Dir(exe)
	if _, err := os.Stat(filepath.Join(dir, "RoundTrip.exe")); err != nil {
		t.Skipf("%s 旁边没有 RoundTrip.exe：%v", dir, err)
	}
	return exe, dir
}

// requireCorpusEnv 只解析「真引擎 + 真语料 + 工作区」，**不碰深度开关**。
//
// 之所以拆出来：函数面那条关卡（fns_test.go）有自己的开关 TTZS_FNS，
// 不该被迫把整个 17 分钟的深档回归也打开。
func requireCorpusEnv(t *testing.T) *corpusEnv {
	t.Helper()
	exe, dir := engineExeAndDir(t)
	root := testutil.CorpusRoot()
	if root == "" {
		t.Skipf("没有真实语料（%s / %s 都没指到一个目录）：设 TTZS_CORPUS，或准备 %s",
			"TDEV_CORPUS", "TTZS_CORPUS", testutil.DefaultCorpusRoot)
	}
	pkgs := testutil.CorpusFiles(root, ".tzs", scratchPrefixes...)
	if len(pkgs) == 0 {
		t.Skipf("%s 下没有 .tzs（排除掉 %s 之后）", root, strings.Join(scratchPrefixes, "/"))
	}
	if n := corpusLimit(t); n > 0 && n < len(pkgs) {
		t.Logf("%s=%d：只跑前 %d 个包（冒烟用，不是回归）", limitEnv, n, n)
		pkgs = pkgs[:n]
	}
	return &corpusEnv{exe: exe, dir: dir, root: root, pkgs: pkgs}
}

func corpusLimit(t *testing.T) int {
	t.Helper()
	v := strings.TrimSpace(os.Getenv(limitEnv))
	if v == "" {
		return 0
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		t.Fatalf("%s=%q 不是一个非负整数", limitEnv, v)
	}
	return n
}

// wantsValidate 决定这个包跑不跑 validate。
//
// 取值不认识时**当场失败**而不是回落到 sample：一个打错的 TTZS_VALIDATE=al 会让整轮悄悄
// 少跑大部分校验，而报告上看起来完全正常。
func wantsValidate(t *testing.T, name string) bool {
	t.Helper()
	switch v := strings.TrimSpace(os.Getenv(validateEnv)); v {
	case "", "sample":
		_, ok := validateSample[name]
		return ok
	case "all":
		return true
	default:
		t.Fatalf("%s=%q 不认识；取 sample（默认，每轴一个）或 all（全量，约 20 分钟）", validateEnv, v)
		return false
	}
}

// workspaceOf 取包所属的工作区：哪个**祖先**目录里有 mta/，它就是工作区。
//
// 推导而不是硬编码一张路径表：一个模块目录之所以是工作区，就是因为那个 mta/ 目录，
// 而 TzpManager 会拒绝配置之外的工作区里的包（batch.sh 的注释）。
//
// 推不出来时返回 ""（调用方跳过），**不回落**到任何缺省 —— 引擎内置的缺省工作区是一个真实
// 客户目录，回落到它上面就是拿客户的表单当草稿纸（与包注释纪律 2 同一条）。
func workspaceOf(p string) string {
	d := filepath.Dir(p)
	for {
		if st, err := os.Stat(filepath.Join(d, "mta")); err == nil && st.IsDir() {
			return d
		}
		parent := filepath.Dir(d)
		if parent == d {
			return ""
		}
		d = parent
	}
}

// scratchPath 是本测试的临时产出落点。
//
// 必须落在**源包旁边**（也就是工作区之内）：把包放到 %TEMP% 再喂给 RoundTrip.exe，实测得到的是
// `NotInCurrentWorkspaceException: … 不在工作目录之下，禁止操作`（SUMMARY 行的 status=FAIL）。
// 所以它写在源包旁边、名字带 _tdev_ 前缀（于是既不算语料，跑崩留下也不会污染下一轮的发现）、
// 用完就删 —— 旧实现也是这么做的，只是它的前缀是 _ai/_dw/…。
func scratchPath(src string, idx int, op, tag string) string {
	return filepath.Join(filepath.Dir(src),
		fmt.Sprintf("_tdev_%d_%03d_%s_%s.tzs", os.Getpid(), idx, op, tag))
}

//---------------------------------------------------------------------------
// RoundTrip 与那条判据
//---------------------------------------------------------------------------

// rtSummary 是 RoundTrip.exe 那一行 `SUMMARY|` 的解析结果。
type rtSummary struct {
	Status                                     string
	Env, Tpl                                   string
	TsdAdds, TsdDrops, FdPathAdds, FdPathDrops int
	Stale, NodesIn, NodesOut, LayoutElems      int
	Raw                                        string
}

// parseRt 解那一行 SUMMARY|。列序被四个 batch 脚本冻住了（batch.sh 顶部注释）：
//
//	SUMMARY|status|env|tpl|tsdAdds|tsdDrops|fdPathAdds|fdPathDrops|stale|tsdNodesIn|tsdNodesOut|layoutElems|sample|path|error
//
// status != ok 时引擎写的是另一种列数（中间一长串空列），所以数字那一段只在长度够的时候按
// 固定下标取 —— 拿一列错位的数字去做判据，说出来的结论会和真相毫无关系。
func parseRt(line string) *rtSummary {
	s := &rtSummary{Raw: line}
	f := strings.Split(line, "|")
	if len(f) < 12 {
		return s
	}
	s.Status = f[1]
	s.Env, s.Tpl = f[2], f[3]
	s.TsdAdds = atoiOr(f[4], -1)
	s.TsdDrops = atoiOr(f[5], -1)
	s.FdPathAdds = atoiOr(f[6], -1)
	s.FdPathDrops = atoiOr(f[7], -1)
	s.Stale = atoiOr(f[8], -1)
	s.NodesIn = atoiOr(f[9], -1)
	s.NodesOut = atoiOr(f[10], -1)
	s.LayoutElems = atoiOr(f[11], -1)
	return s
}

func atoiOr(s string, def int) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return def
	}
	return n
}

// runRoundTrip 跑一次 engine/out/RoundTrip.exe 并返回 SUMMARY 行。任何一步不成立都报错并返回 nil。
//
// 预算是每包的：RoundTrip 是另一个进程，stdio 那条预算管不到它。
func runRoundTrip(t *testing.T, env *corpusEnv, ws, path, label string) *rtSummary {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), perFileBudget)
	defer cancel()
	cmd := exec.CommandContext(ctx, filepath.Join(env.dir, "RoundTrip.exe"), path)
	cmd.Env = engineEnv(ws)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if ctx.Err() != nil {
		t.Errorf("%s: RoundTrip 超过 %s 没有结束（已杀）", label, perFileBudget)
		return nil
	}
	line := ""
	for _, ln := range strings.Split(stdout.String(), "\n") {
		if strings.HasPrefix(ln, "SUMMARY|") {
			line = strings.TrimRight(ln, "\r")
		}
	}
	if line == "" {
		t.Errorf("%s: RoundTrip 没有 SUMMARY 行（exit=%v）；stderr 末尾：\n%s",
			label, err, tailOf(stderr.String(), 600))
		return nil
	}
	s := parseRt(line)
	if s.Status != "ok" {
		t.Errorf("%s: RoundTrip 自己就不是 ok —— %s", label, line)
		return nil
	}
	return s
}

// requireFixedPoint 是那条判据本身（旧 batch.sh 的四个零 + gate-w3.py 的 rt_ok）。
//
// base == nil 表示「这一份是 pristine 自己的度量」：此时只要求 tsd/paths 的增删为 0，
// **不要求 stale 为 0**。stale 说的是「模型记的属性值与 .4fd 里存的值差几个」，而
// capt111(3)/cpmp530(2)/cpmq001(2) 三个包在**没人动过**时就是这样 —— 这是它们本来就有的
// 性质。要求它等于 0 正是旧关卡第一版犯过的那个错（「A gate that fails a file for a
// pre-existing property is worse than no gate」），所以 base==nil 时它只被**记下来**当基线。
//
// base != nil 时（对产出的判定）stale 必须**等于基线**，不是等于 0。
func requireFixedPoint(t *testing.T, label string, got, base *rtSummary) bool {
	t.Helper()
	if got == nil {
		return false
	}
	ok := true
	for _, c := range []struct {
		name string
		got  int
	}{
		{"tsdAdds", got.TsdAdds}, {"tsdDrops", got.TsdDrops},
		{"fdPathAdds", got.FdPathAdds}, {"fdPathDrops", got.FdPathDrops},
	} {
		if c.got != 0 {
			t.Errorf("%s: %s=%d（必须是 0：设计器的模型得把它拿到的每个 spec 节点与布局路径原样还回来）",
				label, c.name, c.got)
			ok = false
		}
	}
	if base == nil {
		return ok
	}
	// 比的是**不比基线差**（`<=`），不是"等于基线"。
	//
	// 写成 `!=`（2026-09-27 之前的版本）与它自己的文案矛盾：上面那句一直说的是"判据是
	// 「不比 pristine 差」"，而 `!=` 实现的是"等于"。差的这一半不是学术问题 —— 实测过一次：
	// `wrap` 在 capt111 上把 stale 从 3 降到 2，也就是**把一条本来就存在的不一致写好了**
	// （这条包没人动过时就有 stale，见下面），而 `!=` 会把"变好"判成失败。
	//
	// 为什么"变少"可以接受：stale 量的是"模型记的属性值与文件里存的对不上几个"。一个写
	// 把某个属性在两边写成同一个值，本来就是它该有的效果，结果比 pristine 更接近不动点。
	//
	// 什么在兜住"变少其实是丢了东西"那种可能：**紧挨着上面的四个零** —— 产出既没有多出或
	// 少掉 spec 节点（tsd*），也没有多出或少掉布局路径（fdPath*），所以一条 stale 的减少
	// 只可能是属性值收敛，不可能是"把带这个属性的东西删掉了"。
	if got.Stale > base.Stale {
		t.Errorf("%s: stale=%d，基线是 %d。判据是「不比 pristine 差」，不是「等于 0」："+
			"capt111/cpmp530/cpmq001 三个包在没人动过时就有 stale，硬编码 0 会为了一个**本来就存在**的"+
			"性质判它们失败 —— 那比没有关卡更糟（旧关卡的原话）",
			label, got.Stale, base.Stale)
		ok = false
	}
	return ok
}

//---------------------------------------------------------------------------
// pin：语料本身不许在回归底下悄悄变
//---------------------------------------------------------------------------

// pinRow 是 corpus.manifest 的一行。
type pinRow struct {
	Sha16 string
	Bytes int64
	Path  string // Windows 形式（清单里是 MSYS 的 /d/…）
}

// pinPath 是固定语料清单的落点。相对包目录（internal/dev/tzs）解析 —— 与 requireCorpus 的
// 缺省 exe 同一种省事办法。
func pinPath() string { return filepath.Join("..", "..", "..", "engine", "corpus.manifest") }

// msysToWindows 把 make-manifest.sh 写下的 MSYS 路径换成本机的 Windows 路径。
//
// 只处理 `/x/…` 这一种（`/d/t100_wrok_dir/...` → `D:\t100_wrok_dir\...`）：清单是那个脚本在
// Git Bash 里生成的，而 Go 测试跑在 Windows 上。两种写法之间必须只有**一个**转换点，
// 否则「清单里的路径」与「发现出来的路径」会各自解释，比对结果就没有意义了。
func msysToWindows(p string) string {
	if len(p) >= 3 && p[0] == '/' && p[2] == '/' {
		return strings.ToUpper(p[1:2]) + ":" + strings.ReplaceAll(p[2:], "/", `\`)
	}
	return p
}

func readCorpusPin(t *testing.T, path string) []pinRow {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读不了 %s：%v（它是 engine/make-manifest.sh 生成的固定清单）", path, err)
	}
	var out []pinRow
	for i, ln := range strings.Split(string(raw), "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" || strings.HasPrefix(ln, "#") {
			continue
		}
		f := strings.Fields(ln)
		if len(f) < 3 {
			t.Fatalf("%s 第 %d 行不是「sha 字节数 路径」：%q", path, i+1, ln)
		}
		n, err := strconv.ParseInt(f[1], 10, 64)
		if err != nil {
			t.Fatalf("%s 第 %d 行的字节数不是数字：%q", path, i+1, f[1])
		}
		// 路径里有空格时 Fields 会把它切碎：以第三个字段开头、剩下的接回去。
		out = append(out, pinRow{Sha16: f[0], Bytes: n,
			Path: msysToWindows(strings.Join(f[2:], " "))})
	}
	if len(out) == 0 {
		t.Fatalf("%s 里一行数据都没有", path)
	}
	return out
}

// TestCorpusPin 钉住语料的字节。
//
// corpus.manifest 是那 67 个包的一份固定清单（sha256 前 16 位 + 字节数 + 路径），它的文件头
// 注释自己讲了为什么要有它：驱动脚本的草稿文件会在两次运行之间出现或消失，没有一份固定清单，
// 基线不可复现。任务是「保留它当 pin」，而**只有把它当判据用**才算钉住 ——
// 没有这条断言，语料会在回归底下悄悄变，而所有判据都跟着悄悄搬家：判据本身还是绿的，
// 因为它从来只在问「和刚才一样吗」。
//
// 只需要文件系统，不需要引擎（所以它跑得飞快 —— 但仍然是语料测试，同样归 TTZS_DEEP）。
func TestCorpusPin(t *testing.T) {
	requireDeep(t)
	root := testutil.CorpusRoot()
	if root == "" {
		t.Skipf("没有真实语料：设 TTZS_CORPUS（或 TDEV_CORPUS），或准备 %s", testutil.DefaultCorpusRoot)
	}
	pins := readCorpusPin(t, pinPath())
	disc := testutil.CorpusFiles(root, ".tzs", scratchPrefixes...)
	if len(disc) == 0 {
		t.Skipf("%s 下没有 .tzs", root)
	}

	byPath := map[string]pinRow{}
	onDisk := 0
	for _, r := range pins {
		byPath[strings.ToLower(r.Path)] = r
		if _, err := os.Stat(r.Path); err == nil {
			onDisk++
		}
	}
	if onDisk == 0 {
		// 清单里的路径是绝对的（写清单那次的工作区）。语料根被指到别处时它们一条都对不上，
		// 这时拿另一份语料去对这份 pin 的 sha 毫无意义 —— 说清并跳过，不是静默通过。
		t.Skipf("pin 里 %d 条路径在本机一条都不存在（语料根=%s）：pin 描述的是另一份语料，"+
			"对不上就是「不适用」，不是「通过」", len(pins), root)
	}

	discSet := map[string]bool{}
	for _, p := range disc {
		discSet[strings.ToLower(p)] = true
	}
	var missing, extra []string
	for _, r := range pins {
		if !discSet[strings.ToLower(r.Path)] {
			missing = append(missing, filepath.Base(r.Path))
		}
	}
	for _, p := range disc {
		if _, ok := byPath[strings.ToLower(p)]; !ok {
			extra = append(extra, filepath.Base(p))
		}
	}
	if len(missing) > 0 || len(extra) > 0 {
		t.Errorf("发现出来的语料与 pin 对不上：pin 里有 %d 个没找到%s，多出 %d 个不在 pin 里%s。\n"+
			"  要么语料真的变了、要么排除前缀变了 —— 两种都该显式重新生成 pin（engine/make-manifest.sh [root]），"+
			"而不是让回归在一批已经换过的输入上继续跑",
			len(missing), clipList(missing), len(extra), clipList(extra))
	}
	if len(disc) != len(pins) {
		t.Errorf("语料数量变了：磁盘上 %d 个、pin 里 %d 个", len(disc), len(pins))
	}

	bad := 0
	for _, r := range pins {
		st, err := os.Stat(r.Path)
		if err != nil {
			continue // missing 已经报过了
		}
		if st.Size() != r.Bytes {
			t.Errorf("%s：字节数 %d，pin 里是 %d", filepath.Base(r.Path), st.Size(), r.Bytes)
			bad++
		}
		got, err := sha16(r.Path)
		if err != nil {
			t.Errorf("%s：算 sha256 失败 %v", r.Path, err)
			bad++
			continue
		}
		if got != r.Sha16 {
			t.Errorf("%s：sha256 前 16 位是 %s，pin 里是 %s —— 语料在回归底下变了",
				filepath.Base(r.Path), got, r.Sha16)
			bad++
		}
	}
	if bad == 0 {
		t.Logf("pin：%d 条全部与磁盘一致（sha256-16 + 字节数）", len(pins))
	}
}

// sha16 取 sha256 的前 16 位十六进制（与 make-manifest.sh 的 cut -c1-16 同一个口径）。
func sha16(path string) (string, error) {
	full, err := sha256full(path)
	if err != nil {
		return "", err
	}
	return full[:16], nil
}

// sha256full 是整条 64 位十六进制摘要 —— 引擎 `save` 的返回里给的就是它。
func sha256full(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

//---------------------------------------------------------------------------
// 四个语料用例共用的那一条流程
//---------------------------------------------------------------------------

// corpusOp 是一次语料回归里的「那一件事」。
//
// 四个用例共用同一条流程（pristine 基线 → 一个 --stdio 进程做一件事 → 产出跑 RoundTrip），
// 差别只有 prepare 一处，所以流程只写一遍。写四遍的代价不是行数，而是四条会各自漂移的流程
// —— 比如给其中一条忘了 validate 的 baseline，那条的断言就永远是绿的。
type corpusOp struct {
	name string
	// key 是这个名字的 ASCII 版，只用在临时文件名里（旧实现的临时文件也全是 ASCII 前缀）。
	key string
	// prepare 在**起进程之前**做进程外的挑选（读包里的 .tsd / .4fd、读工作区的数据字典）。
	// 它返回「进进程后要做什么」的闭包，或者一个跳过原因（非空即跳过，且必须说清缺什么）。
	// 返回 nil 闭包 = 进去只 open → save。
	prepare func(t *testing.T, idx int, src, ws string) (do func(*stdioSession, string) string, skip string)
	// validate 为 true 时在这个包上调两次 validate（第一次是 baseline，见 validate 的注释）。
	validate bool
}

type passStats struct {
	applied, skipped, failed int
	frames, dropped          int
}

func corpusPass(t *testing.T, env *corpusEnv, op corpusOp) {
	t.Helper()
	var st passStats
	for i, src := range env.pkgs {
		name := filepath.Base(src)
		ws := workspaceOf(src)
		if ws == "" {
			st.skipped++
			t.Logf("跳过 %s：祖先目录里没有 mta/，推不出工作区（本用例不替它选 —— 引擎内置的缺省是一个真实客户目录）", name)
			continue
		}
		var do func(*stdioSession, string) string
		if op.prepare != nil {
			var why string
			do, why = op.prepare(t, i, src, ws)
			if why != "" {
				st.skipped++
				t.Logf("跳过 %s：%s", name, why)
				continue
			}
		}
		st.applied++
		oneCorpusFile(t, env, op, i, src, ws, do, &st)
	}
	t.Logf("%s：%d 个包跑过、%d 个跳过、%d 个失败；共 %d 帧请求（丢掉 %d 行不像应答的输出）",
		op.name, st.applied, st.skipped, st.failed, st.frames, st.dropped)
	if st.applied == 0 {
		t.Fatalf("%s：一个包都没跑起来（%d 个跳过）—— 判据或前置条件写错了，不是语料的问题", op.name, st.skipped)
	}
}

// oneCorpusFile 是一个包的全程。
func oneCorpusFile(t *testing.T, env *corpusEnv, op corpusOp, idx int, src, ws string,
	do func(*stdioSession, string) string, st *passStats) {

	t.Helper()
	label := op.name + "/" + filepath.Base(src)

	// 先量一遍没人动过的它。判据必须基线相对（见文件头教训 1），而「相对」的落地方式就是
	// 这一句 —— 不存基线表，每个包自己带自己的标尺。没有这一句，三个本来就 stale 的包
	// （capt111=3 / cpmp530=2 / cpmq001=2）会让整轮变成一片红。
	base := runRoundTrip(t, env, ws, src, label+" pristine")
	if base == nil {
		st.failed++
		return
	}
	if !requireFixedPoint(t, label+" pristine", base, nil) {
		st.failed++
		return
	}
	// 把 pristine 的四个数记下来：报告里要看得见「这一包自己是什么样」（节点数、布局元素数、
	// 以及它是不是那三个本来就 stale 的包之一），否则一条红只能说「某处不对」。
	t.Logf("%s: pristine env=%s tpl=%s spec节点 %d→%d 布局元素 %d stale=%d",
		label, base.Env, base.Tpl, base.NodesIn, base.NodesOut, base.LayoutElems, base.Stale)

	post := scratchPath(src, idx, op.key, "post")
	defer os.Remove(post)
	pre := ""
	if do != nil {
		// 「什么都没干」的对照。没有它，一个静默空操作（写函数 ok:true 却什么都没写）会让
		// 这一包在下面两条断言上全绿 —— 产出是定点，validate 零新增 —— 因为产出和被保存的
		// 源包没差别。gate-w3-fns 用同一招（save(pre) vs save(post) 必须不同）。
		pre = scratchPath(src, idx, op.key, "pre")
		defer os.Remove(pre)
	}

	s := startStdio(t, env.exe, ws, perFileBudget)
	defer func() {
		// 计数器在收摊时统一结算，于是提前 return 的那几条路也不会漏计。
		st.frames += s.sent
		st.dropped += s.dropped
		s.close()
	}()

	opened, ok := ask[openReply](t, s, label+" open", "open", map[string]any{"path": src})
	if !ok || opened.Handle == "" {
		st.failed++
		return
	}
	if opened.State != "Loaded" {
		t.Errorf("%s: open 之后 state=%q，该是 Loaded", label, opened.State)
	}
	h := opened.Handle

	// validate 的第一次调用**就是** baseline（见 validate 的注释），所以它必须在 op 之前。
	deep := op.validate && wantsValidate(t, filepath.Base(src))
	if deep {
		if v := validate(t, s, label+" validate(baseline)", h); v == nil {
			st.failed++
			return
		} else if n := len(v.NewErrors) + len(v.NewWarnings); n > 0 {
			// 首次调用下 newErrors 必然是空的，这里只是把「这个包本来就报什么」记下来。
			t.Logf("%s: pristine 就报 %d 条（after=%d）", label, n, len(v.After))
		}
	}

	if do != nil {
		if !saveTo(t, s, label+" save(pre)", h, pre) {
			st.failed++
			return
		}
		// do 自己报哪些断言失败；这里只把它挑中的目标记下来 —— 报告里要看得见「这一包
		// 实际动的是哪个字段」，否则一条红只会说「某处不对」。
		if detail := do(s, h); detail != "" {
			t.Logf("%s: %s", label, detail)
		}
	}

	if deep {
		v := validate(t, s, label+" validate(after)", h)
		if v == nil {
			st.failed++
			return
		}
		if len(v.NewErrors) != 0 {
			t.Errorf("%s: 改动后冒出 %d 个新 ERROR（判据是**新增**必须为 0，不是「一条都没有」）：%s",
				label, len(v.NewErrors), dumpFindings(v.NewErrors))
		}
	}

	if !saveTo(t, s, label+" save(post)", h, post) {
		st.failed++
		return
	}

	if pre != "" && !mustDiffer(t, label, pre, post) {
		st.failed++
		return
	}
	if got := runRoundTrip(t, env, ws, post, label+" post"); got != nil {
		if !requireFixedPoint(t, label, got, base) {
			st.failed++
		}
	} else {
		st.failed++
	}
}

// saveTo 让引擎把句柄的模型写回 out，并**就地断言两条**：返回的 sha256 就是盘上那份文件的
// 摘要，且返回里带着新包的 key。
//
// 为什么在这条最热的路径上顺手验（它每个包都会跑一遍）：这两个字段的存在理由，就是让调用方
// **不必**再开三次引擎调用去证明"改动落了盘"（2026-09-25 的评测 F10：回读要 save → close →
// open 新包 → 读，而 get_component / verify 读的都是内存模型，拿它们回读是自证循环）。
// 一个"我写了 X"的摘要如果不能与盘上的字节对上，它比没有更坏 —— 所以这条断言与那两个字段
// 是同一次改动的一部分，不是锦上添花。
func saveTo(t *testing.T, s *stdioSession, label, h, out string) bool {
	t.Helper()
	rep, ok := ask[saveReply](t, s, label, "save", map[string]any{"handle": h, "out": out})
	if !ok {
		return false
	}
	if rep.Sha256 == "" {
		t.Errorf("%s: save 的返回里没有 sha256", label)
		return false
	}
	disk, err := sha256full(out)
	if err != nil {
		t.Errorf("%s: 读产出算摘要失败：%v", label, err)
		return false
	}
	if disk != rep.Sha256 {
		t.Errorf("%s: save 说写了 %s…，盘上却是 %s… —— 摘要与文件不符", label, rep.Sha256[:16], disk[:16])
		return false
	}
	if rep.Key == "" {
		t.Errorf("%s: save 的返回里没有 key（新包沿用源包 ProgramKey，回读它要先 close，这个字段就是给那件事的）", label)
		return false
	}
	return true
}

// mustDiffer 断言两个产出不同 —— 即那一次写真的落到了文件上。
//
// 两次 save 的内容在引擎里是确定性的（同一个进程、同一个模型），所以「不同」只可能来自
// 中间那一次写。这一条是这批用例的非空转保证：写函数静默失败时，RoundTrip 判据与 validate
// 判据都会全绿。
func mustDiffer(t *testing.T, label, a, b string) bool {
	t.Helper()
	sa, err1 := sha16(a)
	sb, err2 := sha16(b)
	if err1 != nil || err2 != nil {
		t.Errorf("%s: 算产出 sha256 失败：%v / %v", label, err1, err2)
		return false
	}
	if sa == sb {
		t.Errorf("%s: 写前与写后的产出逐字节相同（%s）—— 那一次写是个静默的空操作，"+
			"后面的定点与 validate 断言在空操作上也全绿，所以这一条必须先失败", label, sa)
		return false
	}
	return true
}

//---------------------------------------------------------------------------
// 进程内用到的、引擎自己的那些调用
//---------------------------------------------------------------------------

// openReply 是 open 的返回（只列我们读的字段）。
type openReply struct {
	Handle  string `json:"handle"`
	Path    string `json:"path"`
	Program string `json:"program"`
	Key     string `json:"key"`
	State   string `json:"state"`
}

// saveReply 是 save 的返回（只列我们读的字段）。
type saveReply struct {
	Path        string `json:"path"`
	Out         string `json:"out"`
	BytesIn     int    `json:"bytesIn"`
	BytesOut    int    `json:"bytesOut"`
	Sha256      string `json:"sha256"`
	Key         string `json:"key"`
	LayoutDirty bool   `json:"layoutDirty"`
	State       string `json:"state"`
}

// elementRef 是「产生了元素」的那几个写函数（add_field / add_widget / insert_at…）的返回形状。
//
// 只列我们读的字段：剩下的留给引擎。在 Go 侧把它的字段抄全，就是在制造第二份会漂的定义
// （manifest.go 的头注释是同一条理由）。
type elementRef struct {
	Path         string       `json:"path"`
	Name         string       `json:"name"`
	Tag          string       `json:"tag"`
	SpecNodeType string       `json:"specNodeType"`
	Promoted     int          `json:"promoted"`
	Added        []elementRef `json:"added"`
	Table        string       `json:"table"`
	Column       string       `json:"column"`
}

// deltaRef 是 set_spec_attr / set_layout_attr 的返回形状。
//
// 一处**必须记住的不对称**（实测，不是推测）：两个函数的「正标记」不是同一个东西。
//
//	set_layout_attr  写了 → applied:true, changed:true；同值 → noop:true, changed:false
//	set_spec_attr    写了 → 只有 layoutDelta.attrs（**没有** changed 字段）；
//	                 同值 → noop:true, changed:false（Noop() 会写 changed）
//
// 所以 `!changed` 在两个分支上都成立，拿它当「写没写进去」的判据在 set_spec_attr 上恒真。
// set_spec_attr 这一侧的判据只能是「不是 noop」。
//
// 也**不要**把 layoutDelta.attrs 当成「写没写」的判据：实测 67 个包里有 33 个的
// set_spec_attr(can_edit) 根本没动 .4fd（attrs 为空），而它们每一次都真的写进去了
// （.tsd 侧 specStatus=u，且写前写后的产出 sha 不同）。
type deltaRef struct {
	Path        string         `json:"path"`
	Kind        string         `json:"kind"`
	Attr        string         `json:"attr"`
	Old         string         `json:"old"`
	Value       string         `json:"value"`
	Written     string         `json:"written"`
	SpecStatus  string         `json:"specStatus"`
	Applied     bool           `json:"applied"`
	Changed     bool           `json:"changed"`
	Noop        bool           `json:"noop"`
	LayoutDelta layoutDeltaRef `json:"layoutDelta"`
}

// layoutDeltaRef 是「布局侧真的动了」的那一段证据（Fns/Attr.cs 的 Delta）。
type layoutDeltaRef struct {
	Path   string            `json:"path"`
	Attrs  map[string]string `json:"attrs"`
	Rename string            `json:"rename"`
}

// validateItem 是 validate 报告里的一条发现（Fns/Validate.cs 的 Rec2Obj：type/key/description）。
type validateItem struct {
	Type        string `json:"type"`
	Key         string `json:"key"`
	Description string `json:"description"`
}

type validateDelta struct {
	Baseline    []validateItem `json:"baseline"`
	After       []validateItem `json:"after"`
	NewErrors   []validateItem `json:"newErrors"`
	NewWarnings []validateItem `json:"newWarnings"`
	ElapsedMs   int64          `json:"elapsedMs"`
}

// validate 跑一次 validate。
//
// **它是有状态的**：一个句柄上第一次调用 validate，这一次的输出就直接成了 baseline
// （Fns/Validate.cs 的 _baseline 缓存），于是 newErrors 恒为空。所以「改动后零新增 error」
// 必须在**同一个进程、同一个句柄**上先量一遍 pristine —— 只调一次的话那条断言是恒真的。
//
// （旧实现 gate-w3.py 段 B 就踩了这个：它的两个 payload 是两个 `--stdio` 进程，第二个进程里
// 的 validate 又成了一次「首次调用」，所以那句 newErrors must be 0 从来没有机会失败。
// 这是照着旧代码抄判据时最容易漏掉的一处，记在这里。）
func validate(t *testing.T, s *stdioSession, label, handle string) *validateDelta {
	t.Helper()
	v, ok := ask[validateDelta](t, s, label, "validate", map[string]any{"handle": handle})
	if !ok {
		return nil
	}
	return &v
}

// treeNode 是 form_tree 的一个节点（只列我们读的字段）。
type treeNode struct {
	Tag          string      `json:"tag"`
	Name         string      `json:"name"`
	Path         string      `json:"path"`
	SpecNodeType string      `json:"specNodeType"`
	Children     []*treeNode `json:"children"`
}

type formTreeReply struct {
	Root         *treeNode `json:"root"`
	ElementCount int       `json:"elementCount"`
}

// formTree 取表单结构树。
//
// 落点/插入目标都用它而不是自己解析 .4fd：add_widget / add_field 收的是 name-path，而路径的
// 定义在引擎的 ElementIndex 里 —— 自己拼就多一处会漂的第二实现，而它漂了的表现是
// 「这个元素不存在」，一句与真因无关的话。
func formTree(t *testing.T, s *stdioSession, label, handle string) *treeNode {
	t.Helper()
	tr, ok := ask[formTreeReply](t, s, label+" form_tree", "form_tree",
		map[string]any{"handle": handle, "depth": 99})
	if !ok {
		return nil
	}
	if tr.Root == nil {
		t.Errorf("%s: form_tree 没有 root（elementCount=%d）", label, tr.ElementCount)
		return nil
	}
	return tr.Root
}

// dropContainers 是设计器自己的落点白名单，照 test/AddField.cs 的 FindAutoTarget 逐字保留。
var dropContainers = []string{"Grid", "Group", "Folder", "Page", "HBox", "VBox"}

// dropTarget 照 test/AddField.cs 的 @auto 挑落点：第一个**有子元素**的 Grid，
// 找不到再在白名单里找第一个有子元素的。
//
// 复刻旧实现的行为而不是自己发明规则：落点换了，67 个包走的代码路径就跟着换，
// 而那条路径正是这批用例要覆盖的东西。
func dropTarget(root *treeNode) *treeNode {
	var all []*treeNode
	walkTree(root, &all)
	for pass := 0; pass < 2; pass++ {
		for _, n := range all {
			if len(n.Children) == 0 {
				continue
			}
			if pass == 0 {
				if n.Tag != "Grid" {
					continue
				}
			} else if !inList(dropContainers, n.Tag) {
				continue
			}
			return n
		}
	}
	return nil
}

// firstPopulatedGrid 是 add_widget Button 的落点（mime 门禁只允许 Grid/Group 收 Button），
// 与 pick-edit-target.py 的 add 模式同一条挑选。
func firstPopulatedGrid(root *treeNode) *treeNode {
	var all []*treeNode
	walkTree(root, &all)
	for _, n := range all {
		if n.Tag == "Grid" && len(n.Children) > 0 {
			return n
		}
	}
	return nil
}

func walkTree(n *treeNode, acc *[]*treeNode) {
	if n == nil {
		return
	}
	*acc = append(*acc, n)
	for _, c := range n.Children {
		walkTree(c, acc)
	}
}

func inList(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}

func dumpFindings(items []validateItem) string {
	b, _ := json.Marshal(items)
	return clip(b)
}

func clipList(names []string) string {
	if len(names) == 0 {
		return ""
	}
	const max = 6
	if len(names) <= max {
		return "（" + strings.Join(names, ", ") + "）"
	}
	return fmt.Sprintf("（%s … 共 %d 个）", strings.Join(names[:max], ", "), len(names))
}

//---------------------------------------------------------------------------
// 用例 1：语料 RoundTrip 定点（旧 batch.sh）
//---------------------------------------------------------------------------

// TestCorpusRoundTrip 证明「设计器读得进每一个包，并且用它自己的模型存出来的东西还是同一个定点」。
//
// 每包：pristine 量基线 → 一个 --stdio 进程 open → save → 对**产出**跑 RoundTrip.exe。
// 判据见 requireFixedPoint（四个零 + stale 等于 pristine 的 stale）。
func TestCorpusRoundTrip(t *testing.T) {
	env := requireCorpus(t)
	corpusPass(t, env, corpusOp{name: "roundtrip", key: "roundtrip"})
}

//---------------------------------------------------------------------------
// 用例 2：语料写路径（旧 batch-write.sh）
//---------------------------------------------------------------------------

// writeContainers 是 add_field 的容器模式轮换表，照 batch-write.sh 的 TYPES 逐字保留：
// 每个包走一条分支，而不是让 67 个包都走同一条。
var writeContainers = []string{"None", "Grid", "Group", "ScrollGrid", "Table", "Tree"}

func writeOp() corpusOp {
	return corpusOp{
		name:     "写路径",
		key:      "write",
		validate: true,
		prepare: func(t *testing.T, idx int, src, ws string) (func(*stdioSession, string) string, string) {
			tbl, col, why := pickColumn(t, src, ws)
			if why != "" {
				return nil, why
			}
			container := writeContainers[idx%len(writeContainers)]
			label := filepath.Base(src)
			return func(s *stdioSession, h string) string {
				root := formTree(t, s, label, h)
				if root == nil {
					return ""
				}
				drop := dropTarget(root)
				if drop == nil {
					t.Errorf("%s: 布局里没有可落点的容器（有子元素的 Grid/Group/Folder/Page/HBox/VBox 一个都没有）", label)
					return ""
				}
				el, ok := ask[elementRef](t, s, label+" add_field", "add_field", map[string]any{
					"handle": h, "path": drop.Path, "table": tbl, "column": col, "container": container,
				})
				if !ok {
					return ""
				}
				if len(el.Added) == 0 {
					t.Errorf("%s: add_field ok:true 但 added 是空的 —— 引擎的「正标记」没了，这次调用可能什么都没做", label)
					return ""
				}
				return fmt.Sprintf("add_field %s.%s container=%s 落在 %s（+%d 个元素）",
					tbl, col, container, drop.Name, len(el.Added))
			}, ""
		},
	}
}

func TestCorpusWritePath(t *testing.T) {
	env := requireCorpus(t)
	corpusPass(t, env, writeOp())
}

// pickColumn 挑一个「能变成控件、而且还没在这张表单里」的列（照 engine/pick-column.py）。
//
// 为什么必须挑**没用过**的列：复用一个已经是字段的列，走的是命名冲突那条路，而这条用例要的是
// 普通那条路 —— 两者混起来不会失败，只会让这条用例在测别的东西。
//
// 为什么跳过 widget 为空的列：UICreator 从列元数据（<col_attr> 里的 <field widget=…>）读控件
// 类型与宽度，没有它的列设计器自己都拖不动（会在 ComponentFactory 里 NRE）。type_t 那张
// 「数据类型参照表」整张都是这样的列。
//
// 读法与旧实现一致：表名来自 .4fd 里 <RecordField> 的 sqlTabName（首次出现序），
// 「已经用过」的是它的 colName；列元数据来自工作区里 <table>.tbl 的 <col_attr> 块。
func pickColumn(t *testing.T, src, ws string) (table, column, why string) {
	t.Helper()
	fd, err := zipEntryText(src, ".4fd")
	if err != nil {
		return "", "", "读不到包里的 .4fd：" + err.Error()
	}
	var tables []string
	used := map[string]bool{}
	for _, m := range reRecordField.FindAllString(fd, -1) {
		if c := reAttr("colName").FindStringSubmatch(m); c != nil && c[1] != "" {
			used[c[1]] = true
		}
		if v := reAttr("sqlTabName").FindStringSubmatch(m); v != nil && v[1] != "" && !inList(tables, v[1]) {
			tables = append(tables, v[1])
		}
	}
	if len(tables) == 0 {
		return "", "", "包里的 .4fd 没有 <RecordField>（没有数据源表，add_field 无处可加）"
	}
	byTable := tblIndex(t, ws)
	if len(byTable) == 0 {
		return "", "", "工作区里一个 <table>.tbl 都没有（列元数据不在，add_field 无从挑列）"
	}
	for _, tb := range tables {
		path := byTable[tb]
		if path == "" {
			continue
		}
		text, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		block := reColAttr.FindStringSubmatch(string(text))
		if block == nil {
			continue
		}
		for _, m := range reColField.FindAllStringSubmatch(block[1], -1) {
			if m[2] != "" && !used[m[1]] {
				return tb, m[1], ""
			}
		}
	}
	return "", "", "这张表单的每一张数据源表都挑不出「有控件类型、且还没被用作字段」的列"
}

// tblIndex 把工作区里所有 <module>/tbl/<table>.tbl 建成 name → 路径。
//
// 走 cron 的字典而不是硬编码模块路径：工作区的形状是模块自己的事（batch.sh 的注释说得很清楚，
// 「a module directory is a workspace because of that mta/ directory」）。
func tblIndex(t *testing.T, ws string) map[string]string {
	t.Helper()
	out := map[string]string{}
	_ = filepath.Walk(ws, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if filepath.Base(filepath.Dir(p)) != "tbl" || !strings.EqualFold(filepath.Ext(p), ".tbl") {
			return nil
		}
		name := strings.TrimSuffix(filepath.Base(p), filepath.Ext(p))
		if _, dup := out[name]; !dup {
			out[name] = p
		}
		return nil
	})
	return out
}

//---------------------------------------------------------------------------
// 用例 3：语料改路径（旧 batch-edit.sh）
//---------------------------------------------------------------------------

func editOp() corpusOp {
	return corpusOp{
		name:     "改路径",
		key:      "edit",
		validate: true,
		prepare: func(t *testing.T, idx int, src, ws string) (func(*stdioSession, string) string, string) {
			// 挑选本身不需要引擎（can_edit 在 .tsd 里），所以它在起进程之前做 ——
			// 缺前置的包于是能在「一个进程都没起」的时候就被说清。
			cands, why := editCandidates(t, src)
			if why != "" {
				return nil, why
			}
			label := filepath.Base(src)
			return func(s *stdioSession, h string) string {
				name, path, was, flip, ok := resolveEditTarget(t, s, label, h, cands)
				if !ok {
					return ""
				}
				d, ok := ask[deltaRef](t, s, label+" set_spec_attr", "set_spec_attr", map[string]any{
					"handle": h, "path": path, "kind": "field", "attr": "can_edit", "value": flip,
				})
				if !ok {
					return ""
				}
				if d.Old != was {
					t.Errorf("%s: 引擎说旧值是 %q，我们读到的是 %q —— 挑选与模型对不上（读的是另一个包？）",
						label, d.Old, was)
				}
				// 「写进去了没有」的判据见 deltaRef 的注释：set_spec_attr 这一侧的 changed
				// 只在 noop 分支上出现（且为 false），所以判据是「不是 noop」。
				//
				// **不要**再去要求 layoutDelta.attrs 非空：实测 67 个包里只有 34 个的 can_edit
				// 真的动了 .4fd 那一侧（33 个是空的，`布局侧 map[]`）。旧实现
				// pick-edit-target.py 的 docstring 说 can_edit 一定映射到 noEntry（「so the run
				// exercises both sides of the write」），实测**不是**这样 —— 那条映射与
				// code_template 有关（Fns/Verify.cs:192 的注释是同一条线索）。attrs 空不代表
				// 这次写没发生：.tsd 侧的 specStatus=u 与下面 mustDiffer 的 sha 比对都证明它发生了。
				if d.Noop {
					t.Errorf("%s: set_spec_attr 回了 noop（%q 已经是 %q）—— 挑选保证翻转，"+
						"出现 noop 说明读到的旧值与模型里的不是同一份", label, d.Attr, d.Value)
				}
				return fmt.Sprintf("can_edit %s %s→%s（specStatus=%s，布局侧 %v）",
					name, was, flip, d.SpecStatus, d.LayoutDelta.Attrs)
			}, ""
		},
	}
}

func TestCorpusEditPath(t *testing.T) {
	env := requireCorpus(t)
	corpusPass(t, env, editOp())
}

// editCand 是一个候选：spec 字段名 + 它当前的 can_edit。
type editCand struct {
	Name    string
	CanEdit string
}

// editCandidates 从 .tsd 里读「带 can_edit 的 <field>」，按文档序（同名取最后一个，照
// pick-edit-target.py 的字典写法：属性在文件里可能出现两次）。
//
// 为什么挑 can_edit 这个属性：它同时映射到一个布局属性（noEntry），所以这一次写会同时走到
// .tsd 与 .4fd 两侧，比只改一侧的属性覆盖得更全（旧实现的原话）。
func editCandidates(t *testing.T, src string) ([]editCand, string) {
	t.Helper()
	ts, err := zipEntryText(src, ".tsd")
	if err != nil {
		return nil, "读不到包里的 .tsd：" + err.Error()
	}
	var out []editCand
	idx := map[string]int{}
	for _, m := range reFieldTag.FindAllString(ts, -1) {
		n := reAttr("name").FindStringSubmatch(m)
		c := reAttr("can_edit").FindStringSubmatch(m)
		if n == nil || n[1] == "" || c == nil || (c[1] != "Y" && c[1] != "N") {
			continue
		}
		if i, dup := idx[n[1]]; dup {
			out[i].CanEdit = c[1]
			continue
		}
		idx[n[1]] = len(out)
		out = append(out, editCand{Name: n[1], CanEdit: c[1]})
	}
	if len(out) == 0 {
		return nil, "包里的 .tsd 没有带 can_edit 的 <field>（没有可翻的字段）"
	}
	return out, ""
}

// resolveEditTarget 在进程里把第一个「既能在布局里定位、又真的绑在一列上」的候选定下来。
//
// 为什么要问引擎而不是自己解析 .4fd：路径的定义在 ElementIndex 里（自己拼就多一处会漂的
// 第二实现），而 `column` 非空才是「数据绑定」这件事的权威答案。
//
// 分批（32 个一问）而不是一次全问：一个包能有 390 个候选，判据几乎总在头几个上命中，
// 分批只在「头几个都不合格」的包上多花一轮。
func resolveEditTarget(t *testing.T, s *stdioSession, label, h string, cands []editCand) (
	name, path, was, flip string, ok bool) {

	t.Helper()
	const chunk = 32
	for start := 0; start < len(cands); start += chunk {
		end := start + chunk
		if end > len(cands) {
			end = len(cands)
		}
		for i := start; i < end; i++ {
			// 一进一出：这里逐帧发，所以每一帧都能用同一个真实句柄 —— 不需要假设句柄串是 h1。
			fc, ok := ask[findReply](t, s, label+" find_component "+cands[i].Name, "find_component",
				map[string]any{"handle": h, "query": cands[i].Name})
			if !ok {
				return "", "", "", "", false
			}
			for _, m := range fc.Matches {
				if m.Name != cands[i].Name || m.Column == "" {
					continue
				}
				return m.Name, m.Path, cands[i].CanEdit, flipYN(cands[i].CanEdit), true
			}
		}
	}
	t.Errorf("%s: %d 个带 can_edit 的字段一个都没能定位到「绑了列」的布局元素上（find_component 全落空）",
		label, len(cands))
	return "", "", "", "", false
}

// findReply 是 find_component 的返回（只列我们读的字段）。
type findReply struct {
	Query      string      `json:"query"`
	Exact      bool        `json:"exact"`
	MatchCount int         `json:"matchCount"`
	Matches    []matchItem `json:"matches"`
}

type matchItem struct {
	Name   string `json:"name"`
	Tag    string `json:"tag"`
	Path   string `json:"path"`
	Table  string `json:"table"`
	Column string `json:"column"`
}

// flipYN 翻一个 Y/N。
//
// 必须与当前值不同：SpecAttributeUndoRedoCommand 会把 old==new 的改动当空操作丢掉，
// 一次同值改动是静默的空操作，而一批全是空操作的用例什么都证明不了（旧实现的原话）。
func flipYN(v string) string {
	if v == "Y" {
		return "N"
	}
	return "Y"
}

//---------------------------------------------------------------------------
// 用例 4：语料 Action（旧 batch-action.sh）
//---------------------------------------------------------------------------

func actionOp() corpusOp {
	return corpusOp{
		name:     "Action",
		key:      "action",
		validate: true,
		prepare: func(t *testing.T, idx int, src, ws string) (func(*stdioSession, string) string, string) {
			// 落点要在进程里从 form_tree 拿（见 formTree 的注释），所以这一条没有进程外挑选。
			// 「有没有可插入的 Grid」于是只能在进程里判 —— 语料里 67 个包都有（实测），
			// 真没有时报一句说清前置的错，而不是静静跳过。
			label := filepath.Base(src)
			return func(s *stdioSession, h string) string {
				root := formTree(t, s, label, h)
				if root == nil {
					return ""
				}
				grid := firstPopulatedGrid(root)
				if grid == nil {
					t.Errorf("%s: 布局里没有有子元素的 Grid（设计器的 mime 门禁只允许 Grid/Group 收 Button）。"+
						"若这是语料真的变了，这条用例该像旧实现那样把它算作 skip 并说清缺什么", label)
					return ""
				}
				el, ok := ask[elementRef](t, s, label+" add_widget", "add_widget", map[string]any{
					"handle": h, "path": grid.Path, "type": "Button",
				})
				if !ok {
					return ""
				}
				// Button 的 SpecNodeType 是 ACTION：SpecificationInfo.Add 现造一个 <act> 节点，
				// 而它必须先被推出 CREATE 状态，否则 ToXml() 会把它从 .tsd 里丢掉
				// （batch-action.sh 的头注释）。这两条正是这一条用例独有的覆盖面。
				if el.SpecNodeType != "ACTION" {
					t.Errorf("%s: add_widget Button 的 specNodeType 是 %q，该是 ACTION", label, el.SpecNodeType)
				}
				if el.Promoted < 1 {
					t.Errorf("%s: add_widget Button promoted=%d（该 ≥1）—— <act> 没被推出 CREATE，会从 .tsd 里丢掉",
						label, el.Promoted)
				}
				return fmt.Sprintf("add_widget Button 落在 %s → %s（promoted=%d）", grid.Name, el.Path, el.Promoted)
			}, ""
		},
	}
}

func TestCorpusActionPath(t *testing.T) {
	env := requireCorpus(t)
	corpusPass(t, env, actionOp())
}

//---------------------------------------------------------------------------
// 包与工作区的只读工具
//---------------------------------------------------------------------------

// zipEntryText 取包里第一个以 ext 结尾的条目。
//
// .tzs 是个 zip，且条目名带程序名前缀（aapp320.4fd / aapp320.tsd / ver / …）——
// 所以按扩展名找，不按固定名字找（程序名每个包都不一样）。
func zipEntryText(zipPath, ext string) (string, error) {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", err
	}
	defer zr.Close()
	for _, f := range zr.File {
		if !strings.EqualFold(filepath.Ext(f.Name), ext) {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", err
		}
		defer rc.Close()
		// 大小上限不是洁癖：坏掉的包会让 io.ReadAll 吃满内存，而这里只想读文本条目。
		b, err := io.ReadAll(io.LimitReader(rc, 64<<20))
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	return "", fmt.Errorf("包里没有 %s 条目", ext)
}

var (
	// reRecordField 匹配 .4fd 里 <Record> 段的数据源字段（pick-column.py 用的形状；
	// 放宽成「到第一个 > 为止」，于是自闭和与非自闭和两种都收）。
	reRecordField = regexp.MustCompile(`<RecordField\b[^>]*>`)
	// reColAttr 是 .tbl 里那一整块列元数据（TableColumnHelper.GetColField 读的就是它）。
	reColAttr = regexp.MustCompile(`(?s)<col_attr>(.*?)</col_attr>`)
	// reColField 是 <col_attr> 里的一列：name 与 widget 都要。
	reColField = regexp.MustCompile(`<field\s+name="([^"]*)"\s+widget="([^"]*)"`)
	// reFieldTag 匹配 .tsd 里 <field> 的开标签（它常带一段 CDATA 正文而不是自闭和，
	// 所以只匹配开标签 —— pick-edit-target.py 的注释里写着这个坑）。
	reFieldTag = regexp.MustCompile(`<field\b[^>]*>`)
	// reAttrCache 缓存每个属性名一个正则（属性值里不会有引号，所以这样够用）。
	reAttrCache = map[string]*regexp.Regexp{}
)

// reAttr 取 ` name="v"` 里的 v。每个属性名一个正则，缓存起来 —— 这个循环要跑几十万次。
func reAttr(name string) *regexp.Regexp {
	if re, ok := reAttrCache[name]; ok {
		return re
	}
	re := regexp.MustCompile(`(?:^|\s)` + regexp.QuoteMeta(name) + `="([^"]*)"`)
	reAttrCache[name] = re
	return re
}

//---------------------------------------------------------------------------
// `out` 闸门（引擎侧，2026-09-24 加）
//---------------------------------------------------------------------------

// TestOutGateRefusesToDestroyTheSource 钉住写入前那两道闸门：`out` 不能指向已打开的包、
// 不能落在工作区之外。
//
// 为什么值得一条真机用例：`Save.Run` 结尾就是 `File.WriteAllBytes(outPath, ...)`，在闸门
// 存在之前，把 `out` 指到源包**会覆盖原始素材**，而阻止它的只有 SKILL 里的一条红线 ——
// 代码零防线。`.tzc` 那条线一直有闸门 + 原子写 + prev 备份 + 源包 sha256，这边一直没有。
//
// 判据三问，缺一不可：
//
//	① 指到**源包**必须被拒，而且源包**一个字节不变** —— 这才是闸门存在的理由
//	② 指到**工作区之外**必须被拒 —— 从前它退 0 成功，那个包之后才 open 不了
//	③ 指到工作区内的**新**路径必须放行 —— 没有这一条，一个"见谁都拒"的实现也能让①②变绿
//
// `field_add` 走的是同一个出口（它也经过 SaveFn），所以这两道闸门对它是同一个实现；
// `field_add --out <源包>` 在 2026-09-24 手工验过（回 E_BAD_PARAM / out-is-open-package）。
func TestOutGateRefusesToDestroyTheSource(t *testing.T) {
	env := requireCorpus(t)
	if len(env.pkgs) == 0 {
		t.Skip("语料里没有包")
	}
	src := env.pkgs[0]
	ws := workspaceOf(src)

	before, err := sha16(src)
	if err != nil {
		t.Fatalf("算源包 sha256 失败：%v", err)
	}

	s := startStdio(t, env.exe, ws, 3*time.Minute)
	defer s.close()

	opened, ok := ask[openReply](t, s, "outgate/open", "open", map[string]any{"path": src})
	if !ok || opened.Handle == "" {
		t.Fatalf("打不开语料包 %s", src)
	}

	// ① out = 源包
	if r, err := s.call("save", map[string]any{"handle": opened.Handle, "out": src}); err != nil {
		t.Errorf("save(out=源包)：调用本身失败：%v", err)
	} else if r.OK {
		t.Error("save(out=源包) 该被拒 —— 它成功了，原始素材会被覆盖")
	} else if r.Error == nil || !strings.Contains(string(r.Error.Detail), "out-is-open-package") {
		t.Errorf("拒绝的 detail 里该有 out-is-open-package，得：%v", r.Error)
	}
	if after, err := sha16(src); err != nil || after != before {
		t.Errorf("被拒的 save 动了源包：%s → %s（err=%v）", before, after, err)
	}

	// ② out 在工作区之外
	outside := filepath.Join(os.TempDir(), fmt.Sprintf("_outgate_%d.tzs", os.Getpid()))
	if inWorkspace(outside, ws) {
		t.Log("os.TempDir() 落在工作区内，用一个明显在外的路径代替")
		outside = filepath.Join(filepath.Dir(ws), "..", "_outgate_outside_"+itoaPID()+".tzs")
	}
	defer os.Remove(outside)
	if r, err := s.call("save", map[string]any{"handle": opened.Handle, "out": outside}); err != nil {
		t.Errorf("save(工作区外)：调用本身失败：%v", err)
	} else if r.OK {
		t.Error("save 到工作区外该被拒（写成功了那个包之后也打不开）—— 它成功了")
	} else if r.Error == nil || !strings.Contains(string(r.Error.Detail), "out-outside-workspace") {
		t.Errorf("拒绝的 detail 里该有 out-outside-workspace，得：%v", r.Error)
	}
	if _, err := os.Stat(outside); err == nil {
		t.Errorf("被拒的 save 还是在工作区外写出了文件：%s", outside)
	}

	// ③ out 在工作区内的新路径 —— 必须放行
	inside := scratchPath(src, 0, "outgate", "ok")
	defer os.Remove(inside)
	if !saveTo(t, s, "outgate/save(工作区内)", opened.Handle, inside) {
		t.Error("save 到工作区内的新路径该成功 —— 被拒说明闸门在误伤正常调用")
	}
	if _, err := os.Stat(inside); err != nil {
		t.Errorf("放行的 save 没有落盘：%v", err)
	}
}

// inWorkspace 是闸门那条判据的测试侧复述（目录前缀、分隔符归一、大小写不敏感）。
func inWorkspace(p, ws string) bool {
	norm := func(s string) string { return strings.ToLower(strings.ReplaceAll(s, "/", `\`)) }
	d := norm(filepath.Dir(p))
	w := norm(ws)
	for len(w) > 1 && strings.HasSuffix(w, `\`) {
		w = w[:len(w)-1]
	}
	return strings.HasPrefix(d+`\`, w+`\`)
}

func itoaPID() string { return strconv.Itoa(os.Getpid()) }

//---------------------------------------------------------------------------
// 布局属性的取值规则：报给人看的那一半，与拒人时给的那一半
//---------------------------------------------------------------------------

// describeLayoutReply 是 `describe_kind --kind layout` 的形状（只列我们读的字段）。
type describeLayoutReply struct {
	Count  int `json:"count"`
	Layout []struct {
		Name    string   `json:"name"`
		Type    string   `json:"type"`
		Values  []string `json:"values"`
		Initial string   `json:"initial"`
	} `json:"layout"`
}

// TestCorpusLayoutValueRuleAgreesWithItsDiscovery —— 取值规则的两半必须是**同一份**。
//
// 同一个知识有两个出口：`describe_kind --kind layout` 在**写之前**把它报出来，
// `set_layout_attr` 在**写错之后**把它放进 `detail.legal`。两半都来自工作区的
// `mta/mod-fd.spec`（SpecValues 一张表、两个方向），所以它们对不上就说明有人只改了一边。
// 而 2026-09-25 之前的状态正是"只有后半个出口"：调用方写之前问不到，只能猜 ——
// 评测 F13 里，执行者猜了 `hidden`（对的）而不敢碰 `invisible`（BOOLEAN，4.2 名 isPassword），
// 值也只能从别处外推。这条把"两个出口一致"钉成机械判据。
//
// 每个**工作区**验一次就够（规则是按 `mta/mod-fd.spec` 定的，同一工作区里的包共用它），
// 而且只写**非法**值（必被拒 ⇒ 模型不动），所以它不改变任何包的产出，也不进 RoundTrip 那条链。
func TestCorpusLayoutValueRuleAgreesWithItsDiscovery(t *testing.T) {
	env := requireCorpus(t)
	seen := map[string]bool{}
	checked := 0
	for _, src := range env.pkgs {
		ws := workspaceOf(src)
		if ws == "" || seen[ws] {
			continue
		}
		seen[ws] = true
		if checkLayoutValueRule(t, env, src, ws) {
			checked++
		}
	}
	if checked == 0 {
		t.Skip("没有一个工作区能验（都缺 mta/mod-fd.spec，或包打不开）")
	}
	t.Logf("验过 %d 个工作区（语料里共 %d 个）", checked, len(seen))
}

// checkLayoutValueRule 在一个工作区上验一次，返回是否真的验到了。
func checkLayoutValueRule(t *testing.T, env *corpusEnv, src, ws string) bool {
	t.Helper()
	s := startStdio(t, env.exe, ws, 3*time.Minute)
	defer s.close()

	opened, ok := ask[openReply](t, s, "layoutrule/open", "open", map[string]any{"path": src})
	if !ok || opened.Handle == "" {
		t.Logf("跳过 %s：打不开", filepath.Base(src))
		return false
	}

	// ① 写之前先问：名字 + 类型 + 取值集
	rep, err := s.call("describe_kind", map[string]any{"handle": opened.Handle, "kind": "layout"})
	if err != nil || rep == nil || !rep.OK {
		t.Errorf("describe_kind --kind layout 该成功：err=%v reply=%+v", err, rep)
		return false
	}
	var dl describeLayoutReply
	if err := json.Unmarshal(rep.Result, &dl); err != nil {
		t.Errorf("describe_kind 的返回解不开：%v（%s）", err, string(rep.Result)[:200])
		return false
	}
	if dl.Count != len(dl.Layout) {
		t.Errorf("count=%d 与记录数 %d 对不上", dl.Count, len(dl.Layout))
	}
	if len(dl.Layout) == 0 {
		t.Errorf("布局属性一条都没有？%s", filepath.Base(src))
		return false
	}
	declared, typed := map[string][]string{}, 0
	for _, a := range dl.Layout {
		if a.Name == "" {
			t.Errorf("有一条记录没有 name：%+v", a)
		}
		if a.Type != "" {
			typed++
		}
		if len(a.Values) > 0 {
			declared[a.Name] = a.Values
		}
	}
	if typed == 0 {
		t.Logf("跳过 %s：这个工作区有 mta/ 但 mod-fd.spec 什么都没声明（SpecValues 对缺文件的容错）",
			filepath.Base(src))
		return false
	}

	// ② 找一个**真的长在元素身上**、且**有声明取值集**的属性。值规则排在白名单之后，属性不在
	//    元素上会先撞 E_ATTR_NOT_WHITELIST，那就没验到这一条 —— 所以要跑到一个真有它的元素上。
	//    表单根节点通常只带 gridWidth 这种（INTEGER，没有取值集），所以要往后走几个：实测
	//    aapt300 的根 Form 一个都没有，第二个元素 HBoxT1 就带 hidden（集合 false|true）。
	root := formTree(t, s, "layoutrule", opened.Handle)
	if root == nil {
		return false
	}
	for _, elPath := range firstTreePaths(root, 6) {
		gc, err := s.call("get_component", map[string]any{"handle": opened.Handle, "path": elPath})
		if err != nil || gc == nil || !gc.OK {
			t.Errorf("get_component(%s) 失败：err=%v reply=%+v", elPath, err, gc)
			return false
		}
		var comp struct {
			Layout map[string]string `json:"layout"`
		}
		if err := json.Unmarshal(gc.Result, &comp); err != nil {
			t.Errorf("get_component 的返回解不开：%v", err)
			return false
		}

		// 不写死"试哪几个属性名"：把这个元素**真实带着**的每个属性挨个对一遍 describe_kind，
		// 取第一个两边都认的。硬编码 hidden/case/… 会漏掉别的表单上不同的那一批。
		names := make([]string, 0, len(comp.Layout))
		for k := range comp.Layout {
			names = append(names, k)
		}
		sort.Strings(names) // 定序，失败信息可复现
		for _, name := range names {
			vals := declared[name]
			if len(vals) == 0 {
				continue
			}

			// ③ 写一个**不在集合里**的值 → 必须被拒，且 detail.legal 逐字等于 ① 报的那一份
			r, err := s.call("set_layout_attr", map[string]any{
				"handle": opened.Handle, "path": elPath, "attr": name, "value": "TT_NOPE",
			})
			if err != nil {
				t.Errorf("set_layout_attr(%s=TT_NOPE) 调用本身失败：%v", name, err)
				return false
			}
			if r.OK {
				t.Errorf("%s=%q 该被拒（合法集 %v）—— 它成功了，静默写脏值", name, "TT_NOPE", vals)
				return false
			}
			if r.Error == nil || r.Error.Code != "E_ATTR_VALUE_ILLEGAL" {
				t.Errorf("%s=TT_NOPE 该回 E_ATTR_VALUE_ILLEGAL，得 %+v", name, r.Error)
				return false
			}
			var detail struct {
				Legal []string `json:"legal"`
				Type  string   `json:"type"`
			}
			if err := json.Unmarshal(r.Error.Detail, &detail); err != nil {
				t.Errorf("拒绝的 detail 解不开：%v（%s）", err, string(r.Error.Detail))
				return false
			}
			if !sameStrings(detail.Legal, vals) {
				t.Errorf("%s：describe_kind 报的取值集是 %v，拒绝时 detail.legal 给的是 %v —— 两个出口不是同一份",
					name, vals, detail.Legal)
				return false
			}
			return true
		}
	}
	t.Logf("跳过 %s：前 6 个元素上都没找到带声明取值集的属性", filepath.Base(src))
	return false
}

// firstTreePaths 按文档序取前 n 个有 path 的节点（宽度优先，父先于子）。
func firstTreePaths(root *treeNode, n int) []string {
	var out []string
	queue := []*treeNode{root}
	for len(queue) > 0 && len(out) < n {
		node := queue[0]
		queue = queue[1:]
		if node == nil {
			continue
		}
		if node.Path != "" {
			out = append(out, node.Path)
		}
		queue = append(queue, node.Children...)
	}
	return out
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// listPackagesReply 是 `list_packages` 的返回（只列我们读的字段）。
type listPackagesReply struct {
	Count     int  `json:"count"`
	Returned  int  `json:"returned"`
	Truncated bool `json:"truncated"`
	Packages  []struct {
		Name    string `json:"name"`
		Path    string `json:"path"`
		Program string `json:"program"`
		Size    int64  `json:"sizeBytes"`
	} `json:"packages"`
}

// TestCorpusListPackagesNamesWhatItSays —— `list_packages` 报的 `program` 是**从文件名推的**，
// 而 `open` 的 `program` 是设计器自己算的：这条在**真实工作区**上把两者对一遍。
//
// 为什么值得一条：那个 `program` 只是"挑文件"的提示，一旦推错（文件名里多一对括号、后缀没剥干净、
// `.tzv` 的少见形态），调用方会以为"这个程序没有包" —— 而它**不是权威**，文档与返回体的 note
// 都这么写。所以判据是"两个答案一致"，顺带把"报的路径真的存在"一起验了。
func TestCorpusListPackagesNamesWhatItSays(t *testing.T) {
	env := requireCorpus(t)
	seen := map[string]bool{}
	checked := 0
	for _, src := range env.pkgs {
		ws := workspaceOf(src)
		if ws == "" || seen[ws] {
			continue
		}
		seen[ws] = true
		s := startStdio(t, env.exe, ws, 3*time.Minute)
		rep, ok := ask[listPackagesReply](t, s, "listpackages", "list_packages",
			map[string]any{"limit": 5})
		if !ok {
			s.close()
			continue
		}
		if rep.Count < rep.Returned {
			t.Errorf("count=%d 比 returned=%d 还小", rep.Count, rep.Returned)
		}
		if rep.Count == 0 {
			t.Errorf("%s 里一个包都没有？工作区明明有 .tzs（%s）", ws, filepath.Base(src))
		}
		opened := false
		for _, p := range rep.Packages {
			if p.Name == "" || p.Path == "" {
				t.Errorf("每条都该有 name 与 path：%+v", p)
				continue
			}
			if _, err := os.Stat(p.Path); err != nil {
				t.Errorf("%s 报的路径不存在：%v", p.Name, err)
			}
			if p.Program == "" || opened {
				continue // 每个工作区开一个就够：其余条目的路径已由上面那次 Stat 覆盖
			}
			o, ok := ask[openReply](t, s, "listpackages/open", "open", map[string]any{"path": p.Path})
			if !ok {
				continue
			}
			if o.Program != p.Program {
				t.Errorf("%s：list_packages 推的 program=%q，open 回的是 %q —— 推导与权威不一致",
					p.Name, p.Program, o.Program)
			}
			s.call("close", map[string]any{"handle": o.Handle})
			opened, checked = true, checked+1
		}
		s.close()
	}
	if checked == 0 {
		t.Skip("没验到（语料里没有推得出工作区的包）")
	}
	t.Logf("在 %d 个工作区上对过 program（list_packages 推的 vs open 回的）", checked)
}
