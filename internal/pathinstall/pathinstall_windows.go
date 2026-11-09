//go:build windows

package pathinstall

import (
	"errors"
	"fmt"
	"path/filepath"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const (
	pathSep       = ";"
	pathValueName = "Path"
)

// userEnvKeyPath HKCU 下的用户环境变量键;测试可覆盖为临时键,避免动真实 PATH。
var userEnvKeyPath = "Environment"

// Get 读取用户 PATH 并判断可执行文件目录是否已在其中。
func Get() Status {
	exe := exePath()
	dir := filepath.Dir(exe)
	st := Status{Supported: true, ExePath: exe, ExeDir: dir}
	cur, err := readUserPath()
	if err != nil {
		st.Note = "读取用户 PATH 失败: " + err.Error()
		return st
	}
	st.UserPath = cur
	st.InUserPath = pathListContains(splitPathList(cur, pathSep), dir, true)
	return st
}

// Preview 计算「加入 exe 目录后」的 PATH 值,但不写注册表。
// 返回当前状态、将要写入的值、以及是否会发生变化。供 `install path --dry-run` 与 Add 共用。
func Preview() (Status, string, bool, error) {
	dir := filepath.Dir(exePath())
	if dir == "" || dir == "." {
		return Status{}, "", false, fmt.Errorf("无法定位可执行文件目录")
	}
	cur, err := readUserPath()
	if err != nil {
		return Status{}, "", false, err
	}
	next := joinPathList(pathListAdd(splitPathList(cur, pathSep), dir, true), pathSep)
	return Get(), next, next != cur, nil
}

// Add 把可执行文件所在目录追加到用户 PATH(已存在则幂等)。
func Add() (Status, error) {
	st, next, changed, err := Preview()
	if err != nil {
		return Status{}, err
	}
	if !changed {
		return st, nil
	}
	if err := writeUserPath(next); err != nil {
		return Status{}, err
	}
	broadcastEnvChange()
	return Get(), nil
}

// Remove 从用户 PATH 移除可执行文件所在目录(不存在则幂等)。
func Remove() (Status, error) {
	dir := filepath.Dir(exePath())
	cur, err := readUserPath()
	if err != nil {
		return Status{}, err
	}
	next := joinPathList(pathListRemove(splitPathList(cur, pathSep), dir, true), pathSep)
	if next != cur {
		if err := writeUserPath(next); err != nil {
			return Status{}, err
		}
		broadcastEnvChange()
	}
	return Get(), nil
}

// readUserPath 读用户 PATH;值不存在时视为空(不报错)。
func readUserPath() (string, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, userEnvKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return "", err
	}
	defer k.Close()
	v, _, err := k.GetStringValue(pathValueName)
	if err != nil {
		if errors.Is(err, syscall.ERROR_FILE_NOT_FOUND) {
			return "", nil
		}
		return "", err
	}
	return v, nil
}

// writeUserPath 写回用户 PATH,保持 REG_EXPAND_SZ(值里常有 %USERPROFILE% 等可展开变量)。
//
// **它是无条件写的**：不读原值是什么类型，所以原值本来是 REG_SZ（不含任何 `%`）时，
// 也会被改写成 REG_EXPAND_SZ。原先会保留原类型的那份实现在 `tt dev install` 里，
// **已随该命令一起删除，现在没有第二份实现兜着了**（2026-09-27 从已删的规划文档转来，
// 那条注着"别丢"）。
//
// 若要改这里的行为，先想清楚两件事：改类型会不会影响 `%VAR%` 的展开、
// 以及卸载时按类型摘除的那条路径还成不成立。
func writeUserPath(v string) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, userEnvKeyPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	return k.SetExpandStringValue(pathValueName, v)
}

// broadcastEnvChange 广播环境变量变更,让已运行的资源管理器/终端尽快感知
// (新开的终端本就会重新读注册表)。
//
// 用 SendNotifyMessageW 而非 SendMessageW:后者对 HWND_BROADCAST 会同步等待
// 每个顶层窗口处理消息,遇到无响应的窗口会长时间阻塞;SendNotifyMessageW 对
// 其它线程的窗口只投递、不等待。
func broadcastEnvChange() {
	user32 := windows.NewLazySystemDLL("user32.dll")
	notify := user32.NewProc("SendNotifyMessageW")
	p, _ := windows.UTF16PtrFromString("Environment")
	const (
		hwndBroadcast   = 0xffff
		wmSettingChange = 0x001A
	)
	_, _, _ = notify.Call(uintptr(hwndBroadcast), uintptr(wmSettingChange), 0, uintptr(unsafe.Pointer(p)))
}
