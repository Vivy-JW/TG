package dict

import (
	"context"
	"fmt"
	"os"
	"time"

	"tg/internal/cli/common"
	"tg/internal/config"
	dictlib "tg/internal/dict"

	"github.com/spf13/cobra"
)

var dbSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "从 TOP GP 拉取字典，重建本地 SQLite",
	Long: `连 TOP GP 的 Oracle，把表档/字段/索引/约束/程序/程序用表/错误消息拉进本地 SQLite。

落盘位置：该环境配置里的 dictDir，默认 ~/.tg/gp-dict/<环境>.db（Web 默认在数据目录/dict）。
字典据点固定按 ds 取（ds 表档最全，2,962 张），与设置页「营运据点」选哪家无关 ——
营运据点只决定直查（dict r.t --env <环境>）连哪个 schema。
环境必须点名：--env <环境> 或 --at <环境>。
各数据族的本地覆盖情况见 tg dict db status。`,
	Example: `  tg dict db sync --env 22主机`,
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		_, envName, err := resolveDbConn(common.Env)
		if err != nil {
			return err
		}
		e, ok := gpEnvNamed(envName)
		if !ok {
			return fmt.Errorf("环境 %s 不是 TOP GP 环境；TG 只同步 TOP GP 的字典", envName)
		}
		return runGPSync(e)
	},
}

func init() {
	dbCmd.AddCommand(dbSyncCmd)
}

// runGPSync 重建 TOP GP 的本地字典库（写 dictlib.DictPath）。
//
// 与 Web 的「同步字典」按钮同一个入口（dictlib.SyncEnv），只是没有 job 包装：
// 进度直接打到 stderr，JSON 模式整包返回。
func runGPSync(env config.NamedSsh) error {
	jsonOut := common.IsJSON()
	fmt.Fprintf(os.Stderr, "正在从 TOP GP 拉取字典数据 (环境: %s) -> %s\n", env.Name, dictlib.DictPath(env))
	start := time.Now()
	steps := []string{}
	ctx, cancel := context.WithTimeout(context.Background(), dictlib.SyncTimeout)
	defer cancel()
	_, err := dictlib.SyncEnv(ctx, env, func(msg string) {
		steps = append(steps, msg)
		if !jsonOut {
			fmt.Fprintln(os.Stderr, "  "+msg)
		}
	}, nil)
	if err != nil {
		return fmt.Errorf("同步 GP 字典失败: %w", err)
	}
	cost := time.Since(start).Round(time.Second)
	if jsonOut {
		return common.PrintJSON(map[string]any{
			"ok": true, "env": env.Name, "target": dictlib.DictPath(env),
			"用时": cost.String(), "步骤": steps,
		})
	}
	fmt.Printf("同步完成: %s (用时 %v)\n", dictlib.DictPath(env), cost)
	return nil
}
