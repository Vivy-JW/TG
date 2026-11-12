package outline

import (
	"path/filepath"
	"strings"

	"tt/internal/cli/common"
	"tt/internal/config"
	"tt/internal/dict"
)

// remoteSource 直接从服务器读一个 .4gl，不落本地：
// $TOP 取环境配置里的 workspace，找文件的具体做法在 common.ReadRemoteFile。
//
// 为什么要有这条路：GP 的一个 $TOP 下有 1.4 万个 4gl（约 700 MB），
// 全量镜像要十几分钟，而看一个大纲只需要一个文件。
func remoteSource(e config.NamedSsh, file string) (*sourceInputs, error) {
	rel := strings.TrimPrefix(filepath.ToSlash(strings.TrimSpace(file)), "/")
	base := filepath.Base(rel)

	cands := make([]string, 0, 3)
	if strings.Contains(rel, "/") {
		cands = append(cands, rel)
	} else {
		// 纯文件名：先用本地字典里记的模块拼 <模块>/4gl/<名>。
		if mod, ok := dict.LookupModule(e, strings.TrimSuffix(base, filepath.Ext(base))); ok && strings.TrimSpace(mod) != "" {
			cands = append(cands, strings.ToLower(strings.TrimSpace(mod))+"/4gl/"+base)
		}
	}

	full, data, err := common.ReadRemoteFile(e, cands, base, common.RemoteReadLimit)
	if err != nil {
		return nil, err
	}
	top := strings.TrimSuffix(strings.TrimSpace(e.Workspace), "/")
	return &sourceInputs{Path: full, Mode: "remote", Root: top, Env: e.Name, Content: data}, nil
}
