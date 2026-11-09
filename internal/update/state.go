package update

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"tt/internal/config"
)

// 升级状态机的取值。前台进程与脱离的更新器进程都读写同一份状态，所以阶段名是契约：
// `tt update log` 与设置页按它回答"上一次升级走到哪一步、卡在哪"。
const (
	PhaseIdle        Phase = "idle"
	PhaseChecking    Phase = "checking"
	PhaseDownloading Phase = "downloading"
	PhaseVerifying   Phase = "verifying"
	PhaseHandedOff   Phase = "handed-off" // 前台已交棒，更新器还没开始动手
	PhaseApplying    Phase = "applying"
	PhaseDone        Phase = "done"
	PhaseFailed      Phase = "failed"
)

// Phase 升级阶段。
type Phase string

// State 升级状态。落 `<数据目录>/.tt-update.json` —— **不是缓存**（见 config/cache.go 的
// 边界）：它记录"上一次升级发生了什么"，删了就没法回答"为什么没升成"，与 .tt-serve.json
// 同类。
type State struct {
	Phase      Phase     `json:"phase"`
	From       string    `json:"from,omitempty"`     // 升级前的版本
	Target     string    `json:"target,omitempty"`   // 目标版本
	Kind       string    `json:"kind,omitempty"`     // 安装形态（便携包 / MSI 安装）
	Artifact   string    `json:"artifact,omitempty"` // 下载到的制品
	LogPath    string    `json:"logPath,omitempty"`
	Error      string    `json:"error,omitempty"`
	Warnings   []string  `json:"warnings,omitempty"` // 升级成功但收尾没做成（如技能没刷上）
	StartedAt  time.Time `json:"startedAt,omitempty"`
	FinishedAt time.Time `json:"finishedAt,omitempty"`
}

// StatePath 状态文件落点。
func StatePath(dataDir string) string { return filepath.Join(dataDir, ".tt-update.json") }

// LogPath 升级日志落点（与 .tt-serve.log 同理：排障要看，且不影响正确性）。
func LogPath(dataDir string) string { return filepath.Join(dataDir, ".tt-update.log") }

// LoadState 读状态；缺失或坏掉时返回 nil（调用方按"没升过"处理）。
func LoadState(dataDir string) *State {
	if dataDir == "" {
		return nil
	}
	b, err := os.ReadFile(StatePath(dataDir))
	if err != nil {
		return nil
	}
	var st State
	if err := json.Unmarshal(b, &st); err != nil {
		return nil
	}
	return &st
}

// SaveState 写状态（原子写）。
func SaveState(dataDir string, st *State) error {
	if dataDir == "" {
		return fmt.Errorf("数据目录不可用，升级状态无处可落")
	}
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return config.AtomicWrite(StatePath(dataDir), append(b, '\n'))
}

// AppendLog 往升级日志追一行（带时间戳）。日志只用来回答"更新器做了什么"，
// 所以失败**不**往上抛：写不进日志不该让升级本身失败。
func AppendLog(logPath, format string, args ...any) {
	if logPath == "" {
		return
	}
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s  %s\n", time.Now().Format("2006-01-02 15:04:05"), fmt.Sprintf(format, args...))
}

// InProgress 报告当前状态是不是"有一次升级正在进行"。
//
// 幂等的入口：前台命令靠它拒绝第二次交棒，更新器靠它认出"这次升级已经做过了"
// （`done` 是终态，重复启动不是错）。
func (s *State) InProgress() bool {
	if s == nil {
		return false
	}
	switch s.Phase {
	case PhaseChecking, PhaseDownloading, PhaseVerifying, PhaseHandedOff, PhaseApplying:
		return true
	}
	return false
}
