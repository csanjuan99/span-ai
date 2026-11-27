//go:build windows

package proc

import "errors"

// canList is false: reading the process table needs more than the standard library offers,
// and killing a pid on a guess is worse than leaving it.
const canList = false

func table() ([]entry, error) { return nil, errors.New("no process table on windows") }

func signalPID(int, string) {}

func signalGroup(int, string) {}

func groupAlive(int) bool { return false }
