// 统一路径管理器：tt 访问的一切自有外部资源（配置、缓存子目录、服务状态与日志、
// 本地字典库、.tzs 引擎）的落点都从这里取。调用方不得自己 filepath.Join 出这些路径 ——
// 那就是第二份位置解析，某天两边悄悄分叉（历史上"服务写一处、命令行读另一处"
// 就是这样来的）。
//
// 数据目录 = 配置所在目录（见 paths.go）。缺省形态下即 %APPDATA%\tt；
// 便携包配置在包内，数据也跟着包走。两种形态下 DataDir 都是"config.json 旁边"。
package config

import (
	"errors"
	"os"
	"path/filepath"
)

// DefaultDictDBName 本地字典库的缺省文件名，放数据目录下。
const DefaultDictDBName = "erp_data.db"

// 服务状态与日志的文件名（数据目录下）。状态文件是**运行中**服务的进程记录，
// 日志是排障依据 —— 与 config.json 同为"不是缓存"的东西（见 cache.go 的清单）。
const (
	ServeStateFileName = ".tt-serve.json"
	ServeLogFileName   = ".tt-serve.log"
)

// Locations 一次运行所需的全部落点，由配置路径派生。
// 从头解析用 ResolveLocations；调用方已拿到配置路径时用 LocationsAt 包装。
type Locations struct {
	ConfigPath string // config.json 的落点
}

// ResolveLocations 解析配置路径并汇出全部落点。
func ResolveLocations(configFlag string, allowMissing bool) (*Locations, error) {
	p, err := ResolvePath(configFlag, allowMissing)
	if err != nil {
		return nil, err
	}
	return &Locations{ConfigPath: p}, nil
}

// LocationsAt 用已解析好的配置路径构造（不检查存在性）。
func LocationsAt(configPath string) *Locations {
	return &Locations{ConfigPath: configPath}
}

// DataDir 数据目录 = 配置所在目录（ents/、srccache/、.tt-serve.json 都跟着它）。
func (l *Locations) DataDir() string {
	if l.ConfigPath == "" {
		return ""
	}
	return filepath.Dir(l.ConfigPath)
}

// CacheDir 数据目录下的缓存子目录；name 必须在 CacheSubdirs 清单里（见 cache.go），
// 否则返回空串 —— 自有存储只允许清单里的名字，写错名字当场变空串，而不是建出新目录。
func (l *Locations) CacheDir(name string) string {
	return CacheDir(l.DataDir(), name)
}

// StateFile 后台服务状态文件（.tt-serve.json）：pid/地址/API 前缀，--stop 靠它寻址。
func (l *Locations) StateFile() string {
	return filepath.Join(l.DataDir(), ServeStateFileName)
}

// LogFile 后台服务日志（.tt-serve.log）。
func (l *Locations) LogFile() string {
	return filepath.Join(l.DataDir(), ServeLogFileName)
}

// DictDBCandidates 本地字典库（erp_data.db）的候选路径，按优先级：
// TDICT_DB 环境变量 → -d/--db（相对路径按当前目录绝对化）→ 数据目录下的缺省名。
func (l *Locations) DictDBCandidates(flagPath string) []string {
	var out []string
	if env := os.Getenv("TDICT_DB"); env != "" {
		out = append(out, env)
	}
	if p := AbsPath(flagPath); p != "" {
		out = append(out, p)
	}
	if dir := l.DataDir(); dir != "" {
		out = append(out, filepath.Join(dir, DefaultDictDBName))
	}
	return out
}

// ResolveDictDB 读侧定位：返回第一个**存在**的候选。全部不存在时报错并列出尝试过的
// 路径 —— 查询命令读不到库就该当场失败，不悄悄换一份别的库。
func (l *Locations) ResolveDictDB(flagPath string) (string, []string, error) {
	var tried []string
	for _, p := range l.DictDBCandidates(flagPath) {
		abs, _ := filepath.Abs(p)
		if _, err := os.Stat(abs); err == nil {
			return abs, tried, nil
		}
		tried = append(tried, abs)
	}
	return "", tried, errors.New("本地数据库文件未找到")
}

// SyncTarget 写侧定位（数据同步往哪写）：
//  1. 配置里显式的 sync.target（设置页与 CLI 必须写同一个文件）；
//  2. 已存在的库 —— 与读侧同一份候选，写进查询命令正读着的那份；
//  3. -d 给的是绝对路径时写那里（文件由同步过程创建）；
//  4. 都没有 → 数据目录下的缺省名。
//
// 永不报错：同步过程负责创建文件与父目录。
func (l *Locations) SyncTarget(configured, flagPath string) string {
	if p := AbsPath(configured); p != "" {
		return p
	}
	cands := l.DictDBCandidates(flagPath)
	for _, p := range cands {
		abs, _ := filepath.Abs(p)
		if _, err := os.Stat(abs); err == nil {
			return abs
		}
	}
	if filepath.IsAbs(flagPath) {
		return AbsPath(flagPath)
	}
	if len(cands) == 0 {
		return ""
	}
	abs, _ := filepath.Abs(cands[len(cands)-1])
	return abs
}

// EngineExe .tzs 引擎 exe 的落点：显式覆盖（tzs.serverExe）优先，
// 否则 <tt.exe 目录>\tzs\tzs-server.exe（与 skills/ 同一种分发形态：exe 同目录的
// 子目录，不进二进制）。定位不到 tt.exe 自身且无覆盖时返回空串。
func EngineExe(override string) string {
	if p := AbsPath(override); p != "" {
		return p
	}
	self, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Join(filepath.Dir(self), "tzs", "tzs-server.exe")
}
