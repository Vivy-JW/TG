package tzs

// state.go —— `.tt-tzs.json` 的读写。
//
// 它**必须与 `.tt-serve.json` 分开**，不能合并成一个「反正都是 pid 记录」的文件：
// serve 那份的 readServeInfo 要求 URL 非空（`if st.PID <= 0 || st.URL == "" { 无效 }`），
// 一旦里面混进一条没有 URL 的 tzs 守护进程记录，`tt serve` 的单实例判断会当场认为
// 状态文件损坏（于是再拉一个服务抢端口），而 `tt serve --stop` 会拿着 tzs 的 pid 去杀
// —— 指错人是这类文件最坏的失败方式：它看起来「成功了」。
//
// 结构按工作区键控（同一个工作区在任意时刻只能有一个守护进程）：引擎的 Boot 只做一次，
// 一个进程永久绑定一个工作区，换工作区 = 换管道名 = 换进程。多个工作区并存是正常的，
// 所以这里是一个 map 而不是一条记录。
//
// 键是**小写的工作区原串**，刻意不做 filepath.Clean：
//
//   - 小写是引擎自己的口径（Rpc.PipeName 里 `ws.ToLowerInvariant()` 才去 hash），
//     Windows 路径大小写不敏感，两种拼法必须是同一条记录。
//   - 不做 Clean/去尾斜杠是因为那会**改变身份**：引擎 hash 的是字面串，
//     `D:\ws` 与 `D:\ws\` 是两个不同的管道名、两个不同的守护进程。
//     在这里把它们归一成一条，第二个守护进程的记录就会覆盖第一个，
//     Reap 从此看不到它。

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	// StateFileName 是状态文件名。与 .tt-serve.json 分开的理由见文件注释。
	StateFileName = ".tt-tzs.json"
	// LogFileName 是守护进程的日志（winproc.SpawnDetached 把 stdout/stderr 都追加到这里）。
	LogFileName = ".tt-tzs.log"
	// StateVersion 是结构版本号（将来改结构时用它判「要不要迁移」）。
	StateVersion = 1
)

// DaemonEntry 是一个工作区当前的守护进程记录。
type DaemonEntry struct {
	PID       int    `json:"pid"`
	Pipe      string `json:"pipe"`
	Workspace string `json:"workspace"`
	Exe       string `json:"exe"`
	Log       string `json:"log"`
	Since     string `json:"since"`
	// Orphans 是被我们挤掉的、仍在运行的**旧构建**守护进程的 pid。
	//
	// 为什么需要它：管道名含 MVID，重编引擎之后同工作区的新守护进程会拿到新名字，
	// 上面这条记录就被覆盖了 —— 旧进程还在跑，而它的 pid 已经无处可查。
	// 没有这个字段，Reap 只能收「从没被重新 spawn 过的工作区」的孤儿，
	// 而重编引擎之后第一次调用恰恰是最常见的情形。
	//
	// 这是对参考结构的一处**新增**（多一个可选字段），老状态文件照样能读。
	Orphans []int `json:"orphans,omitempty"`
}

// LastEntry 是最近一次 open 的指引（`open` 成功时写下来）。
//
// **命令层不读它来补 handle**：每个需要句柄的动词都必须显式给 handle（见
// skills/tt-dev-tzs）。留它是为了"上一次开的是哪个包"这件事有个落点，
// 句柄本身只活在守护进程里（引擎内自增、永不复用），进程一死全部失效 ——
// 拿它去填等于把"可能已经没了"当成已知事实。
type LastEntry struct {
	Workspace string `json:"workspace,omitempty"`
	Handle    string `json:"handle,omitempty"`
	OpenPath  string `json:"openPath,omitempty"`
}

// State 是 `.tt-tzs.json` 的类型化视图。
type State struct {
	Version int                     `json:"version"`
	Daemons map[string]*DaemonEntry `json:"daemons"`
	Last    *LastEntry              `json:"last,omitempty"`
}

// StatePath 返回状态文件路径（dir = 配置目录 / Options.WorkDir）。
func StatePath(dir string) string { return filepath.Join(dir, StateFileName) }

// LogPath 返回守护进程日志路径。
func LogPath(dir string) string { return filepath.Join(dir, LogFileName) }

// stateKey 是工作区的状态键：小写，仅此而已（理由见文件注释）。
func stateKey(ws string) string { return strings.ToLower(ws) }

// Entry 取某工作区的记录；没有返回 nil。
func (s *State) Entry(ws string) *DaemonEntry {
	if s == nil || s.Daemons == nil {
		return nil
	}
	return s.Daemons[stateKey(ws)]
}

// SetEntry 写入/覆盖某工作区的记录。
func (s *State) SetEntry(ws string, e *DaemonEntry) {
	if s.Daemons == nil {
		s.Daemons = map[string]*DaemonEntry{}
	}
	s.Daemons[stateKey(ws)] = e
}

// DropEntry 删掉某工作区的记录（不存在时是 no-op）。
func (s *State) DropEntry(ws string) {
	if s != nil && s.Daemons != nil {
		delete(s.Daemons, stateKey(ws))
	}
}

// Keys 返回全部键，**排序**过（Reap/doctor 的输出必须可复现 —— map 顺序随机会让
// 同一份状态在两个进程里报出不同的清单，看起来像状态在变）。
func (s *State) Keys() []string {
	if s == nil {
		return nil
	}
	out := make([]string, 0, len(s.Daemons))
	for k := range s.Daemons {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// LoadState 严格读取状态文件：文件不存在返回空状态（不是错误），
// 文件存在但读不动/解不动则返回错误。
func LoadState(dir string) (*State, error) {
	b, err := os.ReadFile(StatePath(dir))
	if err != nil {
		if os.IsNotExist(err) {
			return newState(), nil
		}
		return nil, err
	}
	var s State
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, fmt.Errorf("状态文件 %s 不是合法 JSON: %w", StatePath(dir), err)
	}
	if s.Daemons == nil {
		s.Daemons = map[string]*DaemonEntry{}
	}
	return &s, nil
}

// loadState 是容错版：读不动就用空状态继续，并把坏文件挪到 .bad 留证。
//
// 为什么容错：这个文件只是**记录**，不是真相（真相是管道上有没有人应答）。
// 让一个手改坏的、或写到一半断电的 JSON 把 `tt dev tzs` 全线挡住，
// 是拿一条诊断信息否决整个工具 —— servebg.go 的 StopBackground 面对同一个问题
// 选了同一条路（「不让一个陈旧的状态文件把下一次启动挡住」）。
func loadState(dir string) *State {
	s, err := LoadState(dir)
	if err == nil {
		return s
	}
	bad := StatePath(dir) + ".bad"
	if rerr := os.Rename(StatePath(dir), bad); rerr == nil {
		fmt.Fprintf(os.Stderr, "[tt tzs] 状态文件损坏，已挪到 %s 并重新开始: %v\n", bad, err)
	}
	return newState()
}

func newState() *State {
	return &State{Version: StateVersion, Daemons: map[string]*DaemonEntry{}}
}

// SaveState 原子写状态文件（临时文件 + rename）。
//
// 为什么不用 os.WriteFile：两个 `tt dev tzs`（比如人手一条 AI 一条）可能同时更新它，
// 半写状态会被下一次 loadState 判成损坏 —— 而那会把两条守护进程记录一起丢掉。
// rename 在同一目录内是原子的，读到的要么是旧的完整内容、要么是新的完整内容。
func SaveState(dir string, s *State) error {
	if s == nil {
		return nil
	}
	if s.Version == 0 {
		s.Version = StateVersion
	}
	if s.Daemons == nil {
		s.Daemons = map[string]*DaemonEntry{}
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, StateFileName+".tmp*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, StatePath(dir)); err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
}

//---------------------------------------------------------------------------
// last：最近一次 open 的指引
//---------------------------------------------------------------------------

// Remember 记下最近一次 open 的工作区 / 句柄 / 包路径。
//
// 命令层在 `open` 成功之后调它，让后续的 `call` 能省略 `--handle`。
// 记住的东西**不是真相**：句柄只活在守护进程里（引擎内自增、永不复用），进程一死
// 全部失效。所以这里不做任何校验，也绝不在本地「确认句柄还有效」——
// 那需要 Boot，正是我们要避免的事。失效的句柄会让引擎回 E_NO_HANDLE
// （kind=not_found，退 2），那是安全且自解释的失败。
func Remember(o Options, handle, openPath string) error {
	ws, err := o.workspace()
	if err != nil {
		return err
	}
	dir := o.StateDir()
	st := loadState(dir)
	st.Last = &LastEntry{Workspace: ws, Handle: handle, OpenPath: openPath}
	return SaveState(dir, st)
}

// Last 读回最近一次 open 的记录；没有时返回 nil。
func Last(o Options) *LastEntry {
	st := loadState(o.StateDir())
	return st.Last
}
