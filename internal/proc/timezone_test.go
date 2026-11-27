//go:build !windows

package proc

import "testing"

func TestReapFromAnotherTimeZoneStillFindsTheAgent(t *testing.T) {
	dir := t.TempDir()
	cmd, done := start(t)
	t.Setenv("TZ", "Etc/GMT-3")
	Leave(dir, cmd.Process.Pid, "sleep")
	t.Setenv("TZ", "Etc/GMT+9")

	if len(Reap(dir)) == 0 {
		t.Fatal("a reaper in another time zone left the orphan and its record behind")
	}
	<-done
}
