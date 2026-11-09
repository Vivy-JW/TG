// Package model 定义 tt dev tzc 的领域模型：Region / Document / EnvContext。
//
// 依据设计指南 §1 领域模型、§3 工作区格式、§5.2 权限判定。
// 本包只放**纯数据**，不含 IO、不含业务逻辑，供 synth / fence / verify / split / store 共用。
package model

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
)

//---------------------------------------------------------------------------
// 区间
//---------------------------------------------------------------------------

// ByteRange 是文档坐标系里的半开区间 [Start, End)。
type ByteRange struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

// Len 返回区间长度。
func (r ByteRange) Len() int {
	if r.End <= r.Start {
		return 0
	}
	return r.End - r.Start
}

// Empty 判断是否为空区间。
func (r ByteRange) Empty() bool { return r.End <= r.Start }

// Contains 判断偏移是否落在区间内（半开）。
func (r ByteRange) Contains(off int) bool { return off >= r.Start && off < r.End }

// Overlaps 判断两个区间是否相交。
func (r ByteRange) Overlaps(o ByteRange) bool { return r.Start < o.End && o.Start < r.End }

//---------------------------------------------------------------------------
// Region 种类与拒绝原因
//---------------------------------------------------------------------------

// RegionKind 只有两种互斥类型（设计指南 §1）。
type RegionKind string

const (
	RegionPoint   RegionKind = "point"
	RegionSection RegionKind = "section"
)

// 不可编辑原因码：围栏行给机器码，manifest 给完整中文原因。
const (
	DenyNone                   = ""
	DenyReadonlyAttr           = "readonly-attr"        // 点/区段属性 readonly="Y"
	DenyCiteStd                = "cite-std"             // 非标准件且 cite_std="Y"
	DenyIndFunMismatch         = "ind-fun-mismatch"     // .tzf 独立功能程序行业别不匹配
	DenyIndustryMemo           = "industry-memo"        // 标准环境下 global.memo_industry
	DenyEnvEditEnv             = "env-edit-env"         // edit=c/s 与环境组合不允许
	DenyTopstdMode             = "topstd-mode"          // topstd 编辑模式下按 status/src 受限
	DenyLoginTopstd            = "login-topstd"         // login_user=topstd 且 src=c
	DenySectionAnchor          = "section-anchor"       // other_function/dialog/report 锚点区段
	DenySectionReadonly        = "section-readonly"     // TGL 标记 readonly="Y"
	DenySecLocked              = "section-locked"       // 框架未解开（设计指南 §4.2）
	DenyNotExported            = "not-exported"         // --only 范围外
	DenyStructureUnparsable    = "structure-unparsable" // function.* 前缀但正文非函数块
	DenySectionTemplateMissing = "section-template-missing"
	DenyOnlyPoints             = "only-points" // --only 时区段一律只读
)

// DenyReasonText 给出人类可读的中文原因（写进 manifest.json 的 reason 字段）。
func DenyReasonText(code string) string {
	switch code {
	case DenyNone:
		return ""
	case DenyReadonlyAttr:
		return "区段/点属性 readonly=\"Y\"（一票否决）"
	case DenyCiteStd:
		return "本程序为非标准件且该点 cite_std=\"Y\"（内容由标准程序提供，见 README §4.3 G3）"
	case DenyIndFunMismatch:
		return "独立功能程序（.tzf）的行业别与登录行业别不匹配（README §4.3 G1）"
	case DenyIndustryMemo:
		return "标准环境（env=s）下行业别为 sd/空，global.memo_industry 不可编辑（README §4.3 G2）"
	case DenyEnvEditEnv:
		return "插入点 edit= 属性与环境组合不允许编辑（README §4.3 G5）"
	case DenyTopstdMode:
		return "topstd 编辑模式下仅允许新增点或被标准版修改过的点（README §4.3 G6）"
	case DenyLoginTopstd:
		return "登录用户为 topstd 且 src=c（仅新增点可编辑，README §4.3 G7）"
	case DenySectionAnchor:
		return "框架集合锚点区段（other_function / other_dialog / other_report）强制只读，设计器 ProcessSections 硬编码（CodeEditorManager.cs:1127-1130）"
	case DenySectionReadonly:
		return "TGL 区段标记带 readonly=\"Y\"（设计器不允许编辑）"
	case DenySecLocked:
		return "框架未解开，先执行 tt dev tzc unlock"
	case DenyNotExported:
		return "不在本次 --only 导出范围内：部分导出的工作区只允许改被导出的 Region"
	case DenyStructureUnparsable:
		return "该点正文不是可解析的函数块（设计器装载时会抛 FormtException，已知函数名前缀不保证是函数块）"
	case DenySectionTemplateMissing:
		return "TAP 缺少该区段且无 src=c/src=m 模板可 clone（设计器会抛「找不到区段」）"
	case DenyOnlyPoints:
		return "本次为 --only 部分导出：区段一律只读"
	}
	return code
}

// SectionState 是框架解开状态（设计指南 §4.2）。
//
// 单一事实来源：state = Unlocked 当且仅当 workspace 的 pending_unlock 为真
// **或** 源包 TAP 根 section_flag == "Y"；否则 Locked。
// 它取代了 v1 的 --allow-sec 开关：解锁是一次显式、单向、有代价的状态迁移。
type SectionState string

const (
	SectionLocked   SectionState = "Locked"
	SectionUnlocked SectionState = "Unlocked"
)

// UnlockedBy 区分两种解锁来源。
const (
	UnlockedByPkg       = "pkg"        // 包本来就是解开态（section_flag="Y"）
	UnlockedByUnlockCmd = "unlock-cmd" // 本次会话用 tt dev tzc unlock 迁移
)

// EffectiveSectionState 合并「包级事实」与「workspace 意图」，得到唯一状态。
//
// 包级事实优先：TAP 根 section_flag=="Y" 就是已解开（设计器对这类包不设任何拦截）；
// 否则看 workspace 是否已用 unlock 迁移过（pending_unlock）。
func EffectiveSectionState(sectionFlag, pendingUnlock bool) SectionState {
	if sectionFlag || pendingUnlock {
		return SectionUnlocked
	}
	return SectionLocked
}

// UnlockedBy 推导解锁来源：pkg（包本来就是）/ unlock-cmd（仅靠 pending_unlock）。
func UnlockedByOf(sectionFlag, pendingUnlock bool) string {
	switch {
	case sectionFlag:
		return UnlockedByPkg
	case pendingUnlock:
		return UnlockedByUnlockCmd
	}
	return ""
}

// Region 是文档里的一个可归属区间。
//
// 嵌套规则（偏离设计指南 §4 规则 1，见实现计划 D-2）：
// 真实 TGL 里自订点住在区段内（3 个锚点区段的正文就是注入的点块），
// 因此允许 section → point 的一层嵌套；区段不嵌区段（ProcessSections 按序号配对）。
type Region struct {
	Name     string     `json:"name"`
	Kind     RegionKind `json:"kind"`
	Editable bool       `json:"editable"`
	DenyCode string     `json:"deny_code,omitempty"`
	// Reason 是 DenyCode 的中文解释，写进 manifest 供人/AI 直接读。
	Reason string            `json:"reason,omitempty"`
	Meta   map[string]string `json:"meta,omitempty"`

	// 以下字段在 render 产出里即时计算，写进 .tdev/regions.json 时保留。
	ContentSpan ByteRange `json:"content_span"` // 围栏内内容的绝对区间
	// FullSpan 是区间的完整占位：区段含其两个标记行，点等于 ContentSpan。
	FullSpan ByteRange `json:"full_span"`
	// BeginFence / EndFence 是两条围栏行的字节区间。
	// 设计指南 §4 规则 2：围栏行本身计入「围栏外」，AI 不许增删改围栏行。
	// gate1 靠这两个区间把该规则变成机械校验。
	BeginFence ByteRange `json:"begin_fence"`
	EndFence   ByteRange `json:"end_fence"`
	Sha256     string    `json:"sha256"` // 导出时内容摘要

	// StructureSpans 是**绝对不可改**的结构行区间。v2 起语义收窄为「只有 END 行」：
	// 签名行移入 SignatureSpan（结构事务治理，gate1 豁免 + gate2 V1/V5 校验），
	// 描述块移入 DescSpan（可编辑 + gate2 V6 形态校验）。
	StructureSpans []ByteRange `json:"structure_spans,omitempty"`
	// SignatureSpan 是签名行区间（`[PUBLIC|PRIVATE] FUNCTION|DIALOG|REPORT 名(参数…)`）。
	// 它是「结构事务」的输入之一：允许改，但必须过 V1/V5。
	SignatureSpan *ByteRange `json:"signature_span,omitempty"`
	// DescSpan 是描述块区间（签名行之前的连续 `#` 行）。可编辑；形态由 V6 校验；
	// 应用后 Desc 字段由块内容重算。
	DescSpan *ByteRange `json:"desc_span,omitempty"`
	// BodySpan 是真正可写的正文本体（自订定义点）；裸名点为整块内容。
	BodySpan *ByteRange `json:"body_span,omitempty"`

	// Fn / Scope / Desc 是设计器 FunctionInfoWindow 弹窗里的三个字段（设计指南 §4.1）。
	// 它们渲染进围栏行，属「结构事务区」：允许改，但立即触发 V1–V7 校验。
	Fn    string `json:"fn,omitempty"`
	Scope string `json:"scope,omitempty"`
	Desc  string `json:"desc,omitempty"`
	// Plain 表示这是裸名插入点（`[PLAIN]`）：没有函数身份，整块即正文，
	// 除 READONLY 判定外不做结构校验。特殊的是自订定义点，不是所有 point。
	Plain bool `json:"plain,omitempty"`

	// TglTag 是 TGL 里占位符原文（{<point name="X"/>}）；锚点注入点为 nil。
	// 区段回写时必须把内嵌点折叠回这段原文，否则破坏 TglTag 折叠（红线 R4）。
	TglTag []byte `json:"tgl_tag,omitempty"`
	// Origin = placeholder | anchor-injected。
	Origin string `json:"origin"`
	// AnchorType 仅对锚点区段有意义：function | dialog | report。
	AnchorType string `json:"anchor_type,omitempty"`
	// Append 表示该区段允许追加新 point 块（仅 3 个锚点区段）。
	Append bool `json:"append,omitempty"`
	// PadEOL 记录 render 时为对齐行尾而合成的换行；parse 时据此剥回，保证逐字节可逆。
	PadEOL bool `json:"pad_eol,omitempty"`
	// Deleted 表示该 Region 的围栏块在编辑后的文档里被整块删除（设计指南 §4 规则 5）。
	Deleted bool `json:"deleted,omitempty"`

	Children []*Region `json:"children,omitempty"`
}

// DeepCopy 返回深拷贝（fence 需要在不改基线的前提下改区间）。
func (r *Region) DeepCopy() *Region {
	c := *r
	if r.Meta != nil {
		c.Meta = make(map[string]string, len(r.Meta))
		for k, v := range r.Meta {
			c.Meta[k] = v
		}
	}
	if r.TglTag != nil {
		c.TglTag = append([]byte(nil), r.TglTag...)
	}
	if r.StructureSpans != nil {
		c.StructureSpans = append([]ByteRange(nil), r.StructureSpans...)
	}
	if r.BodySpan != nil {
		b := *r.BodySpan
		c.BodySpan = &b
	}
	if r.SignatureSpan != nil {
		b := *r.SignatureSpan
		c.SignatureSpan = &b
	}
	if r.DescSpan != nil {
		b := *r.DescSpan
		c.DescSpan = &b
	}
	if r.Children != nil {
		c.Children = make([]*Region, len(r.Children))
		for i, ch := range r.Children {
			c.Children[i] = ch.DeepCopy()
		}
	}
	return &c
}

// Walk 深度优先遍历自身与所有子区间。
func (r *Region) Walk(fn func(*Region)) {
	fn(r)
	for _, ch := range r.Children {
		ch.Walk(fn)
	}
}

// Span 是不能归属任何 Region 的字节段（文件头尾注释、区段之间的空行等）。
//
// 偏离设计指南 §4 规则 6（见实现计划 D-3）：真实包里必然存在这类字节，
// 禁止它们会让 105/105 个真实包直接失败。它们**同样参与围栏外字节恒等校验**，
// 机械安全性不变，只是从「报错」改为「登记 + info」。
type Span struct {
	Name   string    `json:"name"` // prefix / suffix / gap.<n>
	Span   ByteRange `json:"span"`
	Sha256 string    `json:"sha256"`
}

// EnvContext 是权限判定所需的全部输入。
//
// 比设计指南 §5.2 的列表多 3 个字段（is_standard / is_ind_fun / section_flag），
// 因为设计器决策链本身要用到它们。
type EnvContext struct {
	Env          string `json:"env"`    // TAP 根 env：c=客制 / s=标准
	Topind       string `json:"topind"` // TAP 根 topind：行业别
	LoginUser    string `json:"login_user"`
	ProgType     string `json:"prog_type"`      // TAP 根 type
	IsStandard   bool   `json:"is_standard"`    // std_prog == prog
	IsIndFun     bool   `json:"is_ind_fun"`     // .tzf
	SectionFlag  bool   `json:"section_flag"`   // section_flag == "Y"
	IsTopstdMode bool   `json:"is_topstd_mode"` // 偏好设置 TopstdEditPermission（CLI 默认关）
	// StdSectionVerify = TAP 根 std_section_verify=="Y"：标准环境下非 topstd 用户的
	// **解开授权事实**（服务器端 adzi052 授予，README §3.11）。tdev 只消费，不写入。
	StdSectionVerify bool `json:"std_section_verify"`
	// SectionState 取代 v1 的 allow_sec：Locked 时所有 SectionRegion 强制只读（§4.2）。
	SectionState SectionState `json:"section_state"`
}

// Document 是合成后的完整文档（= 设计器编辑器里看到的内容）。
type Document struct {
	// Text 是文档全文（未加围栏）。
	Text []byte
	// Regions 是顶层区间序列（有序，含嵌套子区间）。
	Regions []*Region
	// Spans 是未归属字节段（prefix / gap / suffix），有序。
	Spans []Span
	// Env 是权限判定上下文。
	Env EnvContext
	// Prog 是程序名（TAP 根 prog）。
	Prog string
	// PkgPath / PkgSha256 / Ver 用于 manifest。
	PkgPath   string
	PkgSha256 string
	Ver       string
	// SectionState / PendingUnlock / UnlockedBy / Only 记录 workspace 的解锁状态与导出模式。
	SectionState  SectionState
	PendingUnlock bool
	UnlockedBy    string
	Only          []string
	// Anchors 记录 other.function / other.dialog / other.report 是否存在。
	Anchors map[string]bool
}

// AllRegions 按文档顺序展开所有区间（含嵌套，深度优先）。
func (d *Document) AllRegions() []*Region {
	var out []*Region
	for _, r := range d.Regions {
		r.Walk(func(x *Region) { out = append(out, x) })
	}
	return out
}

// FindRegion 按名字找区间（含嵌套）。
func (d *Document) FindRegion(name string) *Region {
	for _, r := range d.AllRegions() {
		if r.Name == name {
			return r
		}
	}
	return nil
}

// EditableSpans 返回全部可写字节区间（已排序）。
//
// 这是 gate1 的核心：**所有不属于这些区间的字节都必须逐字节相等**。
// 自订定义点的可写区间 = 描述块（设计指南 §4 规则 3，形态由 V6 校验）+ 正文本体；
// 签名行由结构事务治理（gate1 豁免、gate2 V1/V5 校验）；END 行绝对不可写。
func (d *Document) EditableSpans() []ByteRange {
	var out []ByteRange
	for _, r := range d.AllRegions() {
		if !r.Editable {
			continue
		}
		if r.DescSpan != nil && !r.DescSpan.Empty() {
			out = append(out, *r.DescSpan)
		}
		if r.BodySpan != nil {
			if !r.BodySpan.Empty() {
				out = append(out, *r.BodySpan)
			}
			continue
		}
		if !r.ContentSpan.Empty() {
			out = append(out, r.ContentSpan)
		}
	}
	sortRanges(out)
	return out
}

func sortRanges(rs []ByteRange) {
	for i := 1; i < len(rs); i++ {
		for j := i; j > 0 && rs[j].Start < rs[j-1].Start; j-- {
			rs[j], rs[j-1] = rs[j-1], rs[j]
		}
	}
}

// OwnBytes 返回区间的「自身字节」：内容剔除所有子区间（含其行尾换行）后的字节。
//
// 区段与子点的责任分离全靠它：
//   - 子点区间有各自的可编辑性与授权，子点的合法改动不应把父区段判成被改；
//   - 父区段自己的任何字节改动都必须在自身字节里暴露出来；
//   - I3（区段正文不得含展开后的函数块）也必须只看自身字节 ——
//     锚点区段正文里本来就有**子区间形式**的点块，那是合法的。
func (d *Document) OwnBytes(r *Region) []byte {
	if r == nil {
		return nil
	}
	if len(r.Children) == 0 {
		if r.ContentSpan.End > len(d.Text) || r.ContentSpan.Start > r.ContentSpan.End {
			return nil
		}
		return d.Text[r.ContentSpan.Start:r.ContentSpan.End]
	}
	var out []byte
	pos := r.ContentSpan.Start
	for _, ch := range r.Children {
		s := ch.FullSpan.Start
		e := d.exciseEnd(ch)
		if s > pos && s <= len(d.Text) {
			out = append(out, d.Text[pos:s]...)
		}
		if e > pos {
			pos = e
		}
	}
	if r.ContentSpan.End > pos && r.ContentSpan.End <= len(d.Text) {
		out = append(out, d.Text[pos:r.ContentSpan.End]...)
	}
	return out
}

// exciseEnd 返回剔除子区间时应吃掉的结束位置（含其后的一个行尾）。
func (d *Document) exciseEnd(ch *Region) int {
	e := ch.FullSpan.End
	if e < len(d.Text) && d.Text[e] == '\n' {
		return e + 1
	}
	if e+1 < len(d.Text) && d.Text[e] == '\r' && d.Text[e+1] == '\n' {
		return e + 2
	}
	return e
}

// NormalizeEOL 把 CRLF / 单独的 CR 统一成 LF。
//
// 用途：**行尾等价比较**。工作区文件的受保护字节（围栏行、只读区内容、
// 结构行、围栏外字节）tdev 从不写回包 —— `tapfile.Rewrite` 只在原 `.tap` 的
// 字节区间上做替换。所以「编辑器把行尾归一」不应被当成改动。
// 真机事故复盘：VS Code 保存 capt110 时把 866 个 CRLF 全换成 LF，
// 导致 596/605 个 Region 被误判为改动、apply 以退出码 4 拒绝。
func NormalizeEOL(b []byte) []byte {
	if bytes.IndexByte(b, '\r') < 0 {
		return b
	}
	out := make([]byte, 0, len(b))
	for i := 0; i < len(b); i++ {
		if b[i] == '\r' {
			if i+1 < len(b) && b[i+1] == '\n' {
				continue // CRLF：CR 丢掉，LF 由下一轮原样带出
			}
			out = append(out, '\n') // 单独 CR → LF
			continue
		}
		out = append(out, b[i])
	}
	return out
}

// EqualEOL 判断两段字节在行尾归一后是否相等。
func EqualEOL(a, b []byte) bool {
	if bytes.Equal(a, b) {
		return true
	}
	return bytes.Equal(NormalizeEOL(a), NormalizeEOL(b))
}

// Sha256Bytes 计算摘要（小写 hex）。
func Sha256Bytes(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

// Sha256File 计算文件摘要。
func Sha256File(p string) (string, error) {
	b, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	return Sha256Bytes(b), nil
}

// MarshalJSONStable 以固定两空格缩进序列化（manifest / regions.json 用）。
func MarshalJSONStable(v any) ([]byte, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}
