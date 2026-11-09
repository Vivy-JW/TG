package skill

import (
	"strings"
	"testing"
)

// frontmatter 的几种合法写法都要认，尤其是：缩进行属于 metadata 嵌套块，
// 不能把它底下的 name/description 当成技能自己的。
func TestParseFrontmatterShapes(t *testing.T) {
	cases := []struct {
		name string
		body string
		want Meta
	}{
		{"基本", "---\nname: alpha\ndescription: 说明\n---\n", Meta{"alpha", "说明"}},
		{"带引号", "---\nname: \"alpha\"\ndescription: '带引号的说明'\n---\n", Meta{"alpha", "带引号的说明"}},
		{"块标量 |", "---\nname: alpha\ndescription: |\n  第一行\n  第二行\n---\n", Meta{"alpha", "第一行 第二行"}},
		{"块标量 >-", "---\nname: alpha\ndescription: >-\n  第一行\n  第二行\n---\n", Meta{"alpha", "第一行 第二行"}},
		{"嵌套块不算数", "---\nname: nested\nmetadata:\n  name: 不算\n  description: 也不算\ndescription: 正经说明\n---\n",
			Meta{"nested", "正经说明"}},
		{"CRLF", "---\r\nname: alpha\r\ndescription: 说明\r\n---\r\n", Meta{"alpha", "说明"}},
		{"BOM", "\ufeff---\nname: alpha\ndescription: 说明\n---\n", Meta{"alpha", "说明"}},
		{"注释行跳过", "---\n# 注释\nname: alpha\ndescription: 说明\n---\n", Meta{"alpha", "说明"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ParseFrontmatter([]byte(c.body))
			if err != nil {
				t.Fatalf("不该报错: %v", err)
			}
			if got != c.want {
				t.Fatalf("解析出 %+v, want %+v", got, c.want)
			}
		})
	}
}

func TestParseFrontmatterErrors(t *testing.T) {
	cases := []struct{ name, body, want string }{
		{"没有 frontmatter", "name: alpha\ndescription: x\n", "没有 YAML frontmatter"},
		{"没有闭合", "---\nname: alpha\ndescription: x\n", "没有闭合"},
		{"空文件", "", "没有 YAML frontmatter"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := ParseFrontmatter([]byte(c.body))
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("err = %v, 想要包含 %q", err, c.want)
			}
		})
	}
}

// 不合规的 frontmatter 必须被判死，而不是装出去。
//
// 为什么值得一条：name 与 description 是别的 agent 拿来当索引的两个字段（name 决定技能
// 身份、description 决定何时加载）。装过去不会报错 —— 失败都在我们看不到的地方：
// description 缺失时客户端直接跳过，name 不匹配则各家宽容度不一。
func TestValidateMetaRejects(t *testing.T) {
	long := strings.Repeat("长", DescriptionMax+1)
	cases := []struct{ dir, body, want string }{
		{"alpha", "---\nname: beta\ndescription: x\n---\n", "必须等于目录名"},
		{"alpha", "---\nname: Alpha\ndescription: x\n---\n", "只允许小写字母"},
		{"_alpha", "---\nname: _alpha\ndescription: x\n---\n", "只允许小写字母"},
		{"alpha", "---\nname: alpha-\ndescription: x\n---\n", "只允许小写字母"},
		{"alpha", "---\nname: a--b\ndescription: x\n---\n", "只允许小写字母"},
		{"alpha", "---\ndescription: x\n---\n", "没有 name"},
		{"alpha", "---\nname: alpha\n---\n", "description 是空的"},
		{"alpha", "---\nname: alpha\ndescription: \"\"\n---\n", "description 是空的"},
		{"alpha", "---\nname: alpha\ndescription: " + long + "\n---\n", "超过规范上限"},
	}
	for _, c := range cases {
		meta, err := ParseFrontmatter([]byte(c.body))
		if err != nil {
			continue // 结构层面的错误在 ParseFrontmatter 的用例里覆盖
		}
		probs := ValidateMeta(c.dir, meta)
		if len(probs) == 0 {
			t.Errorf("%q: 应被判不合规", c.body)
			continue
		}
		if got := strings.Join(probs, "；"); !strings.Contains(got, c.want) {
			t.Errorf("%q: 问题里应有 %q, 实得 %q", c.body, c.want, got)
		}
	}
}

func TestValidName(t *testing.T) {
	good := []string{"a", "abc", "a-b", "a1", "tt-dev-tzc", strings.Repeat("a", 64)}
	for _, s := range good {
		if !ValidName(s) {
			t.Errorf("%q 该合规", s)
		}
	}
	bad := []string{"", "-a", "a-", "a--b", "A", "a_b", "a b", "技能", strings.Repeat("a", 65)}
	for _, s := range bad {
		if ValidName(s) {
			t.Errorf("%q 该不合规", s)
		}
	}
}

func TestValidateMetaAcceptsNameEqualsDir(t *testing.T) {
	if probs := ValidateMeta("kb-faq", Meta{Name: "kb-faq", Description: "知识库"}); len(probs) != 0 {
		t.Fatalf("合规技能被判成: %v", probs)
	}
	if probs := ValidateMeta("kb-faq", Meta{Name: "kb-faq", Description: strings.Repeat("长", DescriptionMax)}); len(probs) != 0 {
		t.Fatalf("description 刚好 %d 字符该合规: %v", DescriptionMax, probs)
	}
}
