package cli

import "testing"

func TestNormalizeGPMsgArgs(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{
			name: "负号编号在环境参数前",
			in:   []string{"tt", "gp", "msg", "-0101", "--env", "22主机"},
			want: []string{"tt", "gp", "msg", "--code=-0101", "--env", "22主机"},
		},
		{
			name: "负号编号在环境参数后",
			in:   []string{"tt", "gp", "msg", "--env", "22主机", "-0101"},
			want: []string{"tt", "gp", "msg", "--env", "22主机", "--code=-0101"},
		},
		{
			name: "布尔 flag 后接负号编号",
			in:   []string{"tt", "gp", "msg", "--json", "-0101", "--env", "22主机"},
			want: []string{"tt", "gp", "msg", "--json", "--code=-0101", "--env", "22主机"},
		},
		{
			name: "带值 flag 后接负号编号",
			in:   []string{"tt", "gp", "msg", "--lang", "zh_CN", "-0101"},
			want: []string{"tt", "gp", "msg", "--lang", "zh_CN", "--code=-0101"},
		},
		{
			name: "短布尔 flag 后接负号编号",
			in:   []string{"tt", "gp", "msg", "-v", "-0101"},
			want: []string{"tt", "gp", "msg", "-v", "--code=-0101"},
		},
		{
			name: "p_ze 别名",
			in:   []string{"tt", "gp", "p_ze", "-0101"},
			want: []string{"tt", "gp", "p_ze", "--code=-0101"},
		},
		{
			name: "link 不动",
			in:   []string{"tt", "gp", "link", "-0101"},
			want: []string{"tt", "gp", "link", "-0101"},
		},
		{
			name: "显式 -- 后不动",
			in:   []string{"tt", "gp", "msg", "--", "-0101"},
			want: []string{"tt", "gp", "msg", "--", "-0101"},
		},
		{
			name: "普通编号不动",
			in:   []string{"tt", "gp", "msg", "abc"},
			want: []string{"tt", "gp", "msg", "abc"},
		},
		{
			name: "已有 code flag 不动",
			in:   []string{"tt", "gp", "msg", "--code=-0101"},
			want: []string{"tt", "gp", "msg", "--code=-0101"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := normalizeGPMsgArgs(append([]string(nil), c.in...))
			if len(got) != len(c.want) {
				t.Fatalf("长度 %d，want %d: %v", len(got), len(c.want), got)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Fatalf("第 %d 个 = %q，want %q（全量 %v）", i, got[i], c.want[i], got)
				}
			}
		})
	}
}
