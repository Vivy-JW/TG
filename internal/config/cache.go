package config

import (
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// 配置目录下的**缓存**:可再生的中间数据,删了会自愈,不删会一直涨。
//
// 它们与 config.json 同居一个目录,但性质完全不同 —— config.json 是你的数据
// (口令、环境、路径),删了就没了;下面这几个只是跑出来的副本与快照。
// "清缓存"必须只清后者,所以哪些算缓存**在这里定义一次**,CLI、Web、启动清理
// 都从这一份拿,免得某处漏了或某处手滑把配置一起删了。
//
// 不列入的:
//   - config.json      用户数据
//   - .tt-serve.json   **运行中**服务的 pid 文件,删了 `tt serve --stop` 就找不着它
//   - .tt-serve.log    日志(排障要看,且不影响正确性;想清可以用系统手段)
var CacheSubdirs = []string{
	"ents",      // 企业目录快照(ENT→账号),10 分钟新鲜期,过期自动重查
	"srccache",  // 调试时的源码镜像,只活一轮调试
	"execlog",   // 调试执行的大输出落盘副本
	"debug-bps", // 断点存档
	"spill",     // 查询被截断时落盘的完整结果(见 tt dict 的返回上限)
	"update",    // tt 自身的更新：下载的安装包与最近一次检查结果(删了只是下次重下)
	"runtime",   // 登录后回读的 TOP/ERP/COM（tt file 用；30 分钟新鲜期，可 --refresh 重探）
}

// CacheDir 数据目录下的缓存子目录。name 必须在 CacheSubdirs 清单里，
// 不在返回空串 —— 缓存名只允许这份清单里的，调用方写错名字当场得到空串
// （而不是悄悄建出一个清单外的新目录）。凡是 <数据目录>/<名字> 这种拼接都必须走它。
func CacheDir(dataDir, name string) string {
	if dataDir == "" {
		return ""
	}
	ok := false
	for _, n := range CacheSubdirs {
		if n == name {
			ok = true
			break
		}
	}
	if !ok {
		return ""
	}
	return filepath.Join(dataDir, name)
}

// CacheDirStatus 一个缓存子目录的状态。
type CacheDirStatus struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
	Bytes  int64  `json:"bytes"`
	Files  int    `json:"files"`
}

// CacheStatus 缓存目录的整体状态(设置页与 tt cache 共用的载荷)。
type CacheStatus struct {
	// Dir 配置目录本身 —— 缓存是它下面的子目录,这个路径回答了"东西在哪"。
	Dir   string           `json:"dir"`
	Dirs  []CacheDirStatus `json:"dirs"`
	Bytes int64            `json:"bytes"`
	Files int              `json:"files"`
	// MaxAgeHours 启动清理的年龄阈值(见 CleanCache):比它旧的才删。
	MaxAgeHours int `json:"maxAgeHours"`
}

// CacheMaxAge 启动时清理的年龄阈值。
//
// **刻意不"启动就全清"**:spill 里那些文件的用途正是"刚才那条查询的完整结果,
// 想看全量去读它"—— 启动即清会把上一条命令刚告诉用户的那份删掉。
// 按年龄删既控住了增长,又不动手上的东西;真想立刻清空用 tt cache clear。
const CacheMaxAge = 7 * 24 * time.Hour

// CacheStatusOf 汇总缓存占用。dataDir 为空(定位不到配置目录)时返回零值 + 空 Dir。
func CacheStatusOf(dataDir string) CacheStatus {
	st := CacheStatus{Dir: dataDir, MaxAgeHours: int(CacheMaxAge.Hours())}
	if dataDir == "" {
		return st
	}
	for _, name := range CacheSubdirs {
		d := CacheDirStatus{Name: name, Path: filepath.Join(dataDir, name)}
		if fi, err := os.Stat(d.Path); err == nil && fi.IsDir() {
			d.Exists = true
			d.Bytes, d.Files = dirUsage(d.Path)
		}
		st.Bytes += d.Bytes
		st.Files += d.Files
		st.Dirs = append(st.Dirs, d)
	}
	return st
}

// dirUsage 递归统计一个目录的字节数与文件数(读不到的项跳过,不因此报错)。
func dirUsage(dir string) (bytes int64, files int) {
	_ = filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if fi, err := d.Info(); err == nil {
			bytes += fi.Size()
			files++
		}
		return nil
	})
	return bytes, files
}

// CleanCache 清理缓存,返回删掉的文件数与会释放的字节数。
//
// olderThan > 0:只删**修改时间早于** now-olderThan 的文件(启动清理走这条);
// olderThan == 0:整个子目录删掉(用户手动"清除缓存"走这条)。
//
// 只碰 CacheSubdirs 里列出的目录 —— 配置目录下的其它东西一律不动。
func CleanCache(dataDir string, olderThan time.Duration) (removed int, freed int64, err error) {
	if dataDir == "" {
		return 0, 0, os.ErrNotExist
	}
	cutoff := time.Time{}
	if olderThan > 0 {
		cutoff = time.Now().Add(-olderThan)
	}
	for _, name := range CacheSubdirs {
		dir := filepath.Join(dataDir, name)
		if _, statErr := os.Stat(dir); statErr != nil {
			continue
		}
		if cutoff.IsZero() {
			// 全清:整个子目录删掉
			f, b, rmErr := RemoveTree(dir)
			if rmErr != nil {
				err = rmErr
				continue
			}
			removed, freed = removed+f, freed+b
			continue
		}
		// 按年龄:删旧文件,再回收被清空的目录
		r, f := removeOlderThan(dir, cutoff)
		removed, freed = removed+r, freed+f
		_ = pruneEmptyDirs(dir)
	}
	return removed, freed, err
}

// removeOlderThan 删掉 dir 下修改时间早于 cutoff 的文件。
func removeOlderThan(dir string, cutoff time.Time) (removed int, freed int64) {
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		fi, err := d.Info()
		if err != nil || !fi.ModTime().Before(cutoff) {
			return nil
		}
		if os.Remove(p) == nil {
			removed++
			freed += fi.Size()
		}
		return nil
	})
	return removed, freed
}

// RemoveTree 删掉一整棵目录树(自底向上逐个删),返回删掉的文件数与字节数。
//
// **不用 os.RemoveAll**:在 Windows 上 %APPDATA% 可能被应用沙箱重定向成 reparse
// point(实测 Claude 桌面版的 MSIX 容器里,tt 在它下面建的目录都成了
// `AppData\Local\Packages\<包名>\LocalCache\Roaming\...` 的跳转),而 os.RemoveAll
// 对这类目录报 ELOOP("too many levels of symbolic links")—— 同一个目录
// filepath.WalkDir 走得好好的,所以这不是权限或占用问题。
// 逐个 os.Remove 走得通,且失败时能说清是哪个文件。\n//\n// 导出是因为"删一棵目录"不只缓存要；两处各写一份\n// 迟早会有一处用回 os.RemoveAll,然后把那种容器里的目录删成 ELOOP。
func RemoveTree(dir string) (files int, bytes int64, err error) {
	var dirs []string
	walkErr := filepath.WalkDir(dir, func(p string, d fs.DirEntry, werr error) error {
		if werr != nil {
			return werr
		}
		if d.IsDir() {
			if p != dir {
				dirs = append(dirs, p)
			}
			return nil
		}
		fi, ierr := d.Info()
		if rerr := os.Remove(p); rerr != nil {
			return rerr
		}
		files++
		if ierr == nil {
			bytes += fi.Size()
		}
		return nil
	})
	if walkErr != nil {
		return files, bytes, walkErr
	}
	// 深的先删:父目录要等子目录空了才删得掉
	for i := len(dirs) - 1; i >= 0; i-- {
		if rerr := os.Remove(dirs[i]); rerr != nil {
			return files, bytes, rerr
		}
	}
	return files, bytes, os.Remove(dir)
}

// pruneEmptyDirs 自底向上删掉 dir 下的空目录(保留 dir 自身)。
func pruneEmptyDirs(dir string) error {
	var dirs []string
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err == nil && d.IsDir() && p != dir {
			dirs = append(dirs, p)
		}
		return nil
	})
	// 深的先删:父目录要等子目录空了才可能空
	for i := len(dirs) - 1; i >= 0; i-- {
		_ = os.Remove(dirs[i]) // 非空会失败,正是我们要的
	}
	return nil
}
