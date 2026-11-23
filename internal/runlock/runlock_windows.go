//go:build windows

package runlock

import "os"

// Windows has no flock in the standard library. The lock is not enforced there; the
// in-process Hub claim still keeps one serve process from driving a run twice.
func lockFile(*os.File) error { return nil }

func unlockFile(*os.File) {}

func isBusy(error) bool { return false }
