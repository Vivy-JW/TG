package dict

// pickLangRow 从"每语言一行"的结果里挑出指定语言行(与源系统精确 (键,语言)
// 匹配、无自动回退的策略一致)。未命中时返回该键可用语言清单(供提示 --lang)。
func pickLangRow[T any](rows []T, lang string, langOf func(T) string) (*T, []string) {
	for i := range rows {
		if langOf(rows[i]) == lang {
			return &rows[i], nil
		}
	}
	seen := map[string]bool{}
	var avail []string
	for i := range rows {
		l := langOf(rows[i])
		if !seen[l] {
			seen[l] = true
			avail = append(avail, l)
		}
	}
	return nil, avail
}

// langMissMsg 无指定语言行时的提示(可用语言清单)。
func langMissMsg(key, lang string, avail []string) string {
	return "未找到 '" + key + "' 的 " + lang + " 语言行;可用语言: " + joinAvail(avail) + "(用 --lang 指定)"
}

func joinAvail(avail []string) string {
	s := ""
	for i, a := range avail {
		if i > 0 {
			s += ", "
		}
		s += a
	}
	return s
}
