package output

import (
	"fmt"
	"strings"
)

// Meta 一次输出的"环境信息"块 —— 它回答的是:这次到底落在哪。
//
// 为什么要有:同一个企业编号在开发区与正式区是两套完全不同的库,同一个环境里
// 不同账号又是不同的 schema。少回显一半,就可能把"查错了地方"读成"数据不存在"。
//
// 字段**全可空,空的整段不打印** —— 本地源拿不到 SSH 就不出现 SSH 段,
// 绝不打出 `环境  · SSH  · 区域 ` 这种一个值都没有的空壳。
type Meta struct {
	// ---- 视图 ----
	// Detail 本次结果是摘要还是完整视图（summary | full）；空 = 不区分。
	// 它不是截断标志：截断看 Truncated/Returned；Detail 让 agent 知道默认结果是否
	// 省略了详细字段，下一步该不该加 --full。
	Detail string `json:"detail,omitempty"`

	// ---- 出处 ----
	Source        string `json:"source,omitempty"` // local | live
	Env           string `json:"env,omitempty"`
	SSHHost       string `json:"sshHost,omitempty"`
	Zone          string `json:"zone,omitempty"`
	Topent        string `json:"topent,omitempty"` // 环境配的默认企业编号(原样,可能非数字)
	Ent           int    `json:"ent,omitempty"`    // 本次实际用的企业编号
	Account       string `json:"account,omitempty"`
	AccountSource string `json:"accountSource,omitempty"` // accounts[0] | ent(gzou_t) | flag
	Target        string `json:"target,omitempty"`        // 库地址 host:port/svc
	Dialect       string `json:"dialect,omitempty"`       // oracle | kingbase | sqlite
	Route         string `json:"route,omitempty"`         // client-direct | server-sqlplus | local-sqlite …
	ViaTunnel     bool   `json:"viaTunnel,omitempty"`
	Readonly      bool   `json:"readonly,omitempty"`    // 库会话是只读事务
	LocalDBPath   string `json:"localDbPath,omitempty"` // 仅本地 SQLite 源

	// ---- 计数与截断 ----
	TotalRows int `json:"totalRows"`
	Returned  int `json:"returned"`
	// Offset 是这一页从第几条开始（从 0 数）—— 只有翻页读落盘结果时才非零
	// （tt dict spill show）。它让"这是第 N–M 条"这件事在**信封里**就有答案，
	// 不必让调用方自己数返回数组。omitempty：既有命令的输出一个字节不变。
	Offset        int    `json:"offset,omitempty"`
	Truncated     bool   `json:"truncated,omitempty"`
	TruncReason   string `json:"truncReason,omitempty"`
	ServerLimited bool   `json:"serverLimited,omitempty"`

	// TruncHint 只影响人读/CSV 的"行数"提示，不进入 JSON 信封。
	// 不同命令的全量入口不同（ACC 用 query.limit，SQL 用 --limit 0），
	// 不能让人读头写死一个错的下一步。
	TruncHint string `json:"-"`

	// ---- 落盘与杂项 ----
	LocalPath    string   `json:"localPath,omitempty"`
	SpillPartial bool     `json:"spillPartial,omitempty"`
	Elapsed      float64  `json:"elapsedSeconds,omitempty"`
	Notes        []string `json:"notes,omitempty"`
}

// headerLines 把 Meta 渲染成 `# ` 前缀的注释行(CSV 模式的自描述头)。
//
// 每行都是"非空才拼",整行无值就不出现 —— 与 tt debug sql 的头部同一条规矩。
func (m Meta) headerLines() []string {
	var out []string
	if m.Detail != "" {
		out = append(out, "视图 "+m.Detail)
	}

	// 第一行:环境在哪台机器上(本地源没有这些,整行不出现)
	if l := joinNonEmpty(" · ", kv("环境", m.Env), kv("SSH", m.SSHHost), kv("区域", m.Zone)); l != "" {
		out = append(out, l)
	}

	// 第二行:连的是谁。企业编号与账号**同在一行**,因为"哪个企业 → 哪个账号"是一条链,
	// 拆成两行会让人以为可以各自独立地看。
	var parts []string
	if m.Ent > 0 {
		parts = append(parts, fmt.Sprintf("企业(ENT) %d → 账号 %s", m.Ent, m.Account))
	} else if m.Account != "" {
		parts = append(parts, "账号 "+m.Account)
	}
	if m.AccountSource != "" {
		parts = append(parts, "来源 "+m.AccountSource)
	}
	parts = append(parts, kv("库", m.Dialect), m.Target, m.Route)
	if m.ViaTunnel {
		parts = append(parts, "经 SSH 隧道")
	}
	if m.Readonly {
		parts = append(parts, "只读事务")
	}
	if m.LocalDBPath != "" {
		parts = append(parts, m.LocalDBPath)
	}
	if m.Elapsed > 0 {
		parts = append(parts, fmt.Sprintf("用时 %.2fs", m.Elapsed))
	}
	if l := joinNonEmpty(" · ", parts...); l != "" {
		out = append(out, l)
	}

	out = append(out, m.countLine())
	if m.LocalPath != "" {
		out = append(out, fmt.Sprintf("落盘 %s(完整 %d 行;本次回 %d 行)", m.LocalPath, m.TotalRows, m.Returned))
	}
	for _, n := range m.Notes {
		out = append(out, "注:"+n)
	}
	return out
}

// countLine 行数行。0 行时必须点名"上错号也会是 0 行" —— 那是本工具最容易误读的一种结果。
//
// 截断时把"总共多少、回了多少、完整在哪"一次说清:截断是刻意的保护,但**绝不能静默** ——
// 调用方拿着残缺结果当成全部去下结论,比给一坨大的坏得多。
func (m Meta) countLine() string {
	switch {
	case m.Truncated:
		// ServerLimited 时 TotalRows 只是**下界**(库侧包装时故意多取一行判"还有更多"),
		// 写成"共 N 条"会让人以为结果就这么多;加"库侧返回 ≥"把它说实。
		count := "共 "
		if m.ServerLimited {
			count = "库侧返回 ≥"
		}
		where := ""
		if m.LocalPath != "" {
			where = ",完整结果见落盘文件"
		}
		action := "要直接看全量加 --limit 0"
		if m.TruncHint != "" {
			action = m.TruncHint
		}
		return fmt.Sprintf("行数 %d(已截断:%s%d 条,只回了前 %d 条%s;%s)",
			m.Returned, count, m.TotalRows, m.Returned, where, action)
	case m.TotalRows == 0:
		return "行数 0(注意:上错号/上错环境也会是 0 行,先核对上面的企业编号)"
	default:
		return fmt.Sprintf("行数 %d", m.TotalRows)
	}
}

func kv(k, v string) string {
	if v == "" {
		return ""
	}
	return k + " " + v
}

func joinNonEmpty(sep string, parts ...string) string {
	keep := make([]string, 0, len(parts))
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			keep = append(keep, p)
		}
	}
	return strings.Join(keep, sep)
}
