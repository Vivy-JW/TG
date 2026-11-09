//go:build windows

package config

import (
	"errors"
	"syscall"

	"golang.org/x/sys/windows"
)

// isAccessDeniedPlatform 在 Windows 上补一层 syscall.Errno 判据。
//
// 为什么需要:标准库的 errors.Is(err, fs.ErrPermission) 已经覆盖了 os.Rename /
// os.CreateTemp 这类**高层调用**的映射,但裸的 syscall.Errno(例如我们从
// syscall 层直接拿到、或包装过程中被剥掉包装的)不会命中那一层。多这一条
// 是为了让"重试/给可行动提示"这两件事在两种错误形态下都成立。
func isAccessDeniedPlatform(err error) bool {
	var errno syscall.Errno
	return errors.As(err, &errno) && errno == windows.ERROR_ACCESS_DENIED
}
