package web

// 设置页「技能」页签的后端：第三方技能登记的增删查 + 「装到 agent 目录」。
//
// 与 `tt install skills` 共用 internal/skill 这一份实现 —— 校验、冲突判定、assets 指针、
// 复制全都不是这里写的。这一层只负责三件事：读写配置、把路径绝对化、把结果原样带出去。
//
// 纪律与「检查更新」相同：**安装这个动作只由人点按钮触发**，服务启动不装、也没有
// 定时任务；页面上那一次点击就是那次显式命令。

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"tt/internal/config"
	"tt/internal/skill"
)

// officialSkillsDirFn 是官方技能目录：<exe 目录>\skills，与命令行走同一个落点。
// 可替换（测试用）：否则每个用例都得在测试二进制的旁边造一棵技能树。
var officialSkillsDirFn = defaultOfficialSkillsDir

func defaultOfficialSkillsDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("取不到当前可执行文件路径: %w", err)
	}
	return filepath.Join(filepath.Dir(exe), "skills"), nil
}

// skillRow 是设置页表格里的一行。
type skillRow struct {
	skill.Info        // origin / name / description / dir / index / state / why
	Label      string `json:"label,omitempty"`
	Path       string `json:"path,omitempty"` // 登记表里的原值（未绝对化）
}

// hSkillsGet 返回技能总览：官方（只读）+ 第三方（含磁盘判定）+ 上次装到过哪些目录。
// 不联网、不写盘。
func (s *Server) hSkillsGet(w http.ResponseWriter, r *http.Request) {
	cfgPath, err := s.configPath(true)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	root, err := config.Load(cfgPath)
	if err != nil {
		// 配置坏了要如实说：此时"登记了哪些第三方"无从回答，静默当空表会误导。
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	out := map[string]any{"ok": true, "config": cfgPath}

	var officialNames []string
	if dir, derr := officialSkillsDirFn(); derr != nil {
		out["officialError"] = derr.Error()
	} else if infos, _, ierr := skill.InspectOfficial(dir); ierr != nil {
		// 官方技能不合规/目录缺失不是"页面坏了"，是这个包的 skills/ 有问题 ——
		// 作为一条状态带出去，第三方那一栏照常工作。
		out["officialError"] = ierr.Error()
		out["officialDir"] = dir
	} else {
		official := make([]skill.Info, 0, len(infos))
		for _, info := range infos {
			officialNames = append(officialNames, info.Name)
			official = append(official, info)
		}
		out["official"] = official
		out["officialDir"] = dir
	}

	entries := root.Skills.Entries()
	paths := make([]string, 0, len(entries))
	for _, e := range entries {
		paths = append(paths, e.Path)
	}
	infos := skill.InspectExternal(paths, officialNames)
	rows := make([]skillRow, 0, len(infos))
	for i, info := range infos {
		row := skillRow{Info: info}
		if i < len(entries) {
			row.Label = entries[i].Label
			row.Path = entries[i].Path
		}
		rows = append(rows, row)
	}
	out["externals"] = rows

	writeJSON(w, http.StatusOK, out)
}

// hSkillsAdd POST /api/skills/add {path, label}：登记一个技能目录。
//
// 添加时**当场校验**（与安装时不同）：
//   - 路径不存在 → 允许存下（移动硬盘没插也要能先登记），页面标红；
//   - 有目录但内容不合规 / 与官方或已登记的撞名 → 直接拒。这两种状态装出去一定会被
//     跳过，让用户先看到原因，比"装完发现少一个"省事。
func (s *Server) hSkillsAdd(w http.ResponseWriter, r *http.Request) {
	cfgPath, err := s.configPath(true)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	var req struct {
		Path  string `json:"path"`
		Label string `json:"label,omitempty"`
	}
	if err := decodeBody(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	raw := strings.TrimSpace(req.Path)
	if raw == "" {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("path 不能为空"))
		return
	}
	abs, err := filepath.Abs(raw)
	if err != nil {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("把路径 %s 绝对化失败: %w", raw, err))
		return
	}

	root, err := config.Load(cfgPath)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	officialNames := officialSkillNames()
	existing := root.Skills.Paths()

	// 已经登记过（大小写不敏感）就别重复加 —— 返回当前状态，而不是制造一条重复登记。
	for _, p := range existing {
		if skill.SamePath(p, abs) {
			info := skill.InspectExternal([]string{p}, officialNames)[0]
			writeJSON(w, http.StatusOK, map[string]any{
				"ok": true, "unchanged": true, "entry": skillRow{Info: info, Label: req.Label, Path: p},
			})
			return
		}
	}

	// 追加到末尾再判一次：冲突规则（先登记者优先、官方优先）与安装时完全一致。
	cand := append(append([]string{}, existing...), abs)
	info := skill.InspectExternal(cand, officialNames)[len(cand)-1]
	switch info.State {
	case skill.StateInvalid, skill.StateConflict:
		writeErr(w, http.StatusBadRequest, fmt.Errorf("%s 不能登记：%s", abs, info.Why))
		return
	}

	label := strings.TrimSpace(req.Label)
	if label == "" {
		label = info.Name // 失联时是空串，页面上就显示路径
	}
	err = config.Edit(cfgPath, nil, func(root map[string]any) error {
		sec := config.SectionOrEmpty(root, "skills")
		item := map[string]any{"path": abs}
		if label != "" {
			item["label"] = label
		}
		list, _ := sec["externals"].([]any)
		sec["externals"] = append(list, item)
		root["schemaVersion"] = config.SchemaVersion
		return nil
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok": true, "entry": skillRow{Info: info, Label: label, Path: abs},
		"warning": missingNote(info),
	})
}

// hSkillsRemove POST /api/skills/remove {path}：从登记表里去掉一条，不碰磁盘。
// 幂等：本来就没有也返回 ok（removed=false）。
func (s *Server) hSkillsRemove(w http.ResponseWriter, r *http.Request) {
	cfgPath, err := s.configPath(true)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	var req struct {
		Path string `json:"path"`
	}
	if err := decodeBody(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	target := strings.TrimSpace(req.Path)
	if target == "" {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("path 不能为空"))
		return
	}

	var removed bool
	err = config.Edit(cfgPath, nil, func(root map[string]any) error {
		sec := config.SectionOrEmpty(root, "skills")
		list, _ := sec["externals"].([]any)
		kept := make([]any, 0, len(list))
		for _, it := range list {
			m, _ := it.(map[string]any)
			p, _ := m["path"].(string)
			if p != "" && skill.SamePath(p, target) {
				removed = true
				continue
			}
			kept = append(kept, it)
		}
		sec["externals"] = kept
		root["schemaVersion"] = config.SchemaVersion
		return nil
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "removed": removed})
}

// officialSkillNames 取官方技能名（撞名判定用）。官方目录读不到时返回 nil ——
// 那种情况下"和官方撞名"这件事判不出来，页面上的 officialError 已经说清了原因。
func officialSkillNames() []string {
	dir, err := officialSkillsDirFn()
	if err != nil {
		return nil
	}
	infos, _, err := skill.InspectOfficial(dir)
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(infos))
	for _, i := range infos {
		out = append(out, i.Name)
	}
	return out
}

// missingNote 是「允许但要说一声」的那种情况：路径暂时不在。
func missingNote(info skill.Info) string {
	if info.State == skill.StateMissing {
		return "路径现在访问不到，已保存登记。路径回来后 tt install skills 会自动装上它。"
	}
	return ""
}
