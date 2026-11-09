package erpdb

import "strings"

// ValidIdent 校验 SQL 标识符是否安全(字母/数字/下划线,不以数字开头)。
// 在线查询的表名/字段名一律先过此校验,防止注入。
func ValidIdent(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_') {
			return false
		}
	}
	return !(s[0] >= '0' && s[0] <= '9')
}

// QuoteIdent 返回可直接拼入 SQL 的标识符(白名单校验;不过校验返回空)。
func QuoteIdent(s string) string {
	if !ValidIdent(s) {
		return ""
	}
	return s
}

// QuoteLit 把字符串转义成 SQL 单引号字面量(单引号翻倍),供内联参数安全拼接。
func QuoteLit(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}
