package tzc

import "strings"

// 新函数头模板 —— 与设计器 FunctionGenerator.Generate 同形。
//
// 依据（源码 + 真实包双向印证）：
//   - 设计器：FunctionGenerator.cs:20-67 逐个 AppendLine 出「80 个 # 的框 + 固定字段行」，
//     紧接着才是 `PUBLIC FUNCTION ...`，也就是说这段是**签名行之前的描述块**；
//   - 真实包：语料 armt100.tap 的 function.armt100_qrystr，描述块正是
//     `\n` + 80 个 # + `# Descriptions...: 串查單號` + … + `# Modify.........` + 80 个 #。
//
// 三条设计要求：
//  1. **顶部一个空行**：这段文本最终进 .tap 的 CDATA，空行保证新函数不与上一个函数的
//     END 挨着（也保证渲染后的文档里围栏行下面是空行）；
//  2. `# Date & Author..:` 保留占位符「日期 By 作者」，不自动填今天日期/用户名
//     —— 红线 R7：同一输入必须产出逐字节相同的工作区，不引入时钟；
//  3. 只填已知值：`Usage` 行的函数名用真实新函数名（否则会把示例里的函数名
//     固化进每个新函数），其余占位符逐字保留，交给人/AI 填。

// newFnRule 是模板里那两条 80 个 `#` 的分隔线。
const newFnRule = "################################################################################"

// newFnHeaderTemplate 造新函数的描述块文本（以空行开头，每行以 \n 结尾）。
func newFnHeaderTemplate(fn string) string {
	var b strings.Builder
	b.WriteString("\n") // ① 顶部空行
	b.WriteString(newFnRule + "\n")
	b.WriteString("# Descriptions...: 描述说明\n")
	b.WriteString("# Memo...........:\n")
	b.WriteString("# Usage..........: CALL " + fn + "(传入参数)\n")
	b.WriteString("#                  RETURNING 回传参数\n")
	b.WriteString("# Input parameter: 传入参数变量1   传入参数变量说明1\n")
	b.WriteString("#                : 传入参数变量2   传入参数变量说明2\n")
	b.WriteString("# Return code....: 回传参数变量1   回传参数变量说明1\n")
	b.WriteString("#                : 回传参数变量2   回传参数变量说明2\n")
	b.WriteString("# Date & Author..: 日期 By 作者\n")
	b.WriteString("# Modify.........:\n")
	b.WriteString(newFnRule + "\n")
	return b.String()
}
