// 重新生成 testdata/tzs-mini/ws —— 仓库自带的**最小 .tzs 工作区**。
//
// 为什么要它：`.tzs` 的语料回归从前只有"有真客户语料的那台机器"能跑（一份完整工作区 130 MB，
// 而且 `mta/` + `tbl/` 是数据不是代码）。这份夹具把那份语料**缩到 3.2 MB 并入库**，
// 于是任何人 `cd engine && ./build.sh` 之后就能跑那几条回归。
//
// 用法：
//
//	go run ./tools/tzsmini --src D:\t100_wrok_dir\hengshuo\prd
//	go run ./tools/tzsmini --src <语料根> --out <别处>
//
// `--src` 必须是**一个工作区**（该目录下有 `mta/`），包摊在它顶层。
// （前身是 Python 脚本 testdata/tzs-mini/build.py，2026-10 随「tools 去 Python」改成 Go
// 并挪进 tools/；清单的形状与生成规则一个字节都没变 —— 这是有意的，见下面写 manifest 那段。）
//
// ## 这份工作区里为什么是这几样（每一条都是实测，别"顺手简化"）
//
//	mta/   全量**减去三个大文件**（zooms.xml 4.8M / subroutines.xml 3M / messages.xml 2.6M）。
//	       **其余一个都不能少**：裁到只剩 tables.xml + ver + mod-fd.spec + core-br.spec 时，
//	       引擎照样回 `SUMMARY|ok`，但 `aapt300` 的布局元素从 515 悄悄变成 512 —— 静默少东西，
//	       正是这个仓库最怕的那种失败。留下 `tsd.xsd` 与否不影响这三个包。
//	<模块>/tbl/<表>.tbl
//	       只拷**被包引用到的表**。模块名取 `mta/tables.xml` 里该表 `module` 属性的小写，
//	       **不是**包名前三位（`cpmp530` 的表 `pmdo_t` 住在 `apm/`）。
//	<包>.tzs
//	       用例本身，见下面的 PACKAGES。
//
// ## 两个路径陷阱（踩过）
//
//   - 工作区字符串必须用**反斜杠**。设计器会归一化**包路径**、不会归一化工作区字符串，
//     所以 `TZSCLI_WS=C:/ws` + 包任意形式一律 `NotInCurrentWorkspaceException`。
//     包路径用正斜杠没关系。完整规则见 skills/tt-dev-tzs/SKILL.md §4.4。
//   - 包必须**在工作区目录之下**（纯字符串前缀比较），所以别把包拷到 %TEMP% 再喂给引擎。
package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// 三个包是按"小 / 中 / 大 + 特征"挑的，不是随手抓的（数见 testdata/tzs-mini/README.md 的表）：
var packages = []string{
	"cpmp530(c).tzs", // 最小：12 个规格节点 / 3 种 kind / 9 个容器
	"aapp320(c).tzs", // 中：5 种 kind / 21 种控件 / 3 个标签页
	"aapt300(c).tzs", // 大：6 种 kind（含 tree）/ 26 种控件 / 12 页 / 26 个动作 / 65 个容器
}

// 去掉的只是这三份**目录性**数据；它们不参与表单加载，实测不影响上面三个包的任何计数。
var drop = map[string]bool{
	"zooms.xml":       true,
	"subroutines.xml": true,
	"messages.xml":    true,
}

var (
	tableAttrRe = regexp.MustCompile(`table="([a-zA-Z0-9_]+)"`)
	tableElemRe = regexp.MustCompile(`<table[^>]*name="([a-zA-Z0-9_]+)"`)
	tableTagRe  = regexp.MustCompile(`<table\b([^>]*)>`)
	nameAttrRe  = regexp.MustCompile(`name="([a-zA-Z0-9_]+)"`)
	moduleRe    = regexp.MustCompile(`module="([^"]+)"`)
)

// referencedTables 收一个包里出现过的表名。两种写法都要收：
// `table="x"`（属性）与 `<table ... name="x">`。
func referencedTables(pkg string) (map[string]bool, error) {
	found := map[string]bool{}
	zr, err := zip.OpenReader(pkg)
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		raw, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, err
		}
		// 对应 Python 版的 decode("utf-8", "replace")：坏字节换成 U+FFFD，正则只找 ASCII。
		text := strings.ToValidUTF8(string(raw), "\uFFFD")
		for _, m := range tableAttrRe.FindAllStringSubmatch(text, -1) {
			found[m[1]] = true
		}
		for _, m := range tableElemRe.FindAllStringSubmatch(text, -1) {
			found[m[1]] = true
		}
	}
	delete(found, "true")
	delete(found, "false")
	return found, nil
}

// moduleMap 建表名 → 模块目录名（小写）。tables.xml 里 module 属性与 name 属性的
// 先后不固定，两种都试。
func moduleMap(src string) (map[string]string, error) {
	raw, err := os.ReadFile(filepath.Join(src, "mta", "tables.xml"))
	if err != nil {
		return nil, err
	}
	text := strings.ToValidUTF8(string(raw), "\uFFFD")
	out := map[string]string{}
	for _, m := range tableTagRe.FindAllStringSubmatch(text, -1) {
		attrs := m[1]
		n := nameAttrRe.FindStringSubmatch(attrs)
		mod := moduleRe.FindStringSubmatch(attrs)
		if n != nil && mod != nil {
			out[n[1]] = strings.ToLower(mod[1])
		}
	}
	return out, nil
}

// repoRoot 从当前工作目录逐级上溯找 go.mod —— 工具挪进 tools/ 之后，
// 缺省输出目录得按仓库根定位（包目录/脚本目录都不再是定位锚）。
func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("从 %s 逐级上溯都没找到 go.mod —— 请在仓库里运行本工具", dir)
		}
		dir = parent
	}
}

// copyFile 对应 shutil.copy：拷内容，也拷权限位。
func copyFile(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func main() {
	var srcFlag, outFlag string
	repo, repoErr := repoRoot()
	defaultOut := ""
	if repoErr == nil {
		defaultOut = filepath.Join(repo, "testdata", "tzs-mini", "ws")
	}
	// 手写参数解析，保持与 Python 版相同的两个开关（--src / --out），
	// 报错文案也保持同一句。
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--src":
			if i+1 >= len(args) {
				die("--src 需要一个值")
			}
			i++
			srcFlag = args[i]
		case "--out":
			if i+1 >= len(args) {
				die("--out 需要一个值")
			}
			i++
			outFlag = args[i]
		default:
			die("不认识的参数 %s（用法：go run ./tools/tzsmini --src <真工作区> [--out <别处>]）", args[i])
		}
	}
	if srcFlag == "" {
		die("--src 必填：指向一个真工作区（该目录下有 mta/）")
	}
	if outFlag == "" {
		if repoErr != nil {
			die("%v", repoErr)
		}
		outFlag = defaultOut
	}

	src, err := filepath.Abs(srcFlag)
	if err != nil {
		die("%v", err)
	}
	out, err := filepath.Abs(outFlag)
	if err != nil {
		die("%v", err)
	}
	if st, err := os.Stat(filepath.Join(src, "mta")); err != nil || !st.IsDir() {
		die("%s 不是一个工作区：它下面没有 mta/", src)
	}

	if err := os.RemoveAll(out); err != nil {
		die("%v", err)
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		die("%v", err)
	}

	// mta/ —— 全量减三
	mtaSrc := filepath.Join(src, "mta")
	entries, err := os.ReadDir(mtaSrc)
	if err != nil {
		die("%v", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && !drop[e.Name()] {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	if err := os.MkdirAll(filepath.Join(out, "mta"), 0o755); err != nil {
		die("%v", err)
	}
	for _, name := range names {
		if err := copyFile(filepath.Join(mtaSrc, name), filepath.Join(out, "mta", name)); err != nil {
			die("%v", err)
		}
	}
	nMta := len(names)

	mod, err := moduleMap(src)
	if err != nil {
		die("%v", err)
	}
	nTbl := 0
	for _, pkg := range packages {
		srcPkg := filepath.Join(src, pkg)
		if _, err := os.Stat(srcPkg); err != nil {
			die("找不到包 %s", srcPkg)
		}
		if err := copyFile(srcPkg, filepath.Join(out, pkg)); err != nil {
			die("%v", err)
		}
		tables, err := referencedTables(srcPkg)
		if err != nil {
			die("%v", err)
		}
		sorted := make([]string, 0, len(tables))
		for t := range tables {
			sorted = append(sorted, t)
		}
		sort.Strings(sorted)
		for _, t := range sorted {
			m, ok := mod[t]
			if !ok {
				die("tables.xml 里没有表 %s（包 %s）—— 源语料与包不配套？", t, pkg)
			}
			s := filepath.Join(src, m, "tbl", t+".tbl")
			if _, err := os.Stat(s); err != nil {
				die("找不到 %s（包 %s 用到 %s）", s, pkg, t)
			}
			if err := copyFile(s, filepath.Join(out, m, "tbl", t+".tbl")); err != nil {
				die("%v", err)
			}
			nTbl++
		}
	}

	// 汇总（与 Python 版同一句：KB 是整除）。
	var files []string
	var total int64
	err = filepath.WalkDir(out, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		files = append(files, p)
		total += info.Size()
		return nil
	})
	if err != nil {
		die("%v", err)
	}
	fmt.Printf("%s\n  mta/ %d 个 + tbl/ %d 个 + 包 %d 个 = %d 个文件 / %d KB\n",
		out, nMta, nTbl, len(packages), len(files), total/1024)

	// 钉住：一份**清单**，由 TestMiniCorpusIsPinned 逐条比对。
	//
	// 为什么必须有：那三条回归的判据是**基线相对**的（`requireFixedPoint` 拿 pristine 自己
	// 当基线），所以一份重新生成错了的夹具会**自洽通过** —— 实测过一次：mta/ 裁到只剩四个文件时
	// `aapt300` 的布局元素从 515 静默变成 512，而 `SUMMARY` 还是 `ok`。
	// 清单把"这一份就是那一份"变成可判的。
	//
	// 路径用**仓库相对**的正斜杠：夹具在仓库里，不是机器特定的语料 ——
	// 所以这条断言不需要 engine/corpus.manifest 那种"指着另一份语料就跳过"的退路。
	// 排序按完整 OS 路径（与 Python 版 sorted() 同一口径），保证新旧清单可逐行 diff。
	sorted := append([]string(nil), files...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	var b strings.Builder
	b.WriteString("# 最小语料的固定清单 —— 由 tools/tzsmini 生成，内部/开发者不改。\n")
	b.WriteString("# 重新生成：go run ./tools/tzsmini --src <真工作区>\n")
	b.WriteString("# 形状：<sha256-16>  <字节>  <仓库相对路径>\n")
	b.WriteString("# 为什么要有它：见 tools/tzsmini 里写它那一段注释。\n")
	for _, p := range sorted {
		rel, err := filepath.Rel(out, p)
		if err != nil {
			die("%v", err)
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			die("%v", err)
		}
		sum := sha256.Sum256(raw)
		info, err := os.Stat(p)
		if err != nil {
			die("%v", err)
		}
		b.WriteString(fmt.Sprintf("%s  %8d  %s\n",
			hex.EncodeToString(sum[:8]), info.Size(), filepath.ToSlash(rel)))
	}
	manifest := filepath.Join(filepath.Dir(filepath.Clean(out)), "manifest.txt")
	if err := os.WriteFile(manifest, []byte(b.String()), 0o644); err != nil {
		die("%v", err)
	}
	fmt.Printf("  manifest.txt %d 条\n", len(sorted))
}
