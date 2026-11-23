//go:build windows

package proc

// runs cannot read another process's command line here without more than the standard
// library offers, and killing a pid on a guess is worse than leaving the agent. Reap
// therefore only clears its note on Windows.
func runs(int, string) bool { return false }
