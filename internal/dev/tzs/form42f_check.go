package tzs

// form42f_check.go —— **应然 vs 实然**的静态校验：`.42f`（编译后的表单，声明期权威）↔ tzs 生成物。
//
// 契约（唯一口径）：`.agents/plan/tt-genero-formdef-contract.md`
//   · **C1/C7**：`.42f` 是编译期权威，`.4fd` 的 `fieldId` **不是**稳定标识（B5：每次保存全量重排）；
//   · **A15/A18**：启发式**不得**参与判定 —— 拿不准就标"未判定"，且**不计入错误**。
// 用户决策 **D-NEXT**：M3 的目标是把离线字段表接到消费面（`dict` / `dev(tzs)`），**不做前端适配**。
//
// 三条纪律（与包 README 同一套）：
//
//  1. **本文件不重写解析器**。`.42f` 一律走 `form42f.ParseForm42F`（切片 A 已交付，509/509 对齐）；
//     这里只负责"tzs 侧的表单描述"与两侧比对。tzs 侧是 `.tzs` 包内的 `.4fd`（明文 XML）——
//     它和 `.42f` **属性同名同义**（`colName` / `fieldId` / `widget` / `noEntry` / `required` /
//     `notNull`），所以比对不需要任何换算表；这是选它当"实然"的唯一理由。
//
//  2. **不做 IO 里的业务判断**：`ParseTzsForm4FD` 是纯函数（不 IO、不 panic），包/文件读取
//     另有两个薄壳（`ReadTzsForm4FD` / `ReadTzsFormPackage`）。
//
//  3. **拿不准就不判**（A15）。四条规则里只有"确定性"的部分进差异；控件名不在已知表、
//     某一侧没带控件类型、两侧有重名 colName —— 全部进 **未判定**，且**不计入错误**。
//     `FormCheck.Consistent()` 只看确定差异，所以"全是未判定"仍是"一致"。
//
// 四组差异（顺序固定，见 `RuleOrder`）：
//
//	存在性  tzs 声明的 colName 在 `.42f` 里没有
//	编号    两侧都有但 `fieldId` 不同 ⇒ **可能是另一张表单/版本不同**（不是硬错，B5）
//	可写性  `NoEntry` / `Required` / `NotNull` 两侧相反（方向：`.42f` = 应然，tzs = 实然）
//	控件族  `Widget` 与 tzs 侧控件**确定**不等价
//
// **规则修正（2026-10-17，真实配对 `aimm200_wf` 把旧规则打穿之后，契约 A37/A38）**：
//
//   - **编号是基数感知的**（A37）：先算**全部唯一配对**的差值（tzs − `.42f`）；若恒定且非 0
//     （真实配对是 171 行恒差 +1 —— `.42f` 0 基、`.4fd` 1 基），就报**一条聚合说明**（`Note`），
//     不逐条报；差值散开（或配对数太少不足以谈分布）才逐条报"编号不同"。
//     旧规则裸比两侧 `fieldId`，在真实文件上 86/86 唯一配对全部命中 ⇒ 对"是不是另一张表单"零分辨力。
//   - **可写性按同属性对齐**（A38）：`required`↔`required`、`notNull`↔`notNull`、`noEntry`↔`noEntry`
//     各自比；**同一属性**不同才是"可能确定"；而 `notNull=true` 与 `required=false` 这类**跨属性组合**
//     只进"未判定"（疑似编译器把 NOT NULL 归一成必填，非设计变更），措辞一律"疑似"。
//
// 不做的（刻意）：不判 `Tag`（`.42f` 的 `TableColumn`/`PhantomColumn` 与 `.4fd` 的元素名不是同一套
// 词表，如实测 `.4fd` 里是 `<Phantom>`），不判几何/`tabIndex`（布局归设计器），不查字典、不猜中文名。

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"tt/internal/form42f"
)

// ── 差异分组 ───────────────────────────────────────────────────────────────

// Rule 是一条差异属于哪一组。顺序即 `RuleOrder`。
type Rule string

const (
	// RuleExist 存在性：tzs 侧声明的 colName 在 `.42f` 里没有。
	RuleExist Rule = "exist"
	// RuleFieldID 编号：两侧都有这个 colName，但 `fieldId` 不同。
	RuleFieldID Rule = "fieldid"
	// RuleWritable 可写性：`NoEntry` / `Required` / `NotNull` 两侧相反。
	RuleWritable Rule = "writable"
	// RuleWidget 控件族：`Widget` 与 tzs 侧控件确定不等价。
	RuleWidget Rule = "widget"
)

// RuleOrder 是输出与遍历的固定顺序（**不许按 map 顺序渲染**：同一份差异每次都要长一样）。
var RuleOrder = []Rule{RuleExist, RuleFieldID, RuleWritable, RuleWidget}

// Title 是规则的中文组名（输出里用在【】里）。
func (r Rule) Title() string {
	switch r {
	case RuleExist:
		return "存在性"
	case RuleFieldID:
		return "编号"
	case RuleWritable:
		return "可写性"
	case RuleWidget:
		return "控件族"
	}
	return string(r)
}

// ── 结果 ───────────────────────────────────────────────────────────────────

// Note 是一条**聚合说明**：它陈述一个跨"全部唯一配对"的事实（契约 A37），既不是确定差异，
// 也不是"未判定"。
//
// 为什么需要这一类：真实配对（`aimm200_wf`）里 86 条"编号不同"全部来自同一个原因 —— 两侧
// `fieldId` 的下标基数差 1。逐条报 86 遍既淹没报告、又**支撑不了任何结论**（它对"是不是另一张
// 表单"零分辨力）；报 1 条"恒差 +1"才是那份数据的真实形状。
//
// 但它**不是**差异（`Consistent()` 不看它 ⇒ 退出码不受影响），也**不许**被当成"没有问题"：
// 报告里必须真的印出来（A37 的"信息没丢"）。
type Note struct {
	Rule  Rule          `json:"rule"`            // 挂在哪条规则下（目前只有 RuleFieldID）
	Text  string        `json:"text"`            // 人读的说明（措辞只陈述事实，不猜哪一侧是 0 基）
	Pairs int           `json:"pairs,omitempty"` // 这个事实覆盖了多少对唯一配对
	Cols  []form42f.ColName `json:"cols,omitempty"`  // 样例 colName（人抽查用：说明降级了，但线索还在）
}

// Finding 是一条比对结论（差异或未判定）。
//
// 两侧的值**都要写出来**，因为"哪侧是应然、哪侧是实然"是排障的全部：F42 = `.42f`（应然/权威），
// Tzs = tzs 生成物（实然/待校对）。
type Finding struct {
	Rule      Rule        `json:"rule"`
	ColName   form42f.ColName `json:"colName"`
	F42       string      `json:"f42,omitempty"` // 应然侧的值（人可读：`fieldId=23` / `noEntry=false` / `Edit`）
	Tzs       string      `json:"tzs,omitempty"` // 实然侧的值
	Why       string      `json:"why"`           // 为什么算差异 / 为什么拿不准
	Undecided bool        `json:"undecided,omitempty"`
}

// FormCheck 是一次校验的全部结果。
type FormCheck struct {
	Form            string        `json:"form"`               // 表单名（两侧同一口径：调用方给的名字）
	DeclaredName42F string        `json:"declared_name_42f"`  // `.42f` 里 `<Form name>` 的原值（实测是 `*.tmp`，不能当 scope）
	ProgTzs         string        `json:"prog_tzs"`           // tzs 侧程序名（从 `.4fd` 文件名推）
	F42Fields       int           `json:"f42_fields"`         // `.42f` 条目数（含明细列/幻影列）
	TzsFields       int           `json:"tzs_fields"`         // tzs 侧条目数（带 `fieldId` 的元素）
	TzsUnnamed      int           `json:"tzs_unnamed"`        // tzs 侧没有 `colName` 的条目（不可比，只记数）
	Pairs           int           `json:"pairs"`              // 真正配成对的 colName 数
	Notes           []Note        `json:"notes,omitempty"`    // 聚合说明（A37）：不是差异，也不是未判定
	Findings        []Finding     `json:"findings,omitempty"` // 全部（差异 + 未判定），按文档顺序
	F42Only         []form42f.ColName `json:"f42_only,omitempty"` // 只在 `.42f` 里出现（信息，不算差异）
}

// NoteFor 返回挂在某条规则下的聚合说明。
func (c *FormCheck) NoteFor(r Rule) []Note {
	if c == nil {
		return nil
	}
	var out []Note
	for _, n := range c.Notes {
		if n.Rule == r {
			out = append(out, n)
		}
	}
	return out
}

// Diffs 返回**确定**差异（未判定不在里面）。
func (c *FormCheck) Diffs() []Finding {
	if c == nil {
		return nil
	}
	var out []Finding
	for _, f := range c.Findings {
		if !f.Undecided {
			out = append(out, f)
		}
	}
	return out
}

// Undecided 返回"未判定"条目（A15：不计入错误，只作诊断）。
func (c *FormCheck) Undecided() []Finding {
	if c == nil {
		return nil
	}
	var out []Finding
	for _, f := range c.Findings {
		if f.Undecided {
			out = append(out, f)
		}
	}
	return out
}

// Group 返回某一组里的差异（未判定不进组）。
func (c *FormCheck) Group(r Rule) []Finding {
	var out []Finding
	for _, f := range c.Diffs() {
		if f.Rule == r {
			out = append(out, f)
		}
	}
	return out
}

// Consistent 是"确定差异为 0"。**未判定不影响它**（A15）。
func (c *FormCheck) Consistent() bool { return len(c.Diffs()) == 0 }

// String 渲染成人可读的分组清单。
//
// 刻意**不给百分比**：一个"97% 一致"既不能定位、也不知道错在哪一侧。
func (c *FormCheck) String() string {
	if c == nil {
		return "（没有校验结果）"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "表单 %s：应然 .42f「%s」%d 条 ↔ 实然 tzs「%s」%d 条；配对 %d 对\n",
		c.Form, c.DeclaredName42F, c.F42Fields, c.ProgTzs, c.TzsFields, c.Pairs)
	fmt.Fprintf(&b, "差异 %d 条 · 未判定 %d 条（不计入错误）· 仅 .42f 有 %d 条 · tzs 无 colName %d 条\n",
		len(c.Diffs()), len(c.Undecided()), len(c.F42Only), c.TzsUnnamed)
	// 聚合说明（A37）：**不是**差异，但必须印出来 —— 否则就是"改到不报就完事"。
	if len(c.Notes) > 0 {
		fmt.Fprintf(&b, "\n【说明】%d 条（跨全部配对的聚合事实，不是差异、也不计入错误）\n", len(c.Notes))
		for _, n := range c.Notes {
			// 样例 colName 已经在 Text 里（`--json` 的消费者未必会去读 `cols`），这里不重复印。
			fmt.Fprintf(&b, "  · [%s] %s\n", n.Rule.Title(), n.Text)
		}
	}
	for _, r := range RuleOrder {
		g := c.Group(r)
		fmt.Fprintf(&b, "\n【%s】%d 条\n", r.Title(), len(g))
		for _, f := range g {
			fmt.Fprintf(&b, "  · %s  %s  ↔  %s\n      为什么：%s\n", f.ColName, f.F42, f.Tzs, f.Why)
		}
	}
	und := c.Undecided()
	fmt.Fprintf(&b, "\n【未判定】%d 条（不计入错误，按 A15 只做诊断）\n", len(und))
	for _, f := range und {
		fmt.Fprintf(&b, "  · %s [%s]  %s  ↔  %s\n      为什么：%s\n", f.ColName, f.Rule.Title(), f.F42, f.Tzs, f.Why)
	}
	if len(c.F42Only) > 0 {
		fmt.Fprintf(&b, "\n【仅 .42f 有】%d 条（信息：tzs 侧没声明这些栏目，不算差异）\n  %s\n",
			len(c.F42Only), joinColNames(c.F42Only, 12))
	}
	return b.String()
}

// JSONReport 是命令面 `--json` 的形状（键名在这里定；改键名要同步 README 与调用方）。
func (c *FormCheck) JSONReport() map[string]any {
	groups := map[string][]Finding{}
	for _, r := range RuleOrder {
		g := c.Group(r)
		if g == nil {
			g = []Finding{}
		}
		groups[string(r)] = g
	}
	und := c.Undecided()
	if und == nil {
		und = []Finding{}
	}
	notes := c.Notes
	if notes == nil {
		notes = []Note{}
	}
	only := c.F42Only
	if only == nil {
		only = []form42f.ColName{}
	}
	return map[string]any{
		"ok":                c.Consistent(),
		"form":              c.Form,
		"declared_name_42f": c.DeclaredName42F,
		"prog_tzs":          c.ProgTzs,
		"f42_fields":        c.F42Fields,
		"tzs_fields":        c.TzsFields,
		"tzs_unnamed":       c.TzsUnnamed,
		"pairs":             c.Pairs,
		"diff_count":        len(c.Diffs()),
		"undecided_count":   len(und),
		"note_count":        len(notes),
		"groups":            groups,
		"notes":             notes,
		"undecided":         und,
		"f42_only":          only,
	}
}

// ── tzs 侧的表单描述（`.4fd`）──────────────────────────────────────────────

// TzsField 是 tzs 生成物里的一个字段条目（`.4fd` 里带 `fieldId` 的元素）。
//
// 三个布尔属性**保留原文**（`"" = 未声明`）：比对时"未声明"与"声明为 false"不是一回事，
// 压成 bool 就再也分不开了（A15）。
type TzsField struct {
	Tag      string      // 元素名（Edit / ComboBox / TableColumn / Phantom …，仅诊断用）
	Name     string      // 协议名（如 formonly.pmdl001）
	ColName  form42f.ColName // 栏位身份
	FieldID  form42f.FieldID // 表单内编号（设计期分配 —— 与 `.42f` 的编号**可能不同物**，见 B5）
	Widget   string      // `widget` 属性（缺省退元素名）
	NoEntry  string      // `noEntry` 原值
	Required string      // `required` 原值
	NotNull  string      // `notNull` 原值
}

// TzsForm 是 tzs 侧一张表单的字段表。
type TzsForm struct {
	Prog         string        // 程序名（`.4fd` 文件名不含扩展名；读包时从 zip 条目推）
	DeclaredName string        // `<Form name=…>` 原值（与 `.42f` 的 DeclaredName 对照用）
	Fields       []TzsField    // 文档顺序
	Duplicates   []form42f.ColName // 同名 colName（重名 ⇒ 无法唯一对应，见 CheckForm42F）
	Skipped      int           // 有 `fieldId` 但不是整数（或空）而被跳过的元素数；不猜、不 panic
}

// Unnamed 返回没有 `colName` 的条目数（`NON_DATABASE` 字段：没有栏位身份，不可比）。
func (f *TzsForm) Unnamed() int {
	if f == nil {
		return 0
	}
	n := 0
	for _, fld := range f.Fields {
		if fld.ColName == "" {
			n++
		}
	}
	return n
}

// ParseTzsForm4FD 解析 tzs 生成物里的表单描述（`.4fd` 明文 XML）文本。
//
// **字段条目的判据是"带 `fieldId` 属性"**，理由（实测两包，2026-10-17）：
//   - `.4fd` 里每个控件元素（Edit/ComboBox/CheckBox/ButtonEdit/FFLabel/TextEdit/…）都带
//     `fieldId`，并且都带 `noEntry`/`required`/`notNull`（aapp320 47/47、aapt300 259/259）；
//   - `<RecordField>` **只带 `fieldIdRef`**（指回上面那个字段），它是**记录布局**不是表单字段 ——
//     若把它也算字段，每张字段都会被数两遍（aapp320 就有 56 条 RecordField）。
func ParseTzsForm4FD(prog, text string) (*TzsForm, error) {
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("tzs: %s 的表单描述是空的（空文件或空内容）", prog)
	}
	f := &TzsForm{Prog: prog}
	sawForm := false

	dec := xml.NewDecoder(strings.NewReader(text))
	dec.CharsetReader = func(_ string, in io.Reader) (io.Reader, error) { return in, nil }

	seen := map[form42f.ColName]bool{}
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("tzs: %s 的表单描述不是 XML（%v）", prog, err)
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		a := attrMapTzs(se.Attr)
		if se.Name.Local == "Form" && !sawForm {
			sawForm = true
			if v := a["name"]; v != "" {
				f.DeclaredName = v
			}
		}
		rawID, hasID := a["fieldId"]
		if !hasID {
			continue // Record/RecordField/Label/Grid 一类：不是字段条目
		}
		id, err := strconv.Atoi(strings.TrimSpace(rawID))
		if err != nil {
			f.Skipped++
			continue
		}
		fld := TzsField{
			Tag:      se.Name.Local,
			Name:     a["name"],
			ColName:  form42f.ColName(a["colName"]),
			FieldID:  form42f.FieldID(id),
			Widget:   a["widget"],
			NoEntry:  a["noEntry"],
			Required: a["required"],
			NotNull:  a["notNull"],
		}
		if fld.Widget == "" {
			fld.Widget = se.Name.Local // 与 form42f.ParseForm4fd 同一手法
		}
		if fld.ColName != "" {
			if seen[fld.ColName] {
				f.Duplicates = append(f.Duplicates, fld.ColName)
			} else {
				seen[fld.ColName] = true
			}
		}
		f.Fields = append(f.Fields, fld)
	}
	if !sawForm {
		// 光有元素没有 <Form> 的输入不认：`.4fd` 一定是 <ManagedForm><Form> 的形状。
		return nil, fmt.Errorf("tzs: %s 的表单描述里没有 <Form> 元素（不是 .4fd）", prog)
	}
	return f, nil
}

// ReadTzsForm4FD 读一个明文的 `.4fd`（工作区里摊出来的那份）。
func ReadTzsForm4FD(path string) (*TzsForm, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	prog := strings.TrimSuffix(filepathBase(path), ".4fd")
	return ParseTzsForm4FD(prog, string(b))
}

// ReadTzsFormPackage 读一个表单包（`.tzs` / `.tzv`，zip）里的 `.4fd`。
//
// 只解压**一个**条目：包里还有 `.tsd`/`.bdx`/`ver`，本校验一条都不用。
// 有多个 `.4fd` 时报错并列出候选 —— 挑一个"看着像的"正是要治的病（不猜）。
func ReadTzsFormPackage(pkgPath string) (*TzsForm, error) {
	zr, err := zip.OpenReader(pkgPath)
	if err != nil {
		return nil, fmt.Errorf("tzs: %s 不是合法的表单包（zip 打不开：%v）", pkgPath, err)
	}
	defer zr.Close()

	var hits []*zip.File
	for _, e := range zr.File {
		if strings.EqualFold(filepathExt(e.Name), ".4fd") {
			hits = append(hits, e)
		}
	}
	if len(hits) == 0 {
		return nil, fmt.Errorf("tzs: %s 里没有 .4fd（这不是表单包，或者包里只有 .tsd/.bdx）", pkgPath)
	}
	if len(hits) > 1 {
		names := make([]string, 0, len(hits))
		for _, h := range hits {
			names = append(names, h.Name)
		}
		sort.Strings(names)
		return nil, fmt.Errorf("tzs: %s 里有 %d 个 .4fd（%s）—— 请指明是哪一个，不按名字猜",
			pkgPath, len(hits), strings.Join(names, " / "))
	}
	rc, err := hits[0].Open()
	if err != nil {
		return nil, fmt.Errorf("tzs: 打开 %s 里的 %s 失败：%v", pkgPath, hits[0].Name, err)
	}
	defer rc.Close()
	b, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("tzs: 读 %s 里的 %s 失败：%v", pkgPath, hits[0].Name, err)
	}
	base := filepathBase(hits[0].Name)
	prog := strings.TrimSuffix(base, filepathExt(base))
	return ParseTzsForm4FD(prog, string(b))
}

// ── 比对 ───────────────────────────────────────────────────────────────────

// CheckForm42F 比一张"应然"（`.42f`）与一张"实然"（tzs 生成物）。
//
// 判定口径（逐条写死，别处不要重复实现）：
//
//	存在性  tzs 有 colName、`.42f` 无 ⇒ **差异**（colName 是两侧共同的身份，缺了就是缺了）
//	编号    见 fieldIDFindings：先看"全部唯一配对是不是恒差一个常数"（A37）——
//	        恒定 ⇒ **1 条聚合说明**（`Note`，不算差异）；散开 ⇒ 逐条报"编号不同"（点明 B5）
//	可写性  见 classifyWritability：**同属性**对齐（A38）；跨属性组合只进"未判定"
//	控件族  两侧控件名都已知且不等价 ⇒ **差异**；任一侧缺失/未知 ⇒ **未判定**
//	重名    两侧任一有重名 colName ⇒ 该 colName 的编号/可写性/控件族**都不判**，留 1 条未判定说明
//
// 三遍的结构（为什么不是一遍）：
//
//	第一遍 按 tzs 文档顺序去重，产出"存在性"差异、"重名"未判定，并把**唯一配对**收进 `pairs`
//	      （顺序由实然侧决定，读起来与包内一致）；
//	第二遍 只看 `pairs` 判"编号"（A37 要求先看整体分布，一遍扫的时候还没有全部差值）；
//	第三遍 再按 `pairs` 顺序判"可写性 / 控件族"。
func CheckForm42F(want *form42f.F42Form, got *TzsForm) *FormCheck {
	c := &FormCheck{}
	if want != nil {
		c.Form = string(want.Name)
		c.DeclaredName42F = want.DeclaredName
		c.F42Fields = len(want.Fields)
	}
	if got != nil {
		c.ProgTzs = got.Prog
		c.TzsFields = len(got.Fields)
		c.TzsUnnamed = got.Unnamed()
	}
	if want == nil || got == nil {
		return c
	}

	byCol42F := map[form42f.ColName][]*form42f.F42Field{}
	for i := range want.Fields {
		fld := &want.Fields[i]
		byCol42F[fld.ColName] = append(byCol42F[fld.ColName], fld)
	}
	byColTzs := map[form42f.ColName][]*TzsField{}
	for i := range got.Fields {
		fld := &got.Fields[i]
		if fld.ColName == "" {
			continue // 没有栏位身份：不可比（已计入 TzsUnnamed）
		}
		byColTzs[fld.ColName] = append(byColTzs[fld.ColName], fld)
	}

	// ── 第一遍：存在性 + 重名 + 收集唯一配对 ──
	var pairs []f42Pair
	done := map[form42f.ColName]bool{}
	for i := range got.Fields {
		tf := &got.Fields[i]
		if tf.ColName == "" || done[tf.ColName] {
			continue
		}
		done[tf.ColName] = true

		f42s := byCol42F[tf.ColName]
		if len(f42s) == 0 {
			c.Findings = append(c.Findings, Finding{
				Rule:    RuleExist,
				ColName: tf.ColName,
				F42:     "（无这条栏目）",
				Tzs:     describeTzsField(tf),
				Why: fmt.Sprintf("tzs 侧声明了 colName=%s，但 .42f（声明名「%s」，共 %d 条字段）里没有这个栏目 —— 编译后的表单里不存在它",
					tf.ColName, want.DeclaredName, len(want.Fields)),
			})
			continue
		}
		c.Pairs++

		// 重名 ⇒ 无法唯一对应，三条规则一起不判（只留一条说明，免得输出被灌满）。
		if n42, nTzs := len(f42s), len(byColTzs[tf.ColName]); n42 > 1 || nTzs > 1 {
			c.Findings = append(c.Findings, Finding{
				Rule:      RuleFieldID,
				ColName:   tf.ColName,
				F42:       describeF42Duplicates(f42s),
				Tzs:       describeTzsDuplicates(byColTzs[tf.ColName]),
				Undecided: true,
				Why: fmt.Sprintf("colName 重名（.42f 出现 %d 次、tzs 出现 %d 次）⇒ 编号/可写性/控件族都无法唯一对应，按 A15 不参与判定",
					n42, nTzs),
			})
			continue
		}
		pairs = append(pairs, f42Pair{col: tf.ColName, f42: f42s[0], tzs: tf})
	}

	// ── 第二遍：编号（A37 基数感知）──
	c.Findings = append(c.Findings, fieldIDFindings(pairs, &c.Notes)...)

	// ── 第三遍：可写性（A38）+ 控件族 ──
	for _, p := range pairs {
		c.Findings = append(c.Findings, classifyWritability(p.f42, p.tzs)...)
		c.Findings = append(c.Findings, compareWidget(p.f42.Widget, p.tzs.Widget, p.col)...)
	}

	// 反方向：只在 `.42f` 里有（信息，不算差异 —— tzs 包可以只维护一张表单的一部分）
	for col := range byCol42F {
		if !done[col] {
			c.F42Only = append(c.F42Only, col)
		}
	}
	sort.Slice(c.F42Only, func(i, j int) bool { return c.F42Only[i] < c.F42Only[j] })
	return c
}

// f42Pair 是一对**唯一**配上的栏位（两侧各一个，无重名）。
type f42Pair struct {
	col form42f.ColName
	f42 *form42f.F42Field
	tzs *TzsField
}

// minRadixPairs 是"谈得上基数差"所需的最少唯一配对数。
//
// 为什么不是 1 对：只有一对时"差值恒定"是**恒真**的（一个数在它自己身上当然恒定），
// 那会把"两侧各一个字段、编号 18 对 19"也解释成"基数不同"，等于把唯一的信号吞掉 ——
// cli 层 `TestCmdTzsFormCheck_DiffExitsSix` 就是这个形状（单字段、编号差 1、必须退 6）。
// 2 对起，"全部配对差值相同"才是**观察到的**事实，而不是退化情形。
const minRadixPairs = 2

// radixShift 判"全部唯一配对是否恒差同一个**非零**常数"（A37）。
//
// 差值口径固定为 **tzs − `.42f`**（实然减应然）；恒定且非 0 才算基数差（恒差 0 = 没有问题）。
func radixShift(diffs []int) (shift int, ok bool) {
	if len(diffs) < minRadixPairs {
		return 0, false
	}
	first := diffs[0]
	if first == 0 {
		return 0, false
	}
	for _, d := range diffs[1:] {
		if d != first {
			return 0, false
		}
	}
	return first, true
}

// fieldIDFindings 判"编号"这一组（A37）。
//
// 两条路：
//
//	恒差一个非零常数 ⇒ **1 条聚合说明**（返回值空、往 notes 里塞），**不**逐条报，
//	                  也**不**据此判"另一张表单"（基数差不是设计变更，也不是版本证据）；
//	其余（差值散开 / 配对数不足以谈分布）⇒ 逐条报"编号不同"（这才是真信号）。
func fieldIDFindings(pairs []f42Pair, notes *[]Note) []Finding {
	diffs := make([]int, 0, len(pairs))
	for _, p := range pairs {
		diffs = append(diffs, int(p.tzs.FieldID)-int(p.f42.FieldID))
	}
	if shift, ok := radixShift(diffs); ok {
		*notes = append(*notes, radixNote(pairs, shift))
		return nil
	}
	var out []Finding
	for _, p := range pairs {
		if p.f42.FieldID == p.tzs.FieldID {
			continue
		}
		out = append(out, Finding{
			Rule:    RuleFieldID,
			ColName: p.col,
			F42:     fmt.Sprintf("fieldId=%d", p.f42.FieldID),
			Tzs:     fmt.Sprintf("fieldId=%d", p.tzs.FieldID),
			Why: fmt.Sprintf("同一 colName 两侧编号不同 ⇒ **可能是另一张表单/版本不同**：应然（.42f）是 %d、实然（tzs）是 %d。"+
				"注意 .4fd 的 fieldId 每次保存会全量重排（契约 B5），所以这条是「可疑」而不是「错误」",
				p.f42.FieldID, p.tzs.FieldID),
		})
	}
	return out
}

// radixNote 造那条"两侧基数不同"的聚合说明。
//
// 措辞纪律（A37）：**只陈述事实**（多少对、恒差多少、两侧编号区间）——**不猜**哪一侧是 0 基；
// 也不据此判"另一张表单"。同时把样例 colName 带上，让"降级"不等于"信息没了"。
func radixNote(pairs []f42Pair, shift int) Note {
	minF, maxF := pairs[0].f42.FieldID, pairs[0].f42.FieldID
	minT, maxT := pairs[0].tzs.FieldID, pairs[0].tzs.FieldID
	cols := make([]form42f.ColName, 0, len(pairs))
	for _, p := range pairs {
		if p.f42.FieldID < minF {
			minF = p.f42.FieldID
		}
		if p.f42.FieldID > maxF {
			maxF = p.f42.FieldID
		}
		if p.tzs.FieldID < minT {
			minT = p.tzs.FieldID
		}
		if p.tzs.FieldID > maxT {
			maxT = p.tzs.FieldID
		}
		cols = append(cols, p.col)
	}
	const sample = 8
	sampleCols := cols
	if len(sampleCols) > sample {
		sampleCols = sampleCols[:sample]
	}
	text := fmt.Sprintf("两侧基数不同：全部 %d 对唯一配对的 fieldId **恒差 %+d**（差值 = tzs − .42f；"+
		".42f 是 %d..%d、tzs 是 %d..%d），疑似两侧下标基数不同（一侧 0 基、另一侧 1 基）——"+
		"按 A37 这是基数问题，**不据此判「另一张表单/版本不同」**，也不猜哪一侧是 0 基。"+
		"逐条报 %d 条只会淹没报告，故只留这一条说明（样例 colName：%s）",
		len(pairs), shift, minF, maxF, minT, maxT, len(pairs), joinColNames(sampleCols, sample))
	return Note{Rule: RuleFieldID, Text: text, Pairs: len(pairs), Cols: sampleCols}
}

// classifyWritability 判一对栏位的"可写性"（三属性：`noEntry` / `required` / `notNull`）。
//
// 方向固定：`.42f` = 应然，tzs = 实然。按 A38 分成三支：
//
//	noEntry   与另外两个属性没有已知蕴含关系 ⇒ **同属性**反向就是确定差异；
//	required / notNull  互为参照，联合判：
//	  · 两侧都声明了才谈得上比；tzs 侧**未声明** ⇒ 未判定（"未声明"≠"声明为 false"）；
//	  · 出现 `notNull=true` 而 `required=false` 这个**跨属性组合**（任一侧）⇒ 未判定：
//	    疑似编译器把 NOT NULL 归一成必填，**非设计变更**；一份快照也排除不了"导出后又改"，
//	    所以措辞一律"疑似"；
//	  · 没有那个组合、且**同一属性**反向 ⇒ 确定差异。
func classifyWritability(f42 *form42f.F42Field, tzf *TzsField) []Finding {
	col := tzf.ColName
	var out []Finding

	// ── noEntry ──
	switch {
	case strings.TrimSpace(tzf.NoEntry) == "":
		out = append(out, undeclaredAttr(col, "noEntry", f42.NoEntry))
	case truthyCheck(tzf.NoEntry) != f42.NoEntry:
		out = append(out, Finding{
			Rule:    RuleWritable,
			ColName: col,
			F42:     fmt.Sprintf("noEntry=%t", f42.NoEntry),
			Tzs:     fmt.Sprintf("noEntry=%t", truthyCheck(tzf.NoEntry)),
			Why: fmt.Sprintf("noEntry：应然（.42f）noEntry=%t：%s；实然（tzs）noEntry=%t：%s —— 同一栏位的可写性两侧相反",
				f42.NoEntry, writableMeaning("noEntry", f42.NoEntry),
				truthyCheck(tzf.NoEntry), writableMeaning("noEntry", truthyCheck(tzf.NoEntry))),
		})
	}

	// ── required / notNull（联合判：它们有已知的蕴含关系）──
	reqOK := strings.TrimSpace(tzf.Required) != ""
	nnOK := strings.TrimSpace(tzf.NotNull) != ""
	reqT := truthyCheck(tzf.Required)
	nnT := truthyCheck(tzf.NotNull)

	if !reqOK {
		out = append(out, undeclaredAttr(col, "required", f42.Required))
	}
	if !nnOK {
		out = append(out, undeclaredAttr(col, "notNull", f42.NotNull))
	}
	// 缺一侧声明就没法排除"NOT NULL 归一成必填"这条蕴含 ⇒ 声明了的那侧即使反向也只记未判定。
	if !reqOK || !nnOK {
		if reqOK && reqT != f42.Required {
			out = append(out, Finding{
				Rule: RuleWritable, ColName: col,
				F42:       fmt.Sprintf("required=%t notNull=%t", f42.Required, f42.NotNull),
				Tzs:       fmt.Sprintf("required=%t notNull=（tzs 侧未声明）", reqT),
				Undecided: true,
				Why: fmt.Sprintf("required 两侧反向，但 tzs 侧没声明 notNull ⇒ 排除不了「编译器把 NOT NULL 归一成必填」"+
					"这条蕴含（A38），按 A15 记未判定（应然侧 .42f 是 notNull=%t required=%t）",
					f42.NotNull, f42.Required),
			})
		}
		if nnOK && nnT != f42.NotNull {
			out = append(out, Finding{
				Rule: RuleWritable, ColName: col,
				F42:       fmt.Sprintf("notNull=%t required=%t", f42.NotNull, f42.Required),
				Tzs:       fmt.Sprintf("notNull=%t required=（tzs 侧未声明）", nnT),
				Undecided: true,
				Why: fmt.Sprintf("notNull 两侧反向，但 tzs 侧没声明 required ⇒ 排除不了「NOT NULL 归一成必填」"+
					"这条蕴含（A38），按 A15 记未判定（应然侧 .42f 是 notNull=%t required=%t）",
					f42.NotNull, f42.Required),
			})
		}
		return out
	}

	reqDiff := reqT != f42.Required
	nnDiff := nnT != f42.NotNull
	if !reqDiff && !nnDiff {
		return out
	}

	// 跨属性组合：任一侧出现 `notNull=true` 而 `required=false`。
	cross := (f42.NotNull && !f42.Required) || (nnT && !reqT)
	if cross {
		out = append(out, Finding{
			Rule:      RuleWritable,
			ColName:   col,
			F42:       fmt.Sprintf("notNull=%t required=%t", f42.NotNull, f42.Required),
			Tzs:       fmt.Sprintf("notNull=%t required=%t", nnT, reqT),
			Undecided: true,
			Why: fmt.Sprintf("可写性拿不准：两侧 notNull/required 的**组合**对不上（.42f notNull=%t required=%t；tzs notNull=%t required=%t）。"+
				"`notNull=true` 而 `required=false` 这种**跨属性组合**不得单独判冲突（A38）：疑似编译器把 NOT NULL 归一成必填，"+
				"非设计变更；但一份快照排除不了「导出后又在设计器里改了必填」（.4fd 的 required 是设计期值），所以只记「疑似」，不计入确定差异",
				f42.NotNull, f42.Required, nnT, reqT),
		})
		return out
	}

	if reqDiff {
		out = append(out, Finding{
			Rule:    RuleWritable,
			ColName: col,
			F42:     fmt.Sprintf("required=%t", f42.Required),
			Tzs:     fmt.Sprintf("required=%t", reqT),
			Why: fmt.Sprintf("required：应然（.42f）required=%t：%s；实然（tzs）required=%t：%s —— 同一栏位的必填两侧相反"+
				"（两侧 notNull 一致为 %t，不涉及 NOT NULL 归一那条蕴含）",
				f42.Required, writableMeaning("required", f42.Required), reqT, writableMeaning("required", reqT), f42.NotNull),
		})
	}
	if nnDiff {
		out = append(out, Finding{
			Rule:    RuleWritable,
			ColName: col,
			F42:     fmt.Sprintf("notNull=%t", f42.NotNull),
			Tzs:     fmt.Sprintf("notNull=%t", nnT),
			Why: fmt.Sprintf("notNull：应然（.42f）notNull=%t：%s；实然（tzs）notNull=%t：%s —— 同一栏位的空值约束两侧相反",
				f42.NotNull, writableMeaning("notNull", f42.NotNull), nnT, writableMeaning("notNull", nnT)),
		})
	}
	return out
}

// undeclaredAttr 是"tzs 侧没声明这个属性"的未判定条目（A15：未声明 ≠ 声明为 false）。
func undeclaredAttr(col form42f.ColName, key string, f42v bool) Finding {
	return Finding{
		Rule:      RuleWritable,
		ColName:   col,
		F42:       fmt.Sprintf("%s=%t", key, f42v),
		Tzs:       fmt.Sprintf("%s=（tzs 侧未声明）", key),
		Undecided: true,
		Why: fmt.Sprintf("tzs 侧没声明 %s —— \"未声明\"不等于\"声明为 false\"，按 A15 不比（应然侧 .42f 是 %t）",
			key, f42v),
	}
}

// compareWidget 只判"确定"的控件族关系（A15）。三种情况：
//
//	两侧同一控件（或确定等价：FFLabel≡Label、FFImage≡Image）  → 无条目
//	两侧都是已知控件但不等价                              → 差异（RuleWidget）
//	任一侧为空 / 不在已知表                                → 未判定
func compareWidget(f42Widget, tzsWidget string, col form42f.ColName) []Finding {
	fw, tw := strings.TrimSpace(f42Widget), strings.TrimSpace(tzsWidget)
	if fw == "" || tw == "" {
		return []Finding{{
			Rule:      RuleWidget,
			ColName:   col,
			F42:       orNone(fw),
			Tzs:       orNone(tw),
			Undecided: true,
			Why: fmt.Sprintf("控件类型缺了一侧（.42f=%q、tzs=%q）—— .42f 的 TableColumn/PhantomColumn 是自闭元素、"+
				"没有子控件元素，所以这条比不了", fw, tw),
		}}
	}
	fCanon, fDesc, fOK := widgetCanon(fw)
	tCanon, tDesc, tOK := widgetCanon(tw)
	if !fOK || !tOK {
		unknown := fw
		if fOK {
			unknown = tw
		}
		return []Finding{{
			Rule:      RuleWidget,
			ColName:   col,
			F42:       fw,
			Tzs:       tw,
			Undecided: true,
			Why: fmt.Sprintf("控件名 %q 不在已知映射表里（只认确定见过的 %d 种）⇒ 这个组合我们不确定，按 A15 不参与判定",
				unknown, len(widgetCanonTable)),
		}}
	}
	if fCanon == tCanon {
		return nil
	}
	return []Finding{{
		Rule:    RuleWidget,
		ColName: col,
		F42:     fmt.Sprintf("%s（%s）", fw, fDesc),
		Tzs:     fmt.Sprintf("%s（%s）", tw, tDesc),
		Why: fmt.Sprintf("控件族不同：应然（.42f）是 %s（%s）、实然（tzs）是 %s（%s） —— 同一栏位两侧的控件类型不是一类，值域与录入方式都会不同",
			fw, fDesc, tw, tDesc),
	}}
}

// widgetCanonTable 是**唯一**的"控件名 → 族"表：键是小写控件名，值是该族的说明。
//
// 收录口径（A15：只报确定的映射）：
//   - 键取自两侧**实测见过**的元素名 —— `.42f` 的子控件元素（Edit / ComboBox / CheckBox /
//     TextEdit / ButtonEdit / DateEdit / FFLabel / …）与 `.4fd` 的 `widget` 属性是同一套词表；
//   - 只有 `FFLabel≡Label≡StaticText`、`FFImage≡Image` 这两族做归并（FF 前缀是同一设计器族的
//     装饰性写法，实测两包都有），其余**不允许**跨名归并 —— `Edit` 与 `TextEdit`（单行/多行）、
//     `Edit` 与 `ButtonEdit`（有无开窗按钮）是**不同**控件，要如实报出来；
//   - 表外的一切（含容器 Grid/Table/Tree、`Phantom`）都不参与判定，进"未判定"。
var widgetCanonTable = map[string]string{
	"edit":         "文本输入",
	"textedit":     "多行文本输入",
	"buttonedit":   "开窗输入",
	"combobox":     "下拉选择",
	"radiogroup":   "单选组",
	"checkbox":     "布尔勾选",
	"dateedit":     "日期",
	"datetimeedit": "日期时间",
	"spinedit":     "数值步进",
	"slider":       "数值滑杆",
	"progressbar":  "进度展示",
	"fflabel":      "静态文本",
	"label":        "静态文本",
	"statictext":   "静态文本",
	"ffimage":      "图片",
	"image":        "图片",
}

// widgetCanon 返回控件名的归一键、"族说明"与"是否已知"。
//
// 归一键用作等价比对的依据（`FFLabel` 与 `Label` 归一到同一个键 ⇒ 不算差异）；族说明给人读。
func widgetCanon(name string) (canon, desc string, ok bool) {
	key := strings.ToLower(strings.TrimSpace(name))
	if key == "" {
		return "", "", false
	}
	desc, ok = widgetCanonTable[key]
	if !ok {
		return "", "", false
	}
	return "canon:" + desc, desc, true
}

// ── 小工具 ─────────────────────────────────────────────────────────────────

// truthyCheck 认 `.4fd`/`.tsd` 的布尔写法（与 `.42f` 侧的 truthy42F 同一套）。
func truthyCheck(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes", "y":
		return true
	}
	return false
}

// writableMeaning 把一个布尔属性值翻成人话（差异清单里要看出"哪侧能写"）。
func writableMeaning(key string, v bool) string {
	switch key {
	case "noEntry":
		if v {
			return "不可输入"
		}
		return "可输入"
	case "required":
		if v {
			return "必填"
		}
		return "非必填"
	case "notNull":
		if v {
			return "栏位非空"
		}
		return "允许为空"
	}
	return fmt.Sprintf("%t", v)
}

// describeTzsField 是实然侧一条字段的可读摘要（存在性差异里用）。
func describeTzsField(f *TzsField) string {
	return fmt.Sprintf("fieldId=%d widget=%s tag=%s", f.FieldID, orNone(f.Widget), f.Tag)
}

// describeF42Duplicates / describeTzsDuplicates 写重名时的两侧编号清单。
func describeF42Duplicates(fs []*form42f.F42Field) string {
	ids := make([]string, 0, len(fs))
	for _, f := range fs {
		ids = append(ids, fmt.Sprintf("%d", f.FieldID))
	}
	return "fieldId=" + strings.Join(ids, "/")
}

func describeTzsDuplicates(fs []*TzsField) string {
	ids := make([]string, 0, len(fs))
	for _, f := range fs {
		ids = append(ids, fmt.Sprintf("%d", f.FieldID))
	}
	return "fieldId=" + strings.Join(ids, "/")
}

// orNone 把空串写成可读的"（空）"（输出里空着一格最容易看漏）。
func orNone(s string) string {
	if strings.TrimSpace(s) == "" {
		return "（空）"
	}
	return s
}

// joinColNames 折叠过长的 colName 清单（超过 n 个就写"…另 k 个"）。
func joinColNames(cols []form42f.ColName, n int) string {
	parts := make([]string, 0, len(cols))
	for i, c := range cols {
		if i >= n {
			parts = append(parts, fmt.Sprintf("…另 %d 个", len(cols)-n))
			break
		}
		parts = append(parts, string(c))
	}
	return strings.Join(parts, " ")
}

// attrMapTzs 把属性摊成 map（同包不引 form42f 的私有实现，3 行不值得跨包导出）。
func attrMapTzs(attrs []xml.Attr) map[string]string {
	m := make(map[string]string, len(attrs))
	for _, a := range attrs {
		m[a.Name.Local] = a.Value
	}
	return m
}

// filepathBase / filepathExt 是 path/filepath 的两个薄封装。
//
// 为什么绕一层：包内 zip 条目名一律用正斜杠（zip 规范），而 `filepath.Base` 在 Windows 上
// 只认反斜杠 —— 直接用它，`a/b.4fd` 会整串返回，程序名就取错了。
func filepathBase(name string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	if i := strings.LastIndex(name, "/"); i >= 0 {
		return name[i+1:]
	}
	return name
}

func filepathExt(name string) string {
	base := filepathBase(name)
	if i := strings.LastIndex(base, "."); i >= 0 {
		return base[i:]
	}
	return ""
}