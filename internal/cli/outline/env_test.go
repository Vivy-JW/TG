package outline

import (
	"os"
	"testing"
)

// TestMain 在包级隔离开发机上的真实环境变量。
//
// 本包源码定位（resolveSource）会经 config.ResolvePath 读配置，而 TT_CONFIG 的优先级
// 最高、会盖过用例注入的 common.ConfigPath，导致"未找到环境 e"这类误报。统一清空。
func TestMain(m *testing.M) {
	for _, k := range []string{"TT_CONFIG", "TDICT_DB", "TDICT_TARGET", "T100_CODE"} {
		os.Setenv(k, "")
	}
	os.Exit(m.Run())
}
