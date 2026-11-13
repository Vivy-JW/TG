package dbsync

// GP 字典库的数据族：本地 t_* 表按"哪条命令要用"分成五组。
//
// 这张表三处共用：tt dict db status 的覆盖检查、各命令 --help 末尾的
// 「本地数据」提示、以及同步完后的自检。TG 只支持 TOP GP，所以这里只有
// GP 的表名 —— 原来那套 dz*/gz* 的 T100 族已经删掉了（TG 不会再同步 T100 字典）。

// Family 一个数据族：Key 是内部代号，Name 给人看，Tables 是本地表名，
// Commands 列出依赖它的命令（status 里显示"缺了会影响什么"）。
type Family struct {
	Key      string   `json:"key"`
	Name     string   `json:"name"`
	Tables   []string `json:"tables"`
	Commands []string `json:"commands,omitempty"`
}

// GPFamilies 本地 GP 字典库的五个数据族，顺序即 tg dict db status 的显示顺序。
//
// t_prog_link 由同步写（源 gal_file），归在"程序与模块"一族：入口模块那条链既可以从
// t_prog.module 读单值，也可以从 t_prog_link 读"一个作业链了哪些程序"。
var GPFamilies = []Family{
	{
		Key:    "table",
		Name:   "表字典",
		Tables: []string{"t_table", "t_column", "t_index", "t_constraint"},
		Commands: []string{
			"tg dict r.t", "tg dict r.t <表> --fields", "tg dict prog <程序>",
		},
	},
	{
		Key:      "prog",
		Name:     "程序与模块",
		Tables:   []string{"t_prog", "t_module", "t_prog_link"},
		Commands: []string{"tg dict prog", "tg dict prog <程序> --links"},
	},
	{
		Key:      "progtable",
		Name:     "程序用表",
		Tables:   []string{"t_prog_table"},
		Commands: []string{"tg dict prog <程序> 的表格总数", "tg dict r.t <表> 的程序清单"},
	},
	{
		Key:      "msg",
		Name:     "错误消息",
		Tables:   []string{"t_msg"},
		Commands: []string{"tg dict msg"},
	},
	{
		Key:      "per",
		Name:     "画面文字",
		Tables:   []string{"t_ui_text"},
		Commands: []string{"tg interface show", "tg interface labels"},
	},
}

// FamilyByKey 按 Key 找族；找不到返回 false。
func FamilyByKey(key string) (Family, bool) {
	for _, f := range GPFamilies {
		if f.Key == key {
			return f, true
		}
	}
	return Family{}, false
}

// FamilyKeys 全部族的 Key，按声明顺序。
func FamilyKeys() []string {
	out := make([]string, 0, len(GPFamilies))
	for _, f := range GPFamilies {
		out = append(out, f.Key)
	}
	return out
}
