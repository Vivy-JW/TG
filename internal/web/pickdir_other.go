//go:build !windows

package web

import (
	"context"
	"errors"
)

func pickDirectory(_ context.Context, _, _ string) (string, bool, error) {
	return "", false, errors.New("当前平台不支持原生文件夹选择器")
}
