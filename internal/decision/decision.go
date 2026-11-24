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
	"span/internal/overlay"
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

// Effects on the decision log.
const (
	// Reanchor keeps the stale decisions, now taken against the current story.
	Reanchor = "reanchor"
	// Drop retires the stale decisions. They stay in the log.
	Drop = "drop"
)

// What choosing an option does to the run besides moving it.
const (
	// Record keeps the answer as a decision every later prompt carries.
	Record = "record"
	// Approve releases the plan gate so frame is not asked again.
	Approve = "approve"
)

// Classes of request.
const (
	CommitRejected = blocker.CommitRejected
	// Conflict is a brief that asks for what a rule of the repository forbids.
	Conflict = "spec-conflict"
	// PlanReview is the plan frame proposes, held for a person before any build.
	PlanReview = "plan-review"
	// Stale is decisions taken against a story that has changed since.
	Stale = "overlay-stale"
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
	// Effect is what Span does to the run's decision log when the option is chosen. Only
	// Span sets it: an agent's request never carries one.
	Effect string `json:"effect,omitempty"`
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
	// Subject says what the answer is about, so the logged decision names it.
	Subject string `json:"subject,omitempty"`
	// Covers are the keys of the conflicts this request puts to a person, so an answered
	// one is recognized when frame words it differently.
	Covers []string  `json:"covers,omitempty"`
	Time   time.Time `json:"time"`
}

// Answer is what a person chose, with any words of their own.
type Answer struct {
	// ID is the decision the answer was given for. A page that shows an older decision must
	// not answer a newer one that happens to share an option id.
	ID     string `json:"id"`
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
	if answer.ID != "" && answer.ID != r.ID {
		return Option{}, errors.New("this decision was replaced by another one; reload the page")
	}
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

// Bounds on what an agent may put in a request. The card is read by a person and the words
// of a chosen option are replayed into prompts, so neither may grow or span lines.
const (
	maxLabel       = 200
	maxHappened    = 2000
	maxConsequence = 500
	maxEvidence    = 10
	maxExcerpt     = 4000
	// MaxFile is the largest request file Span reads.
	MaxFile = 64 * 1024
)

// oneLine collapses every run of whitespace, newlines included, into one space.
func oneLine(text string) string { return strings.Join(strings.Fields(text), " ") }

func clip(text string, limit int) string {
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return string(runes[:limit])
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
	req.Title = clip(oneLine(req.Title), maxLabel)
	req.Happened = clip(strings.TrimSpace(req.Happened), maxHappened)
	req.Recommendation = oneLine(req.Recommendation)
	if len(req.Evidence) > maxEvidence {
		req.Evidence = req.Evidence[:maxEvidence]
	}
	for i := range req.Evidence {
		req.Evidence[i].Ref = clip(oneLine(req.Evidence[i].Ref), maxLabel)
		req.Evidence[i].Excerpt = clip(req.Evidence[i].Excerpt, maxExcerpt)
	}
	for i := range req.Options {
		o := &req.Options[i]
		o.ID, o.Label = oneLine(o.ID), clip(oneLine(o.Label), maxLabel)
		o.Consequence = clip(oneLine(o.Consequence), maxConsequence)
		o.Effect = ""
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

// FromStale is the question a changed story puts about the decisions taken on the old one.
func FromStale(stale []overlay.Entry, stage string, now time.Time) Request {
	var evidence []Evidence
	keys := []string{Stale}
	for _, e := range stale {
		evidence = append(evidence, Evidence{Ref: fmt.Sprintf("decision D%d, taken at %s", e.Seq, e.Stage), Excerpt: e.Text})
		keys = append(keys, fmt.Sprint(e.Seq))
	}
	dedupe := statement.Signature(keys...)
	return Request{
		ID: "stale-" + dedupe[:12], Class: Stale, Stage: stage, Time: now, Dedupe: dedupe,
		Title:    "The story changed after you decided",
		Happened: "The story was read again and it is not the one these decisions were taken against. Span applies neither silently nor drops them: say whether they still hold.",
		Evidence: evidence,
		Options: []Option{
			{ID: "keep", Label: "They still hold", Consequence: "The decisions go on top of the new story, as before.", Reversible: true, Then: Retry, Effect: Reanchor},
			{ID: "drop", Label: "Drop them, the new story says what is needed", Consequence: "The decisions stop reaching the agent. They stay in the log.", Reversible: true, Then: Retry, Effect: Drop},
			{ID: "stop", Label: "Stop this run", Consequence: "The run ends so you can reconsider.", Reversible: false, Then: Stop},
		},
	}
}

// Clash is one place where the story asks for what a rule of the repository forbids. The
// frame agent finds it; Span decides what to do about it.
type Clash struct {
	Story string `json:"story"`
	Rule  string `json:"rule"`
	Ref   string `json:"ref"`
}

// ClashKey identifies the rule a story breaks. The line frame points at is steadier than
// the words it uses, so it wins; without one the rule's own text stands in.
func ClashKey(c Clash) string {
	if ref := strings.ToLower(oneLine(c.Ref)); ref != "" {
		return ref
	}
	return strings.ToLower(oneLine(c.Rule))
}

// FromClashes is the question a story that contradicts the repository's rules puts to a
// person. Span never picks a side: the options are Span's, not the agent's, and the one
// that follows the rule is recommended because the rule is enforced and the story is not.
func FromClashes(clashes []Clash, stage string, now time.Time) Request {
	var evidence []Evidence
	keys := []string{Conflict}
	var covers, subjects []string
	for _, c := range clashes {
		evidence = append(evidence,
			Evidence{Ref: "story", Excerpt: c.Story},
			Evidence{Ref: c.Ref, Excerpt: c.Rule},
		)
		key := ClashKey(c)
		keys = append(keys, key)
		covers = append(covers, key)
		subjects = append(subjects, clip(oneLine(c.Rule), maxLabel)+" ("+oneLine(c.Ref)+")")
	}
	dedupe := statement.Signature(keys...)
	return Request{
		ID: "clash-" + dedupe[:12], Class: Conflict, Stage: stage, Time: now, Dedupe: dedupe,
		Covers: covers, Subject: "about " + strings.Join(subjects, "; "),
		Title:    "The story asks for something the repository's rules forbid",
		Happened: fmt.Sprintf("Frame found %d place(s) where the story and a rule of the repository disagree. Span does not pick a side silently.", len(clashes)),
		Evidence: evidence,
		Options: []Option{
			{ID: "follow-rule", Label: "Follow the repository rule", Consequence: "Frame and build follow the rule. Where the story says otherwise, the feature document says what is not met as written.", Reversible: true, Then: Retry, Effect: Record},
			{ID: "follow-story", Label: "Follow the story, I will change the rule", Consequence: "Build follows the story. A hook that enforces the rule can still reject the commit, and you will be asked then.", Reversible: true, Then: Retry, Effect: Record},
			{ID: "stop", Label: "Stop this run", Consequence: "The run ends so the story or the rule can be fixed first.", Reversible: false, Then: Stop},
		},
		Recommendation: "follow-rule",
	}
}

// FromPlan is the plan frame proposes, put in front of a person before anything is built.
func FromPlan(claims, assumptions []string, criteria int, stage string, now time.Time) Request {
	var evidence []Evidence
	for i, claim := range claims {
		evidence = append(evidence, Evidence{Ref: fmt.Sprintf("claim %d", i+1), Excerpt: claim})
	}
	for i, assumption := range assumptions {
		evidence = append(evidence, Evidence{Ref: fmt.Sprintf("assumption %d", i+1), Excerpt: assumption})
	}
	dedupe := statement.Signature(append([]string{PlanReview}, claims...)...)
	return Request{
		ID: "plan-" + dedupe[:12], Class: PlanReview, Stage: stage, Time: now, Dedupe: dedupe,
		Title:    "Review the plan before it is built",
		Happened: fmt.Sprintf("Frame read the story as %d claim(s), %d assumption(s) and %d acceptance criteria. Nothing is built until you say so.", len(claims), len(assumptions), criteria),
		Evidence: evidence,
		Options: []Option{
			{ID: "approve", Label: "Build this plan", Consequence: "The build starts from these claims. Frame does not run again.", Reversible: true, Then: Retry, Effect: Approve},
			{ID: "revise", Label: "Revise it with my words", Consequence: "Frame reads your words and proposes the plan again, and you review that one.", Reversible: true, Then: Retry, Effect: Record},
			{ID: "stop", Label: "Stop this run", Consequence: "The run ends.", Reversible: false, Then: Stop},
		},
		Recommendation: "approve",
	}
}
