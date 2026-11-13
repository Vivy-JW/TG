package testenv

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// DefaultCorpusRoot 是语料的内置缺省（引擎那侧 TZSCLI_WS 的缺省也在它下面）。
const DefaultCorpusRoot = `D:\t100_wrok_dir`

// LocalFileName 是那份机器本地配置的文件名，放在仓库根（.gitignore 已经忽略它）。
const LocalFileName = "config.local.json"

// Config 是 config.local.json 的形状。目前只剩语料根一项（机器路径，可留空）。
//
// 它有一个同义的环境变量（见 CorpusRootDetail），环境变量优先 ——
// 所以这份文件是"省得每次开终端都设变量"，不是必须。
//
// 2026-11-13：engineExe / workspace / designerDir 三项随 T100 设计器包管线一起删掉，
// 读它们的 accessor（EngineExe / Workspace / DesignerDir）同批删；TG 没有 .tzs 引擎这个外部 exe。
type Config struct {
	CorpusRoot string `json:"corpusRoot"` // 语料根；对应 TDEV_CORPUS / TTZS_CORPUS
}

// Local 读本机测试配置。**每次调用都重新读**（不做缓存）：文件很小，而缓存 +
// os.Chdir（internal/config 的测试会 chdir）会得到"读的是上一个工作目录的配置"这种难查的错。
// 文件不存在、读不动、不是合法 JSON —— 一律当作**空配置**：没有这份文件是正常状态。
func Local() Config {
	p := localFilePath()
	if p == "" {
		return Config{}
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return Config{}
	}
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return Config{}
	}
	return c
}

// LocalPath 返回本机配置文件的路径；找不到时返回 ""。
//
// 定位方式：从工作目录逐级上溯到含 go.mod 的那一级（仓库根），取那里的 config.local.json。
// 上溯而不是写死相对层级：测试的工作目录是包目录，层级各包不同。
func LocalPath() string { return localFilePath() }

func localFilePath() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			p := filepath.Join(dir, LocalFileName)
			if _, err := os.Stat(p); err == nil {
				return p
			}
			return ""
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "" // 到顶了：不在仓库里跑，没有这份文件
		}
		dir = parent
	}
}

// ---- 四项：环境变量 > config.local.json > （语料还有内置缺省）----

// corpusEnvVars 是语料根的覆盖变量，按顺序取第一个设置了的。
//
// 两个名字都认是刻意的：TDEV_CORPUS 是 `.tzc` 那条管线先用的，TTZS_CORPUS 是 `.tzs` 那条的。
// 认两个不等于有两份真源 —— 值只有一个，谁先设谁说话。
// （从前 internal/dev/{fence,pkgfile,tapfile,fgl} 各抄了一份**只认 TDEV_CORPUS** 的解析，
// 于是只设 TTZS_CORPUS 的机器在那四个包上会静默跑零个包。见 internal/testkit 的防重复断言。）
var corpusEnvVars = []string{"TDEV_CORPUS", "TTZS_CORPUS"}

// CorpusRoot 解析语料根；解析不出来返回 ""。
//
// 环境变量设了却指不到目录 → 返回 ""（不回落到文件、也不回落到缺省）：那表示调用方
// 想指出一份语料而指错了，静默换一份别的去跑会让"我以为跑的是这份"这个错误结论留下来。
// 文件里写了却指不到，同理。
func CorpusRoot() string {
	for _, k := range corpusEnvVars {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return dirOrEmpty(v)
		}
	}
	if v := strings.TrimSpace(Local().CorpusRoot); v != "" {
		return dirOrEmpty(v)
	}
	return dirOrEmpty(DefaultCorpusRoot)
}

// CorpusRootDetail 讲清**这次语料根是怎么定的、没定下来是卡在哪一处**，供"还差什么"这类
// 地方直接引用（`tt dev tzs doctor` 就是这么用的）。
//
// 为什么要有它：调用方自己猜"是哪一处没指到"一定会猜错 —— 它会拿"配置文件在不在"去推，
// 而真正卡住的是环境变量。**猜错的结果是给出的指引正好指向错的那条路**，比不说更坏。
// 所以四处的取舍只有这一处说了算。
func CorpusRootDetail() string {
	for _, k := range corpusEnvVars {
		v := strings.TrimSpace(os.Getenv(k))
		if v == "" {
			continue
		}
		if dirOrEmpty(v) != "" {
			return k + "=" + v
		}
		return k + "=" + v + "（指不到目录，且不回落）"
	}
	if p := localFilePath(); p != "" {
		v := strings.TrimSpace(Local().CorpusRoot)
		if v == "" {
			return LocalFileName + " 在（" + p + "），但没写 corpusRoot"
		}
		if dirOrEmpty(v) != "" {
			return LocalFileName + " 的 corpusRoot=" + v
		}
		return LocalFileName + " 的 corpusRoot=" + v + "（指不到目录，且不回落）"
	}
	if dirOrEmpty(DefaultCorpusRoot) != "" {
		return "内置缺省 " + DefaultCorpusRoot
	}
	return "四处都没有：" + strings.Join(corpusEnvVars, " / ") + "、" + LocalFileName + "、" +
		DefaultCorpusRoot
}

// dirOrEmpty 是路径存在且是目录时返回它，否则返回 ""。
func dirOrEmpty(p string) string {
	if st, err := os.Stat(p); err == nil && st.IsDir() {
		return p
	}
	return ""
}
