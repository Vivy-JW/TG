package designer

// 这组测试盯的是「能把设计器的站点清单读成 TT 的环境」这条线。
//
// 为什么值得测:这段代码读的是**另一个程序的加密文件**,而它的加密参数(口令/盐/迭代/
// 密钥长度)是我从反编译源码里推出来的 —— 一旦推错,**解不开**的表现是空列表或乱码,
// 而那看起来很像"这台机器没装设计器/设计器里没站点"。所以这里用手工构造的密文做
// 往返验证:加密与解密用同一套参数,只要参数被改坏,往返就会立刻失败。

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/pbkdf2"
)

// encryptForTest 用与 Decrypt 相同的参数加密 —— **独立实现一遍**而不是调生产代码:
// 若直接调生产加密函数,参数写错时两边一起错,往返照样"通过",测不出任何东西。
func encryptForTest(t *testing.T, plain string) string {
	t.Helper()
	dk := pbkdf2.Key([]byte(passphrase), []byte(salt), iterations, keyLen+ivLen, sha1.New)
	block, err := aes.NewCipher(dk[:keyLen])
	if err != nil {
		t.Fatal(err)
	}
	pt := []byte(plain)
	// PKCS#7 填充
	n := aes.BlockSize - len(pt)%aes.BlockSize
	pt = append(pt, []byte(strings.Repeat(string(rune(n)), n))...)
	ct := make([]byte, len(pt))
	cipher.NewCBCEncrypter(block, dk[keyLen:keyLen+ivLen]).CryptBlocks(ct, pt)
	return base64.StdEncoding.EncodeToString(ct)
}

const sampleXML = `<Settings>
  <Setting name="109tst" uid="uid-1" order="0">
    <Connection>
      <Version>1.0</Version>
      <Protocol>ssh</Protocol>
      <IP>172.16.1.109</IP>
      <Port>22</Port>
      <Login>tiptop</Login>
      <Password>secret109</Password>
      <Area>35</Area>
      <Workspace>D:\t100_wrok_dir\109\35\</Workspace>
      <LoginPrompt>Login:</LoginPrompt>
      <PasswordPrompt>Password:</PasswordPrompt>
      <AreaPrompt>(*)Exit</AreaPrompt>
      <UpdateURL>http://172.16.1.109/topprd/clitool/designer/</UpdateURL>
      <Zone>topprd</Zone>
      <ExtraCommand></ExtraCommand>
    </Connection>
  </Setting>
  <Setting name="恒烁" uid="uid-2" order="1">
    <Connection>
      <Protocol>ssh</Protocol>
      <IP>172.19.93.55</IP>
      <Port>22</Port>
      <Login>tiptop</Login>
      <Password> 带空格的口令  </Password>
      <Area>1</Area>
      <Workspace>D:\t100_wrok_dir\hengshuo\prd\</Workspace>
      <Zone>topprd</Zone>
    </Connection>
  </Setting>
  <Folder name="新客户" order="4">
    <Setting name="尚阳通" uid="uid-3" order="0">
      <Connection>
        <Protocol>ssh</Protocol>
        <IP>10.13.0.120</IP>
        <Port>22</Port>
        <Login>tiptop</Login>
        <Password>p3</Password>
        <Area>1</Area>
        <Workspace>D:\t100_wrok_dir\shangyangtong\</Workspace>
        <Zone>topprd</Zone>
      </Connection>
    </Setting>
  </Folder>
  <Setting name="本地目录" uid="uid-4" order="5">
    <Connection>
      <Protocol>file</Protocol>
      <IP></IP>
      <Login></Login>
      <Password></Password>
    </Connection>
  </Setting>
</Settings>`

// writeSites 把加密后的站点文件写到临时目录,并让 SiteFile() 指向它。
func writeSites(t *testing.T, plain string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "sites.xml")
	if err := os.WriteFile(p, []byte(encryptForTest(t, plain)), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvOverride, p)
	return p
}

// 往返:能解开自己加密的内容。
func TestDecryptRoundTrip(t *testing.T) {
	for _, in := range []string{
		"<Settings />",
		sampleXML,
		strings.Repeat("很长的一段内容 ", 500), // 跨多个块,验证填充
	} {
		got, err := Decrypt(encryptForTest(t, in))
		if err != nil {
			t.Fatalf("解密失败: %v", err)
		}
		if got != in {
			t.Fatalf("往返不一致: 得到 %d 字节,期望 %d 字节", len(got), len(in))
		}
	}
}

// 参数不对时必须**报错**,而不是返回空串或乱码 ——
// 空串会被上层读成"文件是空的",乱码会被读成"XML 坏了",两者的排查方向都不对。
func TestDecryptRejectsGarbage(t *testing.T) {
	cases := []struct{ name, in string }{
		{"不是 Base64", "这显然不是 base64!!!"},
		{"合法 Base64 但长度不是块大小倍数", base64.StdEncoding.EncodeToString([]byte("short"))},
		{"空", ""},
		{"长度对但内容是随机字节", base64.StdEncoding.EncodeToString(make([]byte, 32))},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got, err := Decrypt(c.in); err == nil {
				t.Fatalf("该报错却成功了(得到 %q)", got)
			}
		})
	}
}

// 解析:字段映射、文件夹递归、order 排序、非 ssh 的标记。
func TestDiscoverParsesAndMaps(t *testing.T) {
	writeSites(t, sampleXML)

	d := Discover()
	if !d.Found {
		t.Fatalf("应当 Found=true,Reason=%q", d.Reason)
	}
	if len(d.Envs) != 4 {
		t.Fatalf("应当解析出 4 条,得到 %d 条: %+v", len(d.Envs), d.Envs)
	}

	// 顺序:顶层 order 0、1,文件夹 order 4,最后 order 5。
	// **文件夹里的 order 0 不能跑到最前面** —— 那是相对 order。
	wantOrder := []string{"109tst", "恒烁", "尚阳通", "本地目录"}
	for i, want := range wantOrder {
		if d.Envs[i].Name != want {
			t.Errorf("第 %d 条应为 %q,得到 %q", i, want, d.Envs[i].Name)
		}
	}

	// 字段映射:IP→Host、Login→User、Area→Zone、Workspace→Workspace
	first := d.Envs[0]
	if first.Host != "172.16.1.109" || first.Port != 22 || first.User != "tiptop" {
		t.Errorf("连接字段映射不对: %+v", first)
	}
	if first.Zone != "35" {
		t.Errorf("Area 应映射到 Zone(登录区域),得到 %q", first.Zone)
	}
	if first.Workspace != `D:\t100_wrok_dir\109\35\` {
		t.Errorf("Workspace 未带出: %q", first.Workspace)
	}
	if first.ServiceName != "topprd" {
		t.Errorf("设计器的 Zone 应作为服务名带出,得到 %q", first.ServiceName)
	}
	if first.UID != "uid-1" {
		t.Errorf("UID 未带出: %q", first.UID)
	}
	if first.Folder != "" {
		t.Errorf("顶层站点不该有文件夹,得到 %q", first.Folder)
	}

	// 文件夹里的那条要标出所属文件夹
	if d.Envs[2].Folder != "新客户" {
		t.Errorf("文件夹里的站点应带文件夹名,得到 %q", d.Envs[2].Folder)
	}

	// **口令不 Trim**:它可能真的有前后空格,吃掉就登不上
	if d.Envs[1].Password != " 带空格的口令  " {
		t.Errorf("口令被改动了: %q", d.Envs[1].Password)
	}

	// 非 ssh 的仍然列出来,但标记 Skipped(静默少一条更让人困惑)
	last := d.Envs[3]
	if last.Skipped == "" {
		t.Error("非 ssh 的站点应当被标记为跳过,而不是当作可用环境")
	}
	if !strings.Contains(last.Skipped, "file") {
		t.Errorf("跳过原因该点出协议名: %q", last.Skipped)
	}
}

// ToNamedSsh:端口缺省 22,db 留空(设计器里没有数据库配置)。
func TestToNamedSshLeavesDBEmpty(t *testing.T) {
	e := DiscoveredEnv{Name: "X", Host: "h", User: "u", Password: "p", Zone: "36", Workspace: `D:\ws`}
	got := e.ToNamedSsh()
	if got.Name != "X" || got.Host != "h" || got.Port != 22 || got.User != "u" || got.Password != "p" {
		t.Fatalf("字段不对: %+v", got)
	}
	if got.Zone != "36" || got.Workspace != `D:\ws` {
		t.Fatalf("区域/工作区不对: %+v", got)
	}
	// 关键:**DB 必须是 nil** —— 交给设置页显示"未配置",让用户自己补。
	// 若这里塞一个空 Connection,前端会当成"已配置"而显示一个空白的数据库页。
	if got.DB != nil {
		t.Error("从设计器导入的环境不该带数据库配置(设计器里没有这一节)")
	}
}

// 文件不存在 / 路径定位不到:Found=false 且 Reason 说清原因,Envs 为空。
func TestDiscoverMissingFile(t *testing.T) {
	t.Setenv(EnvOverride, filepath.Join(t.TempDir(), "不存在.xml"))
	d := Discover()
	if d.Found {
		t.Error("文件不存在时 Found 应为 false")
	}
	if len(d.Envs) != 0 {
		t.Error("文件不存在时不该有环境")
	}
	if d.Reason == "" {
		t.Error("必须给出原因(否则用户看到空列表会以为'确实没有站点')")
	}
}

// 文件存在但解不开:Found=true(文件确实在)、Reason 指向解密失败。
// 这条与上一条的区别很重要:一个是"没装设计器",一个是"装了但读不了"。
func TestDiscoverUndecryptable(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "sites.xml")
	if err := os.WriteFile(p, []byte("bm90IGEgY2lwaGVy"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvOverride, p)

	d := Discover()
	if !d.Found {
		t.Error("文件在,Found 应为 true")
	}
	if d.Reason == "" || len(d.Envs) != 0 {
		t.Fatalf("解不开时应给出原因且无环境: reason=%q envs=%d", d.Reason, len(d.Envs))
	}
}

// 解开了但没有站点:这是正常结论,Reason 说明白。
func TestDiscoverEmptySettings(t *testing.T) {
	writeSites(t, "<Settings />")
	d := Discover()
	if !d.Found {
		t.Fatal("应当 Found=true")
	}
	if len(d.Envs) != 0 || d.Reason == "" {
		t.Fatalf("空清单应有无环境 + 原因: envs=%d reason=%q", len(d.Envs), d.Reason)
	}
}

// 站点没写 name 时退回主机名(否则会往配置里写一个空名字的环境)。
func TestDiscoverFallsBackToHostAsName(t *testing.T) {
	writeSites(t, `<Settings><Setting uid="u" order="0"><Connection>
	  <Protocol>ssh</Protocol><IP>1.2.3.4</IP><Port>22</Port>
	  <Login>u</Login><Password>p</Password><Area>1</Area>
	</Connection></Setting></Settings>`)
	d := Discover()
	if len(d.Envs) != 1 {
		t.Fatalf("应有 1 条,得到 %d", len(d.Envs))
	}
	if d.Envs[0].Name != "1.2.3.4" {
		t.Errorf("没名字应退回主机名,得到 %q", d.Envs[0].Name)
	}
}
