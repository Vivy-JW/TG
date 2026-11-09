package server

// 本包对 config.json 的读写全部收敛到 internal/config:
// 读走类型化视图(config.Load),写走唯一写路径(config.Edit / EditSection)。
// 以前这里(与 cli/mirror.go、cli/bdldoc.go 等)各自 json.Unmarshal 到匿名结构体、
// 再手工读-改-写整份文件,合并后统一到 internal/config 的一份实现。

import (
	"tt/internal/config"
)

// loadConfig 读类型化配置。文件不存在不是错误(首次运行返回填好缺省值的空配置)。
func loadConfig(path string) (*config.Root, error) { return config.Load(path) }

// 路径型取值的解析(AbsPath / DirStatusOf / FileStatusOf)
// 已上移到 internal/config —— 统一设置页要用同一份判断,两边各算一遍会让同一个
// sync.target 解析成不同结果。
