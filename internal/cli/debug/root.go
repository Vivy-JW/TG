package debug

// tt debug 命令组的共享件:输出开关与配置位置提示。
//
// 合并前本文件还有一整段配置文件位置解析(toolsHome/userConfigDir/userConfigPath/
// isPortable/legacyConfigPaths/looksLikeOwnConfig/migrateLegacyConfig/defaultConfigPath/
// defaultConfigPathHint/resolveConfigPath),那段代码与 TDictCli 逐行相同、两边都写着
// "改动请两边同步" —— 正是合并要消掉的那份重复。现在它只剩一份实现,在
// tt/internal/config(paths.go 的 ResolvePath / DefaultConfigPathHint / FormatTriedPaths),
// 命令里改用 common.ResolveConfig / common.ConfigHint。
//
// 配置节也从旧结构的顶层 debug(sshs + 监听 + 调试参数混在一起)拆成了
// hosts.sshs(三工具共用的环境清单)/ debug.*(调试设置)/ 顶层 listen,
// 读写分别走 config.LoadHosts、config.SaveHosts、config.EditSection、config.SetListen。

import (
	"tt/internal/cli/common"
)

// IsJSON 是否要求 JSON 输出(根命令的持久 --json,绑定在 common.JSON)。
func IsJSON() bool { return common.JSON }

// resolveConfigPath 按统一规则定位配置文件(要求文件已存在)。
// 旧实现在本包里自带一整套候选路径与迁移逻辑,现在一律走统一配置层。
func resolveConfigPath() (string, error) { return common.ResolveConfig(false) }
