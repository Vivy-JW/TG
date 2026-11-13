package dbsync

// 同步引擎：连远端库、开一个临时 SQLite、把远端数据投影进去、原子替换成正式库。
//
// 这里原本是 T100 的"整表复制"（erpdb.SelectAllSQL → RebuildTable → 建主键索引）。
// TG 只支持 TOP GP，而 GP 的字典要 join 五六处再归并，复制语法用不上，
// 所以引擎收缩成"给一个写库函数"，具体怎么读写由调用方决定（现在是 gp.go 的 syncGP）。

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"tg/internal/dbconfig"
)

// Options 一次同步的输入。
type Options struct {
	// Path 目标 <环境>.db；中间的 .tmp 与最终替换都以它为准。
	Path string
	// DSN 远端库连接串（oracle://账号:口令@主机:端口/服务名）。
	DSN string
	// Owner 远端 schema（大写），只写进 meta 供排查。
	Owner string
	// Lib 字典据点代号（如 ds），只写进 meta 供排查。
	Lib string
	// Sync 真正干活的函数：读 src（Oracle）、写 dst（临时 SQLite）。
	// rep 收进度文案，onCount 回报表级进度（Done/Total），meta 收尾写进 meta 表。
	Sync func(ctx context.Context, src, dst *sql.DB, rep func(string), onCount func(done, total int), meta map[string]string) error
	// OnCount 可选：表级进度（Total=表数，Done=已写的表数），给 Web 的进度条用。
	OnCount func(done, total int)
}

// Stats 一次同步的结果，供 CLI 输出与 Web 的 job.Result 使用。
type Stats struct {
	Path        string
	SizeMB      float64
	Tables      int
	Columns     int
	Indexes     int
	Constraints int
	Messages    int
}

// DSN 拼 Oracle 连接串（账号/口令要转义，口令里可能有 @ # 这类字符）。
func DSN(c dbconfig.Connection, account, password string) string {
	return fmt.Sprintf("oracle://%s:%s@%s:%d/%s",
		url.QueryEscape(account), url.QueryEscape(password), c.Host, c.Port, c.Service)
}

// Run 跑一次同步：先写 Path+".tmp"，成功后删旧库、改名、写 meta。
//
// 失败时临时文件会被删掉，旧库原样保留 —— 同步中途断掉不会留下半个库。
// onProgress 逐条收进度文案（可为 nil）。
func Run(ctx context.Context, opt Options, onProgress func(string)) (*Stats, error) {
	if strings.TrimSpace(opt.Path) == "" {
		return nil, fmt.Errorf("没有给目标路径")
	}
	if strings.TrimSpace(opt.DSN) == "" {
		return nil, fmt.Errorf("没有给数据库连接串")
	}
	if opt.Sync == nil {
		return nil, fmt.Errorf("没有给写库函数")
	}
	rep := func(msg string) {
		if onProgress != nil {
			onProgress(msg)
		}
	}
	if err := os.MkdirAll(filepath.Dir(opt.Path), 0o755); err != nil {
		return nil, err
	}
	tmp := opt.Path + ".tmp"
	_ = os.Remove(tmp)

	dst, err := sql.Open("sqlite", tmp)
	if err != nil {
		return nil, err
	}
	if _, err := dst.Exec(gpSchemaSQL); err != nil {
		dst.Close()
		_ = os.Remove(tmp)
		return nil, fmt.Errorf("建 SQLite 失败：%w", err)
	}

	src, err := sql.Open("oracle", opt.DSN)
	if err != nil {
		dst.Close()
		_ = os.Remove(tmp)
		return nil, err
	}
	if err := src.PingContext(ctx); err != nil {
		src.Close()
		dst.Close()
		_ = os.Remove(tmp)
		return nil, fmt.Errorf("连不上数据库：%w", err)
	}

	meta := map[string]string{}
	onCount := opt.OnCount
	if onCount == nil {
		onCount = func(done, total int) {}
	}
	err = opt.Sync(ctx, src, dst, rep, onCount, meta)
	src.Close()
	if err != nil {
		dst.Close()
		_ = os.Remove(tmp)
		return nil, err
	}

	// meta 供 tt dict db status / 数据字典页显示"什么时候同步的"。
	setMeta := func(k, v string) {
		_, _ = dst.Exec(`INSERT OR REPLACE INTO meta(key,value) VALUES(?,?)`, k, v)
	}
	setMeta("synced_at", time.Now().Format(time.RFC3339))
	setMeta("owner", strings.ToUpper(opt.Owner))
	setMeta("company", opt.Lib)
	for k, v := range meta {
		setMeta(k, v)
	}
	dst.Close()

	_ = os.Remove(opt.Path)
	if err := os.Rename(tmp, opt.Path); err != nil {
		return nil, err
	}
	st := &Stats{Path: opt.Path}
	if fi, err := os.Stat(opt.Path); err == nil {
		st.SizeMB = float64(fi.Size()) / 1048576
	}
	st.Tables = atoiMeta(meta["tables"])
	st.Columns = atoiMeta(meta["columns"])
	st.Indexes = atoiMeta(meta["indexes"])
	st.Constraints = atoiMeta(meta["constraints"])
	st.Messages = atoiMeta(meta["messages"])
	rep(fmt.Sprintf("已写入 %s（%.1f MB）", st.Path, st.SizeMB))
	return st, nil
}

// atoiMeta 把 meta 里的计数字符串转回 int；解析不出来就当 0。
func atoiMeta(s string) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return n
		}
		n = n*10 + int(r-'0')
	}
	return n
}
