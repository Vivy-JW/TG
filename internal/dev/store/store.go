// Package store 实现设计指南 §3 的工作区布局与存储层（export / apply 共用的落盘与读取）。
//
// 目录布局（固定，不随环境变化）：
//
//	<dir>/
//	  prog.full.4gl        唯一编辑文件：带围栏的渲染文档
//	  manifest.json        人机共读索引
//	  snapshot/
//	    index.json         原包条目清单：name/sha256/size/role
//	    entries/<条目名>    原包每个条目的逐字节拷贝（恢复源；apply 不从这里读）
//	  .tdev/
//	    base.full.4gl      导出时 prog.full.4gl 的字节拷贝（gate1 的比对基线）
//	    base.sha256        该基线的 sha256
//	    regions.json       Region 表 + spans + 文本长度 + 基线摘要
//	    lock               互斥锁（防两个 tdev 进程同时操作）
//	  .git/                export 时 init + 首次 commit；apply 成功后再 commit
//
// 红线约束：
//   - R6：一切落盘都走 AtomicWrite（同目录临时文件 → Rename），禁止先删后建。
//   - R7：时间戳不是功能。manifest.json / regions.json 里**不含任何时钟**，
//     相同输入必须产出逐字节相同的工作区；唯一允许出现时钟的地方是 .tdev/lock。
//
// 错误约定（设计指南 §2 退出码 5）：本包所有环境 / IO 类错误都是 *IOError。
package store

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"tt/internal/dev/model"
	"tt/internal/dev/pkgfile"
)

// 工作区固定文件名 / 目录名（设计指南 §3）。
const (
	FileProg           = "prog.full.4gl"
	FileManifest       = "manifest.json"
	FileSnapshotIndex  = "snapshot/index.json"
	DirSnapshotEntries = "snapshot/entries"
	DirTdev            = ".tdev"
	FileBase           = ".tdev/base.full.4gl"
	FileBaseSha256     = ".tdev/base.sha256"
	FileRegions        = ".tdev/regions.json"
	FileLock           = ".tdev/lock"
	// FileSectionState 是框架解锁状态文件（设计指南 §3）：Locked|Unlocked + pending_unlock + unlocked_by。
	FileSectionState = ".tdev/section-state"
	// FilePrevPkg 是 apply 覆盖前留下的「上一步」包副本（包是原地覆盖的，留一份可回滚）。
	FilePrevPkg   = "prev.tzc"
	FileGitignore = ".gitignore"
)

// ToolName / ToolVersion 写进 manifest 的 tool / tool_version。
//
// 故意用常量而不是构建期注入的变量：manifest.json 必须由输入唯一决定（红线 R7），
// 同一份输入在不同机器 / 不同时间导出必须逐字节相同。
const (
	ToolName    = "tdev tzc"
	ToolVersion = "1.0"
)

const (
	dirMode  = 0o755
	fileMode = 0o644
)

// defaultGitignore 在首次 commit 前写入，避免把锁文件与 build 缓存带进快照。
const defaultGitignore = ".tdev/lock\n.gotmp/\n.gocache/\n"

// IOError 是退出码 5（IO / 环境失败）的错误类型。
type IOError struct {
	Msg string
	Err error
}

func (e *IOError) Error() string {
	if e.Err == nil {
		return e.Msg
	}
	return e.Msg + "：" + e.Err.Error()
}

func (e *IOError) Unwrap() error { return e.Err }

func (e *IOError) ExitCode() int { return 5 }

// ioErr 把底层错误包成 *IOError。
func ioErr(msg string, err error) *IOError { return &IOError{Msg: msg, Err: err} }

// Workspace 是一个已落盘的工作区视图。Dir 是工作区根目录。
type Workspace struct {
	Dir string
}

//---------------------------------------------------------------------------
// 路径
//---------------------------------------------------------------------------

func (w *Workspace) progPath() string       { return filepath.Join(w.Dir, FileProg) }
func (w *Workspace) manifestPath() string   { return filepath.Join(w.Dir, FileManifest) }
func (w *Workspace) snapIndexPath() string  { return filepath.Join(w.Dir, FileSnapshotIndex) }
func (w *Workspace) snapEntriesDir() string { return filepath.Join(w.Dir, DirSnapshotEntries) }
func (w *Workspace) tdevDir() string        { return filepath.Join(w.Dir, DirTdev) }
func (w *Workspace) basePath() string       { return filepath.Join(w.Dir, FileBase) }
func (w *Workspace) baseShaPath() string    { return filepath.Join(w.Dir, FileBaseSha256) }
func (w *Workspace) regionsPath() string    { return filepath.Join(w.Dir, FileRegions) }
func (w *Workspace) lockPath() string       { return filepath.Join(w.Dir, FileLock) }
func (w *Workspace) sectionStatePath() string {
	return filepath.Join(w.Dir, FileSectionState)
}
func (w *Workspace) gitignorePath() string { return filepath.Join(w.Dir, FileGitignore) }

// snapshotEntryPath 把 zip 条目名映射到 snapshot/entries 下的落盘路径。
//
// 条目名按 zip 惯例用 "/" 分隔；这里拒绝绝对路径与 ".." 穿越（恶意 / 损坏包
// 不能借条目名写到工作区之外）。
func (w *Workspace) snapshotEntryPath(name string) (string, error) {
	rel := filepath.FromSlash(name)
	if rel == "" || !filepath.IsLocal(rel) {
		return "", &IOError{Msg: "快照条目名不安全，拒绝路径穿越：" + name}
	}
	return filepath.Join(w.snapEntriesDir(), rel), nil
}

//---------------------------------------------------------------------------
// Create
//---------------------------------------------------------------------------

// Create 落盘一个新工作区（export 用）。fenced 是渲染后的文档全文。
//
// doc 提供 Region 表与 EnvContext；pkg 提供条目快照；manifest / regions 由本函数生成。
// 顺序：目录树 → prog.full.4gl → snapshot → 基线 → regions.json → manifest.json → git。
//
// 若 <dir>/prog.full.4gl 已存在则拒绝（绝不静默覆盖已有工作区）。
func Create(dir string, doc *model.Document, pkg *pkgfile.Package, fenced []byte) (*Workspace, error) {
	if doc == nil {
		return nil, &IOError{Msg: "Create：doc 不能为 nil"}
	}
	if pkg == nil {
		return nil, &IOError{Msg: "Create：pkg 不能为 nil"}
	}
	if strings.TrimSpace(dir) == "" {
		return nil, &IOError{Msg: "Create：工作区目录为空"}
	}
	w := &Workspace{Dir: filepath.Clean(dir)}

	if _, err := os.Stat(w.progPath()); err == nil {
		return nil, &IOError{
			Msg: "拒绝覆盖已存在的工作区：" + w.progPath() + " 已存在（请换一个空目录，或先自行删除该工作区）",
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, ioErr("无法检查工作区目录 "+w.Dir, err)
	}

	for _, d := range []string{w.Dir, w.snapEntriesDir(), w.tdevDir()} {
		if err := os.MkdirAll(d, dirMode); err != nil {
			return nil, ioErr("创建工作区目录失败 "+d, err)
		}
	}

	// ① 唯一编辑文件
	if err := AtomicWrite(w.progPath(), fenced); err != nil {
		return nil, err
	}

	// ② 原包条目快照（逐字节拷贝）+ 条目清单
	idx, serr := w.SnapshotPkg(pkg)
	if serr != nil {
		return nil, serr
	}

	prog := resolveProg(doc, pkg, "")

	// ③ 基线（gate1 的比对基线）
	if err := AtomicWrite(w.basePath(), fenced); err != nil {
		return nil, err
	}
	if err := AtomicWrite(w.baseShaPath(), []byte(model.Sha256Bytes(fenced))); err != nil {
		return nil, err
	}

	// ④ Region 表 + spans
	if err := w.writeRegionsFile(buildRegionsFile(doc, prog, fenced, nil)); err != nil {
		return nil, err
	}

	// ⑤ 人机共读索引
	if err := w.writeManifest(buildManifest(doc, pkg, prog, idx)); err != nil {
		return nil, err
	}

	// ⑥ 框架解锁状态（设计指南 §3）：包级事实优先，unlock 命令之后由 workspace 记意图。
	if err := w.WriteSectionState(&SectionStateFile{
		State:         string(stateOf(doc)),
		PendingUnlock: doc != nil && doc.PendingUnlock,
		UnlockedBy:    unlockedByOf(doc),
	}); err != nil {
		return nil, err
	}

	// ⑦ git：init + 首次 commit（git 不在 PATH 时降级为「不建版本库」，不算失败）
	if err := AtomicWrite(w.gitignorePath(), []byte(defaultGitignore)); err != nil {
		return nil, err
	}
	if err := w.GitInit(); err != nil && !errors.Is(err, errGitMissing) {
		return nil, err
	}
	if _, err := w.GitCommit("tdev export: " + prog); err != nil && !errors.Is(err, errGitMissing) {
		return nil, err
	}

	return w, nil
}

// stateOf / unlockedByOf 由 doc 推导解锁状态（包级事实优先）。
func stateOf(doc *model.Document) model.SectionState {
	if doc == nil {
		return model.SectionLocked
	}
	if doc.SectionState != "" {
		return doc.SectionState
	}
	return model.EffectiveSectionState(doc.Env.SectionFlag, doc.PendingUnlock)
}

func unlockedByOf(doc *model.Document) string {
	if doc == nil {
		return ""
	}
	if doc.UnlockedBy != "" {
		return doc.UnlockedBy
	}
	return model.UnlockedByOf(doc.Env.SectionFlag, doc.PendingUnlock)
}

// entrySha 取条目摘要（Entry 手工构造时 Sha256 可能为空）。
func entrySha(e *pkgfile.Entry) string {
	if e.Sha256 != "" {
		return e.Sha256
	}
	return pkgfile.Sha256Hex(e.Data)
}

// entryRole 给条目分类：tap|tgl|4gl|ver|bdx|other。
func entryRole(name string) string {
	switch path.Ext(name) {
	case ".tap":
		return "tap"
	case ".tgl":
		return "tgl"
	case ".4gl":
		return "4gl"
	case ".bdx":
		return "bdx"
	}
	if name == "ver" {
		return "ver"
	}
	return "other"
}

// resolveProg 定程序名：doc.Prog → .tap 根属性 prog → 旧值 → 包文件名。
func resolveProg(doc *model.Document, pkg *pkgfile.Package, old string) string {
	if doc != nil && strings.TrimSpace(doc.Prog) != "" {
		return strings.TrimSpace(doc.Prog)
	}
	if pkg != nil {
		if _, _, p := tapRootAttrs(pkg); p != "" {
			return p
		}
	}
	if old != "" {
		return old
	}
	if pkg != nil && pkg.Path != "" {
		return progFromPath(pkg.Path)
	}
	return ""
}

//---------------------------------------------------------------------------
// Open / 读取
//---------------------------------------------------------------------------

// Open 打开工作区并校验必需文件齐全；不齐或不可访问 → *IOError（退出码 5）。
func Open(dir string) (*Workspace, error) {
	if strings.TrimSpace(dir) == "" {
		return nil, &IOError{Msg: "Open：工作区目录为空"}
	}
	w := &Workspace{Dir: filepath.Clean(dir)}
	st, err := os.Stat(w.Dir)
	if err != nil {
		return nil, ioErr("工作区目录不可访问 "+w.Dir, err)
	}
	if !st.IsDir() {
		return nil, &IOError{Msg: "Open：" + w.Dir + " 不是目录"}
	}
	required := []string{
		w.progPath(), w.manifestPath(), w.snapIndexPath(),
		w.basePath(), w.baseShaPath(), w.regionsPath(),
	}
	var missing []string
	for _, p := range required {
		fi, err := os.Stat(p)
		if err != nil || fi.IsDir() {
			missing = append(missing, p)
		}
	}
	if len(missing) > 0 {
		return nil, &IOError{
			Msg: "工作区缺少必需文件（不是 tdev 工作区，或 export 未完成）：" + strings.Join(missing, ", "),
		}
	}
	return w, nil
}

// Manifest 读 manifest.json。
func (w *Workspace) Manifest() (*Manifest, error) {
	b, err := os.ReadFile(w.manifestPath())
	if err != nil {
		return nil, ioErr("读取 manifest.json 失败 "+w.manifestPath(), err)
	}
	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, ioErr("解析 manifest.json 失败 "+w.manifestPath(), err)
	}
	return &m, nil
}

// UnlockState 是 workspace 的解锁状态视图。
type UnlockState struct {
	State         model.SectionState
	PendingUnlock bool
	UnlockedBy    string
	// LegacyAllowSec 为真表示这是 v1 工作区（无 section-state 文件），
	// 状态是从旧的 export.allow_sec 迁移来的，调用方应提示重新 export。
	LegacyAllowSec bool
}

// EffectiveUnlockState 合并三处来源，给出唯一解锁状态：
//
//	① .tdev/section-state（v2 工作区的意图记录）
//	② manifest.export.allow_sec（v1 兼容：true 视同 pending_unlock）
//	③ 包级事实（regions.json 里记录的 env.section_flag）
func (w *Workspace) EffectiveUnlockState() (*UnlockState, error) {
	rf, err := w.Regions()
	if err != nil {
		return nil, err
	}
	st, err := w.ReadSectionState()
	if err != nil {
		return nil, err
	}
	u := &UnlockState{}
	if st != nil {
		u.State = model.SectionState(st.State)
		u.PendingUnlock = st.PendingUnlock
		u.UnlockedBy = st.UnlockedBy
	} else {
		// v1 工作区：无状态文件。
		if mf, err := w.Manifest(); err == nil && mf.Export.AllowSecLegacy != nil && *mf.Export.AllowSecLegacy {
			u.PendingUnlock = true
			u.LegacyAllowSec = true
		}
	}
	u.State = model.EffectiveSectionState(rf.Env.SectionFlag, u.PendingUnlock)
	if u.UnlockedBy == "" {
		u.UnlockedBy = model.UnlockedByOf(rf.Env.SectionFlag, u.PendingUnlock)
	}
	return u, nil
}

// SectionStateFile 是 .tdev/section-state 的内容（设计指南 §3 / §4.2）。
//
// 它只记录 workspace 的**意图**：包级事实（TAP 根 section_flag="Y"）永远优先。
type SectionStateFile struct {
	State         string `json:"state"` // Locked | Unlocked
	PendingUnlock bool   `json:"pending_unlock"`
	UnlockedBy    string `json:"unlocked_by"` // pkg | unlock-cmd | ""
}

// ReadSectionState 读 .tdev/section-state。文件不存在时返回 (nil, nil) —— 那是
// v1 旧工作区，调用方按 manifest 的 legacy allow_sec 与包级 section_flag 推导。
func (w *Workspace) ReadSectionState() (*SectionStateFile, error) {
	b, err := os.ReadFile(w.sectionStatePath())
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, ioErr("读取 section-state 失败 "+w.sectionStatePath(), err)
	}
	var s SectionStateFile
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, ioErr("解析 section-state 失败 "+w.sectionStatePath(), err)
	}
	return &s, nil
}

// WriteSectionState 原子写入 .tdev/section-state。
func (w *Workspace) WriteSectionState(s *SectionStateFile) error {
	if s == nil {
		return ioErr("WriteSectionState：s 不能为 nil", nil)
	}
	b, err := model.MarshalJSONStable(s)
	if err != nil {
		return ioErr("序列化 section-state 失败", err)
	}
	return AtomicWrite(w.sectionStatePath(), b)
}

// Regions 读 .tdev/regions.json。
func (w *Workspace) Regions() (*RegionsFile, error) {
	b, err := os.ReadFile(w.regionsPath())
	if err != nil {
		return nil, ioErr("读取 regions.json 失败 "+w.regionsPath(), err)
	}
	var rf RegionsFile
	if err := json.Unmarshal(b, &rf); err != nil {
		return nil, ioErr("解析 regions.json 失败 "+w.regionsPath(), err)
	}
	return &rf, nil
}

// ReadEdited 读 prog.full.4gl（唯一编辑文件）。
func (w *Workspace) ReadEdited() ([]byte, error) {
	b, err := os.ReadFile(w.progPath())
	if err != nil {
		return nil, ioErr("读取 prog.full.4gl 失败 "+w.progPath(), err)
	}
	return b, nil
}

// ReadBase 读 .tdev/base.full.4gl（gate1 比对基线）。
func (w *Workspace) ReadBase() ([]byte, error) {
	b, err := os.ReadFile(w.basePath())
	if err != nil {
		return nil, ioErr("读取基线 base.full.4gl 失败 "+w.basePath(), err)
	}
	return b, nil
}

// WriteEdited 原子写入 prog.full.4gl。
//
// 只写编辑文件，**不动基线**：基线（含 regions.json 里的 base_sha256）由
// UpdateAfterApply 在 apply 成功后一次性刷新，避免出现「基线已换、Region 表未换」
// 的中间态（那会让 gate1 失去意义）。
func (w *Workspace) WriteEdited(b []byte) error {
	return AtomicWrite(w.progPath(), b)
}

// UpdateAfterApply 在 apply 成功后刷新工作区状态：
// 基线（base.full.4gl + base.sha256）、regions.json（新基线摘要 / 长度 / 刷新后的
// Region 区间与 sha256）、manifest.json 的 regions / spans / stats。
//
// pkg / entries / export / anchors（以及 prog / env / kind / module / erpver）保持原样：
// 快照与导出模式在 export 时就固定了，apply 不改变它们。
func (w *Workspace) UpdateAfterApply(doc *model.Document, fenced []byte) error {
	if doc == nil {
		return &IOError{Msg: "UpdateAfterApply：doc 不能为 nil"}
	}
	m, err := w.Manifest()
	if err != nil {
		return err
	}
	old, err := w.Regions()
	if err != nil {
		return err
	}

	if err := AtomicWrite(w.basePath(), fenced); err != nil {
		return err
	}
	if err := AtomicWrite(w.baseShaPath(), []byte(model.Sha256Bytes(fenced))); err != nil {
		return err
	}

	prog := resolveProg(doc, nil, m.Prog)
	if err := w.writeRegionsFile(buildRegionsFile(doc, prog, fenced, old)); err != nil {
		return err
	}

	refreshManifest(m, doc)
	return w.writeManifest(m)
}

// SnapshotPkg 把某个包的每个条目逐字节写入 snapshot/entries/ 并刷新 snapshot/index.json，
// 返回条目清单。Create 与 apply 后的 RefreshPackage 共用。
func (w *Workspace) SnapshotPkg(pkg *pkgfile.Package) ([]EntryRef, error) {
	idx := make([]EntryRef, 0, len(pkg.Entries))
	for _, e := range pkg.Entries {
		p, err := w.snapshotEntryPath(e.Name)
		if err != nil {
			return nil, err
		}
		if err := os.MkdirAll(filepath.Dir(p), dirMode); err != nil {
			return nil, ioErr("创建快照目录失败 "+filepath.Dir(p), err)
		}
		if err := AtomicWrite(p, e.Data); err != nil {
			return nil, err
		}
		idx = append(idx, EntryRef{
			Name:   e.Name,
			Sha256: entrySha(e),
			Size:   len(e.Data),
			Role:   entryRole(e.Name),
		})
	}
	ib, err := model.MarshalJSONStable(idx)
	if err != nil {
		return nil, ioErr("序列化 snapshot/index.json 失败", err)
	}
	if err := AtomicWrite(w.snapIndexPath(), ib); err != nil {
		return nil, err
	}
	return idx, nil
}

// RefreshPackage 在 apply 成功后，把工作区记录的「当前包」换成刚写出的新包：
// 刷新 manifest.pkg.{path,sha256,ver}、manifest.entries 与 snapshot/。
//
// 为什么必须刷新：apply 里有一道「源包自 export 之后未变」的校验（D-6），
// 若 manifest 仍记着**上一版**包的 sha256，那么同一工作区的第二次 apply
// 一定会被误判成「源包已被改动」而拒绝（退出码 5），
// 也就是「改一次 → apply → 再改 → 再 apply」这个正常迭代循环会断掉。
func (w *Workspace) RefreshPackage(pkg *pkgfile.Package) error {
	if pkg == nil {
		return ioErr("RefreshPackage：pkg 不能为 nil", nil)
	}
	m, err := w.Manifest()
	if err != nil {
		return err
	}
	idx, err := w.SnapshotPkg(pkg)
	if err != nil {
		return err
	}
	sha, err := model.Sha256File(pkg.Path)
	if err != nil {
		return ioErr("计算包摘要失败 "+pkg.Path, err)
	}
	m.Pkg.Path = pkg.Path
	m.Pkg.Sha256 = sha
	m.Pkg.Ver = pkg.Ver.Raw
	m.Entries = idx
	return w.writeManifest(m)
}

// BackupPackage 在覆盖前把当前包整文件拷到 .tdev/prev.tzc（尽力而为，失败不阻断）。
// 包是被**原地覆盖**的，所以留一份「上一步」可回滚。
func (w *Workspace) BackupPackage(srcPath string) (string, error) {
	b, err := os.ReadFile(srcPath)
	if err != nil {
		return "", ioErr("读取待覆盖的包失败 "+srcPath, err)
	}
	dst := filepath.Join(w.tdevDir(), FilePrevPkg)
	if err := AtomicWrite(dst, b); err != nil {
		return "", err
	}
	return dst, nil
}

// SnapshotEntry 读原包条目的逐字节快照。
func (w *Workspace) SnapshotEntry(name string) ([]byte, error) {
	p, err := w.snapshotEntryPath(name)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, ioErr("读取快照条目失败 "+p, err)
	}
	return b, nil
}

// SnapshotIndex 读 snapshot/index.json（原包条目清单）。
func (w *Workspace) SnapshotIndex() ([]EntryRef, error) {
	b, err := os.ReadFile(w.snapIndexPath())
	if err != nil {
		return nil, ioErr("读取 snapshot/index.json 失败 "+w.snapIndexPath(), err)
	}
	var idx []EntryRef
	if err := json.Unmarshal(b, &idx); err != nil {
		return nil, ioErr("解析 snapshot/index.json 失败 "+w.snapIndexPath(), err)
	}
	if idx == nil {
		idx = []EntryRef{}
	}
	return idx, nil
}

//---------------------------------------------------------------------------
// 内部写盘
//---------------------------------------------------------------------------

func (w *Workspace) writeManifest(m *Manifest) error {
	b, err := model.MarshalJSONStable(m)
	if err != nil {
		return ioErr("序列化 manifest.json 失败", err)
	}
	return AtomicWrite(w.manifestPath(), b)
}

func (w *Workspace) writeRegionsFile(rf *RegionsFile) error {
	b, err := model.MarshalJSONStable(rf)
	if err != nil {
		return ioErr("序列化 regions.json 失败", err)
	}
	return AtomicWrite(w.regionsPath(), b)
}
