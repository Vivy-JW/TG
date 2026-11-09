package target

import "strings"

// 层级校验（计划 WP0 的层级定义 + Q7 + Q8 + 契约第 7 节的表）。
//
// 两条纪律写死在代码里：
//
//  1. **Q7：L4 不作寻址层**。矩阵里 `tt debug` 那组的"基准是 L4"（"会话 + 一轮运行"），
//     但"哪条会话"这件事到 L3 就说清了 —— 所以**登记表的组默认直接写 L3（寻址层）**：
//     L4 是**状态语义**（这一组大体在什么状态下干活），寻址层是 **L3**（Q7），两者不是一回事。
//
//  2. **Q8：只登记"组默认 + 具名例外"**，清单**从命令树派生**。
//     于是登记表不是 166/186 行的全清单：与组默认相同的命令一条都不登。

// Layer 是目标层级。
type Layer int

const (
	// Layer0：无远端目标（纯本地：包文件、工作区、本机进程、本地登记）。
	Layer0 Layer = iota
	// Layer1：只要环境（登录 SSH，或按环境读服务器文件/源码）。
	Layer1
	// Layer2：环境 + 企业编号（按企业编号取库账号）。
	Layer2
	// Layer3：一条 SSH 会话（这是本契约的寻址层）。
	Layer3
	// Layer4：会话 + 一轮运行。**不是寻址层**（Q7）：认路只做 L0–L3。
	//
	// **它不作为任何组的默认值**（Q7）：登记表里 `debug` 那组的组默认写 L3（寻址层），
	// 矩阵里 `debug` 的"基准是 L4"只是**状态语义**。本枚举值仍保留 ——
	// 它是矩阵的语义值（`String` / `LayerNeed` 与文档都还要读它）。
	Layer4
	// LayerUndetermined 是矩阵里标"待定"的那几条（契约第 9 节的 W1–W3）。
	//
	// **不猜**：它既不夹到组默认，也不当成 L0 —— 认路要求交调用方（WP4 接线时回填）。
	// 猜错的后果是"看起来判过了"，那正是本次重构要根除的东西。
	LayerUndetermined Layer = -1
)

// String 是层级的显示名（报错文案与文档都读它）。
func (l Layer) String() string {
	switch l {
	case Layer0:
		return "L0"
	case Layer1:
		return "L1"
	case Layer2:
		return "L2"
	case Layer3:
		return "L3"
	case Layer4:
		return "L4"
	case LayerUndetermined:
		return "待定"
	}
	return "?"
}

// LayerNeed 是这个层级"要什么"的一句话（层级校验的报错文案用它）。
func LayerNeed(l Layer) string {
	switch l {
	case Layer0:
		return "无远端目标"
	case Layer1:
		return "要一个环境"
	case Layer2:
		return "要环境 + 企业编号"
	case Layer3:
		return "要一条 SSH 会话"
	case Layer4:
		return "要一条会话 + 一轮运行"
	}
	return "层级待定"
}

// LayerRegistry 是"组默认 + 具名例外"的层级登记（Q8）。
//
// Groups 的键是**命令组路径**（如 `debug` / `dev tzs`；顶层用空串），
// Exceptions 的键是**完整命令路径**（如 `debug sql` / `session new`）。
//
// 查一条命令的层级：先看例外表（精确匹配），再取**最长匹配**的组默认 ——
// 所以矩阵里那 166 条叶子命令不需要在这里各占一行（Q8：清单从命令树派生）。
type LayerRegistry struct {
	Groups     map[string]Layer
	Exceptions map[string]Layer
}

// DefaultLayerRegistry 是 `tt-target-levels.md` 的登记：组默认 7 条 + 具名例外 17 条。
//
// 例外按组：`debug` 7（矩阵「例外条数」19 − 12 条 L3 冗余，见下）+ `dict` 5 + `dev tzs` 3 +
// `session` 2（`ls` 降到 L0、`new` 抬到 L1）。
// `debug` 的组默认是 **L3**（Q7 的寻址层），所以矩阵里那 12 条 `: L3` 在认路时与组默认同值、
// 一条都不改变行为 —— 按 Q8「与组默认相同的命令一条都不登」全部摘掉。
//
// **改这张表要一起改的**：`tt-target-levels.md`（人读那份）与 `layer_test.go` 的条数断言。
//
// 交出来的两份 map 都是**副本**：调用方想试一种别的登记形状（测试、`--dry-run` 式的推演）
// 不会改到包级那份 `layerExceptions`。
func DefaultLayerRegistry() LayerRegistry {
	return LayerRegistry{
		Groups: map[string]Layer{
			// 顶层：cache / config / env / install / serve / update / version + 三个别名。
			"": Layer0,
			// Q7：寻址层就是 L3 —— 矩阵里 `debug` 那组"基准是 L4"是**状态语义**
			//（"在一条活着的 SSH 会话里看/动当前那一轮运行"里的"当前那一轮运行"），
			// 而"哪条会话"到 L3 就说清了，所以**组默认写 L3（= 寻址层）**。
			// 于是组内 12 条 `: L3` 的"例外"与默认同值、一条都不改变行为，按 Q8 一律不登。
			"debug":   Layer3,
			"dict":    Layer0, // 缺省数据源是本地 SQLite 镜像
			"file":    Layer1, // 服务器文件只读：要一台点名环境
			"dev tzc": Layer0,
			"dev tzs": Layer0, // 本机常驻引擎 + 本地包/工作区
			"session": Layer3, // 会话级动作（ent/quit/rm/exec/show）都要一条活着的会话
		},
		Exceptions: cloneLayerMap(layerExceptions),
	}
}

// cloneLayerMap 拷一份命令路径 → 层级的表（`DefaultLayerRegistry` 的两份 map 都过它）。
func cloneLayerMap(m map[string]Layer) map[string]Layer {
	out := make(map[string]Layer, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// layerExceptions 是具名例外（逐条与 `tt-target-levels.md` 的「分组明细」对齐）。
//
// 只登**与组默认不同**的路径：与组默认相同的命令登进来就是"全清单"的开始（Q8 要拦的正是这个）。
// 于是矩阵里 12 条 `debug xxx: L3`（组默认已按 Q7 定成 L3）在这里**不出现** ——
// 它们改变不了任何行为，登着就是登记表违背自己的规矩。
// 例外总条数不是矩阵「例外条数」一列的和：矩阵那份是**逐条判据**（含状态语义），
// 这里是**认路登记表**（只留改变寻址要求的）。
var layerExceptions = map[string]Layer{
	// ---- tt debug（7 条例外：5 L1 + 2 L2） ----
	//
	// 组默认是 L3（Q7 的寻址层），所以组内 12 条 `: L3`（start / wsdebug / stop / exec /
	// interrupt / quit / mode / status / env / topent / serve / logs）**一条都不登** ——
	// 它们在认路时与组默认同值、一条都不改变行为（Q8：清单从命令树派生，
	// 与组默认相同的命令登进来就是"全清单"的开始）。
	// 留下的是真正**改变寻址要求**的：5 条 L1（只要环境）+ 2 条 L2（环境 + 企业编号）。
	"debug wslogs":  Layer1, // C1.3：走环境 + 企业、不认 --session，待定收口为 L1
	"debug source":  Layer1, // 服务器源码白名单只读，走独立短连接
	"debug resolve": Layer1, // 查 gzzz_t 解作业编号→实体程序
	"debug ents":    Layer1, // 环境级的企业→账号清单
	"debug probe":   Layer1, // M0 尖刺：自己就起会话，不认外部会话
	"debug sql":     Layer2, // 只读 SQL，按环境 + 企业编号取库账号
	"debug db":      Layer2, // 探 ENT→账号映射并用该账号连一次库

	// ---- tt dict（4 条例外） ----
	"dict db discover": Layer1, // 登录 SSH 探库要素
	"dict mirror pull": Layer1, // 隐藏转发：SFTP 增量下载该环境镜像
	"dict mirror path": Layer1, // 隐藏转发：该环境镜像目录（与环境绑定）
	"dict db sync":     Layer2, // 账号由企业编号决定

	// ---- tt file（1 条例外） ----
	"file mirror dir": Layer0, // 设的是全局镜像根目录，与环境无关

	// ---- tt dev tzs（3 条待定，W1） ----
	"dev tzs list_tables":  LayerUndetermined,
	"dev tzs list_columns": LayerUndetermined,
	"dev tzs list_records": LayerUndetermined,

	"session ls":  Layer0, // 列会话：只要服务在，不要求已有会话
	"session new": Layer1, // 建会话：点名的环境就是要连的那台；给了 --prog 还会起作业
}

// DeclaredLayer 返回登记表里那一层（可能含 L4 与待定）—— 文档、测试与"这条命令要什么"都用它。
func (r LayerRegistry) DeclaredLayer(cmdPath string) Layer {
	p := normalizeCmdPath(cmdPath)
	if l, ok := r.Exceptions[p]; ok {
		return l
	}
	if l, ok := r.Groups[r.GroupOf(p)]; ok {
		return l
	}
	// 组默认表是齐全的；真查不到就按"无远端目标"（顶层那一组），绝不给一个吓人的默认值。
	return Layer0
}

// AddressingLayer 返回**认路**意义上要求的层级：Q7 —— L4 夹到 L3。
//
// 「待定」原样返回：调用方据此决定"不拦"还是"报出来"（本包不替它猜）。
func (r LayerRegistry) AddressingLayer(cmdPath string) Layer {
	if l := r.DeclaredLayer(cmdPath); l == Layer4 {
		return Layer3
	} else {
		return l
	}
}

// GroupOf 返回一条命令路径所属的**命令组**（最长匹配，空串 = 顶层）。
//
// 前缀必须落在段边界上：`debugx y` 不属于 `debug` 组。
func (r LayerRegistry) GroupOf(cmdPath string) string {
	p := normalizeCmdPath(cmdPath)
	best := ""
	for g := range r.Groups {
		if g == "" {
			continue
		}
		if p == g || strings.HasPrefix(p, g+" ") {
			if len(g) > len(best) {
				best = g
			}
		}
	}
	return best
}

// normalizeCmdPath 把命令路径归一：吃 `tt ` 前缀与首尾空白。
//
// 两种来源都要收：cobra 的 `CommandPath()` 带 `tt ` 前缀，手写的路径不带。
func normalizeCmdPath(p string) string {
	p = strings.TrimSpace(p)
	p = strings.TrimSpace(strings.TrimPrefix(p, "tt "))
	return p
}

// Provided 是一次调用**实际提供了多少信息**（→ 提供到第几层）。
type Provided struct {
	// Env 给了环境（`--env` / `--ssh`，或选择器里的环境段）。
	Env string
	// Ent 给了企业编号（`--ent`，或 `--session 环境/企业编号`）。
	Ent string
	// HasSession 已经落到了**唯一**一条会话。
	HasSession bool
}

// Layer 返回这次调用提供到第几层。
//
// 没有环境时一律算 L0：B23 定的是"环境一律点名"，`--ent` 单独出现解决不了任何事
// （没有环境就不知道该连哪台机器）。
func (p Provided) Layer() Layer {
	if p.HasSession {
		return Layer3
	}
	if p.Env == "" {
		return Layer0
	}
	if p.Ent != "" {
		return Layer2
	}
	return Layer1
}

// CheckLayer 校验"命令声明的层级 vs 实际提供的信息"。
//
// 返回 nil = 够用（待定与 L0 一律放行：前者不猜，后者什么都不需要）。
// 不够用时返回一条 `USAGE` 退 2 的失败 —— **不新增码**（4.1.1 是唯一清单）。
func CheckLayer(r LayerRegistry, cmdPath string, p Provided) *Failure {
	need := r.AddressingLayer(cmdPath)
	if need == LayerUndetermined || need <= Layer0 {
		return nil
	}
	if p.Layer() >= need {
		return nil
	}
	return failureLayerMismatch(normalizeCmdPath(cmdPath), need, p.Layer())
}
