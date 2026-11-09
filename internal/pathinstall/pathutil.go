package pathinstall

import "strings"

// PATH 列表的纯字符串操作(不触碰系统状态,便于测试)。
// sep 为平台分隔符(Windows ";",其它 ":");fold=true 时目录比较忽略大小写。

// splitPathList 按 sep 拆分 PATH 列表,去掉空白与空项。
func splitPathList(list, sep string) []string {
	var out []string
	for _, p := range strings.Split(list, sep) {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// joinPathList 以 sep 拼回 PATH 列表。
func joinPathList(list []string, sep string) string { return strings.Join(list, sep) }

// samePathEntry 判断两个 PATH 项是否指同一目录(忽略尾部分隔符;fold 时忽略大小写)。
func samePathEntry(a, b string, fold bool) bool {
	a, b = strings.TrimRight(strings.TrimSpace(a), `\/`), strings.TrimRight(strings.TrimSpace(b), `\/`)
	if fold {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// pathListContains 列表是否已含 dir。
func pathListContains(list []string, dir string, fold bool) bool {
	for _, p := range list {
		if samePathEntry(p, dir, fold) {
			return true
		}
	}
	return false
}

// pathListAdd 追加 dir(已存在则原样返回,幂等)。
func pathListAdd(list []string, dir string, fold bool) []string {
	if dir == "" || pathListContains(list, dir, fold) {
		return list
	}
	return append(list, dir)
}

// pathListRemove 移除 dir(不存在则原样返回)。
func pathListRemove(list []string, dir string, fold bool) []string {
	out := make([]string, 0, len(list))
	for _, p := range list {
		if !samePathEntry(p, dir, fold) {
			out = append(out, p)
		}
	}
	return out
}
