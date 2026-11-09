package skill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const assetsDirFixture = `D:\my-skills\kb-faq\assets`

// 指针插在正文首行，前后各留一个空行；frontmatter 一个字节不动。
func TestInjectAssetsPointer(t *testing.T) {
	src := "---\nname: kb-faq\ndescription: 说明\n---\n# 标题\n正文\n"
	want := "---\nname: kb-faq\ndescription: 说明\n---\n\n" +
		AssetsPointerLine(assetsDirFixture) + "\n\n# 标题\n正文\n"

	out, changed := InjectAssetsPointer([]byte(src), assetsDirFixture)
	if !changed {
		t.Fatal("第一次注入该算改动")
	}
	if string(out) != want {
		t.Fatalf("注入结果:\n%q\nwant:\n%q", out, want)
	}
	// frontmatter 必须逐字节不变
	if !strings.HasPrefix(string(out), "---\nname: kb-faq\ndescription: 说明\n---\n") {
		t.Fatalf("frontmatter 被改动了: %q", out)
	}
}

// 原文正文前已经有空行时沿用，不重复添。
func TestInjectAssetsPointerKeepsExistingBlank(t *testing.T) {
	src := "---\nname: k\ndescription: x\n---\n\n\n# 标题\n"
	out, _ := InjectAssetsPointer([]byte(src), assetsDirFixture)
	want := "---\nname: k\ndescription: x\n---\n\n\n" + AssetsPointerLine(assetsDirFixture) + "\n\n# 标题\n"
	if string(out) != want {
		t.Fatalf("注入结果:\n%q\nwant:\n%q", out, want)
	}
}

// 重复注入必须幂等：一个字都不多。
func TestInjectAssetsPointerIsIdempotent(t *testing.T) {
	src := []byte("---\nname: k\ndescription: x\n---\n# 标题\n")
	once, _ := InjectAssetsPointer(src, assetsDirFixture)
	twice, changed := InjectAssetsPointer(once, assetsDirFixture)
	if changed {
		t.Error("内容相同不该算改动")
	}
	if string(twice) != string(once) {
		t.Fatalf("第二次注入改变了内容:\n%q\n%q", once, twice)
	}
	// 再刷一次仍是同一份（--force 刷十次也只留一行）
	for i := 0; i < 3; i++ {
		again, _ := InjectAssetsPointer(twice, assetsDirFixture)
		if string(again) != string(once) {
			t.Fatalf("第 %d 次重复注入改变了内容", i+3)
		}
	}
	if n := strings.Count(string(once), assetsPointerSentinel); n != 1 {
		t.Errorf("指针行出现了 %d 次，want 1", n)
	}
}

// 源目录换了位置要就地纠正那一行，而不是再加一行。
func TestInjectAssetsPointerReplacesChangedPath(t *testing.T) {
	src := []byte("---\nname: k\ndescription: x\n---\n# 标题\n")
	old, _ := InjectAssetsPointer(src, `D:\old\k\assets`)
	got, changed := InjectAssetsPointer(old, `E:\new\k\assets`)
	if !changed {
		t.Fatal("路径变了该算改动")
	}
	if bytes := string(got); strings.Contains(bytes, `D:\old`) {
		t.Fatalf("旧路径还在: %q", bytes)
	}
	if n := strings.Count(string(got), assetsPointerSentinel); n != 1 {
		t.Fatalf("指针行有 %d 行，want 1: %q", n, got)
	}
}

// CRLF 与 BOM 都要保住：SKILL.md 是给人也给人写的文件，不该被我们统一换行或吃掉 BOM。
func TestInjectAssetsPointerPreservesCRLFAndBOM(t *testing.T) {
	src := "\ufeff---\r\nname: k\r\ndescription: x\r\n---\r\n# 标题\r\n"
	out, changed := InjectAssetsPointer([]byte(src), assetsDirFixture)
	if !changed {
		t.Fatal("该算改动")
	}
	got := string(out)
	if !strings.HasPrefix(got, "\ufeff---\r\nname: k\r\ndescription: x\r\n---\r\n") {
		t.Fatalf("BOM 或 frontmatter 被动了: %q", got)
	}
	if strings.Contains(strings.ReplaceAll(got, "\r\n", ""), "\n") {
		t.Fatalf("混进了裸 LF: %q", got)
	}
	want := "\ufeff---\r\nname: k\r\ndescription: x\r\n---\r\n\r\n" +
		AssetsPointerLine(assetsDirFixture) + "\r\n\r\n# 标题\r\n"
	if got != want {
		t.Fatalf("CRLF 注入结果:\n%q\nwant:\n%q", got, want)
	}
}

// 没有 frontmatter 的文件原样返回（这种文件本来就过不了 Inspect*）。
func TestInjectAssetsPointerLeavesNonSkillAlone(t *testing.T) {
	src := []byte("# 不是技能\n正文\n")
	out, changed := InjectAssetsPointer(src, assetsDirFixture)
	if changed || string(out) != string(src) {
		t.Fatalf("不该动它: changed=%v out=%q", changed, out)
	}
	// 没有闭合 --- 也一样
	src2 := []byte("---\nname: k\n")
	if out, changed := InjectAssetsPointer(src2, assetsDirFixture); changed || string(out) != string(src2) {
		t.Fatalf("没有闭合的 frontmatter 不该动: %q", out)
	}
}

// 只有 frontmatter、没有正文：追加在末尾。
func TestInjectAssetsPointerEmptyBody(t *testing.T) {
	src := []byte("---\nname: k\ndescription: x\n---\n")
	out, changed := InjectAssetsPointer(src, assetsDirFixture)
	if !changed {
		t.Fatal("该算改动")
	}
	want := "---\nname: k\ndescription: x\n---\n\n" + AssetsPointerLine(assetsDirFixture) + "\n"
	if string(out) != want {
		t.Fatalf("结果 %q, want %q", out, want)
	}
}

func TestHasAssets(t *testing.T) {
	dir := t.TempDir()
	if HasAssets(dir) {
		t.Error("没有 assets/ 时不该说有")
	}
	if err := os.MkdirAll(filepath.Join(dir, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if HasAssets(dir) {
		t.Error("空的 assets/ 不该说有（为一个空目录写指针只会误导模型）")
	}
	if err := os.MkdirAll(filepath.Join(dir, "assets", "deep"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "assets", "deep", "a.pdf"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !HasAssets(dir) {
		t.Error("assets/deep/a.pdf 存在时该说有")
	}
}
