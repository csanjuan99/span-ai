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
	"os"
	"path/filepath"
	"strings"
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
	Stage  string `json:"stage,omitempty"`
	Title  string `json:"title,omitempty"`
	Option string `json:"option,omitempty"`
	Text   string `json:"text,omitempty"`
	// Dedupe names the problem a decision answers. A new decision with the key of an active
	// one supersedes it.
	Dedupe string `json:"dedupe,omitempty"`
	// Epoch is the version of the story a decision was taken against.
	Epoch int `json:"epoch"`
	// Ref is the decision an anchor refers to.
	Ref        int   `json:"ref,omitempty"`
	Supersedes []int `json:"supersedes,omitempty"`
}

// Append adds one line, assigns its Seq and returns it. The file is only ever appended to.
func Append(dir string, e Entry) (Entry, error) {
	if e.Kind == "" {
		return Entry{}, errors.New("an overlay line needs a kind")
	}
	entries, err := Read(dir)
	if err != nil {
		return Entry{}, err
	}
	e.Seq = len(entries) + 1
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
	defer f.Close()
	if _, err := f.Write(append(raw, '\n')); err != nil {
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
	scan := bufio.NewScanner(f)
	scan.Buffer(make([]byte, 0, 64*1024), 4<<20)
	for scan.Scan() {
		if strings.TrimSpace(scan.Text()) == "" {
			continue
		}
		var e Entry
		if err := json.Unmarshal(scan.Bytes(), &e); err != nil {
			return nil, fmt.Errorf("%s line %d: %w", File, len(out)+1, err)
		}
		out = append(out, e)
	}
	return out, scan.Err()
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
		fmt.Fprintf(&b, "[D%d] %s, at %s, on %q: %s", e.Seq, e.Time.UTC().Format("2006-01-02 15:04"), e.Stage, e.Title, e.Text)
		if e.Epoch != epoch {
			b.WriteString(" (taken against an earlier version of the story)")
		}
		b.WriteString("\n")
	}
	return b.String()
}
