package sdd

import (
	"strings"
	"time"

	"tg/internal/cli/common"
	"tg/internal/config"
	"tg/internal/host"
	"tg/internal/output"
)

// remoteFindTimeout 在服务器上按前缀找说明书的上限。
const remoteFindTimeout = 60 * time.Second

// findRemote 直接连服务器，在 $TOP 与 $CUST 两棵树的 */sdd/ 下按 <作業>*.sdd 找。
//
// 为什么不复用 common.ReadRemoteFile 的 findRemote：那条是精确文件名匹配
// （-name '<base>'），而说明书是按前缀成组的（axmt410 → axmt410dis/icd/slk），
// 所以这里自己发一条带通配的 find，命中的相对路径再交给调用方去读。
func findRemote(e config.NamedSsh, job string) ([]match, error) {
	roots := common.RemoteRoots(e)
	if len(roots) == 0 {
		return nil, output.Errorf(output.CodeUsage, 2,
			"环境 %s 没有配置 $TOP（workspace）；在设置页「站点管理 → SSH 服务器」补上", e.Name).
			WithHint("或者用 --dir <目录> 指向本地已拉取的镜像目录")
	}
	conn, err := host.Dial(e.SSHConfig)
	if err != nil {
		return nil, output.Errorf(output.CodeConnFailed, 1, "连接服务器失败: %v", err)
	}
	defer conn.Close()

	fs, err := conn.FS()
	if err != nil {
		return nil, output.Errorf(output.CodeConnFailed, 1, "打开 SFTP 失败: %v", err)
	}
	defer fs.Close()

	seen := map[string]bool{}
	var out []match
	for _, root := range roots {
		cmd := "find " + shq(root) + " -maxdepth 4 -type f -path '*/sdd/*' -iname " +
			shq(job+"*.sdd") + " 2>/dev/null | head -n 40"
		raw, err := conn.Output(cmd, remoteFindTimeout)
		if err != nil {
			return nil, output.Errorf(output.CodeConnFailed, 1, "在服务器上查找 %s*.sdd 失败: %v", job, err)
		}
		for _, line := range strings.Split(raw, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			rel := strings.TrimPrefix(line, strings.TrimSuffix(root, "/")+"/")
			if rel == line || !isSDD(rel) {
				continue // 不在这个根下，或不在 sdd/ 里
			}
			if seen[rel] {
				continue
			}
			seen[rel] = true
			m := match{Path: rel, Source: "server", module: topDir(rel)}
			if st, statErr := fs.Stat(line); statErr == nil {
				m.Bytes = st.Size
			}
			out = append(out, m)
		}
	}
	sortMatches(out)
	return out, nil
}

// shq 单引号转义，和 common 里那条同义（那边没导出）。
func shq(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }
