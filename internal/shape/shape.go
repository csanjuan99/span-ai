// Package shape builds a user story through a guided interview. A session keeps the
// example map (rules, examples, open questions) the agent proposes each turn, and Span,
// not the agent, decides what is true: a question the person answered stays answered, at
// most three new questions arrive per turn, every fact names its source, the repository is
// checked for changes after every turn, and a sealed story never changes again.
package shape

import (
	"fmt"
	"time"
)

// Session statuses.
const (
	// StatusInterviewing means questions are open or the story is not drafted yet.
	StatusInterviewing = "interviewing"
	// StatusReady means no question is open and a story is drafted.
	StatusReady = "ready"
	// StatusSplit means the map holds more than one outcome and a split is proposed.
	StatusSplit = "split"
	// StatusSealed means the story and its digest are stored and the session is frozen.
	StatusSealed = "sealed"
)

// Question statuses.
const (
	// QuestionOpen waits for the person.
	QuestionOpen = "open"
	// QuestionAnswered was answered by the person.
	QuestionAnswered = "answered"
	// QuestionAssumed was closed by accepting the question's proposed assumption.
	QuestionAssumed = "assumed"
)

// Proposal statuses.
const (
	// ProposalProposed waits for the person to accept it.
	ProposalProposed = "proposed"
	// ProposalAccepted was written to the product context file.
	ProposalAccepted = "accepted"
	// ProposalSkipped was dropped by the person; it can be brought back until the session is sealed.
	ProposalSkipped = "skipped"
)

// Source kinds: where a rule or example comes from.
const (
	// SourcePrompt is the idea as the person wrote it.
	SourcePrompt = "prompt"
	// SourceAnswer is an answer the person gave.
	SourceAnswer = "answer"
	// SourceCode is a path in the repository.
	SourceCode = "code"
	// SourceContext is the product context file.
	SourceContext = "context"
)

// Source says where a fact comes from.
type Source struct {
	Kind string `json:"kind"`
	Ref  string `json:"ref"`
}

// Rule is one business rule of the story.
type Rule struct {
	ID     string `json:"id"`
	Text   string `json:"text"`
	Source Source `json:"source"`
}

// Example is a concrete case of a rule.
type Example struct {
	ID     string `json:"id"`
	Rule   string `json:"rule"`
	Text   string `json:"text"`
	Source Source `json:"source"`
}

// Question is something only the person can settle. Assume is what Span writes down when
// the person accepts the assumption instead of answering.
type Question struct {
	ID     string `json:"id"`
	Text   string `json:"text"`
	Blocks string `json:"blocks"`
	Assume string `json:"assume"`
	Status string `json:"status"`
	Answer string `json:"answer"`
	// AnswerID names the answer (A1, A2, ...) once the question is answered or assumed. It
	// stays when an assumption is replaced by an answer, so sources that cite it keep working.
	AnswerID string `json:"answerId"`
}

// Map is the example map: rules, the examples that pin them, and the questions left.
type Map struct {
	Rules     []Rule     `json:"rules"`
	Examples  []Example  `json:"examples"`
	Questions []Question `json:"questions"`
}

// Split is one story of a proposed split, with what it leaves for another story.
type Split struct {
	Story   string `json:"story"`
	LeftOut string `json:"leftOut"`
}

// Proposal is an addition to the product context file the person may accept.
type Proposal struct {
	ID      string `json:"id"`
	Section string `json:"section"`
	Text    string `json:"text"`
	Status  string `json:"status"`
}

// Turn records one agent run and the questions it added.
type Turn struct {
	N     int       `json:"n"`
	At    time.Time `json:"at"`
	Asked []string  `json:"asked"`
}

// JiraRef is the issue created for a sealed story.
type JiraRef struct {
	Key    string `json:"key"`
	URL    string `json:"url"`
	Status string `json:"status"`
}

// Failure kinds: why an interview turn stopped.
const (
	// FailureUsageLimit means the agent CLI reported a usage, rate or quota limit.
	FailureUsageLimit = "usage_limit"
	// FailureRepoChanged means the agent changed the repository during a read-only turn.
	FailureRepoChanged = "repo_changed"
	// FailureError is any other failure of the agent or of its reply.
	FailureError = "error"
)

// Failure says why the last interview turn stopped. It stays on the session until a turn
// succeeds, so the page and the CLI can tell what was kept and offer to resume.
type Failure struct {
	At      time.Time `json:"at"`
	Agent   string    `json:"agent"`
	Kind    string    `json:"kind"`
	Message string    `json:"message"`
}

// Session is one shaping conversation, stored as session.json.
type Session struct {
	ID        string     `json:"id"`
	Workspace string     `json:"workspace"`
	Repo      string     `json:"repo"`
	Idea      string     `json:"idea"`
	Agent     string     `json:"agent"`
	Model     string     `json:"model"`
	Status    string     `json:"status"`
	Map       Map        `json:"map"`
	Split     []Split    `json:"split"`
	Proposals []Proposal `json:"proposals"`
	Turns     []Turn     `json:"turns"`
	Story     string     `json:"story"`
	Digest    string     `json:"digest"`
	Jira      *JiraRef   `json:"jira,omitempty"`
	// Parent is the session this one was split from, or forked from once it was sealed.
	Parent string `json:"parent,omitempty"`
	// Children are the sessions an accepted split created, in the order of the split.
	Children []string `json:"children,omitempty"`
	// MergedInto is the session this one was merged into; this one stays readable and links to it.
	MergedInto string `json:"mergedInto,omitempty"`
	// SplitKept records that the person chose to keep the story whole: no split is proposed again.
	SplitKept bool `json:"splitKept,omitempty"`
	// SealedAt is when the story was sealed.
	SealedAt *time.Time `json:"sealedAt,omitempty"`
	// UpdatedAt is stamped by every Save.
	UpdatedAt time.Time `json:"updatedAt"`
	// Archived hides the session from the default list; it never changes the story.
	Archived bool `json:"archived"`
	// SourceIssue is the Jira key the session started from, when it did.
	SourceIssue string `json:"sourceIssue,omitempty"`
	// Failure is why the last turn stopped, until a turn succeeds.
	Failure *Failure `json:"failure,omitempty"`
}

// Error is a failure the HTTP layer answers with its Code.
type Error struct {
	Code string
	Msg  string
}

// Error is the code, then the message when there is one.
func (e *Error) Error() string {
	if e.Msg == "" {
		return e.Code
	}
	return e.Code + ": " + e.Msg
}

// SpanCode is the error code the HTTP layer answers with.
func (e *Error) SpanCode() string { return e.Code }

func fail(code, format string, args ...any) error {
	return &Error{Code: code, Msg: fmt.Sprintf(format, args...)}
}
