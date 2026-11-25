package shape

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestSaveStampsUpdatedAt(t *testing.T) {
	ws := gitRepo(t)
	s, err := Create(ws, StartInput{Idea: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if time.Since(mustLoad(t, ws, s.ID).UpdatedAt) > time.Minute {
		t.Fatalf("updatedAt not stamped: %v", mustLoad(t, ws, s.ID).UpdatedAt)
	}
}

func TestRefTitleAndOpenQuestions(t *testing.T) {
	s := &Session{
		ID:   "20261125-094428.123456789-ab12cd",
		Idea: "  Customers see why a delivery failed\nand retry it  ",
		Map:  Map{Questions: []Question{{ID: "Q1", Status: QuestionOpen}, {ID: "Q2", Status: QuestionAnswered}}},
	}
	if got := s.Ref(); got != "SES-AB12CD" {
		t.Fatalf("ref %q", got)
	}
	if got := s.Title(); got != "Customers see why a delivery failed" {
		t.Fatalf("title %q", got)
	}
	if got := s.OpenQuestions(); got != 1 {
		t.Fatalf("open %d", got)
	}
	s.Story = "intro\n# Show the failure reason\nbody"
	if got := s.Title(); got != "Show the failure reason" {
		t.Fatalf("story title %q", got)
	}
	s.Jira = &JiraRef{Key: "BILL-2031"}
	if got := s.JiraKey(); got != "BILL-2031" {
		t.Fatalf("jira key %q", got)
	}
	s.Jira = nil
	s.SourceIssue = "BILL-9"
	if got := s.JiraKey(); got != "BILL-9" {
		t.Fatalf("source key %q", got)
	}
}

func TestActivityFallsBackToTheLastTurnThenTheID(t *testing.T) {
	at := time.Date(2026, 11, 24, 10, 5, 0, 0, time.UTC)
	s := &Session{ID: "20261125-094428.123456789-ab12cd", Turns: []Turn{{N: 1, At: at}}}
	if got := s.Activity(); !got.Equal(at) {
		t.Fatalf("turn activity %v", got)
	}
	s.Turns = nil
	if got := s.Activity(); got.Format("2006-01-02 15:04:05") != "2026-11-25 09:44:28" {
		t.Fatalf("id activity %v", got)
	}
	s.UpdatedAt = at.Add(time.Hour)
	if got := s.Activity(); !got.Equal(at.Add(time.Hour)) {
		t.Fatalf("saved activity %v", got)
	}
}

func sample() []*Session {
	return []*Session{
		{ID: "20261125-100000.000000000-aaaaaa", Idea: "Export invoices as CSV", Repo: "api", Status: StatusReady, Agent: "codex", UpdatedAt: time.Date(2026, 11, 25, 10, 5, 0, 0, time.UTC)},
		{ID: "20261125-090000.000000000-bbbbbb", Idea: "Refund a charge", Repo: "web", Status: StatusSealed, Agent: "", Jira: &JiraRef{Key: "BILL-2031"}, UpdatedAt: time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)},
		{ID: "20261124-090000.000000000-cccccc", Idea: "Rate limit the API", Repo: "api", Status: StatusInterviewing, Agent: "claude-code", Archived: true, UpdatedAt: time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC)},
	}
}

func ids(in []*Session) string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		out = append(out, s.ID[9:15])
	}
	return strings.Join(out, ",")
}

func TestFilterHidesArchivedAndSortsByActivity(t *testing.T) {
	got, err := Filter{}.Apply(sample())
	if err != nil {
		t.Fatal(err)
	}
	if ids(got) != "090000,100000" {
		t.Fatalf("default %s", ids(got))
	}
}

func TestFilterByStatusRepoAgentAndArchived(t *testing.T) {
	cases := []struct {
		name string
		f    Filter
		want string
	}{
		{"status", Filter{Status: StatusSealed}, "090000"},
		{"repo", Filter{Repo: "api"}, "100000"},
		{"agent", Filter{Agent: "codex"}, "100000"},
		{"auto agent matches an empty one", Filter{Agent: "auto"}, "090000"},
		{"archived only", Filter{Archived: ArchivedOnly}, "090000"},
		{"archived included", Filter{Archived: ArchivedInclude, Sort: SortCreated}, "100000,090000,090000"},
	}
	for _, c := range cases {
		got, err := c.f.Apply(sample())
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if ids(got) != c.want {
			t.Errorf("%s: got %s want %s", c.name, ids(got), c.want)
		}
	}
}

func TestFilterQueryLooksAtIdeaIdRepoAndJiraKey(t *testing.T) {
	for query, want := range map[string]string{"CSV": "100000", "bbbbbb": "090000", "bill-2031": "090000", "web": "090000", "nothing": ""} {
		got, err := Filter{Query: query}.Apply(sample())
		if err != nil {
			t.Fatal(err)
		}
		if ids(got) != want {
			t.Errorf("query %q: got %s want %s", query, ids(got), want)
		}
	}
}

func TestFilterRefusesUnknownValues(t *testing.T) {
	for _, f := range []Filter{{Status: "done"}, {Sort: "random"}, {Archived: "maybe"}} {
		if _, err := f.Apply(sample()); codeOf(err) != "invalid" {
			t.Errorf("%+v: want invalid, got %v", f, err)
		}
	}
}

func TestAnswerFailureIsSavedOnTheSession(t *testing.T) {
	f := &fake{replies: []string{reply(t, turn([]map[string]any{q("Q1", "who?", "a")}, ""))}}
	e, ws := engine(t, f)
	s := start(t, e, ws)
	f.err = errors.New("claude-code: usage limit reached, resets at 18:00")
	if _, err := e.Answer(context.Background(), ws, s.ID, []Answer{{Question: "Q1", Text: "ops"}}, nil); err == nil {
		t.Fatal("want the runner error")
	}
	got := mustLoad(t, ws, s.ID).Failure
	if got == nil || got.Kind != FailureUsageLimit || !strings.Contains(got.Message, "resets at 18:00") {
		t.Fatalf("failure %+v", got)
	}
	if got.Agent != "claude" || got.At.IsZero() {
		t.Fatalf("failure agent/time %+v", got)
	}
}

func TestOtherFailuresAreKindError(t *testing.T) {
	f := &fake{replies: []string{reply(t, turn([]map[string]any{q("Q1", "who?", "a")}, ""))}}
	e, ws := engine(t, f)
	s := start(t, e, ws)
	f.err = errors.New("agent crashed")
	if _, err := e.Answer(context.Background(), ws, s.ID, []Answer{{Question: "Q1", Text: "ops"}}, nil); err == nil {
		t.Fatal("a crashed agent fails the turn")
	}
	if got := mustLoad(t, ws, s.ID).Failure; got == nil || got.Kind != FailureError {
		t.Fatalf("failure %+v", got)
	}
}

func TestResumeRunsTheNextTurnOnTheNewAgentAndClearsTheFailure(t *testing.T) {
	f := &fake{replies: []string{reply(t, turn([]map[string]any{q("Q1", "who?", "a")}, ""))}}
	e, ws := engine(t, f)
	s := start(t, e, ws)
	f.err = errors.New("usage limit")
	if _, err := e.Answer(context.Background(), ws, s.ID, []Answer{{Question: "Q1", Text: "ops"}}, nil); err == nil {
		t.Fatal("a crashed agent fails the turn")
	}
	f.err = nil
	f.replies = []string{reply(t, turn(nil, "# Story\nbody"))}
	got, err := e.Resume(context.Background(), ws, s.ID, "codex", "o3")
	if err != nil {
		t.Fatal(err)
	}
	if got.Failure != nil || got.Agent != "codex" || got.Model != "o3" || got.Status != StatusReady {
		t.Fatalf("resumed %+v", got)
	}
	if disk := mustLoad(t, ws, s.ID); disk.Failure != nil || disk.Agent != "codex" || len(disk.Turns) != 2 {
		t.Fatalf("disk %+v", disk)
	}
}

func TestResumeFailureKeepsTheNewAgentAndRecordsIt(t *testing.T) {
	f := &fake{replies: []string{reply(t, turn([]map[string]any{q("Q1", "who?", "a")}, ""))}}
	e, ws := engine(t, f)
	s := start(t, e, ws)
	f.err = errors.New("agent crashed")
	if _, err := e.Resume(context.Background(), ws, s.ID, "codex", ""); err == nil {
		t.Fatal("want the runner error")
	}
	disk := mustLoad(t, ws, s.ID)
	if disk.Agent != "codex" || disk.Failure == nil || disk.Failure.Agent != "codex" {
		t.Fatalf("disk %+v", disk)
	}
}

func TestSetAgentChangesOnlyTheAgentAndModel(t *testing.T) {
	f := &fake{replies: []string{reply(t, turn(nil, ""))}}
	e, ws := engine(t, f)
	s := start(t, e, ws)
	got, err := e.SetAgent(ws, s.ID, "codex", "")
	if err != nil || got.Agent != "codex" || got.Model != "" {
		t.Fatalf("set agent %+v %v", got, err)
	}
	if len(mustLoad(t, ws, s.ID).Turns) != 1 {
		t.Fatal("no turn must run")
	}
}

func TestArchiveWorksOnASealedSessionAndKeepsTheDigest(t *testing.T) {
	f := &fake{replies: []string{reply(t, turn(nil, "# Story\nbody"))}}
	e, ws := engine(t, f)
	s := start(t, e, ws)
	sealed, err := e.Seal(ws, s.ID)
	if err != nil {
		t.Fatal(err)
	}
	got, err := SetArchived(ws, s.ID, true)
	if err != nil || !got.Archived || got.Digest != sealed.Digest || got.Status != StatusSealed {
		t.Fatalf("archived %+v %v", got, err)
	}
	if got, err = SetArchived(ws, s.ID, false); err != nil || got.Archived {
		t.Fatalf("unarchived %+v %v", got, err)
	}
}

func TestDeleteRemovesAnOpenSessionAndRefusesASealedOne(t *testing.T) {
	f := &fake{replies: []string{reply(t, turn([]map[string]any{q("Q1", "who?", "a")}, ""))}}
	e, ws := engine(t, f)
	open := start(t, e, ws)
	if err := Delete(ws, open.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(ws, open.ID); codeOf(err) != "shape_not_found" {
		t.Fatalf("still there: %v", err)
	}
	f.replies = []string{reply(t, turn(nil, "# Story\nbody"))}
	sealed := start(t, e, ws)
	if _, err := e.Seal(ws, sealed.ID); err != nil {
		t.Fatal(err)
	}
	if err := Delete(ws, sealed.ID); codeOf(err) != "shape_sealed" {
		t.Fatalf("want shape_sealed, got %v", err)
	}
	if err := Delete(ws, "missing"); codeOf(err) != "shape_not_found" {
		t.Fatalf("want shape_not_found, got %v", err)
	}
}

func TestSourceIssueIsKept(t *testing.T) {
	f := &fake{replies: []string{reply(t, turn(nil, ""))}}
	e, ws := engine(t, f)
	s, err := e.Start(context.Background(), StartInput{Workspace: ws, Idea: "from jira", SourceIssue: "BILL-2031"})
	if err != nil || mustLoad(t, ws, s.ID).SourceIssue != "BILL-2031" {
		t.Fatalf("source issue: %+v %v", s, err)
	}
}
