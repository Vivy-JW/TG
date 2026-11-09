package server

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"tt/internal/config"
)

// writeTempConfig 在临时目录写一份 config.json,返回其路径。
func writeTempConfig(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatalf("写入临时配置失败: %v", err)
	}
	return p
}

// readRoot 把落盘的 config.json 读成 map(测试断言顶层键用)。
func readRoot(t *testing.T, p string) map[string]any {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var root map[string]any
	if err := json.Unmarshal(b, &root); err != nil {
		t.Fatal(err)
	}
	return root
}

// setSection 直接改配置里的某一节。
//
// 替代本包原先的 PUT /api/mirror、PUT /api/dbsync、PUT /api/bdldoc —— 那三个端点写的
// 就是这几节,与 PUT /api/hosts 的分节写入重复,已删除。这些测试关心的是 GET 侧的
// 派生结果(路径解析、文件是否存在),写配置只是准备步骤,走同一份 config.EditSection 即可。
func setSection(t *testing.T, path, key string, vals map[string]any) {
	t.Helper()
	if err := config.EditSection(path, key, nil, func(sec map[string]any) error {
		for k, v := range vals {
			sec[k] = v
		}
		return nil
	}); err != nil {
		t.Fatalf("写配置节 %s 失败: %v", key, err)
	}
}

// clearSectionKey 删掉某一节里的一个键(验证"空 = 回到默认"这条路径)。
func clearSectionKey(t *testing.T, path, key, sub string) {
	t.Helper()
	if err := config.EditSection(path, key, nil, func(sec map[string]any) error {
		delete(sec, sub)
		return nil
	}); err != nil {
		t.Fatalf("清空 %s.%s 失败: %v", key, sub, err)
	}
}
