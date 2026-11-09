package dict

import (
	"os"
	"testing"
)

// TestMain 在包级隔离开发机上的真实环境变量。
//
// 本包大量用例读写 config.json（数据目录、spill 目录、各环境字典库落点都从它推），
// 若外泄 TT_CONFIG / TDICT_DB，这些用例会读到生产配置（真实 TT\config\dict、spill 目录），
// 整包误红且报错会指向机器上的真实文件。这里统一清空，代价是"依赖真实环境变量的
// 行为"在本包不被覆盖 —— 那类行为本就该用显式参数测，不靠宿主环境。
func TestMain(m *testing.M) {
	for _, k := range []string{"TT_CONFIG", "TDICT_DB", "TDICT_TARGET", "T100_CODE"} {
		os.Setenv(k, "")
	}
	os.Exit(m.Run())
}
