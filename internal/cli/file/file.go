// Package file 是“服务器文件”命令组：**只读**地看文件、取文件，以及管理离线镜像。
//
// 三件事的边界：
//
//	ls/cat/head/tail/grep/find/stat/wc/du/get   在线看文件（纯 SFTP，服务器上不起进程）
//	mirror dir|path|pull                       离线镜像（子集 + 增量，原 tt dict mirror）
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
	"path/filepath"
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
	refreshTop  bool // --refresh 重新登录探测 TOP/ERP/COM（忽略缓存）
	allowBinary bool // --binary 允许把二进制内容打到终端
)

// target 一次调用点名的环境与它的只读文件视图。
type target struct {
	env  string
	ssh  *config.NamedSsh
	conn *host.SSHConn
	fs   *host.FS
	top  string // 服务器 TOP（不带尾斜杠）
}

// cur 当前调用的目标；由组 PersistentPreRunE 设置，命令只读它。
var cur *target

// Register 把 file 组挂到根命令。
func Register(root *cobra.Command) {
	g := &cobra.Command{
		Use:   "file",
		Short: "文件：看服务器文件、取文件（只读）+ 离线镜像",
		Long: `文件：在服务器上看文件、取文件（只读），以及管理离线镜像。

  tt file ls    [-la] [--human] [<路径>]                 列出目录（默认 $TOP）
  tt file cat   <文件...>                        输出内容（多文件直接拼接）
  tt file head  [-n N] <文件...>                 前 N 行（默认 10）
  tt file tail  [-n N] <文件...>                 后 N 行（默认 10）
  tt file grep  [-rinl] [-m N] [--include G] <模式> <路径...>
  （grep 用 Go 的 RE2 语法，不是 GNU grep 的扩展语法）
  tt file find  <路径...> [-name G] [-type f|d|l] [-maxdepth N]
  tt file stat  <路径...>                        类型/大小/时间/权限
  tt file wc    [-l] <文件...>                   行/词/字节
  tt file du    [-s] [--human] [<路径>]                   占用大小（按文件大小累计，1K 块）
  tt file get   <路径> [-o <本地|->]             下载一个文件（默认同名落当前目录）
  tt file mirror dir|path|pull                   离线镜像（子集 + 增量）

范围与保证：
  只读 $TOP 之内；不跟随符号链接（--follow 可开，仍限 $TOP 内）；
  服务器上不执行任何命令（纯 SFTP）；没有 put/rm/mv/chmod。
  --at <环境> 点名目标（不猜环境）；tt file --at-list 可看取值。
  $TOP 来自登录探测（缓存 30 分钟，--refresh 强制重探）。

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
			if p == "tt file" || strings.HasPrefix(p, "tt file mirror") {
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
	g.PersistentFlags().BoolVar(&refreshTop, "refresh", false, "重新登录探测 $TOP/$ERP/$COM（忽略本地缓存）")
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
			"需要 --at <环境>（或 --env <环境名>）；可用环境：%s；不知道填什么：tt file --at-list",
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
	if err := t.resolveTop(cfgPath); err != nil {
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

// argPath 把一个参数解析成 $TOP 内的绝对远程路径。
// 空参数 = $TOP 本身；相对路径按 $TOP 解析；越界直接拒。
func (t *target) argPath(arg string) (string, error) {
	arg = strings.TrimSpace(arg)
	if strings.ContainsRune(arg, 0) {
		return "", usageErr("路径里含 NUL 字节")
	}
	var p string
	if arg == "" {
		p = t.top
	} else if path.IsAbs(arg) {
		p = path.Clean(arg)
	} else {
		p = path.Clean(path.Join(t.top, arg))
	}
	if !t.within(p) {
		return "", usageErr("路径越出 $TOP（%s）：%s；本条命令只读 $TOP 之内", t.top, p)
	}
	return p, nil
}

// within 判断 p 是否在 $TOP 之内（按段边界，/topx 不算在 /top 内）。
func (t *target) within(p string) bool {
	p = path.Clean(p)
	return p == t.top || strings.HasPrefix(p, t.top+"/")
}

// resolveSafe 在 $TOP 内解析符号链接。
//
// 默认不跟随（和 grep -r / find 的默认一致）；--follow 时才跟随，且每一跳都必须
// 仍落在 $TOP 内 —— 否则一个指到 /etc 的软链就把围栏绕过去了。
func (t *target) resolveSafe(p string) (string, error) {
	if !t.within(p) {
		return "", usageErr("路径越出 $TOP（%s）：%s", t.top, p)
	}
	rel := strings.TrimPrefix(p, t.top)
	rel = strings.TrimPrefix(rel, "/")
	current := t.top
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
					return "", notFoundErr(t.top, current)
				}
				return "", ioErr("%v", host.WrapPathErr(display(t.top, current), err))
			}
			if !e.IsSymlink() {
				break
			}
			if !followLinks {
				return "", usageErr("符号链接默认不跟随：%s -> %s（确要跟随加 --follow）", display(t.top, current), e.Link)
			}
			hops++
			if hops > 16 {
				return "", usageErr("符号链接层数过多（>16）：%s", display(t.top, p))
			}
			next := e.Link
			if !path.IsAbs(next) {
				next = path.Join(path.Dir(current), next)
			}
			next = path.Clean(next)
			if !t.within(next) {
				return "", usageErr("符号链接指向 $TOP 之外：%s -> %s", display(t.top, current), next)
			}
			current = next
		}
	}
	return current, nil
}

// display 输出用的短路径：$TOP 之内去掉前缀（相对路径可直接回喂给本组命令）。
func display(top, p string) string {
	p = path.Clean(p)
	if p == top {
		return "."
	}
	if strings.HasPrefix(p, top+"/") {
		return strings.TrimPrefix(p, top+"/")
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

func notFoundErr(top, p string) *output.Error {
	return output.Errorf(output.CodeNotFound, 1, "不存在: %s", display(top, p)).
		WithHint("用 tt file ls <目录> 看有什么（路径相对 $TOP）")
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

// dataDirOf 配置所在目录（缓存落点）。
func dataDirOf(cfgPath string) string { return filepath.Dir(cfgPath) }

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
		return nil, ioErr("%v", host.WrapPathErr(display(t.top, dirPath), err))
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
		return nil, usageErr("没有匹配 %q 的文件（在 %s 下）", arg, display(t.top, dirPath))
	}
	return out, nil
}
