package update

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Repo 更新源。**写死在代码里**，不给命令行开关：更新器是一个"下载可执行文件并运行它"
// 的路径，让它能指向别处等于把这条路径交给调用参数。
const (
	RepoSlug  = "luyaogod/TT"
	RepoPage  = "https://github.com/" + RepoSlug
	apiBase   = "https://api.github.com/repos/" + RepoSlug
	LatestAPI = apiBase + "/releases/latest"
	// ListAPI 列发布（`--pre` 用它：/releases/latest 有意跳过预发布）。
	ListAPI = apiBase + "/releases?per_page=20"
)

// Asset 一个发布资产。Digest 形如 "sha256:<hex>"。
//
// **Digest 为空就不许安装**（只允许 check）：没有摘要就没有"下载到的字节 == 发布方
// 发布的字节"这条判据，而这是整条路径上唯一不靠信任的判断。
type Asset struct {
	Name   string `json:"name"`
	Size   int64  `json:"size"`
	Digest string `json:"digest"`
	URL    string `json:"browser_download_url"`
}

// HexDigest 取出 sha256 的十六进制部分；不是 sha256 或格式不对时返回空串。
func (a Asset) HexDigest() string {
	s, ok := strings.CutPrefix(a.Digest, "sha256:")
	if !ok {
		return ""
	}
	s = strings.ToLower(strings.TrimSpace(s))
	if len(s) != 64 {
		return ""
	}
	for _, c := range s {
		if !strings.ContainsRune("0123456789abcdef", c) {
			return ""
		}
	}
	return s
}

// Release 一次发布。
type Release struct {
	Tag         string    `json:"tag_name"`
	Prerelease  bool      `json:"prerelease"`
	PublishedAt time.Time `json:"published_at"`
	Assets      []Asset   `json:"assets"`
}

// Version 取发布对应的版本号（tag 的 v 前缀由 ParseVersion 容忍）。
func (r *Release) Version() (Version, error) { return ParseVersion(r.Tag) }

// ParseRelease 解析 GitHub 的发布 JSON。
//
// 只读它需要的字段，多余字段一律忽略：GitHub 的响应体很大，而我们要的是"这个发布是
// 谁、有哪些资产、摘要是什么"。
func ParseRelease(b []byte) (*Release, error) {
	var r Release
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, fmt.Errorf("解析发布信息失败: %w", err)
	}
	if r.Tag == "" {
		return nil, fmt.Errorf("发布信息里没有 tag_name")
	}
	return &r, nil
}

// ParseReleases 解析 `/releases`（数组），按发布时间从新到旧排序。
func ParseReleases(b []byte) ([]*Release, error) {
	var list []*Release
	if err := json.Unmarshal(b, &list); err != nil {
		return nil, fmt.Errorf("解析发布列表失败: %w", err)
	}
	sort.SliceStable(list, func(i, j int) bool { return list[i].PublishedAt.After(list[j].PublishedAt) })
	return list, nil
}

// AssetFor 按安装形态挑出该下载哪个资产（见 PickAsset）。
func (r *Release) AssetFor(k Kind) (Asset, error) { return PickAsset(r.Assets, k) }

// PickAsset 按安装形态挑出该下载哪个资产。
//
// 名字里带版本号不是判据（今天的 zip 叫 `tt-portable.zip`、MSI 叫 `TT-0.2.1-x64.msi`，
// 两种惯例并存），所以这里只按**扩展名**挑：MSI 一次只能装一个，便携包一次只能解一个
// 。"下到的制品是不是这一次发布的"由摘要与制品自报版本回答，不看文件名。
func PickAsset(assets []Asset, k Kind) (Asset, error) {
	var want string
	switch k {
	case KindMSI:
		want = ".msi"
	case KindPortable:
		want = ".zip"
	default:
		return Asset{}, fmt.Errorf("安装形态 %s 不从发布里取资产", k)
	}
	var found []Asset
	for _, a := range assets {
		if strings.HasSuffix(strings.ToLower(a.Name), want) {
			found = append(found, a)
		}
	}
	switch len(found) {
	case 0:
		return Asset{}, fmt.Errorf("发布里没有 %s 资产", want)
	case 1:
		return found[0], nil
	default:
		names := make([]string, 0, len(found))
		for _, a := range found {
			names = append(names, a.Name)
		}
		return Asset{}, fmt.Errorf("发布里有多个 %s 资产（%s）：不知道该装哪个",
			want, strings.Join(names, " / "))
	}
}
