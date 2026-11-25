package shape

import (
	"context"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"span/internal/jira"
)

// Sorts of a session list.
const (
	// SortActivity orders by the last change, newest first. It is the default.
	SortActivity = "activity"
	// SortCreated orders by creation time, newest first.
	SortCreated = "created"
)

// Archived modes of a session list.
const (
	// ArchivedInclude lists archived sessions with the others.
	ArchivedInclude = "include"
	// ArchivedOnly lists only archived sessions.
	ArchivedOnly = "only"
)

const titleLimit = 120

// Ref is the short name of a session, for people to say aloud: SES- and the random tail
// of its id.
func (s *Session) Ref() string {
	tail := s.ID
	if i := strings.LastIndex(tail, "-"); i >= 0 && i < len(tail)-1 {
		tail = tail[i+1:]
	}
	return "SES-" + strings.ToUpper(tail)
}

// StorySummary is the text of the first "# " heading of a story, or its first line.
func StorySummary(story string) string {
	first := ""
	for _, line := range strings.Split(story, "\n") {
		line = strings.TrimSpace(line)
		if title, ok := strings.CutPrefix(line, "# "); ok && strings.TrimSpace(title) != "" {
			return strings.TrimSpace(title)
		}
		if first == "" {
			first = line
		}
	}
	return first
}

// Title is the story heading once there is a story, else the first line of the idea.
func (s *Session) Title() string {
	if strings.TrimSpace(s.Story) != "" {
		return StorySummary(s.Story)
	}
	line, _, _ := strings.Cut(strings.TrimSpace(s.Idea), "\n")
	line = strings.TrimSpace(line)
	if utf8.RuneCountInString(line) > titleLimit {
		line = string([]rune(line)[:titleLimit-1]) + "…"
	}
	return line
}

// OpenQuestions counts the questions that wait for the person.
func (s *Session) OpenQuestions() int {
	return len(openIDs(s))
}

// JiraKey is the issue created for the story, else the issue the session started from.
func (s *Session) JiraKey() string {
	if s.Jira != nil && s.Jira.Key != "" {
		return s.Jira.Key
	}
	return s.SourceIssue
}

// Activity is when the session last changed: the last Save, or for a session saved before
// that was recorded, its last turn, or the creation time in its id.
func (s *Session) Activity() time.Time {
	if !s.UpdatedAt.IsZero() {
		return s.UpdatedAt
	}
	if n := len(s.Turns); n > 0 {
		return s.Turns[n-1].At
	}
	if len(s.ID) >= len(idTime) {
		if at, err := time.Parse(idTime, s.ID[:len(idTime)]); err == nil {
			return at
		}
	}
	return time.Time{}
}

// idTime is the layout of the creation time at the start of an id.
const idTime = "20060102-150405.000000000"

// Filter narrows and orders a list of sessions. Empty fields do not filter.
type Filter struct {
	// Status is one session status.
	Status string
	// Query matches the idea, title, id, ref, repo and Jira key, ignoring case.
	Query string
	// Repo is the exact repo of the session.
	Repo string
	// Agent is the agent of the session; "auto" matches a session that names none.
	Agent string
	// Sort is SortActivity (default) or SortCreated.
	Sort string
	// Archived is empty (hide archived), ArchivedInclude or ArchivedOnly.
	Archived string
}

// Apply returns the sessions that pass the filter, ordered. An unknown status, sort or
// archived mode is refused with the code invalid.
func (f Filter) Apply(in []*Session) ([]*Session, error) {
	if err := f.check(); err != nil {
		return nil, err
	}
	query := strings.ToLower(strings.TrimSpace(f.Query))
	out := make([]*Session, 0, len(in))
	for _, s := range in {
		if f.keeps(s, query) {
			out = append(out, s)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if f.Sort == SortCreated {
			return out[i].ID > out[j].ID
		}
		a, b := out[i].Activity(), out[j].Activity()
		if !a.Equal(b) {
			return a.After(b)
		}
		return out[i].ID > out[j].ID
	})
	return out, nil
}

func (f Filter) check() error {
	switch f.Status {
	case "", StatusInterviewing, StatusReady, StatusSplit, StatusSealed:
	default:
		return fail("invalid", "status %q is not interviewing, ready, split or sealed", f.Status)
	}
	switch f.Sort {
	case "", SortActivity, SortCreated:
	default:
		return fail("invalid", "sort %q is not activity or created", f.Sort)
	}
	switch f.Archived {
	case "", ArchivedInclude, ArchivedOnly:
	default:
		return fail("invalid", "archived %q is not include or only", f.Archived)
	}
	return nil
}

func (f Filter) keeps(s *Session, query string) bool {
	switch {
	case f.Archived == "" && s.Archived, f.Archived == ArchivedOnly && !s.Archived:
		return false
	case f.Status != "" && s.Status != f.Status:
		return false
	case f.Repo != "" && s.Repo != f.Repo:
		return false
	case f.Agent != "" && !sameAgent(f.Agent, s.Agent):
		return false
	}
	if query == "" {
		return true
	}
	for _, text := range []string{s.Idea, s.Title(), s.ID, s.Ref(), s.Repo, s.JiraKey()} {
		if strings.Contains(strings.ToLower(text), query) {
			return true
		}
	}
	return false
}

func sameAgent(want, have string) bool {
	if strings.EqualFold(want, "auto") {
		return have == "" || strings.EqualFold(have, "auto")
	}
	return strings.EqualFold(want, have)
}

// View is a session as the API and the CLI show it: every stored field, plus the fields
// derived from it, so a client never recomputes them.
type View struct {
	*Session
	Ref           string `json:"ref"`
	Title         string `json:"title"`
	OpenQuestions int    `json:"openQuestions"`
	JiraKey       string `json:"jiraKey"`
}

// View returns the session with its derived fields. UpdatedAt is the session's Activity, so
// a session saved before UpdatedAt existed still has a time.
func (s *Session) View() View {
	shown := *s
	shown.UpdatedAt = s.Activity()
	return View{Session: &shown, Ref: s.Ref(), Title: s.Title(), OpenQuestions: s.OpenQuestions(), JiraKey: s.JiraKey()}
}

// Views is View for each session of a list.
func Views(in []*Session) []View {
	out := make([]View, 0, len(in))
	for _, s := range in {
		out = append(out, s.View())
	}
	return out
}

// IssueReader reads one Jira issue; jira.Client does.
type IssueReader interface {
	Issue(ctx context.Context, key string) (jira.Story, error)
}

// IdeaFromIssue builds the idea of a session from a Jira issue: its brief, then the
// person's own words when they gave some. It returns the issue's key as Jira names it.
func IdeaFromIssue(ctx context.Context, reader IssueReader, key, extra string) (idea, issue string, err error) {
	story, err := reader.Issue(ctx, key)
	if err != nil {
		return "", "", err
	}
	idea = strings.TrimSpace(story.Brief)
	if idea == "" {
		idea = strings.TrimSpace(story.Summary)
	}
	if idea == "" {
		return "", "", fail("invalid", "issue %s has no text to start from", story.Key)
	}
	if extra = strings.TrimSpace(extra); extra != "" {
		idea += "\n\nMore context from the person:\n" + extra
	}
	return idea, story.Key, nil
}
