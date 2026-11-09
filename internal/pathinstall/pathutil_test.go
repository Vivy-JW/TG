package pathinstall

import "testing"

func TestSplitPathList(t *testing.T) {
	got := splitPathList(`C:\a;;C:\b ;  ;C:\c`, ";")
	want := []string{`C:\a`, `C:\b`, `C:\c`}
	if len(got) != len(want) {
		t.Fatalf("split = %q", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("split[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestPathListAddRemove(t *testing.T) {
	list := []string{`C:\a`, `C:\b`}

	// 追加 + 幂等
	list = pathListAdd(list, `C:\c`, true)
	if len(list) != 3 || list[2] != `C:\c` {
		t.Fatalf("add = %q", list)
	}
	list = pathListAdd(list, `c:\C\`, true) // 大小写 + 尾分隔符:视为同一个
	if len(list) != 3 {
		t.Fatalf("幂等失败: %q", list)
	}
	if !pathListContains(list, `C:\c`, true) {
		t.Fatal("contains 应命中")
	}

	// 空目录不入列
	if got := pathListAdd([]string{`C:\a`}, "", true); len(got) != 1 {
		t.Fatalf("空目录不应加入: %q", got)
	}

	// 移除(忽略大小写/尾分隔符)
	list = pathListRemove(list, `C:\C`, true)
	if len(list) != 2 || pathListContains(list, `C:\c`, true) {
		t.Fatalf("remove = %q", list)
	}
	// 移除不存在项 = 原样
	if got := pathListRemove(list, `C:\zz`, true); len(got) != 2 {
		t.Fatalf("移除不存在项应原样: %q", got)
	}
}

// 非折叠(类 Unix)比较区分大小写。
func TestPathListCaseSensitive(t *testing.T) {
	list := []string{"/usr/bin"}
	if pathListContains(list, "/USR/BIN", false) {
		t.Fatal("非折叠比较不应命中大小写不同的路径")
	}
	if !pathListContains(list, "/usr/bin/", false) {
		t.Fatal("应忽略尾部分隔符")
	}
}
