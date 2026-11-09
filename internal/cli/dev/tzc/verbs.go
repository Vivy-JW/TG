// 动词的接线信息：cobra 叶命令的展示面（Use/Short/Long）+ 核心函数。
//
// 拆成一份清单是刻意的：父包 register.go 只负责把清单挂上 cobra 树（路由、组级帮助、
// 退出码透传），动词的"它叫什么、说明书写什么、核心是哪个函数"都住在这层，
// 改动词不用碰父包。
package tzc

// Verb 是一条 tzc 动词的接线信息。Run 收**原始参数**（含 flag）——
// 参数解析权在动词自己的解析器（cli.go 的 parseArgs），cobra 不碰。
type Verb struct {
	Use   string
	Short string
	Long  string
	Run   func(args []string) int
}

// Verbs 按帮助顺序列出八个动词，外加一个生成修改标记的 uuid 小工具。
func Verbs() []Verb {
	return []Verb{
		{"export <pkg.tzc>", "渲染带围栏的 4GL 工作区（只读包）", exportLong, cmdExport},
		{"status [<dir>]", "报告工作区相对上次 export/apply 的改动（不写盘）", statusLong, cmdStatus},
		{"verify [<dir>]", "不写盘跑完整验证管线（gate1 + gate2）", verifyLong, cmdVerify},
		{"apply [<dir>]", "验证管线全阶段 + 原子写包 + git commit（唯一会改 .tzc 的命令）", applyLong, cmdApply},
		{"unlock [<dir>]", "框架解锁：单向状态迁移，只改 workspace", unlockLong, cmdUnlock},
		{"rename [<dir>] <旧函数名> <新函数名>", "结构事务：改自订定义点的函数名（只改 workspace）", renameLong, cmdRename},
		{"newfn [<dir>]", "新增自订定义点（只改 workspace）", newfnLong, cmdNewfn},
		{"uuid", "生成 16 位修改标记（用户拒绝提供单号时用）", uuidLong, cmdUUID},
		{"selftest", "内置 31 项对抗用例（全部合成包）", selftestLong, cmdSelftest},
	}
}

// TzcLong 是 tt dev tzc 的组级说明（与 Usage 的生命周期段同源措辞）。
const TzcLong = `.tzc 代码包管线：Package → Document → Workspace（export，只读包）→ 编辑 prog.full.4gl →
verify（gate1 + gate2）/ apply（三道闸门 + 原子写包）。

四个生命周期动词 + unlock 状态迁移 + uuid 小工具 + selftest。
改框架区段必须先 unlock：解锁改变的是门，不是所有房间（锚点区段永远只读）。

报错定位：验证失败会打印「prog.full.4gl:<行号>」+ 该行内容
（基线侧问题给「.tdev/base.full.4gl:<行号>」）；--json 里有 file/line/snippet 字段。

退出码：0 成功 / 2 包格式错 / 3 验证失败 / 4 写入被拒 / 5 IO·环境失败

工作区动词的 <dir> 可以省略：先 cd 进工作区，命令就不用再写目录。
  cd D:\pkg\capt110-ws
  tt dev tzc status          # = tt dev tzc status D:\pkg\capt110-ws
  tt dev tzc rename adzi999_calc adzi999_count   # 省略 <dir> 时 rename 收 2 个位置参数`

// 各动词的 Long：用法行 + 关键开关，与 Usage 里的对应行同源措辞；
// 动词缺参数时核心自己打的"用法："也是同一行，两处不会漂。
const (
	exportLong = `把 .tzc 代码包渲染成带围栏的 4GL 工作区（只读包）。

用法：tt dev tzc export <pkg.tzc> [-o <dir>] [--only <点名>...] [--json]
  -o 省略时默认导出到 <包所在目录>/<程序名>-ws（身份后缀 (c)/(s) 会去掉）
  --only 部分导出：只渲染这些点为可编辑（可重复/逗号分隔）
  --allow-sec 已废弃：请改用 tt dev tzc unlock <dir>`

	statusLong = `报告工作区相对上次 export/apply 的改动（含行号，不写盘）。

用法：tt dev tzc status [<dir>] [--json]   # <dir> 省略时用当前目录`

	verifyLong = `不写盘跑完整验证管线（gate1 + gate2）。

用法：tt dev tzc verify [<dir>] [--json] [--strict]   # <dir> 省略时用当前目录
  --strict 把 warn 级发现也算作失败（供 CI 使用）`

	applyLong = `验证管线全阶段 + 原子写包 + git commit（唯一会改 .tzc 的命令）。

用法：tt dev tzc apply [<dir>] [-o <pkg.tzc>] [--dry-run] [--yes] [--json]   # <dir> 省略时用当前目录
  --dry-run 跑完管线并输出逐条目 diff，不落盘、不 commit
  --yes 对「写区段（解开框架）」做二次确认`

	unlockLong = `框架解锁：单向状态迁移，只改 workspace（落到 .tzc 仍走 apply）。

用法：tt dev tzc unlock [<dir>] [--yes] [--json]   # <dir> 省略时用当前目录
  解开不可逆：解开后规格调整不再自动生成程序代码；需要显式 --yes（AI 不得自主加）`

	renameLong = `结构事务：改自订定义点的函数名（只改 workspace，落盘仍走 apply）。

用法：tt dev tzc rename [<dir>] <旧函数名> <新函数名> [--scope PUBLIC|PRIVATE] [--desc <描述>] [--json]
  <dir> 省略时用当前目录；围栏 fn + 签名行 + 描述块三处原子同步`

	newfnLong = `新增自订定义点（只改 workspace，落盘仍走 apply）。

用法：tt dev tzc newfn [<dir>] --type FUNCTION|DIALOG|REPORT [--name <名>] [--scope …] [--json]
  <dir> 省略时用当前目录；--name 省略则默认 <prog>_newfunc 并自动去重`

	selftestLong = `内置 31 项对抗用例（全部合成包，不需要真实语料）。

用法：tt dev tzc selftest [--json]`

	uuidLong = `生成一个 16 位十六进制修改标记（8 字节随机数）。

用法：tt dev tzc uuid [--json]
  用户在开工前拒绝提供 <单号>#<项次> 时，用它生成标记（工号写 agent）。
  命令行只负责生成 id；摘要行与 -s/-e 块的写法属于项目规范，见 tt-dev-tzc 技能。`
)
