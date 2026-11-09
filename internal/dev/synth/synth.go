// Package synth 实现设计指南 §5.2「合成层 —— Package → Document」，
// 逐步复刻设计器 CodeEditorManager.LoadContent 的行为，并给出权限判定链。
//
// 依据：
//   - 设计指南 §5.2、§1 领域模型
//   - 反编译源码（标记 / 根元素 / point 属性语义、合成管线六步、point 名字命名空间三层、
//     可编辑性决策链）：
//     CodeEditWindow/Helper/CodeEditorManager.cs:1070-1080（LoadContent）
//     CodeEditorManager.cs:1142-1176（ProcessFunctionTypes，锚点展开）
//     CodeEditorManager.cs:1179-1227（ProcessAddPoints，占位符替换/造空点/记 TglTag）
//     CodeEditorManager.cs:1083-1139（ProcessSections，按序号配对 + 模板 clone + 强制只读）
//     Infrastructure/.../ProgramInformation.cs:655-666（Initial 造空点）
//     Infrastructure/.../AddPointModel.cs:691-747（点权限链 G1–G7）
//     Infrastructure/.../SectionModel.cs:101-133（区段权限链 S1–S5）
package synth

import (
	"bytes"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"tt/internal/dev/fgl"
	"tt/internal/dev/model"
	"tt/internal/dev/pkgfile"
	"tt/internal/dev/tapfile"
	"tt/internal/dev/tglfile"
)

// ExitCodeError 让 CLI 把错误映射为退出码。
type ExitCodeError interface {
	error
	ExitCode() int
}

// SynthesisError = 退出码 2（包结构无法装载，与「包格式错」同级）。
type SynthesisError struct {
	Msg    string
	Detail []string
}

func (e *SynthesisError) Error() string {
	if len(e.Detail) == 0 {
		return e.Msg
	}
	return e.Msg + "：" + strings.Join(e.Detail, "; ")
}

func (e *SynthesisError) ExitCode() int { return 2 }

// Options 是合成选项（对应设计指南 §5.2 的 opts）。
type Options struct {
	// Only 是部分导出：只允许改这些点；空 = 全量导出。
	Only []string
	// SectionState 取代 v1 的 AllowSec（设计指南 §4.2）：
	// Locked → 所有 SectionRegion 强制只读；Unlocked → 按设计器区段决策链判定。
	SectionState model.SectionState
	// PendingUnlock 为真表示 workspace 已用 `tt dev tzc unlock` 迁移过（包还没落盘）。
	PendingUnlock bool
	// TopstdMode 只在复现设计器 topstd 决策链的测试里开启；CLI 默认关。
	TopstdMode bool
}

// designer 的 markRex / envRex（CodeEditorManager.cs:1712/1715，无闭合引号要求）。
var (
	reMark        = regexp.MustCompile(`mark="(\w)`)
	reEditE       = regexp.MustCompile(`edit="(\w)`)
	rePlaceholder = regexp.MustCompile(`\{<point\s+name="(\S+)".*\s*/>\}`)
)

// pointType 由点名前缀推导（AddPointModel.cs:545-580）。
func pointType(name string) (tglfile.AnchorKind, bool) {
	switch {
	case strings.HasPrefix(name, "function."):
		return tglfile.AnchorFunction, true
	case strings.HasPrefix(name, "dialog."):
		return tglfile.AnchorDialog, true
	case strings.HasPrefix(name, "report."):
		return tglfile.AnchorReport, true
	}
	return "", false
}

// statusOf 把 TAP status 属性映射为设计器 Status 语义：
// SpecStatus: c=CREATE d=DELETE u=MODIFY 其它=NULL。
func statusOf(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "c", "create":
		return "CREATE"
	case "d", "delete":
		return "DELETE"
	case "u", "modify":
		return "MODIFY"
	}
	return "NULL"
}

const maxInt = int(^uint(0) >> 1)

// sortIndex 复刻 AddPointModel.SortIndex：order 属性整数，缺失/空白 = int.MaxValue。
func sortIndex(el *tapfile.Element) int {
	v, ok := el.Attr("order")
	if !ok {
		return maxInt
	}
	v = strings.TrimSpace(v)
	if v == "" {
		return maxInt
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return maxInt
	}
	return n
}

// readEnv 从 TAP 根读出权限判定所需的上下文。
func readEnv(d *tapfile.Doc, pkg *pkgfile.Package, state model.SectionState, topstd bool) model.EnvContext {
	prog := d.RootAttrOr("prog", "")
	std := d.RootAttrOr("std_prog", "")
	sf := d.RootAttrOr("section_flag", "")
	return model.EnvContext{
		Env:          d.RootAttrOr("env", ""),
		Topind:       d.RootAttrOr("topind", ""),
		LoginUser:    d.RootAttrOr("login_user", ""),
		ProgType:     d.RootAttrOr("type", ""),
		IsStandard:   std != "" && std == prog,
		IsIndFun:     pkg.IsIndFun,
		SectionFlag:  strings.EqualFold(sf, "Y"),
		IsTopstdMode: topstd,
		// std_section_verify 是服务器端 adzi052 授予的解开授权事实（只读消费）。
		StdSectionVerify: strings.EqualFold(d.RootAttrOr("std_section_verify", ""), "Y"),
		SectionState:     state,
	}
}

//----------------------------------------------------------------------------
// 框架解锁闸门（设计指南 §4.2）
//----------------------------------------------------------------------------

// 设计器原文文案，逐字引用（langs/zh-cn.xaml:448 / :641 / :688 / :689）。
// 不改写、不翻译：用户看到的就是设计器里会看到的那句话。
const (
	TextAfterChangeSession = "您即将解开此程序的框架! 变更后，规格的任何调整都不会再产生对应的程序代码，请问是否确定要变更？"
	TextPropertyChange     = "注意：设定变更后，必须上传程序才会生效。"
	TextSectionVerify      = "您没有解开此程式框架的授权，请找有权限的主管透过adzi052(解开程式框架授权作业)进行授权"
	TextSectionVerifyWarn  = "注意:取得解开程式框架授权后，必须重新下载程式才会生效"
)

// DeniedError = 退出码 4（写入被拒：权限不足）。
type DeniedError struct {
	Msg    string
	Detail []string
}

func (e *DeniedError) Error() string {
	if len(e.Detail) == 0 {
		return e.Msg
	}
	return e.Msg + "：" + strings.Join(e.Detail, "; ")
}

func (e *DeniedError) ExitCode() int { return 4 }

// UnlockDecision 是解锁闸门的判定结果。
type UnlockDecision struct {
	Allow   bool
	Reason  string   // 拒绝原因（中文）
	Detail  []string // 要原样打印给用户的原文
	NeedYes bool     // 是否必须 --yes 二次确认（代价警告分支）
}

// CheckUnlockPermission 逐条重放设计器 checkBox_PreviewMouseLeftButtonDown
// （CodeEditorMainWindow.xaml.cs:557-588）的三分支。
//
// R9：**不提供任何绕过** —— 没有 --force，不改 env/login_user，
// 也不写 std_section_verify 假装有授权（那是服务器 adzi052 的权限域）。
func CheckUnlockPermission(env model.EnvContext, yes bool) UnlockDecision {
	// 分支①：包本来就是解开态（IsSectionModify）→ 设计器直接进，不弹框
	if env.SectionFlag {
		return UnlockDecision{Allow: true, Reason: "包本来就是解开态（section_flag=\"Y\"）"}
	}
	// 分支②：标准环境 + 非 topstd → 看 adzi052 授权事实
	if env.Env == "s" && env.LoginUser != "topstd" {
		if !env.StdSectionVerify {
			return UnlockDecision{
				Allow:  false,
				Reason: "无解开框架授权",
				Detail: []string{TextSectionVerify, TextSectionVerifyWarn},
			}
		}
		if !yes {
			return UnlockDecision{
				Allow:   false,
				Reason:  "需要二次确认",
				Detail:  []string{TextAfterChangeSession, TextPropertyChange},
				NeedYes: true,
			}
		}
		return UnlockDecision{Allow: true, Reason: "已获 adzi052 授权（std_section_verify=\"Y\"）"}
	}
	// 分支③：客制环境或 topstd → 代价警告 + 二次确认
	if !yes {
		return UnlockDecision{
			Allow:   false,
			Reason:  "需要二次确认",
			Detail:  []string{TextAfterChangeSession, TextPropertyChange},
			NeedYes: true,
		}
	}
	return UnlockDecision{Allow: true, Reason: "客制环境/topstd，已确认代价警告"}
}

//----------------------------------------------------------------------------
// 权限判定链
//----------------------------------------------------------------------------

// ResolvePoint 是 AddPointModel.IsEditable 的逐条移植（AddPointModel.cs:691-747）。
func ResolvePoint(name string, meta map[string]string, env model.EnvContext, selfDef, parseOK bool) (bool, string) {
	src := strings.ToLower(meta["src"])
	status := statusOf(meta["status"])

	// G1 独立功能程序行业别不匹配
	if env.IsIndFun && meta["ind_fun"] != env.Topind && name != "global.memo_industry" {
		return false, model.DenyIndFunMismatch
	}
	// G2 标准环境 + 行业别 sd/空 + 行业 memo
	if (env.Topind == "sd" || env.Topind == "") && env.Env == "s" && name == "global.memo_industry" {
		return false, model.DenyIndustryMemo
	}
	// G3 非标准件且 cite_std="Y"
	if !env.IsStandard && strings.EqualFold(meta["cite_std"], "Y") {
		return false, model.DenyCiteStd
	}
	// G4 readonly 一票否决
	if strings.EqualFold(meta["readonly"], "Y") {
		return false, model.DenyReadonlyAttr
	}
	// G5 插入点 edit= 与环境组合
	switch strings.ToLower(meta["edit"]) {
	case "c":
		if env.Env == "s" {
			return false, model.DenyEnvEditEnv
		}
		if env.Env == "c" && env.IsTopstdMode {
			return false, model.DenyEnvEditEnv
		}
	case "s":
		if env.Env == "c" && !env.IsTopstdMode && src != "c" {
			return false, model.DenyEnvEditEnv
		}
	}
	// G6 topstd 编辑模式
	if env.IsTopstdMode {
		switch src {
		case "c":
			if status == "CREATE" {
				return true, model.DenyNone
			}
			return false, model.DenyTopstdMode
		case "s", "m":
			if status == "NULL" || (strings.EqualFold(meta["modi_by_topstd"], "Y") && status == "MODIFY") {
				return true, model.DenyNone
			}
			return false, model.DenyTopstdMode
		}
	}
	// G7 登录用户门禁
	if env.LoginUser == "topstd" && src == "c" && status != "CREATE" {
		return false, model.DenyLoginTopstd
	}
	// 自订定义点正文解析不出来 → 设计器装载会抛异常，tdev 一律不写回
	if selfDef && !parseOK {
		return false, model.DenyStructureUnparsable
	}
	return true, model.DenyNone
}

// ResolveSection 是 SectionModel.IsEditable 的逐条移植（SectionModel.cs:101-133），
// 前面叠加 tdev 自身的政策层（锚点区段永不写、Locked 一律只读）。
//
// 设计指南 §4.2：**解锁改变的是门，不是所有房间** —— Unlocked 之后仍要逐条走设计器链。
func ResolveSection(name string, isReadonly bool, meta map[string]string, env model.EnvContext, prog string) (bool, string) {
	src := strings.ToLower(meta["src"])
	status := statusOf(meta["status"])

	// 政策层 1：三个集合锚点区段的正文是注入的点块，写它等于把展开后的函数写进框架（红线 R4）。
	// 设计器在 type="G" + section_flag=Y 时对 other_dialog 有个例外，我们**不复刻**这个例外。
	if _, isAnchor := tglfile.AnchorSectionID(prog, name); isAnchor {
		return false, model.DenySectionAnchor
	}
	// 政策层 2：框架未解开（设计指南 §4.2）。先 tt dev tzc unlock。
	if env.SectionState != model.SectionUnlocked {
		return false, model.DenySecLocked
	}
	// S1 type="G" 且已解开框架
	if env.ProgType == "G" && env.SectionFlag {
		if name != prog+".other_function" && name != prog+".other_report" {
			return true, model.DenyNone
		}
		return false, model.DenySectionAnchor
	}
	// S2 运行期只读（3 锚点硬编码 或 TGL 标记 readonly="Y"）
	if isReadonly {
		return false, model.DenySectionReadonly
	}
	// S3 readonly 属性
	if strings.EqualFold(meta["readonly"], "Y") {
		return false, model.DenyReadonlyAttr
	}
	// S4 topstd 模式
	if env.IsTopstdMode {
		switch src {
		case "c":
			return false, model.DenyTopstdMode
		case "s", "m":
			mts, hasMTS := meta["modi_by_topstd"]
			if !hasMTS || (strings.EqualFold(mts, "Y") && status == "MODIFY") || meta["status"] == "" {
				return true, model.DenyNone
			}
			return false, model.DenyTopstdMode
		}
	}
	// S5 登录用户门禁
	if env.LoginUser == "topstd" && src == "c" {
		return false, model.DenyLoginTopstd
	}
	return true, model.DenyNone
}

//----------------------------------------------------------------------------
// 合成主流程
//----------------------------------------------------------------------------

// event 是合成过程中扫描出的一个改写点（锚点展开后文本坐标）。
type event struct {
	pos, end int
	kind     int // 0=区段开始 1=区段结束 2=占位符行
	sec      *tglfile.Section
	name     string
	tag      []byte
	lineEnd  int // 占位符行：不含 EOL 的行尾
	eolEnd   int
}

// Synthesize 把 Package 合成为 Document（不含围栏）。
func Synthesize(pkg *pkgfile.Package, opts Options) (*model.Document, error) {
	tapEntry, tglEntry := pkg.Tap(), pkg.Tgl()
	if tapEntry == nil || tglEntry == nil {
		return nil, &SynthesisError{Msg: "包缺少 .tap 或 .tgl，无法合成", Detail: []string{pkg.Path}}
	}
	tapDoc, err := tapfile.Parse(tapEntry.Data)
	if err != nil {
		return nil, &SynthesisError{Msg: "TAP 解析失败", Detail: []string{err.Error()}}
	}
	prog := tapDoc.RootAttrOr("prog", "")
	// 状态合并：包级事实（section_flag="Y"）优先，其次 workspace 的意图
	// （pending_unlock，或调用方显式要求 Unlocked —— unlock 命令的重渲染走这条）。
	flagY := strings.EqualFold(tapDoc.RootAttrOr("section_flag", ""), "Y")
	pending := opts.PendingUnlock || opts.SectionState == model.SectionUnlocked
	state := model.EffectiveSectionState(flagY, pending)
	env := readEnv(tapDoc, pkg, state, opts.TopstdMode)
	src := tglfile.TrimEnd(tglEntry.Data)

	// ①–③ 锚点展开（FUNCTION → DIALOG → REPORT 固定顺序）
	anchors := map[string]bool{"other.function": false, "other.dialog": false, "other.report": false}
	for _, k := range []tglfile.AnchorKind{tglfile.AnchorFunction, tglfile.AnchorDialog, tglfile.AnchorReport} {
		var models []*tapfile.Element
		for _, p := range tapDoc.Points {
			if tapfile.IsDeleted(p) {
				continue
			}
			n, _ := p.Attr("name")
			if pt, ok := pointType(n); ok && pt == k {
				models = append(models, p)
			}
		}
		sort.SliceStable(models, func(i, j int) bool { return sortIndex(models[i]) < sortIndex(models[j]) })
		var list bytes.Buffer
		for i, m := range models {
			n, _ := m.Attr("name")
			if i > 0 {
				list.WriteString("\r\n")
			}
			list.WriteString(`{<point name="` + n + `"/>}`)
		}
		if newSrc, ok := tglfile.ReplaceAnchor(src, k, list.Bytes()); ok {
			anchors["other."+string(k)] = true
			src = newSrc
		}
	}

	// 收集事件：区段标记 + 占位符行
	sections, err := tglfile.FindSections(src)
	if err != nil {
		return nil, &SynthesisError{Msg: "TGL 区段标记有问题", Detail: []string{err.Error()}}
	}
	var events []event
	for _, s := range sections {
		events = append(events,
			event{pos: s.StartMarker.Start, end: s.StartMarker.End, kind: 0, sec: s},
			event{pos: s.EndMarker.Start, end: s.EndMarker.End, kind: 1, sec: s},
		)
	}
	// 逐行找占位符（设计器是逐行匹配后**整行**替换，行内其它内容一并丢弃）
	{
		i := 0
		for {
			j := bytes.IndexByte(src[i:], '\n')
			var lineEnd, eolEnd int
			if j < 0 {
				lineEnd, eolEnd = len(src), len(src)
			} else if j > 0 && src[i+j-1] == '\r' {
				lineEnd, eolEnd = i+j-1, i+j+1
			} else {
				lineEnd, eolEnd = i+j, i+j+1
			}
			if m := rePlaceholder.FindSubmatchIndex(src[i:lineEnd]); m != nil {
				events = append(events, event{
					pos: i, end: lineEnd, kind: 2,
					name:    string(src[i+m[2] : i+m[3]]),
					tag:     append([]byte(nil), src[i+m[0]:i+m[1]]...),
					lineEnd: lineEnd, eolEnd: eolEnd,
				})
			}
			if j < 0 {
				break
			}
			i = eolEnd
		}
	}
	sort.SliceStable(events, func(a, b int) bool {
		if events[a].pos != events[b].pos {
			return events[a].pos < events[b].pos
		}
		return events[a].kind < events[b].kind
	})

	doc := &model.Document{
		Env: env, Prog: prog, Ver: pkg.Ver.Raw,
		SectionState:  state,
		PendingUnlock: opts.PendingUnlock,
		UnlockedBy:    model.UnlockedByOf(flagY, opts.PendingUnlock),
		Only:          opts.Only, Anchors: anchors,
	}
	doc.PkgPath = pkg.Path
	if h, err := model.Sha256File(pkg.Path); err == nil {
		doc.PkgSha256 = h
	}

	var out bytes.Buffer
	out.Grow(len(src) + 8192)
	cursor := 0
	var stack []*model.Region

	attach := func(reg *model.Region) {
		if len(stack) > 0 {
			parent := stack[len(stack)-1]
			parent.Children = append(parent.Children, reg)
			return
		}
		doc.Regions = append(doc.Regions, reg)
	}

	for _, ev := range events {
		if ev.pos < cursor {
			continue // 已被占位符整行替换吞掉
		}
		if ev.pos > cursor {
			out.Write(src[cursor:ev.pos])
			cursor = ev.pos
		}
		switch ev.kind {
		case 0: // 区段开始
			reg := buildSectionRegion(tapDoc, prog, ev.sec, env, opts)
			reg.FullSpan.Start = out.Len()
			out.Write(src[ev.pos:ev.end])
			cursor = ev.end
			reg.ContentSpan.Start = out.Len()
			reg.FullSpan.End = out.Len()
			attach(reg)
			stack = append(stack, reg)
		case 1: // 区段结束
			if len(stack) == 0 {
				return nil, &SynthesisError{Msg: "TGL 区段结束标记多于开始标记", Detail: []string{ev.sec.ID}}
			}
			closing := stack[len(stack)-1]
			closing.ContentSpan.End = out.Len()
			out.Write(src[ev.pos:ev.end])
			cursor = ev.end
			closing.FullSpan.End = out.Len()
			stack = stack[:len(stack)-1]
		case 2: // 占位符行：整行替换为点内容
			content, meta, selfDef, parseOK := resolvePoint(tapDoc, ev.name, ev.tag, env)
			contentStart := out.Len()
			reg := buildPointRegion(ev.name, ev.tag, content, meta, selfDef, parseOK, env, opts, contentStart)
			out.Write(content)
			reg.ContentSpan.End = out.Len()
			reg.FullSpan = reg.ContentSpan
			reg.Sha256 = model.Sha256Bytes(content)
			attach(reg)
			// 行尾 EOL 原样保留（设计器只替换行内容）
			if ev.eolEnd > ev.lineEnd {
				out.Write(src[ev.lineEnd:ev.eolEnd])
			}
			cursor = ev.eolEnd
		}
	}
	if len(stack) != 0 {
		return nil, &SynthesisError{Msg: "TGL 区段未闭合（内部错误）"}
	}
	if cursor < len(src) {
		out.Write(src[cursor:])
	}

	doc.Text = out.Bytes()
	finalize(doc)
	return doc, nil
}

// resolvePoint 取点内容与元数据（含设计器的「造空点」语义）。
func resolvePoint(tapDoc *tapfile.Doc, name string, tglTag []byte, env model.EnvContext) (content []byte, meta map[string]string, selfDef, parseOK bool) {
	el := tapDoc.Point(name) // 第一个未删除元素（设计器语义，跳过 tombstone）
	meta = map[string]string{}
	if el != nil {
		for _, a := range el.Attrs {
			meta[a.Name] = a.Value
		}
		meta["tap_edit"] = meta["edit"]
	} else {
		// 造空点：ProgramInformation.Initial（ProgramInformation.cs:655-666）
		meta["new"] = "Y"
		meta["status"] = "c"
		meta["src"] = env.Env
		meta["order"] = ""
		meta["ver"] = ""
		meta["cite_std"] = "N"
		meta["ch"] = ""
		meta["ind_fun"] = env.Topind
		meta["ind_extra"] = "N"
		meta["readonly"] = ""
		meta["modi_by_topstd"] = ""
	}
	// edit= / mark= 来自 TGL 占位符行（ProcessAddPoints:1195-1206）
	if m := reEditE.FindSubmatch(tglTag); m != nil {
		meta["edit"] = string(m[1])
	} else {
		meta["edit"] = ""
	}
	if m := reMark.FindSubmatch(tglTag); m != nil {
		meta["mark"] = string(m[1])
	}
	if el != nil && el.HasCDATA() {
		content = append([]byte(nil), el.Content(tapDoc.Raw)...)
	}
	_, selfDef = pointType(name)
	parseOK = true
	if selfDef && len(content) > 0 {
		// 按点名前缀选信封种类：function.→FUNCTION、dialog.→DIALOG、report.→REPORT
		// （真实语料里有 18 个 dialog. 点的 CDATA 装的是 DIALOG … END DIALOG 信封）
		kind, ok := fgl.EnvelopeKindFor(name)
		if ok {
			if _, perr := fgl.ParseBlock(string(content), kind); perr != nil {
				parseOK = false
			}
		}
	}
	return content, meta, selfDef, parseOK
}

// buildPointRegion 构造点 Region（区间为文档绝对坐标）。
func buildPointRegion(name string, tglTag, content []byte, meta map[string]string,
	selfDef, parseOK bool, env model.EnvContext, opts Options, contentStart int) *model.Region {

	editable, deny := ResolvePoint(name, meta, env, selfDef, parseOK)
	if len(opts.Only) > 0 && !contains(opts.Only, name) {
		editable, deny = false, model.DenyNotExported
	}

	reg := &model.Region{
		Name:        name,
		Kind:        model.RegionPoint,
		Editable:    editable,
		DenyCode:    deny,
		Reason:      model.DenyReasonText(deny),
		Meta:        meta,
		Origin:      pointOrigin(tapDocOrigin(meta), tglTag),
		TglTag:      append([]byte(nil), tglTag...),
		ContentSpan: model.ByteRange{Start: contentStart, End: contentStart + len(content)},
		FullSpan:    model.ByteRange{Start: contentStart, End: contentStart + len(content)},
		Sha256:      model.Sha256Bytes(content),
	}
	base := contentStart
	if selfDef {
		if parseOK && len(content) > 0 {
			kind, _ := fgl.EnvelopeKindFor(name)
			blk, _ := fgl.ParseBlock(string(content), kind)
			if blk != nil {
				// 设计指南 §4 规则 3 的三类权限区：
				//   描述块 = 签名行之前的所有字节（可编辑，V6 校验形态，Desc 字段是其镜像）
				//   签名行 = 结构事务治理（gate1 豁免 + V1/V5 校验）
				//   END 行 = 绝对不可改（gate1 逐字节）
				if blk.Header.StartByte > 0 {
					d := model.ByteRange{Start: base, End: base + blk.Header.StartByte}
					reg.DescSpan = &d
					reg.Desc = string(content[:blk.Header.StartByte])
				} else {
					z := model.ByteRange{Start: base, End: base}
					reg.DescSpan = &z
				}
				sg := model.ByteRange{Start: base + blk.Header.StartByte, End: base + blk.Header.EndByte}
				reg.SignatureSpan = &sg
				reg.StructureSpans = []model.ByteRange{
					{Start: base + blk.Terminator.StartByte, End: base + blk.Terminator.EndByte},
				}
				reg.Scope = blk.Scope
				// Fn 带参数列表（对齐设计器 FunctionName 的语义：名字(参数)）
				reg.Fn = signatureName(string(content[blk.Header.StartByte:blk.Header.EndByte]))
				if blk.Body.EndByte > blk.Body.StartByte {
					reg.BodySpan = &model.ByteRange{Start: base + blk.Body.StartByte, End: base + blk.Body.EndByte}
				} else {
					z := model.ByteRange{Start: base + blk.Header.EndByte, End: base + blk.Header.EndByte}
					reg.BodySpan = &z
				}
			}
		}
	} else {
		// 裸名插入点：没有函数身份（[PLAIN]），整块即正文，不做结构校验。
		reg.Plain = true
		z := model.ByteRange{Start: base, End: base + len(content)}
		reg.BodySpan = &z
	}
	return reg
}

// signatureName 从签名行里取出「名字(参数)」形式（对齐设计器 FunctionName 的语义）。
//
// 例：`PUBLIC FUNCTION aapp131_calc(p_a, p_b)` → `aapp131_calc(p_a, p_b)`；
// 无参数列表时返回 `aapp131_calc`。
func signatureName(headerLine string) string {
	s := headerLine
	// 去掉前导空白
	s = strings.TrimLeft(s, " \t")
	// 剥掉 PUBLIC/PRIVATE 前缀
	if m := reSigScope.FindStringSubmatch(s); m != nil {
		s = s[len(m[0]):]
	}
	// 剥掉种类关键字
	if m := reSigKind.FindStringSubmatch(s); m != nil {
		s = s[len(m[0]):]
	}
	s = strings.TrimLeft(s, " \t")
	// 去掉行尾注释（# 之后）
	if i := strings.IndexByte(s, '#'); i >= 0 {
		s = s[:i]
	}
	return strings.TrimRight(s, " \t")
}

var (
	reSigScope = regexp.MustCompile(`(?i)^(PUBLIC|PRIVATE)[ \t]+`)
	reSigKind  = regexp.MustCompile(`(?i)^(FUNCTION|DIALOG|REPORT)[ \t]*`)
)

// pointOrigin 记录点来源，便于审计与折叠判断。
func pointOrigin(tapOrigin string, tglTag []byte) string {
	if len(tglTag) == 0 {
		return tapOrigin
	}
	return "placeholder"
}

func tapDocOrigin(meta map[string]string) string { return "anchor-injected" }

// buildSectionRegion 构造区段 Region（含 TAP 缺失时的模板 clone 语义）。
func buildSectionRegion(tapDoc *tapfile.Doc, prog string, s *tglfile.Section,
	env model.EnvContext, opts Options) *model.Region {

	meta := map[string]string{}
	el := tapDoc.Section(s.ID)
	missing := el == nil
	if el != nil {
		for _, a := range el.Attrs {
			meta[a.Name] = a.Value
		}
	} else {
		// 设计器：先找 src="c" 的区段 clone，再找 src="m"，都没有则「找不到区段」（在 ProcessSections 抛）。
		var tpl *tapfile.Element
		for _, pass := range []string{"c", "m"} {
			for _, cand := range tapDoc.Sections {
				if v, _ := cand.Attr("src"); v == pass {
					tpl = cand
					break
				}
			}
			if tpl != nil {
				break
			}
		}
		if tpl != nil {
			for _, a := range tpl.Attrs {
				meta[a.Name] = a.Value
			}
			meta["status"] = "u" // SectionModel.Clone 置 status="u" 且正文为空
		}
	}

	isReadonly := s.ReadonlyMarker
	editable, deny := ResolveSection(s.ID, isReadonly, meta, env, prog)
	if missing {
		if _, isAnchor := tglfile.AnchorSectionID(prog, s.ID); !isAnchor {
			if meta["status"] == "" {
				editable, deny = false, model.DenySectionTemplateMissing
			}
		}
	}
	if len(opts.Only) > 0 {
		editable, deny = false, model.DenyOnlyPoints
	}

	reg := &model.Region{
		Name:     s.ID,
		Kind:     model.RegionSection,
		Editable: editable,
		DenyCode: deny,
		Reason:   model.DenyReasonText(deny),
		Meta:     meta,
		Origin:   "tap-section",
	}
	if at, ok := tglfile.AnchorSectionID(prog, s.ID); ok {
		reg.AnchorType = string(at)
		reg.Append = true // 锚点区段允许追加新 point 块（设计指南 §4 规则 4）
		reg.Origin = "anchor"
	}
	if s.ReadonlyMarker {
		reg.Meta["tgl_readonly"] = "Y"
	}
	return reg
}

func contains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}

//----------------------------------------------------------------------------
// 收尾：未归属字节段
//----------------------------------------------------------------------------

// finalize 计算 prefix / gap / suffix 未归属字节段。
//
// 这些字节不归属任何 Region，但**同样参与围栏外字节恒等校验**（gate1）；
// 登记它们的目的是让 status/verify 能显式报告，而不是让它们「隐形」。
func finalize(doc *model.Document) {
	var covered []model.ByteRange
	for _, r := range doc.Regions {
		covered = append(covered, r.FullSpan)
	}
	sort.Slice(covered, func(i, j int) bool { return covered[i].Start < covered[j].Start })

	cursor, gap := 0, 0
	addSpan := func(name string, s, e int) {
		if e <= s {
			return
		}
		doc.Spans = append(doc.Spans, model.Span{
			Name:   name,
			Span:   model.ByteRange{Start: s, End: e},
			Sha256: model.Sha256Bytes(doc.Text[s:e]),
		})
	}
	for _, c := range covered {
		if c.Start > cursor {
			name := "gap." + strconv.Itoa(gap)
			if cursor == 0 {
				name = "prefix"
			}
			addSpan(name, cursor, c.Start)
			gap++
		}
		if c.End > cursor {
			cursor = c.End
		}
	}
	if cursor < len(doc.Text) {
		addSpan("suffix", cursor, len(doc.Text))
	}
}
