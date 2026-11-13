package db

import "strings"

// Source 是 ERP 数据字典"语义查询"的统一数据源接口:本地 SQLite 镜像(*DB)
// 与远程 ERP 库直查(live.Live,金仓/人大金仓 + Oracle)实现同一组查询方法,
// 返回同一批 DTO,rt/rv/desc/scc/rq 五个查询命令只依赖本接口 —— 查询的表
// (dzea_t~dzeg_t/dzcd~dzch/dzep/gzca/gzcb/dzca~dzcc 等 24 张)在三个后端
// 都是同一套,数据源由配置/CLI 决定,命令层不再区分本地与远程。
type Source interface {
	// rt 表字典:表档 dzea_t / 字段档 dzeb_t / 键值档 dzed_t / 索引档 dzec_t
	QueryTableMeta(tableName string) (*TableMeta, error) // 未收录返回 (nil, nil)
	QueryTable(tableName string) ([]TableInfo, error)
	QueryKeys(tableName string) ([]KeyInfo, error)
	QueryIndexes(tableName string) ([]IndexInfo, error)
	QueryTableList(lang, keyword string) ([]TableListItem, error) // 列表/搜索(--kw)
	// prog 程序字典:程序目录 gaz_file(+zz_file 类别、gal_file 模块)同步成 t_prog；作业与程序在 GP 是同一个东西
	// / 应用参数组 gzzk_t。见 db/prog.go 注释里的表结构说明。
	QueryProgInfo(code, lang string) (*ProgInfo, error) // 未收录返回 (nil, nil)
	QueryProgJobs(code, lang string) ([]ProgJob, error)
	QueryProgList(lang, keyword string) ([]ProgListItem, error)
	// 程序 ↔ 表格:zr_file 同步成 t_prog_table,两侧都按操作类别合并
	QueryProgTables(code, lang string) ([]ProgTableRow, error)
	QueryTablePrograms(table, lang string) ([]TableProgRow, error)
	// 程序链接:gal_file 同步成 t_prog_link(程序 → 模块),入口行是 gal03 == gal01。
	QueryProgLinks(prog string) ([]ProgLinkRow, error)

	// 画面元件译文(gae_file,p_perlang 维护):GP 没有 .42s,画面文字运行期从这张表取。
	QueryUITexts(prog, lang string) ([]UIElementText, error)

	// rq 可复用开窗:gab_file(单头) + gac_file(栏位) + gae_file/gaq_file(标题与显示名)
	QueryWinList(lang, keyword string) ([]WinListRow, error)
	QueryWinHeaders(id, lang string) ([]WinHeaderRow, error)
	QueryWinCols(id, lang string) ([]WinColRow, error)
	// msg 系统消息档:ze_file 同步成 t_msg(按编号/语句/语言多条件查)
	//
	// 一次调用就够:没有命中时 MsgResult.Langs 带上"同条件去掉语言"的可用语言,
	// 命令层不必再问一次(从前是 QueryMsgs + QueryMsgLangs 两个方法)。
	QueryMsgs(q MsgQuery) (MsgResult, error)

	// Close 释放底层连接(SQLite 文件句柄 / 远程连接与隧道)
	Close()
}

// IsMissingTable 判断错误是否源自"字典表/视图不存在",兼容三种后端的错误
// 文本:SQLite "no such table"、金仓/PG "relation ... does not exist"、
// Oracle "ORA-00942"。命令层用它把键值/索引/参数等附表缺失降级为"无"，
// 不中断主查询(与本地缺表容忍语义对齐)。
func IsMissingTable(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	for _, kw := range []string{
		"no such table", "does not exist", "ora-00942", "not found", "不存在",
	} {
		if strings.Contains(s, kw) {
			return true
		}
	}
	return false
}
