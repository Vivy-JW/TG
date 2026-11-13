package debug

// `tg debug source` 出错时给一段**命令行视角**的"怎么办"。
// 服务端文案（internal/debug/manager.go 的 sourceRootsUnknownErr / sourceErrNoEnv）
// 已经说清是哪一步没配；这里再补下一步命令，免得两处都丢了线索。
// 本文件只测这段纯字符串逻辑：不连服务、不读盘。

import (
	"fmt"
	"os"
	"strings"
)

// sourceDirHowTo 命中"源码取不到"这一类错误时返回命令行视角的补丁说明；不命中返回 nil。
//
// 判据只看文案关键词（源码读取没有专用错误码），所以宁可窄：
// 只认这几种已知失败，别把别的错误也包装成同一套提示。
func sourceDirHowTo(errText string) []string {
	switch {
	case strings.Contains(errText, "登录区源码目录未知"),
		strings.Contains(errText, "没有配 $TOP"),
		strings.Contains(errText, "没有可用的 SSH 连接"):
		return []string{
			"怎么办（GP 的源码根是环境里配的 $TOP / $CUST，不猜路径）：",
			"  1) tg env show <环境>        —— 看 workspace（$TOP）与 custRoot（$CUST）是不是空的",
			"  2) 设置页「站点管理 → SSH 服务器」补上，$TOP 形如 /u1/<实例>/tiptop",
			"  3) 只查标准程序时 $CUST 可以不填；客制程序才需要它",
		}
	case strings.Contains(errText, "源码未找到"):
		return []string{
			"怎么办（模块留空或填错都会按本地字典的 p_link 解析实际模块）：",
			"  1) tg site list --env <环境>              —— 确认连的是哪个资料库",
			"  2) tg dict db status --env <环境>         —— 本地字典库要同步过，p_link 才查得到",
			"  3) tg file ls <模块>/4gl --at <环境>      —— 直接看服务器上有没有这个文件",
		}
	}
	return nil
}

// printSourceDirHowTo 把补丁说明打到 stderr（stdout 留给 JSON/正文，别污染管道）。
func printSourceDirHowTo(errText string) {
	for _, ln := range sourceDirHowTo(errText) {
		fmt.Fprintln(os.Stderr, ln)
	}
}
