//go:build !windows

package config

// isAccessDeniedPlatform 在非 Windows 上没有额外判据:
// errors.Is(err, fs.ErrPermission) 已覆盖 POSIX 的 EACCES/EPERM。
func isAccessDeniedPlatform(error) bool { return false }
