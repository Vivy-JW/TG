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
	"tt/internal/dict"
	"tt/internal/mirror"
	"tt/internal/output"
)

// sourceInputs 一次 interface 查询实际要读的东西。
type sourceInputs struct {
	Form        string
	Module      string
	Env         string
	Mode        string // mirror | files | remote
	Root        string // --dir 指向的最终环境镜像目录（mirror 模式）/ $TOP（remote 模式）
	Form42F     string
	Form42FData []byte // Mode=remote 时是已读到的 42f；本地模式为 nil
	LabelSource string // gae_file(local|live:…) / off / none
}

type sourceOpts struct {
	Form   string
	Module string
	Dir    string
	FileF  string
}

// resolveSources 决定这次从哪读 .42f。
//
// 优先级：显式 --42f > --dir > --at 的环境镜像 > --at 直读服务器。
// 最后那条是给 GP 用的：一个 $TOP 下的 42f 有几百个且散在各模块，
// 没拉镜像时按 <模块>/42f/<表单>.42f 直接读一个文件更快。
func resolveSources(opts sourceOpts) (*sourceInputs, error) {
	opts.FileF = strings.TrimSpace(opts.FileF)
	opts.Dir = strings.TrimSpace(opts.Dir)
	form := strings.TrimSpace(opts.Form)

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
				WithHint("确认路径；或给 --at <环境> 直接从服务器读")
		}
		src := &sourceInputs{
			Form: form, Mode: "files", Module: strings.TrimSpace(opts.Module),
			Form42F: f,
		}
		if src.Module == "" {
			src.Module = moduleFromPath(f)
		}
		return src, nil
	}

	// --dir：只在这个目录里找，找不到就是找不到。
	if opts.Dir != "" {
		abs, err := filepath.Abs(opts.Dir)
		if err != nil {
			return nil, usage("解析 --dir 失败: %v", err)
		}
		if st, statErr := os.Stat(abs); statErr != nil || !st.IsDir() {
			return nil, notFound("FORM_NOT_FOUND", 4, "目录不存在: %s", abs)
		}
		if form == "" {
			return nil, usage("需要表单名：tg interface show <form> --dir <目录>")
		}
		f, mod, err := find42F(abs, form, strings.TrimSpace(opts.Module))
		if err != nil {
			return nil, err
		}
		return &sourceInputs{Form: strings.TrimSuffix(filepath.Base(f), ".42f"), Module: mod, Mode: "mirror", Root: abs, Form42F: f}, nil
	}

	env := strings.TrimSpace(common.Env)
	if env == "" {
		return nil, usage("需要点名数据源：--at <环境>（用该环境的镜像，没有就连服务器读），或 --dir <目录>，或 --42f <文件>").
			WithHint("不知道有哪些环境：tg interface --at-list")
	}
	e, err := envByName(env)
	if err != nil {
		return nil, err
	}
	if form == "" {
		return nil, usage("需要表单名：tg interface show <form> --at <环境>")
	}

	// 有镜像就先在镜像里找（快）；镜像里没有就直读服务器。
	if root := mirror.Dir(*e); root != "" {
		if st, statErr := os.Stat(root); statErr == nil && st.IsDir() {
			if f, mod, findErr := find42F(root, form, strings.TrimSpace(opts.Module)); findErr == nil {
				return &sourceInputs{Form: strings.TrimSuffix(filepath.Base(f), ".42f"), Module: mod, Env: env, Mode: "mirror", Root: root, Form42F: f}, nil
			}
		}
	}
	return remote42F(*e, form, strings.TrimSpace(opts.Module))
}

// remote42F 直接从服务器读 <模块>/42f/<表单>.42f。
// 两棵树都试：标准树在 $TOP，客制画面在 $CUST（common.ReadRemoteFile 处理）。
func remote42F(e config.NamedSsh, form, module string) (*sourceInputs, error) {
	base := form + ".42f"
	cands := make([]string, 0, 3)
	mod := strings.TrimSpace(module)
	if mod == "" {
		if m, ok := dict.LookupModule(e, form); ok && strings.TrimSpace(m) != "" {
			mod = strings.ToLower(strings.TrimSpace(m))
		}
	} else {
		mod = strings.ToLower(mod)
	}
	if mod != "" {
		cands = append(cands, mod+"/42f/"+base)
	}
	full, data, err := common.ReadRemoteFile(e, cands, base, common.RemoteReadLimit)
	if err != nil {
		return nil, err
	}
	return &sourceInputs{Form: form, Module: mod, Env: e.Name, Mode: "remote", Root: rootOf(e, full), Form42F: full, Form42FData: data}, nil
}

// rootOf 这个文件落在哪棵树（客制画面别把 Root 记成 $TOP）。
func rootOf(e config.NamedSsh, full string) string {
	cust := strings.TrimSuffix(strings.TrimSpace(e.CustRoot), "/")
	top := strings.TrimSuffix(strings.TrimSpace(e.Workspace), "/")
	if cust != "" && (full == cust || strings.HasPrefix(full, cust+"/")) {
		return cust
	}
	return top
}

func envByName(name string) (*config.NamedSsh, error) {
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
				WithHint("这份镜像没拉画面这一类。拉一次默认镜像即可：tt file mirror pull --at <环境>；或直接给 --42f <文件>")
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

func moduleFromPath(p string) string {
	slash := filepath.ToSlash(p)
	for _, marker := range []string{"/42f/"} {
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
