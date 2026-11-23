package engine

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"span/internal/agent"
	"span/internal/handoff"
	"span/internal/policy"
	"span/internal/prove"
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
	buildErr   error
	wanted     []string
	selectErr  error
}

func (f *fakeAgents) Select(want string) (agent.Selection, error) {
	f.wanted = append(f.wanted, want)
	if f.selectErr != nil {
		return agent.Selection{}, f.selectErr
	}
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
	if f.buildErr != nil {
		return f.buildErr
	}
	if f.skipDoc {
		return nil
	}
	// The document goes where the prompt says, which is the run's own feature document.
	path := filepath.Join(workspace, "odd", "tasks", "payments.md")
	if _, rest, ok := strings.Cut(text, "Feature document: "); ok {
		path, _, _ = strings.Cut(rest, "\n")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	body := "# Payments\n\n## Specs\n\nS1. Charge the card.\n\n## Tasks\n\n- T1. S1.\n\n## Log\n\nL1. Charge the card.\n"
	return os.WriteFile(path, []byte(body), 0o644)
}

func (f *fakeAgents) Inspect(context.Context, string, string, string) (InspectDoc, error) {
	f.seen++
	if f.seen <= f.blocks {
		return InspectDoc{Blocking: true, Reason: "missing spec"}, nil
	}
	return InspectDoc{Reason: "matches the spec"}, nil
}

type fakeSource struct {
	commits   int
	pushes    int
	dirty     bool
	stayDirty bool
	branch    string
	sha       string
	// messages are the commit messages, in order.
	messages []string
}

func (s *fakeSource) Head(context.Context) (string, error) { return s.sha, nil }
func (s *fakeSource) Dirty(context.Context) (bool, error)  { return s.dirty, nil }
func (s *fakeSource) Commit(_ context.Context, message string) (string, error) {
	s.commits++
	s.messages = append(s.messages, message)
	s.sha = "abc123"
	s.dirty = s.stayDirty
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
	base   string
	merged bool
	lookup error
	// openedBase is the base the last Open was given.
	openedBase string
	// extra are more open changes for the same head, after the one url names.
	extra []Change
	// titles are the titles every Open was given, in order.
	titles []string
}

func (r *fakeReview) Find(context.Context, string, string) ([]Change, error) {
	if r.lookup != nil {
		return nil, r.lookup
	}
	if !r.found {
		return r.extra, nil
	}
	return append([]Change{{URL: r.url, Base: r.base}}, r.extra...), nil
}
func (r *fakeReview) Open(_ context.Context, _, _, base, title, _ string) (string, error) {
	r.opens++
	r.titles = append(r.titles, title)
	r.openedBase = base
	r.url = "https://example.test/pull/1"
	r.base = base
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
	calls     []string
	fail      int
	text      string
	byCommand map[string]int
	texts     map[string]string
	failAfter map[string]int
	cases     int
}

func (s *scriptCheck) Run(_ context.Context, command, _ string) (prove.Output, error) {
	s.calls = append(s.calls, command)
	if s.failAfter != nil {
		if left, seen := s.failAfter[command]; seen {
			if left > 0 {
				s.failAfter[command] = left - 1
				return prove.Output{}, nil
			}
			s.cases++
			return prove.Output{ExitCode: 1, Text: s.texts[command] + " " + strconv.Itoa(s.cases)}, nil
		}
	}
	if s.byCommand != nil {
		if left := s.byCommand[command]; left > 0 {
			s.byCommand[command] = left - 1
			return prove.Output{ExitCode: 1, Text: s.texts[command]}, nil
		}
		return prove.Output{}, nil
	}
	if s.fail > 0 {
		s.fail--
		return prove.Output{ExitCode: 1, Text: s.text}, nil
	}
	return prove.Output{}, nil
}

func baseOpt(t *testing.T) (Options, *fakeAgents, *fakeSource, *fakeReview, *fakeRelease, *scriptCheck) {
	t.Helper()
	agents := &fakeAgents{}
	source := &fakeSource{dirty: true, sha: "base"}
	review := &fakeReview{}
	release := &fakeRelease{}
	check := &scriptCheck{}
	var tick int64
	// The baseline run adds calls before the first build; these tests count calls as they
	// were written, so it is off here and on in the tests that cover it.
	off := false
	opt := Options{
		Workspace: t.TempDir(),
		Root:      t.TempDir(),
		Policy: policy.File{
			Feature: "payments",
			Prove:   policy.Bands{Fast: []string{"go test ./..."}, Baseline: &off},
			Promote: policy.Ladder{Environments: []policy.Environment{{Name: "preview", Command: "echo preview"}}},
		},
		Brief:   "Charge the card.",
		Base:    "dev",
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
	second, ok, err := statement.Read(filepath.Join(opt.Root, got.ID, "iter", "2"), build)
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

func (h *handoffCheck) Run(ctx context.Context, command, dir string) (prove.Output, error) {
	out, err := h.scriptCheck.Run(ctx, command, dir)
	if out.ExitCode != 0 && !h.once {
		h.once = true
		entries, readErr := os.ReadDir(h.root)
		if readErr != nil {
			return out, readErr
		}
		if len(entries) == 0 {
			return out, os.ErrNotExist
		}
		path := filepath.Join(h.root, entries[0].Name(), "handoffs")
		if mkErr := os.MkdirAll(path, 0o755); mkErr != nil {
			return out, mkErr
		}
		if wErr := os.WriteFile(filepath.Join(path, "ticket.md"), []byte("extra case"), 0o644); wErr != nil {
			return out, wErr
		}
	}
	return out, err
}

func readCursor(t *testing.T, opt Options, id string) Cursor {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(opt.Root, id, "cursor.json"))
	if err != nil {
		t.Fatal(err)
	}
	var cur Cursor
	if err := json.Unmarshal(raw, &cur); err != nil {
		t.Fatal(err)
	}
	return cur
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

func TestTheSameFailureTwiceStopsInsteadOfBurningTheCap(t *testing.T) {
	opt, _, _, review, _, check := baseOpt(t)
	check.fail = 5
	check.text = "FAIL src/cart.test.ts > totals\nexpected 2 to be 3\n"
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Hold {
		t.Fatalf("%+v", got)
	}
	if !strings.Contains(got.Reason, "no progress") {
		t.Fatalf("reason = %s", got.Reason)
	}
	if runs := len(check.calls); runs != 2 {
		t.Fatalf("the fast command ran %d times, want 2", runs)
	}
	if review.opens != 0 {
		t.Fatalf("opens %d", review.opens)
	}
}

func TestNewFailuresKeepGoingUntilTheCap(t *testing.T) {
	opt, _, _, _, _, check := baseOpt(t)
	opt.Policy.Repair = policy.Repair{MaxAttempts: 3}
	check.failAfter = map[string]int{"go test ./...": 0}
	check.texts = map[string]string{"go test ./...": "FAIL case"}
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Hold {
		t.Fatalf("%+v", got)
	}
	if runs := len(check.calls); runs != 4 {
		t.Fatalf("the fast command ran %d times, want 4: one build plus three repairs", runs)
	}
	if !strings.Contains(got.Reason, "iteration cap 3") {
		t.Fatalf("reason = %s", got.Reason)
	}
}

func TestAFailingQAGateReturnsToBuildWithTheOutput(t *testing.T) {
	opt, _, _, _, _, check := baseOpt(t)
	opt.Policy.QA = policy.QA{Command: "qa-cmd"}
	opt.Policy.Repair = policy.Repair{MaxAttempts: 3}
	opt.Brief = "Charge the card.\n\nAcceptance criteria:\nCharges the card.\n"
	body := "package pay\n\nfunc TestAC1_Charges(t *testing.T) {}\n"
	if err := os.WriteFile(filepath.Join(opt.Workspace, "pay_test.go"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	check.byCommand = map[string]int{"qa-cmd": 1}
	check.texts = map[string]string{"qa-cmd": "FAIL src/checkout.test.ts > charges\nexpected 200 to be 402\n"}
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Advance {
		t.Fatalf("%+v", got)
	}
	first, ok, err := statement.Read(filepath.Join(opt.Root, got.ID, "iter", "1"), qaSt)
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if first.Verdict != statement.Hold {
		t.Fatalf("a failed gate is recorded as %s", first.Verdict)
	}
	if len(first.Commands) == 0 || !strings.Contains(first.Commands[0].Output, "expected 200 to be 402") {
		t.Fatalf("the gate did not keep the output: %+v", first.Commands)
	}
	if len(first.Commands) > 0 && first.Commands[0].LogPath == "" {
		t.Fatal("the full output was not written to a log file")
	}
	second, ok, err := statement.Read(filepath.Join(opt.Root, got.ID, "iter", "2"), build)
	if err != nil || !ok {
		t.Fatalf("a failing gate did not return to build: ok=%v err=%v", ok, err)
	}
	if second.Verdict != statement.Advance {
		t.Fatalf("second build %+v", second)
	}
}

func TestQADoesNotRunTheFastCommandAgainOnACleanTree(t *testing.T) {
	opt, _, source, _, _, check := baseOpt(t)
	opt.Policy.Run = policy.Run{Weight: policy.Standard}
	opt.Policy.QA = policy.QA{Command: "go test ./..."}
	source.stayDirty = false
	opt.Brief = "Charge the card.\n\nAcceptance criteria:\nCharges the card.\n"
	body := "package pay\n\nfunc TestAC1_Charges(t *testing.T) {}\n"
	if err := os.WriteFile(filepath.Join(opt.Workspace, "pay_test.go"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Advance {
		t.Fatalf("%+v", got)
	}
	if runs := len(check.calls); runs != 1 {
		t.Fatalf("the fast command ran %d times, want once", runs)
	}
	gate, ok, err := statement.Read(filepath.Join(opt.Root, got.ID, "iter", "1"), qaSt)
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if !strings.Contains(gate.Reason, "was not run again") {
		t.Fatalf("reason = %s", gate.Reason)
	}
}

func TestASameCommitFlipIsInconclusiveAndDoesNotFailTheGate(t *testing.T) {
	opt, _, source, _, _, check := baseOpt(t)
	opt.Policy.Run = policy.Run{Weight: policy.Standard}
	opt.Policy.QA = policy.QA{Command: "go test ./..."}
	opt.Policy.Repair = policy.Repair{MaxAttempts: 3}
	source.stayDirty = true
	opt.Brief = "Charge the card.\n\nAcceptance criteria:\nCharges the card.\n"
	body := "package pay\n\nfunc TestAC1_Charges(t *testing.T) {}\n"
	if err := os.WriteFile(filepath.Join(opt.Workspace, "pay_test.go"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	check.failAfter = map[string]int{"go test ./...": 1}
	check.texts = map[string]string{"go test ./...": "FAIL src/cart.test.ts > totals expected 2 to be 3"}
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Hold {
		t.Fatalf("%+v", got)
	}
	if !strings.Contains(got.Reason, "inconclusive") {
		t.Fatalf("reason = %s", got.Reason)
	}
}

func TestAQuotaBlockHoldsWithTheActionAndRecordsTheAgent(t *testing.T) {
	opt, agents, _, _, _, _ := baseOpt(t)
	agents.buildErr = &agent.Blocked{
		Agent: "claude", Kind: agent.Quota, Detail: "Claude AI usage limit reached. Your limit resets at 15:00.",
	}
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Hold {
		t.Fatalf("%+v", got)
	}
	for _, want := range []string{"quota exhausted (claude)", "Resume after the limit resets"} {
		if !strings.Contains(got.Reason, want) {
			t.Fatalf("reason %q is missing %q", got.Reason, want)
		}
	}
	cur := readCursor(t, opt, got.ID)
	if cur.AgentBlock == nil || cur.AgentBlock.Kind != agent.Quota || cur.AgentBlock.Agent != "claude" {
		t.Fatalf("agentBlock = %+v", cur.AgentBlock)
	}
	if !strings.Contains(cur.AgentBlock.Detail, "usage limit") {
		t.Fatalf("detail = %q", cur.AgentBlock.Detail)
	}
}

func TestACredentialBlockSaysToLogIn(t *testing.T) {
	opt, agents, _, _, _, _ := baseOpt(t)
	agents.buildErr = &agent.Blocked{
		Agent: "codex", Kind: agent.Credentials, Detail: "No API key found for openai-codex.",
	}
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Hold {
		t.Fatalf("%+v", got)
	}
	for _, want := range []string{"credentials missing (codex)", "Log in to that CLI"} {
		if !strings.Contains(got.Reason, want) {
			t.Fatalf("reason %q is missing %q", got.Reason, want)
		}
	}
}

func TestAnOrdinaryAgentErrorDoesNotRecordABlock(t *testing.T) {
	opt, agents, _, _, _, _ := baseOpt(t)
	agents.buildErr = errors.New("claude: exit status 1: cannot find module ./totals")
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Hold {
		t.Fatalf("%+v", got)
	}
	if cur := readCursor(t, opt, got.ID); cur.AgentBlock != nil {
		t.Fatalf("agentBlock = %+v", cur.AgentBlock)
	}
}
