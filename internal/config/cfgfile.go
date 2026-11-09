package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// writeMu 串行化同一进程内的配置“读-改-写”。
//
// T100 的 /api/hosts 与 GP 的 /api/gp/env 都改同一份 config.json，而且两边的
// 设置页都有 500ms 自动保存；各写各的会造成后写者用旧快照覆盖前者的修改
// （典型：一边刚保存，另一边的 GP/T100 环境被整段写回旧内容）。
//
// 这里用进程级互斥：统一服务里所有写入口都在同一进程，已足够；
// 若以后允许多个 serve 进程同时写同一份配置，需要再补跨进程文件锁。
var writeMu sync.Mutex

// Open 读取并解析配置文件；文件不存在返回空配置（不是错误 —— 首次运行没有文件），
// 空文件视为空配置，非法 JSON 报错。
//
// 返回的 map 保留所有未知顶层键：调用方只该改自己那一节，其余原样带走。
func Open(path string) (map[string]any, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, nil
		}
		return nil, fmt.Errorf("读取配置文件失败(%s): %w", path, err)
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return map[string]any{}, nil // 空文件（或只有空白）→ 空配置
	}
	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil, fmt.Errorf("解析配置文件失败(%s): %v", path, err)
	}
	if root == nil {
		root = map[string]any{} // 字面量 null → 空配置
	}
	return root, nil
}

// Save 原子写回：MarshalIndent + 同目录临时文件 + fsync + 重命名，
// 失败不留半截文件，也不会让读者看到只写了一半的内容。
//
// 配置目录不存在时自动创建 —— 缺省位置在 %APPDATA%\T100\tt\ 下，
// 新机器上首次保存时该目录还没有。
func Save(path string, root map[string]any) error {
	writeMu.Lock()
	defer writeMu.Unlock()
	return save(path, root)
}

// save 是不带锁的内部实现；Edit 已经持有 writeMu，不能再调 Save，否则自锁。
func save(path string, root map[string]any) error {
	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}
	out = append(out, '\n')
	return AtomicWrite(path, out)
}

// AtomicWrite 同目录临时文件 → 写入 → fsync → 保留原权限位 → 重命名替换。
// 与 TDev internal/store/atomic.go 同一套语义，合并后只有这一份。
func AtomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("创建配置目录失败(%s): %w", dir, err)
		}
	}

	// 已存在的文件按原权限位写，新建的用 0600 —— 配置里有明文口令。
	mode := os.FileMode(0o600)
	if fi, err := os.Stat(path); err == nil {
		mode = fi.Mode().Perm()
	}

	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp*")
	if err != nil {
		if isAccessDenied(err) {
			return fmt.Errorf("创建临时文件失败(%s): %w\n"+
				"  这个目录当前不允许写入。配置由运行中的 tt serve 保存 —— "+
				"若它是从某个受限环境(沙箱/只读挂载)启动的,请换成普通终端重新启动再试。", dir, err)
		}
		return fmt.Errorf("创建临时文件失败: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // 成功路径下已被 Rename 掉，这里是失败清理

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("写入配置文件失败: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("同步配置文件失败: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("关闭配置文件失败: %w", err)
	}
	if err := os.Chmod(tmpName, mode); err != nil {
		return fmt.Errorf("设置配置文件权限失败: %w", err)
	}
	// 用重命名替换目标。Windows 上这一步会因为**目标文件暂时被别的手占用**
	// (杀软扫描/索引器/另一个读者)而报 Access is denied —— 那是瞬时的,
	// 所以给几次短重试,而不是让用户的"保存"直接失败。
	if err := renameWithRetry(tmpName, path); err != nil {
		return fmt.Errorf("更新配置文件失败: %w", err)
	}
	return nil
}

// renameAttempts / renameBackoff 是重命名替换的重试策略。
//
// 为什么值得重试:实测(2026-10-15)在 Windows 上,只要目标 config.json 那一刻
// 被另一个句柄占着,os.Rename 就报 `Access is denied` —— 而占用往往是杀软/索引器
// 的瞬时扫描,几十毫秒后就没了。没有重试的话,用户看到的是"删除环境失败:
// 更新配置文件失败: Access is denied",而他什么都没做错。
//
// 次数与间隔刻意小:配置保存是交互动作,不能让人干等;真被长期占用时
// 仍要如实报错(那说明有东西一直占着,重试再多次也没用)。
const (
	renameAttempts = 5
	renameBackoff  = 40 * time.Millisecond
)

func renameWithRetry(from, to string) error {
	var err error
	for i := 0; i < renameAttempts; i++ {
		if err = os.Rename(from, to); err == nil {
			return nil
		}
		if !isAccessDenied(err) {
			return err // 不是"被占用"这一类:重试没有意义
		}
		time.Sleep(renameBackoff * time.Duration(i+1)) // 递增退避
	}
	return err
}

// isAccessDenied 判断错误是不是"拒绝访问"这一类。
//
// 基础判据只用标准库(errors.Is(err, fs.ErrPermission))—— 它跨平台成立,
// 而 Go 在 Windows 上会把 ERROR_ACCESS_DENIED 映射到它。平台特有的补充判据
// (syscall.Errno 直出的情况)放在 isAccessDeniedPlatform 里,按平台分文件。
func isAccessDenied(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, fs.ErrPermission) {
		return true
	}
	return isAccessDeniedPlatform(err)
}

// Edit 统一修改入口：Open → validate(root)（可空）→ mutate(root) → Save。
// validate/mutate 任一失败都不落盘。
//
// config.json 的一切修改都应走本函数，避免各调用方重复"读-改-写"样板
// 与错误语义漂移。注意 mutate 只改自己那一节，Open 保留的其他顶层键会原样写回。
func Edit(path string, validate func(root map[string]any) error, mutate func(root map[string]any) error) error {
	// 锁住整个 Open→mutate→Save，而不是只锁 Save：只锁写会留下
	// “两边都读到旧快照、再先后写”的丢更新窗口。
	writeMu.Lock()
	defer writeMu.Unlock()

	root, err := Open(path)
	if err != nil {
		return err
	}
	if validate != nil {
		if err := validate(root); err != nil {
			return err
		}
	}
	if mutate != nil {
		if err := mutate(root); err != nil {
			return err
		}
	}
	return save(path, root)
}

// NewSkeleton 返回一份最小可用的新配置：空环境清单 + 缺省监听地址。
// 首次运行用它落一个骨架，用户随后在配置页里填。
func NewSkeleton() map[string]any {
	return map[string]any{
		"schemaVersion": SchemaVersion,
		"listen":        DefaultListen,
		"hosts": map[string]any{
			// 只有 sshs:默认环境那个键已整体移除,新骨架里不再写它。
			"sshs": []any{},
		},
		"query": map[string]any{"source": "auto"},
		// 第三方技能登记表：空表是正常态（不下发任何第三方技能）。
		// 官方技能不在这里 —— 它们随包走，见 internal/skill 的 Origin 说明。
		"skills": map[string]any{"externals": []any{}},
		// 骨架里没有环境，所以也没有工作区 —— 工作区按环境配（hosts.sshs[].workspace），
		// 而它是引擎的必填项：配错等于拿别的客户的表单当草稿纸（见 NamedSsh.Workspace）。
		// 设计器目录也不在这里 —— 它由发行包自带，见 engine/src/Designer/Bootstrap.cs 的 Install。
		"tzs": map[string]any{
			"serverExe": "",
		},
	}
}

// EnsureExists 在配置不存在时落一份骨架（已存在则什么都不做）。
//
// 服务启动路径需要它：首次运行时还没有任何环境，但配置页要有文件可编辑。
// 沿用与 Save 相同的原子写，所以不会留下半截文件。
func EnsureExists(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	return Save(path, NewSkeleton())
}

// Section 返回 root 的顶层对象节；缺失或非对象返回统一错误。
// key 为节名，如 "hosts" / "debug"。
func Section(root map[string]any, key string) (map[string]any, error) {
	s, _ := root[key].(map[string]any)
	if s == nil {
		return nil, fmt.Errorf("config.json 缺少 %q 配置节", key)
	}
	return s, nil
}

// SectionOrEmpty 同 Section，但缺失时返回一个已挂回 root 的空节。
// 写路径用它，读路径用 Section。
func SectionOrEmpty(root map[string]any, key string) map[string]any {
	if s, ok := root[key].(map[string]any); ok && s != nil {
		return s
	}
	s := map[string]any{}
	root[key] = s
	return s
}
