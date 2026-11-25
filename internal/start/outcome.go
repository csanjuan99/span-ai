package start

import (
	"errors"
	"fmt"
	"strings"
)

// NumberPartial means some repositories started and others did not.
const NumberPartial Number = 19

// RepoResult is how one repository answered a start.
type RepoResult struct {
	Repo    string   `json:"repo"`
	OK      bool     `json:"ok"`
	ID      string   `json:"id,omitempty"`
	Branch  string   `json:"branch,omitempty"`
	Stage   string   `json:"stage,omitempty"`
	Refusal *Refusal `json:"refusal,omitempty"`
}

// FromError maps a launch failure to a refusal when possible.
func FromError(err error) *Refusal {
	if err == nil {
		return nil
	}
	var ref Refusal
	if errors.As(err, &ref) {
		return &ref
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "no_repo") || strings.Contains(msg, "not a git"):
		r := refuse(NumberNotGit, "not_git", "%s", msg)
		return &r
	case strings.Contains(msg, "does not exist on origin") || strings.Contains(msg, "base branch"):
		r := refuse(NumberBaseMissing, "base_missing", "%s", msg)
		return &r
	case strings.Contains(msg, "already belongs") || strings.Contains(msg, "run_exists"):
		r := refuse(NumberRunExists, "run_exists", "%s", msg)
		return &r
	case strings.Contains(msg, "busy") || strings.Contains(msg, "checkout"):
		r := refuse(NumberCheckoutBusy, "checkout_busy", "%s", msg)
		return &r
	case strings.Contains(msg, "not configured"):
		r := refuse(NumberJiraSignedOut, "jira_signed_out", "%s", msg)
		return &r
	case strings.Contains(msg, "unauthorized") || strings.Contains(msg, "401"):
		r := refuse(NumberTokenRefused, "token_refused", "%s", msg)
		return &r
	case strings.Contains(msg, "no agent"):
		r := refuse(NumberNoAgent, "no_agent", "%s", msg)
		return &r
	default:
		r := refuse(0, "invalid", "%s", msg)
		return &r
	}
}

// ExitCode returns the process exit code for a refusal or partial start.
func ExitCode(ref *Refusal, partial bool) int {
	if partial {
		return int(NumberPartial)
	}
	if ref == nil {
		return 0
	}
	if ref.Number != 0 {
		return int(ref.Number)
	}
	return 3
}

// FormatNotes joins per-repo errors for partial starts.
func FormatNotes(errs []error) string {
	var b strings.Builder
	for _, err := range errs {
		if err == nil {
			continue
		}
		fmt.Fprintf(&b, "%s\n", err)
	}
	return strings.TrimSpace(b.String())
}
