package dict

// db 连接管理:tt dict db list / ping / discover。
// list    :列出各环境挂载的数据库(含 viaSsh 标记与默认项)
// ping    :验证给定环境库可达(含 viaSsh 隧道;只读 SELECT version)
// discover:SSH 自动发现某环境(--env)的数据库连接要素 → 预览/--save 写入该环境的 db
//          (oracle:chenv zone+ORACLE_HOME+TNS/tnsnames;kingbase:实例发现)

import (
	"fmt"
	"strings"

	"tt/internal/cli/common"
	"tt/internal/config"
	"tt/internal/dbconfig"
	"tt/internal/host"
	"tt/internal/output"

	"github.com/spf13/cobra"
)

var (
	discoverType  string
	discoverHost  string
	discoverSave  bool
	dbListSecrets bool
)

// dbListCmd 列出各环境挂载的数据库。
var dbListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出各环境的数据库连接(环境名/类型/地址/账号)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		type row struct {
			Env      string            `json:"env"`
			Type     string            `json:"type"`
			Address  string            `json:"address"`
			Accounts []dbconfig.DBAcct `json:"accounts"`
			ViaSsh   string            `json:"viaSsh,omitempty"`
		}
		var rows []row
		want := strings.TrimSpace(common.Env)
		for _, e := range dbCfg.SSHs {
			if e.DB == nil {
				continue
			}
			// --at/--env 点名了就只列它：列表类命令的“环境”语义是**过滤**，
			// 接受却不照做正是 --at 要消灭的那种静默错误。
			if want != "" && e.Name != want {
				continue
			}
			via := ""
			if e.DB.ViaSSH != nil {
				via = e.DB.ViaSSH.Host
			}
			rows = append(rows, row{Env: e.Name, Type: e.DB.Type, Address: e.DB.Address(),
				Accounts: e.DB.Accounts, ViaSsh: via})
		}
		if Format() != output.FormatTable {
			if dbListSecrets {
				return emit(rows, nil, nil)
			}
			// 账号清单里带明文口令,而这条命令的输出很容易被贴进日志或对话。
			// 打码规则与 tt config show/get 同一份实现(internal/config/redact.go)。
			return emit(config.RedactValue(rows), nil, nil)
		}
		if len(rows) == 0 {
			fmt.Println("(无环境挂载数据库;请在 设置-环境-数据库 页配置)")
			return nil
		}
		fmt.Printf("数据库(按环境, %d):\n", len(rows))
		for _, r := range rows {
			// 原先这里给"默认环境"打一个 `*` —— 默认环境已移除(环境一律点名),
			// 没有"哪个是默认"可言,所以不再有星标。
			via := ""
			if r.ViaSsh != "" {
				via = "  [viaSSH→" + r.ViaSsh + "]"
			}
			fmt.Printf("    %-12s %-8s %-28s 账号%d个%s\n", r.Env, r.Type, r.Address, len(r.Accounts), via)
		}
		return nil
	},
}

// dbDiscoverCmd SSH 自动发现 → 预览/保存连接。
var dbDiscoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "SSH 自动发现数据库连接要素并保存为连接(--env <环境名>)",
	Long: `登录 hosts 环境(--env)的 SSH,按区域探测数据库连接要素:
  oracle  → chenv zone + ORACLE_HOME/sqlplus + TNS 别名 + tnsnames 解析 host/port/service
  kingbase→ 实例发现(ps 找 kingbase + ksql + kingbase.conf 端口 + 库名)
输出候选连接(不落盘);--save 写入该环境的 db(config.json hosts.sshs[].db)。
注意:自动发现给出的是"服务器视角"要素;客户端直连地址(host)需自行确认,
若 DB 仅服务器可达,保存后再补 viaSsh 配置。`,
	Example: `  tt dict db discover --env 正式区                 # oracle(默认),预览
  tt dict db discover --env 正式区 --type kingbase
  tt dict db discover --env 正式区 --type kingbase --save`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ssh, zone, envName, err := discoverSSHFromEnv(common.Env)
		if err != nil {
			return err
		}
		typ := discoverType
		if typ == "" {
			typ = "oracle"
		}
		req := host.DBProbeReq{Host: ssh.Host, Port: ssh.Port, User: ssh.User,
			Password: ssh.Password, Zone: zone, Type: typ}
		out, err := host.ProbeDBConfig(req)
		if err != nil {
			return err
		}
		cand := candidateConn(out)
		if Format() != output.FormatTable {
			return emitOne(cand)
		}
		fmt.Printf("自动发现(SSH %s, env=%s):\n", ssh.Host, envName)
		fmt.Printf("  类型: %s", cand.Type)
		if cand.Type == "oracle" {
			fmt.Printf("  TNS=%s  Host=%s:%d  Service=%s\n",
				out.TNS, cand.Host, cand.Port, cand.Service)
		} else {
			fmt.Printf("  Host=%s:%d  Database=%s\n", cand.Host, cand.Port, cand.Database)
		}
		if out.Note != "" {
			fmt.Printf("  提示: %s\n", out.Note)
		}
		fmt.Println("  说明: host 为服务器视角候选地址,客户端不可达时请改 host 或补 viaSsh")
		if !discoverSave {
			fmt.Println("  保存: 加 --save 写入该环境的 db(config.json hosts.sshs[].db)")
			return nil
		}
		if err := saveDiscoveredConn(cand, envName); err != nil {
			return err
		}
		fmt.Printf("已写入环境 %q 的数据库连接\n", envName)
		return nil
	},
}

func firstLine(s string) string {
	for i, c := range s {
		if c == '\r' || c == '\n' {
			return s[:i]
		}
	}
	return s
}

// discoverSSHFromEnv 从 hosts.sshs 找到匹配环境并返回 SSH+zone。
func discoverSSHFromEnv(name string) (host.SSHConfig, string, string, error) {
	cfgPath, err := common.ResolveConfig(false)
	if err != nil {
		return host.SSHConfig{}, "", "", err
	}
	cfg, err := config.LoadHosts(cfgPath)
	if err != nil {
		return host.SSHConfig{}, "", "", err
	}
	ref, err := cfg.Resolve(name)
	if err != nil {
		return host.SSHConfig{}, "", "", err
	}
	s := ref.Env.SSHConfig
	if s.Port == 0 {
		s.Port = 22
	}
	return s, ref.Env.Zone, ref.Name, nil
}

// candidateConn 由探测结果构造连接要素(type/host/port/service|库名);
// 账号列表不在此生成(独立维护,保存时保留原列表)。
func candidateConn(out *host.DBProbeOut) *dbconfig.Connection {
	c := &dbconfig.Connection{Type: out.Type}
	if out.Type == "kingbase" {
		c.Host = discoverHost
		c.Port = out.Port
		c.Database = out.Database
		if c.Port == 0 {
			c.Port = 54321
		}
	} else {
		c.Host = out.Host
		c.Port = out.Port
		c.Service = out.Service // oracle: service_name
		if c.Port == 0 {
			c.Port = 1521
		}
	}
	if c.Host == "" {
		c.Host = discoverHost
	}
	return c
}

// saveDiscoveredConn 把连接要素写入 config.json hosts.sshs[envName].db(保留账号列表)。
// 环境清单是类型化的,写回整体走 config.EditHosts(唯一写路径)。
func saveDiscoveredConn(c *dbconfig.Connection, envName string) error {
	cfgPath, err := common.ResolveConfig(true)
	if err != nil {
		return err
	}
	return config.EditHosts(cfgPath, func(h *config.Hosts) error {
		e := h.ByName(envName)
		if e == nil {
			return fmt.Errorf("未找到环境 %q", envName)
		}
		// 保留账号列表(账号独立维护,探测只更新连接要素)
		var accounts []dbconfig.DBAcct
		if e.DB != nil {
			accounts = e.DB.Accounts
		}
		nc := *c
		nc.Accounts = accounts
		e.DB = &nc
		return nil
	})
}

func init() {
	dbDiscoverCmd.Flags().StringVar(&discoverType, "type", "", "数据库类型(oracle/kingbase;默认 oracle)")
	dbDiscoverCmd.Flags().StringVar(&discoverHost, "host", "", "客户端可达 DB 地址覆盖(默认取探测/ssh host)")
	dbDiscoverCmd.Flags().BoolVar(&discoverSave, "save", false, "写入该环境的 db(config.json hosts.sshs[].db)")
	dbCmd.AddCommand(dbListCmd, dbDiscoverCmd)
	dbListCmd.Flags().BoolVar(&dbListSecrets, "show-secrets", false, "原样输出账号口令(默认打码)")
}
