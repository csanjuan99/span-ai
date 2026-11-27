//go:build !windows

package proc

import (
	"errors"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

// canList says this system's process table can be read.
const canList = true

// table lists the running processes. ps is on every Unix Span runs on. Zombies are left
// out: they run nothing and only wait to be collected.
func table() ([]entry, error) {
	cmd := exec.Command("ps", "-A", "-ww", "-o", "pid=,ppid=,pgid=,stat=,lstart=,command=")
	// lstart identifies a process across Span runs, so it must not follow the caller's
	// language or time zone.
	cmd.Env = append(os.Environ(), "LC_ALL=C", "TZ=UTC")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var rows []entry
	for _, line := range strings.Split(string(out), "\n") {
		// lstart is five fields: weekday, month, day, time, year.
		fields := strings.Fields(line)
		if len(fields) < 10 || strings.HasPrefix(fields[3], "Z") {
			continue
		}
		pid, errPID := strconv.Atoi(fields[0])
		ppid, errPPID := strconv.Atoi(fields[1])
		pgid, errPGID := strconv.Atoi(fields[2])
		if errPID != nil || errPPID != nil || errPGID != nil || pid == cmd.Process.Pid {
			continue
		}
		rows = append(rows, entry{pid: pid, ppid: ppid, pgid: pgid, started: strings.Join(fields[4:9], " "), command: strings.Join(fields[9:], " ")})
	}
	return rows, nil
}

func signalNamed(name string) syscall.Signal {
	if name == "SIGKILL" {
		return syscall.SIGKILL
	}
	return syscall.SIGTERM
}

func signalPID(pid int, name string) {
	if pid <= 1 {
		return
	}
	logSignal(pid, syscall.Kill(pid, signalNamed(name)))
	// A stopped process only acts on SIGTERM once it is continued.
	logSignal(pid, syscall.Kill(pid, syscall.SIGCONT))
}

func signalGroup(group int, name string) {
	if group > 1 {
		logGroupSignal(group, syscall.Kill(-group, signalNamed(name)))
		logGroupSignal(group, syscall.Kill(-group, syscall.SIGCONT))
	}
}

// groupAlive reports whether anything is left in the group, without a process table.
func groupAlive(group int) bool {
	return group > 1 && !processGone(syscall.Kill(-group, 0))
}

// logGroupSignal is logSignal for a group. macOS answers EPERM for a group with nothing left
// but processes waiting to be collected, which is not a failure.
func logGroupSignal(group int, err error) {
	if !errors.Is(err, syscall.EPERM) {
		logSignal(-group, err)
	}
}
