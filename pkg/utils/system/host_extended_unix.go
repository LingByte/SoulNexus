//go:build !windows

package system

import "syscall"

func processMaxFDs() uint64 {
	var rlimit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rlimit); err != nil {
		return 0
	}
	return rlimit.Cur
}
