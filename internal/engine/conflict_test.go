package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"span/internal/statement"
)

// fakeGit is a source that can integrate the base. Each Integrate returns the next
// scripted result; once they run out the base is already in the branch.
type fakeGit struct {
	*fakeSource
	integrations []Integration
	integrated   int
	merging      bool
	resolved     []map[string]FileResolution
	resolveErr   error
	finishes     int
	aborts       int
	messages     []string
	// preflights are returned in order; once they run out the base merges cleanly.
	preflights []Preflight
	checked    int
	// restored is what Restore reports changed outside the kept paths, call by call.
	restored  [][]string
	snapshots int
	keeps     [][]string
	// pushedAt is how many check commands had run at each push.
	pushedAt     []int
	check        *scriptCheck
	wrongParents bool
	tampered     error
}

func (g *fakeGit) Preflight(context.Context, string) (Preflight, error) {
	g.checked++
	if len(g.preflights) == 0 {
		return Preflight{Checked: true, Base: "origin/dev", BaseSHA: "dev-sha"}, nil
	}
	next := g.preflights[0]
	g.preflights = g.preflights[1:]
	return next, nil
}
func (g *fakeGit) Snapshot(context.Context) (TreeState, error) {
	g.snapshots++
	return TreeState{}, nil
}
func (g *fakeGit) Restore(_ context.Context, _ TreeState, keep []string) (Restored, error) {
	g.keeps = append(g.keeps, keep)
	if len(g.restored) == 0 {
		return Restored{}, nil
	}
	next := g.restored[0]
	g.restored = g.restored[1:]
	return Restored{Paths: next}, nil
}
func (g *fakeGit) PinRepo(context.Context) (RepoPin, error) {
	return RepoPin{WorkTree: "/work", GitDir: "/work/.git", CommonDir: "/work/.git"}, nil
}
func (g *fakeGit) Verify(RepoPin) error       { return g.tampered }
func (g *fakeGit) WithPin(RepoPin) Integrator { return g }
func (g *fakeGit) Parents(context.Context) ([]string, error) {
	if g.wrongParents {
		return []string{"agent-sha"}, nil
	}
	return []string{"head-sha", "dev-sha"}, nil
}
func (g *fakeGit) Push(ctx context.Context) error {
	if g.check != nil {
		g.pushedAt = append(g.pushedAt, len(g.check.calls))
	}
	return g.fakeSource.Push(ctx)
}

func (g *fakeGit) Integrate(context.Context, string) (Integration, error) {
	g.integrated++
	if len(g.integrations) == 0 {
		return Integration{Clean: true, Base: "origin/dev", BaseSHA: "dev-sha"}, nil
	}
	next := g.integrations[0]
	g.integrations = g.integrations[1:]
	g.merging = len(next.Files) > 0 || next.Merged
	return next, nil
}
func (g *fakeGit) Merging(context.Context) (bool, error) { return g.merging, nil }
func (g *fakeGit) Abort(context.Context) error {
	g.aborts++
	g.merging = false
	return nil
}
func (g *fakeGit) Resolve(_ context.Context, files map[string]FileResolution) error {
	g.resolved = append(g.resolved, files)
	return g.resolveErr
}
func (g *fakeGit) FinishMerge(_ context.Context, message string) (string, error) {
	g.finishes++
	g.messages = append(g.messages, message)
	g.merging = false
	g.sha = fmt.Sprintf("merge-%d", g.finishes)
	return g.sha, nil
}
func (g *fakeGit) Preview(_ context.Context, path string, before FileSide, res FileResolution) (string, error) {
	return fmt.Sprintf("--- a/%s (%s)\n+++ b/%s %s\n", path, before.SHA, path, string(res.Content)+res.Blob), nil
}

// proposeAgents answers the conflict prompt with scripted replies.
type proposeAgents struct {
	*fakeAgents
	replies  []ConflictReply
	proposed []string
}

func (p *proposeAgents) Propose(_ context.Context, _, _, text string) (ConflictReply, error) {
	p.proposed = append(p.proposed, text)
	if len(p.replies) == 0 {
		return ConflictReply{}, errors.New("no reply scripted")
	}
	next := p.replies[0]
	if len(p.replies) > 1 {
		p.replies = p.replies[1:]
	}
	return next, nil
}

// refusingReview refuses every merge with err.
type refusingReview struct {
	*fakeReview
	err error
}

func (r *refusingReview) Merge(context.Context, string) error { return r.err }

type events struct{ list []Event }

func (e *events) Publish(_ string, event Event) { e.list = append(e.list, event) }

func (e *events) conflicts() []*Conflict {
	var out []*Conflict
	for _, event := range e.list {
		if event.Type == "conflict" {
			out = append(out, event.Conflict)
		}
	}
	return out
}

func conflictedApp() Integration {
	return Integration{Base: "origin/dev", BaseSHA: "dev-sha", Files: []ConflictFile{{
		Path: "app.go", Kind: "content",
		Ours:   FileSide{Present: true, SHA: "ours-sha", Mode: "100644", Text: "const Name = \"ours\"\n"},
		Theirs: FileSide{Present: true, SHA: "theirs-sha", Mode: "100644", Text: "const Name = \"theirs\"\n"},
		Base:   FileSide{Present: true, SHA: "base-sha", Mode: "100644", Text: "const Name = \"base\"\n"},
	}}}
}

func combineReply(risk string) ConflictReply {
	return ConflictReply{Groups: []ConflictReplyGroup{{
		ID: "g1", Kind: "content", Files: []string{"app.go"}, Summary: "both renamed the constant",
		Options: []ConflictReplyOption{
			{ID: "both", Title: "Keep both names", Summary: "combine", Risk: risk, Action: "combine", Content: "const Name = \"ours\"\nconst Other = \"theirs\"\n"},
			{ID: "mine", Title: "Keep this branch", Summary: "ours", Risk: "medium", Action: "ours"},
		},
	}}}
}

type conflictRig struct {
	opt    Options
	git    *fakeGit
	agents *proposeAgents
	review *fakeReview
	check  *scriptCheck
	events *events
	id     string
}

func newConflictRig(t *testing.T, integrations ...Integration) *conflictRig {
	t.Helper()
	opt, agents, source, review, _, check := baseOpt(t)
	git := &fakeGit{fakeSource: source, integrations: integrations, check: check}
	for _, integration := range integrations {
		if len(integration.Files) > 0 {
			// The preflight sees the conflict the integration then reports.
			git.preflights = []Preflight{{Checked: true, Conflicts: true, Base: "origin/dev", BaseSHA: "dev-sha"}}
			break
		}
	}
	source.dirty = false
	source.sha = "head-sha"
	proposer := &proposeAgents{fakeAgents: agents}
	ev := &events{}
	opt.Source = git
	opt.Agents = proposer
	opt.Watch = ev
	review.found = true
	review.url = "https://example.test/pull/1"
	id := writeCursor(t, opt, Cursor{
		ID: "span-c", Iteration: 1, Stage: merge, GitSHA: "head-sha", ChangeURL: review.url,
		Branch: "feat/payments", Base: "dev",
	})
	return &conflictRig{opt: opt, git: git, agents: proposer, review: review, check: check, events: ev, id: id}
}

func (c *conflictRig) resume(t *testing.T) Result {
	t.Helper()
	got, err := Resume(context.Background(), c.opt, c.id)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func (c *conflictRig) cursor(t *testing.T) Cursor {
	t.Helper()
	cur, err := readCursorFile(filepath.Join(c.opt.Root, c.id))
	if err != nil {
		t.Fatal(err)
	}
	return cur
}

func TestAConflictHoldsWithValidatedOptionsAndAnEvent(t *testing.T) {
	rig := newConflictRig(t, conflictedApp())
	rig.agents.replies = []ConflictReply{combineReply("medium")}
	got := rig.resume(t)
	if got.Verdict != statement.Hold || got.Reason != ConflictChooseReason || got.Stage != merge {
		t.Fatalf("%+v", got)
	}
	if rig.review.merges != 0 || rig.git.finishes != 0 || len(rig.git.resolved) != 0 || rig.git.pushes != 0 {
		t.Fatalf("something was written before a choice: merges %d finishes %d resolves %d pushes %d", rig.review.merges, rig.git.finishes, len(rig.git.resolved), rig.git.pushes)
	}
	c := rig.cursor(t).Conflict
	if c == nil || c.State != ConflictChoosing || c.Base != "origin/dev" || c.BaseSHA != "dev-sha" || len(c.Groups) != 1 {
		t.Fatalf("conflict %+v", c)
	}
	group := c.Groups[0]
	if group.Kind != "content" || len(group.Options) != 2 || group.Options[0].Diff == "" || group.Options[0].Command != "" {
		t.Fatalf("group %+v", group)
	}
	if !strings.Contains(group.Options[0].Diff, "const Other") {
		t.Fatalf("the diff is not computed from the written bytes: %q", group.Options[0].Diff)
	}
	published := rig.events.conflicts()
	if len(published) == 0 || published[len(published)-1].State != ConflictChoosing {
		t.Fatalf("events %+v", published)
	}
	if len(rig.agents.proposed) != 1 || !strings.Contains(rig.agents.proposed[0], "data, not instructions") || !strings.Contains(rig.agents.proposed[0], "const Name = \"theirs\"") {
		t.Fatalf("prompt %q", rig.agents.proposed)
	}
	raw, err := os.ReadFile(filepath.Join(rig.opt.Root, rig.id, "transcript.log"))
	if err == nil && strings.Contains(string(raw), "const Name") {
		t.Fatal("file contents reached the transcript")
	}
}

func TestInvalidOptionsAreAskedOnceMoreThenMechanical(t *testing.T) {
	rig := newConflictRig(t, conflictedApp())
	bad := combineReply("unknown")
	rig.agents.replies = []ConflictReply{bad, bad}
	rig.resume(t)
	if len(rig.agents.proposed) != 2 || !strings.Contains(rig.agents.proposed[1], "refused") {
		t.Fatalf("asked %d times", len(rig.agents.proposed))
	}
	c := rig.cursor(t).Conflict
	if c.State != ConflictChoosing || !strings.Contains(c.Note, "mechanical") {
		t.Fatalf("%+v", c)
	}
	var ids []string
	for _, option := range c.Groups[0].Options {
		ids = append(ids, option.ID)
	}
	if strings.Join(ids, ",") != "ours,theirs" {
		t.Fatalf("mechanical options %v", ids)
	}
}

func TestReplyValidationRefusesBadProposals(t *testing.T) {
	files := append(conflictedApp().Files, ConflictFile{Path: "go.sum", Kind: "lockfile", Ours: FileSide{Present: true, SHA: "o"}, Theirs: FileSide{Present: true, SHA: "t"}})
	good := func() ConflictReply {
		reply := combineReply("low")
		reply.Groups = append(reply.Groups, ConflictReplyGroup{ID: "g2", Files: []string{"go.sum"}, Options: []ConflictReplyOption{
			{ID: "regen", Title: "Regenerate", Risk: "low", Action: "regenerate", Command: "go mod tidy"},
			{ID: "mine", Title: "Mine", Risk: "medium", Action: "ours"},
		}})
		return reply
	}
	if _, _, err := validateReply(good(), files); err != nil {
		t.Fatalf("a valid reply was refused: %v", err)
	}
	cases := map[string]func(*ConflictReply){
		"missing file": func(r *ConflictReply) { r.Groups = r.Groups[:1] },
		"unknown file": func(r *ConflictReply) { r.Groups[0].Files = append(r.Groups[0].Files, "other.go") },
		"one option":   func(r *ConflictReply) { r.Groups[0].Options = r.Groups[0].Options[:1] },
		"five options": func(r *ConflictReply) {
			o := r.Groups[0].Options[1]
			r.Groups[0].Options = append(r.Groups[0].Options, o, o, o)
		},
		"bad risk":   func(r *ConflictReply) { r.Groups[0].Options[0].Risk = "none" },
		"bad action": func(r *ConflictReply) { r.Groups[0].Options[0].Action = "rebase" },
		"markers":    func(r *ConflictReply) { r.Groups[0].Options[0].Content = "<<<<<<< HEAD\na\n=======\nb\n>>>>>>> dev\n" },
		"regenerate content": func(r *ConflictReply) {
			r.Groups[0].Options[1] = ConflictReplyOption{ID: "x", Title: "x", Risk: "low", Action: "regenerate", Command: "go mod tidy"}
		},
		"shell command":        func(r *ConflictReply) { r.Groups[1].Options[0].Command = "go mod tidy; rm -rf ~" },
		"empty command":        func(r *ConflictReply) { r.Groups[1].Options[0].Command = " " },
		"unlisted program":     func(r *ConflictReply) { r.Groups[1].Options[0].Command = "rm -rf node_modules" },
		"program by path":      func(r *ConflictReply) { r.Groups[1].Options[0].Command = "./go mod tidy" },
		"newline in command":   func(r *ConflictReply) { r.Groups[1].Options[0].Command = "go mod tidy\nrm x" },
		"duplicate group":      func(r *ConflictReply) { r.Groups[1].ID = "g1" },
		"bad group id":         func(r *ConflictReply) { r.Groups[0].ID = "../g" },
		"file in two groups":   func(r *ConflictReply) { r.Groups[1].Files = append(r.Groups[1].Files, "app.go") },
		"combine with nothing": func(r *ConflictReply) { r.Groups[0].Options[0].Content = "" },
	}
	for name, change := range cases {
		reply := good()
		change(&reply)
		if _, _, err := validateReply(reply, files); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestChoosingAppliesCommitsPushesAndProvesAgain(t *testing.T) {
	rig := newConflictRig(t, conflictedApp())
	rig.agents.replies = []ConflictReply{combineReply("medium")}
	rig.resume(t)
	if err := ChooseConflict(rig.opt.Root, rig.id, map[string]string{"g1": "both"}); err != nil {
		t.Fatal(err)
	}
	proves := len(rig.check.calls)
	got := rig.resume(t)
	if got.Verdict != statement.Advance || got.Stage != note {
		t.Fatalf("%+v", got)
	}
	if rig.git.finishes != 1 || len(rig.git.resolved) != 1 || rig.review.merges != 1 {
		t.Fatalf("finishes %d resolves %d merges %d", rig.git.finishes, len(rig.git.resolved), rig.review.merges)
	}
	written := rig.git.resolved[0]["app.go"]
	if !written.Write || !strings.Contains(string(written.Content), "const Other") {
		t.Fatalf("resolution %+v", written)
	}
	if rig.git.messages[0] != "merge: bring origin/dev into feat/payments (resolve conflicts)" {
		t.Fatalf("message %q", rig.git.messages[0])
	}
	if rig.git.pushes != 1 || len(rig.git.pushedAt) != 1 || rig.git.pushedAt[0] <= proves {
		t.Fatalf("the merge commit must be pushed once, after prove ran on it: pushes %d at %v (prove calls before %d)", rig.git.pushes, rig.git.pushedAt, proves)
	}
	if len(rig.check.calls) <= proves {
		t.Fatal("prove did not run again on the merged head")
	}
	cur := rig.cursor(t)
	if cur.Conflict != nil || cur.Iteration != 2 || cur.GitSHA != "merge-1" {
		t.Fatalf("cursor %+v", cur)
	}
	prove2, ok, _ := statement.Read(filepath.Join(rig.opt.Root, rig.id, "iter", "2"), proveSt)
	if !ok || prove2.GitSHA != "merge-1" {
		t.Fatalf("prove statement %+v", prove2)
	}
	old, _, _ := statement.Read(filepath.Join(rig.opt.Root, rig.id, "iter", "1"), merge)
	if !strings.Contains(old.Reason, "resolved the conflict") || !strings.Contains(old.Reason, "Keep both names") {
		t.Fatalf("merge note %q", old.Reason)
	}
	for _, name := range []string{conflictChoiceFile, conflictPlanFile} {
		if _, err := os.Stat(filepath.Join(rig.opt.Root, rig.id, name)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("%s left behind", name)
		}
	}
}

func TestAChoiceMustNameEveryGroupWithAKnownOption(t *testing.T) {
	rig := newConflictRig(t, conflictedApp())
	rig.agents.replies = []ConflictReply{combineReply("medium")}
	if err := ChooseConflict(rig.opt.Root, rig.id, map[string]string{"g1": "both"}); !errors.Is(err, ErrNoConflict) {
		t.Fatalf("a choice before any conflict: %v", err)
	}
	rig.resume(t)
	for _, choices := range []map[string]string{{}, {"g1": "nope"}, {"g1": "both", "g9": "x"}} {
		if err := ChooseConflict(rig.opt.Root, rig.id, choices); err == nil {
			t.Fatalf("%v accepted", choices)
		}
	}
}

func TestAHintProposesAgainAtMostThreeTimes(t *testing.T) {
	rig := newConflictRig(t, conflictedApp())
	rig.agents.replies = []ConflictReply{combineReply("medium")}
	rig.resume(t)
	for i := 0; i < 3; i++ {
		if err := HintConflict(rig.opt.Root, rig.id, "keep the base's name"); err != nil {
			t.Fatal(err)
		}
		got := rig.resume(t)
		if got.Reason != ConflictChooseReason {
			t.Fatalf("%+v", got)
		}
	}
	if len(rig.agents.proposed) != 3 {
		t.Fatalf("proposed %d times, want 3", len(rig.agents.proposed))
	}
	if !strings.Contains(rig.agents.proposed[1], "keep the base's name") {
		t.Fatal("the hint is not in the prompt")
	}
	c := rig.cursor(t).Conflict
	if !strings.Contains(c.Note, "mechanical") || c.Groups[0].Options[0].ID != "ours" {
		t.Fatalf("after three proposals: %+v", c)
	}
	if err := HintConflict(rig.opt.Root, rig.id, "   "); err == nil {
		t.Fatal("an empty hint was accepted")
	}
}

func TestAbortUndoesTheMergeAndHolds(t *testing.T) {
	rig := newConflictRig(t, conflictedApp())
	rig.agents.replies = []ConflictReply{combineReply("medium")}
	rig.resume(t)
	if err := AbortConflict(rig.opt.Root, rig.id); err != nil {
		t.Fatal(err)
	}
	got := rig.resume(t)
	if got.Verdict != statement.Hold || got.Reason != ConflictAbortedReason {
		t.Fatalf("%+v", got)
	}
	cur := rig.cursor(t)
	if rig.git.aborts != 1 || rig.git.merging || cur.Conflict != nil || cur.GitSHA != "head-sha" || rig.git.pushes != 0 {
		t.Fatalf("aborts %d merging %v cursor %+v", rig.git.aborts, rig.git.merging, cur)
	}
}

func TestAutoAppliesOnlyWhenEveryGroupIsLowRisk(t *testing.T) {
	rig := newConflictRig(t, conflictedApp())
	rig.opt.Policy.Stages.Conflict = "auto"
	rig.agents.replies = []ConflictReply{combineReply("low")}
	got := rig.resume(t)
	if got.Verdict != statement.Advance || rig.git.finishes != 1 {
		t.Fatalf("%+v finishes %d", got, rig.git.finishes)
	}

	risky := newConflictRig(t, conflictedApp())
	risky.opt.Policy.Stages.Conflict = "auto"
	risky.agents.replies = []ConflictReply{combineReply("medium")}
	if got := risky.resume(t); got.Reason != ConflictChooseReason || risky.git.finishes != 0 {
		t.Fatalf("a medium-risk option was applied: %+v", got)
	}

	held := newConflictRig(t, conflictedApp())
	held.agents.replies = []ConflictReply{combineReply("low")}
	if got := held.resume(t); got.Reason != ConflictChooseReason || held.git.finishes != 0 {
		t.Fatalf("hold applied a choice: %+v", got)
	}
}

func TestRegenerateRunsOnlyAfterItIsChosen(t *testing.T) {
	lock := Integration{Base: "origin/dev", BaseSHA: "dev-sha", Files: []ConflictFile{{
		Path: "go.sum", Kind: "lockfile",
		Ours: FileSide{Present: true, SHA: "o", Mode: "100644"}, Theirs: FileSide{Present: true, SHA: "t", Mode: "100644"},
	}}}
	rig := newConflictRig(t, lock)
	rig.opt.Policy.Stages.Conflict = "auto"
	rig.agents.replies = []ConflictReply{{Groups: []ConflictReplyGroup{{ID: "deps", Files: []string{"go.sum"}, Options: []ConflictReplyOption{
		{ID: "tidy", Title: "Regenerate", Risk: "low", Action: "regenerate", Command: "go mod tidy"},
		{ID: "mine", Title: "Mine", Risk: "low", Action: "ours"},
	}}}}}
	rig.resume(t)
	if contains(rig.check.calls, "go mod tidy") {
		t.Fatal("the regenerate command ran before anyone chose it")
	}
	c := rig.cursor(t).Conflict
	if c.State != ConflictChoosing || c.Groups[0].Options[0].Command != "go mod tidy" || c.Groups[0].Kind != "lockfile" {
		t.Fatalf("%+v", c)
	}
	if err := ChooseConflict(rig.opt.Root, rig.id, map[string]string{"deps": "tidy"}); err != nil {
		t.Fatal(err)
	}
	rig.resume(t)
	if !contains(rig.check.calls, "go mod tidy") || rig.git.finishes != 1 {
		t.Fatalf("calls %v finishes %d", rig.check.calls, rig.git.finishes)
	}
	last := rig.git.resolved[len(rig.git.resolved)-1]
	if !last["go.sum"].Stage {
		t.Fatalf("the regenerated file was not staged: %+v", rig.git.resolved)
	}
}

func TestAHostConflictRefusalEntersTheSamePath(t *testing.T) {
	rig := newConflictRig(t, conflictedApp())
	rig.git.preflights = nil
	rig.opt.Review = &refusingReview{fakeReview: rig.review, err: fmt.Errorf("%w: gh pr merge: not mergeable", ErrConflict)}
	rig.agents.replies = []ConflictReply{combineReply("medium")}
	got := rig.resume(t)
	if got.Reason != ConflictChooseReason || rig.git.integrated != 1 {
		t.Fatalf("%+v integrated %d", got, rig.git.integrated)
	}
	if cur := rig.cursor(t); cur.Pending != nil || strings.Contains(cur.Reason, "unknown") {
		t.Fatalf("cursor %+v", cur)
	}
}

func TestAHostConflictWithoutAnIntegratorIsNotAnUnknownOutcome(t *testing.T) {
	opt, _, _, review, _, _ := baseOpt(t)
	opt.Review = &refusingReview{fakeReview: review, err: fmt.Errorf("%w: not mergeable", ErrConflict)}
	id := writeCursor(t, opt, Cursor{ID: "span-n", Iteration: 1, Stage: merge, ChangeURL: "https://example.test/pull/1", Base: "dev"})
	got, err := Resume(context.Background(), opt, id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Hold || !strings.HasPrefix(got.Reason, "merge conflict: ") {
		t.Fatalf("%+v", got)
	}
}

func TestABranchMerelyBehindItsBaseMergesAsBefore(t *testing.T) {
	rig := newConflictRig(t)
	got := rig.resume(t)
	if got.Verdict != statement.Advance || rig.git.checked != 1 || rig.git.integrated != 0 || rig.git.finishes != 0 || rig.git.pushes != 0 || rig.review.merges != 1 {
		t.Fatalf("%+v checked %d integrated %d finishes %d pushes %d merges %d", got, rig.git.checked, rig.git.integrated, rig.git.finishes, rig.git.pushes, rig.review.merges)
	}
	if cur := rig.cursor(t); cur.Iteration != 1 {
		t.Fatalf("a clean base sent the run back to prove: %+v", cur)
	}
}

func TestAnUncheckedBaseLeavesTheMergeToTheHost(t *testing.T) {
	rig := newConflictRig(t)
	rig.git.preflights = []Preflight{{Note: "the base dev could not be fetched from origin"}}
	got := rig.resume(t)
	if got.Verdict != statement.Advance || rig.review.merges != 1 || rig.git.integrated != 0 {
		t.Fatalf("%+v", got)
	}
	st, _, _ := statement.Read(filepath.Join(rig.opt.Root, rig.id, "iter", "1"), merge)
	if !strings.Contains(st.Reason, "could not be fetched") {
		t.Fatalf("merge note %q", st.Reason)
	}
}

func TestAPreflightConflictThatMergesCleanlyIsAbortedAndMerged(t *testing.T) {
	rig := newConflictRig(t, Integration{Clean: true, Merged: true, Base: "origin/dev"})
	rig.git.preflights = []Preflight{{Checked: true, Conflicts: true, Base: "origin/dev"}}
	got := rig.resume(t)
	if got.Verdict != statement.Advance || rig.git.finishes != 0 || rig.git.aborts != 1 || rig.review.merges != 1 {
		t.Fatalf("%+v finishes %d aborts %d merges %d", got, rig.git.finishes, rig.git.aborts, rig.review.merges)
	}
}

func TestResolutionsAreCappedPerRun(t *testing.T) {
	rig := newConflictRig(t, conflictedApp())
	cur := rig.cursor(t)
	cur.Resolutions = 3
	if err := writeCursorFile(filepath.Join(rig.opt.Root, rig.id), cur); err != nil {
		t.Fatal(err)
	}
	got := rig.resume(t)
	if got.Verdict != statement.Hold || got.Reason != "conflict: too many resolutions; the base keeps moving" || rig.git.integrated != 0 {
		t.Fatalf("%+v integrated %d", got, rig.git.integrated)
	}
}

func TestAFailedProveNeverPublishesTheMergeCommit(t *testing.T) {
	rig := newConflictRig(t, conflictedApp())
	rig.agents.replies = []ConflictReply{combineReply("medium")}
	rig.resume(t)
	if err := ChooseConflict(rig.opt.Root, rig.id, map[string]string{"g1": "both"}); err != nil {
		t.Fatal(err)
	}
	rig.check.fail = 20
	got := rig.resume(t)
	if got.Verdict != statement.Hold || rig.git.finishes != 1 {
		t.Fatalf("%+v finishes %d", got, rig.git.finishes)
	}
	if rig.git.pushes != 0 || rig.review.merges != 0 {
		t.Fatalf("pushed %d merged %d after a failing prove", rig.git.pushes, rig.review.merges)
	}
}

func TestAMergeHeldForApprovalAsksAgainForTheMergedHead(t *testing.T) {
	rig := newConflictRig(t, conflictedApp())
	rig.opt.Policy.Stages.Merge = "hold"
	rig.agents.replies = []ConflictReply{combineReply("medium")}
	cur := rig.cursor(t)
	cur.Hold = true
	if err := writeCursorFile(filepath.Join(rig.opt.Root, rig.id), cur); err != nil {
		t.Fatal(err)
	}
	if err := Approve(rig.opt.Root, rig.id, merge, ""); err != nil {
		t.Fatal(err)
	}
	if got := rig.resume(t); got.Reason != ConflictChooseReason {
		t.Fatalf("%+v", got)
	}
	if err := ChooseConflict(rig.opt.Root, rig.id, map[string]string{"g1": "both"}); err != nil {
		t.Fatal(err)
	}
	got := rig.resume(t)
	if got.Reason != "merge is hold" || rig.git.pushes != 0 || rig.review.merges != 0 {
		t.Fatalf("the merged head was published without an approval: %+v pushes %d", got, rig.git.pushes)
	}
	if err := Approve(rig.opt.Root, rig.id, merge, ""); err != nil {
		t.Fatal(err)
	}
	got = rig.resume(t)
	if got.Verdict != statement.Advance || rig.git.pushes != 1 || rig.review.merges != 1 {
		t.Fatalf("%+v pushes %d merges %d", got, rig.git.pushes, rig.review.merges)
	}
}

func TestEffectiveRiskIgnoresALowLabelOnALossyOption(t *testing.T) {
	rig := newConflictRig(t, conflictedApp())
	rig.opt.Policy.Stages.Conflict = "auto"
	reply := combineReply("low")
	reply.Groups[0].Options = []ConflictReplyOption{
		{ID: "drop", Title: "Delete it", Risk: "low", Action: "delete"},
		{ID: "base", Title: "Take the base", Risk: "low", Action: "theirs"},
		{ID: "mine", Title: "Keep mine", Risk: "low", Action: "ours"},
	}
	rig.agents.replies = []ConflictReply{reply}
	got := rig.resume(t)
	if got.Reason != ConflictChooseReason || rig.git.finishes != 0 {
		t.Fatalf("a delete labelled low was applied: %+v", got)
	}
	var risks []string
	for _, option := range rig.cursor(t).Conflict.Groups[0].Options {
		risks = append(risks, option.Risk)
	}
	if strings.Join(risks, ",") != "high,high,medium" {
		t.Fatalf("card risks %v", risks)
	}
}

func TestStrayAgentEditsAreRestoredAndNoted(t *testing.T) {
	rig := newConflictRig(t, conflictedApp())
	rig.agents.replies = []ConflictReply{combineReply("medium")}
	rig.git.restored = [][]string{{"b.txt", "d.txt"}}
	rig.resume(t)
	if rig.git.snapshots == 0 || len(rig.git.keeps) == 0 {
		t.Fatal("the tree was not checked after the agent ran")
	}
	c := rig.cursor(t).Conflict
	if c.State != ConflictChoosing || !strings.Contains(c.Note, "outside the conflict") || !strings.Contains(c.Note, "b.txt") {
		t.Fatalf("%+v", c)
	}
}

func TestARegenerateThatTouchesOtherFilesIsRefused(t *testing.T) {
	lock := Integration{Base: "origin/dev", BaseSHA: "dev-sha", Files: []ConflictFile{{
		Path: "go.sum", Kind: "lockfile",
		Ours: FileSide{Present: true, SHA: "o", Mode: "100644"}, Theirs: FileSide{Present: true, SHA: "t", Mode: "100644"},
	}}}
	rig := newConflictRig(t, lock)
	rig.agents.replies = []ConflictReply{{Groups: []ConflictReplyGroup{{ID: "deps", Files: []string{"go.sum"}, Options: []ConflictReplyOption{
		{ID: "tidy", Title: "Regenerate", Risk: "low", Action: "regenerate", Command: "go mod tidy"},
		{ID: "mine", Title: "Mine", Risk: "low", Action: "ours"},
	}}}}}
	rig.resume(t)
	rig.git.restored = [][]string{{"go.mod"}}
	if err := ChooseConflict(rig.opt.Root, rig.id, map[string]string{"deps": "tidy"}); err != nil {
		t.Fatal(err)
	}
	got := rig.resume(t)
	c := rig.cursor(t).Conflict
	if got.Reason != ConflictChooseReason || rig.git.finishes != 0 || c == nil || !strings.Contains(c.Note, "go.mod") {
		t.Fatalf("%+v %+v", got, c)
	}
	if keep := rig.git.keeps[len(rig.git.keeps)-1]; strings.Join(keep, ",") != "go.sum" {
		t.Fatalf("the command's own files were not kept: %v", keep)
	}
}

func TestATypeConflictOffersOnlyOneSideOrTheOther(t *testing.T) {
	files := []ConflictFile{{Path: "f", Kind: "other", Ours: FileSide{Present: true, SHA: "l", Mode: "120000"}, Theirs: FileSide{Present: true, SHA: "t", Mode: "100644"}, Temps: []string{"f~abc1234"}}}
	reply := ConflictReply{Groups: []ConflictReplyGroup{{ID: "g1", Files: []string{"f"}, Options: []ConflictReplyOption{
		{ID: "both", Title: "Combine", Risk: "low", Action: "combine", Content: "x\n"},
		{ID: "mine", Title: "Mine", Risk: "low", Action: "ours"},
	}}}}
	if _, _, err := validateReply(reply, files); err == nil {
		t.Fatal("combine was accepted for a type conflict")
	}
	reply.Groups[0].Options[0] = ConflictReplyOption{ID: "base", Title: "Base", Risk: "low", Action: "theirs"}
	_, options, err := validateReply(reply, files)
	if err != nil {
		t.Fatal(err)
	}
	if res := options["g1"]["mine"].Files["f"]; res.Mode != "120000" || len(res.Remove) != 1 {
		t.Fatalf("ours %+v", res)
	}
}

func TestASecondWakeWhileApplyingDoesNotCommitTwice(t *testing.T) {
	rig := newConflictRig(t, conflictedApp())
	rig.agents.replies = []ConflictReply{combineReply("medium")}
	rig.resume(t)
	// The first wake committed the merge and died before it saved the cursor.
	cur := rig.cursor(t)
	cur.Conflict.State = ConflictApplying
	cur.Conflict.Groups[0].Chosen = "both"
	if err := writeCursorFile(filepath.Join(rig.opt.Root, rig.id), cur); err != nil {
		t.Fatal(err)
	}
	rig.git.merging = false
	rig.git.sha = "committed-sha"
	got := rig.resume(t)
	if got.Verdict != statement.Advance || rig.git.finishes != 0 || len(rig.git.resolved) != 0 || rig.git.pushes == 0 {
		t.Fatalf("%+v finishes %d resolves %d pushes %d", got, rig.git.finishes, len(rig.git.resolved), rig.git.pushes)
	}
}

func TestAFailedPushIsPushedAgainOnResume(t *testing.T) {
	rig := newConflictRig(t, conflictedApp())
	rig.agents.replies = []ConflictReply{combineReply("medium")}
	rig.resume(t)
	if err := ChooseConflict(rig.opt.Root, rig.id, map[string]string{"g1": "both"}); err != nil {
		t.Fatal(err)
	}
	push := &failingPush{fakeGit: rig.git, left: 1}
	rig.opt.Source = push
	got := rig.resume(t)
	if got.Verdict != statement.Hold || !strings.Contains(got.Reason, "not pushed") || rig.review.merges != 0 {
		t.Fatalf("%+v", got)
	}
	got = rig.resume(t)
	if got.Verdict != statement.Advance || rig.git.finishes != 1 || rig.git.pushes != 1 || rig.review.merges != 1 {
		t.Fatalf("%+v finishes %d pushes %d", got, rig.git.finishes, rig.git.pushes)
	}
}

type failingPush struct {
	*fakeGit
	left int
}

func (f *failingPush) Push(ctx context.Context) error {
	if f.left > 0 {
		f.left--
		return errors.New("rejected")
	}
	return f.fakeGit.Push(ctx)
}

func TestRewindRefusesWhileAConflictIsOpen(t *testing.T) {
	rig := newConflictRig(t, conflictedApp())
	rig.agents.replies = []ConflictReply{combineReply("medium")}
	rig.resume(t)
	if err := Rewind(rig.opt.Root, rig.id, build); err == nil || !strings.Contains(err.Error(), "conflict") {
		t.Fatalf("err = %v", err)
	}
}

func TestTheConflictJSONMatchesTheContract(t *testing.T) {
	raw, err := json.Marshal(Event{Type: "conflict", Conflict: &Conflict{
		Base: "origin/dev", BaseSHA: "s", State: ConflictChoosing, Groups: []ConflictGroup{{
			ID: "g1", Kind: "lockfile", Files: []string{"go.sum"}, Summary: "x", Chosen: "o1",
			Options: []ConflictOption{{ID: "o1", Title: "t", Summary: "s", Risk: "low", Action: "regenerate", Command: "go mod tidy", Diff: "d"}},
		}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"type":"conflict","conflict":{"base":"origin/dev","baseSHA":"s","state":"choosing","note":"","groups":[{"id":"g1","kind":"lockfile","files":["go.sum"],"summary":"x","options":[{"id":"o1","title":"t","summary":"s","risk":"low","action":"regenerate","command":"go mod tidy","diff":"d"}],"chosen":"o1"}]}}`
	if string(raw) != want {
		t.Fatalf("got  %s\nwant %s", raw, want)
	}
}

func TestConflictMarkersNeedAllThreeLinesInOrder(t *testing.T) {
	for text, want := range map[string]bool{
		"<<<<<<< HEAD\na\n=======\nb\n>>>>>>> dev\n":     true,
		"<<<<<<< HEAD\r\na\r\n=======\r\nb\r\n>>>>>>> x": true,
		"a quoted line\n>>>>>>> dev\n":                   false,
		"<<<<<<< HEAD\nonly a start\n":                   false,
		"Title\n=======\n":                               false,
		">>>>>>> x\n=======\n<<<<<<< y\n":                false,
		"<<<<<<< a\n=======\x00\n>>>>>>> b\n":            false,
	} {
		if got := HasConflictMarkers([]byte(text)); got != want {
			t.Errorf("HasConflictMarkers(%q) = %v, want %v", text, got, want)
		}
	}
}

func TestABehindRefusalIsAKnownOutcome(t *testing.T) {
	rig := newConflictRig(t)
	rig.opt.Review = &refusingReview{fakeReview: rig.review, err: fmt.Errorf("%w: not up to date with the base branch", ErrBehind)}
	got := rig.resume(t)
	if got.Verdict != statement.Hold || got.Reason != BehindReason {
		t.Fatalf("%+v", got)
	}
	if cur := rig.cursor(t); cur.Pending != nil {
		t.Fatalf("pending %+v", cur.Pending)
	}
}

func TestAMergeWhoseParentsAreNotSpansIsRefused(t *testing.T) {
	rig := newConflictRig(t, conflictedApp())
	rig.agents.replies = []ConflictReply{combineReply("medium")}
	rig.resume(t)
	if err := ChooseConflict(rig.opt.Root, rig.id, map[string]string{"g1": "both"}); err != nil {
		t.Fatal(err)
	}
	rig.git.wrongParents = true
	got := rig.resume(t)
	c := rig.cursor(t).Conflict
	if got.Verdict != statement.Hold || c == nil || c.State != ConflictFailed || !strings.Contains(c.Note, "not the merge span prepared") {
		t.Fatalf("%+v %+v", got, c)
	}
	if rig.git.pushes != 0 || rig.review.merges != 0 {
		t.Fatal("a foreign commit was pushed or merged")
	}
}

func TestATamperedRepositoryFailsClosedAndAbortRunsNoGit(t *testing.T) {
	rig := newConflictRig(t, conflictedApp())
	rig.agents.replies = []ConflictReply{combineReply("medium")}
	rig.resume(t)
	rig.git.tampered = &TamperedError{Path: "/work/.git"}
	if err := AbortConflict(rig.opt.Root, rig.id); err != nil {
		t.Fatal(err)
	}
	got := rig.resume(t)
	want := "conflict: the agent changed the repository structure; span did not run git. Inspect /work/.git by hand"
	if got.Reason != want || rig.git.aborts != 0 {
		t.Fatalf("%+v aborts %d", got, rig.git.aborts)
	}
	if c := rig.cursor(t).Conflict; c == nil || c.State != ConflictFailed {
		t.Fatalf("%+v", c)
	}
}
