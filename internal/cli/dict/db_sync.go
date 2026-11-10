package dict

import (
	"context"
	"fmt"
	"os"
	"time"

	"tt/internal/cli/common"
	"tt/internal/config"
	"tt/internal/dict/dbsync"
	"tt/internal/gp"

	"github.com/spf13/cobra"
)

var (
	dbSyncTable string
)

var dbSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "从 ERP 刷新本地查询数据",
	Long: `从 ERP 刷新本地查询数据(表字典、校验、分类码、画面规格、开窗、消息、参数、
程序与作业等全部内容,约 86 万行),查询命令读的就是它。
写入该环境自己的库：<数据目录>/dict/<环境>.db（-d/--db 可显式覆盖；旧版 TDICT_DB 只对裸 local 生效）；原库自动备份为 .bak。
每个环境一个文件，两个环境互不覆盖。
环境必须点名：--env <环境> 或 --at <环境>。
--table 可只刷部分;环境用 --env <环境名> 或 --at <环境> 点名（没有默认环境）。
各数据族与本地覆盖情况见 tt dict db status。
也可在 tt serve 的「数据同步」页面里选环境执行(带进度)。`,
	Example: `  tt dict db sync --env 正式区
  tt dict db sync --env 正式区 --table dzea_t,dzeal_t`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		conn, envName, err := resolveDbConn(common.Env)
		if err != nil {
			return err
		}
		// TOP GP 环境的同步走 gp.Service：它按 GP 的字典表(zta_file/all_tab_columns/
		// gaz_file/zz_file/zr_file/ze_file)重建本地库，与 T100 那套 dbsync 不是一批表。
		if e, ok := gpEnvNamed(envName); ok {
			return runGPSync(e)
		}
		tables := dbsync.DictTables
		if dbSyncTable != "" {
			tables = splitNames(dbSyncTable)
			if len(tables) == 0 {
				return fmt.Errorf("--table 未指定有效表名")
			}
		}

		// 目标 SQLite：按环境分文件；-d/TDICT_DB 显式覆盖。
		target := dictDBSyncTarget(envName)

		jsonOut := common.IsJSON()
		if !jsonOut {
			fmt.Printf("正在从 ERP 拉取字典数据 (环境: %s) -> %s\n", envName, target)
		}
		st, err := dbsync.Run(context.Background(), *conn, target, tables, func(p dbsync.Progress) {
			// 每张表完成时打印一行(开始回调 TableDur=0,跳过)；JSON 模式不往 stdout 写进度。
			if !jsonOut && p.Phase == "table" && p.TableDur > 0 {
				fmt.Printf("  %-10s %8d 行 (%v)\n", p.Table, p.TableRows, p.TableDur.Round(time.Millisecond))
			}
		})
		if err != nil {
			return err
		}
		if jsonOut {
			return common.PrintJSON(map[string]any{
				"ok": true, "env": envName, "target": st.Target, "tables": st.Tables,
				"rows": st.Rows, "backup": st.Backup, "warnings": st.Warnings,
			})
		}
		for _, w := range st.Warnings {
			fmt.Println(w)
		}
		fmt.Printf("同步完成: %d 张表, 共 %d 行\n", st.Tables, st.Rows)
		fmt.Printf("数据库已更新: %s\n", st.Target)
		if st.Backup != "" {
			fmt.Printf("原数据库已备份到: %s\n", st.Backup)
		}
		return nil
	},
}

func init() {
	dbSyncCmd.Flags().StringVar(&dbSyncTable, "table", "", "仅同步指定的表 (逗号分隔, 默认全部字典表)")
	dbCmd.AddCommand(dbSyncCmd)
}

// runGPSync 用 gp.Service 重建 TOP GP 的本地字典库（写 gp.DictPath）。
//
// 进度由 sync 的 job 回调给出；人类模式打到 stderr，JSON 模式整包返回。
func runGPSync(env config.NamedSsh) error {
	jsonOut := common.IsJSON()
	svc := gp.New(dictCfgPath())
	fmt.Fprintf(os.Stderr, "正在从 TOP GP 拉取字典数据 (环境: %s) -> %s\n", env.Name, gp.DictPath(env))
	start := time.Now()
	steps := []string{}
	err := svc.Sync(env, func(msg string) {
		steps = append(steps, msg)
		if !jsonOut {
			fmt.Fprintln(os.Stderr, "  "+msg)
		}
	})
	if err != nil {
		return fmt.Errorf("同步 GP 字典失败: %w", err)
	}
	cost := time.Since(start).Round(time.Second)
	if jsonOut {
		return common.PrintJSON(map[string]any{
			"ok": true, "env": env.Name, "target": gp.DictPath(env),
			"用时": cost.String(), "步骤": steps,
		})
	}
	fmt.Printf("同步完成: %s (用时 %v)\n", gp.DictPath(env), cost)
	return nil
}
