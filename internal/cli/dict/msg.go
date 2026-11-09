package dict

// tt dict msg:查询系统消息档(gzze_t,由作业 azzi920 维护;运行时 cl_err/cl_getmsg
// 按编号+语言取用)。AI 在代码里遇到消息编号(如 std-00006 / azz-00041 / -263)
// 时用它查文本、建议处理方式与技术细节。
//
// 查询条件照 azzi920 的查询画面来,同时成立:编号(gzze001)、信息语句
// (gzze003,--text)、信息类型(gzze007,--type)、状态(gzzestus,--status)、
// 建议运行作业(gzze005,--prog)、语言别(gzze002,--lang,缺省 zh_CN)。语言进
// WHERE,不再像以前那样"查出全部语言行再挑一行";唯一的例外是零结果时补查一次
// "这组条件在哪些语言下有行"(printMsgMiss),那一步刻意不带语言。
// 编号/语句的精确与模糊之别见 db.MsgQuery;语言为什么不算"筛子"见 MsgQuery.HasFilter。
//
// 输出:命中一条打详情块(与过去单条查询完全一致);命中多条打列表 —— 搜索能
// 命中几十上百条,一条一块详情没法看,要看某条再拿编号精确查一次(--full 可
// 强制全部打详情)。
//
// 数据源与 r.t/r.v 等一致(本地 SQLite 镜像 / --env <环境> 远程直查)。

import (
	"fmt"
	"strings"

	"tt/internal/dict/db"
	"tt/internal/output"

	"github.com/spf13/cobra"
)

var (
	msgLang   string
	msgText   string
	msgType   string
	msgStatus string
	msgProg   string
	msgFull   bool
)

var msgCmd = &cobra.Command{
	Use:   "msg [编号[,<编号>...]]",
	Short: "查询系统消息 (编号/语句/类型/状态/建议作业 多条件)",
	Long: `查询系统消息:所有提示/报错消息的记录。程序报错/日志里出现编号(如
std-00006)时,查它的完整文本、建议处理方式与技术细节。

查询条件(同时成立;编号/语句/类型/状态/建议作业任给其一即可):
  编号     位置参数,逗号分隔可给多个(gzze001);默认精确匹配,写了 * 当通配
           (azz-* 列出该模块全部消息,*-00006 找同流水号)
  语句     --text <关键字>(gzze003);默认子串匹配(如 --text 密码),同样支持 *
  类型     --type <0|1|2>(gzze007,SCC 106: 0=警告 1=错误 2=资讯)
  状态     --status <Y|N>(gzzestus: Y=启用 N=停用)
  建议作业  --prog <作业编号>(gzze005);查"哪些消息建议跑这个作业"
  语言     --lang <语言别>(gzze002);缺省 zh_CN。它决定看哪个语言的份,
           不是筛选条件 —— 只给 --lang 等于没筛
类型/状态/建议作业都可以逗号分隔给多个(列内是 OR),不同 flag 之间是 AND。
编号形如 std-00001 / azz-00041 / lib-xxxxx(类型-流水);负整数为 SQLCODE
(如 -263,查询时需用 -- 分隔: tt dict msg -- -263)。

命中一条打详情;命中多条打列表(编号/类型/状态/语句),要看某条的完整内容
再拿它的编号查一次,或加 --full 全部打详情。
返回: 文本、建议处理、建议作业(附名称)、技术细节、类型(0=警告 1=错误
2=资讯)、状态。`,
	Example: `  tt dict msg std-00006                  # 精确查一个编号(zh_CN)
  tt dict msg "std-00006,azz-00041"      # 一次查多个编号
  tt dict msg "azz-*" --limit 0          # 列 azz 模块的全部消息
  tt dict msg --text 密码                  # 按信息语句的子串搜
  tt dict msg "baa-*" --text 密码          # 编号与语句两个条件同时成立
  tt dict msg --type 1 --status Y        # 只看启用中的错误消息
  tt dict msg --type 1,2 --text 审核       # 类型多选 + 语句子串
  tt dict msg --prog axmt500             # 哪些消息建议跑 axmt500
  tt dict msg azz-00041 --lang zh_TW     # 换语言(语言也是查询条件)
  tt dict msg -- -100                    # 负整数编号(SQLCODE)需用 -- 分隔
  tt dict msg baa-00010 --env 正式区       # 远程直查该环境`,
	Args: msgArgs,
	RunE: runMsg,
}

// 两个封闭值域。类型码来自 SCC 106(gzze007),状态码是 gzzestus 的 Y/N。
// 都用码、不用中文标签:家族里其它命令一律用码(tt dict scc 4、--lang zh_TW),
// 两种写法并存只会让 --help 和将来的 --conditions 说不清该写哪个 —— 报错里
// 自带中文对照表,填错了当场能看到该填什么。
var (
	msgTypeValues   = []string{"0", "1", "2"}
	msgStatusValues = []string{"Y", "N"}
)

// checkMsgEnum 校验枚举码(不分大小写);非法时的报错带值域对照表。
func checkMsgEnum(subject, field string, got, allowed []string, label func(string) string) error {
	for _, v := range got {
		ok := false
		for _, a := range allowed {
			if strings.EqualFold(v, a) {
				ok = true
				break
			}
		}
		if ok {
			continue
		}
		hints := make([]string, len(allowed))
		for i, a := range allowed {
			hints[i] = a + "=" + label(a)
		}
		return fmt.Errorf("未知的%s '%s'(%s);可用值: %s", subject, v, field, strings.Join(hints, " "))
	}
	return nil
}

// msgArgs 校验参数:位置参数只收一个(编号,逗号分隔给多个),外加 --type/--status
// 两个枚举 flag 的取值。
//
// 枚举校验放在这里而不是 RunE:cobra 的顺序是 ValidateArgs → PersistentPreRunE
// (命令组在这一步打开查询数据源)→ RunE。放进 RunE 意味着"值填错了"也要先连一次库,
// 而且报错会被数据源的毛病(如本地库不存在)盖住 —— 填错一个字母却看到"数据库文件
// 未找到",是能把人带偏的。
func msgArgs(cmd *cobra.Command, args []string) error {
	if len(args) > 1 {
		return fmt.Errorf("位置参数太多(%d 个): %s\n"+
			"编号只占一个位置参数(逗号分隔可给多个);`--` 之后的都算参数,`--env` 这类要写在 `--` 前面",
			len(args), strings.Join(args, " "))
	}
	if err := checkMsgEnum("信息类型", "gzze007", splitNames(msgType), msgTypeValues, msgTypeLabel); err != nil {
		return err
	}
	return checkMsgEnum("状态", "gzzestus", splitNames(msgStatus), msgStatusValues, statusLabel)
}

func runMsg(cmd *cobra.Command, args []string) error {
	var codes []string
	if len(args) == 1 {
		codes = splitNames(args[0])
	}
	q := db.MsgQuery{
		Codes: codes, Text: msgText, Lang: msgLang,
		Types: splitNames(msgType), Status: splitNames(msgStatus), Progs: splitNames(msgProg),
	}
	if !q.HasFilter() {
		return fmt.Errorf("至少给一个查询条件:编号(位置参数)或 --text / --type / --status / --prog\n" +
			"例: tt dict msg std-00006 / tt dict msg --text 密码 / tt dict msg --type 1 --status Y\n" +
			"(--lang 只决定看哪个语言的份,不算筛选条件)")
	}

	rows, err := GetDB().QueryMsgs(q)
	if err != nil {
		if db.IsMissingTable(err) {
			return missingTableErr("消息档 (gzze_t/gzzal_t)")
		}
		return err
	}
	if len(rows) == 0 {
		if Format() == output.FormatTable {
			printMsgMiss(q)
			return nil
		}
		// 空结果也是**结构化的**空结果:data 为空数组、行数 0,退出 0。
		// 以前 --json 遇到空结果会打一句中文散文到 stdout,管道那头拿到的是非 JSON。
		return emit([]db.MsgRow{}, msgColumns, nil)
	}
	if Format() != output.FormatTable {
		return emit(rows, msgColumns, msgTable(rows))
	}
	if len(rows) == 1 || msgFull {
		for i := range rows {
			printMsg(&rows[i])
		}
		return nil
	}
	printMsgList(rows)
	return nil
}

// printMsg 文本模式输出单条消息。
func printMsg(m *db.MsgRow) {
	fmt.Printf("=== %s (%s) ===\n", m.Code, m.Lang)
	fmt.Printf("类型:     %s\n", typeWithCode(msgTypeLabel(m.TypeCode), m.TypeCode))
	fmt.Printf("状态:     %s\n", statusLabel(m.Status))
	fmt.Printf("文本:     %s\n", m.Text)
	if m.Action != "" {
		fmt.Printf("建议处理: %s\n", m.Action)
	}
	if m.ExecProg != "" && m.ExecProg != ":EXEPROG" {
		line := "建议作业: " + m.ExecProg
		if m.ProgName != "" {
			line += " (" + m.ProgName + ")"
		}
		fmt.Println(line)
	}
	if m.Detail != "" {
		fmt.Println("技术细节 (gzze006):")
		for _, line := range strings.Split(m.Detail, "\n") {
			fmt.Println("  " + line)
		}
	}
	fmt.Println()
}

// printMsgList 多条命中时的列表。搜索能命中几十上百条(实测 --text 密码 =
// 65 条),一条一块详情没法看;列表给编号,要细节再用编号精确查一次。
func printMsgList(rows []db.MsgRow) {
	shown := rows
	if limitOf() > 0 && len(shown) > limitOf() {
		shown = shown[:limitOf()]
	}
	table := make([][]string, 0, len(shown))
	for _, m := range shown {
		table = append(table, []string{m.Code, orDash(msgTypeLabel(m.TypeCode)),
			orDash(statusLabel(m.Status)), msgBrief(m.Text)})
	}
	output.PrintTable([]string{"编号", "类型", "状态", "信息语句"}, table)

	fmt.Printf("\n共 %d 条", len(rows))
	if len(shown) < len(rows) {
		fmt.Printf(",仅显示前 %d 条(--limit 0 显示全部)", len(shown))
	}
	fmt.Println(";某条的完整内容(建议处理/建议作业/技术细节)用 tt dict msg <编号> 查")
}

// msgColumns 消息行的列定义。JSON 的键与 CSV 的表头**同源于 MsgRow 这一份 DTO** ——
// 从前两处各自手写,结果已经漂了:CSV 漏掉「强制开窗」、把「类型码」写成「类型」。
var msgColumns = []string{"编号", "语言", "类型", "状态", "文本",
	"建议处理", "建议作业", "作业名称", "技术细节", "强制开窗"}

// msgTable 把消息行摊平成与 msgColumns 同序的表格(CSV 主体用)。
// 导出语义:不受 --limit 约束,给的就是全部命中。
func msgTable(rows []db.MsgRow) [][]string {
	t := make([][]string, 0, len(rows))
	for _, m := range rows {
		t = append(t, []string{m.Code, m.Lang, msgTypeLabel(m.TypeCode), statusLabel(m.Status),
			m.Text, m.Action, m.ExecProg, m.ProgName, m.Detail, m.ForceWin})
	}
	return t
}

// printMsgMiss 零结果时的提示。补查一次"同一组条件在别的语言下有没有行":
// 源系统按 (编号,语言) 精确匹配、没有自动回退,"换个语言就有了"是零结果最常见
// 的原因,说清可用语言比干说"没找到"有用。
func printMsgMiss(q db.MsgQuery) {
	if avail, err := GetDB().QueryMsgLangs(q); err == nil && len(avail) > 0 {
		if msgOneCode(q) {
			fmt.Println(langMissMsg(q.Codes[0], q.Lang, avail))
		} else {
			fmt.Printf("未找到匹配的消息的 %s 语言行;可用语言: %s(用 --lang 指定)\n",
				q.Lang, joinAvail(avail))
		}
		return
	}
	if msgOneCode(q) {
		fmt.Printf("未找到消息编号 '%s'。\n", q.Codes[0])
		fmt.Println("提示: 编号默认精确匹配,模糊找要写 * (如 azz-*);也可能是该编号没有 zh_CN 行(换 --lang 试)。")
		return
	}
	fmt.Printf("未找到匹配的消息(%s)。\n", msgQueryDesc(q))
	switch {
	case q.Text != "":
		fmt.Println("提示: 语句是子串匹配,可换更短的词;或换 --lang(该语句可能只有其它语言)。")
	case len(q.Codes) > 0:
		fmt.Println("提示: 编号默认精确匹配,模糊找要写 * (如 azz-*)。")
	default:
		fmt.Println("提示: 类型/状态/建议作业之间是 AND,同时成立时没有数据;去掉一个再试。")
	}
}

// msgOneCode 是否"只给了一个编号、别的条件都没给"——这种零结果能点名编号,
// 提示比通用的那句具体。
func msgOneCode(q db.MsgQuery) bool {
	return len(q.Codes) == 1 && q.Text == "" && len(q.Types) == 0 && len(q.Status) == 0 && len(q.Progs) == 0
}

// msgQueryDesc 把查询条件描述成一行,用于"未找到"时说明查的是什么。
func msgQueryDesc(q db.MsgQuery) string {
	var parts []string
	if len(q.Codes) > 0 {
		parts = append(parts, "编号="+strings.Join(q.Codes, ","))
	}
	if q.Text != "" {
		parts = append(parts, "语句含 "+q.Text)
	}
	if len(q.Types) > 0 {
		parts = append(parts, "类型="+strings.Join(q.Types, ","))
	}
	if len(q.Status) > 0 {
		parts = append(parts, "状态="+strings.Join(q.Status, ","))
	}
	if len(q.Progs) > 0 {
		parts = append(parts, "建议作业="+strings.Join(q.Progs, ","))
	}
	if q.Lang != "" {
		parts = append(parts, "语言="+q.Lang)
	}
	return strings.Join(parts, " ")
}

// msgBrief 信息语句的单行摘要:列表一行一条,多行文本只取首行、过长截断。
func msgBrief(s string) string {
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		s = s[:i]
	}
	s = strings.TrimSpace(s)
	const max = 60
	if r := []rune(s); len(r) > max {
		return string(r[:max]) + "…"
	}
	return s
}

// msgTypeLabel 讯息类型 (SCC 106): 0警告/1错误/2资讯。
func msgTypeLabel(code string) string {
	switch code {
	case "0":
		return "警告"
	case "1":
		return "错误"
	case "2":
		return "资讯"
	default:
		return code
	}
}

// statusLabel 状态 Y/N → 启用/停用。
func statusLabel(code string) string {
	switch code {
	case "Y":
		return "启用"
	case "N":
		return "停用"
	default:
		return code
	}
}

func init() {
	msgCmd.Flags().StringVar(&msgLang, "lang", "zh_CN", "语言别 (gzze002;缺省 zh_CN)。决定看哪个语言的份,不算筛选条件")
	msgCmd.Flags().StringVar(&msgText, "text", "", "按信息语句 (gzze003) 关键字查(默认子串匹配,支持 *)")
	msgCmd.Flags().StringVar(&msgType, "type", "", "信息类型 (gzze007; 0=警告 1=错误 2=资讯;可逗号分隔多个)")
	msgCmd.Flags().StringVar(&msgStatus, "status", "", "状态 (gzzestus; Y=启用 N=停用;可逗号分隔多个)")
	msgCmd.Flags().StringVar(&msgProg, "prog", "", "建议运行作业 (gzze005);查哪些消息建议跑这个作业;可逗号分隔多个")
	msgCmd.Flags().BoolVar(&msgFull, "full", false, "多条命中时也逐条打详情(默认只列表)")
	// 负整数编号(SQLCODE,如 -263)会被 flag 解析吃掉,报 "unknown shorthand flag: '2' in -263",
	// 看不出该怎么办。这里翻成可操作的提示(唯一需要 -- 分隔的参数位置)。
	msgCmd.SetFlagErrorFunc(func(c *cobra.Command, err error) error {
		if strings.Contains(err.Error(), "unknown shorthand flag") {
			return fmt.Errorf("%w\n提示: 负整数编号(SQLCODE)要写成 `tt dict msg -- -263`(`--` 之后才当参数解析)", err)
		}
		return err
	})
	Group.AddCommand(msgCmd)
}
