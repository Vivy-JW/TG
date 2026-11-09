package target

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// 审计行（计划 WP6 的验收项之一：**每次会话操作有一条审计行**）（谁 / 哪条 / 干了什么）。
//
// 形状取本仓库既有审计的同一族：`internal/form42f` 的 `AppendAudit`（`internal/form42f/do.go:571-605`）
// 写的就是"一条 JSON 行"，字段名固定、时间为 RFC3339。这里不另立一种日记格式。
//
// 「哪条」写的是**规范 id**（`ref`）：契约 C5.4 明写"机器对账、错误文案、审计行用它"，
// 因为它在会话生命周期内恒定不变（C2.0）—— 拿会漂的 `display`（含作业段）记审计，
// 事后就没法把两条记录对上同一条会话。

// ActorEnv 是"谁"的缺省来源：环境变量 `TT_ACTOR`。
//
// 它是"这次调用算谁的动作"的唯一开关，由**发起方**在起子进程时设置。为什么需要它：那条链路是
// "人点界面按钮 → serve 起 tt 子进程 → 子进程再调调试线 REST"，而 CLI 默认声明 X-Actor: ai
// （命令行本来是 AI 的接口）。于是纯人工模式下，人点出来的动作会被自己设的模式闸门拦下
// （实测 2026-10-16："纯人工模式:AI 不可操作"）。serve 起这条链路时把 TT_ACTOR=human 传下去，
// 子进程逐层继承（exec.Command 默认继承环境）。
//
// 调试面（`internal/debug`）、它的客户端（`internal/debugclient`）与审计行都读这一份，
// **不另立第二个开关**，也不在各自包里重复定义常量。
const ActorEnv = "TT_ACTOR"

// AuditEntry 是一条会话操作审计。
type AuditEntry struct {
	Time   time.Time `json:"time"`
	Actor  string    `json:"actor"`  // 谁
	Ref    string    `json:"ref"`    // 哪条：规范 id（C2.0 / C5.4）
	Action string    `json:"action"` // 干了什么
}

// NewAuditEntry 造一条审计。actor 取不到（空/空白）时给字面量占位 ——
// 审计行缺一段比写着一个 `?` 更难查。
func NewAuditEntry(actor, ref, action string, at time.Time) AuditEntry {
	if strings.TrimSpace(actor) == "" {
		actor = UnknownPlaceholder
	}
	return AuditEntry{Time: at, Actor: actor, Ref: ref, Action: action}
}

// JSONLine 把这条审计序列化成**一行**（不含换行），字段顺序固定。
//
// 没给时间就补当前时间（与 `internal/form42f.AppendAudit` 同一口径）：审计行不许出现零值时间。
func (e AuditEntry) JSONLine() (string, error) {
	if e.Time.IsZero() {
		e.Time = time.Now()
	}
	b, err := json.Marshal(e)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// HumanLine 是人话形态：`审计 <time> · 谁 <actor> · 哪条 <ref> · 干了什么 <action>`。
func (e AuditEntry) HumanLine() string {
	at := e.Time
	if at.IsZero() {
		at = time.Now()
	}
	return fmt.Sprintf("审计 %s · 谁 %s · 哪条 %s · 干了什么 %s",
		at.Format(time.RFC3339), e.Actor, e.Ref, e.Action)
}

// DefaultActor 是"谁"的缺省来源：读 `TT_ACTOR`（见 ActorEnv）。
//
// getenv 由调用方注入（本包不直接读环境变量：那会让纯函数变成不可测的副作用）。
// 取不到就给字面量占位，绝不返回空串。
func DefaultActor(getenv func(string) string) string {
	if getenv != nil {
		if v := strings.TrimSpace(getenv(ActorEnv)); v != "" {
			return v
		}
	}
	return UnknownPlaceholder
}
