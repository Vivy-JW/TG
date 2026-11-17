package config

import "strings"

// PathSafeSeg 把任意字符串压成单个安全的路径段。
//
// 保留中文与常见符号（环境名可能是「示例测试区」），只替换分隔/通配类字符；
// 空、`.`、`..` 一律压成 `_`，保证拼出来的路径逃不出根目录。
//
// 凡是「按环境名落盘」的目录都走它：<数据目录>/srccache/<环境段>/ 这类
// 缓存镜像、以及改环境名时的迁移计算 —— 两处必须同一套规则，否则改名之后
// 会算不中原来的目录，留下一堆孤儿。
func PathSafeSeg(s string) string {
	out := strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|', 0:
			return '_'
		}
		return r
	}, s)
	out = strings.TrimSpace(out)
	if out == "" || out == "." || out == ".." {
		return "_"
	}
	return out
}

// EnvSeg 目录名里的环境段：优先环境名，退化为 主机-区域。
//
// 环境名是给人看的、可能是中文，所以只能经 PathSafeSeg 清洗后当路径用；
// 老配置可能没有名字，用「主机-区域」兜底（两个区同名文件时不会撞在一起）。
func EnvSeg(envName, sshHost, zone string) string {
	s := strings.TrimSpace(envName)
	if s == "" {
		s = sshHost
		if zone != "" {
			s += "-" + zone
		}
	}
	return PathSafeSeg(s)
}
