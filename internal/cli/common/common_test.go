package common

import (
	"encoding/json"
	"io/fs"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"tt/internal/config"
	"tt/internal/output"
	"tt/internal/testkit"
)

// 本包是**叶子包**（三个命令组共享的 CLI 上下文），所以它自己的状态就是全局状态：
// 8 个包级 var + 一个 MetaProvider 函数变量，由根命令的 persistent flags 绑定。
// 测试里动过就必须还回去 —— 不还的话下一条用例拿到的是上一条留下的值，
// 而那种失败看起来像"被测函数算错了"。
//
// resetCommon 就是这一步；它**必须是**每一条会动这些 var 的用例的第一行。

// resetCommon 快照并还原本包的全部可变状态。
//
// 为什么不用 ResetForTest() 之类的生产 API：本包全是内部测试包（同包），
// 本来就能直接读写这些 var —— 加生产 API 买到的是零，却引入"只有测试用的生产代码"。
func resetCommon(t *testing.T) {
	t.Helper()
	old := struct {
		configPath, format, env string
		json, csv, verbose      bool
		webFS                   fs.FS
		version                 string
		meta                    func() output.Meta
	}{ConfigPath, Format, Env, JSON, CSV, Verbose, WebFS, Version, MetaProvider}
	t.Cleanup(func() {
		ConfigPath, Format, Env = old.configPath, old.format, old.env
		JSON, CSV, Verbose = old.json, old.csv, old.verbose
		WebFS, Version = old.webFS, old.version
		MetaProvider = old.meta
	})
}

// TestResetCommonActuallyRestores 先钉住 resetCommon 自己 ——
// 它是下面所有用例的地基；它坏了，别的测试会以难查的方式互相污染。
func TestResetCommonActuallyRestores(t *testing.T) {
	ConfigPath, Format, Env, JSON, CSV, Verbose, Version = "a", "b", "c", true, true, true, "d"
	MetaProvider = func() output.Meta { return output.Meta{Env: "旧"} }

	t.Run("内层改一圈", func(t *testing.T) {
		resetCommon(t)
		ConfigPath, Format, Env, JSON, CSV, Verbose, Version = "x", "y", "z", false, false, false, "w"
		MetaProvider = func() output.Meta { return output.Meta{Env: "新"} }
	})

	// 子测试结束后（cleanup 已跑）应当回到进内层之前那组值。
	if ConfigPath != "a" || Format != "b" || Env != "c" || Version != "d" {
		t.Errorf("字符串项没还原：%q %q %q %q", ConfigPath, Format, Env, Version)
	}
	if !JSON || !CSV || !Verbose {
		t.Errorf("布尔项没还原：%v %v %v", JSON, CSV, Verbose)
	}
	if !reflect.DeepEqual(MetaProvider(), output.Meta{Env: "旧"}) {
		t.Errorf("MetaProvider 没还原：%+v", MetaProvider())
	}
	// 收尾：把这条用例自己设的初值也还回去。
	resetCommon(t)
}

// TestOutputFormat 三个开关的合流规则：--json / --csv 是 --format 的语法糖。
func TestOutputFormat(t *testing.T) {
	cases := []struct {
		name  string
		json_ bool
		csv_  bool
		fmtS  string
		want  output.Format
	}{
		{"默认是 json", false, false, "", output.FormatJSON},
		{"--json", true, false, "", output.FormatJSON},
		{"--csv", false, true, "", output.FormatCSV},
		{"--format csv", false, false, "csv", output.FormatCSV},
		{"--format table", false, false, "table", output.FormatTable},
		{"--format text 等价于 table", false, false, "text", output.FormatTable},
		{"大小写与空白都容错", false, false, "  CSV  ", output.FormatCSV},
		{"认不出的形态落回 json（不是报错）", false, false, "yaml", output.FormatJSON},
		{"--json 压过 --format table", true, false, "table", output.FormatJSON},
		{"--csv 压过 --format json", false, true, "json", output.FormatCSV},
		{"--json 与 --csv 同时给时 --json 优先（switch 的先后）", true, true, "", output.FormatJSON},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			resetCommon(t)
			JSON, CSV, Format = c.json_, c.csv_, c.fmtS
			if got := OutputFormat(); got != c.want {
				t.Errorf("得 %v，想 %v", got, c.want)
			}
		})
	}
}

func TestCurrentMetaWithoutProvider(t *testing.T) {
	resetCommon(t)
	MetaProvider = nil
	if got := CurrentMeta(); !reflect.DeepEqual(got, output.Meta{}) {
		t.Errorf("没注入 provider 时该返回零值（而不是 panic），得 %+v", got)
	}
}

func TestCurrentMetaDelegatesToProvider(t *testing.T) {
	resetCommon(t)
	want := output.Meta{Env: "E1", Zone: "36", Account: "ds", Readonly: true}
	MetaProvider = func() output.Meta { return want }
	if got := CurrentMeta(); !reflect.DeepEqual(got, want) {
		t.Errorf("得 %+v，想 %+v", got, want)
	}
}

// TestConfigHintAndResolveConfigDelegate 钉的是**接线**，不是取值：
// 这两个函数的取值依赖环境（便携包判定、用户配置目录），断言具体值会在别人的机器上红。
func TestConfigHintAndResolveConfigDelegate(t *testing.T) {
	t.Setenv("TT_CONFIG", "") // 隔离开发机上的真实 TT_CONFIG,否则会盖过显式 --config(见 config.ResolvePath)
	resetCommon(t)
	if got, want := ConfigHint(), config.DefaultConfigPathHint(); got != want {
		t.Errorf("ConfigHint 该原样转发 config.DefaultConfigPathHint：得 %q，想 %q", got, want)
	}
	for _, allowMissing := range []bool{false, true} {
		resetCommon(t)
		ConfigPath = ""
		gotPath, gotErr := ResolveConfig(allowMissing)
		wantPath, wantErr := config.ResolvePath("", allowMissing)
		if gotPath != wantPath {
			t.Errorf("allowMissing=%v：路径得 %q，想 %q", allowMissing, gotPath, wantPath)
		}
		if (gotErr == nil) != (wantErr == nil) {
			t.Errorf("allowMissing=%v：错误对不上（得 %v，想 %v）", allowMissing, gotErr, wantErr)
		}
	}
	// 显式给了 --config 时它必须压过一切解析规则。
	resetCommon(t)
	ConfigPath = `D:\somewhere\tt-config.json`
	got, err := ResolveConfig(true)
	if err != nil || got != ConfigPath {
		t.Errorf("显式 --config 该原样返回：得 (%q, %v)", got, err)
	}
}

func TestWebFrontendReturnsWhateverWasInjected(t *testing.T) {
	resetCommon(t)
	if WebFrontend() != nil {
		t.Error("没注入时该是 nil（调用方据此回落引导页）")
	}
}

// TestPrintJSONEmitsCompactJSON 打印是纯输出，可以抓回来对；stdout 的 JSON 必须是一行。
func TestPrintJSONEmitsCompactJSON(t *testing.T) {
	_, out := testkit.CaptureStdout(t, func() int {
		if err := PrintJSON(map[string]any{"a": 1}); err != nil {
			t.Errorf("PrintJSON: %v", err)
		}
		return 0
	})
	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("打出来的不是合法 JSON：%v\n%s", err, out)
	}
	if got["a"] != float64(1) {
		t.Errorf("内容不对：%v", got)
	}
	if got := strings.TrimSpace(out); got != `{"a":1}` {
		t.Errorf("该是紧凑单行 JSON，得 %q", got)
	}
}

// TestUnknownSubcommandListsWhatIsAvailable 这条错误存在的理由就是**替调用方省一步**：
// 只给一句 "unknown command" 的话，下一步还得自己去 --help。
func TestUnknownSubcommandListsWhatIsAvailable(t *testing.T) {
	// 夹具要**可运行**（带 Run）—— cobra 的 IsAvailableCommand() 对"既没 Run 也没子命令"
	// 的空壳返回 false，拿空壳当夹具会得到一份空清单（第一版就是这么写的，红的却不是被测代码）。
	// 真实的 tt dict / tt debug 子命令都有 Run。
	noop := func(*cobra.Command, []string) {}
	root := &cobra.Command{Use: "tt"}
	root.AddCommand(&cobra.Command{Use: "alpha", Run: noop})
	root.AddCommand(&cobra.Command{Use: "beta", Run: noop})
	root.AddCommand(&cobra.Command{Use: "secret", Hidden: true, Run: noop})

	err := UnknownSubcommand(root, []string{"nope"})
	if err == nil {
		t.Fatal("该返回一个错误（不设 RunE 的组命令会让 cobra 打份帮助就退 0，" +
			"把「我打错了命令」读成成功）")
	}
	msg := err.Error()
	for _, want := range []string{`"nope"`, "tt", "alpha", "beta"} {
		if !strings.Contains(msg, want) {
			t.Errorf("错误里该有 %q：%s", want, msg)
		}
	}
	if strings.Contains(msg, "secret") {
		t.Errorf("隐藏的子命令不该列出来：%s", msg)
	}
}
