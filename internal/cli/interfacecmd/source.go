package interfacecmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"tt/internal/cli/common"
	"tt/internal/config"
	"tt/internal/output"
)

// sourceInputs 一次 interface 查询实际要读的两个文件。
type sourceInputs struct {
	Form       string
	Module     string
	Env        string
	Mode       string // mirror | files
	Root       string // --dir 指向的最终环境镜像目录（mirror 模式）
	Form42F    string
	Form42S    string
	LabelsSeen bool // 没给 42s 也允许，只是标签全 unresolved
}

type sourceOpts struct {
	Form   string
	Module string
	Dir    string
	FileF  string
	FileS  string
}

// resolveSources 决定这次从哪读 .42f/.42s。
//
// 优先级：显式 --42f/--42s > --dir > --at 对应的环境镜像目录。
// --dir 是“最终环境目录”（内含 erp/、com/），不是全局根；这样设置页里每个
// 环境单独指定目录的语义与 mirror 完全一致。
func resolveSources(opts sourceOpts) (*sourceInputs, error) {
	opts.FileF = strings.TrimSpace(opts.FileF)
	opts.FileS = strings.TrimSpace(opts.FileS)
	opts.Dir = strings.TrimSpace(opts.Dir)
	form := strings.TrimSpace(opts.Form)

	if opts.FileS != "" && opts.FileF == "" {
		return nil, usage("--42s 需要同时给 --42f（本命令不自动猜另一份文件）")
	}
	if opts.FileF != "" {
		if form == "" {
			form = strings.TrimSuffix(filepath.Base(opts.FileF), filepath.Ext(opts.FileF))
		}
		if form == "" {
			return nil, usage("无法从 --42f 推出表单名：请同时给 <form> 位置参数")
		}
		f, err := filepath.Abs(opts.FileF)
		if err != nil {
			return nil, usage("解析 --42f 路径失败: %v", err)
		}
		if _, err := os.Stat(f); err != nil {
			return nil, notFound("FORM_NOT_FOUND", 4, "42f 文件不存在: %s", f).
				WithHint("确认路径，或先用 tt file mirror pull --at <环境> 建立镜像")
		}
		src := &sourceInputs{
			Form: form, Mode: "files", Module: strings.TrimSpace(opts.Module),
			Form42F: f, LabelsSeen: opts.FileS != "",
		}
		if opts.FileS != "" {
			s, err := filepath.Abs(opts.FileS)
			if err != nil {
				return nil, usage("解析 --42s 路径失败: %v", err)
			}
			if _, err := os.Stat(s); err != nil {
				return nil, notFound("LABELS_INVALID", 1, "42s 文件不存在: %s", s).
					WithHint("标签文件缺失可以只给 --42f；此时页面/字段标签会标为 unresolved")
			}
			src.Form42S = s
		}
		if src.Module == "" {
			src.Module = moduleFromPath(f)
		}
		return src, nil
	}

	// 镜像模式
	root := opts.Dir
	env := strings.TrimSpace(common.Env)
	if root == "" {
		if env == "" {
			return nil, usage("需要点名数据源：--at <环境>（用该环境的镜像目录），或 --dir <目录>，或 --42f <文件>").
				WithHint("不知道有哪些环境：tt interface --at-list")
		}
		cfgPath, err := common.ResolveConfig(false)
		if err != nil {
			return nil, output.Errorf(output.CodeConfigBroken, 2, "%v", err)
		}
		rootCfg, err := config.Load(cfgPath)
		if err != nil {
			return nil, output.Errorf(output.CodeConfigBroken, 2, "%v", err)
		}
		e := rootCfg.Hosts.ByName(env)
		if e == nil {
			return nil, output.Errorf(output.CodeEnvUnknown, 2, "未找到环境 %q（tt env list 看全部）", env)
		}
		root = e.MirrorPath(rootCfg.Mirror.Dir)
		if root == "" {
			return nil, usage("环境 %s 的镜像目录未设置。请先执行: tt file mirror dir <目录> --at %s", env, env).
				WithHint("或直接给 --dir <该环境的镜像目录>")
		}
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, usage("解析镜像目录失败: %v", err)
	}
	if st, err := os.Stat(abs); err != nil || !st.IsDir() {
		return nil, notFound("FORM_NOT_FOUND", 4, "镜像目录不存在: %s", abs).
			WithHint("先拉取：tt file mirror pull --at <环境>")
	}
	form = strings.TrimSpace(form)
	if form == "" {
		return nil, usage("需要表单名：tt interface show <form> --at <环境>")
	}
	f, mod, err := find42F(abs, form, strings.TrimSpace(opts.Module))
	if err != nil {
		return nil, err
	}
	if mod != "" {
		form = strings.TrimSuffix(filepath.Base(f), ".42f")
	}
	s, err := find42S(abs, form, mod)
	if err != nil && !isNotFound(err) {
		return nil, err
	}
	src := &sourceInputs{
		Form: form, Module: mod, Env: env, Mode: "mirror", Root: abs,
		Form42F: f, Form42S: s, LabelsSeen: s != "",
	}
	return src, nil
}

// find42F 在环境镜像中找一个 <form>.42f；同名多模块时要求 --module。
//
// 找不到时分两种说：
//   - 目录里**一个 .42f 也没有** → 不是名字拼错，是这份镜像没拉过画面这一类；
//   - 目录里有 42f 但没有这个表单 → 名字/模块不对。
//
// 两者对应两种完全不同的动作，混成一句“表单不存在”会把人带到错方向（他会去改代码，
// 而实际只是镜像没拉全）。
func find42F(root, form, module string) (path, mod string, err error) {
	var hits []string
	sawAny42F := false
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if p != root && (strings.HasPrefix(name, ".staging-") || strings.HasPrefix(name, ".old-") || strings.HasPrefix(name, ".tmp")) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".42f") {
			return nil
		}
		sawAny42F = true
		if !strings.EqualFold(d.Name(), form+".42f") {
			return nil
		}
		m := moduleFromPath(p)
		if module != "" && !strings.EqualFold(m, module) {
			return nil
		}
		hits = append(hits, p)
		return nil
	})
	sort.Strings(hits)
	if len(hits) == 0 {
		if !sawAny42F {
			return "", "", notFound("FORM_NOT_FOUND", 4, "镜像里没有任何 .42f（镜像: %s）", root).
				WithHint("这份镜像没拉画面这一类。拉一次默认镜像即可（source 已含 42f/42s）：tt file mirror pull --at <环境>；或直接给 --42f <文件>")
		}
		return "", "", notFound("FORM_NOT_FOUND", 4, "没有找到 %s.42f（镜像: %s）", form, root).
			WithHint("镜像里有别的 42f，所以是名字/模块不对：确认表单名；同名多个模块时加 --module（镜像里有 42f 说明 profile 没问题）")
	}
	if len(hits) > 1 {
		return "", "", output.Errorf("FORM_AMBIGUOUS", 2,
			"表单 %s 命中 %d 个模块：%s；请加 --module 指定", form, len(hits), strings.Join(shortPaths(root, hits), "、"))
	}
	return hits[0], moduleFromPath(hits[0]), nil
}

// find42S 在同一个模块下找 .42s；42s 可以位于 42s/zh_CN 或更深层
// （服务器白名单同时覆盖这两种布局）。找不到不算错误，返回空。
func find42S(root, form, module string) (string, error) {
	var hits []string
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if p != root && (strings.HasPrefix(name, ".staging-") || strings.HasPrefix(name, ".old-")) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.EqualFold(d.Name(), form+".42s") {
			return nil
		}
		slash := filepath.ToSlash(p)
		if !strings.Contains(slash, "/42s/") || !strings.Contains(slash, "/zh_CN/") {
			return nil
		}
		if module != "" && !strings.EqualFold(moduleFromPath(p), module) {
			return nil
		}
		hits = append(hits, p)
		return nil
	})
	sort.Strings(hits)
	if len(hits) == 0 {
		return "", notFound("LABELS_MISSING", 4, "没有找到 %s.42s（镜像: %s）", form, root)
	}
	if len(hits) > 1 {
		return "", output.Errorf("LABELS_AMBIGUOUS", 2,
			"表单 %s 命中 %d 个 42s：%s；请用 --module 或 --dir 缩小范围", form, len(hits), strings.Join(shortPaths(root, hits), "、"))
	}
	return hits[0], nil
}

func moduleFromPath(p string) string {
	slash := filepath.ToSlash(p)
	for _, marker := range []string{"/42f/", "/42s/"} {
		if i := strings.Index(slash, marker); i >= 0 {
			parent := slash[:i]
			return filepath.Base(parent)
		}
	}
	return ""
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

func isNotFound(err error) bool {
	if oe, ok := err.(*output.Error); ok {
		return oe.Code == "LABELS_MISSING" || oe.Code == "NOT_FOUND"
	}
	return false
}

func usage(format string, a ...any) *output.Error {
	return output.Errorf(output.CodeUsage, 2, format, a...)
}

func notFound(code string, exit int, format string, a ...any) *output.Error {
	return output.Errorf(code, exit, format, a...)
}

func (e *sourceInputs) String() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("%s (%s)", e.Form, e.Mode)
}
