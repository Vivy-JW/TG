package outline

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"tt/internal/cli/common"
	"tt/internal/config"
	"tt/internal/mirror"
	"tt/internal/output"
)

type sourceOpts struct {
	File string
	Dir  string
	Env  string
}

type sourceInputs struct {
	Path    string // 本地绝对路径，或服务器上的完整路径
	Mode    string // file | mirror | remote
	Root    string // mirror 的最终环境目录 / remote 的 $TOP
	Env     string
	Content []byte // Mode=remote 时是已读到的源码；本地模式为 nil
}

// resolveSource 决定这次读哪份 .4gl：
//
//	本地已有这个文件           → 直接读
//	--dir <目录>               → 在该目录（可以是镜像目录）里按名字找
//	--at/--env 有镜像          → 在 mirror.Dir(环境) 里按名字找
//	--at/--env 没有镜像/没命中 → 直接连服务器读 $TOP/<模块>/4gl/<名>
//
// 为什么要最后那条直读：GP 的一个 $TOP 下有 1.4 万个 4gl（约 700 MB），
// 全量镜像要十几分钟；看一个大纲只需要一个文件，没必要先镜像。
// 同名文件命中多个时必须报 FILE_AMBIGUOUS，不能替调用方挑一个。
func resolveSource(opts sourceOpts) (*sourceInputs, error) {
	file := strings.TrimSpace(opts.File)
	dir := strings.TrimSpace(opts.Dir)
	env := strings.TrimSpace(opts.Env)
	if file == "" {
		return nil, usage("需要 .4gl 文件；示例：tg outline p_zta.4gl --at 22主机")
	}
	if !strings.EqualFold(filepath.Ext(file), ".4gl") {
		return nil, usage("只接受 .4gl 文件（当前 %q）", file)
	}
	if dir != "" && env != "" {
		return nil, usage("--dir 与 --at/--env 不能同时给：选一个数据源")
	}

	// 显式存在的本地文件优先。
	if src, ok, err := localFile(file); ok || err != nil {
		return src, err
	}

	if dir != "" {
		abs, err := filepath.Abs(dir)
		if err != nil {
			return nil, usage("解析 --dir 失败: %v", err)
		}
		src, ok, err := sourceInDir(abs, "", file)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, notFound("目录 %s 里没有找到 %s", abs, file).
				WithHint("--dir 只在这个目录里找；纯文件名会扫整棵目录树")
		}
		return src, nil
	}

	if env == "" {
		return nil, notFound("4GL 文件不存在: %s", file).
			WithHint("确认路径；或在环境镜像里/服务器上找，加 --at <环境> 或 --dir <目录>")
	}

	e, err := envNamed(env)
	if err != nil {
		return nil, err
	}
	// 先看本地镜像（有就快），没有再连服务器。
	if root := mirror.Dir(*e); root != "" {
		if st, statErr := os.Stat(root); statErr == nil && st.IsDir() {
			if src, ok, err := sourceInDir(root, env, file); ok || err != nil {
				return src, err
			}
		}
	}
	return remoteSource(*e, file)
}

func localFile(file string) (*sourceInputs, bool, error) {
	st, err := os.Stat(file)
	if err != nil {
		return nil, false, nil
	}
	if st.IsDir() {
		return nil, false, usage("位置参数是目录，不是 .4gl 文件: %s", file)
	}
	abs, err := filepath.Abs(file)
	if err != nil {
		return nil, false, usage("解析文件路径失败: %v", err)
	}
	return &sourceInputs{Path: abs, Mode: "file"}, true, nil
}

// sourceInDir 在 root 里找 file：带路径的相对参数按原路径找，纯文件名扫全树。
// 返回 ok=false 表示没找到（交给调用方决定下一步：镜像没命中就去服务器读）。
func sourceInDir(root, env, file string) (*sourceInputs, bool, error) {
	// 带路径的相对参数先按原相对路径找；不再用 basename 到处搜，
	// 因为“a/b/c.4gl”本来就是在点名一个位置。
	if filepath.IsAbs(file) || strings.ContainsAny(file, `/\`) {
		if filepath.IsAbs(file) {
			return nil, false, nil
		}
		cand := filepath.Join(root, file)
		if st, err := os.Stat(cand); err == nil && !st.IsDir() {
			return mirrorSource(root, env, cand), true, nil
		}
		return nil, false, nil
	}
	found, err := findMirrorFile(root, file)
	if err != nil {
		return nil, false, err
	}
	if found == "" {
		return nil, false, nil
	}
	return mirrorSource(root, env, found), true, nil
}

func envNamed(name string) (*config.NamedSsh, error) {
	cfgPath, err := common.ResolveConfig(false)
	if err != nil {
		return nil, output.Errorf(output.CodeConfigBroken, 2, "%v", err)
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, output.Errorf(output.CodeConfigBroken, 2, "%v", err)
	}
	e := cfg.Hosts.ByName(name)
	if e == nil {
		return nil, output.Errorf(output.CodeEnvUnknown, 2, "未找到环境 %q（tg env list 看全部）", name)
	}
	return e, nil
}

func mirrorSource(root, env, path string) *sourceInputs {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = filepath.Clean(path)
	}
	return &sourceInputs{Path: abs, Mode: "mirror", Root: root, Env: env}
}

// findMirrorFile 在镜像里按文件名找；找不到返回 ("", nil)，交给上层去服务器读。
func findMirrorFile(root, name string) (string, error) {
	var hits []string
	walkErr := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if p != root && transientDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.EqualFold(d.Name(), name) {
			return nil
		}
		hits = append(hits, p)
		return nil
	})
	if walkErr != nil {
		return "", output.Errorf(output.CodeLocalIO, 1, "扫描镜像目录失败: %v", walkErr)
	}
	sort.Strings(hits)
	if len(hits) == 0 {
		return "", nil
	}
	if len(hits) > 1 {
		return "", output.Errorf("FILE_AMBIGUOUS", 2,
			"文件 %s 命中 %d 个：%s；请给相对路径或缩小 --dir",
			name, len(hits), strings.Join(shortPaths(root, hits), "、"))
	}
	return hits[0], nil
}

func transientDir(name string) bool {
	return strings.HasPrefix(name, ".staging-") ||
		strings.HasPrefix(name, ".old-") ||
		strings.HasPrefix(name, ".tmp")
}

func shortPaths(root string, paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		if rel, err := filepath.Rel(root, p); err == nil && !strings.HasPrefix(rel, "..") {
			out = append(out, filepath.ToSlash(rel))
		} else {
			out = append(out, p)
		}
	}
	return out
}

func sourceMap(src *sourceInputs) map[string]any {
	m := map[string]any{"mode": src.Mode}
	if src.Env != "" {
		m["env"] = src.Env
	}
	if src.Root != "" {
		m["root"] = src.Root
	}
	return m
}

func usage(format string, a ...any) *output.Error {
	return output.Errorf(output.CodeUsage, 2, format, a...)
}

func notFound(format string, a ...any) *output.Error {
	return output.Errorf(output.CodeNotFound, 4, format, a...)
}
