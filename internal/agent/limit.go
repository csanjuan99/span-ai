package agent

import (
	"errors"
	"fmt"
	"strings"
)

// The two ways a CLI cannot work while the project is fine.
const (
	Quota       = "quota"
	Credentials = "credentials"
)

// quotaLines are phrases a CLI prints when the account ran out of room. They are
// phrases, not status codes: a bare "429" also appears inside a diff or a test log.
var quotaLines = []string{
	"usage limit",
	"rate limit",
	"rate_limit",
	"ratelimit",
	"too many requests",
	"quota exceeded",
	"exceeded your current quota",
	"insufficient quota",
	"insufficient credits",
	"credit balance",
	"weekly limit",
	"monthly limit",
	"token limit",
	"out of tokens",
	"limit reached",
}

// credentialLines are phrases a CLI prints when it has no usable key.
var credentialLines = []string{
	"no api key",
	"api key not found",
	"missing api key",
	"invalid api key",
	"api key is invalid",
	"incorrect api key",
	"not logged in",
	"please log in",
	"login required",
	"authentication_error",
	"authentication failed",
	"failed to authenticate",
	"oauth session expired",
	"session expired",
	"token expired",
	"could not be refreshed",
	"unauthorized",
	"invalid bearer",
	"credentials not found",
	"no credentials",
}

// Blocked is an agent that cannot work until a person acts. It is not a failure of
// the change: the same prompt succeeds once the account can run again.
type Blocked struct {
	Agent  string
	Kind   string
	Detail string
	Err    error
}

func (b *Blocked) Error() string {
	if b.Kind == Credentials {
		return fmt.Sprintf("%s: credentials missing or rejected: %s", b.Agent, b.Detail)
	}
	return fmt.Sprintf("%s: quota exhausted: %s", b.Agent, b.Detail)
}

func (b *Blocked) Unwrap() error { return b.Err }

// AsBlocked reports whether an error, wrapped or not, is a blocked agent.
func AsBlocked(err error) (*Blocked, bool) {
	var blocked *Blocked
	if errors.As(err, &blocked) {
		return blocked, true
	}
	return nil, false
}

// Classify names the reason a CLI stopped working, and keeps the line that says so.
// The answer is deterministic: no model, no extra process, no network.
func Classify(text string) (kind, detail string, ok bool) {
	lower := strings.ToLower(text)
	for _, line := range strings.Split(lower, "\n") {
		if hit := match(line, quotaLines); hit != "" {
			return Quota, lineFor(text, lower, hit), true
		}
		if hit := match(line, credentialLines); hit != "" {
			return Credentials, lineFor(text, lower, hit), true
		}
	}
	return "", "", false
}

func match(line string, needles []string) string {
	for _, needle := range needles {
		if strings.Contains(line, needle) {
			return needle
		}
	}
	return ""
}

// lineFor returns the original line that matched, trimmed and capped. The lowercased
// copy exists only for matching, so the reader sees the CLI's own words.
func lineFor(text, lower, needle string) string {
	at := strings.Index(lower, needle)
	if at < 0 {
		return needle
	}
	start := strings.LastIndex(lower[:at], "\n") + 1
	end := strings.Index(lower[at:], "\n")
	if end < 0 {
		end = len(text)
	} else {
		end += at
	}
	line := strings.TrimSpace(text[start:end])
	if len(line) > 240 {
		line = line[:240] + "…"
	}
	return line
}
