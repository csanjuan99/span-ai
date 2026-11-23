package engine

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"span/internal/agent"
	"span/internal/handoff"
	"span/internal/policy"
	"span/internal/statement"
)

type fakeAgents struct {
	prompts    []string
	blocks     int
	seen       int
	builds     int
	skipDoc    bool
	questions  []string
	acceptance []string
	outOfScope bool
	frameErr   error
}

func (f *fakeAgents) Select(string) (agent.Selection, error) {
	return agent.Selection{Build: "cursor-agent", Inspect: "claude"}, nil
}

func (f *fakeAgents) Frame(_ context.Context, _, _ string, text string) (FrameDoc, error) {
	f.prompts = append(f.prompts, text)
	if f.frameErr != nil {
		return FrameDoc{}, f.frameErr
	}
	if f.outOfScope {
		return FrameDoc{OutOfScope: true}, nil
	}
	if len(f.questions) > 0 {
		questions := f.questions
		f.questions = nil
		return FrameDoc{Questions: questions, Acceptance: f.acceptance}, nil
	}
	return FrameDoc{Claims: []string{"charge the card"}, Acceptance: f.acceptance}, nil
}

func (f *fakeAgents) Build(_ context.Context, _, workspace, text string) error {
	f.builds++
	f.prompts = append(f.prompts, text)
	if f.skipDoc {
		return nil
	}
	dir := filepath.Join(workspace, "odd", "tasks")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	body := "# Payments\n\n## Specs\n\nS1. Charge the card.\n\n## Tasks\n\n- T1. S1.\n\n## Log\n\nL1. Charge the card.\n"
	return os.WriteFile(filepath.Join(dir, "payments.md"), []byte(body), 0o644)
}

func (f *fakeAgents) Inspect(context.Context, string, string, string) (InspectDoc, error) {
	f.seen++
	if f.seen <= f.blocks {
		return InspectDoc{Blocking: true, Reason: "missing spec"}, nil
	}
	return InspectDoc{Reason: "matches the spec"}, nil
}

type fakeSource struct {
	commits int
	pushes  int
	dirty   bool
	branch  string
	sha     string
}

func (s *fakeSource) Head(context.Context) (string, error) { return s.sha, nil }
func (s *fakeSource) Dirty(context.Context) (bool, error)  { return s.dirty, nil }
func (s *fakeSource) Commit(context.Context, string) (string, error) {
	s.commits++
	s.sha = "abc123"
	s.dirty = false
	return s.sha, nil
}
func (s *fakeSource) Push(context.Context) error { s.pushes++; return nil }
func (s *fakeSource) Branch(context.Context) (string, error) {
	if s.branch == "" {
		return "feat/payments", nil
	}
	return s.branch, nil
}
func (s *fakeSource) EnsureBranch(_ context.Context, base, branch string) error {
	s.branch = branch
	if base == "" {
		return errors.New("missing base")
	}
	return nil
}

type fakeReview struct {
	opens  int
	merges int
	found  bool
	url    string
	merged bool
	lookup error
}

func (r *fakeReview) Find(context.Context, string, string) (string, bool, error) {
	if r.lookup != nil {
		return "", false, r.lookup
	}
	return r.url, r.found, nil
}
func (r *fakeReview) Open(context.Context, string, string, string, string) (string, error) {
	r.opens++
	r.url = "https://example.test/pull/1"
	r.found = true
	return r.url, nil
}
func (r *fakeReview) Merged(context.Context, string) (bool, error) { return r.merged, nil }
func (r *fakeReview) Merge(context.Context, string) error {
	r.merges++
	r.merged = true
	return nil
}

type fakeRelease struct {
	calls []string
	err   error
}

func (r *fakeRelease) Run(_ context.Context, command, _, _ string) error {
	r.calls = append(r.calls, command)
	return r.err
}

type scriptCheck struct {
	calls []string
	fail  int
}

func (s *scriptCheck) Run(context.Context, string, string) (int, error) {
	s.calls = append(s.calls, "run")
	if s.fail > 0 {
		s.fail--
		return 1, nil
	}
	return 0, nil
}

func baseOpt(t *testing.T) (Options, *fakeAgents, *fakeSource, *fakeReview, *fakeRelease, *scriptCheck) {
	t.Helper()
	agents := &fakeAgents{}
	source := &fakeSource{dirty: true, sha: "base"}
	review := &fakeReview{}
	release := &fakeRelease{}
	check := &scriptCheck{}
	var tick int64
	opt := Options{
		Workspace: t.TempDir(),
		Root:      t.TempDir(),
		Policy: policy.File{
			Feature: "payments",
			Prove:   policy.Bands{Fast: []string{"go test ./..."}},
			Promote: policy.Ladder{Environments: []policy.Environment{{Name: "preview", Command: "echo preview"}}},
		},
		Brief:   "Charge the card.",
		Agents:  agents,
		Check:   check,
		Source:  source,
		Review:  review,
		Release: release,
		Now: func() time.Time {
			tick++
			return time.Unix(tick*60, 0).UTC()
		},
	}
	return opt, agents, source, review, release, check
}

func TestHappyPathPromotesOnce(t *testing.T) {
	opt, agents, source, review, release, _ := baseOpt(t)
	if err := os.MkdirAll(filepath.Join(opt.Workspace, "span", "stages"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(opt.Workspace, "span", "stages", "frame.md"), []byte("Prefer the ledger."), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Advance || got.Stage != note {
		t.Fatalf("%+v", got)
	}
	if review.opens != 1 || review.merges != 1 || len(release.calls) != 1 || source.commits != 1 {
		t.Fatalf("opens %d merges %d promote %#v commits %d", review.opens, review.merges, release.calls, source.commits)
	}
	if !contains(agents.prompts, "ODD is mandatory") {
		t.Fatal("build prompt did not require ODD")
	}
	frame, ok, err := statement.Read(filepath.Join(opt.Root, got.ID, "iter", "1"), frame)
	if err != nil || !ok || frame.InstructionsDigest == "" {
		t.Fatalf("frame digest ok=%v err=%v %+v", ok, err, frame)
	}
	noteSt, _, _ := statement.Read(filepath.Join(opt.Root, got.ID, "iter", "1"), note)
	if noteSt.Reason == "" || noteSt.ParentDigest == "" {
		t.Fatalf("note %+v", noteSt)
	}
}

func TestOpenWritesWorkspaceBeforeTheRun(t *testing.T) {
	opt, _, _, _, _, _ := baseOpt(t)
	id, err := Open(opt)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(opt.Root, id, "cursor.json"))
	if err != nil {
		t.Fatal(err)
	}
	var cur Cursor
	if err := json.Unmarshal(raw, &cur); err != nil {
		t.Fatal(err)
	}
	if cur.Workspace != opt.Workspace || cur.Stage != intake {
		t.Fatalf("%+v", cur)
	}
}

func TestStoryFilesAreOpenableOnTheFirstFrame(t *testing.T) {
	opt, agents, _, _, _, _ := baseOpt(t)
	opt.Files = []handoff.File{{Name: "design/board.png", Body: []byte("png-bytes")}}
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(opt.Root, got.ID, "handoffs", "board.png"))
	if err != nil || string(raw) != "png-bytes" {
		t.Fatalf("%q %v", raw, err)
	}
	want := ".span/runs/" + got.ID + "/handoffs/board.png"
	if !contains(agents.prompts, want) {
		t.Fatalf("prompts missing %s", want)
	}
}

func TestMissingFeatureDocumentStopsBeforeOpen(t *testing.T) {
	opt, agents, _, review, release, _ := baseOpt(t)
	agents.skipDoc = true
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Stop || got.Stage != build {
		t.Fatalf("%+v", got)
	}
	if review.opens != 0 || len(release.calls) != 0 {
		t.Fatalf("opened %d promoted %#v", review.opens, release.calls)
	}
}

func TestAttributableFailureStartsAnotherIteration(t *testing.T) {
	opt, _, _, review, _, check := baseOpt(t)
	check.fail = 1
	checkWrite := &handoffCheck{scriptCheck: check, root: opt.Root}
	opt.Check = checkWrite
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Advance {
		t.Fatalf("%+v", got)
	}
	if review.opens != 1 {
		t.Fatalf("opens %d", review.opens)
	}
	second, ok, err := statement.Read(filepath.Join(opt.Root, got.ID, "iter", "2"), frame)
	if err != nil || !ok || len(second.HandoffDigests) != 1 {
		t.Fatalf("ok=%v err=%v %+v", ok, err, second)
	}
}

func TestFourthAttributableFailureHolds(t *testing.T) {
	opt, _, _, review, _, check := baseOpt(t)
	check.fail = 10
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Hold || review.opens != 0 {
		t.Fatalf("%+v opens %d", got, review.opens)
	}
}

func TestResumeDoesNotOpenAnExistingChange(t *testing.T) {
	opt, _, source, review, _, _ := baseOpt(t)
	opt.Policy.Stages.Merge = "hold"
	id := writeCursor(t, opt, Cursor{
		ID: "span-1", Iteration: 1, Stage: open, GitSHA: "abc123",
		Pending: &Pending{Kind: open, Branch: "feat/payments", SHA: "abc123"},
	})
	review.found = true
	review.url = "https://example.test/pull/9"
	got, err := Resume(context.Background(), opt, id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Hold || got.Stage != merge {
		t.Fatalf("%+v", got)
	}
	if review.opens != 0 || source.pushes != 0 || got.ChangeURL != review.url {
		t.Fatalf("opens %d pushes %d url %s", review.opens, source.pushes, got.ChangeURL)
	}
}

func TestUnknownPromoteHoldsWithoutRunning(t *testing.T) {
	opt, _, _, _, release, _ := baseOpt(t)
	id := writeCursor(t, opt, Cursor{
		ID: "span-2", Iteration: 1, Stage: promote, GitSHA: "abc123",
		Pending: &Pending{Kind: promote, Environment: "preview", SHA: "abc123"},
	})
	got, err := Resume(context.Background(), opt, id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Hold || len(release.calls) != 0 {
		t.Fatalf("%+v calls %#v", got, release.calls)
	}
}

func TestStoppedSpanStaysStopped(t *testing.T) {
	opt, agents, _, _, _, _ := baseOpt(t)
	id := writeCursor(t, opt, Cursor{ID: "span-3", Iteration: 1, Stage: frame, Stop: true, Reason: "open questions"})
	got, err := Resume(context.Background(), opt, id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Stop || len(agents.prompts) != 0 {
		t.Fatalf("%+v prompts %d", got, len(agents.prompts))
	}
}

func TestInspectCapHolds(t *testing.T) {
	opt, agents, _, review, _, _ := baseOpt(t)
	agents.blocks = 5
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Hold || got.Stage != inspect || review.opens != 0 {
		t.Fatalf("%+v opens %d", got, review.opens)
	}
}

func TestOnlyTheThreeBranchPrefixesAreCreated(t *testing.T) {
	opt, _, source, _, _, _ := baseOpt(t)
	opt.Branch = "chore/nope"
	opt.Base = "dev"
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Hold || got.Stage != intake || source.branch != "" {
		t.Fatalf("%+v branch %q", got, source.branch)
	}
	opt.Branch = "feat/add-saved-cards"
	opt.Base = "dev"
	opt.Source = missingBase{source}
	got, err = Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Hold || !strings.Contains(got.Reason, "base branch dev is missing") {
		t.Fatalf("%+v", got)
	}
}

type missingBase struct{ *fakeSource }

func (missingBase) EnsureBranch(context.Context, string, string) error {
	return errors.New("base branch dev is missing")
}

type handoffCheck struct {
	*scriptCheck
	root string
	once bool
}

func (h *handoffCheck) Run(ctx context.Context, command, dir string) (int, error) {
	code, err := h.scriptCheck.Run(ctx, command, dir)
	if code != 0 && !h.once {
		h.once = true
		entries, readErr := os.ReadDir(h.root)
		if readErr != nil {
			return code, readErr
		}
		if len(entries) == 0 {
			return code, os.ErrNotExist
		}
		path := filepath.Join(h.root, entries[0].Name(), "handoffs")
		if mkErr := os.MkdirAll(path, 0o755); mkErr != nil {
			return code, mkErr
		}
		if wErr := os.WriteFile(filepath.Join(path, "ticket.md"), []byte("extra case"), 0o644); wErr != nil {
			return code, wErr
		}
	}
	return code, err
}

func writeCursor(t *testing.T, opt Options, cur Cursor) string {
	t.Helper()
	dir := filepath.Join(opt.Root, cur.ID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "brief.md"), []byte(opt.Brief), 0o644); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(cur)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cursor.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	return cur.ID
}

type fakeBoard struct {
	comments []string
	moves    []string
	moveErr  error
}

func (f *fakeBoard) Comment(_ context.Context, key, text string) error {
	f.comments = append(f.comments, key+"\n"+text)
	return nil
}

func (f *fakeBoard) Transition(_ context.Context, key, status string) error {
	f.moves = append(f.moves, key+" "+status)
	return f.moveErr
}

func TestFrameQuestionsWithAnIssueCommentAndBlock(t *testing.T) {
	opt, agents, _, review, _, _ := baseOpt(t)
	opt.Issue = "AET-12"
	opt.Brief = "AET-12 Charge\n\nAcceptance criteria:\nCharges the card.\n"
	agents.questions = []string{"Which currency?"}
	board := &fakeBoard{}
	opt.Issues = board
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Hold || got.Stage != frame || agents.builds != 0 || review.opens != 0 {
		t.Fatalf("%+v builds %d", got, agents.builds)
	}
	if len(board.comments) != 1 || !strings.Contains(board.comments[0], "Which currency?") || !strings.Contains(board.comments[0], "waiting for an answer") {
		t.Fatalf("%q", board.comments)
	}
	if len(board.moves) != 0 {
		t.Fatalf("%q", board.moves)
	}
}

func TestFrameQuestionsWithoutAnIssueStayLocal(t *testing.T) {
	opt, agents, _, _, _, _ := baseOpt(t)
	agents.questions = []string{"Which currency?"}
	board := &fakeBoard{}
	opt.Issues = board
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Hold || got.Stage != frame || len(board.comments) != 0 || len(board.moves) != 0 {
		t.Fatalf("%+v comments %d", got, len(board.comments))
	}
}

func TestMissingAcceptanceWithAnIssueBlocksBeforeBuild(t *testing.T) {
	opt, agents, _, _, _, _ := baseOpt(t)
	opt.Issue = "AET-12"
	opt.Brief = "AET-12 Charge the card.\n"
	board := &fakeBoard{}
	opt.Issues = board
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Hold || agents.builds != 0 || len(board.comments) != 1 || len(board.moves) != 0 {
		t.Fatalf("%+v builds %d moves %q", got, agents.builds, board.moves)
	}
	if !strings.Contains(board.comments[0], "Acceptance criteria are missing.") || strings.Contains(got.Reason, "jira account is not configured") {
		t.Fatalf("%q comment %q", got.Reason, board.comments[0])
	}
}

func TestStoryCriteriaAreCarriedWithoutATranslatedHeading(t *testing.T) {
	opt, agents, _, _, _, _ := baseOpt(t)
	opt.Issue = "SAL-3"
	opt.Brief = "SAL-3 Estado\n\nQuien tiene el permiso ve la entrada.\n"
	agents.acceptance = []string{"Quien tiene el permiso ve la entrada."}
	agents.questions = []string{"Which permission atom is Pago.Ver?"}
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Hold || strings.Contains(got.Reason, missingAcceptance) || agents.builds != 0 {
		t.Fatalf("%+v builds %d", got, agents.builds)
	}
	raw, err := os.ReadFile(filepath.Join(opt.Root, got.ID, "brief.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "Acceptance criteria:\nQuien tiene el permiso ve la entrada.\n") {
		t.Fatalf("%s", raw)
	}
}

func TestAnswerContinuesTheSameSpan(t *testing.T) {
	opt, agents, _, _, _, _ := baseOpt(t)
	agents.questions = []string{"Which currency?"}
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Hold {
		t.Fatalf("%+v", got)
	}
	next, err := Continue(context.Background(), opt, got.ID, []Answer{{Question: "Which currency?", Body: "USD"}})
	if err != nil {
		t.Fatal(err)
	}
	if next.Verdict != statement.Advance || !contains(agents.prompts, "USD") {
		t.Fatalf("%+v\n%q", next, agents.prompts)
	}
}

func TestReloadReadsTheStoryAndContinues(t *testing.T) {
	opt, agents, _, _, _, _ := baseOpt(t)
	opt.Brief = "SAL-3 Vieja\n"
	id := writeCursor(t, opt, Cursor{ID: "span-wait", Iteration: 1, Stage: frame, Stop: true, Reason: "open questions: Which permission atom?"})
	agents.acceptance = []string{"Ve la entrada."}
	got, err := Reload(context.Background(), opt, id, "SAL-3 Historia nueva\n\nVe la entrada.\n")
	if err != nil {
		t.Fatal(err)
	}
	if got.Stage == frame || strings.HasPrefix(got.Reason, "open questions:") || !contains(agents.prompts, "Historia nueva") {
		t.Fatalf("%+v", got)
	}
	raw, err := os.ReadFile(filepath.Join(opt.Root, id, "brief.md"))
	if err != nil || !strings.Contains(string(raw), "Acceptance criteria:\nVe la entrada.\n") {
		t.Fatalf("%s %v", raw, err)
	}
}

func TestContinueLeavesAnOutOfScopeStop(t *testing.T) {
	opt, agents, _, _, _, _ := baseOpt(t)
	id := writeCursor(t, opt, Cursor{ID: "span-stop", Iteration: 1, Stage: frame, Stop: true, Reason: "out of scope"})
	got, err := Continue(context.Background(), opt, id, []Answer{{Question: "Which currency?", Body: "USD"}})
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Stop || len(agents.prompts) != 0 {
		t.Fatalf("%+v prompts %d", got, len(agents.prompts))
	}
}

func TestQAMissingIDHoldsAndACoveredIDAdvances(t *testing.T) {
	opt, _, _, review, _, _ := baseOpt(t)
	opt.Brief = "Charge\n\nAcceptance criteria:\nCharges the card.\nKeeps a receipt.\n"
	if err := os.WriteFile(filepath.Join(opt.Workspace, "pay_test.go"), []byte("package pay\nfunc TestAC1_Charges(t *testing.T) {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	held, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if held.Verdict != statement.Hold || held.Stage != qaSt || review.opens != 0 {
		t.Fatalf("%+v", held)
	}
	raw, err := os.ReadFile(filepath.Join(opt.Root, held.ID, "qa-gate.json"))
	if err != nil || !strings.Contains(string(raw), `"verdict": "unverified"`) || !strings.Contains(string(raw), `"id": "AC-2"`) {
		t.Fatalf("%s %v", raw, err)
	}

	opt, _, _, review, _, _ = baseOpt(t)
	opt.Brief = "Charge\n\nAcceptance criteria:\nCharges the card.\n"
	if err := os.WriteFile(filepath.Join(opt.Workspace, "pay_test.go"), []byte("package pay\nfunc TestAC1_Charges(t *testing.T) {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Advance || got.Stage != note || review.opens != 1 {
		t.Fatalf("%+v", got)
	}
}

func TestARepeatedFrameFailureIsShownOnTheNextRun(t *testing.T) {
	opt, agents, _, _, _, _ := baseOpt(t)
	agents.frameErr = errors.New("agent reply has no JSON object")
	first, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if first.Verdict != statement.Hold || len(agents.prompts) != 1 {
		t.Fatalf("%+v prompts %d", first, len(agents.prompts))
	}
	if strings.Contains(agents.prompts[0], "Do not repeat them") {
		t.Fatal("the first failure was injected before it was stored")
	}
	second, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if second.Verdict != statement.Hold || len(agents.prompts) != 2 {
		t.Fatalf("%+v prompts %d", second, len(agents.prompts))
	}
	if !strings.Contains(agents.prompts[1], "Do not repeat them") || !strings.Contains(agents.prompts[1], "agent reply has no JSON object") {
		t.Fatal(agents.prompts[1])
	}
}

func TestBuildPromptNamesPlaywrightAsAnAuthoringAid(t *testing.T) {
	opt, agents, _, _, _, _ := baseOpt(t)
	opt.Brief = "Charge\n\nAcceptance criteria:\nCharges the card.\n"
	if err := os.WriteFile(filepath.Join(opt.Workspace, "playwright.config.ts"), []byte("export default {};\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(opt.Workspace, "pay_test.go"), []byte("package pay\nfunc TestAC1_Charges(t *testing.T) {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Start(context.Background(), opt); err != nil {
		t.Fatal(err)
	}
	if !contains(agents.prompts, "getByRole") || !contains(agents.prompts, "@playwright/mcp --caps=testing") || !contains(agents.prompts, "AC-1") {
		t.Fatalf("%q", agents.prompts)
	}
}

func contains(items []string, needle string) bool {
	for _, item := range items {
		if len(item) >= len(needle) {
			for i := 0; i+len(needle) <= len(item); i++ {
				if item[i:i+len(needle)] == needle {
					return true
				}
			}
		}
	}
	return false
}
