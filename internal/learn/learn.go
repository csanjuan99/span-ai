// Package learn keeps a small per-project ledger of sealed stage mistakes.
// The next prompt for that stage reads a few lines. Nothing here calls a model.
package learn

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	maxLessons = 32
	maxShown   = 5
	maxRule    = 160
)

// Lesson is one sealed reason for one stage.
type Lesson struct {
	Stage     string `json:"stage"`
	Signature string `json:"signature"`
	Rule      string `json:"rule"`
	Seen      int    `json:"seen"`
	Helpful   int    `json:"helpful"`
	Harmful   int    `json:"harmful"`
}

// Book is the on-disk ledger.
type Book struct {
	Lessons []Lesson `json:"lessons"`
}

// Applies reports whether this stage shows lessons to an agent.
func Applies(stage string) bool {
	switch stage {
	case "frame", "build", "inspect":
		return true
	default:
		return false
	}
}

// Load reads the ledger. A missing file is an empty book.
func Load(path string) (*Book, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return &Book{}, nil
	}
	if err != nil {
		return nil, err
	}
	var book Book
	if err := json.Unmarshal(raw, &book); err != nil {
		return nil, err
	}
	return &book, nil
}

// Save writes the ledger atomically.
func Save(path string, book *Book) error {
	if book == nil {
		return errors.New("no lesson book")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(book, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Select is the prompt block for stage and the signatures it included.
func (b *Book) Select(stage string) (string, []string) {
	if b == nil {
		return "", nil
	}
	order := make([]int, 0, len(b.Lessons))
	for i, lesson := range b.Lessons {
		if lesson.Stage != stage || lesson.Seen < 1 || retired(lesson) {
			continue
		}
		order = append(order, i)
	}
	sort.SliceStable(order, func(i, j int) bool {
		left, right := b.Lessons[order[i]], b.Lessons[order[j]]
		if score(left) != score(right) {
			return score(left) > score(right)
		}
		return left.Seen > right.Seen
	})
	if len(order) == 0 {
		return "", nil
	}
	if len(order) > maxShown {
		order = order[:maxShown]
	}
	lines := make([]string, 0, len(order))
	sigs := make([]string, 0, len(order))
	for _, i := range order {
		lines = append(lines, "- "+b.Lessons[i].Rule)
		sigs = append(sigs, b.Lessons[i].Signature)
	}
	return strings.Join(lines, "\n"), sigs
}

// Observe records a sealed verdict. shown are the signatures Select put in the prompt.
// It reports whether the book changed.
func (b *Book) Observe(stage, verdict, reason string, shown []string) bool {
	if b == nil || !Applies(stage) {
		return false
	}
	changed := false
	display, sig, ok := splitReason(reason)
	switch verdict {
	case "advance":
		for _, id := range shown {
			if b.credit(stage, id, true) {
				changed = true
			}
		}
	case "hold", "stop":
		for _, id := range shown {
			if id == sig && b.credit(stage, id, false) {
				changed = true
			}
		}
		if ok && b.remember(stage, sig, display) {
			changed = true
		}
	}
	if changed {
		b.dropRetired()
	}
	return changed
}

func (b *Book) remember(stage, sig, display string) bool {
	for i := range b.Lessons {
		if b.Lessons[i].Stage == stage && b.Lessons[i].Signature == sig {
			b.Lessons[i].Seen++
			return true
		}
	}
	item := Lesson{Stage: stage, Signature: sig, Rule: display, Seen: 1}
	if len(b.Lessons) < maxLessons {
		b.Lessons = append(b.Lessons, item)
		return true
	}
	worst := 0
	for i := 1; i < len(b.Lessons); i++ {
		if worse(b.Lessons[i], b.Lessons[worst]) {
			worst = i
		}
	}
	if b.Lessons[worst].Seen > 1 || score(b.Lessons[worst]) > 0 {
		return false
	}
	b.Lessons[worst] = item
	return true
}

func (b *Book) credit(stage, sig string, helped bool) bool {
	for i := range b.Lessons {
		if b.Lessons[i].Stage != stage || b.Lessons[i].Signature != sig {
			continue
		}
		if helped {
			b.Lessons[i].Helpful++
		} else {
			b.Lessons[i].Harmful++
		}
		return true
	}
	return false
}

func (b *Book) dropRetired() {
	kept := make([]Lesson, 0, len(b.Lessons))
	for _, lesson := range b.Lessons {
		if retired(lesson) {
			continue
		}
		kept = append(kept, lesson)
	}
	b.Lessons = kept
}

func splitReason(reason string) (display, sig string, ok bool) {
	display = strings.Join(strings.Fields(strings.TrimSpace(reason)), " ")
	sig = strings.ToLower(display)
	if !memorable(sig) {
		return "", "", false
	}
	return display, sig, true
}

func memorable(sig string) bool {
	if sig == "" || len(sig) > maxRule {
		return false
	}
	return !strings.HasPrefix(sig, "open questions:")
}

func score(lesson Lesson) int {
	return lesson.Helpful - lesson.Harmful
}

func retired(lesson Lesson) bool {
	return lesson.Harmful >= 2 && lesson.Helpful == 0
}

func worse(a, b Lesson) bool {
	if score(a) != score(b) {
		return score(a) < score(b)
	}
	return a.Seen < b.Seen
}
