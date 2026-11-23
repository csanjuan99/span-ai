// Package journal keeps an append-only record of what happened to a run and who did it.
// Statements already seal what each stage decided. The journal adds the rest: every stage
// transition and every action a person took (stop, pause, rewind, approve, agent choice),
// each numbered and chained to the one before, so a missing or edited line shows.
package journal

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"span/internal/runlock"
)

// File is the journal's name inside a run's directory.
const File = "events.jsonl"

// Who took an action.
const (
	// Person is a human action through the page or the command line.
	Person = "person"
	// Span is the engine moving the run.
	Span = "span"
)

// Event is one line of the journal. Detail holds short, fixed-meaning fields such as a stage
// or a CLI id. Free text a person typed is never stored here, only its length.
type Event struct {
	Seq    int               `json:"seq"`
	Time   time.Time         `json:"time"`
	Actor  string            `json:"actor"`
	Type   string            `json:"type"`
	Detail map[string]string `json:"detail,omitempty"`
	// Prev is the SHA-256 of the previous line's bytes. The first event has none.
	Prev string `json:"prev,omitempty"`
}

// lockWait bounds how long an append waits for another process to finish its own.
const lockWait = 5 * time.Second

// Append adds one event. It takes a short lock on the file so the serve process and a
// resumed run, which are different processes, cannot reuse a sequence number.
func Append(dir, actor, typ string, detail map[string]string) error {
	lock, err := acquire(filepath.Join(dir, File+".lock"))
	if err != nil {
		return err
	}
	defer lock.Release()
	last, line, err := tail(filepath.Join(dir, File))
	if err != nil {
		return err
	}
	event := Event{Seq: last.Seq + 1, Time: time.Now().UTC(), Actor: actor, Type: typ, Detail: detail}
	if len(line) > 0 {
		event.Prev = digest(line)
	}
	raw, err := json.Marshal(event)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(filepath.Join(dir, File), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	if _, err := file.Write(append(raw, '\n')); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func acquire(path string) (*runlock.Lock, error) {
	deadline := time.Now().Add(lockWait)
	for {
		lock, err := runlock.Acquire(runlock.Journal, path, "journal")
		if err == nil {
			return lock, nil
		}
		var busy *runlock.BusyError
		if !errors.As(err, &busy) || time.Now().After(deadline) {
			return nil, err
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// Read returns every event in order. A run with no journal has none.
func Read(dir string) ([]Event, error) {
	lines, err := readLines(filepath.Join(dir, File))
	if err != nil {
		return nil, err
	}
	events := make([]Event, 0, len(lines))
	for _, line := range lines {
		var event Event
		if err := json.Unmarshal(line, &event); err != nil {
			return events, fmt.Errorf("journal line %d: %w", len(events)+1, err)
		}
		events = append(events, event)
	}
	return events, nil
}

// Verify checks the sequence and the chain. It returns the seq of the first event that does
// not follow from the one before it, or 0 when the whole journal holds together.
func Verify(dir string) (int, error) {
	lines, err := readLines(filepath.Join(dir, File))
	if err != nil {
		return 0, err
	}
	var previous []byte
	for index, line := range lines {
		var event Event
		if err := json.Unmarshal(line, &event); err != nil {
			return index + 1, nil
		}
		want := ""
		if previous != nil {
			want = digest(previous)
		}
		if event.Seq != index+1 || event.Prev != want {
			return index + 1, nil
		}
		previous = line
	}
	return 0, nil
}

func readLines(path string) ([][]byte, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var lines [][]byte
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for scanner.Scan() {
		if text := bytes.TrimSpace(scanner.Bytes()); len(text) > 0 {
			lines = append(lines, append([]byte(nil), text...))
		}
	}
	return lines, scanner.Err()
}

// tail is the last event and its bytes. An empty or missing journal has neither.
func tail(path string) (Event, []byte, error) {
	lines, err := readLines(path)
	if err != nil || len(lines) == 0 {
		return Event{}, nil, err
	}
	var event Event
	if err := json.Unmarshal(lines[len(lines)-1], &event); err != nil {
		return Event{}, nil, fmt.Errorf("journal is damaged at line %d: %w", len(lines), err)
	}
	return event, lines[len(lines)-1], nil
}

func digest(line []byte) string {
	sum := sha256.Sum256(line)
	return hex.EncodeToString(sum[:])
}
