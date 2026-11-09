package debug

// 断点持久化(fgldeb 状态文件的 Go 版):
// <DataDir>/debug-bps/<module>__<prog>.json,记录 file:line + 源码行文本 + 启用态。
// 恢复时按行文本重新定位,源码变更不会把断点恢复到错误位置(fgldeb 同款防护)。

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"tt/internal/config"
)

// StoredBP 持久化的断点
//
// Cond/Ignore/Kind 是 A20（契约 §5）新增的三个字段：**全部 omitempty**，
// 所以旧版本写出的存档（没有这三个键）读进来就是零值（"" / 0 / ""），
// 而新存档里没设的字段也不会多出 `"cond":""` 这种噪声键。
type StoredBP struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	LineText string `json:"lineText,omitempty"` // 下断时的源码行文本(重定位依据)
	Func     string `json:"func,omitempty"`
	Enabled  bool   `json:"enabled"`
	Cond     string `json:"cond,omitempty"`   // 条件表达式(空 = 无条件)
	Ignore   int    `json:"ignore,omitempty"` // 忽略前 N 次命中(0 = 不忽略)
	Kind     string `json:"kind,omitempty"`   // "break" | "watch"(空 = "break")
}

type bpStoreFile struct {
	Module      string     `json:"module"`
	Prog        string     `json:"prog"`
	SavedAt     string     `json:"savedAt"`
	Breakpoints []StoredBP `json:"breakpoints"`
}

var bpNameRe = regexp.MustCompile(`[^A-Za-z0-9._-]`)

// bpStoreDir 断点存档目录名 —— 必须与 config.CacheSubdirs 里的清单一致,
// config.CacheDir 会对它做校验。
const bpStoreDir = "debug-bps"

// bpProduct 产品命名空间：T100 的 Product 为空，统一落到 t100；GP 为 gp。
//
// A 方案只隔离产品，不隔离 T100 的环境：同一程序在不同 T100 环境之间仍然共享断点，
// 这是原行为。是否应该按环境隔离（或按源码版本隔离）待与 TT 开发者确认。
func bpProduct(product string) string {
	p := strings.ToLower(strings.TrimSpace(product))
	if p == "" {
		return "t100"
	}
	return bpNameRe.ReplaceAllString(p, "_")
}

func bpStorePath(dataDir, product, module, prog string) string {
	name := bpNameRe.ReplaceAllString(module+"__"+prog, "_")
	return filepath.Join(config.CacheDir(dataDir, bpStoreDir), bpProduct(product), name+".json")
}

// bpLegacyStorePath 旧版扁平落点：debug-bps/<module>__<prog>.json。
// 只给 T100 读取做兼容；GP 不读，避免把 T100 的同名程序断点带进 GP。
func bpLegacyStorePath(dataDir, module, prog string) string {
	name := bpNameRe.ReplaceAllString(module+"__"+prog, "_")
	return filepath.Join(config.CacheDir(dataDir, bpStoreDir), name+".json")
}

// saveBPs 原子写入断点存档;dataDir 为空时不落盘
func saveBPs(dataDir, product, module, prog string, bps []StoredBP) error {
	if dataDir == "" {
		return nil
	}
	p := bpStorePath(dataDir, product, module, prog)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(bpStoreFile{
		Module: module, Prog: prog,
		SavedAt:     time.Now().Format(time.RFC3339),
		Breakpoints: bps,
	}, "", "  ")
	if err != nil {
		return err
	}
	// 与 config.json 同一套原子写：Windows 上目标文件被瞬时占用时重试，
	// 不再手写“WriteFile + Rename”。断点存档丢了不会报错，只会静默少一条。
	return config.AtomicWrite(p, data)
}

// loadBPs 读取断点存档(不存在返回错误,由调用方忽略)
func loadBPs(dataDir, product, module, prog string) (*bpStoreFile, error) {
	if dataDir == "" {
		return nil, fmt.Errorf("未启用断点持久化")
	}
	data, err := os.ReadFile(bpStorePath(dataDir, product, module, prog))
	if err != nil && bpProduct(product) == "t100" {
		// 旧版扁平文件只可能是 T100 时代留下的；读一次做兼容，
		// 下次 saveBPs 会写到新命名空间。GP 不读，避免跨产品串断点。
		data, err = os.ReadFile(bpLegacyStorePath(dataDir, module, prog))
	}
	if err != nil {
		return nil, err
	}
	var st bpStoreFile
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, err
	}
	return &st, nil
}
