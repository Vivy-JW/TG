package tzs

// tzs_formcheck.go —— `tt dev tzs formcheck`：**静态**把"应然"与"实然"比一遍。
//
// 这是 M3 的消费面入口（契约 `.agents/plan/tt-genero-formdef-contract.md` 的 C1/C7 + A15，
// 用户决策 D-NEXT）：一条命令回答"设计器里的这张表单，跟编译后那张表单是不是同一件事"。
//
// 三条纪律：
//
//  1. **不开引擎、不连服务器、不碰工作区。** 只读两个文件：`.42f`（编译后，应然）与
//     `.tzs`/`.4fd`（tzs 生成物，实然）。所以它可以在没有设计器、没有工作区、没有会话的
//     机器上跑 —— 这与本包其余动词（都要引擎）是**相反的**前提，别把两者混起来。
//  2. **比对全部在 internal/dev/tzs 里**（`CheckForm42F`），命令层只负责读文件、排版、退出码。
//     判定口径绝不在这里抄第二份。
//  3. **未判定不计入错误**（A15）：有差异退 6、只有未判定退 0。判断标准只有一条 ——
//     拿不准的东西不参与判定，但**必须打印出来**。

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tt/internal/cli/dev/common"
	"tt/internal/dev/tzs"
	"tt/internal/form42f"
)

// tzsFormCheckUsage 是这个动词的帮助（`tt dev tzs formcheck --help`）。
const tzsFormCheckUsage = `tt dev tzs formcheck —— 静态比对：应然（.42f）↔ 实然（tzs 生成物）

用法：
  tt dev tzs formcheck --form42f <表单.42f> --tzs <包.tzs|表单.4fd> [--json]

  --form42f  编译后的表单文件（**应然**：声明期权威）
  --tzs      tzs 生成物：表 单包 .tzs/.tzv（取其包内 .4fd），或摊在工作区里的明文 .4fd
  --json     结构化输出（键名见 internal/dev/tzs/README.md）

比对四条规则（按规则分组输出，每条给 colName、两侧值、为什么）：
  存在性   tzs 声明的 colName 在 .42f 里没有
  编号     两侧都有但 fieldId 不同 ⇒ 可能是另一张表单/版本不同
  可写性   NoEntry / Required / NotNull 两侧相反（方向：.42f = 应然，tzs = 实然）
  控件族   Widget 与 tzs 侧控件**确定**不等价

拿不准的（控件名不在已知表、某一侧没带控件类型、colName 重名）进"未判定"，**不计入错误**。

退出码：0 一致（含"只有未判定"） / 2 参数或读取错 / 6 有确定差异
`

// cmdTzsFormCheck 是动词入口。args 是动词名之后的原始参数。
func cmdTzsFormCheck(args []string) int {
	fs := flag.NewFlagSet("tzs formcheck", flag.ContinueOnError)
	fs.Usage = func() { fmt.Fprint(os.Stderr, tzsFormCheckUsage) }
	f42Path := fs.String("form42f", "", "编译后的表单文件（.42f）")
	tzsPath := fs.String("tzs", "", "tzs 生成物（.tzs/.tzv 包，或明文 .4fd）")
	asJSON := fs.Bool("json", false, "输出 JSON")
	if err := common.ParseArgs(fs, args, "form42f", "tzs"); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0 // `--help` 是"还能用的那条命令"，不该按用法错退 2
		}
		return 2
	}
	if fs.NArg() > 0 {
		// 位置参数一律报错：这个命令的两个输入都有名字（--form42f / --tzs），
		// 靠"第几个参数是什么"去猜是最容易悄悄错的一种写法。
		return common.Fail(tzsCheckError{Msg: "不接受位置参数：请用 --form42f <文件> --tzs <包|文件>",
			Detail: []string{"收到的位置参数：" + strings.Join(fs.Args(), " ")}}, *asJSON)
	}
	if *f42Path == "" || *tzsPath == "" {
		return common.Fail(tzsCheckError{Msg: "两个输入都要给：--form42f <表单.42f> --tzs <包.tzs|表单.4fd>"},
			*asJSON)
	}

	// 应然侧：`.42f` 走 internal/form42f 的解析器（不在这里重写）
	text, err := os.ReadFile(*f42Path)
	if err != nil {
		return common.Fail(tzsCheckError{Msg: "读不到 .42f：" + err.Error()}, *asJSON)
	}
	form := common.StripIdentitySuffix(*f42Path) // 表单名按文件名（`WITH FORM` 那一侧的 scope）
	want, err := form42f.ParseForm42F(form, string(text))
	if err != nil {
		return common.Fail(tzsCheckError{Msg: "解析 .42f 失败：" + err.Error()}, *asJSON)
	}

	// 实然侧：包（zip）或摊出来的明文
	var got *tzs.TzsForm
	switch strings.ToLower(filepath.Ext(*tzsPath)) {
	case ".tzs", ".tzv":
		got, err = tzs.ReadTzsFormPackage(*tzsPath)
	case ".4fd":
		got, err = tzs.ReadTzsForm4FD(*tzsPath)
	default:
		return common.Fail(tzsCheckError{Msg: "`--tzs` 只吃 .tzs / .tzv 包或明文 .4fd",
			Detail: []string{"输入：" + *tzsPath}}, *asJSON)
	}
	if err != nil {
		return common.Fail(tzsCheckError{Msg: "读 tzs 侧失败：" + err.Error()}, *asJSON)
	}

	check := tzs.CheckForm42F(want, got)
	if *asJSON {
		common.EmitJSON(os.Stdout, check.JSONReport())
	} else {
		fmt.Print(check.String())
	}
	if !check.Consistent() {
		return tzsCheckExitDiff
	}
	return 0
}

// tzsCheckExitDiff 是"有确定差异"的退出码。
//
// 为什么是 6 而不是 3/4：本命令组的码表是 0/1/2/4/5（README「退出码」一节，明确"没有 3"），
// 6 借自表单契约 A13 的同一含义（「体检不过」），不新造语义、也不占 3/4 的既有含义。
const tzsCheckExitDiff = 6

// tzsCheckError 是"参数或环境错"，退出码 2（本命令组的口径）。
type tzsCheckError struct {
	Msg    string
	Detail []string
}

func (e tzsCheckError) Error() string {
	if len(e.Detail) == 0 {
		return e.Msg
	}
	return e.Msg + "（" + strings.Join(e.Detail, "；") + "）"
}

// ExitCode 让 common.Fail / ExitCodeOf 把它映射成 2（参数或环境错）。
func (e tzsCheckError) ExitCode() int { return 2 }
