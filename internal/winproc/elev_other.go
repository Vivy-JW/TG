//go:build !windows

package winproc

import "errors"

// TokenElevated reports whether a process runs with an elevated token.
// Privilege separation only matters on Windows, which is the only platform the
// Genero Desktop Client flow targets here.
func TokenElevated(pid int) (bool, error) {
	return false, errors.New("winproc: TokenElevated is only implemented on Windows")
}
