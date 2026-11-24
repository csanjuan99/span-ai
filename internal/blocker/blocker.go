// Package blocker names why a stage cannot move forward and whether trying again can help.
// The class comes from deterministic signals, never from a model: a model may explain a
// blocker, it never decides whether the run retries.
package blocker

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"span/internal/statement"
)

// CommitRejected is a commit the repository's own hook refused.
const CommitRejected = "commit-rejected"

// Blocker is a classified obstacle kept on the run until something changes.
type Blocker struct {
	Class string `json:"class"`
	// Fingerprint is the stable identity of the failure output: the same rejection twice
	// shares it, whatever the colors, spacing or case.
	Fingerprint string `json:"fingerprint"`
	// Count is how many times in a row this exact failure came back.
	Count int `json:"count"`
	// Evidence is the tail of what the repository said, without terminal colors.
	Evidence string `json:"evidence"`
}

// Retryable reports whether running the stage again can still change the outcome. The
// same failure twice means the agent did not fix it, and a third identical run would only
// spend the same money to reach the same wall.
func (b *Blocker) Retryable() bool {
	return b == nil || b.Count < 2
}

var ansi = regexp.MustCompile(`\x1b\[[0-9;?]*[A-Za-z]`)

// Clean removes terminal colors and carriage returns from tool output.
func Clean(text string) string {
	return strings.ReplaceAll(ansi.ReplaceAllString(text, ""), "\r", "")
}

// FromCommit classifies a failed commit. prev is the blocker already on the run: when the
// new output has the same fingerprint the count grows, otherwise it starts over.
func FromCommit(err error, prev *Blocker) *Blocker {
	text := Clean(err.Error())
	var kept []string
	for _, line := range strings.Split(text, "\n") {
		// The command echo carries the run id and the message, which never change what the
		// hook thinks of the code, so it must not decide whether two failures are the same.
		if strings.Contains(line, "git commit -m") {
			if _, rest, ok := strings.Cut(line, "exit status"); ok {
				line = strings.TrimLeft(rest, " 0123456789:")
			} else {
				continue
			}
		}
		if strings.TrimSpace(line) != "" {
			kept = append(kept, strings.TrimRight(line, " \t"))
		}
	}
	body := strings.Join(kept, "\n")
	b := &Blocker{
		Class:       CommitRejected,
		Fingerprint: statement.Signature(append([]string{CommitRejected}, kept...)...),
		Count:       1,
		Evidence:    tail(body, statement.TailBytes),
	}
	if prev != nil && prev.Class == b.Class && prev.Fingerprint == b.Fingerprint {
		b.Count = prev.Count + 1
	}
	return b
}

// Summary is one line a person reads first.
func (b *Blocker) Summary() string {
	if b.Retryable() {
		return "commit rejected by the repository's hook"
	}
	return "commit rejected by the repository's hook, with the same output again: running the build again cannot pass"
}

func tail(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	cut := text[len(text)-limit:]
	for len(cut) > 0 && !utf8.RuneStart(cut[0]) {
		cut = cut[1:]
	}
	return cut
}
