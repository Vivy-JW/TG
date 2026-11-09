package update

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tt/internal/config"
)

// Kind 是 tt 这份二进制的**安装形态**，它决定"怎么装新版"。
//
// 只按布局判定，不看版本号：注入的版本号为空是另一件事（源码态的 `make build` 产物），
// 由 InstallRefusal 一并说清 —— 一个函数判定，一处解释，避免两边各猜一半。
type Kind int

const (
	// KindOther 认不出来的布局：没有便携标记、也不在 MSI 的安装目录里。
	// 只允许 check。**不许**给它"随便解压一个便携包"的待遇：那会在用户的目录里
	// 凭空多出一棵 tt-portable/ 树。
	KindOther Kind = iota
	// KindPortable 便携包：exe 同目录有 config.PortableMark，整包解压即升级。
	KindPortable
	// KindMSI 安装器装的：exe 在 %LOCALAPPDATA%\Programs\TT（见 installer/tt.wxs
	// 的 INSTALLFOLDER），升级交给 msiexec（per-user，不需要管理员）。
	KindMSI
)

// 计划文件与日志用的稳定记号。**故意与 String() 分开**：String() 是给人读的中文，
// 改了不该让已经落盘的升级计划失效（那是跨进程契约）。
const (
	TokenPortable = "portable"
	TokenMSI      = "msi"
)

func (k Kind) String() string {
	switch k {
	case KindPortable:
		return "便携包"
	case KindMSI:
		return "MSI 安装"
	default:
		return "未知布局"
	}
}

// Token 跨进程用的记号（升级计划里写它）。
func (k Kind) Token() string {
	switch k {
	case KindPortable:
		return TokenPortable
	case KindMSI:
		return TokenMSI
	default:
		return "other"
	}
}

// DetectKind 判形态。判定顺序：便携标记优先于目录 —— 便携标记是包自己带的事实。
func DetectKind(exePath string) Kind {
	dir := filepath.Dir(exePath)
	if _, err := os.Stat(filepath.Join(dir, config.PortableMark)); err == nil {
		return KindPortable
	}
	if installDir := msiInstallDir(); installDir != "" && samePath(dir, installDir) {
		return KindMSI
	}
	return KindOther
}

// InstallRefusal 说明为什么这份 tt 不能自装，可以装时返回空串。
//
// 这段文字是给使用者看的判据，所以每种情形都指名"下一步该敲什么"，而不是笼统地拒绝。
func InstallRefusal(k Kind, exePath, version string) string {
	if version == "" {
		return "这份 tt 没有版本号（本地 go build 的产物）：源码态由 `make build` 更新，" +
			"要装发行版请从 " + RepoPage + "/releases 取 MSI 或便携包"
	}
	switch k {
	case KindPortable, KindMSI:
		return ""
	default:
		return fmt.Sprintf("认不出 %s 的安装布局：既没有 %s 标记、也不在 MSI 的安装目录里。"+
			"要自装请用发行版——MSI 装到 %s，便携包解压后目录里会带 %s 标记",
			filepath.Dir(exePath), config.PortableMark, msiInstallDir(), config.PortableMark)
	}
}

// msiInstallDir MSI 的安装目录：%LOCALAPPDATA%\Programs\TT。取不到环境变量时返回空串
// （判不出 MSI，退回 KindOther，不会误判成别的东西）。
func msiInstallDir() string {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		return ""
	}
	return filepath.Join(base, "Programs", "TT")
}

// samePath 比较两个路径是否指向同一处：忽略大小写与结尾分隔符
// （Windows 路径大小写不敏感，而环境变量里的写法与 os.Executable 的写法不一定一致）。
func samePath(a, b string) bool {
	norm := func(p string) string {
		p = filepath.Clean(p)
		return strings.ToLower(strings.TrimRight(p, `\/`))
	}
	return norm(a) == norm(b)
}
