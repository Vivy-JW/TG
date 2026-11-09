package debug

import (
	"testing"

	"tt/internal/cli/common"
)

// TestWithoutGlobalEnv 钉住台账 D27 的判据：
//
// `tt debug source --env <环境>` 在真机上报 `dial tcp :0` —— 因为 `source` 读的是**本机 serve**
// （源码目录由 serve 连的环境决定），而全局 `--env` 会把 API 基址解析带偏。
// 修法是"临时摘掉全局 --env"，所以这个摘/还原必须成对、且不能伪造"本来给过"。
func TestWithoutGlobalEnv(t *testing.T) {
	const env = "主机109"

	t.Run("给了 --env ⇒ 摘掉并说明", func(t *testing.T) {
		commonEnvSet(t, env)
		restore, had := withoutGlobalEnv()
		if !had {
			t.Fatal("本来给了 --env，应当报 had=true")
		}
		if got := commonEnvGet(); got != "" {
			t.Fatalf("摘掉后 common.Env 该是空，实得 %q", got)
		}
		restore()
		if got := commonEnvGet(); got != env {
			t.Fatalf("还原后该回到 %q，实得 %q", env, got)
		}
	})

	t.Run("没给 --env ⇒ 不动它", func(t *testing.T) {
		commonEnvSet(t, "")
		restore, had := withoutGlobalEnv()
		if had {
			t.Fatal("本来没给 --env，不该报 had=true")
		}
		restore() // 空操作也不许炸
		if got := commonEnvGet(); got != "" {
			t.Fatalf("没给时不该被改写，实得 %q", got)
		}
	})
}

func commonEnvSet(t *testing.T, v string) {
	t.Helper()
	old := common.Env
	common.Env = v
	t.Cleanup(func() { common.Env = old })
}

func commonEnvGet() string { return common.Env }
