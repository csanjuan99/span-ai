// Package decision is the question Span puts to a person when a run cannot choose for
// itself: what happened, the evidence, and a few answers with their consequences. A
// request is data. The person answers it from the desk and the run resumes where the
// answer says, so nothing depends on a free-text question in a transcript.
package decision

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"span/internal/blocker"
	"span/internal/statement"
)

// What an answer does next. There is deliberately no value that skips a repository check:
// every way forward either changes the work, changes the rule in the repository, or ends.
const (
	// Build restarts the build with the answer as guidance.
	Build = "build"
	// Retry runs the stage again without a new agent run when the blocker is a check, for
	// a person who changed the repository since.
	Retry = "retry"
	// Stop ends the run.
	Stop = "stop"
)

// Classes of request.
const (
	CommitRejected = blocker.CommitRejected
	// Conflict is a brief that asks for what a rule of the repository forbids.
	Conflict = "spec-conflict"
	// Question is anything else the agent cannot decide alone.
	Question = "agent-question"
)

// Evidence is one quoted source: a file and line, or the output of a check.
type Evidence struct {
	Ref     string `json:"ref"`
	Excerpt string `json:"excerpt"`
}

// Option is one answer. Consequence says what Span does, in words a person can weigh.
type Option struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Consequence string `json:"consequence"`
	Reversible  bool   `json:"reversible"`
	Then        string `json:"then"`
}

// Request is one open question. Dedupe identifies the problem, not the occurrence, so the
// same conflict found twice is one question.
type Request struct {
	ID             string     `json:"id"`
	Class          string     `json:"class"`
	Stage          string     `json:"stage"`
	Title          string     `json:"title"`
	Happened       string     `json:"happened"`
	Evidence       []Evidence `json:"evidence,omitempty"`
	Options        []Option   `json:"options"`
	Recommendation string     `json:"recommendation,omitempty"`
	Dedupe         string     `json:"dedupe"`
	Time           time.Time  `json:"time"`
}

// Answer is what a person chose, with any words of their own.
type Answer struct {
	Option string `json:"option"`
	Text   string `json:"text"`
}

// Validate checks a request before a person ever sees it.
func (r Request) Validate() error {
	if strings.TrimSpace(r.Title) == "" || strings.TrimSpace(r.Happened) == "" {
		return errors.New("a decision needs a title and what happened")
	}
	if n := len(r.Options); n < 2 || n > 4 {
		return fmt.Errorf("a decision offers 2 to 4 options, not %d", n)
	}
	seen := map[string]bool{}
	for _, option := range r.Options {
		if strings.TrimSpace(option.ID) == "" || strings.TrimSpace(option.Label) == "" || strings.TrimSpace(option.Consequence) == "" {
			return errors.New("every option needs an id, a label and its consequence")
		}
		if seen[option.ID] {
			return fmt.Errorf("option %q appears twice", option.ID)
		}
		seen[option.ID] = true
		switch option.Then {
		case Build, Retry, Stop:
		default:
			return fmt.Errorf("option %q does something Span does not offer: %q", option.ID, option.Then)
		}
	}
	if r.Recommendation != "" && !seen[r.Recommendation] {
		return fmt.Errorf("the recommendation %q is not an option", r.Recommendation)
	}
	return nil
}

// Choose finds the option an answer names.
func (r Request) Choose(answer Answer) (Option, error) {
	for _, option := range r.Options {
		if option.ID == answer.Option {
			return option, nil
		}
	}
	return Option{}, fmt.Errorf("%q is not an option of this decision", answer.Option)
}

// FromCommit is the question a repeated hook rejection puts to a person.
func FromCommit(b *blocker.Blocker, stage string, now time.Time) Request {
	return Request{
		ID: "commit-" + b.Fingerprint[:12], Class: CommitRejected, Stage: stage, Time: now,
		Dedupe:   statement.Signature(CommitRejected, b.Fingerprint),
		Title:    "The repository's hook keeps rejecting the commit",
		Happened: fmt.Sprintf("The commit was refused %d times with the same output, so another build would end the same way. The hook enforces a rule of the repository; Span never skips it.", b.Count),
		Evidence: []Evidence{{Ref: "git commit", Excerpt: b.Evidence}},
		Options: []Option{
			{ID: "fix-code", Label: "Change the work to satisfy the hook", Consequence: "The build runs again with your words as guidance and the hook output in front of it.", Reversible: true, Then: Build},
			{ID: "rule-changed", Label: "I changed the rule or the hook in the repository", Consequence: "Span tries the commit again without running the agent.", Reversible: true, Then: Retry},
			{ID: "stop", Label: "Stop this run", Consequence: "The run ends. The branch and the changes stay as they are.", Reversible: false, Then: Stop},
		},
		Recommendation: "fix-code",
	}
}

// Parse reads a request the build agent wrote. The agent chooses the words; Span chooses
// the class, the id and the dedupe key, so a request cannot claim to be something else.
func Parse(raw []byte, stage string, now time.Time) (Request, error) {
	var req Request
	if err := json.Unmarshal(raw, &req); err != nil {
		return Request{}, fmt.Errorf("the decision file is not valid JSON: %w", err)
	}
	if req.Class != Conflict {
		req.Class = Question
	}
	req.Stage = stage
	req.Time = now
	var parts []string
	for _, e := range req.Evidence {
		parts = append(parts, e.Ref)
	}
	req.Dedupe = statement.Signature(append([]string{req.Class, req.Title}, parts...)...)
	req.ID = "agent-" + req.Dedupe[:12]
	if err := req.Validate(); err != nil {
		return Request{}, err
	}
	return req, nil
}
