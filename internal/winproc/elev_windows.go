//go:build windows

package winproc

import (
	"fmt"
	"unsafe"
)

// TokenElevated reports whether a process runs with an elevated token.
//
// pid <= 0 means "this process". The check exists because driving a Genero
// Desktop Client window from a differently-privileged process silently fails
// (window messages are refused) or steals the foreground, so TT treats a
// mismatch as a pre-flight failure instead of a warning.
func TokenElevated(pid int) (bool, error) {
	const (
		processQueryLimited = 0x1000
		tokenQuery          = 0x0008
		tokenElevationClass = 20 // TokenElevation
	)
	var h uintptr
	if pid <= 0 {
		h = uintptr(currentProcess())
	} else {
		ph, _, err := procOpenProcess.Call(processQueryLimited, 0, uintptr(uint32(pid)))
		if ph == 0 {
			return false, fmt.Errorf("winproc: OpenProcess(%d): %v", pid, err)
		}
		defer procCloseHandle.Call(ph)
		h = ph
	}
	var tok uintptr
	ok, _, err := procOpenProcessToken.Call(h, tokenQuery, uintptr(unsafe.Pointer(&tok)))
	if ok == 0 {
		return false, fmt.Errorf("winproc: OpenProcessToken: %v", err)
	}
	defer procCloseHandle.Call(tok)

	type tokenElevation struct {
		TokenIsElevated uint32
	}
	var te tokenElevation
	var ret uint32
	ok, _, err = procGetTokenInformation.Call(tok, tokenElevationClass,
		uintptr(unsafe.Pointer(&te)), unsafe.Sizeof(te), uintptr(unsafe.Pointer(&ret)))
	if ok == 0 {
		return false, fmt.Errorf("winproc: GetTokenInformation: %v", err)
	}
	return te.TokenIsElevated != 0, nil
}
