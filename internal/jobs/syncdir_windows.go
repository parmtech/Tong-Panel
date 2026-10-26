//go:build windows

package jobs

// Windows does not support fsync on directory handles; the journal file itself
// is flushed before replacement. Unix builds additionally sync the parent dir.
func syncDir(string) error { return nil }
