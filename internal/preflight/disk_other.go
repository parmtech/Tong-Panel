//go:build !linux

package preflight

func detectRootDiskAvailable() (uint64, bool) {
	return 0, false
}
