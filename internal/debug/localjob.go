package debug

// 本地字典库优先的作业解析（2026-10-23）。
//
// 「点启动 → 服务端解析作业」原来每次都走 SSH 起 ksql 连库查 gzzz_t（实测约 3s，
// 金仓客户端冷启动是固定开销），是"点启动到弹画面"等待的大头。而本地已经有一份
// 同步好的字典库（<数据目录>/dict/<环境>.db，`tt dict db sync` 的产物），里面有
// 同一张 gzzz_t / gzza_t。与字段解析"本地 sqlite → 服务器"的三级链同一纪律：
// **本地命中就不碰 SSH**；本地没有（库不存在/没同步过程序族/作业不在）才回退远程。
//
// 本地库最多落后一个同步周期。对"作业→实体程序"这种极低频变化的映射，可接受；
// 与字典查询以本地为准的口径一致。

import (
	"database/sql"
	"log"
	"os"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// lookupJobLocalDB 在给定本地字典库文件里解析作业 → 实体程序。
// 任何失败（文件不存在 / 不是 sqlite / 表不在 / 无记录）都返回 ok=false，
// 由调用方回退远程链路 —— 本地查找只是加速路径，绝不能因为它的异常挡住启动。
func lookupJobLocalDB(dbPath, job string) (JobResolve, bool) {
	job = strings.TrimSpace(job)
	if dbPath == "" || job == "" {
		return JobResolve{}, false
	}
	if st, err := os.Stat(dbPath); err != nil || st.IsDir() || st.Size() == 0 {
		log.Printf("[launch-timing] 本地解析跳过: 库文件不可用 %s (%v)", dbPath, err)
		return JobResolve{}, false
	}
	d, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Printf("[launch-timing] 本地解析跳过: 打开失败 %v", err)
		return JobResolve{}, false
	}
	defer d.Close()
	// 字典同步是「写临时库 → 原子替换」，替换那一瞬可能撞上 SQLITE_BUSY。等一小会儿再放弃，
	// 别让一次偶发占用就把本地优先降级成远程（那条路要多花约 3 秒）。
	_, _ = d.Exec("PRAGMA busy_timeout=2000")
	t0 := time.Now()
	// 与 dbResolveJob 同一查询语义（gendbg.4gl 口径）：
	//   gzza001=实体程序 | gzzz004=额外参数 | gzza004=启动引用 | gzzz005=模块
	// 一个程序可挂多个作业；作业号不区分大小写匹配（用户输入大写也能中）。
	row := d.QueryRow(
		`select gzza001, coalesce(gzzz004,' '), coalesce(gzza004,' '), coalesce(gzzz005,'-')
		 from gzzz_t join gzza_t on gzza001 = gzzz002
		 where lower(gzzz001) = lower(?)`, job)
	var prog, gzzz004, gzza004, gzzz005 string
	if err := row.Scan(&prog, &gzzz004, &gzza004, &gzzz005); err != nil {
		log.Printf("[launch-timing] 本地解析跳过: 查询失败 %v", err)
		return JobResolve{}, false
	}
	jr := JobResolve{
		Prog:      prog,
		Extra:     strings.TrimSpace(gzzz004),
		Module:    gzzz005,
		LaunchRef: strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(gzza004), "$FGLRUN")),
	}
	log.Printf("[launch-timing] 作业解析 %s: 本地字典库命中 耗时 %v", job, time.Since(t0).Round(time.Millisecond))
	return jr, true
}
