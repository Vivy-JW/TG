package debug

import (
	"testing"
	"time"

	"tt/internal/host"
)

// 停站看门狗的默认值。配置里没写这个键时取 DefaultWatchdogSeconds ——
// 这里写死 1800 而**不引那个常量**,要钉住的正是"这个数是多少":
// 文档一度写 1800 而代码里是 180,引常量的话两边一起漂就没人发现了。
func TestWatchdogDefaultSeconds(t *testing.T) {
	c := &Config{}
	c.applySsh(&host.NamedSsh{Name: "未配看门狗的环境"})
	if c.WatchdogSeconds != 1800 {
		t.Errorf("环境没覆盖时看门狗 = %d 秒,期望 1800", c.WatchdogSeconds)
	}
}

// 每环境的覆盖项优先于默认值;覆盖为 0(未配)不把默认值抹掉 ——
// applySsh 先合覆盖项、再 fillDefaults,顺序反了这里就会露馅。
func TestWatchdogEnvOverride(t *testing.T) {
	c := &Config{}
	c.applySsh(&host.NamedSsh{Name: "覆盖过的环境", WatchdogSeconds: 654})
	if c.WatchdogSeconds != 654 {
		t.Errorf("环境覆盖应生效:期望 654,得到 %d", c.WatchdogSeconds)
	}
}

// 提前提醒的落点：总时长的 80%。固定提前量（比如一律提前 60 秒）在超时被设成
// 60 秒时会让整段停站都在喊“即将放行”，成了噪音。
func TestWatchdogApproachDelay(t *testing.T) {
	cases := []struct{ total, want time.Duration }{
		{300 * time.Second, 240 * time.Second},
		{1800 * time.Second, 1440 * time.Second},
		{60 * time.Second, 48 * time.Second},
		{0, 0},
		{-1 * time.Second, 0},
	}
	for _, c := range cases {
		if got := watchdogApproachDelay(c.total); got != c.want {
			t.Errorf("watchdogApproachDelay(%v) = %v，期望 %v", c.total, got, c.want)
		}
	}
	if watchdogRetryDelay <= 0 {
		t.Error("重试间隔必须为正：0 会让重试变成忙等")
	}
}
