//go:build !windows

package winproc

import "errors"

// ListProcesses is only implemented on Windows.
func ListProcesses(name string) ([]int, error) {
	return nil, errors.New("winproc: ListProcesses is only implemented on Windows")
}
