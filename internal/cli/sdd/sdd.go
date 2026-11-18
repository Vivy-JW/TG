// Package sdd 实现 tg sdd：读 TOP GP 的作業說明書（<模块>/sdd/<作業>.sdd）。
//
// 一个作业的说明书按作业编号前缀成组：axmt410 会同时命中
// axmt410dis.sdd（单据）、axmt410icd.sdd（资料维护）、axmt410slk.sdd（查询）
// 三份。默认打第一份并把其余列出来，--list 只列文件不打印内容。
//
// 数据源优先级与 tg outline 一致：
//
//	--dir <目录>               → 在该目录（可以是镜像目录）里按前缀找
//	--at/--env 有镜像         → 在 mirror.DirFor(环境) 里按前缀找
//	--at/--env 没有镜像/没命中 → 直接连服务器在 $TOP 与 $CUST 的 */sdd/ 下找
package sdd

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/spf13/cobra"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/transform"

	"tg/internal/cli/common"
	"tg/internal/config"
	"tg/internal/mirror"
	"tg/internal/output"
)

// schemaVersion 是 JSON 信封的标识，与 tg outline / tg interface 同风格。
const schemaVersion = "tg.sdd/v1"

// match 是一份命中的说明书。
type match struct {
	Path   string `json:"path"`   // 相对根目录的路径（镜像里与服务器上是同一套相对路径）
	Source string `json:"source"` // mirror | server
	Bytes  int64  `json:"bytes"`
	module string // 顶层模块目录名，只用于人读表格
}

// result 是 --format json 的输出（默认）。
type result struct {
	OK       bool    `json:"ok"`
	Schema   string  `json:"schema"`
	Job      string  `json:"job"`
	Source   string  `json:"source,omitempty"`
	Path     string  `json:"path,omitempty"`
	Bytes    int64   `json:"bytes,omitempty"`
	Encoding string  `json:"encoding,omitempty"`
	Matches  []match `json:"matches"`
	Content  string  `json:"content,omitempty"`
}

type options struct {
	dir  string
	list bool
}

// Register 把 tg sdd 挂到根命令上。
func Register(root *cobra.Command) { root.AddCommand(newCmd()) }

func newCmd() *cobra.Command {
	o := &options{}
	cmd := &cobra.Command{
		Use:   "sdd <作業>",
		Short: "读作业的作業說明書（<模块>/sdd/<作業>*.sdd）",
		Long: `读 TOP GP 的作業說明書：服务器上每个模块的 sdd/ 目录里都有 <作業>*.sdd，
内容是这份作业的系统名称、执行指令、操作功能、逐字段说明与数据处理逻辑。

一个作业编号往往对应几份：axmt410 有 axmt410dis（单据）、axmt410icd（资料维护）、
axmt410slk（查询）。默认打第一份（有 <作業>.sdd 就优先它），其余列出来；--list 只列不打印。

数据源优先级：
  --dir <目录>            在该目录里按前缀找（可以是镜像目录）
  --at/--env 有镜像      在环境镜像里找（tg file mirror pull 拉过就有）
  --at/--env 没有镜像/没命中  直接连服务器，在 $TOP 与 $CUST 的 */sdd/ 下找

示例：
  tg sdd axmt410 --at 22主机          # 有镜像用镜像，没有就直读服务器
  tg sdd axmt410 --at 22主机 --list   # 只列这份作业有几份说明书
  tg sdd axmt410 --dir D:\tg\mirror   # 只看本地镜像`,
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) != 1 {
				return output.Errorf(output.CodeUsage, 2, "需要且只接受一个作业编号；示例：tg sdd axmt410 --at 22主机")
			}
			if strings.TrimSpace(args[0]) == "" {
				return output.Errorf(output.CodeUsage, 2, "作业编号不能为空；示例：tg sdd axmt410 --at 22主机")
			}
			return nil
		},
		RunE: func(_ *cobra.Command, args []string) error {
			return run(args[0], o)
		},
	}
	f := cmd.Flags()
	f.StringVar(&o.dir, "dir", "", "镜像目录（与 --at/--env 互斥）：在该目录里按前缀找")
	f.BoolVar(&o.list, "list", false, "只列出命中的说明书文件，不打印内容")
	return cmd
}

func run(job string, o *options) error {
	job = normalizeJob(job)
	env := strings.TrimSpace(common.Env)
	dir := strings.TrimSpace(o.dir)
	if dir != "" && env != "" {
		return output.Errorf(output.CodeUsage, 2, "--dir 与 --at/--env 不能同时给：选一个数据源").
			WithHint("看本地镜像用 --dir；要自动找镜像/直读服务器用 --at <环境>")
	}
	if dir == "" && env == "" {
		return output.Errorf(output.CodeUsage, 2, "要指名数据源：--at <环境> 或 --dir <目录>").
			WithHint("先用 tg file mirror pull --at <环境> 拉一份镜像，再用 --at <环境> 直接读")
	}

	var (
		matches []match
		err     error
		root    string // 本地镜像/目录的绝对路径
		envCfg  *config.NamedSsh
	)
	if dir != "" {
		if root, err = filepath.Abs(dir); err != nil {
			return output.Errorf(output.CodeUsage, 2, "解析 --dir 失败: %v", err)
		}
		if st, statErr := os.Stat(root); statErr != nil || !st.IsDir() {
			return output.Errorf(output.CodeNotFound, 4, "目录不存在: %s", root).
				WithHint("先 tg file mirror pull --at <环境> 拉一份，或 --dir 指到已有镜像")
		}
		matches, err = scanLocal(root, job)
	} else {
		if envCfg, err = envNamed(env); err != nil {
			return err
		}
		cfgPath, cfgErr := common.ResolveConfig(false)
		if cfgErr == nil {
			root = mirror.DirFor(cfgPath, *envCfg)
		}
		if root != "" {
			if st, statErr := os.Stat(root); statErr == nil && st.IsDir() {
				matches, err = scanLocal(root, job)
			}
		}
	}
	if err != nil {
		return err
	}

	// 镜像/目录没命中就去服务器读（只在点名环境时）。
	if len(matches) == 0 && envCfg != nil {
		matches, err = findRemote(*envCfg, job)
		if err != nil {
			return err
		}
	}
	if len(matches) == 0 {
		return output.Errorf(output.CodeNotFound, 4, "没有找到作业 %s 的说明书", job).
			WithHint("确认作业编号（只要前缀，如 axmt410）；看有哪些模块/文件可以 tg file find --at <环境> . -name '*.sdd'")
	}

	picked := pickOne(matches, job)
	others := otherPaths(matches, picked)

	if o.list {
		return renderList(job, matches)
	}

	data, err := readMatch(picked, root, envCfg)
	if err != nil {
		return err
	}
	// 说明书是 Big5（繁中）编码：先转成 UTF-8 再往外给，否则终端与 JSON 里都是乱码。
	data, enc := toUTF8(data)

	if common.IsJSON() {
		return common.PrintJSON(result{
			OK:       true,
			Schema:   schemaVersion,
			Job:      job,
			Source:   picked.Source,
			Path:     picked.Path,
			Bytes:    picked.Bytes,
			Encoding: enc,
			Matches:  matches,
			Content:  string(data),
		})
	}
	if _, err := os.Stdout.Write(data); err != nil {
		return output.Errorf(output.CodeLocalIO, 1, "写输出失败: %v", err)
	}
	if len(others) > 0 {
		fmt.Fprintf(os.Stderr, "（另有 %d 份：%s；加 --list 看全部）\n", len(others), strings.Join(others, "、"))
	}
	return nil
}

// normalizeJob 去掉首尾空白、盘符路径与 .sdd 后缀，统一小写。
func normalizeJob(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, ".sdd")
	s = strings.TrimSuffix(s, ".SDD")
	s = strings.ReplaceAll(s, "\\", "/")
	s = strings.TrimPrefix(s, "./")
	return strings.ToLower(strings.TrimSpace(s))
}

// scanLocal 在目录里按前缀找 */sdd/*.sdd。
// job 里带 "/" 时按相对路径前缀匹配（如 axm/axmt410），否则只比文件名。
func scanLocal(root, job string) ([]match, error) {
	var out []match
	byPath := strings.Contains(job, "/")
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(root, p)
		if relErr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if !isSDD(rel) {
			return nil
		}
		low := strings.ToLower(rel)
		if byPath {
			// 带路径的写法：<模块>/<前缀> → <模块>/sdd/<前缀>*.sdd（模块可以是 a/b 多层）。
			head, prefix := job[:strings.LastIndexByte(job, '/')], job[strings.LastIndexByte(job, '/')+1:]
			if !strings.HasPrefix(low, head+"/sdd/"+prefix) {
				return nil
			}
		} else if !strings.HasPrefix(path.Base(low), job) {
			return nil
		}
		info, infoErr := d.Info()
		if infoErr != nil {
			return nil
		}
		out = append(out, match{Path: rel, Source: "mirror", Bytes: info.Size(), module: topDir(rel)})
		return nil
	})
	if err != nil {
		return nil, output.Errorf(output.CodeLocalIO, 1, "读目录 %s 失败: %v", root, err)
	}
	sortMatches(out)
	return out, nil
}

// isSDD 判断相对路径是不是某模块 sdd/ 目录下的 .sdd（<模块>/sdd/<文件>.sdd）。
func isSDD(rel string) bool {
	if !strings.HasSuffix(strings.ToLower(rel), ".sdd") {
		return false
	}
	parts := strings.Split(rel, "/")
	if len(parts) < 3 {
		return false
	}
	return parts[len(parts)-2] == "sdd"
}

func topDir(rel string) string {
	if i := strings.IndexByte(rel, '/'); i > 0 {
		return rel[:i]
	}
	return rel
}

func sortMatches(ms []match) {
	sort.Slice(ms, func(i, j int) bool { return ms[i].Path < ms[j].Path })
}

// pickOne 默认挑 <作業>.sdd；没有就按名字取第一份。
func pickOne(ms []match, job string) match {
	base := path.Base(job)
	for _, m := range ms {
		if path.Base(strings.ToLower(m.Path)) == base+".sdd" {
			return m
		}
	}
	return ms[0]
}

func otherPaths(ms []match, picked match) []string {
	var out []string
	for _, m := range ms {
		if m.Path != picked.Path {
			out = append(out, m.Path)
		}
	}
	return out
}

func renderList(job string, ms []match) error {
	if common.IsJSON() {
		return common.PrintJSON(result{OK: true, Schema: schemaVersion, Job: job, Matches: ms})
	}
	rows := make([][]string, 0, len(ms))
	for _, m := range ms {
		rows = append(rows, []string{job, m.Path, humanBytes(m.Bytes), m.Source})
	}
	return output.WriteTable(os.Stdout, []string{"作业", "路径", "大小", "来源"}, rows)
}

// readMatch 读中选的那份：镜像里直接读文件，服务器上走 SFTP。
func readMatch(m match, root string, e *config.NamedSsh) ([]byte, error) {
	if m.Source == "mirror" {
		p := filepath.Join(root, filepath.FromSlash(m.Path))
		data, err := os.ReadFile(p)
		if err != nil {
			return nil, output.Errorf(output.CodeLocalIO, 1, "读 %s 失败: %v", p, err)
		}
		return data, nil
	}
	if e == nil {
		return nil, output.Errorf(output.CodeUsage, 2, "没有点名环境，读不到服务器上的文件")
	}
	_, data, err := common.ReadRemoteFile(*e, []string{m.Path}, path.Base(m.Path), 0)
	return data, err
}

// toUTF8 把说明书转成 UTF-8：GP 的 .sdd 是 Big5（繁中），少数已经是 UTF-8。
// 判据是「能不能按 UTF-8 解码」——能就原样返回，不能就按 Big5 解。
// 返回的第二个值是这次用的编码，写进 JSON 让人知道原文是什么。
func toUTF8(data []byte) ([]byte, string) {
	if utf8.Valid(data) {
		return data, "utf-8"
	}
	out, _, err := transform.Bytes(traditionalchinese.Big5.NewDecoder(), data)
	if err != nil {
		return data, "big5?(解码失败)"
	}
	return out, "big5"
}

func humanBytes(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(n)/(1<<10))
	}
	return fmt.Sprintf("%d B", n)
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
