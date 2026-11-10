package dict

import "testing"

// TestProgBaseCode 子程序/子元件编号还原成主程序编号;普通程序编号与开窗码不动。
func TestProgBaseCode(t *testing.T) {
	cases := map[string]string{
		"aapq110_01":     "aapq110", // 子程序
		"aapq110_02":     "aapq110",
		"aapq110_x01":    "aapq110", // 查询列印元件
		"aapr110_k01":    "aapr110", // 凭单列印元件
		"aapp350_g01":    "aapp350", // 报表元件
		"aimi100_s01":    "aimi100", // 子画面
		"axmi125_wf":     "axmi125", // 流程
		"aapt110_01_rep": "aapt110", // 多级后缀
		"q_bxmi002_wf":   "q_bxmi002",
		// 以下不是子程序形态,应原样返回空串(不提示"主程序")
		"aapi011":   "",
		"aooi301":   "",
		"q_adzi052": "",
		"cl_abi":    "",
		"":          "",
		"_01":       "", // 只剩后缀:不还原成空主程序
	}
	for in, want := range cases {
		if got := progBaseCode(in); got != want {
			t.Errorf("progBaseCode(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestIsSubSuffix(t *testing.T) {
	yes := []string{"01", "12", "x01", "g02", "k01", "s01", "wf", "rep"}
	no := []string{"", "abc", "x", "adzi052", "bxmi002", "_01"}
	for _, s := range yes {
		if !isSubSuffix(s) {
			t.Errorf("isSubSuffix(%q) 应为 true", s)
		}
	}
	for _, s := range no {
		if isSubSuffix(s) {
			t.Errorf("isSubSuffix(%q) 应为 false", s)
		}
	}
}

// TestCategoryLabel 程序类别码的中文名（TOP GP 口径，来源 p_zz.sdd 的 zz03 定义）。
// ERP 里存的是大写（zz_file.zz03='I'），大小写都要能翻译出来。
func TestCategoryLabel(t *testing.T) {
	cases := map[string]string{
		"I": "建档维护", "i": "建档维护",
		"M": "目录程式", "Q": "查询", "R": "报表类", "P": "处理类",
		"T": "异动单据", "U": "公用程式", "S": "设定", "O": "其他",
		"": "", "Z": "",
	}
	for in, want := range cases {
		if got := categoryLabel(in); got != want {
			t.Errorf("categoryLabel(%q) = %q, want %q", in, got, want)
		}
	}
}
