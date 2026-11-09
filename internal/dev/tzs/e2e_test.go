package tzs

// 需要**真引擎**的测试：默认 `go test ./...` 全部跳过（会打印跳过原因）。
//
// 为什么要这个开关：真引擎要 Boot 一次 WPF 设计器（约 1 s）、要装一份设计器、
// 要一个 .tzs 工作区 —— 这三样都不是 CI 或别人的机器上一定有的东西。
// 而把它们写进默认测试，`go test ./...` 就会变成「时好时坏」，
// 那种假失败比不跑更糟（corpus_test.go 的 requireDeep 是同一个理由）。
//
// 开启方式（三样都得给，工作区**绝不留缺省** —— 引擎自己的缺省是一个真实客户目录）：
//
//	$env:TTZS_E2E="1"
//	$env:TTZS_EXE="D:\...\engine\out\tzs-server.exe"
//	$env:TTZS_WS="D:\t100_wrok_dir\<模块>"        # 一个真的 .tzs 工作区
//	$env:TTZS_INSTALL="D:\APPS\T100设计器_1.0.0.251_免安装"   # 可省：引擎有内置缺省
//	go test ./internal/dev/tzs/ -run 'TestE2E' -timeout 5m -v

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"tt/internal/testenv"
)

// closeIfOpen 把某个包在当前工作区守护进程里的会话关掉（没有就什么都不做）。
//
// 用途只有一个：让一条会改内存模型的用例可以从干净状态起跑 —— 模型是**先改后存**，
// save 被拒也会留下改动，于是第二次跑就撞重名。
func closeIfOpen(ctx context.Context, t *testing.T, o Options, m *Manifest, pkg string) {
	t.Helper()
	pipe, err := Ensure(ctx, o)
	if err != nil {
		t.Fatalf("Ensure 失败: %v", err)
	}
	raw, err := BuildArgsFromJSON(m, "list_open", json.RawMessage(`{}`))
	if err != nil {
		return
	}
	r, err := Call(ctx, DefaultDialer(), pipe, 99, "list_open", raw, DefaultTimeout)
	if err != nil || !r.OK {
		return
	}
	var rows []struct{ Handle, Path string }
	if json.Unmarshal(r.Result, &rows) != nil {
		return
	}
	for _, row := range rows {
		if !strings.EqualFold(filepath.ToSlash(row.Path), filepath.ToSlash(pkg)) {
			continue
		}
		a, err := BuildArgsFromJSON(m, "close", json.RawMessage(`{"handle":`+JSONString(row.Handle)+`}`))
		if err != nil {
			continue
		}
		_, _ = Call(ctx, DefaultDialer(), pipe, 98, "close", a, DefaultTimeout)
	}
}

// requireE2E 把「对真引擎」的测试设为显式开关。
func requireE2E(t *testing.T) (exe, ws, install string) {
	t.Helper()
	if os.Getenv("TTZS_E2E") == "" {
		t.Skip("需要真引擎：设 TTZS_E2E=1 + 引擎 exe + 工作区（TTZS_EXE / TTZS_WS，" +
			"或写进仓库根的 config.local.json），可选 TTZS_INSTALL。见本文件顶部注释")
	}
	exe = testenv.EngineExe()
	if exe == "" {
		// 从包目录出发的相对缺省（本仓库里引擎的产物就在那儿），只是省事，不是保证。
		exe = filepath.Join("..", "..", "..", "engine", "out", "tzs-server.exe")
	}
	if _, err := os.Stat(exe); err != nil {
		t.Skipf("找不到引擎 exe（%s）：%v", exe, err)
	}
	ws = testenv.Workspace()
	if strings.TrimSpace(ws) == "" {
		t.Skip("没给工作区（TTZS_WS 或 config.local.json 的 workspace）。" +
			"工作区绝不替你选：引擎自己的缺省是一个真实客户目录")
	}
	return exe, ws, testenv.DesignerDir()
}

// TestE2EManifest 用真 manifest 校验我们的解析层：两个「不 Boot」的开关，
// 所以这条既便宜又最能证明「我们读的字段名没写错」。
func TestE2EManifest(t *testing.T) {
	exe, ws, _ := requireE2E(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	m, err := FetchManifest(ctx, exe)
	if err != nil {
		t.Fatalf("--manifest 失败: %v", err)
	}
	// 契约说 ~50 个函数。给个下界即可（引擎会先声明全部、再慢慢补实现）。
	if len(m.Fns) < 40 {
		t.Errorf("函数表只有 %d 个，看着不像完整表", len(m.Fns))
	}
	for _, fn := range []string{"open", "save", "close", "list_open", "form_tree", "set_spec_attr", "validate"} {
		if m.ByName(fn) == nil {
			t.Errorf("函数表里该有 %s", fn)
		}
	}
	// stop 不在 manifest 里（传输级）。这条是契约里最容易写错的一条。
	if m.ByName(FnStop) != nil {
		t.Errorf("%s 不该在 manifest 里", FnStop)
	}
	// 真 manifest 里的类型必须全部被我们认识 —— 尤其是契约清单里漏掉的 handle。
	for _, f := range m.Fns {
		for _, a := range f.Args {
			if !KnownType(a.Type) {
				t.Errorf("%s.%s 的类型 %q 我们不认识（引擎加了新类型？）", f.Name, a.Name, a.Type)
			}
		}
	}
	// 实测过的两处：from:* 真的只在 attr 上；add_action 的 type 是普通 string。
	if a := m.ByName("set_spec_attr").Param("attr"); a == nil || a.From != "spec:<kind>" {
		t.Errorf("set_spec_attr.attr 该带 from=spec:<kind>：%+v", a)
	}
	if ty := m.ByName("add_action").Param("type"); ty == nil || ty.From != "" {
		t.Errorf("add_action.type 该是普通 string（无 from）：%+v", ty)
	}
	if m.ByName("list_open").NeedsHandle {
		t.Errorf("list_open 必须 needsHandle=false（它是最常用来探活的一次性只读动词；" +
			"注意就绪握手本身已经不再调它了，见 server.go 的 probeReady）")
	}

	name, err := PipeName(ctx, exe, ws)
	if err != nil {
		t.Fatalf("--pipe-name 失败: %v", err)
	}
	if !regexp.MustCompile(`^tzs-cli-[0-9a-f]{8}-[0-9a-f]{8}$`).MatchString(name) {
		t.Errorf("管道名形状不对：%q", name)
	}
	// 换工作区名字就该换管道名（工作区是 hash 的输入之一）。
	other, err := PipeName(ctx, exe, filepath.Join("D:", "绝对不存在的模块"))
	if err != nil {
		t.Fatalf("--pipe-name（另一个工作区）失败: %v", err)
	}
	if other == name {
		t.Errorf("不同工作区该得到不同管道名（%q）", name)
	}
	t.Logf("真 manifest：%d 个函数；工作区 %s → 管道 %s", len(m.Fns), ws, name)
}

// TestE2EEnsureCallStop 走一遍完整生命周期：spawn → 就绪 → 一次真调用 → stop。
//
// 用 list_open 当那次调用：它是 needsHandle=false 的只读函数，不需要打开任何包，
// 所以这条测试不会改动工作区里的任何东西。
func TestE2EEnsureCallStop(t *testing.T) {
	exe, ws, install := requireE2E(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	o := Options{Exe: exe, Workspace: ws, InstallDir: install}

	// 先确保干净：上一次跑留下的守护进程不该影响这次。
	_ = Stop(ctx, o)

	pipe, err := Ensure(ctx, o)
	if err != nil {
		t.Fatalf("Ensure 失败: %v", err)
	}
	info, err := LookupDaemon(ctx, o)
	if err != nil {
		t.Fatalf("LookupDaemon 失败: %v", err)
	}
	if !info.Running {
		t.Fatalf("Ensure 之后该有一个应答的守护进程：%+v", info)
	}
	if info.Pipe != pipe {
		t.Errorf("Ensure 返回的管道名与快照不一致：%q vs %q", pipe, info.Pipe)
	}

	// 就绪握手只证明"管道连得上"（不再发任何帧）；这里走一遍完整的 Call 路径（带参数定型）。
	// 用 list_open 当那次调用：只读、不需要打开任何包，所以不会改动工作区里的东西。
	args, err := BuildArgsFromJSON(mustManifestFromEngine(t, exe), "list_open", nil)
	if err != nil {
		t.Fatalf("list_open 的 args 该是空的：%v", err)
	}
	r, err := Call(ctx, DefaultDialer(), pipe, 1, "list_open", args, DefaultTimeout)
	if err != nil {
		t.Fatalf("list_open 调用失败: %v", err)
	}
	if !r.OK {
		t.Fatalf("list_open 该成功：%+v", r.Error)
	}
	if got := ExitCode(r, nil); got != ExitOK {
		t.Errorf("该退 0，得 %d", got)
	}

	// 再 Ensure 一次：必须**复用**（不重启），否则每次调用都要重 Boot 一次。
	pipe2, err := Ensure(ctx, o)
	if err != nil {
		t.Fatalf("第二次 Ensure 失败: %v", err)
	}
	if pipe2 != pipe {
		t.Errorf("同一个工作区该复用同一个管道：%q → %q", pipe, pipe2)
	}

	// stop：幂等，且绝不 spawn。
	if err := Stop(ctx, o); err != nil {
		t.Fatalf("Stop 失败: %v", err)
	}
	if err := Stop(ctx, o); err != nil {
		t.Fatalf("第二次 Stop 该是幂等的：%v", err)
	}
	if info, _ := LookupDaemon(ctx, o); info != nil && info.Running {
		t.Errorf("stop 之后不该还有应答的守护进程：%+v", info)
	}
}

// TestE2ELogicalKey 走一遍**逻辑键寻址**：open 一个真包，然后用程序名（而不是句柄）寻址。
//
// 这条要真包，所以额外要求 TTZS_PKG（一个真 .tzs 的**绝对路径**，必须在本工作区内 ——
// 引擎的 TzpManager 会拒绝工作区之外的文件）。没给就跳过。
//
// 它证明的是 Go 与引擎对 `--form` 的理解一致：Go 把程序名写进 handle 字段，
// 引擎按程序名/ProgramKey 解析（Rpc.FindByKey）。只测 Go 那一半证明不了这件事。
func TestE2ELogicalKey(t *testing.T) {
	exe, ws, install := requireE2E(t)
	pkg := os.Getenv("TTZS_PKG")
	if strings.TrimSpace(pkg) == "" {
		t.Skip("没给 TTZS_PKG（一个本工作区内的真 .tzs 路径）：逻辑键寻址要用真包验")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	o := Options{Exe: exe, Workspace: ws, InstallDir: install}

	m := mustManifestFromEngine(t, exe)
	pipe, err := Ensure(ctx, o)
	if err != nil {
		t.Fatalf("Ensure 失败: %v", err)
	}

	// open（句柄只用来在最后 close，全程不用它寻址）
	openArgs, err := BuildArgsFromJSON(m, "open", json.RawMessage(`{"path":`+JSONString(pkg)+`}`))
	if err != nil {
		t.Fatal(err)
	}
	opened, err := Call(ctx, DefaultDialer(), pipe, 1, "open", openArgs, DefaultTimeout)
	if err != nil {
		t.Fatalf("open 失败: %v", err)
	}
	if !opened.OK {
		t.Fatalf("open 该成功：%+v", opened.Error)
	}
	var openRes struct {
		Handle  string `json:"handle"`
		Program string `json:"program"`
		Key     string `json:"key"`
	}
	if err := json.Unmarshal(opened.Result, &openRes); err != nil {
		t.Fatalf("open 的 result 解不动：%v", err)
	}
	if openRes.Program == "" || openRes.Key == "" {
		t.Fatalf("open 该回 program 与 key（逻辑键就是它们）：%s", opened.Result)
	}
	defer func() {
		closeArgs, _ := BuildArgsFromJSON(m, "close", json.RawMessage(`{"handle":`+JSONString(openRes.Handle)+`}`))
		_, _ = Call(ctx, DefaultDialer(), pipe, 9, "close", closeArgs, DefaultTimeout)
	}()

	for _, addr := range []string{openRes.Program, openRes.Key} {
		// 用**逻辑键**寻址：Go 侧把它写进 handle 字段（--form 走的就是这条路）。
		args, err := BuildArgsForForm(m, "form_tree", json.RawMessage(`{"depth":1}`), addr)
		if err != nil {
			t.Fatalf("按 %q 定型失败：%v", addr, err)
		}
		if !strings.Contains(string(args), `"handle":`+JSONString(addr)) {
			t.Fatalf("逻辑键没被写进 handle：%s", args)
		}
		r, err := Call(ctx, DefaultDialer(), pipe, 2, "form_tree", args, DefaultTimeout)
		if err != nil {
			t.Fatalf("按 %q 调用失败：%v", addr, err)
		}
		if !r.OK {
			t.Errorf("按逻辑键 %q 该能寻址，却失败：%+v", addr, r.Error)
		}
	}

	// 不存在的名字要给出**可自纠**的错误：点名两种写法，并列出当前开着的候选。
	badArgs, _ := BuildArgsForForm(m, "form_tree", json.RawMessage(`{"depth":1}`), "绝不存在_zzz")
	r, err := Call(ctx, DefaultDialer(), pipe, 3, "form_tree", badArgs, DefaultTimeout)
	if err != nil {
		t.Fatalf("调用失败：%v", err)
	}
	if r.OK {
		t.Fatal("不存在的逻辑键不该成功")
	}
	if got := ExitCode(r, nil); got != ExitUsage {
		t.Errorf("kind=not_found 该退 2，得 %d", got)
	}
	// 错误必须**教怎么改**：两种寻址写法 + 当前开着的候选（上一轮在 Rpc.Resolve 里加的）。
	if !strings.Contains(r.Error.Message, "程序名") || !strings.Contains(r.Error.Message, "ProgramKey") {
		t.Errorf("错误文案该点出两种寻址写法：%s", r.Error.Message)
	}
	var det struct {
		Candidates []map[string]any `json:"candidates"`
	}
	if len(r.Error.Detail) > 0 {
		if err := json.Unmarshal(r.Error.Detail, &det); err != nil {
			t.Errorf("error.detail 解不动：%v", err)
		}
		if len(det.Candidates) == 0 {
			t.Errorf("detail.candidates 该列出开着的会话：%s", r.Error.Detail)
		} else if _, ok := det.Candidates[0]["key"]; !ok {
			t.Errorf("候选里该带 key（那正是可用的逻辑键）：%v", det.Candidates[0])
		}
	} else {
		t.Error("该带 error.detail")
	}
}

// TestE2EFieldAdd 走一遍**任务级动词**：只给 file（不给 handle、不先 open），让它自己开包、
// 自己挑容器、自己报校验增量，并可存新包。要真包（TTZS_PKG）。
//
// 它证明的是"一次请求做完一条链"真的成立，而不只是把 9 次调用换个写法：
//
//	· opened 第一次 true、第二次 false —— 复用会话语，不再撞 E_KEY_IN_USE；
//	· container 是自动挑的（真实包上应落到 worksheet 或 *layout*）；
//	· 校验是真增量，不是"首调按构造为空"的假象；
//	· out 存出来的是**新**包，源包一个字节没动；
//	· 返回体够小（它是省 context 的手段，不是副产品）。
func TestE2EFieldAdd(t *testing.T) {
	exe, ws, install := requireE2E(t)
	pkg := os.Getenv("TTZS_PKG")
	if strings.TrimSpace(pkg) == "" {
		t.Skip("没给 TTZS_PKG（本工作区内的真 .tzs 路径）：任务级动词要用真包验")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	o := Options{Exe: exe, Workspace: ws, InstallDir: install}
	m := mustManifestFromEngine(t, exe)

	spec := m.ByName("field_add")
	if spec == nil {
		t.Fatal("真 manifest 里该有 field_add")
	}
	if spec.NeedsHandle {
		t.Error("field_add 必须 needsHandle=false —— 它自己解析 file/handle")
	}

	before, err := os.ReadFile(pkg)
	if err != nil {
		t.Fatal(err)
	}
	shaBefore := sha256.Sum256(before)

	// `out` 必须落在**工作区内**：引擎写盘时会拒（2026-09-24 加的闸门），而这条测试原先
	// 写进 t.TempDir() —— 那正是闸门要拦的写法（从前它退 0 成功，那个包之后才 open 不了）。
	// 前缀沿用语料关卡那一套（`_tdev_<pid>_…`，语料发现把它排除在外），
	// 于是就算跑在别人的工作区上，也不会留下一个被当成语料的东西。
	// 让这次运行**从头开始**：`field_add --file` 会复用已经开着的会话，而上一次运行可能已经
	// 往那个会话的模型里加过这两个字段了 —— 加字段是**先改模型、再 save**，所以哪怕 save 被拒
	// （比如闸门拦下 out），改动也留在内存里。不清干净的话，第二次跑就会撞重名：那是"时好时坏"，
	// 比不测更糟（本文件第二条纪律）。实测过：紧接在闸门拒绝之后的那一次就失败了。
	closeIfOpen(ctx, t, o, m, pkg)

	outPkg := filepath.Join(ws, fmt.Sprintf("_tdev_%d_e2e_fieldadd.tzs", os.Getpid()))
	defer os.Remove(outPkg)
	call := func(id int, args string) *Reply {
		t.Helper()
		raw, err := BuildArgsFromJSON(m, "field_add", json.RawMessage(args))
		if err != nil {
			t.Fatalf("定型失败（%s）：%v", args, err)
		}
		pipe, err := Ensure(ctx, o)
		if err != nil {
			t.Fatalf("Ensure 失败: %v", err)
		}
		r, err := Call(ctx, DefaultDialer(), pipe, id, "field_add", raw, DefaultTimeout)
		if err != nil {
			t.Fatalf("调用失败: %v", err)
		}
		return r
	}

	// ① 只给 file：它应该自己开，并自动挑容器。
	r1 := call(1, `{"file":`+JSONString(pkg)+`,"table":"pmdl_t",`+
		`"columns":["pmdlent","pmdlsite"],"out":`+JSONString(outPkg)+`}`)
	if !r1.OK {
		t.Fatalf("field_add 该成功：%+v", r1.Error)
	}
	var rep struct {
		Form      string `json:"form"`
		Opened    bool   `json:"opened"`
		Container string `json:"container"`
		Validate  struct {
			NewErrorCount   int `json:"newErrorCount"`
			NewWarningCount int `json:"newWarningCount"`
		} `json:"validate"`
		BaselineCached bool `json:"baselineCached"`
		Saved          *struct {
			Out      string `json:"out"`
			BytesOut int64  `json:"bytesOut"`
		} `json:"saved"`
	}
	if err := json.Unmarshal(r1.Result, &rep); err != nil {
		t.Fatalf("返回体解不动：%v\n%s", err, r1.Result)
	}
	if rep.Form == "" || rep.Container == "" {
		t.Errorf("该回 form 与 container：%s", r1.Result)
	}
	if !rep.Opened {
		t.Error("只给了 file 而包没开着，opened 该是 true（它自己开的）")
	}
	if rep.BaselineCached {
		t.Error("第一次调用没有缓存基线，baselineCached 该是 false")
	}
	if rep.Saved == nil || rep.Saved.Out == "" || rep.Saved.BytesOut <= 0 {
		t.Errorf("给了 out 就该存出新包：%s", r1.Result)
	}
	if len(r1.Result) > 4096 {
		t.Errorf("返回体 %d 字节，太大了 —— 任务动词不该回表单全量与校验全表", len(r1.Result))
	}

	// ② 同一个 file 再调一次：复用会话（E_KEY_IN_USE 陷阱不该再出现），基线已缓存。
	r2 := call(2, `{"file":`+JSONString(pkg)+`,"table":"pmdl_t","columns":["pmdlunit"]}`)
	if !r2.OK {
		t.Fatalf("第二次该成功（复用会话）：%+v", r2.Error)
	}
	var rep2 struct {
		Opened         bool `json:"opened"`
		BaselineCached bool `json:"baselineCached"`
	}
	if err := json.Unmarshal(r2.Result, &rep2); err != nil {
		t.Fatal(err)
	}
	if rep2.Opened {
		t.Error("第二次不该重新 open（同一个文件已经开着，应当复用）")
	}
	if !rep2.BaselineCached {
		t.Error("第二次该用上已有的基线（省掉一次全表校验）")
	}

	// ③ 源包一个字节没动。
	after, err := os.ReadFile(pkg)
	if err != nil {
		t.Fatal(err)
	}
	if sha256.Sum256(after) != shaBefore {
		t.Error("源包被改了 —— save 只该写新包")
	}

	// ④ 新包真的存在且非空。
	if fi, err := os.Stat(outPkg); err != nil || fi.Size() == 0 {
		t.Fatalf("新包没写出来或为空: %v", err)
	}

	// 收尾：关掉它自己开的那个会话。
	if pipe, err := Ensure(ctx, o); err == nil {
		closeRaw, _ := BuildArgsFromJSON(m, "close", json.RawMessage(`{"handle":`+JSONString(rep.Form)+`}`))
		_, _ = Call(ctx, DefaultDialer(), pipe, 9, "close", closeRaw, DefaultTimeout)
	}
}

// mustManifestFromEngine 取真 manifest（E2E 里已经证明能取到）。
func mustManifestFromEngine(t *testing.T, exe string) *Manifest {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	m, err := FetchManifest(ctx, exe)
	if err != nil {
		t.Fatal(err)
	}
	return m
}
