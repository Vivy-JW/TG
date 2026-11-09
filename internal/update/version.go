// Package update 是 tt 自身的更新机制：查有没有新版、下载并校验制品、把"替换二进制"
// 这个动作交给一个脱离子进程去做。
//
// 边界（不做什么，比职责更值得写清）：
//
//   - **不自动更新**。只有人敲 `tt update` 才走到"装"这一步（`--yes` 才免确认）。
//     一个能替换自身二进制、还能拉起安装器的路径，默认必须由人点头。
//   - **不对开发构建动手**。`make build` 出的仓库根 tt.exe 是源码态（与 skills/、
//     tzs/ 同级），只允许 check，装会被拒绝 —— 见 Kind。
//   - **不做后台轮询**。`tt serve` 与日常命令都不查，只有显式命令查（`tt update check`）。
//   - 不做 tt 之外的软件管理：引擎 exe 与设计器 dll 随包走，由整包交换保证一致。
//
// 落点：下载物与检查结果在 `<数据目录>/update/`（在 config.CacheSubdirs 清单内，
// 删了只是下次重下）；升级状态与日志在数据目录根（`.tt-update.json` / `.tt-update.log`，
// 与 `.tt-serve.json` 同类 —— 排障要看，不是缓存）。
package update

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Version 是版本号的类型化形态：MAJOR.MINOR.PATCH 三段数字，外加一个可选的预发布后缀。
//
// 为什么不用 semver 库：仓库需要的判断只有"哪个更新"与"是不是同一个"，而版本号的
// **形状**已经有一处权威（根目录 VERSION 文件，由 `make version-check` 与 build_portable.bat
// 各验一遍）。多引一个库只会多一份对"预发布后缀"的解释。
//
// 注入二进制的那一份永远不带后缀（VERSION 文件是严格的 MAJOR.MINOR.PATCH）；后缀只
// 可能来自**发布 tag**（如 v0.3.0-rc1）。两者用同一个解析器，是因为比较它们时必须
// 知道后缀的存在：同一个数字段上，`0.3.0-rc1` 比 `0.3.0` **旧** —— 否则已经装了正式版
// 的人会被劝去"升级"到更早的候选版。
type Version struct {
	Major, Minor, Patch int
	Pre                 string // 预发布后缀（不含前导 -、+）；空 = 正式版
}

// versionRe 三段数字 + 可选后缀，容忍 tag 上的 v 前缀。
var versionRe = regexp.MustCompile(`^v?([0-9]+)\.([0-9]+)\.([0-9]+)(?:[-+]([0-9A-Za-z.-]+))?$`)

// ParseVersion 解析版本号。空串是错误：调用方拿到空串应该先想清楚"这份二进制到底
// 有没有版本号"，而不是让它悄悄参与比较（本地 go build 的 Version 就是空的）。
func ParseVersion(s string) (Version, error) {
	m := versionRe.FindStringSubmatch(s)
	if m == nil {
		return Version{}, fmt.Errorf("版本号 %q 不是 MAJOR.MINOR.PATCH（可带 v 前缀与预发布后缀）", s)
	}
	var v Version
	for i, dst := range []*int{&v.Major, &v.Minor, &v.Patch} {
		n, err := strconv.Atoi(m[i+1])
		if err != nil {
			return Version{}, fmt.Errorf("版本号 %q 的第 %d 段不是数字: %w", s, i+1, err)
		}
		*dst = n
	}
	v.Pre = m[4]
	return v, nil
}

// String 还原成不带 v 前缀的形态（与 VERSION 文件一致，便于直接比较打印结果）。
func (v Version) String() string {
	s := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.Pre != "" {
		s += "-" + v.Pre
	}
	return s
}

// Compare 返回 -1 / 0 / 1：v 比 o 旧 / 相同 / 更新。
func (v Version) Compare(o Version) int {
	for _, d := range [][2]int{{v.Major, o.Major}, {v.Minor, o.Minor}, {v.Patch, o.Patch}} {
		switch {
		case d[0] < d[1]:
			return -1
		case d[0] > d[1]:
			return 1
		}
	}
	// 数字段相同：带后缀的比正式的旧（semver 的规矩，也是"已装正式版别被劝装 rc"
	// 所需要的）。两个后缀之间按字典序 —— 只要确定、可重现，不追求 semver 的逐段规则。
	switch {
	case v.Pre == o.Pre:
		return 0
	case v.Pre == "":
		return 1
	case o.Pre == "":
		return -1
	default:
		return strings.Compare(v.Pre, o.Pre)
	}
}

// NewerThan 报告 v 是否比 o 新。
func (v Version) NewerThan(o Version) bool { return v.Compare(o) > 0 }
