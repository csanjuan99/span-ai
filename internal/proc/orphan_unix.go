//go:build !windows

package proc

import (
	"os/exec"
	"strings"
)

// runs reports whether the command line of pid names name. ps is on every Unix Span runs on.
func runs(pid int, name string) bool {
	out, err := exec.Command("ps", "-o", "command=", "-p", itoa(pid)).Output()
	return err == nil && strings.Contains(string(out), name)
}
