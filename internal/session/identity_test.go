package session

// 身份规则的判据：序号的作用域、空号复用、已断开不占号、世代唯一。
//
// 这几条都是"改一行就静默出错"的类型：序号串了会让两条会话撞地址（`恒烁正式区#1` 指向谁
// 不确定），世代重了会让"旧句柄失效"这条保护整个失效 —— 而两种错都不会报错。

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestNextSeqIsScopedToEnvironment：序号按**环境**算，企业/全局都不参与。
func TestNextSeqIsScopedToEnvironment(t *testing.T) {
	slots := []Slot{
		{Env: "主机109", N: 1},
		{Env: "主机109", N: 2},
		{Env: "测试1012", N: 1},
	}
	if got := NextSeq("主机109", slots); got != 3 {
		t.Fatalf("主机109 的下一个号该是 3，实得 %d", got)
	}
	if got := NextSeq("测试1012", slots); got != 2 {
		t.Fatalf("测试1012 的下一个号该是 2（另一个环境不参与），实得 %d", got)
	}
	if got := NextSeq("从没出现过的环境", slots); got != 1 {
		t.Fatalf("新环境该从 1 开始，实得 %d", got)
	}
}

// TestNextSeqTakesTheSmallestHole：会话消失后它的号要能被接手（最小空号，不是最大+1）。
func TestNextSeqTakesTheSmallestHole(t *testing.T) {
	slots := []Slot{
		{Env: "主机109", N: 1},
		// 2 空着（那条会话已经没了）
		{Env: "主机109", N: 3},
	}
	if got := NextSeq("主机109", slots); got != 2 {
		t.Fatalf("该接手空出来的 2，实得 %d", got)
	}
}

// TestNextSeqIgnoresExitedAndUnassigned：已断开的残留不占号；N<=0 的未分配也不算占用。
func TestNextSeqIgnoresExitedAndUnassigned(t *testing.T) {
	slots := []Slot{
		{Env: "主机109", N: 1, Exited: true}, // 已断开
		{Env: "主机109", N: 0},               // 还没分配
	}
	if got := NextSeq("主机109", slots); got != 1 {
		t.Fatalf("已断开的 #1 不该挡住新会话，实得 #%d", got)
	}
}

// TestNewEpochIsMonotonicAndUnique：世代号单调、唯一、base36。
func TestNewEpochIsMonotonicAndUnique(t *testing.T) {
	const n = 64
	seen := make(map[string]bool, n)
	prev := int64(0)
	for i := 0; i < n; i++ {
		e := NewEpoch()
		if e == "" {
			t.Fatal("世代号不许为空")
		}
		if seen[e] {
			t.Fatalf("世代号重复：%q —— 旧句柄失效这条保护会整个化为乌有", e)
		}
		seen[e] = true
		v, err := strconv.ParseInt(e, 36, 64)
		if err != nil {
			t.Fatalf("世代号必须是 base36：%q (%v)", e, err)
		}
		if v <= prev {
			t.Fatalf("世代号必须单调递增：%q 之后给了 %q", strconv.FormatInt(prev, 36), e)
		}
		prev = v
	}
}

// TestEpochIsIssuedOnlyHere：**世代号只有一处发号**。
//
// 这条是给未来的自己看的：一旦有人在别处又写一个 `strconv.FormatInt(xxx.Add(1), 36)`，
// 就会出现两套互不相识的计数 —— 两边都可能发出同一个号，而"旧句柄失效"正是靠"号绝不重复"。
// 那种错不会报错，只会在某次复用之后把旧句柄当成有效的。
func TestEpochIsIssuedOnlyHere(t *testing.T) {
	const root = ".." // internal/
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("读 internal/ 失败：%v", err)
	}
	for _, e := range entries {
		name := e.Name()
		if !e.IsDir() || name == "session" {
			continue // 本包自己
		}
		if err := filepath.WalkDir(filepath.Join(root, name), func(path string, d os.DirEntry, werr error) error {
			if werr != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return werr
			}
			b, rerr := os.ReadFile(path)
			if rerr != nil {
				return rerr
			}
			src := string(b)
			// 只看代码行：注释里提"世代号"不违规。
			for _, line := range strings.Split(src, "\n") {
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, "//") {
					continue
				}
				if strings.Contains(trimmed, "epochSeq.Add(") {
					t.Errorf("%s 里出现了 epochSeq.Add —— 世代号只许在 internal/session 里发（理由见本文件头）", path)
				}
			}
			return nil
		}); err != nil {
			t.Fatalf("遍历失败：%v", err)
		}
	}
}
