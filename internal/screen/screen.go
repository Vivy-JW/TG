// Package screen 把 .42f 的页面/字段结构与画面文字合成“前端画面模型”。
//
// 边界：
//   - 结构、字段、控件来自 .42f；标签来自调用方给的 Labels（TG 用 gae_file 的译文）；
//   - 找不到标签就显式标 unresolved，不拿 lbl_* 冒充中文；
//   - 页面 id 用 42f 的 XML name（必要时退回 path），textKey 只用于查标签。
//
// 命令层负责取文件、查译文、选择页面和输出；本包不碰 IO。
package screen

import (
	"fmt"
	"strings"

	"tt/internal/form42f"
)

// Labels 是 token → 译文 的只读查询。
//
// 语义与之前的字符串表一致：ok=false 表示没有这个 token；ambiguous=true 表示
// 同一 token 对应多个不同文本（此时不猜，交给调用方标 ambiguous）。
type Labels interface {
	Unique(token string) (text string, ok, ambiguous bool)
}

// MapLabels 是内存里的 token → 译文，给一处译文只有一条的场景用。
type MapLabels map[string]string

// Unique 实现 Labels；重复 key 在 map 里天然合并，不存在歧义。
func (m MapLabels) Unique(token string) (string, bool, bool) {
	text, ok := m[token]
	return text, ok, false
}

// PageView 是一页及其页面级标签。
type PageView struct {
	ID          string
	Name        string
	TextKey     string
	Label       string
	LabelState  string // resolved | unresolved | ambiguous
	Reason      string
	CommentKey  string
	ParentID    string
	Path        string
	Depth       int
	FieldCount  int
	HasChildren bool
	Children    []*PageView
	Fields      []FieldView
}

// FieldView 是一个字段及其最终标签。
type FieldView struct {
	Order       int
	PageID      string
	PageTextKey string
	PageLabel   string
	FieldID     int
	ColName     string
	TextKey     string
	LabelSource string // field | label-token | label-geometry | none
	Label       string
	LabelState  string
	Reason      string
	Widget      string
	SQLType     string
	Required    bool
	NotNull     bool
	Tag         string
	TabIndex    int
	Geom        form42f.F42Geom
}

// Unresolved 是机器可读的“为什么这里没有中文”。
type Unresolved struct {
	Kind    string `json:"kind"` // page | field
	PageID  string `json:"pageId,omitempty"`
	FieldID int    `json:"fieldId,omitempty"`
	ColName string `json:"colName,omitempty"`
	Token   string `json:"token,omitempty"`
	Reason  string `json:"reason"`
}

// LabelStats 是页面和字段标签的解析统计。
type LabelStats struct {
	PagesResolved    int `json:"pagesResolved"`
	PagesUnresolved  int `json:"pagesUnresolved"`
	FieldsResolved   int `json:"fieldsResolved"`
	FieldsUnresolved int `json:"fieldsUnresolved"`
}

// View 是一次合成的完整结果。
type View struct {
	Form             string
	Module           string
	Build            string
	Roots            []*PageView
	Pages            []*PageView // 扁平，XML 顺序
	Fields           []FieldView // 扁平，XML 顺序
	GlobalFieldCount int
	TotalFields      int
	Unresolved       []Unresolved
	LabelStats       LabelStats
	Warnings         []string
}

// Build 合成页面树与字段表。labels 为 nil 表示没有画面文字：结构仍然可用，
// 所有依赖标签的位置都标成 unresolved。
func Build(form, module string, layout *form42f.F42Layout, labels Labels) *View {
	v := &View{Form: form, Module: module}
	if layout == nil || layout.Form == nil {
		v.Warnings = append(v.Warnings, "42f 布局为空，未生成画面")
		return v
	}
	v.Build = layout.Form.Build
	if labels == nil {
		v.Warnings = append(v.Warnings, "没有画面文字（gae_file），页面/字段标签未解析")
	}

	byID := make(map[string]*PageView, len(layout.Pages))
	for _, p := range layout.Pages {
		pv := &PageView{
			ID:         p.ID,
			Name:       p.Name,
			TextKey:    p.TextKey,
			CommentKey: p.CommentKey,
			ParentID:   p.ParentID,
			Path:       p.Path,
			Depth:      p.Depth,
			FieldCount: len(p.FieldIndexes),
		}
		key := p.TextKey
		if key == "" {
			key = p.CommentKey
		}
		pv.Label, pv.LabelState, pv.Reason = resolveChain(labels, form, p.Name, p.TextKey, p.CommentKey)
		if pv.LabelState == "resolved" {
			v.LabelStats.PagesResolved++
		} else {
			v.LabelStats.PagesUnresolved++
			v.addUnresolved(Unresolved{Kind: "page", PageID: p.ID, Token: key, Reason: pv.Reason})
		}
		byID[p.ID] = pv
		v.Pages = append(v.Pages, pv)
	}
	for _, p := range layout.Pages {
		pv := byID[p.ID]
		pv.HasChildren = len(p.Children) > 0
		if p.ParentID == "" {
			v.Roots = append(v.Roots, pv)
		} else if parent := byID[p.ParentID]; parent != nil {
			parent.Children = append(parent.Children, pv)
		} else {
			// 父页面缺失不应让整页丢失；当作顶层并留一条警告。
			v.Roots = append(v.Roots, pv)
		}
	}

	pageLabel := make(map[string]string, len(v.Pages))
	pageTextKey := make(map[string]string, len(v.Pages))
	for _, p := range v.Pages {
		pageLabel[p.ID] = p.Label
		pageTextKey[p.ID] = p.TextKey
	}

	for i, lf := range layout.Fields {
		if lf.Index < 0 || lf.Index >= len(layout.Form.Fields) {
			continue
		}
		f := layout.Form.Fields[lf.Index]
		fv := FieldView{
			Order:       i + 1,
			PageID:      lf.PageID,
			FieldID:     int(f.FieldID),
			ColName:     string(f.ColName),
			TextKey:     lf.LabelKey,
			LabelSource: lf.LabelSource,
			Widget:      f.Widget,
			SQLType:     f.SQLType,
			Required:    f.Required,
			NotNull:     f.NotNull,
			Tag:         f.Tag,
			TabIndex:    f.TabIndex,
			Geom:        f.Geom,
		}
		if lf.PageID != "" {
			fv.PageTextKey = pageTextKey[lf.PageID]
			fv.PageLabel = pageLabel[lf.PageID]
		} else {
			v.GlobalFieldCount++
		}

		// GP 的 gae_file 按“字段名/元件名”登记（zta01、page01），T100 的 .42s
		// 才按 <Label text> 登记。所以先按列名查，再退回标签自身的 textKey。
		keys := []string{string(f.ColName)}
		if s := strings.TrimSpace(lf.LabelKey); s != "" {
			keys = append(keys, s)
		}
		if strings.TrimSpace(lf.LabelKey) == "" && strings.TrimSpace(string(f.ColName)) == "" {
			fv.LabelState = "unresolved"
			fv.Reason = "label-not-associated"
		} else {
			fv.Label, fv.LabelState, fv.Reason = resolveChain(labels, form, keys...)
			// 没关联到标签、按列名也查不到：报“没关联”，比“token 不存在”更能指出
			// 是 42f 里这个字段压根没配 Label（几何关联失败）。
			if fv.LabelState == "unresolved" && strings.TrimSpace(lf.LabelKey) == "" && fv.Reason == "token-not-found" {
				fv.Reason = "label-not-associated"
			}
		}
		if fv.LabelState == "resolved" {
			v.LabelStats.FieldsResolved++
		} else {
			v.LabelStats.FieldsUnresolved++
			v.addUnresolved(Unresolved{
				Kind: "field", PageID: fv.PageID, FieldID: fv.FieldID,
				ColName: fv.ColName, Token: fv.TextKey, Reason: fv.Reason,
			})
		}
		v.Fields = append(v.Fields, fv)
	}
	v.TotalFields = len(v.Fields)
	v.GlobalFieldCount = 0
	for _, f := range v.Fields {
		if f.PageID == "" {
			v.GlobalFieldCount++
		}
	}
	return v
}

func (v *View) addUnresolved(u Unresolved) {
	v.Unresolved = append(v.Unresolved, u)
}

// FieldsInPage 返回某页直接挂的字段（不含子页）。
func (v *View) FieldsInPage(pageID string) []FieldView {
	out := make([]FieldView, 0)
	for _, f := range v.Fields {
		if f.PageID == pageID {
			out = append(out, f)
		}
	}
	return out
}

// FindPage 按页面 id 找；不存在返回 nil。
func (v *View) FindPage(id string) *PageView {
	for _, p := range v.Pages {
		if p.ID == id {
			return p
		}
	}
	return nil
}

// FindPageByToken 按 textKey 找；唯一命中才返回，避免 page_1 双关。
func (v *View) FindPageByToken(token string) (*PageView, bool) {
	var hit *PageView
	for _, p := range v.Pages {
		if p.TextKey == token {
			if hit != nil {
				return nil, false
			}
			hit = p
		}
	}
	return hit, hit != nil
}

// resolveChain 按顺序试多个候选 token（字段名/元件名优先，其次标签文案）。
// 任一命中即用；都不命中时报“都试过”，不猜。
func resolveChain(t Labels, form string, keys ...string) (label, state, reason string) {
	tried := false
	for _, k := range keys {
		if strings.TrimSpace(k) == "" {
			continue
		}
		tried = true
		label, state, reason = resolveToken(t, form, k)
		if state != "unresolved" || reason != "token-not-found" {
			return label, state, reason
		}
	}
	if !tried {
		return "", "unresolved", "no-token"
	}
	return "", "unresolved", "token-not-found"
}

// resolveToken 先精确查，再只做“去掉表单前后缀”的规范化；不做模糊猜测。
func resolveToken(t Labels, form, token string) (label, state, reason string) {
	token = strings.TrimSpace(token)
	if token == "" {
		return "", "unresolved", "no-token"
	}
	if t == nil {
		return "", "unresolved", "labels-missing"
	}
	if text, ok, amb := t.Unique(token); ok {
		if amb {
			return "", "ambiguous", "token-ambiguous"
		}
		return text, "resolved", ""
	}
	norm := normalizeToken(form, token)
	if norm != token {
		if text, ok, amb := t.Unique(norm); ok {
			if amb {
				return "", "ambiguous", "token-ambiguous"
			}
			return text, "resolved", ""
		}
	}
	return "", "unresolved", "token-not-found"
}

func normalizeToken(form, token string) string {
	form = strings.TrimSpace(form)
	token = strings.TrimSpace(token)
	if form == "" {
		return token
	}
	if strings.HasPrefix(token, form+".") {
		token = strings.TrimPrefix(token, form+".")
	}
	if strings.HasSuffix(token, "."+form) {
		token = strings.TrimSuffix(token, "."+form)
	}
	return token
}

// HumanSummary 是 --human 的表头材料；不参与 JSON 契约。
func (v *View) HumanSummary() string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%s  [%s]  build %s  (%d 页 / %d 字段)", v.Form, v.Module, v.Build, len(v.Pages), v.TotalFields)
}
