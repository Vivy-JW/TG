// Package file 是“服务器文件”命令组：**只读**地看文件、取文件，以及管理离线镜像。
//
// 三件事的边界：
//
//	ls/cat/head/tail/grep/find/stat/wc/du/get   在线看文件（纯 SFTP，服务器上不起进程）
//	mirror dir|path|pull                       离线镜像（子集 + 增量，原 tg dict mirror）
//
// 只读不是靠“少给几个命令”，而是靠结构：
//   - 本包只调用 host.FS 的读原语，没有 Create/Remove/Rename/Mkdir/Chmod；
//   - 不在服务器上执行任何命令（没有 exec、没有 sh -c、没有服务端 grep/find）；
//   - 路径限制在 $TOP 之内；符号链接默认不跟随（--follow 也只允许指向 $TOP 内）。
//
// 范围/退出码写进组帮助；守卫测试见 guard_test.go（源码级）与 path_test.go（路径级）。
package file

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/spf13/cobra"

	"tt/internal/cli/common"
	"tt/internal/config"
	"tt/internal/host"
	"tt/internal/output"
)

// 组级开关（只影响本组）。
var (
	followLinks bool // --follow 跟随符号链接（仍限 $TOP 内）
	allowBinary bool // --binary 允许把二进制内容打到终端
)

// target 一次调用点名的环境与它的只读文件视图。
type target struct {
	env  string
	ssh  *config.NamedSsh
	conn *host.SSHConn
	fs   *host.FS
	inst string // 实例目录 /u1/<实例>：越界判定的根，$TOP 与 $CUST 的父目录
	top  string // $TOP（不带尾斜杠）：不带前缀的相对路径按它解释
	cust string // $CUST（不带尾斜杠）：首段写 topcust/ 时走它
}

// cur 当前调用的目标；由组 PersistentPreRunE 设置，命令只读它。
var cur *target

// Register 把 file 组挂到根命令。
func Register(root *cobra.Command) {
	g := &cobra.Command{
		Use:   "file",
		Short: "文件：看服务器文件、取文件（只读）+ 离线镜像",
		Long: `文件：在服务器上看文件、取文件（只读），以及管理离线镜像。

  tg file ls    [-la] [--human] [<路径>]                 列出目录（默认实例目录）
  tg file cat   <文件...>                        输出内容（多文件直接拼接）
  tg file head  [-n N] <文件...>                 前 N 行（默认 10）
  tg file tail  [-n N] <文件...>                 后 N 行（默认 10）
  tg file grep  [-rinl] [-m N] [--include G] <模式> <路径...>
  （grep 用 Go 的 RE2 语法，不是 GNU grep 的扩展语法）
  tg file find  <路径...> [-name G] [-type f|d|l] [-maxdepth N]
  tg file stat  <路径...>                        类型/大小/时间/权限
  tg file wc    [-l] <文件...>                   行/词/字节
  tg file du    [-s] [--human] [<路径>]                   占用大小（按文件大小累计，1K 块）
  tg file get   <路径> [-o <本地|->]             下载一个文件（默认同名落当前目录）
  tg file mirror dir|path|pull                   离线镜像（4gl / 42f / inc + 增量）

路径怎么写（GP 的源码是两棵并列的树）：
  $TOP  = workspace  （/u1/<实例>/tiptop）
  $CUST = custRoot   （/u1/<实例>/topcust）
  根的落点是两者的公共父目录 —— 实例目录 /u1/<实例>：
    不带参数                    → 列实例目录（能看到 tiptop/ 与 topcust/）
    azz/4gl/p_zta.4gl           → $TOP/azz/4gl/p_zta.4gl（老写法，不变）
    tiptop/azz/4gl/x.4gl        → 同一处（显式写树名）
    topcust/cbm/4gl/x.4gl       → $CUST/cbm/4gl/x.4gl（客制树）

范围与保证：
  只读实例目录之内（含两棵树）；不跟随符号链接（--follow 可开，仍限实例目录内）；
  文件动词走纯 SFTP，服务器上不执行命令；只有 mirror 用一条只读 find 拿清单
  （不写服务器、不留文件）；没有 put/rm/mv/chmod。
  --at <环境> 点名目标（不猜环境）；tg file --at-list 可看取值。

数据默认 JSON；人读用 --human（等价 --format table）。

退出码：0 成功 / 1 无匹配或无此文件 / 2 用法或越界 / 4 连接或传输失败 / 5 本地写失败`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
		// 组自己补跑根命令的 PersistentPreRunE（cobra 只跑最靠叶的那一个），
		// 再把本次目标准备好；mirror 子树不需要 SSH，由各自命令自己读配置。
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if root := cmd.Root(); root.PersistentPreRunE != nil {
				if err := root.PersistentPreRunE(cmd, args); err != nil {
					return err
				}
			}
			p := cmd.CommandPath()
			if p == "tg file" || strings.HasPrefix(p, "tg file mirror") {
				return nil
			}
			return prepare()
		},
		PersistentPostRun: func(*cobra.Command, []string) {
			if cur != nil {
				cur.close()
				cur = nil
			}
		},
	}
	g.PersistentFlags().BoolVar(&followLinks, "follow", false, "跟随符号链接（目标仍必须在 $TOP 内）")
	g.PersistentFlags().BoolVar(&allowBinary, "binary", false, "允许输出二进制内容（默认检测到 NUL 就拒绝）")

	g.AddCommand(
		newLsCmd(), newCatCmd(), newHeadCmd(), newTailCmd(),
		newGrepCmd(), newFindCmd(), newStatCmd(), newWcCmd(), newDuCmd(),
		newGetCmd(), NewMirrorCmd(),
	)
	root.AddCommand(g)
}

// prepare 解析环境、连 SSH、打开 SFTP、取 TOP。
//
// 环境一律显式点名：Hosts.Resolve 在没给名字时直接报错并列出可用环境，
// 这里不提供任何“默认环境”的回落。
func prepare() error {
	cfgPath, err := common.ResolveConfig(false)
	if err != nil {
		return output.Errorf(output.CodeConfigBroken, 2, "%v", err)
	}
	hosts, err := config.LoadHosts(cfgPath)
	if err != nil {
		return output.Errorf(output.CodeConfigBroken, 2, "%v", err)
	}
	env := strings.TrimSpace(common.Env)
	if env == "" {
		return output.Errorf(output.CodeUsage, 2,
			"需要 --at <环境>（或 --env <环境名>）；可用环境：%s；不知道填什么：tg file --at-list",
			strings.Join(hosts.Names(), "、"))
	}
	ref, err := hosts.Resolve(env)
	if err != nil {
		return output.Errorf(output.CodeEnvUnknown, 2, "%v", err)
	}
	conn, err := host.Dial(ref.Env.SSHConfig)
	if err != nil {
		return output.Errorf(output.CodeConnFailed, 4, "连接 %s 失败: %v", ref.Env.SSHConfig.Addr(), err)
	}
	fs, err := conn.FS()
	if err != nil {
		conn.Close()
		return output.Errorf(output.CodeConnFailed, 4, "打开 SFTP 失败: %v", err)
	}
	t := &target{env: ref.Name, ssh: ref.Env, conn: conn, fs: fs}
	if err := t.resolveRoots(); err != nil {
		t.close()
		return err
	}
	cur = t
	return nil
}

func (t *target) close() {
	if t == nil {
		return
	}
	if t.fs != nil {
		t.fs.Close()
	}
	if t.conn != nil {
		t.conn.Close()
	}
}

// ---- 路径 ----

// argPath 把一个参数解析成实例目录内的绝对远程路径。
//
// 解析规则（两段式，保住老用法）：
//
//	空参数        → 实例目录本身（ls 一下就能看到 tiptop/ 与 topcust/）
//	首段 tiptop/  → 实例目录 + 参数（即 $TOP 下的路径）
//	首段 topcust/ → 实例目录 + 参数（即 $CUST 下的路径）
//	其它相对路径  → $TOP + 参数（与以前一致：azz/4gl/x.4gl 仍指 $TOP/azz/4gl/x.4gl）
//	绝对路径      → 原样，但必须落在实例目录内
func (t *target) argPath(arg string) (string, error) {
	arg = strings.TrimSpace(arg)
	if strings.ContainsRune(arg, 0) {
		return "", usageErr("路径里含 NUL 字节")
	}
	var p string
	switch {
	case arg == "":
		p = t.inst
	case path.IsAbs(arg):
		p = path.Clean(arg)
	case t.treeArg(arg):
		// 首段是树名：换成配置里的规范写法（Linux 区分大小写，
		// 写 TOPCUST/ 不该真的去 stat /u1/<实例>/TOPCUST）。
		base := path.Base(t.top)
		if t.cust != "" && strings.EqualFold(firstSeg(arg), path.Base(t.cust)) {
			base = path.Base(t.cust)
		}
		rest := strings.TrimPrefix(arg, firstSeg(arg))
		p = path.Clean(path.Join(t.inst, base, strings.TrimPrefix(rest, "/")))
	default:
		p = path.Clean(path.Join(t.top, arg))
	}
	if !t.within(p) {
		return "", usageErr("路径越出实例目录（%s）：%s；本条命令只读实例目录之内", t.inst, p)
	}
	return p, nil
}

// firstSeg 路径参数的第一段（没有斜杠就是整个参数）。
func firstSeg(arg string) string {
	if i := strings.IndexByte(arg, '/'); i >= 0 {
		return arg[:i]
	}
	return arg
}

// treeArg 首段是不是两棵树的名字（tiptop / topcust，大小写不敏感）。
func (t *target) treeArg(arg string) bool {
	seg := firstSeg(arg)
	if seg == "" || seg == "." || seg == ".." {
		return false
	}
	if strings.EqualFold(seg, path.Base(t.top)) {
		return true
	}
	return t.cust != "" && strings.EqualFold(seg, path.Base(t.cust))
}

// within 判断 p 是否在实例目录之内（按段边界，/topx 不算在 /top 内）。
func (t *target) within(p string) bool {
	p = path.Clean(p)
	return p == t.inst || strings.HasPrefix(p, t.inst+"/")
}

// resolveSafe 在实例目录内解析符号链接。
//
// 默认不跟随（和 grep -r / find 的默认一致）；--follow 时才跟随，且每一跳都必须
// 仍落在实例目录内 —— 否则一个指到 /etc 的软链就把围栏绕过去了。
func (t *target) resolveSafe(p string) (string, error) {
	if !t.within(p) {
		return "", usageErr("路径越出实例目录（%s）：%s", t.inst, p)
	}
	rel := strings.TrimPrefix(p, t.inst)
	rel = strings.TrimPrefix(rel, "/")
	current := t.inst
	hops := 0
	for _, part := range strings.Split(rel, "/") {
		if part == "" || part == "." {
			continue
		}
		current = path.Join(current, part)
		for {
			e, err := t.fs.Lstat(current)
			if err != nil {
				if host.NotFound(err) {
					return "", notFoundErr(t, current)
				}
				return "", ioErr("%v", host.WrapPathErr(t.short(current), err))
			}
			if !e.IsSymlink() {
				break
			}
			if !followLinks {
				return "", usageErr("符号链接默认不跟随：%s -> %s（确要跟随加 --follow）", t.short(current), e.Link)
			}
			hops++
			if hops > 16 {
				return "", usageErr("符号链接层数过多（>16）：%s", t.short(p))
			}
			next := e.Link
			if !path.IsAbs(next) {
				next = path.Join(path.Dir(current), next)
			}
			next = path.Clean(next)
			if !t.within(next) {
				return "", usageErr("符号链接指向实例目录之外：%s -> %s", t.short(current), next)
			}
			current = next
		}
	}
	return current, nil
}

// short 输出用的短路径。
//
// $TOP 之内的仍相对 $TOP（老习惯：azz/4gl/x.4gl 可以直接回喂给本组命令）；
// 客制树与实例目录本身相对实例目录（topcust/cbm/4gl/x.4gl）。
func (t *target) short(p string) string {
	p = path.Clean(p)
	if p == t.top {
		return "."
	}
	if strings.HasPrefix(p, t.top+"/") {
		return strings.TrimPrefix(p, t.top+"/")
	}
	if p == t.inst {
		return "."
	}
	if strings.HasPrefix(p, t.inst+"/") {
		return strings.TrimPrefix(p, t.inst+"/")
	}
	return p
}

// ---- 输出与错误 ----

// wantJSON 数据类命令默认 JSON；--human（或 --format table/csv）才走人读形态。
func wantJSON(_ *cobra.Command) bool { return common.IsJSON() }

func usageErr(format string, a ...any) *output.Error {
	return output.Errorf(output.CodeUsage, 2, format, a...)
}

func ioErr(format string, a ...any) *output.Error {
	return output.Errorf(output.CodeConnFailed, 4, format, a...)
}

func notFoundErr(t *target, p string) *output.Error {
	return output.Errorf(output.CodeNotFound, 1, "不存在: %s", t.short(p)).
		WithHint("用 tg file ls <目录> 看有什么（路径可以写 $TOP 下的相对路径，也可以写 topcust/… 指客制树）")
}

// humanBytes 人性化大小（du -h / ls -h 用）。
func humanBytes(n int64) string {
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.1fG", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.1fM", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1fK", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%d", n)
	}
}

// writeOut 统一走 stdout，便于测试时替换。
var writeOut = func(format string, a ...any) { fmt.Fprintf(os.Stdout, format, a...) }

// silentExit 只要退出码，不打印任何东西（grep/find 的“没有匹配”就是退出码 1，
// 与 Unix 一致 —— 不该把“没找到”报成一条错误）。
type silentExit struct{ code int }

func (e silentExit) Error() string         { return "" }
func (e silentExit) ExitCode() int         { return e.code }
func (e silentExit) AlreadyReported() bool { return true }

// outputErrConn 把底层传输错误包成连接类错误（退出码 4）。
func outputErrConn(err error) error {
	return output.Errorf(output.CodeConnFailed, 4, "%v", err)
}

// expandArg 展开参数里的通配（* ? [）。
//
// 我们没有 shell，通配必须自己展开：在**目录这一层**列一次目录，再按基名匹配。
// 只在最后一段支持通配（erp/apm/4gl/*.4gl），不做 ** 递归 —— 递归用 find。
func (t *target) expandArg(arg string) ([]string, error) {
	if !strings.ContainsAny(arg, "*?[") {
		return []string{arg}, nil
	}
	dir, base := path.Split(strings.TrimSpace(arg))
	dirPath, err := t.argPath(dir)
	if err != nil {
		return nil, err
	}
	dirPath, err = t.resolveSafe(dirPath)
	if err != nil {
		return nil, err
	}
	entries, err := t.fs.List(dirPath)
	if err != nil {
		return nil, ioErr("%v", host.WrapPathErr(t.short(dirPath), err))
	}
	var out []string
	for _, e := range entries {
		ok, err := path.Match(base, e.Name)
		if err != nil {
			return nil, usageErr("通配写错了：%v", err)
		}
		if ok {
			out = append(out, e.Path)
		}
	}
	if len(out) == 0 {
		return nil, usageErr("没有匹配 %q 的文件（在 %s 下）", arg, t.short(dirPath))
	}
	return out, nil
}
