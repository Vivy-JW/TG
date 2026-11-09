package tzs

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// TestParsePipeName 钉住 `--pipe-name` 输出的形状校验。
//
// 为什么必须校验：这条命令的成功退出码只说明 exe 跑起来了。拿 tt.exe、或者一个把
// usage 打到 stdout 的版本去问，我们会得到一个「名字」，然后花 60 秒等一个不存在的
// 管道就绪 —— 报出来的是「冷启动 60 s 未就绪」，与真实原因（问错了 exe）毫无关系。
//
// 校验只做形状，不做转换：名字原样拿去 CreateFile。大小写敏感是因为 **Windows 的
// 命名管道对象名区分大小写**，而引擎永远用 ToString("x8") 产出小写；
// 出现大写说明对我们说话的不是那个 exe。
func TestParsePipeName(t *testing.T) {
	const real = "tzs-cli-01a04826-11c46f9d" // 实测：tzs-server --pipe-name 的输出
	cases := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{"带换行", real + "\n", real, false},
		{"带 CRLF", real + "\r\n", real, false},
		{"带 BOM", bomString + real + "\n", real, false},
		{"带空白", "   " + real + "  \n", real, false},
		{"空输出", "", "", true},
		{"只有换行", "\n", "", true},
		{"usage 文本", "tzs-server -- T100 .tzs 设计器的长驻 JSON-RPC 服务\n", "", true},
		{"大写十六进制", "tzs-cli-01A04826-11C46F9D\n", "", true},
		{"前缀不对", "tzs-server-01a04826-11c46f9d\n", "", true},
		{"少一段", "tzs-cli-01a04826\n", "", true},
		{"多一段", real + "-extra\n", "", true},
		{"十六进制位数不对", "tzs-cli-1a2b3c4-89abcdef\n", "", true},
		{"非十六进制字符", "tzs-cli-01a0482g-11c46f9d\n", "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := parsePipeName([]byte(c.in))
			if c.wantErr {
				if err == nil {
					t.Fatalf("该被拒，却得到了 %q", got)
				}
				var te *TransportError
				if !errors.As(err, &te) {
					t.Fatalf("该是传输错误，得 %T", err)
				}
				if te.ExitCode() != ExitTransport {
					t.Errorf("该退 5，得 %d", te.ExitCode())
				}
				return
			}
			if err != nil {
				t.Fatalf("该能过：%v", err)
			}
			if got != c.want {
				t.Errorf("得 %q，想要 %q", got, c.want)
			}
		})
	}
}

// TestPipeNameRefusesWithoutWorkspace：问管道名时**必须**给工作区。
//
// 不给的话引擎会回落到它自己的缺省工作区（一个真实客户目录）的管道名，
// 我们就拿着那个名字去连别人的守护进程了。
//
// 这两条都在 exec 之前返回，所以默认测试里一次 exec 都不会发生。
func TestPipeNameRefusesWithoutWorkspace(t *testing.T) {
	_, err := PipeName(context.Background(), `C:\不存在\tzs-server.exe`, "   ")
	if err == nil {
		t.Fatal("没有工作区该被拒")
	}
	var ue *UsageError
	if !errors.As(err, &ue) {
		t.Fatalf("该是用法错（退 2），得 %T：%v", err, err)
	}
	if !strings.Contains(err.Error(), "工作区") {
		t.Errorf("文案该点名工作区：%v", err)
	}
}

// TestPipeNameRefusesWithoutExe：没有 exe 路径时也是环境失败（退 5），且不 exec。
func TestPipeNameRefusesWithoutExe(t *testing.T) {
	_, err := PipeName(context.Background(), "  ", `D:\ws`)
	if err == nil {
		t.Fatal("没有 exe 该被拒")
	}
	var te *TransportError
	if !errors.As(err, &te) {
		t.Fatalf("该是传输/环境失败（退 5），得 %T：%v", err, err)
	}
	if te.ExitCode() != ExitTransport {
		t.Errorf("该退 5，得 %d", te.ExitCode())
	}
}
