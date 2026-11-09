package main

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"tt/internal/cli"
)

// webDist 前端构建产物（web/dist）。
//
// 合并前这里嵌的是两套 SPA（调试工作台与字典页，各占 dist 下一个子目录）；
// 字典页已并入调试工作台里的统一设置页，所以现在只有一套，直接产出在 dist 根。
// 未构建时目录由 web/dist/.gitkeep 占位，go build 仍能通过，服务返回引导页
// 告诉用户去构建 —— 这样后端开发不被前端依赖卡住。
//
// AI 技能文件不内嵌：以仓库根目录的 skills/ 随发行包（便携版）一起分发，
// 用户可直接编辑；`tt install skills` 把它复制到目标目录。
// 与合并前的 TDebug / TDev / TDictCli 三边约定一致。
//
//go:embed all:web/dist
var webDist embed.FS

func main() {
	// 把 FS 的根收敛到 dist 目录本身。embed 的根是模块路径（web/dist/...），
	// 而所有消费者（internal/web 的 SPA 服务、调试服务的静态兜底）都按
	// "dist 根"理解路径 —— 在这里剥一层，免得每个调用方各自去猜该不该剥。
	dist, err := fs.Sub(webDist, "web/dist")
	if err != nil {
		fmt.Fprintf(os.Stderr, "内嵌前端目录异常(web/dist): %v\n", err)
		os.Exit(1)
	}
	// 开发覆盖：TT_WEB_DIR 指向一份磁盘 dist 时优先用它。
	//
	// 为什么需要它：前端产物内嵌在二进制里，换掉 web/dist 后不重新编译就还是旧界面，
	// 而重编又必须重启 serve —— 重启会清掉**内存里的调试会话**（实测代价很痛：
	// 会话池不落盘，正在调的两条作业全丢）。设了这个变量之后，改前端只要重建 dist、
	// 刷新页面就生效，后端与会话都不动。
	//
	// 发布场景不设它，行为与从前完全一致（仍用内嵌产物）。
	if d := strings.TrimSpace(os.Getenv("TT_WEB_DIR")); d != "" {
		if st, statErr := os.Stat(d); statErr != nil || !st.IsDir() {
			fmt.Fprintf(os.Stderr, "TT_WEB_DIR 不是目录(%s)，继续用内嵌前端\n", d)
		} else if disk := os.DirFS(d); func() bool { _, e := fs.Stat(disk, "index.html"); return e == nil }() {
			dist = disk
		} else {
			fmt.Fprintf(os.Stderr, "TT_WEB_DIR 下没有 index.html(%s)，继续用内嵌前端\n", d)
		}
	}
	cli.Execute(dist)
}
