package dict

import (
	"fmt"
	"os"

	"tt/internal/cli/common"
	"tt/internal/config"
	"tt/internal/dbconfig"
	"tt/internal/host"

	"github.com/spf13/cobra"
)

var (
	dbCfg *host.Hosts
)

// dbCmd 管理 ERP 数据库连接(每个 SSH 环境一对一挂载的 db)与同步/远程直查。
//
// It defines its own PersistentPreRunE, which overrides the group's query-source
// hook (cobra runs the first persistent pre-run found walking leaf→root),
// so `tt dict db *` does not depend on the local erp_data.db.
var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "管理 ERP 数据库连接与数据同步",
	Long: `从 config.json 读取配置 (--config / TT_CONFIG)。数据库连接按环境一对一挂载:
hosts.sshs[].db 即该环境的库(显式 host/port/service|库名 + 账号列表)。
子命令: sync (从 ERP 拉取字典数据写入 SQLite) / list (列各环境的库) /
status (看本地库覆盖) / discover (SSH 自动发现连接要素并写入环境 db)。
连库测试用 tt db ping（客户端直连）或 tt db probe（服务器侧）。
支持连接类型: kingbase (金仓, PostgreSQL 协议)、oracle (go-ora)。`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := skipQuerySource(cmd, args); err != nil {
			return err
		}
		path, err := common.ResolveConfig(false)
		if err != nil {
			return err
		}
		if common.Verbose {
			fmt.Fprintf(os.Stderr, "[dict] 配置文件: %s\n", path)
		}
		cfg, err := config.LoadHosts(path)
		if err != nil {
			return err
		}
		dbCfg = cfg
		return nil
	},
}

func init() {
	Group.AddCommand(dbCmd)
}

// resolveDbConn 按环境名解析其 db（--env/--at <环境名>）；name 必须有值，
// 空值由 config.Resolve 报错并列出可用环境（不回落首条）。返回连接的深拷贝与所属环境名。
func resolveDbConn(name string) (*dbconfig.Connection, string, error) {
	if dbCfg == nil {
		return nil, "", fmt.Errorf("配置未加载")
	}
	ref, err := dbCfg.Resolve(name)
	if err != nil {
		return nil, name, err
	}
	cc, err := dbConnOf(ref)
	if err != nil {
		return nil, ref.Name, err
	}
	return cc, ref.Name, nil
}
