package file

// $TOP 的登录探测缓存。
//
// ProbeTEnv 要开 PTY、等区域菜单、回读环境变量，一次 2–5 秒 —— 比读文件本身贵得多。
// 所以探测结果落一份缓存（TTL 30 分钟，--refresh 强制重探）；T100 路径不允许静态配置，
// 这份缓存只是“刚刚回读到的值”，不是配置。
//
// 缓存放在数据目录下的 runtime/，删了会自愈；环境 + zone 一起做键 —— 同一个环境
// 换了登录区域，TOP 就会变，不能拿旧值复用。

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"tt/internal/config"
	"tt/internal/host"
)

const runtimeTTL = 30 * time.Minute

type cachedRuntime struct {
	Env             string    `json:"env"`
	Zone            string    `json:"zone"`
	TOP             string    `json:"top"`
	ERP             string    `json:"erp"`
	COM             string    `json:"com"`
	FGLDir          string    `json:"fgldir"`
	FGLResourcePath string    `json:"fglresourcepath"`
	Topent          string    `json:"topent"`
	FetchedAt       time.Time `json:"fetchedAt"`
}

// resolveTop 取本次的 $TOP：缓存命中就直接用，否则登录探测并写缓存。
func (t *target) resolveTop(cfgPath string) error {
	cacheDir := config.CacheDir(dataDirOf(cfgPath), "runtime")
	if !refreshTop && cacheDir != "" {
		if rt, ok := loadCachedRuntime(cacheDir, t.env, t.ssh.Zone); ok && time.Since(rt.FetchedAt) < runtimeTTL {
			t.top = cleanTop(rt.TOP)
			if t.top != "" {
				return nil
			}
		}
	}
	if strings.TrimSpace(t.ssh.Zone) == "" {
		return usageErr("环境 %s 没有配置登录区域（zone），无法探测 $TOP；在设置页「站点管理」补上 zone", t.env)
	}
	rt, err := host.ProbeTEnv(t.conn, t.ssh.Zone)
	if err != nil {
		return ioErr("登录探测 T100 目录失败（环境 %s，zone %s）：%v", t.env, t.ssh.Zone, err)
	}
	if rt == nil || strings.TrimSpace(rt.TOP) == "" {
		return ioErr("登录探测没有回读到 $TOP（环境 %s，zone %s）；检查该环境的登录脚本", t.env, t.ssh.Zone)
	}
	rt.FetchedAt = time.Now()
	t.top = cleanTop(rt.TOP)
	if cacheDir != "" {
		saveCachedRuntime(cacheDir, t.env, t.ssh.Zone, rt) // 尽力而为：写不进去不影响本次
	}
	return nil
}

func cleanTop(top string) string {
	top = strings.TrimSpace(top)
	if top == "" {
		return ""
	}
	return strings.TrimSuffix(top, "/")
}

func runtimeFile(dir, env, zone string) string {
	sum := sha1.Sum([]byte(env + "\x00" + zone))
	return filepath.Join(dir, fmt.Sprintf("%x", sum)[:16]+".json")
}

func loadCachedRuntime(dir, env, zone string) (*cachedRuntime, bool) {
	b, err := os.ReadFile(runtimeFile(dir, env, zone))
	if err != nil {
		return nil, false
	}
	var rt cachedRuntime
	if err := json.Unmarshal(b, &rt); err != nil {
		return nil, false
	}
	if rt.Env != env || rt.Zone != zone || rt.TOP == "" {
		return nil, false
	}
	return &rt, true
}

func saveCachedRuntime(dir, env, zone string, rt *host.RuntimeEnv) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	payload := cachedRuntime{
		Env: env, Zone: zone,
		TOP: rt.TOP, ERP: rt.ERP, COM: rt.COM,
		FGLDir: rt.FGLDIR, FGLResourcePath: rt.FGLResourcePath,
		Topent: rt.Topent, FetchedAt: rt.FetchedAt,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return
	}
	tmp, err := os.CreateTemp(dir, ".runtime-*.tmp")
	if err != nil {
		return
	}
	name := tmp.Name()
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		os.Remove(name)
		return
	}
	if err := tmp.Close(); err != nil {
		os.Remove(name)
		return
	}
	_ = os.Rename(name, runtimeFile(dir, env, zone))
}
