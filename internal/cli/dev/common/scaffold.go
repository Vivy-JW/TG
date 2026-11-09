// tdev 两条线（tzc / tzs）共用的输出与解析脚手架。
//
// 为什么单独一个叶子包：selftest 的 31 项对抗用例是 **tdev 全集**（tzc 的 30 项 +
// tzs 的 1 项），所以 tzc 必然要引 tzs；而两条线又都要同一份输出契约 ——
// 公共件放任何一条线里，另一条就得反向依赖，成环。这里中立，两条线都只向下引。
//
// 这几件是**契约件**，只此一份，谁都不许在包外再写：
//   - Fail：--json 下 stdout 恰好一个 {"ok":false,"exit_code":N,"error":"…"}，否则 stderr 散文；
//   - EmitJSON：稳定键序的紧凑 JSON 输出（stdout 一行；文件仍用模型侧 MarshalJSONStable）；
//   - ParseArgs：「位置参数 + 选项」任意顺序（标准库 flag 做不到，先稳定重排）；
//   - StripIdentitySuffix：包主名（去扩展名、去设计器身份后缀 (c)/(s)）。
package common

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"tt/internal/config"
)

// reIdentitySuffix 匹配设计器给包名加的客制/标准身份后缀，如 capt110(c).tzc 的 "(c)"。
var reIdentitySuffix = regexp.MustCompile(`\([A-Za-z]\)$`)

// ParseArgs 允许「位置参数 + 选项」任意顺序（设计指南 §2 的用法把 -o/--json 写在
// 位置参数之后，而标准库 flag 遇到第一个位置参数就停止解析，所以先做一次稳定重排）。
// valueFlags 列出"取值为下一个参数"的开关名，重排时把值跟着开关走。
func ParseArgs(fs *flag.FlagSet, args []string, valueFlags ...string) error {
	isVal := map[string]bool{}
	for _, v := range valueFlags {
		v = strings.TrimLeft(v, "-")
		isVal["-"+v] = true
		isVal["--"+v] = true
	}
	var flags, pos []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "-") || a == "-" {
			pos = append(pos, a)
			continue
		}
		flags = append(flags, a)
		name := a
		if eq := strings.IndexByte(a, '='); eq >= 0 {
			name = a[:eq]
		}
		if isVal[name] && !strings.Contains(a, "=") && i+1 < len(args) {
			flags = append(flags, args[i+1])
			i++
		}
	}
	return fs.Parse(append(flags, pos...))
}

// EmitJSON 以稳定键序输出一个紧凑 JSON 对象（一行）。
func EmitJSON(w io.Writer, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		fmt.Fprintf(os.Stderr, "JSON 序列化失败: %v\n", err)
		return
	}
	w.Write(append(b, '\n'))
}

// Line 是人读输出的一行（自动补换行）。
func Line(w io.Writer, format string, a ...any) { fmt.Fprintf(w, format+"\n", a...) }

// Short 把 sha256/commit 哈希裁到 12 位（清单与日志用）。
func Short(s string) string {
	if len(s) > 12 {
		return s[:12]
	}
	return s
}

// Fail 打一次失败并返回退出码：--json 下 stdout 恰好一个
// {"ok":false,"exit_code":N,"error":"…"}（**走 stdout**，不在 stderr）；
// 否则 stderr 散文一行。退出码由 err 自带的 ExitCode() 决定（见 ExitCodeOf）。
func Fail(err error, asJSON bool) int {
	code := ExitCodeOf(err)
	if asJSON {
		EmitJSON(os.Stdout, map[string]any{
			"ok": false, "exit_code": code, "error": err.Error(),
		})
		return code
	}
	fmt.Fprintf(os.Stderr, "错误（退出码 %d）：%v\n", code, err)
	return code
}

// ExitCodeOf 把错误映射为退出码：错误实现了 ExitCode() int 就用它的，否则 1
// （1 = 未分类的内部错误，仅作兜底）。
func ExitCodeOf(err error) int {
	if err == nil {
		return 0
	}
	var ec interface{ ExitCode() int }
	if errors.As(err, &ec) {
		return ec.ExitCode()
	}
	return 1
}

// StripIdentitySuffix 返回包文件的主名：去扩展名、去设计器身份后缀 (c)/(s)，
// 空了退 "pkg"。tzc 工作区目录（-ws）与 tzs 解压目录（-unzip）的缺省名都从它拼出来。
func StripIdentitySuffix(pkgPath string) string {
	base := strings.TrimSuffix(filepath.Base(pkgPath), filepath.Ext(pkgPath))
	if m := reIdentitySuffix.FindString(base); m != "" {
		base = strings.TrimSuffix(base, m) // capt110(c) → capt110
	}
	if base == "" {
		base = "pkg"
	}
	return base
}

// LoadTdevSettings 读统一配置的 `tdev` 节；任何一步不成立都返回零值
// （零值即"没有配置"，调用方回退硬编码默认）。
//
// 这是 tdev 唯一一处「行为可配置」的地方（tzc 的 workspaceSuffix / tzs 的 defaultOut），
// 三条纪律写死在这里：
//
//  1. **命令行 flag 永远优先**。本函数只在 flag 取到空串时才被调用；
//     只要用户显式写了 -o，配置里的值一律不参与运算 —— 一次性的显式意图
//     不该被一个上次顺手写下的持久默认值盖掉。
//  2. **绝不因为配置失败而失败**。配置文件缺失、定位不到、JSON 非法、没有 `tdev` 节
//     —— 一律静默退回硬编码行为。没有配置文件的 tdev 用户必须照常可用。
//  3. **不进热路径**。只有「确实要用默认值」的那一处（-o 缺省）才读一次配置。
func LoadTdevSettings() config.TdevSettings {
	// allowMissing=true：配置文件不存在时返回缺省落点而不是报错，
	// 正好对上「没有配置文件的 tdev 用户照常可用」。
	path, err := config.ResolvePath("", true)
	if err != nil || path == "" {
		return config.TdevSettings{}
	}
	root, err := config.Load(path)
	if err != nil {
		return config.TdevSettings{}
	}
	return root.Tdev
}
