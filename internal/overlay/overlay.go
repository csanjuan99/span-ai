// Package overlay is the run's append-only record of what people decided and wrote. The
// story is fetched again whenever a person reloads it, so a correction cannot live in the
// story. It lives here, is applied on top of the story every time a prompt is built, and is
// never edited: a decision that changes is superseded by a new line, and both stay.
package overlay

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Kinds of line.
const (
	// Decision is something a person decided. It goes into every prompt.
	Decision = "decision"
	// Note is a version of what a person wrote for one stage. It is history: the stage's
	// instructions file is what the prompt reads.
	Note = "note"
	// Anchor says a decision still holds against a newer story.
	Anchor = "anchor"
	// Supersede retires decisions without replacing them.
	Supersede = "supersede"
)

// File is the log's name inside a run's directory.
const File = "decisions.jsonl"

// Entry is one line of the log. Seq is its identity, and the only thing a later line may
// point at.
type Entry struct {
	Seq  int       `json:"seq"`
	Time time.Time `json:"time"`
	Kind string    `json:"kind"`
	// Stage is where the person was when they wrote it.
	Stage string `json:"stage,omitempty"`
	// Class is the kind of question a decision answered.
	Class  string `json:"class,omitempty"`
	Title  string `json:"title,omitempty"`
	Option string `json:"option,omitempty"`
	Text   string `json:"text,omitempty"`
	// Dedupe names the problem a decision answers. A new decision with the key of an active
	// one supersedes it.
	Dedupe string `json:"dedupe,omitempty"`
	// Covers are the conflict keys a decision answered.
	Covers []string `json:"covers,omitempty"`
	// Epoch is the version of the story a decision was taken against.
	Epoch int `json:"epoch"`
	// Ref is the decision an anchor refers to.
	Ref        int   `json:"ref,omitempty"`
	Supersedes []int `json:"supersedes,omitempty"`
}

var mu sync.Mutex

// Append adds one line, assigns its Seq and returns it. The file is only ever appended to.
func Append(dir string, e Entry) (Entry, error) {
	if e.Kind == "" {
		return Entry{}, errors.New("an overlay line needs a kind")
	}
	// Appends from the page and from a run share this process; the lock keeps two of them
	// from taking the same Seq.
	mu.Lock()
	defer mu.Unlock()
	entries, err := Read(dir)
	if err != nil {
		return Entry{}, err
	}
	for _, old := range entries {
		e.Seq = max(e.Seq, old.Seq)
	}
	e.Seq++
	if e.Time.IsZero() {
		e.Time = time.Now().UTC()
	}
	if e.Kind == Decision && e.Dedupe != "" {
		for _, old := range Active(entries) {
			if old.Dedupe == e.Dedupe {
				e.Supersedes = append(e.Supersedes, old.Seq)
			}
		}
	}
	raw, err := json.Marshal(e)
	if err != nil {
		return Entry{}, err
	}
	f, err := os.OpenFile(filepath.Join(dir, File), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return Entry{}, err
	}
	if _, err := f.Write(append(raw, '\n')); err != nil {
		_ = f.Close()
		return Entry{}, err
	}
	// A failed close can be the only sign the line never reached the disk.
	if err := f.Close(); err != nil {
		return Entry{}, err
	}
	return e, nil
}

// Read returns every line, in order. A missing file is an empty log.
func Read(dir string) ([]Entry, error) {
	f, err := os.Open(filepath.Join(dir, File))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []Entry
	reader := bufio.NewReader(f)
	for n := 1; ; n++ {
		line, readErr := reader.ReadBytes('\n')
		if len(strings.TrimSpace(string(line))) > 0 {
			var e Entry
			if err := json.Unmarshal(line, &e); err != nil {
				// One torn line must not strand the run: it is skipped and said, and every
				// other decision still stands.
				log.Printf("%s line %d skipped: %v", File, n, err)
			} else {
				out = append(out, e)
			}
		}
		if errors.Is(readErr, io.EOF) {
			return out, nil
		}
		if readErr != nil {
			return nil, readErr
		}
	}
}

// Active is the decisions that still stand: not superseded, with the epoch of the newest
// anchor that confirmed each one.
func Active(entries []Entry) []Entry {
	gone := map[int]bool{}
	epoch := map[int]int{}
	for _, e := range entries {
		for _, seq := range e.Supersedes {
			gone[seq] = true
		}
		if e.Kind == Anchor {
			epoch[e.Ref] = e.Epoch
		}
	}
	var out []Entry
	for _, e := range entries {
		if e.Kind != Decision || gone[e.Seq] {
			continue
		}
		if confirmed, ok := epoch[e.Seq]; ok {
			e.Epoch = confirmed
		}
		out = append(out, e)
	}
	return out
}

// Stale is the active decisions taken against an older version of the story than the
// current one. They are neither applied nor dropped until a person says which.
func Stale(entries []Entry, epoch int) []Entry {
	var out []Entry
	for _, e := range Active(entries) {
		if e.Epoch != epoch {
			out = append(out, e)
		}
	}
	return out
}

// oneLine keeps a stored text on one line, so what a person or an agent wrote cannot start a
// line of its own and pass for another decision.
func oneLine(text string) string { return strings.Join(strings.Fields(text), " ") }

// Render is the block a prompt carries: each decision with who took it, when, and where,
// so an agent can weigh it and a reader can find it.
func Render(entries []Entry, epoch int) string {
	active := Active(entries)
	if len(active) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("Decisions a person took for this run. Where the story says otherwise, follow them. They never override a rule the repository enforces (a hook, CI).\n")
	for _, e := range active {
		fmt.Fprintf(&b, "[D%d] %s, at %s, on %q: %s", e.Seq, e.Time.UTC().Format("2006-01-02 15:04"), e.Stage, oneLine(e.Title), oneLine(e.Text))
		if e.Epoch != epoch {
			b.WriteString(" (taken against an earlier version of the story)")
		}
		b.WriteString("\n")
	}
	return b.String()
}
