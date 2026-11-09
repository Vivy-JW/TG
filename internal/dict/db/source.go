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
	// prog 程序与作业字典:程序档 gzza_t / 程序名称 gzzal_t / 作业 zzz_t(作业挂程序)
	// / 应用参数组 gzzk_t。见 db/prog.go 注释里的表结构说明。
	QueryProgInfo(code, lang string) (*ProgInfo, error) // 未收录返回 (nil, nil)
	QueryProgJobs(code, lang string) ([]ProgJob, error)
	QueryProgList(lang, keyword string) ([]ProgListItem, error)
	// 程序 ↔ 表格:gzdg_t(程序与应用表格功能分析表,参考作业 azzq902),两侧都按操作类别合并
	QueryProgTables(code, lang string) ([]ProgTableRow, error)
	QueryTablePrograms(table, lang string) ([]TableProgRow, error)
	// 子程序与元件:gzde_t(子程序及应用元件基本数据表,参考作业 azzi901)+ gzdel_t(说明)
	QuerySubProgInfo(code, lang string) (*SubProgInfo, error) // 未收录返回 (nil, nil)
	QuerySubProgList(lang, keyword string) ([]SubProgListItem, error)
	// rv 校验带值:dzcd_t/dzce_t/dzch_t
	QueryCheckList(lang, keyword string) ([]CheckListRow, error)
	QueryCheckHeaders(id, lang string) ([]CheckHeaderRow, error)
	QueryCheckParams(id, lang string) ([]CheckParamRow, error)
	QueryCheckConds(id string) ([]CheckCondRow, error)
	// desc 字段规格:dzep_t(附 dzeb_t/dzebl_t 名称)
	QuerySpecRows(tableName, lang string) ([]SpecRow, error)
	// scc 系统分类码:gzca_t/gzcb_t
	QuerySccList(lang, keyword string) ([]SccListRow, error)
	QuerySccHeader(id, lang string) (*SccHeaderRow, error) // 未命中返回 (nil, nil)
	QuerySccValues(id, lang string) ([]SccValueRow, error)
	// 应用分类码 ACC：定义在 gzaa/gzac/gzad，作业注册在 gzzz，值在 oocq/oocql。
	QueryACCMeta(input, lang string) (*ACCMeta, error) // 未命中返回 (nil, nil)
	QueryACCValues(category string, ent int, lang string, includeInactive bool) ([]ACCValue, error)
	// rq 可复用开窗:dzca_t/dzcb_t/dzcc_t
	QueryWinList(lang, keyword string) ([]WinListRow, error)
	QueryWinHeaders(id, lang string) ([]WinHeaderRow, error)
	QueryWinParams(id, lang string) ([]WinParamRow, error)
	QueryWinCols(id string) ([]WinColRow, error)
	// msg 系统消息档:gzze_t(azzi920 维护;按编号/语句/语言多条件查)+ gzzal_t 作业名称
	QueryMsgs(q MsgQuery) ([]MsgRow, error)
	QueryMsgLangs(q MsgQuery) ([]string, error) // 同条件去掉语言:该编号有哪些语言行
	// param 参数定义档:gzsz_t/gzszl_t(azzi990 系统参数、azzi991 单据别参数
	// 共用;全语言行)+ gzsy_t 单据性质绑定
	QueryParam(code string) ([]ParamDefRow, error)
	QueryDocTypes(code, lang string) ([]DocTypeRow, error)
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
