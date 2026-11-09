package form42f

// 本文件在字段表之上补一层“页面/控件布局”，供 tt interface 合成前端画面。
//
// ParseForm42F 仍只负责字段表；ParseLayout 额外回答两个问题：
//   - 字段属于哪个 <Page>；
//   - 字段的中文标签引用是哪个 lbl_*（字段自己的 text，或同页同容器的独立 <Label>）。
//
// 为什么不改 ParseForm42F 的输出：tt interface / tzs 都依赖它，
// 页面归属只是 interface 的消费需求，不能顺手改变已有命令的 JSON。

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// F42Page 是 .42f 里的一个 <Page>。
type F42Page struct {
	ID           string // 页面唯一标识：XML name；name 重复时退回 path
	Name         string
	TextKey      string // 页面标题的 lbl_* / page_* 键
	CommentKey   string // 页面说明的 cmt_* 键
	ParentID     string
	Path         string
	Depth        int
	Children     []*F42Page
	FieldIndexes []int // 指向 Form.Fields 的下标（只含直接挂在本页的字段）
}

// F42Label 是一个独立的 <Label> 控件（不是字段条目本身）。
//
// 页面 1 这类手工排版页把中文放在字段旁边的 Label 里；TableColumn 则通常把
// lbl_* 写在列自己的 text 属性上。两种都要能喂给 interface。
type F42Label struct {
	Name       string
	TextKey    string
	CommentKey string
	PageID     string
	Container  string // 父容器路径，避免把相邻 Grid 的标签配错
	Geom       F42Geom
}

// F42LayoutField 是“字段 + 页面 + 标签来源”的组合。
type F42LayoutField struct {
	Index       int    // Form.Fields 下标
	PageID      string // 空 = 页面外字段（工具栏/RecordView 等）
	Container   string
	LabelKey    string // 最终用于查 .42s 的键
	LabelSource string // field | label-token | label-geometry | none
}

// F42Layout 是 .42f 的页面树与字段归属。
type F42Layout struct {
	Form   *F42Form
	Pages  []*F42Page // 扁平，便于按 ID 找
	Roots  []*F42Page // 顶层页面，保留 XML 顺序
	Fields []F42LayoutField
	Labels []F42Label
}

// ParseLayout 解析页面树、字段归属和独立 Label 控件。
//
// 它先调用 ParseForm42F 得到字段表，再走一遍 XML：字段用 colName#fieldId
// 对回字段表，绝不重新猜字段身份。XML 只用于拿页面、容器与 Label 的位置。
func ParseLayout(form, text string) (*F42Layout, error) {
	f, err := ParseForm42F(form, text)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("form42f: %s 的表单内容是空的", form)
	}

	dec := newFormDecoder42F(form, strings.NewReader(text))
	out := &F42Layout{Form: f}
	usedPageIDs := map[string]bool{}

	type frame struct {
		name      string
		container bool
		page      *F42Page
		field     bool
	}
	var stack []frame

	currentPage := func() *F42Page {
		for i := len(stack) - 1; i >= 0; i-- {
			if stack[i].page != nil {
				return stack[i].page
			}
		}
		return nil
	}
	containerPath := func() string {
		parts := make([]string, 0, 8)
		for _, fr := range stack {
			if !fr.container {
				continue
			}
			label := fr.name
			if fr.page != nil && fr.page.Name != "" {
				label = "Page:" + fr.page.Name
			}
			parts = append(parts, label)
		}
		return strings.Join(parts, "/")
	}
	hasOpenField := func() bool {
		for i := len(stack) - 1; i >= 0; i-- {
			if stack[i].field {
				return true
			}
			if stack[i].page != nil {
				break
			}
		}
		return false
	}

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("form42f: %s 不是 XML（%v）", form, err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			name := t.Name.Local
			a := attrMap42F(t.Attr)
			parentPage := currentPage()
			parentContainer := containerPath()

			fr := frame{name: name, container: isLayoutContainer42F(name)}

			switch {
			case name == "Page":
				pageName := strings.TrimSpace(a["name"])
				path := pageName
				if parentPage != nil {
					path = strings.TrimSuffix(parentPage.Path, "/") + "/" + pageName
				}
				if path == "" {
					path = fmt.Sprintf("page-%d", len(out.Pages)+1)
				}
				id := pageName
				if id == "" || usedPageIDs[id] {
					id = path
				}
				usedPageIDs[id] = true
				p := &F42Page{
					ID:         id,
					Name:       pageName,
					TextKey:    strings.TrimSpace(a["text"]),
					CommentKey: strings.TrimSpace(a["comment"]),
					Path:       path,
					Depth:      1,
				}
				if parentPage != nil {
					p.ParentID = parentPage.ID
					p.Depth = parentPage.Depth + 1
					parentPage.Children = append(parentPage.Children, p)
				} else {
					out.Roots = append(out.Roots, p)
				}
				out.Pages = append(out.Pages, p)
				fr.page = p
				fr.container = true

			case fieldTags42F[name]:
				col := strings.TrimSpace(a["colName"])
				idText := strings.TrimSpace(a["fieldId"])
				fr.field = true
				if col != "" && idText != "" {
					var fid int
					if _, err := fmt.Sscanf(idText, "%d", &fid); err == nil {
						if idx, ok := f.ByKey[fmt.Sprintf("%s#%d", col, fid)]; ok {
							pageID := ""
							if parentPage != nil {
								pageID = parentPage.ID
								parentPage.FieldIndexes = append(parentPage.FieldIndexes, idx)
							}
							out.Fields = append(out.Fields, F42LayoutField{
								Index:     idx,
								PageID:    pageID,
								Container: parentContainer,
							})
						}
					}
				}

			case name == "Label" && !hasOpenField():
				// 字段自己的 text/comment 由 ParseForm42F 收集；这里只收独立 Label。
				textKey := strings.TrimSpace(a["text"])
				commentKey := strings.TrimSpace(a["comment"])
				if textKey != "" || commentKey != "" {
					pageID := ""
					if parentPage != nil {
						pageID = parentPage.ID
					}
					out.Labels = append(out.Labels, F42Label{
						Name:       strings.TrimSpace(a["name"]),
						TextKey:    textKey,
						CommentKey: commentKey,
						PageID:     pageID,
						Container:  parentContainer,
						Geom: F42Geom{
							Width:     atoi42F(a["width"]),
							Height:    atoi42F(a["height"]),
							PosX:      atoi42F(a["posX"]),
							PosY:      atoi42F(a["posY"]),
							GridWidth: atoi42F(a["gridWidth"]),
						},
					})
				}
			}
			stack = append(stack, fr)

		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		}
	}

	out.associateLabels()
	return out, nil
}

func (l *F42Layout) associateLabels() {
	// 先把字段的默认状态定下来，后面的几何配对只填“仍然空着”的字段。
	for i := range l.Fields {
		l.Fields[i].LabelSource = "none"
	}
	usedLabel := make([]bool, len(l.Labels))

	for i := range l.Fields {
		lf := &l.Fields[i]
		fld := &l.Form.Fields[lf.Index]

		// 1) 字段条目自己带 text（TableColumn 的常见形态）。
		if strings.TrimSpace(fld.TextKey) != "" {
			lf.LabelKey = strings.TrimSpace(fld.TextKey)
			lf.LabelSource = "field"
			continue
		}

		// 2) 独立 Label 的键名符合 lbl_<colName>：比几何更可靠。
		want := "lbl_" + strings.ToLower(string(fld.ColName))
		var tokenHits []int
		for li, lab := range l.Labels {
			if lab.PageID != lf.PageID || lab.Container != lf.Container {
				continue
			}
			if strings.ToLower(lab.TextKey) == want || strings.ToLower(lab.Name) == want {
				tokenHits = append(tokenHits, li)
			}
		}
		if len(tokenHits) == 1 && l.Labels[tokenHits[0]].TextKey != "" {
			lf.LabelKey = l.Labels[tokenHits[0]].TextKey
			lf.LabelSource = "label-token"
			usedLabel[tokenHits[0]] = true
		}
	}

	// 3) 几何配对：每个 Label 只配给“它右侧最近的那个字段”。
	//
	// 不能反向做（每个字段找左侧最近 Label）：一行的主字段已经有 lbl_x 时，
	// 同一个 Label 会被后面的 _desc 字段再次抢走，得到“发料储位”这类错标。
	// 最近的字段已经有标签，就说明这个 Label 不是给更远字段的。
	for li, lab := range l.Labels {
		if usedLabel[li] || strings.TrimSpace(lab.TextKey) == "" {
			continue
		}
		best := -1
		bestX := int(^uint(0) >> 1)
		tie := false
		for fi := range l.Fields {
			lf := &l.Fields[fi]
			if lf.PageID != lab.PageID || lf.Container != lab.Container {
				continue
			}
			fld := &l.Form.Fields[lf.Index]
			if fld.Geom.PosY != lab.Geom.PosY || fld.Geom.PosX <= lab.Geom.PosX {
				continue
			}
			switch {
			case fld.Geom.PosX < bestX:
				bestX = fld.Geom.PosX
				best = fi
				tie = false
			case fld.Geom.PosX == bestX:
				tie = true
			}
		}
		if best < 0 || tie || l.Fields[best].LabelSource != "none" {
			continue
		}
		l.Fields[best].LabelKey = lab.TextKey
		l.Fields[best].LabelSource = "label-geometry"
		usedLabel[li] = true
	}
}

func isLayoutContainer42F(name string) bool {
	switch name {
	case "Form", "Page", "HBox", "VBox", "Grid", "Table", "Folder", "RecordView":
		return true
	}
	return false
}

// newFormDecoder42F 与 ParseForm42F 的编码处理保持一致。
// 单字节编码原样读；非 ASCII 名字只是字节级保留，不会在这里转码。
func newFormDecoder42F(form string, r io.Reader) *xml.Decoder {
	dec := xml.NewDecoder(r)
	dec.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		switch strings.ToLower(strings.TrimSpace(charset)) {
		case "ascii", "us-ascii", "utf-8", "utf8", "iso-8859-1", "latin1", "windows-1252":
			return input, nil
		}
		return nil, fmt.Errorf("form42f: %s 的表单声明了不认识的编码 %q", form, charset)
	}
	return dec
}
