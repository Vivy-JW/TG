package common

import (
	"io"
	"strconv"
	"strings"
	"time"

	"tt/internal/config"
	"tt/internal/host"
	"tt/internal/output"
)

// RemoteReadLimit 单文件远程读取上限。GP 的 4GL 单个最大约 2.6 MB（saapt110），
// 64 MB 足够包住任何正常文件，也挡住误打到一个超大文件上。
const RemoteReadLimit = 64 << 20

// remoteFindTimeout 在服务器上按文件名找一次的上限。
const remoteFindTimeout = 60 * time.Second

// ReadRemoteFile 直接连服务器读一个文件，不落本地。
//
// 找法：先按 cands 里的相对路径逐个试（相对 $TOP），都不在就让服务器在
// $TOP 下 maxdepth 3 按文件名找一次（$TOP/<模块>/{4gl,42f}/<名>）。
//
// 为什么要有这条路：GP 的一个 $TOP 下有 1.4 万个 4gl（约 700 MB），
// 全量镜像要十几分钟；读一个文件没必要先镜像。
func ReadRemoteFile(e config.NamedSsh, cands []string, base string, limit int64) (string, []byte, error) {
	top := strings.TrimSuffix(strings.TrimSpace(e.Workspace), "/")
	if top == "" {
		return "", nil, output.Errorf(output.CodeUsage, 2,
			"环境 %s 没有配置 $TOP（workspace）；在设置页「站点管理 → SSH 服务器」补上，例如 /u1/toptest/tiptop", e.Name).
			WithHint("或者用 --dir <目录> 指向本地已拉取的镜像目录")
	}
	if limit <= 0 {
		limit = RemoteReadLimit
	}

	conn, err := host.Dial(e.SSHConfig)
	if err != nil {
		return "", nil, output.Errorf(output.CodeConnFailed, 1, "连接服务器失败: %v", err)
	}
	defer conn.Close()

	fs, err := conn.FS()
	if err != nil {
		return "", nil, output.Errorf(output.CodeConnFailed, 1, "打开 SFTP 失败: %v", err)
	}
	defer fs.Close()

	rel := ""
	for _, cand := range cands {
		cand = strings.TrimPrefix(strings.TrimSpace(cand), "/")
		if cand == "" {
			continue
		}
		if st, statErr := fs.Stat(top + "/" + cand); statErr == nil && !st.IsDir() {
			rel = cand
			break
		}
	}
	if rel == "" {
		found, findErr := findRemote(conn, top, base)
		if findErr != nil {
			return "", nil, findErr
		}
		rel = found
	}
	full := top + "/" + rel

	st, err := fs.Stat(full)
	if err != nil || st.IsDir() {
		return "", nil, output.Errorf(output.CodeNotFound, 4, "服务器上没有这个文件: %s", full).
			WithHint("文件名要带后缀；带模块的相对路径更准，如 azz/4gl/p_zta.4gl")
	}
	if st.Size > limit {
		return "", nil, output.Errorf(output.CodeUsage, 2, "文件太大（%s 字节，上限 %s）：%s",
			strconv.FormatInt(st.Size, 10), strconv.FormatInt(limit, 10), full)
	}

	f, err := fs.Open(full)
	if err != nil {
		return "", nil, output.Errorf(output.CodeConnFailed, 1, "打开 %s 失败: %v", full, err)
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, limit))
	if err != nil {
		return "", nil, output.Errorf(output.CodeConnFailed, 1, "读取 %s 失败: %v", full, err)
	}
	return full, data, nil
}

// findRemote 让服务器在 $TOP 下 maxdepth 3 按文件名找；命中多个报 FILE_AMBIGUOUS。
func findRemote(conn *host.SSHConn, top, base string) (string, error) {
	cmd := "find " + shQuote(top) + " -maxdepth 3 -type f -name " + shQuote(base) + " 2>/dev/null | head -n 5"
	out, err := conn.Output(cmd, remoteFindTimeout)
	if err != nil {
		return "", output.Errorf(output.CodeConnFailed, 1, "在服务器上查找 %s 失败: %v", base, err)
	}
	hits := make([]string, 0, 4)
	for _, line := range strings.Split(out, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			hits = append(hits, line)
		}
	}
	switch len(hits) {
	case 0:
		return "", output.Errorf(output.CodeNotFound, 4, "服务器 %s 下没有找到 %s", top, base).
			WithHint("带模块的相对路径更准，如 azz/4gl/" + base)
	case 1:
		return strings.TrimPrefix(hits[0], top+"/"), nil
	}
	return "", output.Errorf("FILE_AMBIGUOUS", 2, "文件 %s 命中 %d 个：%s；请给相对 $TOP 的路径",
		base, len(hits), strings.Join(hits, "、"))
}

func shQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }
