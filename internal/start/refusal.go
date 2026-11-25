// Package start plans and refuses a run before anything is written.
package start

import "fmt"

// Number is the refusal code the N4 boards show beside each case.
type Number int

const (
	// NumberRunExists: a run for this branch already exists.
	NumberRunExists Number = 6
	// NumberJiraSignedOut: the story comes from Jira and the folder has no session.
	NumberJiraSignedOut Number = 8
	// NumberNotGit: the folder is not a git repository.
	NumberNotGit Number = 14
	// NumberBaseMissing: the base branch is not in the repository.
	NumberBaseMissing Number = 15
	// NumberCheckoutBusy: another run holds the checkout.
	NumberCheckoutBusy Number = 16
	// NumberNoAgent: no agent CLI is ready.
	NumberNoAgent Number = 17
	// NumberTokenRefused: the server refused the access token.
	NumberTokenRefused Number = 18
)

// Refusal is a coded reason a run must not start yet.
type Refusal struct {
	Number  Number `json:"number"`
	Code    string `json:"code"`
	Message string `json:"message"`
	RunID   string `json:"runId,omitempty"`
	Branch  string `json:"branch,omitempty"`
	Agent   string `json:"agent,omitempty"`
	Base    string `json:"base,omitempty"`
}

// Error is the refusal's message, so a Refusal is an error.
func (r Refusal) Error() string { return r.Message }

// SpanCode is the stable error name scripts and HTTP use.
func (r Refusal) SpanCode() string { return r.Code }

func refuse(number Number, code, format string, args ...any) Refusal {
	return Refusal{Number: number, Code: code, Message: fmt.Sprintf(format, args...)}
}
