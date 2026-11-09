package debug

// 会话操作的**审计行**（WP6 的验收项之一：每次会话操作一条 —— 谁 / 哪条 / 干了什么）。
//
// # 钉住的条款
//
//   - **C5.4**：「机器对账、错误文案、**审计行**用它」—— 所以"哪条"写的是**规范 id**
//     （`环境#序号`，生命周期内恒定，C2.0），不是会随 `SetRun` 漂的 `display`；
//   - **C3.7c / Q20**：跨轮追踪一条会话用别名/规范 id，不用序号 —— 审计行照这条写；
//   - **计划 WP6 的审计落点**：会话的**建立 / 复用 / 收口 / 歧义拒绝**至少这四处。
//
// # 形状与住处
//
// 形状**不在这里定义**：它是 `internal/target` 的 `AuditEntry`（`NewAuditEntry` /
// `JSONLine` / `HumanLine` / `DefaultActor`），与 target 的 `AppendAudit`
// 是同一族（一行一条 JSON、字段名固定、时间 RFC3339）。本文件只做两件事：
//
//  1. 把某个会话的当前身份投影成审计行的 `ref`（`auditRefOf`）；
//  2. 把审计行落盘（`Manager.audit`）——**写不进去只记一行日志，绝不让会话操作失败**：
//     审计是旁路，它坏了不该把"会话起不来"这种后果带给用户。
//
// 落点是 `<DataDir>/audit/sessions.jsonl`。它**不进** `config.CacheSubdirs`：
// 那份清单是"可再生的缓存，清缓存可以删"，而审计行是事实记录（与 `ents` 快照同一层
// 的落盘口径：`<DataDir>/<子目录>/…`，见 ents.go 的 entSnapshotPath）。
// `DataDir` 为空（单测里的 `NewManager(&Config{})` 与未开持久化的部署）时**不落盘**，
// 与断点存档/镜像同一款兜底。

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"tt/internal/target"
)

// auditDir / auditFile 审计行的落点：<DataDir>/audit/sessions.jsonl。
const (
	auditDir  = "audit"
	auditFile = "sessions.jsonl"
)

// auditPath 审计文件路径；DataDir 为空 = 不落盘（返回空串，调用方跳过）。
func auditPath(dataDir string) string {
	if dataDir == "" {
		return ""
	}
	return filepath.Join(dataDir, auditDir, auditFile)
}

// 审计动作名（**干了什么**）。字面量写在这里，判据测试照着它们断言。
const (
	// auditCreate：会话在服务端登记（建立）。四条造会话的路径都过 assignIdentityLocked。
	auditCreate = "session.create"
	// auditReuse：复用空闲宿主跑新一轮（SetRun 换代；序号与别名不动，C2.0 / C2.5b）。
	auditReuse = "session.reuse"
	// auditCollect：换目标/换名字时收口一条空闲宿主（它的别名与序号当场释放，C3.7b / Q19）。
	auditCollect = "session.collect"
	// auditAmbiguous：落不上目标 —— 报歧义并列候选，绝不替调用方挑一条（F1 / C8.1）。
	auditAmbiguous = "session.ambiguous"
	// auditBusy：点名的别名/唯一宿主不可复用（正在跑 / 已停站，F20 / Q23）。
	auditBusy = "session.busy"
)

// auditRefOf 取一条会话的**规范 id**（C5.4 / C2.0）。
//
// 走 `SessionBrief` → `target.Session.CanonicalRef()` 这条**唯一**的投影路（与候选清单
// 同一个函数，C5.12 的"候选键集唯一出处"是同一条纪律）：不许在这里手拼第二个地址 ——
// 手拼的那一份与服务端产出的 `ref` 一旦漂开，审计行就对不上会话了。
//
// 序号还没分配（手工造的会话）时 `CanonicalRef()` 退到原始句柄 `id`，绝不给空串。
func (m *Manager) auditRefOf(s *Session) string {
	if s == nil {
		return ""
	}
	return sessionView(m.briefOf(s)).CanonicalRef()
}

// audit 记一条审计（谁 / 哪条 / 干了什么）。
//
// 调用方**不必持有 `m.mu`**：它自己取会话的锁（`briefOf`），与 `Snapshot` 同一款。
func (m *Manager) audit(s *Session, action string) {
	path := m.auditPath
	if path == "" {
		return
	}
	e := target.NewAuditEntry(m.auditActor(target.ActorEnv), m.auditRefOf(s), action, time.Now())
	line, err := e.JSONLine()
	if err != nil {
		log.Printf("[audit] 序列化失败(%s %s): %v", action, e.Ref, err)
		return
	}
	if err := appendAuditLine(path, line); err != nil {
		// 审计是旁路：写不进去记一行日志，不把失败往上抛（那会让"会话起不来"看起来
		// 像审计的问题，而真相常常只是磁盘满 / 权限）。
		log.Printf("[audit] 追加失败 %s: %v", path, err)
	}
}

// appendAuditLine 把一行审计追加到 path（不存在就建目录 / 建文件）。
//
// 单独一个函数是为了让它可测：`audit` 那一层要真会话，这一层只要一个路径。
func appendAuditLine(path, line string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	// 一行一条：JSONLine 自己不带换行，这里补上（形状见 target 的 AppendAudit）。
	_, err = f.WriteString(line + "\n")
	return err
}

// auditLines 读出审计文件的全部行（判据测试用；读不到就是空）。
func auditLines(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var out []string
	for _, ln := range strings.Split(string(data), "\n") {
		if ln = strings.TrimRight(ln, "\r"); ln != "" {
			out = append(out, ln)
		}
	}
	return out
}

// auditEntryOf 解析一行审计（判据测试按字段断言，不靠字符串包含）。
func auditEntryOf(line string) (target.AuditEntry, error) {
	var e target.AuditEntry
	err := json.Unmarshal([]byte(line), &e)
	return e, err
}
