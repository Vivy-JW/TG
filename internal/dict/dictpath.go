// Package dict 是 TOP GP 的字典层：本地字典库的位置口径、营运据点解析、
// 开窗 SQL 拼装、程序查模块。
//
// 这些东西原来挤在一个包里：那个包既管 /api/gp/* 的设置页与会话，又管字典。
// 依赖方向能说明问题：它会 import internal/dict/live 才能查库，而 internal/cli/dict
// 又反过来 import 它 —— 两头互相要。现在字典层自成一体，执行面在 internal/web。
// 分层说明见 docs/gp-scope.md。
package dict

import (
	"os"
	"path/filepath"
	"strings"

	"tg/internal/config"
)

// SyncCompany 是本地字典库**固定**取自的据点代号。
//
// ds 是 GP 的系统示范据点：zta_file 里 2,962 张表（另一家 2,902），
// 表档/字段档最全。字典同步是"把 GP 的表清单抄一份到本地"，抄哪一份都行，
// 但要**确定**是同一份 —— 所以写死，不跟配置里的营运据点联动。
// 直查（dict r.t --env <环境>）才按营运据点选 schema。
const SyncCompany = "ds"

// DictPath 返回本地 GP 字典库路径。
// 环境可单独指定 DictDir，否则落在 GP 数据目录下。
//
// 导出给 CLI 与 Web 共用：字典同步与 CLI 查询必须落到同一个 <环境>.db，
// 路径规则只能有一份实现。
func DictPath(e config.NamedSsh) string { return dictPath(e) }

// dictPath 本地字典库路径。环境可单独指定 DictDir，否则落在 gp 数据目录下。
func dictPath(e config.NamedSsh) string {
	dir := e.DictDir
	if dir == "" {
		dir = os.Getenv("TG_GP_DICT_DIR")
	}
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".tg", "gp-dict")
	}
	return filepath.Join(dir, e.Name+".db")
}

// Company 返回该 GP 环境直查时用的资料库编号（azp_file.azp03 = zta_file.zta02），空 = "ds"。
//
// 为什么是 azp03 而不是据点编号 azp01：aooi901 里「资料库编号」是独立栏位，
// 默认 'ds'，两者实测有 15/30 不同（COF → cof_demo、DEMO1 → demo1）；拿据点
// 编号小写去猜 schema 会查到空。
//
// 为什么不是"账号就是它"：GP 一个库里装了几十家公司，字典表按资料库编号分行
// 存；连库账号只决定以谁的身份登录。本项目这套库里账号叫 digiwin，而
// zta02='ds' 有 2,962 张表、'digiwin' 只有 2,902 张 —— 混着用就会出现
// "本地查得到、远程说表不存在"，表格总数也对不上。
//
// 只用于直查；本地字典库的同步固定用 SyncCompany。
func Company(e config.NamedSsh) string {
	if s := strings.ToLower(strings.TrimSpace(e.Company)); s != "" {
		return s
	}
	return SyncCompany
}

// Stat 读本地字典库的文件状态（存在/大小），与 T100 的 dbSyncView 同一口径：
// 只 os.Stat，不打开 SQLite。表数是数据字典页按需查询的信息，不该挂在环境清单上。
func Stat(e config.NamedSsh) (exists bool, sizeMB float64) {
	st, err := os.Stat(dictPath(e))
	if err != nil {
		return false, 0
	}
	return true, float64(st.Size()) / 1048576
}
