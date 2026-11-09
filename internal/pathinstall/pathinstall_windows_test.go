//go:build windows

package pathinstall

import (
	"testing"

	"golang.org/x/sys/windows/registry"
)

// 用临时注册表键验证 添加/幂等/移除,不触碰真实用户 PATH。
func TestUserPathInstallWindows(t *testing.T) {
	const keyPath = `Software\TDictPathInstallTest`
	_ = registry.DeleteKey(registry.CURRENT_USER, keyPath)
	k, _, err := registry.CreateKey(registry.CURRENT_USER, keyPath, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		t.Fatalf("创建临时注册表键失败: %v", err)
	}
	k.Close()
	defer registry.DeleteKey(registry.CURRENT_USER, keyPath)

	old := userEnvKeyPath
	userEnvKeyPath = keyPath
	defer func() { userEnvKeyPath = old }()

	if st := Get(); !st.Supported || st.InUserPath {
		t.Fatalf("初始状态不符: %+v", st)
	}

	st, err := Add()
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if !st.InUserPath {
		t.Fatalf("添加后应命中: %+v", st)
	}

	// 幂等:重复添加只保留一项
	if _, err := Add(); err != nil {
		t.Fatalf("add2: %v", err)
	}
	if st = Get(); len(splitPathList(st.UserPath, ";")) != 1 {
		t.Fatalf("重复添加应幂等, 现有: %q", st.UserPath)
	}

	st, err = Remove()
	if err != nil {
		t.Fatalf("remove: %v", err)
	}
	if st.InUserPath {
		t.Fatalf("移除后不应命中: %+v", st)
	}
	if st.UserPath != "" {
		t.Fatalf("移除唯一项后应为空, got %q", st.UserPath)
	}
}
