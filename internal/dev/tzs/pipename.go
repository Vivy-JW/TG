package tzs

// pipename.go —— 管道名只能**问**引擎，不能自己算。
//
// 引擎的 Rpc.PipeName 把两个东西混进名字里：
//
//	工作区      —— 一个进程 Boot 一次就永久绑定一个工作区，所以名字里必须带上它，
//	               否则两个不同模块的守护进程会撞名（W2-A/W2-B 真撞过）。
//	TzsCli.Designer.dll 的 MVID —— 每次重编都变。
//
// 第二个是关键：客户端**绝不允许连到跑陈旧字节的守护进程**。设计器把这个决定写成了
// 架构性质本身，代价是「上一次构建起的守护进程成孤儿」—— 它接受这个代价
// （「Orphaned daemons from older builds are the cost, which is why `stop` takes no
// arguments and just works」），所以孤儿由 server.go 的 Reap 去收，不由名字去兼容。
//
// 于是这里不许做两件事：
//
//  1. **不许自己重算**。hash 是对 UTF-16 码元逐个 `h*31+c` 再 `& 0x7fffffff` 的
//     int32 回绕，MVID 又在程序集元数据里 —— 算错任何一半都是静默失败：
//     客户端连到一个没人监听的管道，判定「没有守护进程」，spawn 一个，然后
//     还是连不上（连上的那个监听的是另一个名字），报出来的错是「冷启动 60 s 未就绪」。
//  2. **不许缓存**。MVID 变了名字就变了，缓存下来的名字会让重编引擎之后的所有命令
//     都去敲一扇已经拆掉的门。每次调用都问一次，代价是几十毫秒（这个开关不 Boot）。

import (
	"bytes"
	"context"
	"regexp"
	"strings"
)

// rePipeName 是引擎产出的管道名形状：tzs-cli-<8 位十六进制工作区摘要>-<8 位十六进制 MVID>。
// 例：tzs-cli-01a04826-11c46f9d
var rePipeName = regexp.MustCompile(`^tzs-cli-[0-9a-f]{8}-[0-9a-f]{8}$`)

// PipeName 跑一次 `<exe> --pipe-name --workspace <ws>`，返回引擎本构建会监听的管道名。
//
// 每次都问，不缓存（见文件注释）。失败是传输/环境失败（退出码 5）：它说明我们到不了引擎，
// 而不是调用方写错了什么 —— 除非 exe 路径本身是空的/错的，那其实是配置问题，
// 但归到 5（IO·环境失败）比归到 2（用法错）更贴近用户要修的东西。
func PipeName(ctx context.Context, exe, workspace string) (string, error) {
	if strings.TrimSpace(exe) == "" {
		return "", &TransportError{Code: CodeServerDied, Msg: "没有指定 tzs-server 的路径（--exe / 配置）"}
	}
	ws := strings.TrimSpace(workspace)
	if ws == "" {
		// 不传 --workspace 的话引擎会回落到它自己的缺省（一个真实客户目录）的管道名，
		// 我们会拿着那个名字去连别人 —— 所以宁可在这里就断掉。
		return "", &UsageError{Msg: "问管道名时必须给出工作区（引擎缺省是一个真实客户目录）"}
	}
	args := []string{"--pipe-name", "--workspace", ws}
	out, err := runEngine(ctx, exe, args...)
	if err != nil {
		return "", &TransportError{Code: CodeServerDied,
			Msg: "问不到管道名（" + exe + " --pipe-name）", Err: err}
	}
	return parsePipeName(out)
}

// parsePipeName 校验并裁剪 `--pipe-name` 的输出。
//
// 为什么必须校验：这条命令的成功退出码只说明 exe 跑起来了。拿 `tt.exe --pipe-name`、
// 或者一个把 usage 打到 stdout 的旧版本去问，我们会得到一个「名字」然后花 60 秒
// 等一个不存在的管道就绪。当场认出「这不是管道名」比那 60 秒便宜得多。
func parsePipeName(out []byte) (string, error) {
	s := strings.TrimSpace(string(bytes.TrimPrefix(out, []byte{0xEF, 0xBB, 0xBF})))
	s = strings.TrimSpace(s)
	if !rePipeName.MatchString(s) {
		return "", &TransportError{Code: CodeServerDied,
			Msg: "--pipe-name 打印的不是管道名：" + clip([]byte(s))}
	}
	return s, nil
}
