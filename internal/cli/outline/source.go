package outline

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"tt/internal/cli/common"
	"tt/internal/config"
	"tt/internal/output"
)

type sourceOpts struct {
	File string
	Dir  string
	Env  string
}

type sourceInputs struct {
	Path string // 绝对路径
	Mode string // file | mirror
	Root string // mirror 的最终环境目录
	Env  string
}

// resolveSource 决定这次读哪个 .4gl：
//
//	显式存在的文件 > --dir 镜像目录 > --at/--env 对应的镜像目录
//
// --dir 是最终环境目录（例如 D:\TT-Work\109tst），不是全局镜像根。
// 同名文件命中多个时必须报 FILE_AMBIGUOUS，不能替调用方挑一个。
func resolveSource(opts sourceOpts) (*sourceInputs, error) {
	file := strings.TrimSpace(opts.File)
	dir := strings.TrimSpace(opts.Dir)
	env := strings.TrimSpace(opts.Env)
	if file == "" {
		return nil, usage("需要 .4gl 文件；示例：tt outline aapt320.4gl --at 109tst")
	}
	if !strings.EqualFold(filepath.Ext(file), ".4gl") {
		return nil, usage("只接受 .4gl 文件（当前 %q）", file)
	}
	if dir != "" && env != "" {
		return nil, usage("--dir 与 --at/--env 不能同时给：选一个数据源")
	}

	// 显式存在的文件优先；--dir/--at 只负责按名字找镜像里的文件。
	if st, err := os.Stat(file); err == nil {
		if st.IsDir() {
			return nil, usage("位置参数是目录，不是 .4gl 文件: %s", file)
		}
		abs, err := filepath.Abs(file)
		if err != nil {
			return nil, usage("解析文件路径失败: %v", err)
		}
		return &sourceInputs{Path: abs, Mode: "file"}, nil
	}

	root, err := resolveRoot(dir, env)
	if err != nil {
		return nil, err
	}
	if root == "" {
		return nil, notFound("4GL 文件不存在: %s", file).
			WithHint("确认路径；如果文件在环境镜像里，加 --at <环境> 或 --dir <目录>")
	}
	if err := requireDir(root); err != nil {
		return nil, err
	}

	// 带路径的相对参数先按原相对路径找；不再用 basename 到处搜，
	// 因为“a/b/c.4gl”本来就是在点名一个位置。
	if filepath.IsAbs(file) || strings.ContainsAny(file, `/\`) {
		if filepath.IsAbs(file) {
			return nil, notFound("4GL 文件不存在: %s", file)
		}
		cand := filepath.Join(root, file)
		if st, err := os.Stat(cand); err == nil && !st.IsDir() {
			return mirrorSource(root, env, cand), nil
		}
		return nil, notFound("4GL 文件不存在: %s（镜像根: %s）", file, root).
			WithHint("确认相对路径；用 --dir 指向该环境的最终镜像目录")
	}

	found, err := findMirrorFile(root, file)
	if err != nil {
		return nil, err
	}
	return mirrorSource(root, env, found), nil
}

func resolveRoot(dir, env string) (root string, err error) {
	if dir != "" {
		abs, err := filepath.Abs(dir)
		if err != nil {
			return "", usage("解析 --dir 失败: %v", err)
		}
		return abs, nil
	}
	if env == "" {
		return "", nil
	}
	cfgPath, err := common.ResolveConfig(false)
	if err != nil {
		return "", output.Errorf(output.CodeConfigBroken, 2, "%v", err)
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return "", output.Errorf(output.CodeConfigBroken, 2, "%v", err)
	}
	e := cfg.Hosts.ByName(env)
	if e == nil {
		return "", output.Errorf(output.CodeEnvUnknown, 2, "未找到环境 %q（tt env list 看全部）", env)
	}
	root = e.MirrorPath(cfg.Mirror.Dir)
	if strings.TrimSpace(root) == "" {
		return "", usage("环境 %s 的镜像目录未设置。请先执行: tt file mirror dir <目录> --at %s", env, env).
			WithHint("或直接给 --dir <该环境的镜像目录>")
	}
	return root, nil
}

func requireDir(root string) error {
	st, err := os.Stat(root)
	if err != nil || !st.IsDir() {
		return notFound("镜像目录不存在: %s", root).
			WithHint("先拉取源码镜像：tt file mirror pull --profile source --at <环境>")
	}
	return nil
}

func mirrorSource(root, env, path string) *sourceInputs {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = filepath.Clean(path)
	}
	return &sourceInputs{Path: abs, Mode: "mirror", Root: root, Env: env}
}

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
	hits = sortedHits(hits)
	if len(hits) == 0 {
		return "", notFound("镜像里没有找到 %s（根: %s）", name, root).
			WithHint("确认文件名；新镜像先跑 tt file mirror pull --profile source --at <环境>")
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
