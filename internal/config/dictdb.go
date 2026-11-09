package config

// 本地字典库的落点：**一个环境一个 SQLite 文件**。
//
// 以前只有一个 <数据目录>/erp_data.db，两个环境先后同步会写进同一个文件 ——
// 查「本地」时可能读到另一个客户的数据，而表现和「数据不存在」一模一样。
// 分文件之后，同步/查询/状态/过期判断都按环境各管各的，结构上不可能串。
//
// 旧文件 erp_data.db **不再默认使用**：它没有环境标记，拆不出来。显式 -d/--db
// 仍可打开它做对照。

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// DictDBDirName 是数据目录下存放各环境本地字典库的子目录。
const DictDBDirName = "dict"

var errLocalDBNotFound = errors.New("本地数据库文件未找到")

// DictDBDir 返回本地字典库目录（数据目录下 dict/）。
func DictDBDir(dataDir string) string {
	if strings.TrimSpace(dataDir) == "" {
		return ""
	}
	return filepath.Join(dataDir, DictDBDirName)
}

// DictDBFileName 把环境名变成安全的文件名：Windows 不允许的字符换成 _，
// 去掉首尾的点与空格；名字被改写过时追加短哈希，避免“A/B”与“A_B”撞车。
func DictDBFileName(env string) string {
	name := strings.TrimSpace(env)
	safe := strings.Map(func(r rune) rune {
		switch {
		case r == 92, r == '/', r == ':', r == '*', r == '?', r == '"', r == '<', r == '>', r == '|', r == 0:
			return '_'
		}
		if r < 0x20 {
			return '_'
		}
		return r
	}, name)
	safe = strings.Trim(safe, ". ")
	if safe == "" {
		safe = "env"
	}
	if safe != name && name != "" {
		sum := sha1.Sum([]byte(name))
		safe += "-" + hex.EncodeToString(sum[:])[:8]
	}
	return safe + ".db"
}

// DictDBPath 某环境的本地字典库文件：<数据目录>/dict/<环境>.db。
func DictDBPath(dataDir, env string) string {
	dir := DictDBDir(dataDir)
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, DictDBFileName(env))
}

// DictDBPathIn 在给定目录下拼某环境的库文件（设置页与同步任务共用同一份命名）。
func DictDBPathIn(dir, env string) string {
	if strings.TrimSpace(dir) == "" {
		return ""
	}
	return filepath.Join(dir, DictDBFileName(env))
}

// DictDBDir 数据目录下的本地字典库目录。
func (l *Locations) DictDBDir() string { return DictDBDir(l.DataDir()) }

// DictDBPath 某环境的本地字典库文件。
func (l *Locations) DictDBPath(env string) string { return DictDBPath(l.DataDir(), env) }

// DictDBDirFor 本地字典库根目录（读配置）：顶层 dict.dir 优先，否则 <数据目录>/dict。
//
// 库位置一旦可配，CLI 与 Web 必须读**同一处**配置，才不会出现"命令行读 A、界面读 B"。
// 所以这里统一传配置路径、内部 Load，而不是让各调用点自己算。
// 配置读不到 / 写坏时安静回退到缺省落点 —— 库位置不该因为配置有问题就变成空串。
func DictDBDirFor(cfgPath string) string {
	if strings.TrimSpace(cfgPath) != "" {
		if r, err := Load(cfgPath); err == nil {
			if d := strings.TrimSpace(r.Dict.Dir); d != "" {
				return AbsPath(d)
			}
		}
	}
	return LocationsAt(cfgPath).DictDBDir()
}

// DictDBDirForEnv 某环境实际使用的库目录：环境 dictDir > 顶层 dict.dir > <数据目录>/dict。
//
// 只有目录能按环境覆盖，**文件名始终是 <环境>.db** ——「每环境一个任意文件路径」会让
// 命令行、Web、查询三边可能指向不同的库，而"读到另一个环境的库"与"数据不存在"
// 在输出上完全一样，排查时无从下手。
func DictDBDirForEnv(cfgPath, env string) string {
	if strings.TrimSpace(cfgPath) != "" && strings.TrimSpace(env) != "" {
		if r, err := Load(cfgPath); err == nil {
			for i := range r.Hosts.SSHs {
				if r.Hosts.SSHs[i].Name != env {
					continue
				}
				if d := strings.TrimSpace(r.Hosts.SSHs[i].DictDir); d != "" {
					return AbsPath(d)
				}
				break
			}
			if d := strings.TrimSpace(r.Dict.Dir); d != "" {
				return AbsPath(d)
			}
		}
	}
	return LocationsAt(cfgPath).DictDBDir()
}

// DictDBPathForEnv 某环境的本地字典库文件（读配置，含环境级覆盖）。
func DictDBPathForEnv(cfgPath, env string) string {
	return DictDBPathIn(DictDBDirForEnv(cfgPath, env), env)
}

// DictDBCandidatesFor 某环境本地库的候选：-d/--db → 该环境的库文件
// （环境 dictDir > 顶层 dict.dir > <数据目录>/dict 下的 <环境>.db）。
//
// 这里**必须**走 DictDBPathForEnv 而不是 DictDBPath：同步/状态/Web 写的都是前者，
// 查询若用后者，用户把 dict.dir 改到新目录之后就是"同步写 A、查询读 B" —— 表现成
// "本地字典库不存在"，或者在旧位置还留着旧库时静默读到过期数据。
//
// **不认 TDICT_DB**：那是旧版“一个共享库”的全局开关，留着它会让一个环境变量
// 悄悄覆盖所有环境（隔离失效）。点名了 local:<环境> 就只看这个环境的文件；
// 确实要打开旧共享库，用 -d/--db 或裸 local + TDICT_DB。
func (l *Locations) DictDBCandidatesFor(env, flagPath string) []string {
	var out []string
	if p := AbsPath(flagPath); p != "" {
		out = append(out, p)
	}
	if p := DictDBPathForEnv(l.ConfigPath, env); p != "" {
		out = append(out, p)
	}
	return out
}

// ResolveDictDBFor 按环境定位本地库：返回第一个存在的候选；都缺时报错并列出尝试过的路径。
func (l *Locations) ResolveDictDBFor(env, flagPath string) (string, []string, error) {
	var tried []string
	for _, p := range l.DictDBCandidatesFor(env, flagPath) {
		abs, _ := filepath.Abs(p)
		if _, err := os.Stat(abs); err == nil {
			return abs, tried, nil
		}
		tried = append(tried, abs)
	}
	return "", tried, errLocalDBNotFound
}
