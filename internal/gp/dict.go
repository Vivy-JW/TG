package gp

// GP 字典同步（Go 实现）：Oracle 的 zta_file / gat_file / gaq_file / gaz_file …
// 拉进本地 SQLite（<dictDir>/<环境>.db），供离线查询。
//
// SQL 与表结构照搬 ttgp 那套 Python 实现：两边共用同一份 .db，
// 谁写的都能被另一边读，迁移期间不会出现"两套库"。

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/sijms/go-ora/v2"
	_ "modernc.org/sqlite"

	"tt/internal/config"
	"tt/internal/dbconfig"
)

const gpSchemaSQL = `
CREATE TABLE IF NOT EXISTS meta(key TEXT PRIMARY KEY, value TEXT);
CREATE TABLE IF NOT EXISTS t_table(
  tab TEXT PRIMARY KEY, tab_cn TEXT, module TEXT, tab_type TEXT,
  col_count INTEGER, remark TEXT, raw TEXT);
CREATE TABLE IF NOT EXISTS t_column(
  tab TEXT, seq INTEGER, col TEXT, col_cn TEXT, dtype TEXT,
  dlen INTEGER, dprec INTEGER, dscale INTEGER, nullable TEXT,
  dflt TEXT, remark TEXT, fmt TEXT, PRIMARY KEY(tab, col));
CREATE INDEX IF NOT EXISTS ix_col_tab ON t_column(tab);
CREATE INDEX IF NOT EXISTS ix_col_cn  ON t_column(col_cn);
CREATE INDEX IF NOT EXISTS ix_tab_cn  ON t_table(tab_cn);
CREATE TABLE IF NOT EXISTS t_index(
  tab TEXT, idx TEXT, uniqueness TEXT, cols TEXT, PRIMARY KEY(tab, idx));
CREATE TABLE IF NOT EXISTS t_constraint(
  tab TEXT, cons TEXT, ctype TEXT, status TEXT, validated TEXT, cols TEXT,
  PRIMARY KEY(tab, cons));
CREATE TABLE IF NOT EXISTS t_module(
  module TEXT PRIMARY KEY, form_dir TEXT, rep_dir TEXT);
CREATE TABLE IF NOT EXISTS t_prog(
  prog TEXT PRIMARY KEY, name TEXT, module TEXT, memo TEXT, name2 TEXT);
CREATE INDEX IF NOT EXISTS ix_prog_name ON t_prog(name);
CREATE TABLE IF NOT EXISTS t_prog_link(
  prog TEXT, module TEXT, linked_prog TEXT, link_flag TEXT,
  PRIMARY KEY(prog, module, linked_prog));
CREATE INDEX IF NOT EXISTS ix_prog_link_prog ON t_prog_link(prog);
`

const (
	sqlGPTables = `
SELECT z.zta01, z.zta02, z.zta03, z.zta09,
       g.gat03, g.gat06, g.gat07
  FROM zta_file z
  LEFT JOIN gat_file g ON g.gat01 = z.zta01 AND g.gat02 = '2'
 WHERE z.zta02 = :1
 ORDER BY z.zta01`

	sqlGPCols = `
SELECT lower(c.table_name), c.column_id, lower(c.column_name),
       lower(c.data_type), c.data_length, c.data_precision,
       c.data_scale, c.nullable, c.data_default
  FROM all_tab_columns c
 WHERE upper(c.owner) = upper(:1)
 ORDER BY c.table_name, c.column_id`

	sqlGPGAQ = `SELECT gaq01, gaq03, gaq05, gaq07 FROM gaq_file WHERE gaq02 = '2'`

	// 索引：Oracle 的 all_indexes + all_ind_columns。按索引聚合字段顺序，
	// 本地 t_index 的 cols 用逗号分隔；uniqueness 保留 UNIQUE/NONUNIQUE。
	sqlGPIndexes = `
SELECT lower(i.table_name), lower(i.index_name), i.uniqueness,
       LISTAGG(lower(c.column_name), ',') WITHIN GROUP (ORDER BY c.column_position)
  FROM all_indexes i
  JOIN all_ind_columns c
    ON c.index_owner = i.owner AND c.index_name = i.index_name
 WHERE upper(i.owner) = upper(:1)
 GROUP BY lower(i.table_name), lower(i.index_name), i.uniqueness
 ORDER BY lower(i.table_name), lower(i.index_name)`

	// 约束：all_constraints + all_cons_columns。ctype 保留 Oracle 原码
	// （P 主键 / U 唯一 / R 外键 / C 检查 / O 只读），status/validated 也保留原值。
	sqlGPConstraints = `
SELECT lower(c.table_name), lower(c.constraint_name), c.constraint_type,
       c.status, c.validated,
       LISTAGG(lower(cc.column_name), ',') WITHIN GROUP (ORDER BY cc.position)
  FROM all_constraints c
  LEFT JOIN all_cons_columns cc
    ON cc.owner = c.owner AND cc.constraint_name = c.constraint_name
 WHERE upper(c.owner) = upper(:1)
 GROUP BY lower(c.table_name), lower(c.constraint_name), c.constraint_type,
          c.status, c.validated
 ORDER BY lower(c.table_name), lower(c.constraint_name)`

	sqlGPProg = `
SELECT g.gaz01, g.gaz03, g.gaz06, g.gaz04,
       COALESCE(
         (SELECT MAX(l.gal02) FROM gal_file l
           WHERE l.gal01 = g.gaz01 AND lower(l.gal03) = lower(g.gaz01)),
         (SELECT MAX(l.gal02) FROM gal_file l WHERE l.gal01 = g.gaz01)
       )
  FROM gaz_file g
 WHERE g.gaz02 = '2'
 ORDER BY g.gaz01`

	// 说明：不在同步时把 gal_file 的所有链接行拉进 t_prog_link。
	// gal_file 很大，全表扫描会让 50 秒的同步卡到数分钟；入口模块已经由
	// sqlGPProg 里“gal03 == gaz01”的精确子查询写进 t_prog，足够启动使用。
	// t_prog_link 保留给以后需要做“一个作业多程序”的完整视图时再按需填充。

	sqlGPModule = `SELECT gao01, gao02, gao04 FROM gao_file`
)

// DictPath 返回本地 GP 字典库路径。
// 环境可单独指定 DictDir，否则落在 gp 数据目录下。
//
// 导出给 CLI 的 tt gp table 用：Web 的字典同步与 CLI 查询必须落到同一个
// <环境>.db，路径规则只能有一份实现。
func DictPath(e config.NamedSsh) string { return dictPath(e) }

// dictPath 本地字典库路径。环境可单独指定 DictDir，否则落在 gp 数据目录下。
func dictPath(e config.NamedSsh) string {
	dir := e.DictDir
	if dir == "" {
		dir = os.Getenv("TT_GP_DICT_DIR")
	}
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".tt", "gp-dict")
	}
	return filepath.Join(dir, e.Name+".db")
}

func gpDSN(db *dbconfig.Connection, account, password string) string {
	return fmt.Sprintf("oracle://%s:%s@%s:%d/%s",
		url.QueryEscape(account), url.QueryEscape(password), db.Host, db.Port, db.Service)
}

func clean(s sql.NullString) string { return strings.TrimSpace(s.String) }

// runSync 把 GP 字典整份同步到本地 SQLite。
func (s *Service) runSync(env config.NamedSsh) {
	fail := func(err error) {
		s.setJob(env.Name, func(j *Job) {
			j.State = "error"
			j.Error = err.Error()
			j.Ended = time.Now()
		})
	}
	if env.DB == nil || env.DB.Host == "" {
		fail(fmt.Errorf("这个环境没配数据库"))
		return
	}
	var account, password string
	for _, a := range env.DB.Accounts {
		account, password = a.Account, a.Password
		if account != "" {
			break
		}
	}
	if account == "" {
		fail(fmt.Errorf("这个环境的数据库没有账号"))
		return
	}

	s.progress(env.Name, "连接 "+env.DB.Host+":"+fmt.Sprint(env.DB.Port)+"/"+env.DB.Service+" …")
	oc, err := sql.Open("oracle", gpDSN(env.DB, account, password))
	if err != nil {
		fail(err)
		return
	}
	defer oc.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	if err := oc.PingContext(ctx); err != nil {
		fail(fmt.Errorf("连不上数据库：%w", err))
		return
	}
	s.progress(env.Name, "已连上（账号 "+account+"）")

	lib, owner := "ds", "DS"
	if env.DB.Service != "" {
		owner = "DS"
	}

	// 表字典
	tables := map[string]struct{ module, cn string }{}
	rows, err := oc.QueryContext(ctx, sqlGPTables, lib)
	if err != nil {
		fail(fmt.Errorf("查 zta_file 失败：%w", err))
		return
	}
	type tabRow struct{ tab, cn, module, ttype, remark string }
	var tabs []tabRow
	for rows.Next() {
		var tab, libv, module, ttype sql.NullString
		var cn, gat06, gat07 sql.NullString
		if err := rows.Scan(&tab, &libv, &module, &ttype, &cn, &gat06, &gat07); err != nil {
			continue
		}
		t := strings.ToLower(clean(tab))
		if t == "" {
			continue
		}
		mod := clean(module)
		if mod == "" {
			mod = clean(gat06)
		}
		tt := clean(ttype)
		if tt == "" {
			tt = clean(gat07)
		}
		tables[t] = struct{ module, cn string }{mod, clean(cn)}
		tabs = append(tabs, tabRow{t, clean(cn), mod, tt, ""})
	}
	rows.Close()
	s.progress(env.Name, fmt.Sprintf("表字典 zta_file：%d 张", len(tabs)))
	s.setJob(env.Name, func(j *Job) {
		j.Done = 0
		j.Total = int64(len(tabs))
	})

	// 字段中文说明（全局，按字段名贴）
	gaq := map[string]struct{ cn, remark, fmtv string }{}
	if r2, err := oc.QueryContext(ctx, sqlGPGAQ); err == nil {
		for r2.Next() {
			var name, cn, remark, fmtv sql.NullString
			if r2.Scan(&name, &cn, &remark, &fmtv) != nil {
				continue
			}
			gaq[strings.ToLower(clean(name))] = struct{ cn, remark, fmtv string }{clean(cn), clean(remark), clean(fmtv)}
		}
		r2.Close()
	}

	// 字段（只留字典里登记过的表）
	type colRow struct {
		tab, col, cn, dtype, nullable, dflt, remark, fmtv string
		seq, dlen, dprec, dscale                          int
	}
	var cols []colRow
	r3, err := oc.QueryContext(ctx, sqlGPCols, owner)
	if err != nil {
		fail(fmt.Errorf("查 all_tab_columns 失败：%w", err))
		return
	}
	for r3.Next() {
		var tab, col, dtype, nullable, dflt sql.NullString
		var seq, dlen, dprec, dscale sql.NullInt64
		if r3.Scan(&tab, &seq, &col, &dtype, &dlen, &dprec, &dscale, &nullable, &dflt) != nil {
			continue
		}
		t := strings.ToLower(clean(tab))
		if _, ok := tables[t]; !ok {
			continue
		}
		c := strings.ToLower(clean(col))
		g := gaq[c]
		cols = append(cols, colRow{
			tab: t, col: c, cn: g.cn, dtype: clean(dtype),
			nullable: clean(nullable), dflt: clean(dflt), remark: g.remark, fmtv: g.fmtv,
			seq: int(seq.Int64), dlen: int(dlen.Int64), dprec: int(dprec.Int64), dscale: int(dscale.Int64),
		})
	}
	r3.Close()
	s.progress(env.Name, fmt.Sprintf("字段：%d 个", len(cols)))

	// 索引 + 约束：只保留字典里登记过的表；与 T100 一样把索引/约束当作表结构的一部分。
	type idxRow struct{ tab, idx, uniq, cols string }
	var indexes []idxRow
	r6, err := oc.QueryContext(ctx, sqlGPIndexes, owner)
	if err != nil {
		fail(fmt.Errorf("查 all_indexes 失败：%w", err))
		return
	}
	for r6.Next() {
		var tab, idx, uniq, cols sql.NullString
		if r6.Scan(&tab, &idx, &uniq, &cols) != nil {
			continue
		}
		t := strings.ToLower(clean(tab))
		if _, ok := tables[t]; !ok {
			continue
		}
		indexes = append(indexes, idxRow{t, strings.ToLower(clean(idx)), clean(uniq), strings.ToLower(clean(cols))})
	}
	r6.Close()
	s.progress(env.Name, fmt.Sprintf("索引：%d 条", len(indexes)))

	type consRow struct{ tab, cons, ctype, status, validated, cols string }
	var constraints []consRow
	r7, err := oc.QueryContext(ctx, sqlGPConstraints, owner)
	if err != nil {
		fail(fmt.Errorf("查 all_constraints 失败：%w", err))
		return
	}
	for r7.Next() {
		var tab, cons, ctype, status, validated, cols sql.NullString
		if r7.Scan(&tab, &cons, &ctype, &status, &validated, &cols) != nil {
			continue
		}
		t := strings.ToLower(clean(tab))
		if _, ok := tables[t]; !ok {
			continue
		}
		constraints = append(constraints, consRow{
			t, strings.ToLower(clean(cons)), clean(ctype), clean(status), clean(validated), strings.ToLower(clean(cols)),
		})
	}
	r7.Close()
	s.progress(env.Name, fmt.Sprintf("约束：%d 条", len(constraints)))

	// 写 SQLite（先写临时文件，完成后原子替换，避免中途失败留下半个库）
	path := dictPath(env)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		fail(err)
		return
	}
	tmp := path + ".tmp"
	_ = os.Remove(tmp)
	lc, err := sql.Open("sqlite", tmp)
	if err != nil {
		fail(err)
		return
	}
	if _, err := lc.Exec(gpSchemaSQL); err != nil {
		lc.Close()
		fail(fmt.Errorf("建 SQLite 失败：%w", err))
		return
	}
	tx, _ := lc.Begin()
	if tx != nil {
		st, _ := tx.Prepare(`INSERT OR REPLACE INTO t_table(tab,tab_cn,module,tab_type,col_count,remark,raw) VALUES(?,?,?,?,?,?,?)`)
		if st != nil {
			for i, t := range tabs {
				raw, _ := json.Marshal(t)
				_, _ = st.Exec(t.tab, t.cn, t.module, t.ttype, 0, t.remark, string(raw))
				s.setJob(env.Name, func(j *Job) { j.Done = int64(i + 1) })
			}
			st.Close()
		}
		stc, _ := tx.Prepare(`INSERT OR REPLACE INTO t_column(tab,seq,col,col_cn,dtype,dlen,dprec,dscale,nullable,dflt,remark,fmt) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`)
		if stc != nil {
			for _, c := range cols {
				_, _ = stc.Exec(c.tab, c.seq, c.col, c.cn, c.dtype, c.dlen, c.dprec, c.dscale, c.nullable, c.dflt, c.remark, c.fmtv)
			}
			stc.Close()
		}
		sti, _ := tx.Prepare(`INSERT OR REPLACE INTO t_index(tab,idx,uniqueness,cols) VALUES(?,?,?,?)`)
		if sti != nil {
			for _, ix := range indexes {
				_, _ = sti.Exec(ix.tab, ix.idx, ix.uniq, ix.cols)
			}
			sti.Close()
		}
		stk, _ := tx.Prepare(`INSERT OR REPLACE INTO t_constraint(tab,cons,ctype,status,validated,cols) VALUES(?,?,?,?,?,?)`)
		if stk != nil {
			for _, cs := range constraints {
				_, _ = stk.Exec(cs.tab, cs.cons, cs.ctype, cs.status, cs.validated, cs.cols)
			}
			stk.Close()
		}
		_, _ = tx.Exec(`UPDATE t_table SET col_count = (SELECT COUNT(*) FROM t_column c WHERE c.tab = t_table.tab)`)
		_, _ = tx.Exec(`INSERT OR REPLACE INTO meta(key,value) VALUES('synced_at', ?)`, time.Now().Format("2006-01-02 15:04:05"))
		_, _ = tx.Exec(`INSERT OR REPLACE INTO meta(key,value) VALUES('indexes', ?)`, fmt.Sprint(len(indexes)))
		_, _ = tx.Exec(`INSERT OR REPLACE INTO meta(key,value) VALUES('constraints', ?)`, fmt.Sprint(len(constraints)))
		_, _ = tx.Exec(`INSERT OR REPLACE INTO meta(key,value) VALUES('owner', ?)`, owner)
		_, _ = tx.Exec(`INSERT OR REPLACE INTO meta(key,value) VALUES('tables', ?)`, fmt.Sprint(len(tabs)))
		_ = tx.Commit()
	}

	// 作业目录 + 模块（失败不影响主流程）
	if r4, err := oc.QueryContext(ctx, sqlGPProg); err == nil {
		stp, _ := lc.Prepare(`INSERT OR REPLACE INTO t_prog(prog,name,module,memo,name2) VALUES(?,?,?,?,?)`)
		if stp != nil {
			n := 0
			for r4.Next() {
				var prog, name, name2, memo, module sql.NullString
				if r4.Scan(&prog, &name, &name2, &memo, &module) != nil {
					continue
				}
				_, _ = stp.Exec(strings.ToLower(clean(prog)), clean(name), strings.ToLower(clean(module)), clean(memo), clean(name2))
				n++
			}
			stp.Close()
			s.progress(env.Name, fmt.Sprintf("作业目录 gaz_file：%d 条", n))
		}
		r4.Close()
	}
	if r5, err := oc.QueryContext(ctx, sqlGPModule); err == nil {
		stm, _ := lc.Prepare(`INSERT OR REPLACE INTO t_module(module,form_dir,rep_dir) VALUES(?,?,?)`)
		if stm != nil {
			for r5.Next() {
				var m, f, rp sql.NullString
				if r5.Scan(&m, &f, &rp) != nil {
					continue
				}
				_, _ = stm.Exec(strings.ToLower(clean(m)), clean(f), clean(rp))
			}
			stm.Close()
		}
		r5.Close()
	}
	lc.Close()

	_ = os.Remove(path)
	if err := os.Rename(tmp, path); err != nil {
		fail(err)
		return
	}
	st, _ := os.Stat(path)
	mb := 0.0
	if st != nil {
		mb = float64(st.Size()) / 1048576
	}
	s.progress(env.Name, fmt.Sprintf("已写入 %s（%.1f MB）", path, mb))
	s.setJob(env.Name, func(j *Job) {
		j.State = "done"
		j.Done = int64(len(tabs))
		j.Total = int64(len(tabs))
		j.Result = map[string]any{
			"path": path, "tables": len(tabs), "columns": len(cols),
			"indexes": len(indexes), "constraints": len(constraints), "sizeMB": mb,
		}
		j.Ended = time.Now()
	})
}

// dictStat 读本地字典库的文件状态（存在/大小），与 T100 的 dbSyncView 同一口径：
// 只 os.Stat，不打开 SQLite。表数是数据字典页按需查询的信息，不该挂在环境清单上。
func dictStat(e config.NamedSsh) (exists bool, sizeMB float64) {
	path := dictPath(e)
	st, err := os.Stat(path)
	if err != nil {
		return false, 0
	}
	return true, float64(st.Size()) / 1048576
}
