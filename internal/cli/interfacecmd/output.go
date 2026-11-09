package interfacecmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tt/internal/cli/common"
	"tt/internal/output"
	"tt/internal/screen"
)

const schemaVersion = "tt.interface/v1"

type envelope struct {
	OK     bool   `json:"ok"`
	Schema string `json:"schema"`
	Data   any    `json:"data"`
}

func emitJSON(data any) error {
	return common.PrintJSON(envelope{OK: true, Schema: schemaVersion, Data: data})
}

func sourceMap(src *sourceInputs) map[string]any {
	m := map[string]any{"mode": src.Mode}
	if src.Env != "" {
		m["env"] = src.Env
	}
	if src.Root != "" {
		m["root"] = src.Root
	}
	if src.Form42F != "" {
		m["form42f"] = shortPath(src.Root, src.Form42F)
	}
	if src.Form42S != "" {
		m["form42s"] = shortPath(src.Root, src.Form42S)
	}
	return m
}

func shortPath(root, p string) string {
	if root != "" {
		if rel, err := filepath.Rel(root, p); err == nil && !strings.HasPrefix(rel, "..") {
			return filepath.ToSlash(rel)
		}
	}
	return p
}

func labelSource(src *sourceInputs) string {
	if src != nil && src.Form42S != "" {
		return "42s"
	}
	return "none"
}

func pageBase(p *screen.PageView, brief bool) map[string]any {
	m := map[string]any{
		"id":         p.ID,
		"textKey":    p.TextKey,
		"label":      p.Label,
		"fieldCount": p.FieldCount,
	}
	if p.LabelState != "" && p.LabelState != "resolved" {
		m["labelState"] = p.LabelState
		if p.Reason != "" {
			m["labelReason"] = p.Reason
		}
	}
	if !brief {
		if p.ParentID == "" {
			m["parentId"] = nil
		} else {
			m["parentId"] = p.ParentID
		}
		m["depth"] = p.Depth
		m["path"] = p.Path
		m["hasChildren"] = p.HasChildren
	}
	return m
}

func pageViewMap(v *screen.View, p *screen.PageView, level, depth int, brief, withFields bool) (map[string]any, bool) {
	m := pageBase(p, brief)
	if withFields {
		fs := v.FieldsInPage(p.ID)
		if len(fs) > 0 {
			m["fields"] = fieldMaps(fs, brief)
		}
	}
	cut := false
	if len(p.Children) > 0 {
		if depth < 0 || level < depth {
			children := make([]map[string]any, 0, len(p.Children))
			for _, c := range p.Children {
				cm, ccut := pageViewMap(v, c, level+1, depth, brief, withFields)
				children = append(children, cm)
				cut = cut || ccut
			}
			m["children"] = children
		} else {
			cut = true
		}
	}
	return m, cut
}

func pagesTree(v *screen.View, depth int, brief, withFields bool) ([]map[string]any, bool) {
	out := make([]map[string]any, 0, len(v.Roots))
	cut := false
	for _, p := range v.Roots {
		m, c := pageViewMap(v, p, 1, depth, brief, withFields)
		out = append(out, m)
		cut = cut || c
	}
	return out, cut
}

func showData(v *screen.View, src *sourceInputs, depth int, withFields, brief bool) map[string]any {
	pages, cut := pagesTree(v, depth, brief, withFields)
	unresolved := v.Unresolved
	if !withFields {
		unresolved = make([]screen.Unresolved, 0)
		for _, u := range v.Unresolved {
			if u.Kind == "page" {
				unresolved = append(unresolved, u)
			}
		}
	}
	return map[string]any{
		"kind":             "interface.show",
		"form":             v.Form,
		"module":           v.Module,
		"build":            v.Build,
		"source":           sourceMap(src),
		"labelSource":      labelSource(src),
		"pages":            pages,
		"globalFieldCount": v.GlobalFieldCount,
		"totalFields":      v.TotalFields,
		"labelStats":       v.LabelStats,
		"truncated":        cut,
		"unresolved":       unresolved,
		"warnings":         warningsOf(v),
	}
}

func pagesData(v *screen.View, src *sourceInputs, brief bool) map[string]any {
	pages := make([]map[string]any, 0, len(v.Pages))
	for _, p := range v.Pages {
		pages = append(pages, pageBase(p, brief))
	}
	return map[string]any{
		"kind":             "interface.pages",
		"form":             v.Form,
		"module":           v.Module,
		"source":           sourceMap(src),
		"labelSource":      labelSource(src),
		"total":            len(v.Pages),
		"globalFieldCount": v.GlobalFieldCount,
		"pages":            pages,
		"warnings":         warningsOf(v),
	}
}

func fieldsData(v *screen.View, src *sourceInputs, fields []screen.FieldView, page *screen.PageView, total int, truncated, brief bool) map[string]any {
	data := map[string]any{
		"kind":        "interface.fields",
		"form":        v.Form,
		"module":      v.Module,
		"source":      sourceMap(src),
		"labelSource": labelSource(src),
		"total":       total,
		"returned":    len(fields),
		"truncated":   truncated,
		"fields":      fieldMaps(fields, brief),
		"warnings":    warningsOf(v),
	}
	if page != nil {
		data["page"] = map[string]any{
			"id": page.ID, "textKey": page.TextKey, "label": page.Label,
			"fieldCount": page.FieldCount,
		}
	}
	if f.page == "" {
		data["globalFieldCount"] = v.GlobalFieldCount
	}
	return data
}

func fieldMaps(fields []screen.FieldView, brief bool) []map[string]any {
	out := make([]map[string]any, 0, len(fields))
	for _, f := range fields {
		out = append(out, fieldMap(f, brief))
	}
	return out
}

func fieldMap(f screen.FieldView, brief bool) map[string]any {
	m := map[string]any{
		"fieldId":    f.FieldID,
		"colName":    f.ColName,
		"label":      f.Label,
		"labelState": f.LabelState,
	}
	if f.PageID != "" {
		m["pageId"] = f.PageID
	}
	if f.Widget != "" {
		m["widget"] = f.Widget
	}
	if f.SQLType != "" {
		m["sqlType"] = f.SQLType
	}
	m["required"] = f.Required
	if f.LabelState != "resolved" && f.Reason != "" {
		m["labelReason"] = f.Reason
	}
	if !brief {
		m["pageTextKey"] = f.PageTextKey
		m["pageLabel"] = f.PageLabel
		m["textKey"] = f.TextKey
		m["labelSource"] = f.LabelSource
		m["tag"] = f.Tag
		m["tabIndex"] = f.TabIndex
		m["notNull"] = f.NotNull
	}
	return m
}

func warningsOf(v *screen.View) []string {
	if v == nil || len(v.Warnings) == 0 {
		return []string{}
	}
	return v.Warnings
}

func printShowHuman(v *screen.View, depth int, withFields bool) error {
	fmt.Fprintln(os.Stdout, v.HumanSummary())
	if len(v.Roots) == 0 {
		fmt.Fprintln(os.Stdout, "(没有页面)")
	} else {
		for i, p := range v.Roots {
			if err := printPageHuman(v, p, "", i == len(v.Roots)-1, 1, depth, withFields); err != nil {
				return err
			}
		}
	}
	if v.GlobalFieldCount > 0 {
		fmt.Fprintf(os.Stdout, "└─ 未归页字段  %d\n", v.GlobalFieldCount)
	}
	if depth > 0 && hasDeeper(v.Roots, 1, depth) {
		fmt.Fprintf(os.Stdout, "(页面深度已截到 %d；加 --depth all 看全)\n", depth)
	}
	return nil
}

func printPageHuman(v *screen.View, p *screen.PageView, prefix string, last bool, level, depth int, withFields bool) error {
	branch, next := "├─ ", prefix+"│  "
	if last {
		branch, next = "└─ ", prefix+"   "
	}
	label := p.Label
	if label == "" {
		label = "【未命名】"
	}
	fmt.Fprintf(os.Stdout, "%s%s%s  [%s]  %s  (%d 字段)\n", prefix, branch, p.ID, p.TextKey, label, p.FieldCount)
	if withFields {
		fs := v.FieldsInPage(p.ID)
		for i, f := range fs {
			flast := i == len(fs)-1
			fb := "├─ "
			if flast {
				fb = "└─ "
			}
			fl := f.Label
			if fl == "" {
				if f.TextKey != "" {
					fl = "【未命名 " + f.TextKey + "】"
				} else {
					fl = "【未命名】"
				}
			}
			fmt.Fprintf(os.Stdout, "%s%s%s  (%s)  %s  %s\n", next, fb, fl, f.ColName, f.Widget, f.SQLType)
		}
	}
	if (depth < 0 || level < depth) && len(p.Children) > 0 {
		for i, c := range p.Children {
			if err := printPageHuman(v, c, next, i == len(p.Children)-1, level+1, depth, withFields); err != nil {
				return err
			}
		}
	}
	return nil
}

func hasDeeper(ps []*screen.PageView, level, depth int) bool {
	if depth < 0 {
		return false
	}
	for _, p := range ps {
		if level >= depth && len(p.Children) > 0 {
			return true
		}
		if hasDeeper(p.Children, level+1, depth) {
			return true
		}
	}
	return false
}

func printPagesHuman(v *screen.View, brief bool) error {
	cols := []string{"ID", "textKey", "页面", "字段"}
	rows := make([][]string, 0, len(v.Pages))
	for _, p := range v.Pages {
		rows = append(rows, []string{p.ID, p.TextKey, p.Label, fmt.Sprint(p.FieldCount)})
	}
	if brief {
		cols = []string{"ID", "页面", "字段"}
		rows = rows[:0:0]
		for _, p := range v.Pages {
			rows = append(rows, []string{p.ID, p.Label, fmt.Sprint(p.FieldCount)})
		}
	}
	if v.GlobalFieldCount > 0 {
		if brief {
			rows = append(rows, []string{"_global", "未归页字段", fmt.Sprint(v.GlobalFieldCount)})
		} else {
			rows = append(rows, []string{"_global", "", "未归页字段", fmt.Sprint(v.GlobalFieldCount)})
		}
	}
	return output.WriteTable(os.Stdout, cols, rows)
}

func printFieldsHuman(v *screen.View, fields []screen.FieldView, brief bool) error {
	cols := []string{"页面", "字段", "列名", "控件", "类型", "必填"}
	if !brief {
		cols = append(cols, "标签键", "状态")
	}
	rows := make([][]string, 0, len(fields))
	for _, f := range fields {
		page := f.PageID
		if page == "" {
			page = "_global"
		}
		req := ""
		if f.Required {
			req = "是"
		}
		row := []string{page, f.Label, f.ColName, f.Widget, f.SQLType, req}
		if !brief {
			row = append(row, f.TextKey, f.LabelState)
		}
		rows = append(rows, row)
	}
	return output.WriteTable(os.Stdout, cols, rows)
}

func printLabelsHuman(entries []map[string]any) error {
	if len(entries) == 0 {
		fmt.Println("(42s 缺失或没有标签)")
		return nil
	}
	for _, e := range entries {
		fmt.Fprintf(os.Stdout, "%-32s %s\n", e["token"], e["text"])
	}
	return nil
}
