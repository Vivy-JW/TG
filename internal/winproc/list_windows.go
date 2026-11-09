//go:build windows

package winproc

import (
	"fmt"
	"strings"
	"unsafe"
)

// ListProcesses returns the PIDs whose executable name matches name
// (case-insensitive, with or without the .exe suffix).
//
// It exists for the privilege pre-flight: an already-running instance started
// with different privileges is the failure mode that makes the whole approach
// fail silently, and it has to be detected before anything is launched.
func ListProcesses(name string) ([]int, error) {
	const (
		th32csSnapProcess = 0x00000002
		invalidHandle     = ^uintptr(0)
	)
	snap, _, err := procCreateToolhelp32Snapshot.Call(th32csSnapProcess, 0)
	if snap == invalidHandle {
		return nil, fmt.Errorf("winproc: CreateToolhelp32Snapshot: %v", err)
	}
	defer procCloseHandle.Call(snap)

	type processEntry32 struct {
		Size            uint32
		CntUsage        uint32
		ProcessID       uint32
		DefaultHeapID   uintptr
		ModuleID        uint32
		CntThreads      uint32
		ParentProcessID uint32
		PriClassBase    int32
		Flags           uint32
		ExeFile         [260]uint16
	}
	var pe processEntry32
	pe.Size = uint32(unsafe.Sizeof(pe))

	want := strings.ToLower(name)
	if !strings.HasSuffix(want, ".exe") {
		want += ".exe"
	}
	var out []int
	ok, _, err := procProcess32First.Call(snap, uintptr(unsafe.Pointer(&pe)))
	for ok != 0 {
		exe := strings.ToLower(syscallUTF16ToString(pe.ExeFile[:]))
		if exe == want {
			out = append(out, int(pe.ProcessID))
		}
		pe.Size = uint32(unsafe.Sizeof(pe))
		ok, _, err = procProcess32Next.Call(snap, uintptr(unsafe.Pointer(&pe)))
	}
	_ = err
	return out, nil
}

// syscallUTF16ToString converts a NUL-terminated UTF-16 buffer.
func syscallUTF16ToString(b []uint16) string {
	var sb strings.Builder
	for _, c := range b {
		if c == 0 {
			break
		}
		sb.WriteRune(rune(c))
	}
	return sb.String()
}
