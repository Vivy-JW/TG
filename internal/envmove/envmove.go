// Package envmove 把「环境改名」里按名字落盘的东西一起搬走。
//
// 环境名是跨文件的持久键：dict/<环境>.db、ents/<环境>.json、srccache/<环境>/、
// execlog/<环境>/、mirror.dir/<环境>/（tt file mirror 的源码镜像）。只改 config.json 里的 name，
// 这些目录就变成孤儿；查询随后报「本地库不存在」，而它与「数据不存在」在输出上
// 无法区分。所以改名必须是一次迁移：
//
//	plan → 搬文件 → 改配置（config.RenameHost）→ 失败则回滚
//
// 先搬文件后改配置：配置写失败可以回滚，配置写成功却发现文件搬不动就回不去了。
package envmove

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"tg/internal/config"
	"tg/internal/entdir"
)

// Move 一份要搬的落点（文件或整个目录）。
type Move struct {
	Kind string `json:"kind"` // dict / dict-bak / ents / srccache / execlog / mirror
	From string `json:"from"`
	To   string `json:"to"`
}

// pair 是 Plan 内部的候选落点（stat 之后才变成 Move）。
type pair struct{ kind, from, to string }

// Plan 计算 from → to 需要搬哪些落点。只读，不落盘。
//
// 路径必须与**实际读写点**用同一套解析：
//   - dict：环境 dictDir > 顶层 dict.dir > <数据目录>/dict（config.DictDBDirForEnv）；
//   - ents / srccache：envdir.EnvSeg；
//   - mirror：环境自己的 mirrorDir 是绝对路径、与名字无关，跳过；
//     否则把顶层 mirror.dir/<旧名> 一起搬到 <新名>（`tt file mirror` 的源码镜像）。
//
// 源不存在就跳过（没同步过字典、没查过企业、没调试过都很正常）；
// 目标已存在则整体失败 —— 覆盖一份别人的库比报错危险得多。
func Plan(cfgPath, from, to, host, zone string) ([]Move, error) {
	from, to = strings.TrimSpace(from), strings.TrimSpace(to)
	if from == "" || to == "" {
		return nil, fmt.Errorf("环境名不能为空")
	}
	if from == to {
		return nil, fmt.Errorf("新旧环境名相同：%q", from)
	}
	if err := config.ValidateEnvName(to); err != nil {
		return nil, err
	}
	dataDir := config.LocationsAt(cfgPath).DataDir()
	if dataDir == "" {
		return nil, fmt.Errorf("无法定位数据目录（配置路径为空）")
	}
	root, err := config.Load(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置失败：%w", err)
	}

	segFrom := entdir.EnvSeg(from, host, zone)
	segTo := entdir.EnvSeg(to, host, zone)

	var pairs []pair
	// dict：必须走 DictDBDirForEnv，否则用户把 dict.dir / 环境 dictDir 配到别处时，
	// 这里会去 <数据目录>/dict 找一个不存在的文件，改名后库仍挂在旧名下。
	dictDir := config.DictDBDirForEnv(cfgPath, from)
	pairs = append(pairs,
		pair{"dict", config.DictDBPathIn(dictDir, from), config.DictDBPathIn(dictDir, to)},
		pair{"dict-bak", config.DictDBPathIn(dictDir, from) + ".bak", config.DictDBPathIn(dictDir, to) + ".bak"},
		pair{"ents", entdir.Path(dataDir, segFrom), entdir.Path(dataDir, segTo)},
		pair{"srccache", filepath.Join(dataDir, "srccache", segFrom), filepath.Join(dataDir, "srccache", segTo)},
		// execlog 与 srccache 同源（都是 <dataDir>/<种类>/<环境段>/[runNN/]）：
		// 调试跑过命令的大输出副本按环境段落盘，不搬就走成孤儿目录。它只堆磁盘、
		// 不影响查询正确性，但既然目标是"按名字落盘的东西都跟着走"，就不该落下它。
		pair{"execlog", filepath.Join(dataDir, "execlog", segFrom), filepath.Join(dataDir, "execlog", segTo)},
	)

	// mirror：环境自配 mirrorDir 时是绝对路径、与名字无关，不搬；否则
	// `<mirror.dir>/<旧名>` → `<mirror.dir>/<新名>`。
	if e := root.Hosts.ByName(from); e != nil && strings.TrimSpace(e.MirrorDir) == "" {
		if rootDir := strings.TrimSpace(root.Mirror.Dir); rootDir != "" {
			base := config.AbsPath(rootDir)
			if base != "" {
				pairs = append(pairs, pair{"mirror", filepath.Join(base, from), filepath.Join(base, to)})
			}
		}
	}

	out := make([]Move, 0, len(pairs))
	for _, p := range pairs {
		if p.from == "" || p.to == "" || p.from == p.to {
			continue // 环境名里只有路径非法字符被替换时，新旧指向同一份
		}
		if _, err := os.Stat(p.from); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("检查 %s 失败：%w", p.from, err)
		}
		if _, err := os.Stat(p.to); err == nil {
			return nil, fmt.Errorf("目标已存在，拒绝覆盖：%s（先确认它是不是另一个环境的数据）", p.to)
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("检查 %s 失败：%w", p.to, err)
		}
		out = append(out, Move{Kind: p.kind, From: p.from, To: p.to})
	}
	return out, nil
}

// Execute 按计划搬迁。任一失败即回滚已搬的部分，不留半套。
// 企业快照搬完后顺手把文件内的 env 字段改成新名（它只用于展示，但留着旧名会误导排查）。
// oldName 用于回滚时把快照里的 env 改回去。
func Execute(moves []Move, newName, oldName string) error {
	done := make([]Move, 0, len(moves))
	for _, m := range moves {
		if err := renameWithRetry(m.From, m.To); err != nil {
			uerr := undo(done, oldName)
			return mergeErr(fmt.Errorf("搬 %s 失败：%w", m.From, renameHint(err)), uerr)
		}
		done = append(done, m)
		if m.Kind == "ents" && newName != "" {
			if err := SetSnapshotName(m.To, newName); err != nil {
				uerr := undo(done, oldName)
				return mergeErr(fmt.Errorf("更新企业快照 %s 失败：%w", m.To, err), uerr)
			}
		}
	}
	return nil
}

// Undo 反向搬（配置写失败时回滚用），并把企业快照里的 env 字段改回旧名。
// 返回回滚过程中的错误（可能有多个），由调用方决定怎么报。
func Undo(moves []Move, oldName string) error {
	return undo(moves, oldName)
}

func undo(done []Move, oldName string) error {
	var errs []error
	for i := len(done) - 1; i >= 0; i-- {
		m := done[i]
		if err := renameWithRetry(m.To, m.From); err != nil {
			errs = append(errs, fmt.Errorf("回滚 %s → %s 失败：%w", m.To, m.From, err))
			continue
		}
		if m.Kind == "ents" && oldName != "" {
			if err := SetSnapshotName(m.From, oldName); err != nil {
				errs = append(errs, fmt.Errorf("回滚企业快照 %s 失败：%w", m.From, err))
			}
		}
	}
	return errors.Join(errs...)
}

// renameWithRetry 在 Windows 上重试“文件暂时被占用”。
//
// SQLite / 杀软 / 索引器在 go-ora 或查询刚结束时可能还握着瞬时句柄，
// os.Rename 会报 Access is denied；退几十毫秒通常就好了。真正被常驻服务
// 锁住时仍然失败，最终由 renameHint 给出“先停服务”的建议。
func renameWithRetry(from, to string) error {
	var err error
	for i := 0; i < 6; i++ {
		err = os.Rename(from, to)
		if err == nil {
			return nil
		}
		if !retryableRename(err) {
			return err
		}
		time.Sleep(time.Duration(i+1) * 50 * time.Millisecond)
	}
	return err
}

func retryableRename(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, fs.ErrPermission) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "being used") ||
		strings.Contains(msg, "another process") ||
		strings.Contains(msg, "正由另一进程使用") ||
		strings.Contains(msg, "另一个程序")
}

// renameHint 给常见的“文件被占用 / 跨卷”失败补一句可执行的建议。
func renameHint(err error) error {
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "access is denied"),
		strings.Contains(msg, "being used"),
		strings.Contains(msg, "another process"),
		strings.Contains(msg, "正由另一进程使用"),
		strings.Contains(msg, "另一个程序"):
		return fmt.Errorf("%w（文件可能正被 tg serve / 查询进程占用：先 tg serve --stop 或关闭占用它的程序再试）", err)
	case strings.Contains(msg, "cross-device"), strings.Contains(msg, "invalid cross-device"),
		strings.Contains(msg, "different disk drive"):
		return fmt.Errorf("%w（落点在不同磁盘上，无法 rename；先手工搬到同一卷再改名，或把 dictDir/mirrorDir 指到同一卷）", err)
	}
	return err
}

func mergeErr(primary, undoErr error) error {
	if undoErr == nil {
		return primary
	}
	return fmt.Errorf("%w；同时回滚落点失败：%v", primary, undoErr)
}

// SetSnapshotName 把企业快照里的 env 字段改成新环境名（新名由调用方给）。
// 迁移后调用一次：文件名对了、内容还写着旧名，排查时会被带偏。
func SetSnapshotName(snapshotPath, name string) error {
	return setSnapshotEnv(snapshotPath, name)
}

func setSnapshotEnv(snapshotPath, name string) error {
	if name == "" {
		name = strings.TrimSuffix(filepath.Base(snapshotPath), ".json")
	}
	b, err := os.ReadFile(snapshotPath)
	if err != nil {
		return err
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	m["env"] = name
	out, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return config.AtomicWrite(snapshotPath, out)
}
