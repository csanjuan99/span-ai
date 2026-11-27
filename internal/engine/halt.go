package engine

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"time"

	"span/internal/journal"
	"span/internal/proc"
	"span/internal/runlock"
)

// PauseReason is what a person who asked for a pause is told.
const PauseReason = "an agent in flight is stopped and the run holds; any other stage finishes first"

// Relaunchable reports why a run cannot be relaunched. driven says something is running it
// now. The page and the command line both ask here, so they refuse alike.
func Relaunchable(cur Cursor, driven bool) error {
	switch {
	case cur.Stop || cur.Done:
		return errors.New("this run will not continue")
	case Waiting(cur):
		return errors.New("this run is waiting for an answer")
	// An agent that cannot run is a hold, and Relaunch starts that stage again. Every other
	// hold stays paused until Resume, so a waiting answer is not skipped.
	case !driven && cur.Hold && cur.AgentBlock == nil:
		return errors.New("this run is paused")
	}
	return nil
}

// HaltWait is how long a driver gets to give up a run, on top of proc.Grace.
var HaltWait = 20 * time.Second

// HaltLimit bounds the whole wait for a driver to end.
func HaltLimit() time.Duration { return proc.Grace + HaltWait }

// Halt stops whoever is driving the run, this process or another, and returns once the run
// lock is free. The driver releases it only after its agent's whole tree has ended. A run
// nobody drives is left alone. A driver that does not end in time is a runlock.BusyError.
func Halt(root, id string) error {
	deadline := time.Now().Add(HaltLimit())
	for {
		holder, busy := RunBusy(root, id)
		if !busy {
			return nil
		}
		if time.Now().After(deadline) {
			return &runlock.BusyError{Scope: runlock.Run, Holder: holder}
		}
		// Asked again while it is busy: a flight that was only starting drops an earlier marker.
		if _, err := os.Stat(filepath.Join(root, id, stopFile)); errors.Is(err, os.ErrNotExist) {
			if err := RequestStop(root, id); err != nil {
				return err
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// ReapRun ends the agent processes still recorded for a run nobody is driving, journals
// each one and returns how many there were. A driven run is left to its driver.
func ReapRun(root, id string) int {
	if !validID(id) {
		return 0
	}
	dir := filepath.Join(root, id)
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return 0
	}
	lock, err := runlock.Acquire(runlock.Run, runLockPath(root, id), "reap "+id)
	if err != nil {
		var busy *runlock.BusyError
		if !errors.As(err, &busy) {
			log.Printf("reap %s: %v", id, err)
		}
		return 0
	}
	defer lock.Release()
	return reapHeld(dir, id)
}

// reapHeld is ReapRun for a caller that already holds the run lock.
func reapHeld(dir, id string) int {
	ends := proc.Reap(dir)
	for _, end := range ends {
		if err := journal.Append(dir, journal.Span, "agent.terminated", terminationDetail(end)); err != nil {
			log.Printf("journal %s: %v", id, err)
		}
	}
	return len(ends)
}

// ReapRuns reaps every run under root, so no agent outlives its run across a restart.
func ReapRuns(root string) {
	entries, err := os.ReadDir(root)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Printf("reap the runs in %s: %v", root, err)
		}
		return
	}
	for _, entry := range entries {
		if entry.IsDir() {
			if n := ReapRun(root, entry.Name()); n > 0 {
				log.Printf("run %s: ended %d agent processes left behind", entry.Name(), n)
			}
		}
	}
}
