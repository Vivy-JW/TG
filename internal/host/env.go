// 包 host：远程服务器能力共享层 —— 环境模型/SSH 连接/登录动态路径探测/源码镜像。
// 供 debug（会话调度）与 mirror/db/source/env 等 CLI 命令共同依赖，命令之间互不 import。
// tt 除 serve 外均为一次性 CLI：本包不维护跨进程状态，探测结果只在单进程内使用。
//
// 合并说明：本包以 TDebug/host 为基（它在每个有分叉的文件上都是更成熟、更安全的一版：
// ssh.go 用 channel 消除数据竞争并新增 OutputStdin，tenv.go 用定界符协议挡掉 PTY 回显
// 污染，dbprobe.go 对工具路径/账号做白名单校验且 SQL 走 stdin 而非拼进命令串），
// 再补入 TDictCli/host 独有的 mirror.go（源码镜像引擎）。
//
// 环境模型（SSHConfig/EntValue/NamedSsh）现在只在 internal/config 里定义一次，
// 本文件以类型别名暴露，因此 host.NamedSsh 与 config.NamedSsh 是同一个类型 ——
// 调用方无需转换，也不会出现两份定义漂移。
package host

import (
	"tt/internal/config"
	"tt/internal/dbconfig"
)

// 环境模型的别名。定义在 internal/config/schema.go。
type (
	// SSHConfig 远程服务器连接配置
	SSHConfig = config.SSHConfig
	// EntValue 企业编号(TOPENT)：数字或文本均可
	EntValue = config.EntValue
	// NamedSsh 服务器环境：SSH 连接 + 登录区域 + 默认企业 + 一对一挂载的库连接
	NamedSsh = config.NamedSsh
	// Hosts 服务器环境清单
	Hosts = config.Hosts
	// Connection 数据库连接（别名到 dbconfig，方便本包与调用方少写一个 import）
	Connection = dbconfig.Connection
)

// LoadHosts 读取 config.json 的环境清单（要求至少一个环境）。
// 合并前 TDictCli/host.LoadHosts 的替代：那时它还要兼容顶层 debug 键，
// 现在兼容性由 config 的一次性迁移做掉，读路径只认 hosts。
var LoadHosts = config.LoadHosts

// LoadConfigRoot 读取完整配置的类型化视图（需要环境之外的节时用它）。
var LoadConfigRoot = config.Load

// ResolveConfigPath 解析配置文件路径（--config / TT_CONFIG / 便携包 / 统一用户目录）。
var ResolveConfigPath = config.ResolvePath

// DefaultConfigPathHint 给 --help 用的一行提示。
var DefaultConfigPathHint = config.DefaultConfigPathHint
