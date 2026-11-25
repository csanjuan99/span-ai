package engine

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"time"

	"span/internal/fsutil"
)

// stopFile is the marker `span stop` and the page leave for the process driving a run.
const stopFile = "stop"

// stopPoll is how often a flight looks for its stop marker.
const stopPoll = 300 * time.Millisecond

// RequestStop asks the process driving the run, whichever it is, to cancel the stage in
// flight. Like a pause it is a file beside the cursor, so it crosses processes. An unknown
// run is os.ErrNotExist.
func RequestStop(root, id string) error {
	if !validID(id) {
		return os.ErrNotExist
	}
	dir := filepath.Join(root, id)
	if _, err := os.Stat(filepath.Join(dir, "cursor.json")); err != nil {
		return err
	}
	return fsutil.WriteAtomic(filepath.Join(dir, stopFile), []byte("stopped by a person\n"))
}

// StopOnRequest returns a context that is cancelled when a stop marker appears for the run,
// and the function that ends the watch. A marker left from before the flight started is
// discarded first, so an old request never cancels a later resume.
func StopOnRequest(parent context.Context, root, id string) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(parent)
	if !validID(id) {
		return ctx, cancel
	}
	path := filepath.Join(root, id, stopFile)
	dropMarker(id, path)
	go func() {
		tick := time.NewTicker(stopPoll)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				if _, err := os.Stat(path); err == nil {
					dropMarker(id, path)
					cancel()
					return
				}
			}
		}
	}()
	return ctx, cancel
}

// dropMarker removes a stop marker. A marker that is already gone is fine; any other
// failure is logged, because a marker that stays would cancel the next flight.
func dropMarker(id, path string) {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		log.Printf("remove %s %s: %v", id, filepath.Base(path), err)
	}
}
