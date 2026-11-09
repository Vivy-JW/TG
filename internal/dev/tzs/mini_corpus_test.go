package tzs

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"tt/internal/dev/testutil"
)

// mini_corpus_test.go —— 跑在**仓库自带**的那份最小语料上的三条回归。
//
// ## 为什么要有它
//
// corpus_test.go 那几条只认 `TTZS_CORPUS` 指的真语料：一份完整工作区 **130 MB**，而且
// `mta/` + `tbl/` 是数据不是代码（客户字典与 schema）。于是那几条回归
// **只有有那份语料的那台机器能跑** —— 干净克隆上全跳。
//
// 这里把挑出来的三个典型包 + 它们需要的最小元数据放进 `testdata/tzs-mini/ws`
// （**3.2 MB**，生成方式见 `tools/tzsmini` 与该目录的 README.md），
// 于是任何人 clone 之后 `cd engine && ./build.sh`，就能跑 roundtrip / 写路径 / 改路径。
//
// ## 它不替代什么
//
// 三个包能证明"这条链路没坏"，**证明不了**"上百个真实包的形状分布还都对"。
// 那一条仍然归 `TTZS_DEEP`（见 corpus_test.go 顶部那两条硬纪律）。
// 这份夹具的定位是**冒烟网**，不是回归网。
//
// ## 前置：引擎还要有，夹具不许缺
//
// 引擎不在仓库里（`engine/out/` 是构建产物，见 engine/BUILD.md），所以**引擎缺了就跳过** ——
// 与 corpus_test.go 同一个 `engineExeAndDir`，同一句跳过文案。
//
// 而**夹具缺了不跳过**：它是仓库本体，缺了就是仓库坏了。依据是 internal/cli/dev/tzs_verb_test.go
// 立的那条（"这些属于仓库本体，不是语料那样的外部数据 —— 缺了就是仓库坏了"），
// internal/dev/fgl 的双份夹具守卫用的是同一条。

// miniCorpusRoot 从包目录逐级上溯找 `testdata/tzs-mini/ws`。
//
// 上溯而不是写死相对路径：测试的工作目录是包目录，而夹具落在仓库根 —— 与
// internal/dev/fgl/outline_fixtures_test.go:147 找 `testdata/fgl-fixtures` 是同一个做法。
func miniCorpusRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("拿不到工作目录：%v", err)
	}
	for {
		cand := filepath.Join(dir, "testdata", "tzs-mini", "ws")
		if st, err := os.Stat(filepath.Join(cand, "mta")); err == nil && st.IsDir() {
			return cand
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatalf("缺少最小语料 testdata/tzs-mini/ws（从包目录 %s 上溯未找到）——\n"+
		"它是仓库本体，不是可选的语料：缺了就是仓库坏了。\n"+
		"若是误删，用 `go run ./tools/tzsmini --src <真工作区>` 重新生成。",
		mustGetwd(t))
	return ""
}

// requireMiniCorpus 把「真引擎 + 仓库自带的语料」拼成一次回归的上下文。
//
// 它**不要求 TTZS_DEEP**，也**不读 TTZS_CORPUS**：语料是仓库里那份，跟机器无关。
// 唯一的外部前置是引擎。
func requireMiniCorpus(t *testing.T) *corpusEnv {
	t.Helper()
	exe, dir := engineExeAndDir(t)
	root := miniCorpusRoot(t)
	pkgs := testutil.CorpusFiles(root, ".tzs", scratchPrefixes...)
	if len(pkgs) == 0 {
		t.Fatalf("%s 里一个 .tzs 都没有 —— 夹具被清空了（它是仓库本体，不是外部语料）", root)
	}
	return &corpusEnv{exe: exe, dir: dir, root: root, pkgs: pkgs}
}

func mustGetwd(t *testing.T) string {
	t.Helper()
	d, err := os.Getwd()
	if err != nil {
		return "?"
	}
	return d
}

// TestMiniCorpusRoundTrip 证明"设计器读得进这三个包，并且用它自己的模型存出来的还是同一个定点"。
// 判据与 TestCorpusRoundTrip 逐条相同（四个零 + stale 等于 pristine 的 stale）。
func TestMiniCorpusRoundTrip(t *testing.T) {
	corpusPass(t, requireMiniCorpus(t), corpusOp{name: "roundtrip", key: "roundtrip"})
}

// TestMiniCorpusWritePath 是 TestCorpusWritePath 的小号：往三个包里各加一个字段。
func TestMiniCorpusWritePath(t *testing.T) {
	corpusPass(t, requireMiniCorpus(t), writeOp())
}

// TestMiniCorpusEditPath 是 TestCorpusEditPath 的小号：改一个 spec 属性并回读。
func TestMiniCorpusEditPath(t *testing.T) {
	corpusPass(t, requireMiniCorpus(t), editOp())
}

// TestMiniCorpusIsPinned 钉住夹具的字节。
//
// ## 为什么这条不是"锦上添花"
//
// 上面三条的判据是**基线相对**的：`requireFixedPoint` 拿 pristine **自己**当基线。
// 于是有个反直觉的后果 —— **一份重新生成错了的夹具会自洽通过**。实测过一次：
// 把 `mta/` 裁到只剩 `tables.xml`/`ver`/`mod-fd.spec`/`core-br.spec` 时，
// 引擎照样回 `SUMMARY|ok`，而 `aapt300` 的布局元素已经从 515 悄悄变成 512；
// 上面三条全能过，因为 512 与 512 自己比是相等的。
//
// 所以"这一份就是那一份"必须自己是一条判据。这与 `engine/corpus.manifest` + `TestCorpusPin`
// 是同一个装置，区别只有一个：**那份语料是机器路径，这份在仓库里** ——
// 所以这里不需要 corpus.manifest 那种"pin 指着另一份语料就跳过"的退路，缺一条就是红。
//
// 不需要引擎，所以它比上面三条跑得快得多（也只有它能在没有引擎的机器上守住夹具）。
func TestMiniCorpusIsPinned(t *testing.T) {
	root := miniCorpusRoot(t)
	manifest := filepath.Join(filepath.Dir(root), "manifest.txt") // ws/ 的兄弟

	// 复用 corpus_test.go 的解析器：清单形状是同一个（`<sha16> <字节> <路径>`），
	// 唯一差别是这里的路径是**仓库相对**的。再写一份解析器就是"同一件事两份实现"。
	pins := map[string]string{}
	for _, r := range readCorpusPin(t, manifest) {
		pins[r.Path] = r.Sha16
	}

	// 磁盘上的实际文件（仓库相对 + 正斜杠），与清单同一套口径。
	onDisk := map[string]string{}
	err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, rerr := filepath.Rel(root, p)
		if rerr != nil {
			return rerr
		}
		onDisk[filepath.ToSlash(rel)] = p
		return nil
	})
	if err != nil {
		t.Fatalf("遍历 %s 失败：%v", root, err)
	}

	var missing, extra, changed []string
	for rel, want := range pins {
		p, ok := onDisk[rel]
		if !ok {
			missing = append(missing, rel)
			continue
		}
		got, herr := sha16(p)
		if herr != nil {
			t.Fatalf("算 %s 的 sha256 失败：%v", p, herr)
		}
		if got != want {
			changed = append(changed, rel)
		}
	}
	for rel := range onDisk {
		if _, ok := pins[rel]; !ok {
			extra = append(extra, rel)
		}
	}
	if len(missing)+len(extra)+len(changed) > 0 {
		sort.Strings(missing)
		sort.Strings(extra)
		sort.Strings(changed)
		t.Errorf("最小语料与 manifest.txt 对不上：\n"+
			"  清单里有、磁盘上没有：%v\n  磁盘上有、清单里没有：%v\n  内容变了：%v\n\n"+
			"这是**故意的摩擦**：三条回归的基线是从这份夹具自己量的，所以它一旦悄悄变了，\n"+
			"判据会跟着一起搬而全是绿的。改夹具请连 manifest 一起重新生成：\n"+
			"  go run ./tools/tzsmini --src <真工作区>",
			missing, extra, changed)
	}
}
