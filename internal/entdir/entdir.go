// Package entdir 企业目录(ENT → 账号)的共享件:映射表、落盘快照、环境指纹与新鲜期。
//
// 它只放**与传输无关**的那一半 —— 怎么连服务器、怎么在库上查 gzou_t 由调用方注入,
// 因为两条路径的传输方式本来就不同:
//
//	tt debug  在 T100 服务器侧跑 sqlplus/ksql(SSH + stdin)
//	tt dict   从客户端直连库(erpdb + 可选 SSH 隧道)
//
// 但两边读的是**同一张 gzou_t**、写的是**同一个快照文件**。这个包存在的意义就是让
// 它们对"企业 99 是哪个账号"必须给出同一个答案:文件名、指纹、新鲜期、序列化格式
// 全部只有一份定义,任何一边改了另一边自动跟上。
//
// 注意"企业 → 数据库"这个说法是不准的:库是**环境级**的(hosts.sshs[].db,一对一挂),
// 企业编号只决定库里的账号/schema(gzou_t.gzou003),所以解析出来的永远是账号。
//
// 快照是**可丢弃的缓存,不是数据源**:读坏了 / 写不出 / 指纹不符一律当没有,绝不因此报错。
package entdir

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"tt/internal/config"
)

const (
	// TTL 快照与进程内缓存共用的新鲜期。
	//
	// 两条路径必须用同一个值:否则 agent 会拿到两条互相矛盾的时间线
	// (一边说企业 99 在,一边说不在)。
	TTL = 10 * time.Minute

	dirName = "ents"
	// Version 快照格式版本;不符即当没有,不做兼容读。
	Version = 1
)

// Mapping 一个企业编号 → 账号。
type Mapping struct {
	Ent     int    `json:"ent"`
	Account string `json:"account"` // gzou003:账号/schema 名;"-" 表示没配
	// Placeholder = 这个企业存在,但 gzou003 为空/占位,没有可用账号。
	//
	// 必须单独标出来:光看 Account 是 "-" 会让人(和 agent)以为那就是账号名,
	// 进而把"没配账号"读成"这个企业不存在" —— 两种结论差别很大。
	// 不用 omitempty:agent 要能依赖这个键始终存在。
	Placeholder bool `json:"placeholder"`
}

// Snapshot 落盘的企业目录快照(<数据目录>/ents/<环境段>.json)。
type Snapshot struct {
	Version     int       `json:"version"`
	Env         string    `json:"env"`
	Fingerprint string    `json:"fingerprint"`
	Zone        string    `json:"zone"`
	Dialect     string    `json:"dialect"`
	Target      string    `json:"target"`
	Mappings    []Mapping `json:"mappings"`
	FetchedAt   time.Time `json:"fetchedAt"`
}

// DBIdent 库身份片段(type|host|port|svc),指纹的输入之一。
// 没有库时调用方传 "none"。
func DBIdent(dialect, host string, port int, svc string) string {
	return dialect + "|" + host + "|" + strconv.Itoa(port) + "|" + svc
}

// Fingerprint 环境指纹:换机器 / 换区域 / 换库 = 换了另一份 gzou_t,快照立即失效。
//
// zone 与库身份都必须在:zone 决定登录后的 T100 环境(31 开发 / 36 正式),
// 而 oracle 的 service 本身就带 zone 语义(如 t35prd)。只按 host 做 key 会把
// 开发区的企业清单当成正式区的。
func Fingerprint(sshHost, sshUser, zone, dbIdent string) string {
	if dbIdent == "" {
		dbIdent = "none"
	}
	return sshHost + "|" + sshUser + "|" + zone + "|" + dbIdent
}

// EnvSeg 快照文件名里的环境段:优先环境名,退化为 主机-区域。
// 与源码镜像走同一套命名,中文环境名可用、`..` 逃不出去。
func EnvSeg(envName, sshHost, zone string) string {
	s := strings.TrimSpace(envName)
	if s == "" {
		s = sshHost
		if zone != "" {
			s += "-" + zone
		}
	}
	return PathSafeSeg(s)
}

// Path 快照路径;dataDir 为空返回空串(调用方据此跳过落盘)。
// 子目录名经 config.CacheDir 校验 —— 名字不在缓存清单里会得到空串。
func Path(dataDir, seg string) string {
	sub := config.CacheDir(dataDir, dirName)
	if sub == "" {
		return ""
	}
	return filepath.Join(sub, seg+".json")
}

// Read 读快照。不存在 / 坏掉 / 版本不符 / 指纹不符一律返回 nil(当没有)。
// **不判过期** —— 过期快照在"现查失败"时是唯一的答案来源,用不用由调用方定。
func Read(path, fingerprint string) *Snapshot {
	if path == "" {
		return nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var s Snapshot
	if err := json.Unmarshal(b, &s); err != nil {
		return nil
	}
	if s.Version != Version || s.Fingerprint != fingerprint || len(s.Mappings) == 0 {
		return nil
	}
	return &s
}

// Write 原子落盘。失败只返回错误,由调用方降级成一条提示 ——
// 快照写不出去不该让"查到了"这件事变成失败。
func Write(path string, s *Snapshot) error {
	if path == "" {
		return errors.New("未配置数据目录")
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return config.AtomicWrite(path, b)
}

// PathSafeSeg 把任意字符串压成单个安全的路径段。
// 保留中文与常见符号(环境名可能是"示例测试区"),只替换分隔/通配类字符。
func PathSafeSeg(s string) string {
	out := strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|', 0:
			return '_'
		}
		return r
	}, s)
	out = strings.TrimSpace(out)
	// 兜底:空、当前目录、上级目录都不能作为路径段(防 `..` 逃出镜像根)
	if out == "" || out == "." || out == ".." {
		return "_"
	}
	return out
}
