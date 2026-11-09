// Package designer 提供「从 T100 设计器的站点清单导入环境」的能力。
//
// 背景:TT 的环境配置(hosts.sshs[])在设计上就是**模仿 T100 设计器的站台管理**做的 ——
// 主机、端口、账号、口令、登录区域、工作区这些字段两边同义。初次装 TT 的机器上大概率
// 已经装了设计器,所以这里把设计器那份清单读出来,让用户勾选导入,省掉逐字段手抄。
//
// # 设计器的站点文件在哪、怎么加密的
//
// 依据反编译源码 designer-src/SpecDesignerCommon/site/SiteFileHelper.cs:
//
//   - 路径:GetSettingDir() = %APPDATA%\SpecDesigner,s文件名 sites.xml
//     (Environment.SpecialFolder.ApplicationData + "SpecDesigner")
//   - 内容:整份 XML 用 AES 加密后 Base64 落盘
//   - 密钥:Rfc2898DeriveBytes("T100 SiteManager.SiteFile", "T100 SpecDesigner")
//     —— 即 PBKDF2-SHA1、默认 1000 次迭代、48 字节;前 32 字节作 Key、后 16 字节作 IV
//   - AesManaged 取 LegalKeySizes[0].MaxSize / LegalBlockSizes[0].MaxSize,
//     Windows 上是 256 位密钥 + 128 位块 ⇒ AES-256-CBC + PKCS#7 填充
//
// ⚠ 那个 `entropy` 字段(25 字节,构造函数里随机生成)**从未参与加解密**,是烟幕:
// 加解密两边都只用了上面那对硬编码的口令/盐。所以"加密"实际只是混淆 —— 任何能读到
// 程序集的人都能解开。这也意味着**解出来的口令是明文**(实测设计器自己就存明文)。
//
// # 本包的纪律
//
//   - **只读。** 绝不写回 sites.xml —— 那是设计器的文件,我们不拥有它。
//   - 解析失败/文件不存在都给**结构化原因**,不静默当成"这台机器没装设计器"
//     (那样用户看到空列表,会以为是"确实没有环境",而实际可能是文件坏了)。
package designer

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/crypto/pbkdf2"
	"tt/internal/config"
)

// 加解密参数(来自 SiteFileHelper.cs,见包注释)
const (
	passphrase = "T100 SiteManager.SiteFile"
	salt       = "T100 SpecDesigner"
	iterations = 1000 // Rfc2898DeriveBytes 的默认值
	keyLen     = 32   // AES-256
	ivLen      = 16   // AES 块大小
)

// EnvOverride 是"站点文件在哪"的环境变量覆盖(测试与排障用)。
//
// 存在的理由:自动化测试不能去动用户真实的 %APPDATA%\SpecDesigner —— 那既是别人的
// 文件,也让测试结果依赖这台机器装没装设计器。设了它就只认这个路径。
const EnvOverride = "TT_DESIGNER_SITES"

// SiteFile 返回设计器站点文件的路径与它是否存在的判断依据。
//
// 注意 AppData 取不到时(极少数环境)返回空路径,调用方据此报"定位不到"。
func SiteFile() string {
	if p := strings.TrimSpace(os.Getenv(EnvOverride)); p != "" {
		return p
	}
	appData := strings.TrimSpace(os.Getenv("APPDATA"))
	if appData == "" {
		// 非 Windows 或环境异常:退回 os.UserConfigDir(%AppData% 在 Windows 上就是它)
		if d, err := os.UserConfigDir(); err == nil {
			appData = d
		}
	}
	if appData == "" {
		return ""
	}
	return filepath.Join(appData, "SpecDesigner", "sites.xml")
}

// Discovered 是导入弹窗要的一份数据:解析出来的环境 + 这份数据从哪来 + 没导进来的原因。
type Discovered struct {
	// SitesFile 实际读的那个文件(绝对路径;定位不到时为空)。
	SitesFile string `json:"sitesFile"`
	// Found 文件是否存在。false 时 Envs 必为空,Reason 说明判断依据。
	Found bool `json:"found"`
	// Envs 解析出来的环境,已按设计器里的 order 排好(文件夹按其中最小 order 参与排序)。
	Envs []DiscoveredEnv `json:"envs"`
	// Reason 人类可读的说明:文件不存在、解密失败、XML 坏了、一条都没解析出来……
	// 为空 = 一切正常。**不用 error 表达**:这是"查询结果",不是失败 ——
	// 前端要把它和 Envs 一起展示(例如"没找到设计器"是个正常结论,不是错误弹窗)。
	Reason string `json:"reason,omitempty"`
}

// DiscoveredEnv 是设计器里的一条站点,按 TT 的字段命名(+ 几个原样带出的设计器字段)。
type DiscoveredEnv struct {
	// Name 设计器里的站点名(Setting@name),直接用作 TT 的环境名。
	Name string `json:"name"`
	// UID 设计器给这条记录的唯一 id(order/去重排障时有用,TT 侧不存)。
	UID string `json:"uid,omitempty"`
	// Folder 所属文件夹名(不在文件夹里时为空)。仅用于展示分组来源。
	Folder string `json:"folder,omitempty"`
	// Protocol 原样带出(只导入 ssh 的,其它值会被跳过但不静默)。
	Protocol string `json:"protocol,omitempty"`
	// 以下与 TT 的 NamedSsh 同义:
	Host      string `json:"host"`
	Port      int    `json:"port"`
	User      string `json:"user"`
	Password  string `json:"password"`
	Zone      string `json:"zone,omitempty"`      // ← 设计器的 Area(登录区域代码)
	Workspace string `json:"workspace,omitempty"` // ← 设计器的 Workspace(直接可用)
	// ServiceName 设计器的 Zone 字段。**它不是区域**,是服务名(T100 侧对应 db.service)。
	// 单独带出来给用户参考:TT 那边它在 hosts.sshs[].db.service 里。
	ServiceName string `json:"serviceName,omitempty"`
	// Skipped 非空 = 这条没有导入价值,附上原因(如协议不是 ssh)。
	// 仍然列出来:静默少一条比列出来更让人困惑。
	Skipped string `json:"skipped,omitempty"`
}

// ---- XML 形状(只声明我们用的部分) ----

type xmlSettings struct {
	XMLName  xml.Name     `xml:"Settings"`
	Settings []xmlSetting `xml:"Setting"`
	Folders  []xmlFolder  `xml:"Folder"`
}

type xmlFolder struct {
	Name     string       `xml:"name,attr"`
	Order    int          `xml:"order,attr"`
	Settings []xmlSetting `xml:"Setting"`
	Folders  []xmlFolder  `xml:"Folder"` // 设计器允许嵌套,递归处理
}

type xmlSetting struct {
	Name       string        `xml:"name,attr"`
	UID        string        `xml:"uid,attr"`
	Order      int           `xml:"order,attr"`
	Connection xmlConnection `xml:"Connection"`
}

type xmlConnection struct {
	Version        string `xml:"Version"`
	Protocol       string `xml:"Protocol"`
	IP             string `xml:"IP"`
	Port           string `xml:"Port"`
	Login          string `xml:"Login"`
	Password       string `xml:"Password"`
	Area           string `xml:"Area"`
	Workspace      string `xml:"Workspace"`
	LoginPrompt    string `xml:"LoginPrompt"`
	PasswordPrompt string `xml:"PasswordPrompt"`
	AreaPrompt     string `xml:"AreaPrompt"`
	UpdateURL      string `xml:"UpdateURL"`
	Zone           string `xml:"Zone"`
	ExtraCommand   string `xml:"ExtraCommand"`
}

// Discover 读并解析设计器的站点清单。
//
// 任何"读不到"都通过 Discovered.Reason 表达,error 只留给**调用方用法错**(目前没有)。
// 这样调用方只有一条展示路径,不必区分"错误弹窗"与"空结果提示"两种 UI。
func Discover() Discovered {
	path := SiteFile()
	out := Discovered{SitesFile: path}
	if path == "" {
		out.Reason = "定位不到 %APPDATA%\\SpecDesigner(读不到用户配置目录)"
		return out
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			out.Reason = "这台机器上没有找到设计器的站点清单(未安装设计器,或从未保存过站点)"
			return out
		}
		out.Reason = fmt.Sprintf("读取站点清单失败: %v", err)
		return out
	}
	out.Found = true

	xmlText, err := Decrypt(string(raw))
	if err != nil {
		out.Reason = fmt.Sprintf("站点清单解不开(%v);文件可能来自更新的设计器版本", err)
		return out
	}

	var parsed xmlSettings
	if err := xml.Unmarshal([]byte(xmlText), &parsed); err != nil {
		out.Reason = fmt.Sprintf("站点清单解开了但 XML 解析失败: %v", err)
		return out
	}

	out.Envs = collect(parsed)
	if len(out.Envs) == 0 {
		out.Reason = "站点清单里没有任何站点"
	}
	return out
}

// collect 把「顶层 Setting + 各层 Folder 里的 Setting」拉平成一个有序列表。
//
// 排序:设计器自己用 order 属性决定显示顺序,且**文件夹与站点共用一个 order 空间**
// (实测:4 个顶层站点 order 0..3,一个文件夹 order 4)。所以这里也按 order 排,
// 文件夹用它自己那个 order —— 展示顺序与设计器里看到的一致。
func collect(s xmlSettings) []DiscoveredEnv {
	type entry struct {
		order int
		env   DiscoveredEnv
	}
	var entries []entry
	var walk func(folder string, order int, sets []xmlSetting, folders []xmlFolder)
	walk = func(folder string, order int, sets []xmlSetting, folders []xmlFolder) {
		for _, s := range sets {
			o := s.Order
			if folder != "" {
				// 文件夹内的相对 order 不参与全局排序:用文件夹的 order,
				// 否则文件夹里的 0 会跑到所有顶层站点前面去。
				o = order
			}
			entries = append(entries, entry{order: o, env: toEnv(s, folder)})
		}
		for _, f := range folders {
			walk(f.Name, f.Order, f.Settings, f.Folders)
		}
	}
	walk("", 0, s.Settings, s.Folders)

	// 稳定排序:order 相同时保持文件里的出现顺序
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].order < entries[j].order })
	out := make([]DiscoveredEnv, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.env)
	}
	return out
}

// toEnv 把一条设计器站点翻成 TT 的字段。
func toEnv(s xmlSetting, folder string) DiscoveredEnv {
	c := s.Connection
	e := DiscoveredEnv{
		Name:        strings.TrimSpace(s.Name),
		UID:         strings.TrimSpace(s.UID),
		Folder:      folder,
		Protocol:    strings.TrimSpace(c.Protocol),
		Host:        strings.TrimSpace(c.IP),
		User:        strings.TrimSpace(c.Login),
		Password:    c.Password, // 口令不 Trim:它可能真的有前后空格
		Zone:        strings.TrimSpace(c.Area),
		Workspace:   strings.TrimSpace(c.Workspace),
		ServiceName: strings.TrimSpace(c.Zone),
	}
	// 端口是文本,解析失败就按缺省 22(TT 侧也这么兜底)
	if n, err := strconv.Atoi(strings.TrimSpace(c.Port)); err == nil && n > 0 {
		e.Port = n
	} else {
		e.Port = 22
	}
	if e.Name == "" {
		e.Name = e.Host
	}

	// 只导入 ssh 的站点:其它协议(如本地目录)在 TT 这边没有对应连接方式。
	// 不跳过整条、而是标记出来,让用户在列表里看到"这条为什么不能选"。
	if p := strings.ToLower(e.Protocol); p != "" && p != "ssh" {
		e.Skipped = fmt.Sprintf("协议是 %s,不是 ssh —— TT 的环境只能走 SSH", e.Protocol)
	} else if e.Host == "" {
		e.Skipped = "没有填 IP 主机"
	}
	return e
}

// Decrypt 解开设计器站点文件的内容(Base64 密文 → XML 文本)。
//
// 参数推导见包注释;参数不对时返回错误而不是空串 —— 空串会让"解不开"和"文件是空的"
// 混在一起,而这两件事的处置完全不同。
func Decrypt(b64 string) (string, error) {
	ct, err := base64.StdEncoding.DecodeString(strings.TrimSpace(b64))
	if err != nil {
		return "", fmt.Errorf("不是合法的 Base64")
	}
	if len(ct) == 0 || len(ct)%aes.BlockSize != 0 {
		return "", fmt.Errorf("密文长度 %d 不是 AES 块大小(%d)的整数倍", len(ct), aes.BlockSize)
	}
	dk := pbkdf2.Key([]byte(passphrase), []byte(salt), iterations, keyLen+ivLen, sha1.New)
	block, err := aes.NewCipher(dk[:keyLen])
	if err != nil {
		return "", err
	}
	pt := make([]byte, len(ct))
	cipher.NewCBCDecrypter(block, dk[keyLen:keyLen+ivLen]).CryptBlocks(pt, ct)

	// 去 PKCS#7 填充。填充不对 = 密钥/参数猜错了,要说清而不是给出乱码。
	n := int(pt[len(pt)-1])
	if n == 0 || n > aes.BlockSize || n > len(pt) {
		return "", fmt.Errorf("填充不合法(密钥或参数不匹配)")
	}
	for _, b := range pt[len(pt)-n:] {
		if int(b) != n {
			return "", fmt.Errorf("填充不合法(密钥或参数不匹配)")
		}
	}
	return string(pt[:len(pt)-n]), nil
}

// ToNamedSsh 把一条发现的环境翻成 TT 的配置结构(用于"添加进环境清单")。
//
// 只带 SSH 那一半:**db 留空** —— 设计器的站点清单里没有数据库配置(它只连 SSH 跑设计器),
// 所以数据库那一节必须由用户自己补(设置页「数据库」页签)。
func (e DiscoveredEnv) ToNamedSsh() config.NamedSsh {
	port := e.Port
	if port == 0 {
		port = 22
	}
	return config.NamedSsh{
		Name: e.Name,
		Zone: e.Zone,
		SSHConfig: config.SSHConfig{
			Host:     e.Host,
			Port:     port,
			User:     e.User,
			Password: e.Password,
		},
		Workspace: e.Workspace,
		// DB 刻意留空:nil = 尚未配置,设置页会显示"未配置"并让用户去填。
	}
}
