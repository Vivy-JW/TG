package update

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"tt/internal/config"
)

// CheckTTL 检查结果的保鲜期。
//
// 检查只在人敲命令时发生（不做后台轮询），所以这个数字不用于"省网络"，只用于回答
// "这条缓存还能不能拿来提示"。两个条件必须同时成立才用：记的当前版本 == 现在跑的版本，
// 且没超过这个时长 —— 否则升级完之后还会看到"有新版本"。
const CheckTTL = 7 * 24 * time.Hour

// UpdateDir 更新机制的缓存目录（下载物与检查结果）。
// 走 config.CacheDir：名字必须在 CacheSubdirs 清单里，否则拿到空串 —— 这就是"自有存储
// 只允许清单里的名字"这条规矩在这里的落点。
func UpdateDir(dataDir string) string { return config.CacheDir(dataDir, "update") }

// CheckCachePath 最近一次检查结果的落点。
func CheckCachePath(dataDir string) string {
	d := UpdateDir(dataDir)
	if d == "" {
		return ""
	}
	return filepath.Join(d, "check.json")
}

// CheckResult 一次检查的结果。它是"离线提示"的唯一依据：`tt version` 与设置页
// 都只读它，不再联网。
type CheckResult struct {
	Current     string    `json:"current"` // 检查时手上这份的版本
	Latest      string    `json:"latest"`  // 远端最新（去 v 前缀）
	Newer       bool      `json:"newer"`   // Latest 是否比 Current 新
	Prerelease  bool      `json:"prerelease,omitempty"`
	PublishedAt time.Time `json:"publishedAt,omitempty"`
	CheckedAt   time.Time `json:"checkedAt"`
	Assets      []Asset   `json:"assets,omitempty"`
	Source      string    `json:"source,omitempty"` // 更新源（当前只有 luyaogod/TT）
}

// Check 查一次远端最新，并与 current 比较。不落盘：落盘由命令层决定（便于测试与
// `--dry-run`）。
func Check(ctx context.Context, c *Client, current string, pre bool) (*CheckResult, error) {
	cur, err := ParseVersion(current)
	if err != nil {
		return nil, fmt.Errorf("手上这份 tt 的版本号不可用: %w", err)
	}
	rel, err := c.Latest(ctx, pre)
	if err != nil {
		return nil, err
	}
	latest, err := rel.Version()
	if err != nil {
		return nil, fmt.Errorf("远端发布的 tag %q 不可用: %w", rel.Tag, err)
	}
	return &CheckResult{
		Current:     cur.String(),
		Latest:      latest.String(),
		Newer:       latest.NewerThan(cur),
		Prerelease:  rel.Prerelease,
		PublishedAt: rel.PublishedAt,
		CheckedAt:   time.Now(),
		Assets:      rel.Assets,
		Source:      RepoSlug,
	}, nil
}

// SaveCheck 落盘检查结果。目录不存在时建出来。
func SaveCheck(dataDir string, r *CheckResult) error {
	p := CheckCachePath(dataDir)
	if p == "" {
		return fmt.Errorf("数据目录不可用，检查结果无处可落")
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return fmt.Errorf("建 %s 失败: %w", filepath.Dir(p), err)
	}
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return config.AtomicWrite(p, append(b, '\n'))
}

// LoadCachedCheck 读缓存，**只在还能拿来提示时返回结果**：
// 记的当前版本必须等于现在跑的版本，且不超过 CheckTTL。文件缺失/坏掉/不新鲜一律返回 nil
// （它是提示，不是契约 —— 读不到就不提示，不报错）。
func LoadCachedCheck(dataDir, current string, now time.Time) *CheckResult {
	p := CheckCachePath(dataDir)
	if p == "" {
		return nil
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return nil
	}
	var r CheckResult
	if err := json.Unmarshal(b, &r); err != nil {
		return nil
	}
	if r.Latest == "" {
		return nil
	}
	// 记的当前版本必须等于现在跑的版本：版本变了（升级过了）→ 旧缓存不许再提"有新版本"。
	// 两边都归一化后再比（注入的版本与缓存里存的写法可能差一个 v 前缀）。
	cv, err := ParseVersion(current)
	if err != nil || cv.String() != r.Current {
		return nil
	}
	if now.Sub(r.CheckedAt) > CheckTTL || r.CheckedAt.After(now.Add(time.Hour)) {
		return nil
	}
	return &r
}

// Message 一句话说清这次检查的结论（`tt version` 的那一行与设置页共用同一句）。
func (r *CheckResult) Message() string {
	if r == nil || !r.Newer {
		return ""
	}
	return fmt.Sprintf("有新版本 %s（当前 %s，%s 发布）——跑 tt update 升级",
		r.Latest, r.Current, r.PublishedAt.Local().Format("2006-01-02"))
}
