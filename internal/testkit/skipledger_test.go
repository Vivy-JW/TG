package testkit

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// 本文件是**跳过台账**：让"多一处跳过"变成一次必须改文件、因此会被 review 的动作。
//
// 为什么值得：跳过的测试与通过的测试在报告里长得一样无害。而这个仓库里
// `go test ./...` 默认档的覆盖面，正是靠"到底跳了多少、跳的是哪一类"来定义的 ——
// 一处新的 t.Skip 悄悄加进来，默认档就少验一块，而没有任何东西会响。
//
// 台账登记每一处 t.Skip / t.Skipf 的**位置**（文件 + 文案）与**它属于哪一层**。
// 多一处、少一处、文案改了 —— 都红。
//
// 两条设计取舍，都是刻意的：
//
//  1. **只记文件 + 文案，不记行号**。记行号的话，随便在文件前面加一行注释就会让
//     整份台账失效；而行号只在报错时现算。反过来，改**文案**会让它红 —— 那是对的：
//     文案是"这条跳过在说什么"，改它本就该被看见。
//  2. **扫描范围是 `internal/` 下所有 `.go`，不只是 `_test.go`**。这不是过度设计：
//     `internal/testkit/corpus.go` 是**普通 .go 文件**，里面有 2 处 t.Skipf
//     （语料发现的两条收场），而它是全仓共用的 —— 只扫 `_test.go` 会把这两处漏掉。
//     只看 `_test.go` 的数字也会因此偏小（36 而不是 38）。

// skipLayer 是这条跳过属于哪一层。取值见【二】的分层定义。
type skipLayer string

const (
	// layL3 真语料回归：语料 / 引擎产物 / 工作区不在这台机器上。
	layL3 skipLayer = "L3"
	// layL4 真机 E2E：真引擎 exe / 真 SSH。
	layL4 skipLayer = "L4"
	// layL3Missing 语料在，但里面没有**这一件**（没有 .tzs / 没有那个叶子 / 没有包……）。
	//
	// 与 layL3 分开是因为它**没有开关可点名** —— 补救是"换一份带这个的语料"，
	// 不是"设一个环境变量"。混在一起的话，"why 必须点名开关"那条纪律会把
	// 一堆本来就说得清清楚楚的条目判成不合格。
	layL3Missing skipLayer = "L3-缺件"
	// layEnv 本机环境：没有 git / 没有用户配置目录 / 没有旧配置。
	// 这些**不影响"全绿"这个结论** —— 它们说的是"这台机器上没那个东西"。
	layEnv skipLayer = "L0-环境"
	// layPremise 测试自己构造的前提没满足。归这一类的要特别当心：
	// 前提长期不成立 = 这条测试**静默变成空转**，与删掉它没有区别。
	layPremise skipLayer = "前提"
	// layPending 契约里**已冻结、但还没实现**的东西：测试先写好，等对应工作包落地。
	//
	// 与 layPremise 分开是因为这两件事的风险**不一样**，混在一起会让"前提长期不成立"
	// 那条警告失效：
	//   - layPremise 是"测试自己造的前提"，没有外部交付物，长期不成立就是静默空转；
	//   - layPending 有**明确的交付方与交付物**（WP3 的 ref/epoch/解析器、WP4 的
	//     `--session`/`--name`/新码），落地那天跳过会**自动变成断言**（测试里写了
	//     "落地了就把这条改成断言"的守卫），所以它不会烂在那里。
	//   它的文案必须写清"等哪个工作包、等它的什么东西"——写"待实现"这种空话不算。
	//
	// 这一层与 L3/L4 的区别也要写清：L3/L4 是"真语料/真机器不在这台机器上"，
	// 而 layPending 是"代码还没写"——两者都不影响「全绿」这个结论，
	// 但补救方式完全不同（前者换机器/换语料，后者写代码）。
	layPending skipLayer = "未落地"
)

// switchNames 是"跳过时该设哪个变量"的候选名。
//
// **只列会出现在跳过文案里的那些**，不是全仓开关总表：`TTZS_INSTALL`（设计器目录）
// 与 `TTZS_VALIDATE`（validate 的范围）都是真的开关，但没有一处跳过以它们为门槛，
// 列进来只会让"表里没有死名字"那条测试误报。
//
// 另注意：扫描器只抓 `t.Skip/Skipf` 的**第一个**字符串字面量，而有的跳过文案是
// 拼接出来的（如 e2e_test.go 那条，`TTZS_INSTALL` 在后半段未被捕获）。
// 所以这张表要按**捕获得到**的名字列。
var switchNames = []string{
	"TDEV_DEEP", "TTZS_DEEP", "TTZS_E2E", "TTZS_FNS",
	"TTZS_CORPUS", "TDEV_CORPUS",
	"TTZS_EXE", "TTZS_WS", "TTZS_PKG",
	// 表单真样本那一路（见下面 `TT_FORM_*` 那一组台账条目）：
	// 真 .42f 与真归档语料表不随仓库走，缺了就用这三个开关指过去。
	"TT_FORM_SAMPLE_DIR", "TT_FORM_CORPUS_MAP",
	// 现场真配对：要 .tzs 目录与 .42f **同一个现场**的那一份，两个开关得同时指对。
	"TT_TZS_REAL_DIR", "TT_FORM_42F",
}

// skipSite 是台账里的一处跳过。
type skipSite struct {
	file  string // 仓库相对路径（斜杠分隔）
	msg   string // t.Skip/Skipf 的第一个字符串字面量（**照抄**，标点都不能改）
	layer skipLayer
	why   string // 为什么它不算假绿 / 属于哪一层
}

// skipLedger 是**全部**跳过点的台账，逐条按 文件+文案 与代码比对。
//
// 条目是脚本从源码里抽出来的（避免手抄错），层与理由是人逐条标的。
var skipLedger = []skipSite{
	{file: "internal/cli/dev/tzc/corpus_test.go", msg: "深度语料回归未启用：设 TDEV_DEEP=1 并给足 -timeout 30m（见 README「测试与验收」）", layer: layL3, why: "需真实语料：开关 TDEV_DEEP=1"},
	{file: "internal/cli/dev/tzs/tzs_verb_e2e_test.go", msg: "需要真引擎：设 TTZS_E2E=1（可选 TTZS_EXE），见本文件顶部注释", layer: layL4, why: "需真引擎：开关 TTZS_E2E=1（可选 TTZS_EXE）"},
	{file: "internal/cli/dev/tzs/tzs_verb_e2e_test.go", msg: "找不到引擎 exe（TTZS_EXE=%s）：%v", layer: layL4, why: "需真引擎：开关 TTZS_E2E=1 + TTZS_EXE"},
	{file: "internal/cli/dev/tzs/tzs_verb_test.go", msg: "没有引擎 exe（%v）：这条测的是期限机制本身", layer: layL4, why: "需真引擎：开关 TTZS_E2E=1（这条测的是期限机制本身）"},
	{file: "internal/dev/pkgfile/zipshape_test.go", msg: "语料里没找到设计器形态的 .tap 包", layer: layL3Missing, why: "需真实语料：这一类形态只在真语料里出现"},
	{file: "internal/dev/pkgfile/zipshape_test.go", msg: "语料里没有 Zip64 形态的 .tap 包", layer: layL3Missing, why: "需真实语料：这一类形态只在真语料里出现"},
	{file: "internal/dev/store/store_test.go", msg: "环境里没有 git，跳过 git 相关断言", layer: layEnv, why: "本机没装 git（装了才跑 git 相关断言）"},
	// 「发现环境」对真实设计器文件的冒烟检查:它**必须**在没装设计器的机器上跳过,
	// 而那个跳过正是它的价值所在 —— 有设计器的机器(开发机)会真的拿 .NET 生成的密文
	// 解一遍,没有的机器不假红。所以这两条不影响「全绿」这个结论。
	{file: "internal/designer/sites_local_test.go", msg: "没有 APPDATA,跳过本机真实文件检查", layer: layEnv, why: "非 Windows / 环境异常时没有 %APPDATA%，定位不到设计器目录；合成密文的用例仍然全跑"},
	{file: "internal/designer/sites_local_test.go", msg: "这台机器没有设计器站点清单,跳过", layer: layEnv, why: "未安装 T100 设计器或从未保存过站点。**不影响「全绿」这个结论**：参数是否正确由合成密文的往返用例保证，这条只是拿真实 .NET 密文再验一次（装了设计器的机器上必跑）"},
	{file: "internal/dev/tapfile/tapfile_test.go", msg: "%s 下没有 .tap", layer: layL3Missing, why: "语料在，但里面没有 .tap"},
	{file: "internal/dev/tzs/chain_test.go", msg: "语料里没有 %s —— 这条链要一个真实的数据绑定叶子；换个语料就得同时改 chainQuery", layer: layL3Missing, why: "需真实语料里那个数据绑定叶子；换语料要同时改 chainQuery"},
	{file: "internal/dev/tzs/chain_test.go", msg: "%s 的祖先目录里没有 mta/，且没给 TTZS_WS —— 工作区绝不替你选", layer: layL3, why: "需真工作区：开关 TTZS_WS（工作区绝不替你选）"},
	{file: "internal/dev/tzs/chain_test.go", msg: "%s 里没有 %s（find_component 回 matchCount=%d）—— 这条链要一个已知存在的代号", layer: layL3Missing, why: "需真实语料里那个已知存在的代号"},
	{file: "internal/dev/tzs/chain_test.go", msg: "%s 的 %s 没有 posX 这个布局属性（可写属性：%s）", layer: layL3Missing, why: "需真实语料里带 posX 布局属性的叶子"},
	{file: "internal/dev/tzs/chain_test.go", msg: "%s 的 posX=%q 不是整数，无法「+1」出一个必然不同的值", layer: layL3Missing, why: "需真实语料里 posX 是整数的叶子（要 +1 出一个必然不同的值）"},
	{file: "internal/dev/tzs/corpus_test.go", msg: "语料深度回归未启用：设 TTZS_DEEP=1 并给足 -timeout 30m（见 corpus_test.go 顶部注释）", layer: layL3, why: "需真实语料：开关 TTZS_DEEP=1"},
	{file: "internal/dev/tzs/corpus_test.go", msg: "找不到引擎 exe（TTZS_EXE=%s）：%v；先在 engine/ 里跑 build.sh", layer: layL4, why: "需真引擎：开关 TTZS_E2E=1 + TTZS_EXE（先跑 engine/build.sh）"},
	{file: "internal/dev/tzs/corpus_test.go", msg: "%s 旁边没有 RoundTrip.exe：%v", layer: layL4, why: "需真引擎：TTZS_EXE 旁边的 RoundTrip.exe（engine/build.sh 产出）"},
	{file: "internal/dev/tzs/corpus_test.go", msg: "没有真实语料（%s / %s 都没指到一个目录）：设 TTZS_CORPUS，或准备 %s", layer: layL3, why: "需真实语料：开关 TTZS_CORPUS"},
	{file: "internal/dev/tzs/corpus_test.go", msg: "%s 下没有 .tzs（排除掉 %s 之后）", layer: layL3Missing, why: "语料在，但里面没有 .tzs"},
	{file: "internal/dev/tzs/corpus_test.go", msg: "没有真实语料：设 TTZS_CORPUS（或 TDEV_CORPUS），或准备 %s", layer: layL3, why: "需真实语料：开关 TTZS_CORPUS（或 TDEV_CORPUS）"},
	{file: "internal/dev/tzs/corpus_test.go", msg: "%s 下没有 .tzs", layer: layL3Missing, why: "语料在，但里面没有 .tzs"},
	{file: "internal/dev/tzs/fns_test.go", msg: "函数面关卡未启用：设 %s=1（它会对每个语料包驱动一遍全部写函数，见 fns_test.go 顶部）", layer: layL4, why: "需真引擎 + 真语料：开关 TTZS_FNS=1（它自己对每个包起一个 --stdio 进程驱动全部写函数）"},
	{file: "internal/dev/tzs/fns_test.go", msg: "找不到引擎 exe（TTZS_EXE=%s）：%v；先在 engine/ 里跑 build.sh", layer: layL4, why: "需真引擎：开关 TTZS_EXE 指到 tzs-server.exe（或先跑 engine/build.sh）"},
	{file: "internal/dev/tzs/corpus_test.go", msg: "pin 里 %d 条路径在本机一条都不存在（语料根=%s）：pin 描述的是另一份语料，", layer: layL3, why: "pin 描述的是另一份语料：用 TTZS_CORPUS 指到正确那份"},
	{file: "internal/dev/tzs/corpus_test.go", msg: "语料里没有包", layer: layL3Missing, why: "语料在，但里面没有包"},
	{file: "internal/dev/tzs/corpus_test.go", msg: "没有一个工作区能验（都缺 mta/mod-fd.spec，或包打不开）", layer: layL3, why: "需真工作区：开关 TTZS_WS"},
	{file: "internal/dev/tzs/corpus_test.go", msg: "没验到（语料里没有推得出工作区的包）", layer: layL3Missing, why: "需真实语料里能推出工作区的包"},
	{file: "internal/dev/tzs/dryrun_test.go", msg: "没验到（语料里挑不出可写的目标）", layer: layL3Missing, why: "需真实语料里可写的目标"},
	{file: "internal/dev/tzs/e2e_test.go", msg: "需要真引擎：设 TTZS_E2E=1 + 引擎 exe + 工作区（TTZS_EXE / TTZS_WS，", layer: layL4, why: "需真引擎：开关 TTZS_E2E=1 + TTZS_EXE + TTZS_WS"},
	{file: "internal/dev/tzs/e2e_test.go", msg: "找不到引擎 exe（%s）：%v", layer: layL4, why: "需真引擎：开关 TTZS_E2E=1 + TTZS_EXE"},
	{file: "internal/dev/tzs/e2e_test.go", msg: "没给工作区（TTZS_WS 或 config.local.json 的 workspace）。", layer: layL4, why: "需真工作区：开关 TTZS_WS（工作区绝不替你选）"},
	{file: "internal/dev/tzs/e2e_test.go", msg: "没给 TTZS_PKG（一个本工作区内的真 .tzs 路径）：逻辑键寻址要用真包验", layer: layL4, why: "需真包：开关 TTZS_PKG（逻辑键寻址要用真包验）"},
	{file: "internal/dev/tzs/e2e_test.go", msg: "没给 TTZS_PKG（本工作区内的真 .tzs 路径）：任务级动词要用真包验", layer: layL4, why: "需真包：开关 TTZS_PKG（任务级动词要用真包验）"},
	{file: "internal/dev/tzs/oplog_test.go", msg: "工作区 %s 里没有包，这条用例没法跑", layer: layL3, why: "需真工作区且里面有包：开关 TTZS_WS"},
	{file: "internal/dev/verify/verify_test.go", msg: "fence.Parse 自己就拦住了（%v）—— 这条测的是 gate1，不是解析层", layer: layPremise, why: "解析层先拦住了（那更好）—— 这条测试的对象就不存在了，如实说明而不是假装通过"},
	{file: "internal/testkit/corpus.go", msg: "没有真实语料：设 TDEV_CORPUS 或 TTZS_CORPUS 指到一份，或准备 %s", layer: layL3, why: "需真实语料：开关 TDEV_CORPUS / TTZS_CORPUS"},
	{file: "internal/testkit/corpus.go", msg: "%s 下没有 %s", layer: layL3Missing, why: "语料在，但里面没有该扩展名的文件"},
	// ---- WP2（会话寻址重构）的契约测试：等 WP3/WP4/WP4c 把冻结契约落地 ----
	//
	// 这一组原先是 layPending（见那一层的定义）：契约**已冻结**（WP1 交付），
	// 测试先写好把形状钉住。**现在一条都不剩了** —— 它们等的交付物全部到位，
	// 各处的守卫响了，按各自的交代**改成了真断言**，于是台账按性质逐条删掉
	// （不再跳过的东西不该留在跳过台账里）：
	//
	//   - `TestContractSessionEnvelopeKeysMatchSessionBrief`（缺键 → 判红）
	//   - `TestContractNewCodesAreRegisteredInOutput`（缺码 → 判红）
	//   - `TestContractRefKeyIsNotYetInSessionBrief` → `TestContractIdentityKeysAreInSessionBrief`
	//   - `TestContractEpochStalenessIsNotYetEnforced` → `TestContractEpochStalenessIsEnforced`
	//   - `TestContractNameIdempotencyIsNotYetEnforced` → `TestContractNameIdempotencyIsEnforced`
	//   - `TestContractHTTPFailureCodeIsNotYetLanded` → `TestContractHTTPFailureCodeIsLanded`
	//   - `TestContractInvalidFormsAreNotYetEnforced` → `TestContractInvalidFormsServerSideEnvelopes`
	//     ＋ `TestContractNameTakenEnvelopeIsWritten`（服务端判得出来的那几条逐条发真请求）
	//   - `TestContractIdentityLineIsNotYetLanded` → `TestContractIdentityLineIsLanded`
	//     （**最后一条**：C5.9 的身份行本批接进了 CLI，判据在
	//     `internal/debug/contract_debug_test.go`（服务端数据 + C5.9 渲染）
	//     与 `internal/cli/debug/envtarget_test.go` 的 `TestDBGIdentity*`（那一行真的被打出来））
	//
	// 那一批的其余条目（`internal/erpdb` 等）与本次重构无关，照旧登记在上面。
	//
	// `layPending` 这个层本身**保留**：它是"契约已冻结、代码还没写"这一类跳过的唯一归宿，
	// 下次再先写测试后写实现时还要用它（本批归零不等于这一层没用了）。
	//
	// ---- 已经**不再登记**的那一条（WP4b 交付时删掉的）----
	//
	// `internal/testkit/first_session_source_test.go` 的 T2.3 静态检查曾经在这里有一条
	// layPending（"全仓「取第一个会话」基线 5 处，等 WP4b 删"）。WP4b 把服务端最后那 5 处
	// `Manager.Current()` 兜底删掉之后，那条检查**命中数为 0、跳过消失、改成真断言** ——
	// 于是这条台账按它的性质一起删掉：**不再跳过的东西不该留在跳过台账里**。
	//
	// 这一条删得对不对，由台账自己反向守着：`TestSkipLedger` 的"台账里登记了、代码里却没有"
	// 那一半会在"删了跳过却没同步台账"时红；反过来"同步了台账却还留着跳过"也会红。
	// 所以这里不需要另加判据 —— 台账与代码必须严格对上，这正是它的存在意义。
	//
	// ---- WP7（只读红线）那一条没有进来 ----
	//
	// `internal/erpdb` 的文本层从"前缀匹配"换成 `internal/safesql.Check` 时，
	// 旧测试 `TestCheckReadOnlySQLIsAPrefixCheckNotAParser` 的那半句断言（"前导注释会绕过"）
	// 当场反向。处理方式是**把它整条删掉**、判据搬进同一文件的三条新测试
	// （`TestCheckReadOnlySQLRejectsWritesAndMultipleStatements` /
	// `…AllowsReadOnlySelects` / `…BlocksTheReadOnlyTransactionBypass`），
	// **没有留 t.Skip**：旧断言作废不值得在台账里长期占一行（留注释说明即可）。
	// 于是本台账里 `layPending` 这一层现在是 **0 条** —— 它保留着以备下次"先写测试后写实现"。

	// ---- 表单真样本那一路（`fixtures/forms/`）：TT_FORM_* 开关 + 现场真配对 ----
	//
	// 这一批 9 条的病因是同一个：**真 `.42f` 与真归档语料表不随仓库走**。
	// 契约 C7.2/C7.3 的口径是"仓库内只放合成夹具，缺真样本时写明原因、
	// 不拿合成件顶替" —— 所以跳过在这里是**对的**，但它必须登记：
	// 否则默认档少验一整块表单解析，而报告里什么都看不出来。
	//
	// 分层按"怎么补救"分：
	//   · 几乎全是 L3，开关名统一落在 TT_FORM_SAMPLE_DIR / TT_FORM_CORPUS_MAP 上；
	//   · 三条是「L3-缺件」——语料（或仓库自带的最小语料）在，但里面没有**这一件**，
	//     补救是换份语料/重新生成，没有开关可点名；
	//   · `form42f_realpair_test.go` 那条要两个开关同时指对（TT_TZS_REAL_DIR +
	//     TT_FORM_42F）：要的不是一份 .42f，而是**同一个现场**打出来的 .tzs 与 .42f。
	//
	// 注意这批文件的归属：`dev/tzs`、`form42f` 都另有切片在改。
	// 台账只登记**当前源文件里的字面量**；那些文件里的跳过文案一改（含标点），
	// 这里就会红 —— 那是对的：文案是"这条跳过在说什么"。
	{file: "internal/dev/tzs/form42f_check_test.go", msg: "没有真样本（%s 打不开：%v）—— 设 TT_FORM_SAMPLE_DIR 指向 fixtures/forms/", layer: layL3, why: "需真样本：开关 TT_FORM_SAMPLE_DIR"},
	{file: "internal/dev/tzs/form42f_check_test.go", msg: "没有仓库自带的最小语料（%s：%v）—— 它是仓库本体，缺了用 `go run ./tools/tzsmini` 重新生成", layer: layL3Missing, why: "仓库自带的最小语料缺了：它是仓库本体，补救是重新生成（go run ./tools/tzsmini），没有开关可点名"},
	{file: "internal/dev/tzs/form42f_realpair_test.go", msg: "没有现场真配对样本：%s —— 设 TT_TZS_REAL_DIR 与 TT_FORM_42F 指向真 .tzs 目录与真 .42f（契约 C7.3：缺样本写明原因，不拿合成件顶替）", layer: layL3, why: "需真现场配对样本：开关 TT_TZS_REAL_DIR + TT_FORM_42F"},
	// D8：本机 bash 会展开**单引号内**的命令替换（WSL/中间层特性），`$(…)` 与反引号那两类
	// 用例在这台机器上测不出真结果。不是 `shQuote` 的问题（它是标准写法），换一台 POSIX 环境
	// 就有意义 —— 所以归 L0-环境，不是"缺开关"。
	{file: "internal/host/dbprobe_test.go", msg: "本机 bash 会展开单引号内的命令替换：这条用例在这台机器上测不出真结果（台账 D8）", layer: layEnv, why: "本机 bash 对单引号内的 `$(…)` 仍做命令替换（WSL/中间层特性）：换了 POSIX 环境自然跑；与 TT 代码无关"},
	{file: "internal/host/dbprobe_test.go", msg: "本机 bash 会展开单引号内的命令替换：这条脚本在这台机器上测不出真结果（台账 D8）", layer: layEnv, why: "同上：本机 bash 会展开单引号内的命令替换"},
	{file: "internal/form42f/form42f_test.go", msg: "没有真样本（%s 打不开：%v）—— 设 TT_FORM_SAMPLE_DIR 指向 fixtures/forms/", layer: layL3, why: "需真样本：开关 TT_FORM_SAMPLE_DIR"},
	{file: "internal/form42f/form42f_test.go", msg: "没有归档语料表（%v）—— 设 TT_FORM_CORPUS_MAP 指向 colname-fieldid.json", layer: layL3, why: "需真归档语料表：开关 TT_FORM_CORPUS_MAP"},
	{file: "internal/form42f/form42f_test.go", msg: "归档语料表里没有 w_axmt500（换台机器/换份语料就跳）", layer: layL3Missing, why: "归档语料表在，但里面没有 w_axmt500（换台机器/换份语料就有）"},
	{file: "internal/form42f/form42f_test.go", msg: "没有真样本（%v）", layer: layL3, why: "需真样本：开关 TT_FORM_SAMPLE_DIR"},
}

// reSkipCall 匹配 t.Skip / t.Skipf 且第一个参数是同行的字符串字面量。
//
// 已知边界（**注意**）：它匹配不到**转义过的**引号 —— 文案里写 `\"` 时，正则会在
// 第一个 `\"` 前停下，抓到的与台账里的对不上（实测踩过：文案里的 `只排除 \"/\" 与 \"#\"`
// 让台账当场报"多一处、少一处"）。所以跳过文案里**不要用半角双引号**，
// 要引用就写中文引号或直接写字符（如「只排除 / 与 #」）。
// 将来若有人写成跨行或拼接，这里会匹配不到 → 台账对不上 → 红。
// 那时该改的是**扫描规则**，不是把台账删掉。
var reSkipCall = regexp.MustCompile(`t\.Skipf?\(\s*"([^"]*)"`)

// scanSkipSites 走 internal/ 下所有 .go，按**文件名字典序 + 文件内出现顺序**产出跳过点。
//
// 顺序不重要（比对按 文件+文案 做键），但必须**确定**：同一份代码扫两次要给同一个结果，
// 否则报错信息会时有时无。
func scanSkipSites(t *testing.T) []skipSite {
	t.Helper()
	root := RepoRoot(t)
	base := filepath.Join(root, "internal")
	var out []skipSite
	err := filepath.WalkDir(base, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".go") {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		rel, relErr := filepath.Rel(root, p)
		if relErr != nil {
			rel = p
		}
		rel = filepath.ToSlash(rel)
		for _, m := range reSkipCall.FindAllStringSubmatch(string(b), -1) {
			out = append(out, skipSite{file: rel, msg: m[1]})
		}
		return nil
	})
	if err != nil {
		t.Fatalf("扫 %s 失败：%v", base, err)
	}
	return out
}

// ledgerKey 是台账比对的键。用 \x00 分隔两个字段，避免"文件结尾恰好连着文案"这种巧合。
func ledgerKey(file, msg string) string { return file + "\x00" + msg }

// TestSkipLedger 台账与代码必须**严格对上**：多一处、少一处、文案改了都红。
//
// 它买到三件事：
//   - "静默跳过"变成"必须改台账"，而改台账是一个会被 review 的 diff
//   - 台账就是分层归属的唯一来源：L3 几条 = 开 TDEV_DEEP 该多跑多少条；
//     L4 几条 = 开 TTZS_E2E 该多跑多少条。"这次到底跳了多少"不再靠数
//   - L3/L4 的 why 必须点名开关名 → **人看到 skip 就知道怎么打开**
func TestSkipLedger(t *testing.T) {
	found := scanSkipSites(t)

	// 防空转：扫描坏了的话（路径变了、正则被改坏），下面所有比对都会"通过"。
	if len(found) < 20 {
		t.Fatalf("只扫到 %d 处 t.Skip —— 扫描本身坏了，此后的比对等于空转", len(found))
	}

	want := make(map[string]skipSite, len(skipLedger))
	for _, s := range skipLedger {
		k := ledgerKey(s.file, s.msg)
		if _, dup := want[k]; dup {
			t.Errorf("台账里 %s 的这条跳过登记了两次：%q", s.file, s.msg)
		}
		want[k] = s
	}

	seen := make(map[string]bool, len(found))
	var unregistered []skipSite
	for _, s := range found {
		k := ledgerKey(s.file, s.msg)
		seen[k] = true
		if _, ok := want[k]; !ok {
			unregistered = append(unregistered, s)
		}
	}
	if len(unregistered) > 0 {
		var b strings.Builder
		for _, s := range unregistered {
			b.WriteString("\n  " + s.file + "\n      " + s.msg)
		}
		t.Errorf("这些跳过没登记：%s\n\n"+
			"新增跳过必须登记进 skipLedger —— 这不是形式主义：\n"+
			"  · 属于**默认档**（L1/L2）的，它应当是断言而不是跳过，请改。\n"+
			"  · 属于 L3/L4 的，登记并注明**开关名**（TDEV_DEEP / TTZS_DEEP / TTZS_E2E /\n"+
			"    TTZS_CORPUS …）—— 人看到 skip 要能知道怎么打开。\n"+
			"  · 属于「L0-环境」的（无 git 之类），写清为什么它不影响「全绿」这个结论。\n"+
			"  · 属于「前提」的要特别当心：前提长期不成立 = 这条测试静默变成空转。\n"+
			"  · 属于「未落地」的（契约已冻结、等某个工作包实现）**必须写清等谁、等什么**\n"+
			"    （例：等 WP3 实现 ref 字段），并让测试里带一句「落地了就把它改成断言」的守卫。\n\n"+
			"台账在 internal/testkit/skipledger_test.go。", b.String())
	}

	// 反向：台账里登记了、代码里却没有 —— 删了跳过却没同步台账。
	// 少了这条，台账会慢慢变成一份没人信的历史。
	var stale []skipSite
	for _, s := range skipLedger {
		if !seen[ledgerKey(s.file, s.msg)] {
			stale = append(stale, s)
		}
	}
	if len(stale) > 0 {
		var b strings.Builder
		for _, s := range stale {
			b.WriteString("\n  " + s.file + "\n      " + s.msg)
		}
		t.Errorf("台账里这些跳过，代码里已经没有了：%s\n\n"+
			"（删掉跳过、或改动了它的文案，都要同步台账。文案改动也算 ——\n"+
			"  文案是「这条跳过在说什么」，改它本就该被看见。）", b.String())
	}
}

// TestSkipLedgerL3L4NamesItsSwitch L3/L4 的台账条目，`why` 里必须点名一个开关。
//
// **这条测的是我写的 why，不是代码** —— 它拦不住"why 写得对但跳过本身是错的"。
// 它挡的是另一件常见的事：加一处跳过、why 随手写一句"环境不对"，于是没人知道
// 该设哪个变量才能跑起来。所以它是纪律断言，不是判据。
func TestSkipLedgerL3L4NamesItsSwitch(t *testing.T) {
	for _, s := range skipLedger {
		// layL3Missing 不在此列：那一类没有开关可点名（见它的注释）。
		if s.layer != layL3 && s.layer != layL4 {
			continue
		}
		named := false
		for _, n := range switchNames {
			if strings.Contains(s.why, n) {
				named = true
				break
			}
		}
		if !named {
			t.Errorf("%s（%s）的 why 没点名开关：%q\n"+
				"    改成「需…：开关 XXX=1」这样 —— 人看到 skip 要能知道怎么打开。\n"+
				"    跳过的是：%q", s.file, s.layer, s.why, s.msg)
		}
	}
}

// TestSkipLedgerHasNoStaleSwitchNames 反向：switchNames 里的每个名字，
// 至少要有一处跳过在用它 —— 否则它多半已经改名了，该从表里删掉。
func TestSkipLedgerHasNoStaleSwitchNames(t *testing.T) {
	joined := ""
	for _, s := range skipLedger {
		joined += s.why + "\n" + s.msg + "\n"
	}
	for _, n := range switchNames {
		if !strings.Contains(joined, n) {
			t.Errorf("开关名 %q 没有任何一处跳过在用它 —— 改名了？从 switchNames 里删掉", n)
		}
	}
}
