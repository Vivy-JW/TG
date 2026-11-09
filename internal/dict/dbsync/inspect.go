package dbsync

import (
	"os"
	"strings"
	"time"

	"tt/internal/dict/db"
)

// FamilyStatus 本地库里某个数据族的覆盖情况。
type FamilyStatus struct {
	Family
	Present  int      `json:"已同步表数"`
	Rows     int64    `json:"行数"`
	Missing  []string `json:"缺表"`
	Complete bool     `json:"已同步"`
}

// LocalStatus 本地库的整体状态。
type LocalStatus struct {
	Path     string         `json:"本地库"`
	Exists   bool           `json:"存在"`
	Size     int64          `json:"字节"`
	Modified time.Time      `json:"修改时间"`
	Families []FamilyStatus `json:"数据族"`
}

// Summary 一行式摘要,如 "表字典 ✓ · 校验带值 ✗ · 分类码 ✗ …";all 为 true 时返回
// "全部已同步"。供 tt dict --help / 各命令 --help 的「本地数据」提示复用。
func (s *LocalStatus) Summary(keys ...string) string {
	fams := s.Families
	if len(keys) > 0 {
		want := make(map[string]bool, len(keys))
		for _, k := range keys {
			want[k] = true
		}
		fams = nil
		for _, f := range s.Families {
			if want[f.Key] {
				fams = append(fams, f)
			}
		}
	}
	if !s.Exists {
		return "本地库不存在(先 tt dict db sync --env <环境>)"
	}
	parts := make([]string, 0, len(fams))
	all := true
	for _, f := range fams {
		if f.Complete {
			parts = append(parts, f.Name+" ✓")
		} else {
			all = false
			parts = append(parts, f.Name+" ✗")
		}
	}
	if all {
		if len(fams) == len(s.Families) {
			return "全部已同步"
		}
		return strings.Join(parts, " · ")
	}
	return strings.Join(parts, " · ") + " → 补齐: tt dict db sync --env <环境>"
}

// InspectLocal 只读检查本地库对各数据族的覆盖情况(表是否存在 + 行数)。
// 库文件不存在时返回 Exists=false 而不是报错。
func InspectLocal(path string) (*LocalStatus, error) {
	st := &LocalStatus{Path: path, Families: make([]FamilyStatus, 0, len(Families))}
	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			for _, f := range Families {
				st.Families = append(st.Families, FamilyStatus{Family: f, Missing: append([]string(nil), f.Tables...)})
			}
			return st, nil
		}
		return nil, err
	}
	st.Exists, st.Size, st.Modified = true, fi.Size(), fi.ModTime()

	d, err := db.Open(path)
	if err != nil {
		return nil, err
	}
	defer d.Close()

	for _, f := range Families {
		counts, err := d.TableRowCounts(f.Tables)
		if err != nil {
			return nil, err
		}
		fs := FamilyStatus{Family: f, Present: len(counts)}
		for _, t := range f.Tables {
			if n, ok := counts[t]; ok {
				fs.Rows += n
			} else {
				fs.Missing = append(fs.Missing, t)
			}
		}
		fs.Complete = len(fs.Missing) == 0
		st.Families = append(st.Families, fs)
	}
	return st, nil
}
