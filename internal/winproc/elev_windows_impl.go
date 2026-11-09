//go:build windows

package winproc

import "syscall"

var (
	modKernel32             = syscall.NewLazyDLL("kernel32.dll")
	modAdvapi32             = syscall.NewLazyDLL("advapi32.dll")
	procOpenProcess         = modKernel32.NewProc("OpenProcess")
	procCloseHandle         = modKernel32.NewProc("CloseHandle")
	procOpenProcessToken    = modAdvapi32.NewProc("OpenProcessToken")
	procGetTokenInformation = modAdvapi32.NewProc("GetTokenInformation")
)

// currentProcess returns the pseudo handle of the calling process.
func currentProcess() uintptr { return ^uintptr(0) } // (HANDLE)-1 == GetCurrentProcess()

var (
	procCreateToolhelp32Snapshot = modKernel32.NewProc("CreateToolhelp32Snapshot")
	procProcess32First           = modKernel32.NewProc("Process32FirstW")
	procProcess32Next            = modKernel32.NewProc("Process32NextW")
)
