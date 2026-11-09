package form42f

// 本文件是主计划 E 的 `.42f`（编译后的表单文件）解析。
//
// 契约（唯一口径）：`.agents/plan/tt-genero-formdef-contract.md` 的 C1 / C2 / C7。
// 上游事实（不再重证）：`.42f` 是**明文 XML**（`GDC-逆向报告.md` §17）；真 T100 表单同样（§18）；
// 离线解析与运行期语料逐条比对 **509/509 = 100%**（§18、`explore-round6 §3`）。
//
// 三条硬约束（写死在代码里）：
//  1. **字段载体不止 `<FormField>`** —— 明细网格的列是 `<TableColumn>`，还有
//     `<PhantomColumn>` / `<PhantomFormField>`。只收 FormField 会得到"看起来对、其实错"的结果
//     （实测 `axmt500_wf.42f`：563 条 = TableColumn 397 + FormField 132 + PhantomFormField 22 + PhantomColumn 12）。
//     这条由 `TestParseForm42F_CollectsTableColumns`（反证测试）守着。
//  2. **绝不用 `.4fd` 的 `fieldId`** —— 设计期编号每次保存全量重排（`engine/SPEC.md:257`）。
//     本文件解析的是**编译后**的 `.42f`，它的 `fieldId` 才稳定（表单变体内；带作用域后 100%）。
//  3. **本解析器不做业务语义** —— 不查字典、不猜中文名；`cmt_*` / `lbl_*` 只是原样带出来。

import (
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// ── 牙 a：类型即契约 ─────────────────────────────────────────────────────────
//
// 四类都是**独立类型**，混用编译不过 —— 这是防"三个编号混用"（设计期 fieldId /
// 表单 fieldId / 会话 id）最硬的手段，比注释可靠。
// 注：`SessionID` 本轮**刻意不引入**（契约 §4 的 A4）：它要穿过既有 Model/do.go 的
// 大量 int，属 M2 寻址层改造；现在引入会把 WP2 变成大重构。

// ColName 是字段身份（如 "b_xmda001"）。它**不是** fieldId，也不是会话 id。
type ColName string

// FieldID 是字段在某张表单（某个变体）内的编号。
type FieldID int

// FormName 是表单名，不含扩展名（如 "axmt500_wf"）。
type FormName string

// Variant 是表单变体（如 "gui" / "tui"）。编号按"表单 + 变体"各自独立。
type Variant string

// ── 结构 ───────────────────────────────────────────────────────────────────

// F42Geom 是控件在网格里的几何（原样带出，不做换算）。
type F42Geom struct {
	Width     int
	Height    int
	PosX      int
	PosY      int
	GridWidth int
}

// F42Field 是 `.42f` 里的一条字段条目（可能是 FormField，也可能是明细列）。
type F42Field struct {
	Tag      string  // FormField | TableColumn | PhantomColumn | PhantomFormField
	Name     string  // 协议名（如 formonly.b_xmda001）
	ColName  ColName // 字段身份
	FieldID  FieldID // 表单内编号
	TabIndex int     // 0 = 未声明
	SQLType  string
	SQLTab   string
	NotNull  bool
	Required bool
	NoEntry  bool
	Comment  string  // cmt_*（条目或子控件上，取先有的）
	TextKey  string  // lbl_*（条目或子控件上，取先有的）
	Widget   string  // Edit | ComboBox | CheckBox | ...（该条目下第一个非字段元素）
	Geom     F42Geom // 来自那个子控件
}

// F42Link 是 RecordView 里的一条链接：colName → fieldIdRef。
type F42Link struct {
	ColName    ColName
	FieldIDRef FieldID
}

// F42RecordView 是一个 RecordView（表名 + 它的 links）。
type F42RecordView struct {
	TabName string
	Links   []F42Link
}

// F42Form 是一张表单的完整解析结果。
type F42Form struct {
	// Name 是**调用方给的表单名**（scope 用；来自文件名 / 4GL 里的 `WITH FORM`）。
	Name FormName
	// DeclaredName 是文件里 `<Form name="…">` 写的东西 —— 实测 T100 的表单会写成
	// `axmt500_wf.tmp`（编译产物的中间名），所以**不能拿它当 scope**，只作对照。
	DeclaredName string

	Build   string
	Style   string
	Spacing string
	Width   int
	Height  int

	Fields []F42Field

	// ByColName：colName → fieldId。**只有表单内唯一时才有意义**（重名见 Duplicates）。
	ByColName map[ColName]FieldID
	// ByKey：双键 "colName#fieldId" → 在 Fields 里的下标。定位走这个，避免重名歧义。
	ByKey map[string]int

	// Duplicates：同表单内**重名**的 colName（数据异常，定位时必须拒）。
	Duplicates []string

	RecordViews []F42RecordView
	LStrKeys    []string

	// Skipped：缺 `colName` 或缺 `fieldId`（或 fieldId 非数字）而被跳过的条目数。
	// 不 panic、不猜（契约 C1.6）。
	Skipped int
}

// fieldTags42F 是"字段载体"的完整集合（契约 C1.3）。
var fieldTags42F = map[string]bool{
	"FormField":        true,
	"TableColumn":      true,
	"PhantomColumn":    true,
	"PhantomFormField": true,
}

// ParseForm42F 解析编译后的表单文件（`.42f`）文本。
//
// 签名与同包既有写法对齐：`ParseForm4fd(form, text string) []SrcField`（source.go:54）。
// 与它不同的是这里**必须返回 error** —— 解析失败要能归到 `FORM_UNKNOWN`（契约 C1.7 / C4.2）。
//
// `form` 是调用方知道的表单名（用于 scope）；文件里 `<Form name>` 的值另存 `DeclaredName`。
func ParseForm42F(form, text string) (*F42Form, error) {
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("form42f: %s 的表单内容是空的（空文件或空内容）", form)
	}

	f := &F42Form{
		Name:      FormName(form),
		ByColName: map[ColName]FieldID{},
		ByKey:     map[string]int{},
	}

	dec := xml.NewDecoder(strings.NewReader(text))
	// Genero 写出来的表单声明的是单字节编码（自带样本是 `encoding='ASCII'`，T100 的表单是
	// `encoding='UTF-8'`）。Go 的解码器遇到非 UTF-8 声明会要求 CharsetReader；这里按"字节原样"
	// 处理 —— `.42f` 里的名字/键都是 ASCII 安全字符，原样读不会错（非 ASCII 也只是字节级保留）。
	dec.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		switch strings.ToLower(strings.TrimSpace(charset)) {
		case "ascii", "us-ascii", "utf-8", "utf8", "iso-8859-1", "latin1", "windows-1252":
			return input, nil
		}
		return nil, fmt.Errorf("form42f: %s 的表单声明了不认识的编码 %q", form, charset)
	}

	var cur *F42Field          // 当前正在收集的字段条目
	var curView *F42RecordView // 当前 RecordView
	sawForm := false           // 没见到 <Form> 就不是 `.42f`

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			// 语法坏 → 明确说"不是 XML"，把底层错误带上（便于分辨截断/编码问题）
			return nil, fmt.Errorf("form42f: %s 不是 XML（%v）", form, err)
		}

		switch t := tok.(type) {
		case xml.StartElement:
			name := t.Name.Local
			a := attrMap42F(t.Attr)

			switch {
			case name == "Form":
				sawForm = true
				if v := a["name"]; v != "" {
					f.DeclaredName = v
				}
				f.Build, f.Style, f.Spacing = a["build"], a["style"], a["spacing"]
				f.Width, f.Height = atoi42F(a["width"]), atoi42F(a["height"])

			case name == "RecordView":
				f.RecordViews = append(f.RecordViews, F42RecordView{TabName: a["tabName"]})
				curView = &f.RecordViews[len(f.RecordViews)-1]

			case name == "Link" && curView != nil:
				curView.Links = append(curView.Links, F42Link{
					ColName:    ColName(a["colName"]),
					FieldIDRef: FieldID(atoi42F(a["fieldIdRef"])),
				})

			case name == "LStr":
				if v := a["text"]; v != "" {
					f.LStrKeys = append(f.LStrKeys, v)
				} else if v := a["comment"]; v != "" {
					f.LStrKeys = append(f.LStrKeys, v)
				}

			case fieldTags42F[name]:
				col, idText := a["colName"], a["fieldId"]
				id, idErr := strconv.Atoi(strings.TrimSpace(idText))
				if col == "" || idErr != nil {
					// 缺属性（或 fieldId 非数字）：跳过并计数（绝不猜）
					f.Skipped++
					cur = nil
					continue
				}
				f.Fields = append(f.Fields, F42Field{
					Tag:      name,
					Name:     a["name"],
					ColName:  ColName(col),
					FieldID:  FieldID(id),
					TabIndex: atoi42F(a["tabIndex"]),
					SQLType:  a["sqlType"],
					SQLTab:   a["sqlTabName"],
					NotNull:  truthy42F(a["notNull"]),
					Required: truthy42F(a["required"]),
					NoEntry:  truthy42F(a["noEntry"]),
					Comment:  a["comment"],
					TextKey:  a["text"],
				})
				cur = &f.Fields[len(f.Fields)-1]

			default:
				// 子控件（Edit / ComboBox / CheckBox / …）：只取每个条目的**第一个**
				if cur != nil && cur.Widget == "" {
					cur.Widget = name
					cur.Geom = F42Geom{
						Width:     atoi42F(a["width"]),
						Height:    atoi42F(a["height"]),
						PosX:      atoi42F(a["posX"]),
						PosY:      atoi42F(a["posY"]),
						GridWidth: atoi42F(a["gridWidth"]),
					}
					if cur.Comment == "" {
						cur.Comment = a["comment"]
					}
					if cur.TextKey == "" {
						cur.TextKey = a["text"]
					}
				}
			}

		case xml.EndElement:
			switch t.Name.Local {
			case "FormField", "TableColumn", "PhantomColumn", "PhantomFormField":
				cur = nil
			case "RecordView":
				curView = nil
			}
		}
	}

	// 没有 <Form> 元素 ⇒ 这不是 `.42f`。
	//
	// 为什么需要这条：纯文本（没有 '<'）在 Go 的解码器眼里是**合法的字符数据**，
	// 它会一路 CharData 到 EOF 而不报错 —— 不补这一条，垃圾输入会"解析成功"成一张空表。
	if !sawForm {
		return nil, fmt.Errorf("form42f: %s 不是 XML（没有 <Form> 元素，无法当成 .42f）", form)
	}

	// 建索引（契约 C1.5）
	for i := range f.Fields {
		fld := &f.Fields[i]
		key := fmt.Sprintf("%s#%d", fld.ColName, fld.FieldID)
		f.ByKey[key] = i
		if _, dup := f.ByColName[fld.ColName]; dup {
			f.Duplicates = append(f.Duplicates, string(fld.ColName))
			continue
		}
		f.ByColName[fld.ColName] = fld.FieldID
	}

	return f, nil
}

// ── 小工具 ──────────────────────────────────────────────────────────────────

func attrMap42F(attrs []xml.Attr) map[string]string {
	m := make(map[string]string, len(attrs))
	for _, a := range attrs {
		m[a.Name.Local] = a.Value
	}
	return m
}

func atoi42F(s string) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0
	}
	return n
}

func truthy42F(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes", "y":
		return true
	}
	return false
}
