package tzs

// tzs_detail.go —— 把一帧错误的 `detail` 渲染成人读的几行。
//
// 为什么值得单独一份：引擎侧为了"让调用方能自己纠正"专门做了 detail ——
// `legal`（合法值集）、`hint`（近似值）、`candidates`（候选，含可直接重试的 key）、
// `E_ATTR_PARTIAL` 的 `applied`/`failed` 两栏 —— 而默认输出只打 `code (kind): message`，
// 那一半**只有加 --json 才看得见**，等于没做。
//
// **不在 Go 侧定义 detail 的 schema**（`internal/dev/tzs/client.go:155-159` 明确拒绝
// 在 Go 侧抄第二份），所以这里只按 **JSON 的类型**分派（数组 / 字符串 / 其它），
// 键名只用来挑一个中文标签；表里没有的键**按原名兜底打出来**。
// 后果是引擎加一个新键不需要改这里，这里也不可能"漂移成错的" —— 最坏是标签不够好看。

import (
	"encoding/json"
	"io"
	"sort"
	"strings"

	"tt/internal/cli/dev/common"
	"tt/internal/dev/tzs"
)

// detailOrder 是"键 → 中文标签"的**排版表**，也是输出顺序。
//
// ⚠️ 它是排版选择，不是 schema 声明。引擎里键的词汇在 `result` 与 `detail` 之间大量重叠
// （同一个文件里 `path`/`value`/`kind`/`attr`/`written` 两边都写），所以：
//
//   - 表可以随引擎增键而补，但**不许**被当成"引擎只会发这些键"；
//   - 这个文件**只吃 `error.detail`**，绝不碰 `result` —— 两股流本来不交叉，
//     别在这里把它们接起来（成功帧会被套上错误样式的标签）。
var detailOrder = []struct{ key, label string }{
	{"param", "参数"},
	{"reason", "原因"},
	{"what", "对象"},
	{"path", "位置"},
	{"kind", "种类"},
	{"attr", "属性"},
	{"value", "值"},
	{"expected", "期望"},
	{"got", "实际"},
	{"type", "类型"},
	{"types", "类型"},
	{"legal", "合法值"},
	{"hint", "提示"},
	{"candidates", "候选（key 可直接拿去重试）"},
	{"applied", "已写入"},
	{"noop", "未变化"},
	{"failed", "失败"},
	{"retry", "可重试"},
	{"timeoutSeconds", "超时秒数"},
	{"exception", "异常"},
	{"at", "抛出位置"},
	{"source", "依据"},
	{"written", "写入的值"},
	{"conflict", "冲突的路径"},
	{"message", "说明"},
}

// maxDetailItems 是每个列表最多打几行。
//
// 上限是拍的，判据是"最大的合法集要能完整打出来"：spec 侧属性白名单实测最大 23 个
// （`field` 类），40 留了一倍余量。真正会被截的是 `candidates`（一张全开的表单上
// 每个句柄一条）。超出的部分**必须说出来**（见下），这是本项目"截断绝不静默"的同一句话。
const maxDetailItems = 40

// printWireDetail 把 `e.Detail` 渲染到 w（缩进 4 空格，跟在错误那一行下面）。
//
// 永不 panic、永不因为看不懂就吞掉：解不动就原样打出来。没有 detail 时一个字都不打。
func printWireDetail(w io.Writer, e *tzs.WireError) {
	if e == nil || len(e.Detail) == 0 {
		return
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(e.Detail, &m); err != nil {
		// detail 不是一个对象。引擎没发过这种，但真发了也不该在这里死 ——
		// 原样打出来，读的人至少知道引擎说了什么。
		if s := strings.TrimSpace(string(e.Detail)); s != "" && s != "null" {
			common.Line(w, "    detail（原样）：%s", s)
		}
		return
	}

	seen := make(map[string]bool, len(m))
	// `legal` 有两种含义，而判据**是结构不是语义**：同一个 detail 里带 `value` 时它是
	// "这个值该取哪些"（值集），不带时是"这个位置接受哪些"（属性名集 / 类型集）。
	// 真机上见过后者，标签写"合法值"会读成"case 的合法值是 tag/posX/…"，那是错的。
	// 这条判据用到的只是"有没有那一项"，不是引擎的规则。
	legalLabel := "合法值"
	if _, hasValue := m["value"]; !hasValue {
		legalLabel = "可选项"
	}

	for _, d := range detailOrder {
		raw, ok := m[d.key]
		if !ok {
			continue
		}
		seen[d.key] = true
		// `message` 常常就是上面那一行 error.message 的复述，重复一遍只是噪音。
		if d.key == "message" && unquote(raw) == e.Message {
			continue
		}
		label := d.label
		if d.key == "legal" {
			label = legalLabel
		}
		printDetailValue(w, label, raw)
	}

	// 表里没有的键：按名字排序、用原名当标签。**这就是"引擎加键不用改 Go"的落点。**
	var rest []string
	for k := range m {
		if !seen[k] {
			rest = append(rest, k)
		}
	}
	sort.Strings(rest)
	for _, k := range rest {
		printDetailValue(w, k, m[k])
	}
}

// printDetailValue 按 **JSON 类型**渲染一个值。
//
// 按类型而不是按键名分派，是因为同一种形状在不同函数里的键名不同
// （`legal` / `candidates` / `applied` / `failed` 都是数组），
// 而"数组就是列表、其它就是一行"这条判据对它们全都成立。
func printDetailValue(w io.Writer, label string, raw json.RawMessage) {
	var list []json.RawMessage
	if err := json.Unmarshal(raw, &list); err == nil {
		if len(list) == 0 {
			return
		}
		common.Line(w, "    %s：", label)
		for i, el := range list {
			if i == maxDetailItems {
				common.Line(w, "      …还有 %d 个没打（完整 detail 用 --json 看）", len(list)-maxDetailItems)
				break
			}
			common.Line(w, "      - %s", unquote(el))
		}
		return
	}

	if s, ok := unquoteOK(raw); ok {
		if s == "" {
			// 空串是"没设/继承"，打出来只会让人以为有内容。
			return
		}
		common.Line(w, "    %s：%s", label, s)
		return
	}
	if strings.TrimSpace(string(raw)) == "null" {
		return
	}
	// 对象或其它：紧凑 JSON 原样打。**不逐字段拆 candidates 那种对象** ——
	// 那正是"在 Go 侧抄一份 schema"，抄了就会漂。
	common.Line(w, "    %s：%s", label, strings.TrimSpace(string(raw)))
}

// unquote 把一个 JSON 值按字符串打出来：是字符串就去引号，不是就返回紧凑原文。
func unquote(raw json.RawMessage) string {
	if s, ok := unquoteOK(raw); ok {
		return s
	}
	return strings.TrimSpace(string(raw))
}

func unquoteOK(raw json.RawMessage) (string, bool) {
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return "", false
	}
	return s, true
}
