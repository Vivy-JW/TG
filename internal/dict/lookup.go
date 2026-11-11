package dict

import (
	"database/sql"
	"os"
	"strings"

	"tt/internal/config"
)

// LookupModule 从本地 GP 字典里查“程序 → 实际模块”。
//
// 优先查 t_prog_link：p_link 的 gal_file 是一对多，只有 linked_prog 等于
// 当前程序名的那一行才是入口模块。不能像旧实现那样用 MAX(gal02) 压成一条——
// csfi300_icd 会误选到 CSUB（它其实是 cs_icdpost 等子程序的模块），而不是 CSF。
//
// 旧库没有 t_prog_link 时退回 t_prog.module；返回 ok=false 表示本地库不存在、
// 还没同步，或该程序没有 p_link 记录。
func LookupModule(env config.NamedSsh, prog string) (module string, ok bool) {
	prog = strings.TrimSpace(prog)
	if prog == "" {
		return "", false
	}
	path := dictPath(env)
	if _, err := os.Stat(path); err != nil {
		return "", false
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return "", false
	}
	defer db.Close()

	// 先按用户输入查；再兼容“带模块前缀”的写法（cbm_abmi600 → abmi600）。
	candidates := []string{prog}
	if len(prog) > 4 && prog[3] == '_' {
		candidates = append(candidates, prog[4:])
	}
	for _, p := range candidates {
		// 精确匹配入口程序：gal03 == gal01。
		var mod string
		err := db.QueryRow(
			`SELECT module FROM t_prog_link
			  WHERE lower(prog) = lower(?) AND lower(linked_prog) = lower(?)
			    AND trim(module) <> ''
			  ORDER BY module LIMIT 1`, p, p,
		).Scan(&mod)
		if err == nil && strings.TrimSpace(mod) != "" {
			return strings.ToLower(strings.TrimSpace(mod)), true
		}
	}
	// 旧库或没有精确入口行时，用 t_prog 的单值兜底。
	for _, p := range candidates {
		var mod string
		err := db.QueryRow(
			`SELECT module FROM t_prog WHERE lower(prog) = lower(?) LIMIT 1`, p,
		).Scan(&mod)
		if err == nil && strings.TrimSpace(mod) != "" {
			return strings.ToLower(strings.TrimSpace(mod)), true
		}
	}
	return "", false
}
