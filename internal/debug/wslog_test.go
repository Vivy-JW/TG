package debug

import (
	"strings"
	"testing"
)

// wslogWhere 的表驱动单测:条件拼装是拼 SQL 的敏感环节(防注入),
// 同时钉死"对齐 awsq990 原生口径"的三件事:排除 docno.storage / 等值过滤 / 时间窗字段归属。
func TestWSLogWhere(t *testing.T) {
	cases := []struct {
		name string
		f    WSLogFilter
		want []string // 必须出现的片段(AND 分隔的顺序无关)
		not  []string // 必须不出现的片段
	}{
		{
			name: "空条件:只有基线(排除 docno.storage)",
			f:    WSLogFilter{},
			want: []string{"1=1", "wsfa001 != 'docno.storage'"},
			not:  []string{"1=0", "wsfa006", "wsfa013", "wsfa018", "wsfa003", "wsfa004"},
		},
		{
			name: "服务名:通配符翻译为 LIKE 且大写",
			f:    WSLogFilter{Service: "icd.erp.wo*"},
			want: []string{"UPPER(wsfa001) LIKE 'ICD.ERP.WO%'"},
		},
		{
			name: "服务名:? 翻译为 _",
			f:    WSLogFilter{Service: "wssp90?"},
			want: []string{"UPPER(wsfa001) LIKE 'WSSP90_'"},
		},
		{
			name: "服务名:非法字符退化为空集(不报错)",
			f:    WSLogFilter{Service: "a'; DROP"},
			want: []string{"AND 1=0"},
			not:  []string{"DROP"},
		},
		{
			name: "处理结果/发起端/服务端:三个等值条件",
			f:    WSLogFilter{Result: "000", Origin: "OA", Server: "T100"},
			want: []string{"wsfa006 = '000'", "wsfa013 = 'OA'", "wsfa018 = 'T100'"},
		},
		{
			name: "等值条件:值含空格/中文照常引用",
			f:    WSLogFilter{Server: "T100 正式区"},
			want: []string{"wsfa018 = 'T100 正式区'"},
		},
		{
			name: "等值条件:单引号是唯一逃逸手段,出现即退化为空集",
			f:    WSLogFilter{Origin: "OA' OR 1=1--"},
			want: []string{"AND 1=0"},
			not:  []string{"OR 1=1"},
		},
		{
			name: "等值条件:前后空白被剔除",
			f:    WSLogFilter{Result: "  000  "},
			want: []string{"wsfa006 = '000'"},
		},
		{
			// 界面「服务程序」列就是 wsfa002。用它 + 服务名可精确定位某一次调用,
			// 也是 rowid/ctid 失效(日志表 purge 或重组)时的回退定位手段。
			name: "服务程序序号:按 wsfa002 等值",
			f:    WSLogFilter{Service: "wssp01131", PID: "861637"},
			want: []string{"UPPER(wsfa001) LIKE 'WSSP01131'", "wsfa002 = '861637'"},
		},
		{
			name: "服务程序序号:带单引号退化为空集(与其它等值条件同一白名单)",
			f:    WSLogFilter{PID: "1' OR '1'='1"},
			want: []string{"AND 1=0"},
			not:  []string{"OR '1'"},
		},
		{
			// wsfa012 作业编号。按"哪个作业"找日志的唯一入口 ——
			// 服务名是 oa.schema.data.get 这类反域名,光看服务名认不出是什么作业。
			name: "作业编号:通配符翻译为 LIKE 且大写",
			f:    WSLogFilter{Job: "wssp*"},
			want: []string{"UPPER(wsfa012) LIKE 'WSSP%'"},
			not:  []string{"UPPER(wsfa001) LIKE"},
		},
		{
			name: "作业编号:精确值退化为等价的 LIKE(与原生 = 结果一致)",
			f:    WSLogFilter{Job: "wssp01131"},
			want: []string{"UPPER(wsfa012) LIKE 'WSSP01131'"},
		},
		{
			name: "作业编号:? 翻译为 _",
			f:    WSLogFilter{Job: "wssp0113?"},
			want: []string{"UPPER(wsfa012) LIKE 'WSSP0113_'"},
		},
		{
			name: "作业编号:非法字符退化为空集(与 service 同一白名单)",
			f:    WSLogFilter{Job: "a'; DROP"},
			want: []string{"AND 1=0"},
			not:  []string{"DROP"},
		},
		{
			name: "作业编号与服务名可叠加(AND)",
			f:    WSLogFilter{Service: "oa.schema.data.get", Job: "wssp01131"},
			want: []string{"UPPER(wsfa001) LIKE 'OA.SCHEMA.DATA.GET'", "UPPER(wsfa012) LIKE 'WSSP01131'"},
		},
		{
			name: "仅失败:本工具扩展",
			f:    WSLogFilter{OnlyFail: true},
			want: []string{"wsfa006 <> '000'"},
		},
		{
			name: "时间窗:下界过滤 wsfa003、上界过滤 wsfa004 且纯日期补到当天末尾",
			f:    WSLogFilter{StartFrom: "2026-09-11", EndTo: "2026-09-12"},
			want: []string{"wsfa003 >= '2026-09-11'", "wsfa004 <= '2026-09-12 23:59:59.99999'"},
			not:  []string{"wsfa003 <= ", "wsfa004 >= "},
		},
		{
			name: "时间窗:带时间戳的上界原样使用(不补后缀)",
			f:    WSLogFilter{EndTo: "2026-09-12 12:00:00"},
			want: []string{"wsfa004 <= '2026-09-12 12:00:00'"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := wslogWhere(c.f)
			if err != nil {
				t.Fatalf("不该报错: %v", err)
			}
			for _, w := range c.want {
				if !strings.Contains(got, w) {
					t.Errorf("缺少片段 %q\n实际: %s", w, got)
				}
			}
			for _, n := range c.not {
				if strings.Contains(got, n) {
					t.Errorf("不该出现片段 %q\n实际: %s", n, got)
				}
			}
		})
	}
}

// 时间格式非法仍要报错(与原接口行为一致,由调用方回 500)
func TestWSLogWhereBadTime(t *testing.T) {
	for _, bad := range []string{"2026/09/12", "2026-09-12; drop", "abcdefghij"} {
		if _, err := wslogWhere(WSLogFilter{StartFrom: bad}); err == nil {
			t.Errorf("起始时间 %q 应报格式非法", bad)
		}
		if _, err := wslogWhere(WSLogFilter{EndTo: bad}); err == nil {
			t.Errorf("结束时间 %q 应报格式非法", bad)
		}
	}
}

// 界面回传的"改过的入参"校验:空 = 用原报文(放行);超上限拒绝
func TestCheckReplayOverride(t *testing.T) {
	if err := checkReplayOverride(""); err != nil {
		t.Fatalf("空入参表示用原报文,不该报错: %v", err)
	}
	ok := strings.Repeat("x", maxReplayPayload)
	if err := checkReplayOverride(ok); err != nil {
		t.Fatalf("%d 字节(正好上限)应放行: %v", len(ok), err)
	}
	over := strings.Repeat("x", maxReplayPayload+1)
	err := checkReplayOverride(over)
	if err == nil {
		t.Fatalf("%d 字节(超一字节)应被拒绝", len(over))
	}
	if !strings.Contains(err.Error(), "256KB") {
		t.Errorf("错误信息应点明上限,实际: %v", err)
	}
}

// 每页条数/页码的钳位:非法值回落到默认,不因调用方传参崩掉
func TestWSLogPageClamp(t *testing.T) {
	// 只验证钳位分支本身(不触网):越界值与默认值的期望
	for _, c := range []struct{ in, want int }{{0, 200}, {-1, 200}, {501, 200}, {50, 50}, {500, 500}} {
		size := c.in
		if size <= 0 || size > 500 {
			size = 200
		}
		if size != c.want {
			t.Errorf("PageSize %d 应钳为 %d,得到 %d", c.in, c.want, size)
		}
	}
}

// wsfa_t 是 ds 下的共享表,查接口日志恒用系统账号 —— 这个"为什么"必须随结果带出去,
// 否则"日志为空"会被读成"这段时间没有调用",而不是"账号/企业可能用错了"。
func TestWSLogAcctFor(t *testing.T) {
	cases := []struct {
		name       string
		topent     string
		wantWarn   bool // 期望带 TOPENT 警告
		wantInWarn string
	}{
		{name: "有效企业编号", topent: "99"},
		{name: "据点码填错位置", topent: "SITE01", wantWarn: true, wantInWarn: "SITE01"},
		{name: "空", topent: "", wantWarn: true, wantInWarn: "没有生效的 TOPENT"},
		{name: "零", topent: "0", wantWarn: true, wantInWarn: "0"},
		{name: "带空白", topent: "  99  "},
	}
	for _, c := range cases {
		a := wslogAcctFor(c.topent)
		if a.Account != wsSysAccount {
			t.Errorf("[%s] 账号应当恒为系统账号 %s,得到 %q", c.name, wsSysAccount, a.Account)
		}
		if a.Reason == "" {
			t.Errorf("[%s] 必须说明为什么用这个账号", c.name)
		}
		if (a.TopentWarning != "") != c.wantWarn {
			t.Errorf("[%s] TOPENT 警告 = %q, 期望有无=%v", c.name, a.TopentWarning, c.wantWarn)
		}
		if c.wantInWarn != "" && !strings.Contains(a.TopentWarning, c.wantInWarn) {
			t.Errorf("[%s] 警告里应当点出原始值 %q: %q", c.name, c.wantInWarn, a.TopentWarning)
		}
	}
}

// parseWSLogChunks 的表驱动单测。分块读 CLOB 的关键不在"读多少",而在**块边界**:
// sqlplus 会裁行尾空格、并在每条语句后多打一个空行,所以每块的值都以 STX 结尾;
// 解析必须去掉哨兵、保留块内换行与块尾空格,块数还要能对上。
func TestParseWSLogChunks(t *testing.T) {
	cases := []struct {
		name             string
		out              string
		wantReq, wantRsp string
		wantReqN         int
	}{
		{
			name:     "单块",
			out:      "@@TTCHUNK:req:0@@\n{\n  \"a\": 1\n}\x02\n\n",
			wantReq:  "{\n  \"a\": 1\n}",
			wantReqN: 1,
		},
		{
			name:     "两块在同一行内续接:块边界不引入换行",
			out:      "@@TTCHUNK:req:0@@\nabc\x02\n\n@@TTCHUNK:req:1@@\ndef\x02\n\n",
			wantReq:  "abcdef",
			wantReqN: 2,
		},
		{
			name:     "块本身以换行结尾:保留报文自己的换行",
			out:      "@@TTCHUNK:req:0@@\na\n\x02\n\n@@TTCHUNK:req:1@@\nb\x02\n\n",
			wantReq:  "a\nb",
			wantReqN: 2,
		},
		{
			name:     "块边界落在缩进中间:块尾空格不能被吃掉",
			out:      "@@TTCHUNK:req:0@@\n{\n    \x02\n\n@@TTCHUNK:req:1@@\n    \"a\": 1\n}\x02\n\n",
			wantReq:  "{\n        \"a\": 1\n}",
			wantReqN: 2,
		},
		{
			name:     "请求/响应分开,各自多块",
			out:      "@@TTCHUNK:req:0@@\n<re\x02\n\n@@TTCHUNK:req:1@@\nq/>\x02\n\n@@TTCHUNK:rsp:0@@\n<rs\x02\n\n@@TTCHUNK:rsp:1@@\np/>\x02\n\n",
			wantReq:  "<req/>",
			wantRsp:  "<rsp/>",
			wantReqN: 2,
		},
		{
			name:     "第一个块之前是 sqlplus 噪声:一律不算内容",
			out:      "set echo off\n\n@@TTCHUNK:req:0@@\nX\x02\n\n",
			wantReq:  "X",
			wantReqN: 1,
		},
		{
			name:     "CRLF 行尾",
			out:      "@@TTCHUNK:req:0@@\r\nA\r\nB\x02\r\n\r\n",
			wantReq:  "A\nB",
			wantReqN: 1,
		},
		{
			name:     "空输出",
			out:      "",
			wantReq:  "",
			wantRsp:  "",
			wantReqN: 0,
		},
	}
	for _, c := range cases {
		req, rsp, reqN, rspN := parseWSLogChunks(c.out)
		if req != c.wantReq {
			t.Errorf("[%s] 请求 = %q, 期望 %q", c.name, req, c.wantReq)
		}
		if rsp != c.wantRsp {
			t.Errorf("[%s] 响应 = %q, 期望 %q", c.name, rsp, c.wantRsp)
		}
		if reqN != c.wantReqN {
			t.Errorf("[%s] 请求块数 = %d, 期望 %d", c.name, reqN, c.wantReqN)
		}
		if c.wantRsp == "" && rspN != 0 {
			t.Errorf("[%s] 响应块数 = %d, 期望 0", c.name, rspN)
		}
	}
}

// wslogChunkCount:块数要能整除向上取整,调用方用它判断"有没有块没读到"。
func TestWSLogChunkCount(t *testing.T) {
	cases := []struct{ n, want int }{{0, 0}, {1, 1}, {2000, 1}, {2001, 2}, {3967, 2}, {4000, 2}, {4001, 3}}
	for _, c := range cases {
		if got := wslogChunkCount(c.n); got != c.want {
			t.Errorf("wslogChunkCount(%d) = %d, 期望 %d", c.n, got, c.want)
		}
	}
}

// parseWSLogClobLengths:只认 "数字|数字" 那一行,sqlplus 的回显/空行/报错都不该被当成长度。
func TestParseWSLogClobLengths(t *testing.T) {
	cases := []struct {
		name    string
		out     string
		wantReq int
		wantRsp int
	}{
		{"正常", "set heading off\n4007|1035\n", 4007, 1035},
		{"前面有空行", "\n\n3967|719\n", 3967, 719},
		{"没有入库", "0|0\n", 0, 0},
		{"报错行", "ORA-00942: table or view does not exist\n", 0, 0},
		{"空输出", "", 0, 0},
	}
	for _, c := range cases {
		req, rsp := parseWSLogClobLengths(c.out)
		if req != c.wantReq || rsp != c.wantRsp {
			t.Errorf("[%s] 得到 %d|%d, 期望 %d|%d", c.name, req, rsp, c.wantReq, c.wantRsp)
		}
	}
}

// 数据库报错行不能只认 ORA-:金仓报的是 `ERROR: relation "wsfa_t" does not exist`,
// 漏掉它会让"表不存在"被静默当成"这段时间没有日志"。
// 反过来也要挡住:数据行里的报文字段本身可能带 ORA- 字样,那不该让整次查询失败。
func TestIsDBErrorLine(t *testing.T) {
	yes := []string{
		`ORA-00942: table or view does not exist`,
		`  SP2-0306: invalid option`, // 前导空白也要认
		`ERROR: relation "wsfa_t" does not exist`,
		`FATAL: password authentication failed`,
	}
	for _, ln := range yes {
		if !isDBErrorLine(ln) {
			t.Errorf("%q 应当被认成数据库报错", ln)
		}
	}
	no := []string{
		"",
		"AAJp0sAATAAFn8UAAA|wssp01131|1234|2026-09-17 21:00:00|0.1|Y|bsft001_wf",
		// 报文里出现了 ORA- 字样,但这是**数据行**,不是报错行
		"ctid|wssp01131|||||ORA-01403 在业务里被 catch 了|||||||",
		"12 rows selected.",
	}
	for _, ln := range no {
		if isDBErrorLine(ln) {
			t.Errorf("%q 不该被当成数据库报错(会让一次正常查询整批失败)", ln)
		}
	}
}
