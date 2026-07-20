//go:build windows

package system

// Windows has no RLIMIT_NOFILE; MaxFDs is reported as 0.
func processMaxFDs() uint64 {
	return 0
}
