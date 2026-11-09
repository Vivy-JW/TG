package debug

// 源码本地镜像 + 会话外读取连接复用。
//
// 镜像:<DataDir>/srccache/<目标作用域>/<服务器路径>。**这轮调试经历过的文件**都留一份
// 本地拷贝,并在返回值里带出本地路径,好让 AI 整读、反复搜,不必为每个文件再走一次 SSH、
// 也不必迁就 source 的行段读取。
//
// 两个来源:程序停过的文件(mirrorStopFile,每轮调试的真实经历)与显式读过的文件
// (ResolveSource / ReadSourceStandalone)。
//
// 它**只活一轮调试**:每轮启动前清空**本会话**的镜像。服务器上的源码是会改的,
// 留着上一轮的文件只会让 AI 拿着过期代码推理 —— 宁可贵一点重拉。
// 权威性始终在 tt debug source(每次真读服务器)那一侧。
//
// # 目标作用域（WP5 / T5.1，缺陷 X）
//
// 环境段（`mirrorScopeOf` 的第一段）隔离的是**测试区/正式区**:同名文件在两个区的登录
// 路径与内容都可能不同。但它今天**不够**:两条会话可能同环境（只差企业编号或作业），
// 而"清空镜像"是**会话级**动作（每条会话开始新的一轮就清一次）。于是并发启动时
// A 的清理会把 B 刚拉的镜像删掉 —— 这就是缺陷 X 的前一半。
//
// 后一半在 `Session.mirrored`:它是**会话内**的"这个文件我镜像过"标记，
// 而 A 拉过的文件落在**共享**目录里，B 永远看不到 ⇒ "B 永远拉不到 A 拉过的文件"。
//
// 所以路径的第二段是**运行作用域**（`Session.runScope`，按会话唯一）：
// "已镜像"标记与它落盘的目录**同一个作用域**，两件事从此同进同退。
// 会话之间互删不了（各清各的子树），会话之内"新的一轮清空"照旧（`clearMirrorForSession`）。
//
// 整个代码库的检索不在本工具的范围内。

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"tt/internal/config"
	"tt/internal/entdir"
	"tt/internal/host"

	"github.com/pkg/sftp"
)

// srcMirrorDir 镜像根目录名(位于 DataDir 下,与断点存档同级)
const srcMirrorDir = "srccache"

// pathSafeSeg 把任意字符串压成单个安全的路径段。
// 实现在 internal/entdir:企业目录快照的文件名用同一套清洗,两处不能各写一份。
func pathSafeSeg(s string) string { return entdir.PathSafeSeg(s) }

// mirrorEnvSeg 镜像目录里的环境段:优先环境名,退化为 主机-区域。
func mirrorEnvSeg(envName, sshHost, zone string) string {
	return entdir.EnvSeg(envName, sshHost, zone)
}

// mirrorScopeOf 拼出镜像目录的**目标作用域**（缺陷 X / T5.1）:环境段 + 会话运行作用域。
//
// 两段都必须有:
//   - 环境段隔离测试区/正式区（同名文件在两个区的路径与内容都可能不同）;
//   - 运行作用域隔离**同一环境下的两条会话** —— 它们各自"只活一轮"地清自己的镜像,
//     互删不了对方的（`clearMirrorForSession` 只删自己那一段）。
//
// runScope 为空时退化成单段（环境段）:那种调用只可能来自"没有会话身份"的老式用法，
// 在生产路径上不会出现（会话一定先建立，见 Session.runScope）。
func mirrorScopeOf(envSeg, runScope string) string {
	envSeg = pathSafeSeg(strings.TrimSpace(envSeg))
	if s := strings.TrimSpace(runScope); s != "" {
		return filepath.Join(envSeg, pathSafeSeg(s))
	}
	return envSeg
}

// srcMirrorPathScoped 服务器绝对路径 → 本地镜像绝对路径（按目标作用域）。
// 后缀沿用服务器路径(去掉前导 /)以保留目录层次,方便人肉对照;逐段清洗防越界。
func srcMirrorPathScoped(dataDir, scope, serverPath string) string {
	if dataDir == "" || serverPath == "" {
		return ""
	}
	rel := strings.TrimPrefix(filepath.ToSlash(serverPath), "/")
	if rel == "" {
		return ""
	}
	segs := strings.Split(rel, "/")
	for i, sg := range segs {
		segs[i] = pathSafeSeg(sg)
	}
	return filepath.Join(append([]string{dataDir, srcMirrorDir, scope}, segs...)...)
}

// srcMirrorPath 服务器绝对路径 → 本地镜像绝对路径（作用域只有一段时的简写）。
//
// 单段作用域是**目标作用域**的一种特例（`mirrorScopeOf(envSeg, "")`），
// 保留这个入口是为了路径拼装的那组单测不必先造一条会话。
func srcMirrorPath(dataDir, envSeg, serverPath string) string {
	return srcMirrorPathScoped(dataDir, envSeg, serverPath)
}

// writeMirrorScoped 原子落盘镜像并返回本地路径;写不进去返回空串。
// 镜像只是给 AI 的便利副本,任何失败都只记录不报错 —— 绝不能因为它读不到源码。
func writeMirrorScoped(dataDir, scope, serverPath string, data []byte) string {
	p := srcMirrorPathScoped(dataDir, scope, serverPath)
	if p == "" {
		return ""
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		log.Printf("[srccache] 建目录失败 %s: %v", filepath.Dir(p), err)
		return ""
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		log.Printf("[srccache] 写镜像失败 %s: %v", tmp, err)
		return ""
	}
	if err := os.Rename(tmp, p); err != nil {
		log.Printf("[srccache] 落位失败 %s: %v", p, err)
		return ""
	}
	return p
}

// writeMirror 同 writeMirrorScoped,作用域只有一段时的简写（见 srcMirrorPath）。
func writeMirror(dataDir, envSeg, serverPath string, data []byte) string {
	return writeMirrorScoped(dataDir, envSeg, serverPath, data)
}

// srcMirrorRoot 镜像根目录(<DataDir>/srccache)。落点走 config.CacheDir(统一路径管理器),
// 名字不在缓存清单里会得到空串。
func srcMirrorRoot(dataDir string) string {
	return config.CacheDir(dataDir, srcMirrorDir)
}

// srcMirrorScopeRoot 某个目标作用域在镜像根下的子树(<DataDir>/srccache/<作用域>)。
// 清理**只删这一棵子树** —— 这是"会话之间互删不了"的落点（缺陷 X）。
func srcMirrorScopeRoot(dataDir, scope string) string {
	root := srcMirrorRoot(dataDir)
	if root == "" || strings.TrimSpace(scope) == "" {
		return ""
	}
	return filepath.Join(root, scope)
}

// execLogDir 命令完整输出的落盘目录(<DataDir>/execlog)。
// 与源码镜像分开放:一个是"读过的源码",一个是"跑过的命令输出",生命周期相同但语义不同。
const execLogDir = "execlog"

// writeExecLog 把一次命令的完整输出落成本地文件,返回路径;写不进去返回空串。
//
// 为什么要有它:回给调用方的正文有上限(超了就只给头部),但"完整内容"必须留得住 ——
// 否则想多看一点就得重跑命令,而重跑会改变现场。落本地之后可以随时 grep/整读,零往返。
// 与镜像同款:原子落位、失败只记录不报错。
//
// scope 与源码镜像同源（`Session.runScope`,见 mirrorScopeOf）:清理是按同一棵子树做的,
// 两处分别按各自的会话清（缺陷 X 的第二半 —— 从前 execlog 是全局目录,
// A 启动会顺手删掉 B 刚落下的大输出副本）。
func writeExecLog(dataDir, scope, name string, data []byte) string {
	if dataDir == "" || name == "" {
		return ""
	}
	dir := filepath.Join(config.CacheDir(dataDir, execLogDir), scope)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Printf("[execlog] 建目录失败 %s: %v", dir, err)
		return ""
	}
	p := filepath.Join(dir, pathSafeSeg(name))
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		log.Printf("[execlog] 写失败 %s: %v", tmp, err)
		return ""
	}
	if err := os.Rename(tmp, p); err != nil {
		log.Printf("[execlog] 落位失败 %s: %v", p, err)
		return ""
	}
	return p
}

// srcConnScopeSeg 会话外读取没有会话身份,作用域退化成**目标**（环境段）。
func srcConnScopeSeg(cfg *Config) string {
	if cfg == nil {
		return ""
	}
	return mirrorEnvSeg(cfg.EnvName(), cfg.SSH.Host, cfg.Zone)
}

// clearMirrorForSession 清空**本会话**这一轮攒下的所有本地副本（缺陷 X / T5.1）:
//   - srccache/<本会话作用域>/:读过的源码(代码会更新,跨轮次留着只会误导判断);
//   - execlog/<本会话作用域>/:跑过的命令输出(同理,而且它随轮次增长没有意义)。
//
// 每轮启动调试前调用,失败只记录。
//
// **只删本会话那棵子树**:并发时 A 的启动不该销毁 B 刚拉下来的镜像（那正是缺陷 X）。
// 同一条会话开始新的一轮时照旧清空（"只活一轮"这条语义不变）——
// 作用域按会话唯一且跨轮不变（`Session.runScope`),所以它清的正是自己上一轮的那一份。
func clearMirrorForSession(dataDir, scope string) {
	if dataDir == "" || strings.TrimSpace(scope) == "" {
		return
	}
	roots := []string{srcMirrorScopeRoot(dataDir, scope)}
	if d := config.CacheDir(dataDir, execLogDir); d != "" {
		roots = append(roots, filepath.Join(d, scope))
	}
	for _, root := range roots {
		if err := os.RemoveAll(root); err != nil {
			log.Printf("[srccache] 清理失败 %s: %v", root, err)
		}
	}
}

// clearMirror 清空**整个**镜像与命令输出目录（作用域只有一段时的简写，见 srcMirrorPath）。
//
// 生产路径不再用它：会话级的清理走 clearMirrorForSession（缺陷 X）。
// 留这个入口是为了路径拼装那组单测仍能一次清干净。
func clearMirror(dataDir string) {
	if dataDir == "" {
		return
	}
	for _, root := range []string{srcMirrorRoot(dataDir), config.CacheDir(dataDir, execLogDir)} {
		if root == "" {
			continue
		}
		if err := os.RemoveAll(root); err != nil {
			log.Printf("[srccache] 清理失败 %s: %v", root, err)
		}
	}
}

// ---------- 会话外读取连接复用 ----------

// srcConn 会话外读取共用的连接。
// 以前每次 tt debug source 都 host.Dial 一次:连着读十几段源码就是十几次完整 SSH 握手。
type srcConn struct {
	conn *host.SSHConn
	cl   *sftp.Client
	key  string
}

func (c *srcConn) close() {
	if c == nil {
		return
	}
	if c.cl != nil {
		_ = c.cl.Close()
	}
	if c.conn != nil {
		_ = c.conn.Close()
	}
}

// srcConnKey 连接归属:目标或账号变了就换连接。
//
// **按目标配置算，不看 `m.cfg`**（缺陷 Z / T5.3）:
//   - 从前它读 `m.cfg.SSH`，于是"环境 A 的读"与"环境 B 的读"可能落到同一条连接上;
//   - 更坏的是 key 只有 `server+user` **没有环境维度**:同机同账号的两个环境
//     （只差登录区域/企业编号 —— 正是 `sameTarget` 要分开的那一对）在 key 上**长得一样**,
//     换了环境仍在用旧连接、按旧区域解析出的动态路径读源码。
//
// 所以 key = 连接四元组（host:port/user）+ **环境名** + **区域**。
// 环境名进 key 是 C5.4 那条"同机不同环境不是同一个目标"在连接缓存上的落点;
// 区域进 key 是"同一个环境的两个区域解析出的 TOP/ERP 不同"的落点。
func srcConnKey(cfg *Config) string {
	if cfg == nil {
		return ""
	}
	return cfg.SSH.Addr() + "/" + cfg.SSH.User + "|" + cfg.EnvName() + "|" + cfg.Zone
}

// withSrcConn 借出会话外读取连接执行 fn（cfg = 本次调用认的那一份配置快照,见 sourceCfg）。
//
// 借用前先探活(SFTP 一次 RealPath 往返):空闲连接会被服务端回收,网络也可能断,
// 探活失败就丢弃重建。用探活而不是"失败后重试",是为了让调用方不必关心连接生命周期
// —— 探活那一次往返远比重新握手便宜。
//
// **并发口径（T5.3 的判据）:按连接排队,不按全局排队。**
// 锁是**每个连接 key 一把**（srcKeyLocks）:不同目标（不同环境/区域/服务器）各借各的连接、
// 真正并行;同一目标共用一条 SSH 连接与一个 SFTP 客户端,排队是诚实的（SSH 连接本身
// 就是那条共享资源,用多条连接去"并发读同一台机"只会让服务端多背几个会话）。
// 从前的写法是**一条**全局连接 + 一把全局锁:换目标即换连接,两个环境不可能同时在读（缺陷 Z）。
func (m *Manager) withSrcConn(cfg *Config, fn func(*host.SSHConn, *sftp.Client) error) error {
	release := m.lockSrcKey(srcConnKey(cfg))
	defer release()
	conn, cl, err := m.srcConn(cfg)
	if err != nil {
		return err
	}
	return fn(conn, cl)
}

// lockSrcKey 取得某个连接 key 的独占权,返回释放函数。
func (m *Manager) lockSrcKey(key string) func() {
	m.srcLocksMu.Lock()
	if m.srcKeyLocks == nil {
		m.srcKeyLocks = map[string]*sync.Mutex{}
	}
	mu := m.srcKeyLocks[key]
	if mu == nil {
		mu = &sync.Mutex{}
		m.srcKeyLocks[key] = mu
	}
	m.srcLocksMu.Unlock()
	mu.Lock()
	return mu.Unlock
}

// srcConn 取(或建立)指定目标的会话外读取连接（调用方已持有该 key 的锁）。
func (m *Manager) srcConn(cfg *Config) (*host.SSHConn, *sftp.Client, error) {
	key := srcConnKey(cfg)
	m.srcMu.Lock()
	cur := m.src[key]
	m.srcMu.Unlock()

	if cur != nil {
		if _, err := cur.cl.RealPath("."); err == nil {
			return cur.conn, cur.cl, nil
		}
		m.dropSrcConn(cur) // 死连接:关掉重建
	}

	conn, err := host.Dial(cfg.SSH)
	if err != nil {
		return nil, nil, fmt.Errorf("SSH 连接失败: %w", err)
	}
	// 动态路径(登录区域 → 环境脚本):探针+缓存;失败即报错(无静态配置可回退)。
	// 探到的结果**不回写任何共享配置**（缺陷 Y / T5.2）:它是那个目标的事实,
	// 由调用方按需拿去用（会话外读取只是要确认"这个目标解析得出来"）。
	if _, err := m.getRuntimeEnv(conn, zoneOf(cfg)); err != nil {
		_ = conn.Close()
		return nil, nil, err
	}
	cl, err := conn.SFTP()
	if err != nil {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("打开 SFTP 失败: %w", err)
	}
	nc := &srcConn{conn: conn, cl: cl, key: key}
	m.srcMu.Lock()
	old := m.src[key]
	m.src[key] = nc
	m.srcMu.Unlock()
	old.close() // 同 key 的旧连接没人用了
	return conn, cl, nil
}

// dropSrcConn 丢弃缓存连接(仅当它仍是该 key 缓存的那条,免得误关刚建好的)
func (m *Manager) dropSrcConn(c *srcConn) {
	if c == nil {
		return
	}
	m.srcMu.Lock()
	if m.src[c.key] == c {
		delete(m.src, c.key)
	}
	m.srcMu.Unlock()
	c.close()
}
