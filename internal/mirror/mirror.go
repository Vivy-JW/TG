// Package mirror 把 TOP GP 服务器上的源码与表单文件镜像到本地。
//
// 收三类：`*/4gl/*.4gl`（源码，tg outline 读）、`*/42f/*`（编译表单，tg interface 读）、
// `*.inc`（4GL include，grep/cat 读）。GP 的根是 $TOP（正式区）与 $CUST（客制区）两棵，
// 都在镜像范围里。
//
// 不收 42s：T100 的 42s 是"把标签编译进文件"，GP 没有这类文件 —— 它的表单标签在
// gae_file 里（由 tg dict db sync 落成本地 t_ui_text），属于字典线，不属于镜像线。
// 标签怎么取见 internal/dict 与 tg interface --labels。
//
// 与字典（internal/dict）分开成两个包，是因为两者的落点、增量规则、失败影响都不同：
// 字典换库失败只影响查询，镜像换目录失败会让调试器找不到源码。
package mirror

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/sftp"

	"tg/internal/config"
	"tg/internal/host"
)

// Dir 该环境的镜像目录：环境自己指定优先，否则 <默认根>/<环境名>。
//
// 默认根可以用 TG_GP_MIRROR_DIR 覆盖（多套环境放同一块盘时用）。
//
// 注意：这里不读 config.json 顶层 mirror.dir —— 需要那个兜底时用 DirFor。
func Dir(e config.NamedSsh) string { return DirFor("", e) }

// DirFor 同 Dir，但会先看 config.json 顶层 mirror.dir（tg file mirror dir 写的全局根）。
//
// 优先级：环境自己的 mirrorDir > 全局根/<环境名> > TG_GP_MIRROR_DIR/<环境名> > ~/.tg/gp-mirror/<环境名>。
// cfgPath 为空（不知道配置在哪）时跳过全局根这一层。
func DirFor(cfgPath string, e config.NamedSsh) string {
	if strings.TrimSpace(e.MirrorDir) != "" {
		return e.MirrorDir
	}
	if cfgPath != "" {
		if root, err := config.Load(cfgPath); err == nil {
			if d := strings.TrimSpace(root.Mirror.Dir); d != "" {
				return filepath.Join(d, e.Name)
			}
		}
	}
	base := os.Getenv("TG_GP_MIRROR_DIR")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".tg", "gp-mirror")
	}
	return filepath.Join(base, e.Name)
}

// Manifest 镜像根目录下 manifest.json 的内容：上一次拉取的规模与时间。
type Manifest struct {
	Files   int            `json:"files"`
	SizeMB  float64        `json:"sizeMB"`
	Kinds   map[string]int `json:"kinds,omitempty"`
	Source  string         `json:"source"`
	Updated string         `json:"at"`
}

// ReadManifest 读 <dir>/manifest.json；没有或坏了都返回 false。
func ReadManifest(dir string) (Manifest, bool) {
	b, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return Manifest{}, false
	}
	var m Manifest
	if json.Unmarshal(b, &m) != nil {
		return Manifest{}, false
	}
	return m, true
}

// WriteManifest 写 <dir>/manifest.json（不返回错误：清单只是给人看的）。
func WriteManifest(dir string, files int, sizeMB float64, source string, kinds map[string]int) {
	_ = os.MkdirAll(dir, 0o755)
	m := Manifest{Files: files, SizeMB: sizeMB, Kinds: kinds, Source: source, Updated: time.Now().Format("2006-01-02 15:04:05")}
	b, _ := json.MarshalIndent(m, "", "  ")
	_ = os.WriteFile(filepath.Join(dir, "manifest.json"), b, 0o644)
}

// Roots 一轮镜像要拉的两棵源码树：$TOP 与 $CUST。
//
// 返回的是"该环境配了的"那几棵；两棵都没配时返回空，由调用方报错。
func Roots(env config.NamedSsh) []struct{ Label, Path string } {
	var out []struct{ Label, Path string }
	for _, r := range []struct{ Label, Path string }{
		{"tiptop", env.Workspace},
		{"topcust", env.CustRoot},
	} {
		if r.Path != "" {
			out = append(out, r)
		}
	}
	return out
}

// Pull 连服务器，把两棵源码树里的 4gl / 42f / inc 拉进 Dir(env)。
//
// mode="full" 会顺手删掉服务器上已不存在的本地文件（与 T100 的镜像行为一致）；
// 其它值一律按增量处理，只补有差异的文件。
//
// 返回按树分组的统计（供 job.Result 直接回给前端），progress 用来报进度。
func Pull(env config.NamedSsh, mode string, progress func(done, total int, msg string)) (map[string]any, error) {
	if env.Workspace == "" && env.CustRoot == "" {
		return nil, fmt.Errorf("这个环境没配 $TOP / $CUST 源码根")
	}
	conn, err := host.Dial(config.SSHConfig{Host: env.Host, Port: env.Port, User: env.User, Password: env.Password})
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	sc, err := conn.SFTP()
	if err != nil {
		return nil, err
	}
	defer sc.Close()

	base := Dir(env)
	out := map[string]any{}
	doneBase, totalBase := 0, 0
	for _, src := range Roots(env) {
		dst := filepath.Join(base, src.Label)
		st, err := tree(conn, sc, src.Path, dst, mode, func(done, total int, msg string) {
			if progress != nil {
				progress(doneBase+done, totalBase+total, msg)
			}
		})
		if err != nil {
			return nil, fmt.Errorf("%s: %w", src.Label, err)
		}
		st["src"] = env.User + "@" + env.Host + ":" + src.Path
		st["dst"] = dst
		out[src.Label] = st
		doneBase += st["processed"].(int)
		totalBase += st["total"].(int)
	}
	n, sizeMB, kinds := localStats(base)
	WriteManifest(base, n, sizeMB, "mirror", kinds)
	return out, nil
}

// localStats 数一遍镜像目录里现有的镜像文件：总数、总大小、按类别。
//
// manifest 记的是"目录里现在有什么"，不是"这一轮下载了多少" —— 增量跑完常常
// 一个文件都没下载，记下载数会让 tg file mirror path 显示成 0 个 / 0 MB。
func localStats(base string) (int, float64, map[string]int) {
	files, size := 0, int64(0)
	kinds := map[string]int{}
	_ = filepath.Walk(base, func(p string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		rel, rerr := filepath.Rel(base, p)
		if rerr != nil {
			return nil
		}
		k := KindOf(rel)
		if k == "" {
			return nil
		}
		files++
		size += info.Size()
		kinds[k]++
		return nil
	})
	return files, float64(size) / 1048576.0, kinds
}

// localFiles 只取文件数（WriteManifest 的简化入口，需要完整统计时用 localStats）。
func localFiles(base string) int {
	n, _, _ := localStats(base)
	return n
}

// tree 把 src 下白名单内的文件（4gl / 42f / inc）拉进 dst。
//
// 差异判定：大小 + mtime。mtime 在下载时按服务器写回，所以下一次能直接比 ——
// 只比大小会漏掉「长度一样但内容改了」的源码。
type file struct {
	rpath, rrel string
	size        int64
	mt          time.Time
}

// KindOf 按相对路径判断镜像文件类别：4gl（源码）、42f（编译表单）、inc（include）。
// 不在白名单里的路径返回 ""。
func KindOf(rel string) string {
	p := "/" + strings.ToLower(strings.TrimPrefix(filepath.ToSlash(rel), "./"))
	switch {
	case strings.Contains(p, "/4gl/") && strings.HasSuffix(p, ".4gl"):
		return "4gl"
	case strings.Contains(p, "/42f/"):
		return "42f"
	case strings.HasSuffix(p, ".inc"):
		return "inc"
	}
	return ""
}

// collectByFind 用服务器侧一条 find 拿完整清单，避免逐目录 SFTP ReadDir。
// 输出用 NUL 分隔（文件名里可以有空格/换行/制表符，不能按行切）。
// find 不支持 -printf 或执行失败时返回错误，由调用方回退到 ReadDir。
//
// 白名单与 T100 的 mirror 同义：4gl 下的 .4gl、整个 42f 目录、任何 .inc。
func collectByFind(conn *host.SSHConn, src string) ([]file, error) {
	q := "'" + strings.ReplaceAll(src, "'", `'\''`) + "'"
	cmd := "find " + q + ` -type f \( -path '*/4gl/*' -iname '*.4gl' -o -path '*/42f/*' -o -iname '*.inc' \) -printf '%P\0%s\0%T@\0' 2>/dev/null`
	out, err := conn.Output(cmd, 2*time.Minute)
	if err != nil {
		return nil, err
	}
	parts := strings.Split(out, "\x00")
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	if len(parts)%3 != 0 {
		return nil, fmt.Errorf("find 输出格式不完整（%d 段）", len(parts))
	}
	files := make([]file, 0, len(parts)/3)
	for i := 0; i+2 < len(parts); i += 3 {
		rel := parts[i]
		size, err := strconv.ParseInt(parts[i+1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("解析 %q 大小失败：%w", rel, err)
		}
		sec, err := strconv.ParseFloat(parts[i+2], 64)
		if err != nil {
			return nil, fmt.Errorf("解析 %q 修改时间失败：%w", rel, err)
		}
		files = append(files, file{
			rpath: strings.TrimSuffix(src, "/") + "/" + rel,
			rrel:  strings.TrimPrefix(rel, "/"),
			size:  size,
			mt:    time.Unix(int64(sec), 0),
		})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].rrel < files[j].rrel })
	return files, nil
}

func tree(conn *host.SSHConn, sc *sftp.Client, src, dst, mode string, progress func(done, total int, msg string)) (map[string]any, error) {
	var files []file
	t0 := time.Now()

	files, err := collectByFind(conn, src)
	if err != nil {
		// find 不可用（BusyBox/权限/路径含不支持字符）时保留原逐目录实现。
		files = nil
		var collect func(rdir, rel string) error
		collect = func(rdir, rel string) error {
			entries, err := sc.ReadDir(rdir)
			if err != nil {
				return nil // 没权限的目录跳过，不拖垮整轮
			}
			sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
			for _, e := range entries {
				rpath := strings.TrimSuffix(rdir, "/") + "/" + e.Name()
				rrel := strings.TrimPrefix(rel+"/"+e.Name(), "/")
				if e.IsDir() {
					if err := collect(rpath, rrel); err != nil {
						return err
					}
					continue
				}
				if KindOf(rrel) == "" {
					continue
				}
				files = append(files, file{rpath: rpath, rrel: rrel, size: e.Size(), mt: e.ModTime()})
			}
			return nil
		}
		if err := collect(src, ""); err != nil {
			return nil, err
		}
	}

	kinds := map[string]int{}
	for _, f := range files {
		if k := KindOf(f.rrel); k != "" {
			kinds[k]++
		}
	}
	total := len(files)
	if progress != nil {
		progress(0, total, "")
	}
	got, skipped, processed := 0, 0, 0
	failed := 0
	var firstFail string
	var bytes int64
	var pruned int
	fail := func(p string, err error) {
		failed++
		if firstFail == "" {
			firstFail = p + ": " + err.Error()
		}
	}
	for _, f := range files {
		processed++
		lpath := filepath.Join(dst, filepath.FromSlash(f.rrel))
		if mode != "full" {
			if st, err := os.Stat(lpath); err == nil && st.Size() == f.size && st.ModTime().Unix() == f.mt.Unix() {
				skipped++
				if progress != nil && (processed%50 == 0 || processed == total) {
					progress(processed, total, fmt.Sprintf("  %d/%d 个（跳过 %d）", processed, total, skipped))
				}
				continue
			}
		}
		if err := os.MkdirAll(filepath.Dir(lpath), 0o755); err != nil {
			fail(f.rrel, err)
			continue
		}
		rf, err := sc.Open(f.rpath)
		if err != nil {
			fail(f.rrel, err)
			continue
		}
		lf, err := os.Create(lpath)
		if err != nil {
			rf.Close()
			fail(f.rrel, err)
			continue
		}
		n, cerr := io.Copy(lf, rf)
		lf.Close()
		rf.Close()
		if cerr != nil {
			fail(f.rrel, cerr)
			continue
		}
		_ = os.Chtimes(lpath, f.mt, f.mt)
		got++
		bytes += n
		if progress != nil && (processed%50 == 0 || processed == total) {
			progress(processed, total, fmt.Sprintf("  %d/%d 个 … %.1f MB", processed, total, float64(bytes)/1048576))
		}
	}
	// 与 T100 对齐：只有全量模式清理本地残留；增量不动已有文件。
	if mode == "full" {
		n, err := pruneStale(dst, files)
		if err != nil {
			return nil, err
		}
		pruned = n
	}
	return map[string]any{
		"files": got, "skipped": skipped, "processed": processed, "total": total,
		"failed": failed, "firstFail": firstFail,
		"pruned": pruned, "bytes": bytes, "sizeMB": float64(bytes) / 1048576,
		"kinds": kinds, "elapsed": time.Since(t0).Seconds(),
	}, nil
}

// pruneStale 全量模式下删除 dst 里服务器已不存在的镜像文件（4gl/42f/inc），并清掉空目录。
// 增量模式不调用。
func pruneStale(dst string, files []file) (int, error) {
	if _, err := os.Stat(dst); err != nil {
		return 0, nil
	}
	keep := make(map[string]bool, len(files))
	for _, f := range files {
		keep[filepath.FromSlash(f.rrel)] = true
	}
	var stale []string
	var dirs []string
	err := filepath.Walk(dst, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // 无权限的条目跳过，不拖垮整轮
		}
		if info.IsDir() {
			if path != dst {
				dirs = append(dirs, path)
			}
			return nil
		}
		rel, err := filepath.Rel(dst, path)
		if err != nil {
			return nil
		}
		if KindOf(rel) == "" {
			return nil // 不认识的残留（如 T100 时代的 42s）不动
		}
		if !keep[rel] {
			stale = append(stale, path)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	for _, p := range stale {
		_ = os.Remove(p)
	}
	// 从最深目录往回删，避免留下空壳。
	sort.Slice(dirs, func(i, j int) bool { return len(dirs[i]) > len(dirs[j]) })
	for _, d := range dirs {
		_ = os.Remove(d) // 非空会失败，正好保留
	}
	return len(stale), nil
}
