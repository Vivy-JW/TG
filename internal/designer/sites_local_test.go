package designer

// 本机真实文件的冒烟检查:若这台机器真装了设计器且存过站点,就把生产代码跑一遍。
//
// 为什么要一条"依赖本机环境"的测试:上面那些用例喂的是**我自己造的**密文,加密与解密
// 共用同一套参数常量 —— 万一我把参数理解错了(PBKDF2 迭代次数、AesManaged 取到的是
// 256 位还是 128 位密钥),自造的数据照样能往返成功,而真实文件一个都解不开。
// 这条测试拿真正由 .NET 生成的密文来验,是唯一能证伪那类错误的东西。
//
// 没装设计器(或从未保存过站点)的机器上 **Skip**,不算失败 —— 它本来就是本机检查。

import (
	"os"
	"testing"
)

func TestDiscoverRealFileIfPresent(t *testing.T) {
	// 不设 EnvOverride:走真实的 %APPDATA%\SpecDesigner\sites.xml 定位逻辑
	p := os.Getenv("APPDATA")
	if p == "" {
		t.Skip("没有 APPDATA,跳过本机真实文件检查")
	}
	if _, err := os.Stat(SiteFile()); err != nil {
		t.Skip("这台机器没有设计器站点清单,跳过")
	}

	d := Discover()
	if !d.Found {
		t.Fatalf("文件在却 Found=false: %s", d.Reason)
	}
	// 关键断言:真实密文必须**解得开**。解不开说明参数推导错了(见文件头注释)。
	if d.Reason != "" && len(d.Envs) == 0 {
		t.Fatalf("真实站点文件没解析出任何环境: %s", d.Reason)
	}
	t.Logf("真实文件 %s:%d 个站点", d.SitesFile, len(d.Envs))
	for _, e := range d.Envs {
		if e.Skipped != "" {
			t.Logf("  - %s(跳过:%s)", e.Name, e.Skipped)
			continue
		}
		t.Logf("  - %s  %s@%s:%d zone=%s ws=%s", e.Name, e.User, e.Host, e.Port, e.Zone, e.Workspace)
		// 真实数据里这些字段必须都拿到了 —— 全都空说明元素名映射错了
		if e.Host == "" || e.User == "" {
			t.Errorf("%s: 主机/账号为空,元素名可能映射错了: %+v", e.Name, e)
		}
	}
}
