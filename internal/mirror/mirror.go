// Package mirror 把 TOP GP 服务器上的 4GL 源码镜像到本地。
//
// 只收 *.4gl：GP 的源码根是 $TOP（正式区）与 $CUST（客制区）两棵，
// 42f/42s/4fd 这些编译产物不在镜像范围里（要读就直连服务器或用字典）。
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

	"tt/internal/config"
	"tt/internal/host"
)

// Dir 该环境的镜像目录：环境自己指定优先，否则 <默认根>/<环境名>。
//
// 默认根可以用 TT_GP_MIRROR_DIR 覆盖（多套环境放同一块盘时用）。
func Dir(e config.NamedSsh) string {
	if e.MirrorDir != "" {
		return e.MirrorDir
	}
	base := os.Getenv("TT_GP_MIRROR_DIR")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".tt", "gp-mirror")
	}
	return filepath.Join(base, e.Name)
}

// Manifest 镜像根目录下 manifest.json 的内容：上一次拉取的规模与时间。
type Manifest struct {
	Files   int     `json:"files"`
	SizeMB  float64 `json:"sizeMB"`
	Source  string  `json:"source"`
	Updated string  `json:"at"`
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
func WriteManifest(dir string, files int, sizeMB float64, source string) {
	_ = os.MkdirAll(dir, 0o755)
	m := Manifest{Files: files, SizeMB: sizeMB, Source: source, Updated: time.Now().Format("2006-01-02 15:04:05")}
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

// Pull 连服务器，把两棵源码树里的 *.4gl 拉进 Dir(env)。
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
	total, bytes := 0, int64(0)
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
		total += st["files"].(int)
		bytes += st["bytes"].(int64)
	}
	WriteManifest(base, total, float64(bytes)/1048576.0, "mirror")
	return out, nil
}

// tree 把 src 下所有 *.4gl 拉进 dst。
//
// 差异判定：大小 + mtime。mtime 在下载时按服务器写回，所以下一次能直接比 ——
// 只比大小会漏掉「长度一样但内容改了」的源码。
type file struct {
	rpath, rrel string
	size        int64
	mt          time.Time
}

// collectByFind 用服务器侧一条 find 拿完整清单，避免逐目录 SFTP ReadDir。
// 输出用 NUL 分隔（文件名里可以有空格/换行/制表符，不能按行切）。
// find 不支持 -printf 或执行失败时返回错误，由调用方回退到 ReadDir。
func collectByFind(conn *host.SSHConn, src string) ([]file, error) {
	q := "'" + strings.ReplaceAll(src, "'", `'\''`) + "'"
	cmd := "find " + q + ` -type f -iname '*.4gl' -printf '%P\0%s\0%T@\0' 2>/dev/null`
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
				if !strings.HasSuffix(strings.ToLower(e.Name()), ".4gl") {
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

	total := len(files)
	if progress != nil {
		progress(0, total, "")
	}
	got, skipped, processed := 0, 0, 0
	var bytes int64
	var pruned int
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
			continue
		}
		rf, err := sc.Open(f.rpath)
		if err != nil {
			continue
		}
		lf, err := os.Create(lpath)
		if err != nil {
			rf.Close()
			continue
		}
		n, cerr := io.Copy(lf, rf)
		lf.Close()
		rf.Close()
		if cerr != nil {
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
		"pruned": pruned, "bytes": bytes, "sizeMB": float64(bytes) / 1048576,
		"elapsed": time.Since(t0).Seconds(),
	}, nil
}

// pruneStale 全量模式下删除 dst 里服务器已不存在的 *.4gl，并清掉空目录。
// 只处理 .4gl，和 GP 镜像目录只收 .4gl 的范围一致；增量模式不调用。
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
		if !strings.HasSuffix(strings.ToLower(info.Name()), ".4gl") {
			return nil
		}
		rel, err := filepath.Rel(dst, path)
		if err != nil {
			return nil
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
