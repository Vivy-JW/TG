package gp

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"tt/internal/cli/common"
	dictlib "tt/internal/dict"
	"tt/internal/output"
)

// newSyncCmd 把 GP 字典整份同步到本地 SQLite。
//
// 为什么 CLI 要做这件事：GP 的字典查询（tt gp table / tt dict prog）读的都是本地
// <环境>.db，而原来只有 Web 的「同步字典」按钮能生成它 —— 配好环境、在服务器上
// 没有浏览器的人就只能干等。这段与 Web 走同一个 runSync，不另写一份同步逻辑。
func newSyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "把 GP 字典同步到本地 SQLite（表结构 / 程序 / 作业 / 程序用表）",
		Long: `把该环境的 GP 字典整份拉到本地 <环境>.db。

  tt gp sync --env <环境>

一次同步包含四族数据：
  表结构     zta_file / gat_file / all_tab_columns / all_indexes / all_constraints
  表中文名   gaq_file（字段级）
  程序/作业  gaz_file（GP 里程序即作业）+ gal_file 的入口模块
  程序用表   zr_file（GP 的 p_get_zr 扫 4GL 源码生成）

同步是只读的：只连该环境配置的库读字典，不写 ERP 库；写的是本机这个 .db。
Web 的「同步字典」按钮与这条命令走同一段代码、落到同一个文件。`,
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			// local: 前缀在这里没有意义（同步本来就要连库），但允许写，
			// 免得用户从 tt gp table 那里顺手带过来时被一句"不支持"挡住。
			t, err := resolveTarget(true)
			if err != nil {
				return err
			}
			if t.env.DB == nil {
				return envErr(t, "DB", 2, "环境 %q 没有配置数据库连接", t.env.Name).
					WithHint("在设置 → 站点管理 → 数据库里配置该环境的 Oracle 连接")
			}
			started := time.Now()
			var steps []string
			ctx, cancel := context.WithTimeout(context.Background(), dictlib.SyncTimeout)
			defer cancel()
			_, err = dictlib.SyncEnv(ctx, t.env, func(msg string) {
				steps = append(steps, msg)
				if common.IsJSON() {
					return // JSON 信封里带步骤，不往 stdout 掺进度
				}
				fmt.Fprintln(os.Stderr, "  "+msg)
			}, nil)
			if err != nil {
				return envErr(t, "GP_SYNC", 5, "GP 字典同步失败: %v", err)
			}

			dict := dictlib.DictPath(t.env)
			elapsed := time.Since(started).Round(time.Second)
			company := dictlib.SyncCompany
			if !common.IsJSON() {
				fmt.Printf("同步完成：%s（字典据点 %s，用时 %s）\n", dict, company, elapsed)
				return nil
			}
			meta := metaNotes(t.meta, "字典库 "+dict)
			meta.LocalDBPath = dict
			meta.Readonly = true
			return output.Emit(stdout, output.Options{
				Format: common.OutputFormat(),
				Meta:   meta,
				Data: map[string]any{
					"环境":   t.env.Name,
					"字典库":  dict,
					"字典据点": company,
					"用时":   elapsed.String(),
					"步骤":   steps,
				},
			})
		},
	}
}
