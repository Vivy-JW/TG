package db

import "strings"

// LangCode 把界面语言别翻成 GP 字典表里的语言码。
//
// TOP GP 的字典型表(lookup 表)用数字写语言：0 = 繁體中文，2 = 简体中文
// (gay_file 只有这两行)。命令层的 --lang 是 T100 形态的 zh_TW / zh_CN，
// 到 SQL 前必须转一次，否则 ze02 = 'zh_CN' 永远查不到行。
//
// 已经是数字码的原样返回；空串返回空串(表示"不限语言")；认不出的语言别
// 退到简体——GP 客户端默认就是简体，退到它最不容易让人看空结果。
func LangCode(lang string) string {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "":
		return ""
	case "zh_tw", "zhtw", "tw", "big5", "0":
		return "0"
	case "zh_cn", "zhcn", "cn", "gbk", "2":
		return "2"
	default:
		return "2"
	}
}
