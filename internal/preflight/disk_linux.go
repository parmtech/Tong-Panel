//go:build linux

package preflight

import "syscall"

func detectRootDiskAvailable() (uint64, bool) {
	var st syscall.Statfs_t
	if err := syscall.Statfs("/", &st); err != nil {
		return 0, false
	}
	return st.Bavail * uint64(st.Bsize), true
}
