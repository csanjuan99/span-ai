package serve

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/fstest"

	"span/internal/jira"
	"span/internal/shape"
)

// fakeBacklog records the calls it receives and answers with canned values.
type fakeBacklog struct {
	mu        sync.Mutex
	previews  int
	creates   []jira.Draft
	preview   jira.Preview
	created   jira.Created
	issue     jira.Story
	issueKeys []string
	err       error
}

func (f *fakeBacklog) Preview(_ context.Context, _ string, _ int) (jira.Preview, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.previews++
	return f.preview, f.err
}

func (f *fakeBacklog) CreateInBacklog(_ context.Context, d jira.Draft) (jira.Created, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.creates = append(f.creates, d)
	return f.created, f.err
}

// Issue makes the fake an issue reader too; the story it answers is f.issue.
func (f *fakeBacklog) Issue(_ context.Context, key string) (jira.Story, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.issueKeys = append(f.issueKeys, key)
	return f.issue, f.err
}

func (f *fakeBacklog) createCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.creates)
}

func agentReply(t *testing.T, v map[string]any) string {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return "ok\n" + string(raw)
}

func question(id string) map[string]any {
	return map[string]any{"id": id, "text": "who?", "assume": "everyone"}
}

type shapeRig struct {
	t       *testing.T
	ws      string
	handler http.Handler
	backlog *fakeBacklog
	run     func(ctx context.Context, dir, prompt string) (string, error)
	pages   fstest.MapFS
}

func newShapeRig(t *testing.T, reply func() string) *shapeRig {
	t.Helper()
	ws := t.TempDir()
	cmd := exec.Command("git", "init", "-q")
	cmd.Dir = ws
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, out)
	}
	rig := &shapeRig{t: t, ws: ws, backlog: &fakeBacklog{}, pages: fstest.MapFS{"index.html": {Data: []byte("<html>index</html>")}}}
	rig.run = func(context.Context, string, string) (string, error) { return reply(), nil }
	rig.handler = New(t.TempDir(), "", nil, nil, nil, rig.pages,
		WithShapeAgents(func(string, string, string) shape.Runner {
			return func(ctx context.Context, dir, prompt string) (string, error) { return rig.run(ctx, dir, prompt) }
		}),
		WithBacklog(func(string) (Backlog, error) { return rig.backlog, nil }),
	)
	return rig
}

func (r *shapeRig) do(method, path string, body any) *httptest.ResponseRecorder {
	r.t.Helper()
	var reader *strings.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			r.t.Fatal(err)
		}
		reader = strings.NewReader(string(raw))
	} else {
		reader = strings.NewReader("")
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.handler.ServeHTTP(rec, req)
	return rec
}

func (r *shapeRig) session(rec *httptest.ResponseRecorder) shape.Session {
	r.t.Helper()
	var s shape.Session
	if err := json.Unmarshal(rec.Body.Bytes(), &s); err != nil {
		r.t.Fatalf("body %q: %v", rec.Body.String(), err)
	}
	return s
}

func (r *shapeRig) start() shape.Session {
	r.t.Helper()
	rec := r.do(http.MethodPost, "/shapes", map[string]any{"workspace": r.ws, "idea": "a login"})
	if rec.Code != http.StatusCreated {
		r.t.Fatalf("start %d %s", rec.Code, rec.Body)
	}
	return r.session(rec)
}

func errCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body %q: %v", rec.Body.String(), err)
	}
	return body["error"]
}

var openReply = func(t *testing.T) func() string {
	return func() string {
		return agentReply(t, map[string]any{"map": map[string]any{"questions": []any{question("Q1")}}})
	}
}

var readyReply = func(t *testing.T) func() string {
	return func() string {
		return agentReply(t, map[string]any{"story": "# Log in\n\nAs a user I log in."})
	}
}

// sealed returns a sealed session of a rig whose agent drafts a story straight away.
func (r *shapeRig) sealed() shape.Session {
	r.t.Helper()
	s := r.start()
	rec := r.do(http.MethodPost, "/shapes/"+s.ID+"/seal", map[string]any{"workspace": r.ws})
	if rec.Code != http.StatusOK {
		r.t.Fatalf("seal %d %s", rec.Code, rec.Body)
	}
	return r.session(rec)
}

func TestShapeStartReturns201Session(t *testing.T) {
	rig := newShapeRig(t, openReply(t))
	s := rig.start()
	if s.ID == "" || s.Status != shape.StatusInterviewing || len(s.Map.Questions) != 1 || s.Idea != "a login" {
		t.Fatalf("session %+v", s)
	}
}

func TestShapeGetAndListReturnSessions(t *testing.T) {
	rig := newShapeRig(t, openReply(t))
	s := rig.start()
	one := rig.do(http.MethodGet, "/shapes/"+s.ID+"?workspace="+rig.ws, nil)
	if one.Code != http.StatusOK || rig.session(one).ID != s.ID {
		t.Fatalf("get %d %s", one.Code, one.Body)
	}
	list := rig.do(http.MethodGet, "/shapes?workspace="+rig.ws, nil)
	var got []shape.Session
	if err := json.Unmarshal(list.Body.Bytes(), &got); err != nil || list.Code != http.StatusOK || len(got) != 1 {
		t.Fatalf("list %d %s %v", list.Code, list.Body, err)
	}
}

func TestShapeListWithUnreadableSessionKeepsTheRestAndWarns(t *testing.T) {
	rig := newShapeRig(t, openReply(t))
	rig.start()
	bad := shape.Dir(rig.ws) + "/broken"
	if err := os.MkdirAll(bad, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bad+"/session.json", []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	rec := rig.do(http.MethodGet, "/shapes?workspace="+rig.ws, nil)
	var got []shape.Session
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || len(got) != 1 {
		t.Fatalf("list %d %s %v", rec.Code, rec.Body, err)
	}
	if !strings.Contains(rec.Header().Get("X-Span-Warning"), "broken") {
		t.Fatalf("warning %q", rec.Header().Get("X-Span-Warning"))
	}
}

func TestShapeAnswersRunsTheNextTurn(t *testing.T) {
	rig := newShapeRig(t, openReply(t))
	s := rig.start()
	rig.run = func(context.Context, string, string) (string, error) {
		return agentReply(t, map[string]any{"story": "# Done"}), nil
	}
	rec := rig.do(http.MethodPost, "/shapes/"+s.ID+"/answers", map[string]any{
		"workspace": rig.ws, "answers": []any{map[string]string{"question": "Q1", "text": "admins"}},
	})
	got := rig.session(rec)
	if rec.Code != http.StatusOK || got.Status != shape.StatusReady || got.Map.Questions[0].Answer != "admins" || len(got.Turns) != 2 {
		t.Fatalf("answers %d %+v", rec.Code, got)
	}
}

func TestShapeAnswersAcceptsAnAssumption(t *testing.T) {
	rig := newShapeRig(t, openReply(t))
	s := rig.start()
	rec := rig.do(http.MethodPost, "/shapes/"+s.ID+"/answers", map[string]any{"workspace": rig.ws, "assume": []string{"Q1"}})
	got := rig.session(rec)
	if rec.Code != http.StatusOK || got.Map.Questions[0].Status != shape.QuestionAssumed {
		t.Fatalf("assume %d %+v", rec.Code, got.Map.Questions)
	}
}

func TestShapeContextAcceptWritesTheProposal(t *testing.T) {
	rig := newShapeRig(t, func() string {
		return agentReply(t, map[string]any{"proposals": []any{map[string]string{"id": "P1", "section": "Glossary", "text": "- Member: a user"}}})
	})
	s := rig.start()
	rec := rig.do(http.MethodPost, "/shapes/"+s.ID+"/context", map[string]any{"workspace": rig.ws, "accept": []string{"P1"}})
	if rec.Code != http.StatusOK || rig.session(rec).Proposals[0].Status != shape.ProposalAccepted {
		t.Fatalf("context %d %s", rec.Code, rec.Body)
	}
	raw, err := os.ReadFile(shape.ContextPath(rig.ws))
	if err != nil || !strings.Contains(string(raw), "Member: a user") {
		t.Fatalf("context file %q %v", raw, err)
	}
}

func TestShapeSealNotReadyListsOpenQuestionsAs409(t *testing.T) {
	rig := newShapeRig(t, openReply(t))
	s := rig.start()
	rec := rig.do(http.MethodPost, "/shapes/"+s.ID+"/seal", map[string]any{"workspace": rig.ws})
	if rec.Code != http.StatusConflict || errCode(t, rec) != "shape_not_ready" || !strings.Contains(rec.Body.String(), "Q1") {
		t.Fatalf("seal %d %s", rec.Code, rec.Body)
	}
}

func TestShapeSealStoresStoryAndDigest(t *testing.T) {
	rig := newShapeRig(t, readyReply(t))
	s := rig.sealed()
	if s.Status != shape.StatusSealed || s.Digest == "" {
		t.Fatalf("sealed %+v", s)
	}
}

func TestShapeAnswersOnASealedSessionIs409(t *testing.T) {
	rig := newShapeRig(t, readyReply(t))
	s := rig.sealed()
	rec := rig.do(http.MethodPost, "/shapes/"+s.ID+"/answers", map[string]any{"workspace": rig.ws})
	if rec.Code != http.StatusConflict || errCode(t, rec) != "shape_sealed" {
		t.Fatalf("answers %d %s", rec.Code, rec.Body)
	}
}

func TestShapeRepoChangedIs409(t *testing.T) {
	rig := newShapeRig(t, openReply(t))
	rig.run = func(_ context.Context, dir, _ string) (string, error) {
		return "", os.WriteFile(dir+"/stray.txt", []byte("x"), 0o644)
	}
	rec := rig.do(http.MethodPost, "/shapes", map[string]any{"workspace": rig.ws, "idea": "a login"})
	if rec.Code != http.StatusConflict || errCode(t, rec) != "shape_repo_changed" {
		t.Fatalf("start %d %s", rec.Code, rec.Body)
	}
}

func TestShapeNotGitIs400(t *testing.T) {
	rig := newShapeRig(t, openReply(t))
	plain := t.TempDir()
	rec := rig.do(http.MethodPost, "/shapes", map[string]any{"workspace": plain, "idea": "a login"})
	if rec.Code != http.StatusBadRequest || errCode(t, rec) != "shape_not_git" {
		t.Fatalf("start %d %s", rec.Code, rec.Body)
	}
}

func TestShapeUnknownIdIs404(t *testing.T) {
	rig := newShapeRig(t, openReply(t))
	rec := rig.do(http.MethodGet, "/shapes/nope?workspace="+rig.ws, nil)
	if rec.Code != http.StatusNotFound || errCode(t, rec) != "shape_not_found" {
		t.Fatalf("get %d %s", rec.Code, rec.Body)
	}
}

func TestShapeWorkspaceMustBeAnExistingFolder(t *testing.T) {
	rig := newShapeRig(t, openReply(t))
	for _, ws := range []string{"", "relative/dir", rig.ws + "/missing"} {
		rec := rig.do(http.MethodPost, "/shapes", map[string]any{"workspace": ws, "idea": "x"})
		if rec.Code != http.StatusBadRequest || errCode(t, rec) != "invalid" {
			t.Fatalf("workspace %q: %d %s", ws, rec.Code, rec.Body)
		}
	}
}

func TestShapeRelativeWorkspaceResolvesAgainstTheServerFolder(t *testing.T) {
	rig := newShapeRig(t, openReply(t))
	t.Chdir(rig.ws)
	rec := rig.do(http.MethodPost, "/shapes", map[string]any{"workspace": ".", "idea": "x"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("start %d %s", rec.Code, rec.Body)
	}
	got := rig.session(rec)
	want, err := filepath.EvalSymlinks(rig.ws)
	if err != nil {
		t.Fatal(err)
	}
	stored, err := filepath.EvalSymlinks(got.Workspace)
	if err != nil || stored != want {
		t.Fatalf("workspace %q, want %q (%v)", got.Workspace, want, err)
	}
}

func TestShapePreviewHasSummaryAndDescription(t *testing.T) {
	rig := newShapeRig(t, readyReply(t))
	rig.backlog.preview = jira.Preview{Board: jira.Board{ID: 7, Name: "Kanban"}, BacklogStatuses: []string{"Backlog"}}
	s := rig.sealed()
	rec := rig.do(http.MethodGet, "/shapes/"+s.ID+"/jira/preview?workspace="+rig.ws+"&project=abc&board=7", nil)
	var got struct {
		Project         string
		Board           jira.Board
		BacklogStatuses []string
		Summary         string
		Description     string
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || rec.Code != http.StatusOK {
		t.Fatalf("preview %d %s %v", rec.Code, rec.Body, err)
	}
	if got.Project != "ABC" || got.Board.ID != 7 || got.Summary != "Log in" || got.Description != s.Story || got.BacklogStatuses[0] != "Backlog" {
		t.Fatalf("preview %+v", got)
	}
}

func TestShapePreviewNeedsASealedSession(t *testing.T) {
	rig := newShapeRig(t, openReply(t))
	s := rig.start()
	rec := rig.do(http.MethodGet, "/shapes/"+s.ID+"/jira/preview?workspace="+rig.ws+"&project=ABC", nil)
	if rec.Code != http.StatusConflict || errCode(t, rec) != "shape_not_sealed" || rig.backlog.previews != 0 {
		t.Fatalf("preview %d %s", rec.Code, rec.Body)
	}
}

func TestShapeJiraWithoutConfirmIs400AndCallsNothing(t *testing.T) {
	rig := newShapeRig(t, readyReply(t))
	s := rig.sealed()
	rec := rig.do(http.MethodPost, "/shapes/"+s.ID+"/jira", map[string]any{"workspace": rig.ws, "project": "ABC"})
	if rec.Code != http.StatusBadRequest || errCode(t, rec) != "confirm_required" || rig.backlog.createCount() != 0 {
		t.Fatalf("jira %d %s", rec.Code, rec.Body)
	}
}

func TestShapeJiraWhenNotSealedIs409(t *testing.T) {
	rig := newShapeRig(t, openReply(t))
	s := rig.start()
	rec := rig.do(http.MethodPost, "/shapes/"+s.ID+"/jira", map[string]any{"workspace": rig.ws, "project": "ABC", "confirm": true})
	if rec.Code != http.StatusConflict || errCode(t, rec) != "shape_not_sealed" || rig.backlog.createCount() != 0 {
		t.Fatalf("jira %d %s", rec.Code, rec.Body)
	}
}

func TestShapeJiraStoresTheKeyAndIsIdempotent(t *testing.T) {
	rig := newShapeRig(t, readyReply(t))
	rig.backlog.created = jira.Created{Key: "ABC-1", URL: "https://x/browse/ABC-1", Status: "Backlog"}
	s := rig.sealed()
	body := map[string]any{"workspace": rig.ws, "project": "ABC", "board": 7, "confirm": true}
	first := rig.do(http.MethodPost, "/shapes/"+s.ID+"/jira", body)
	got := rig.session(first)
	if first.Code != http.StatusOK || got.Jira == nil || got.Jira.Key != "ABC-1" {
		t.Fatalf("jira %d %s", first.Code, first.Body)
	}
	draft := rig.backlog.creates[0]
	if draft.Project != "ABC" || draft.Board != 7 || draft.Summary != "Log in" || draft.Description != s.Story {
		t.Fatalf("draft %+v", draft)
	}
	second := rig.do(http.MethodPost, "/shapes/"+s.ID+"/jira", body)
	if second.Code != http.StatusOK || rig.session(second).Jira.Key != "ABC-1" || rig.backlog.createCount() != 1 {
		t.Fatalf("second %d %s creates %d", second.Code, second.Body, rig.backlog.createCount())
	}
}

func TestShapeJiraBacklogDisabledIs409AndStoresNoKey(t *testing.T) {
	rig := newShapeRig(t, readyReply(t))
	rig.backlog.err = &jira.Error{Code: "jira_backlog_disabled", Msg: "the Backlog column is empty"}
	s := rig.sealed()
	rec := rig.do(http.MethodPost, "/shapes/"+s.ID+"/jira", map[string]any{"workspace": rig.ws, "project": "ABC", "confirm": true})
	if rec.Code != http.StatusConflict || errCode(t, rec) != "jira_backlog_disabled" {
		t.Fatalf("jira %d %s", rec.Code, rec.Body)
	}
	stored, err := shape.Load(rig.ws, s.ID)
	if err != nil || stored.Jira != nil {
		t.Fatalf("stored %+v %v", stored, err)
	}
}

func TestShapeJiraWithoutAKeyOrAnErrorIs400AndStoresNothing(t *testing.T) {
	rig := newShapeRig(t, readyReply(t))
	s := rig.sealed()
	rec := rig.do(http.MethodPost, "/shapes/"+s.ID+"/jira", map[string]any{"workspace": rig.ws, "project": "ABC", "confirm": true})
	if rec.Code != http.StatusBadRequest || errCode(t, rec) != "invalid" {
		t.Fatalf("jira %d %s", rec.Code, rec.Body)
	}
	stored, err := shape.Load(rig.ws, s.ID)
	if err != nil || stored.Jira != nil {
		t.Fatalf("stored %+v %v", stored, err)
	}
}

func TestShapeJiraNotInBacklogKeepsTheCreatedKey(t *testing.T) {
	rig := newShapeRig(t, readyReply(t))
	rig.backlog.created = jira.Created{Key: "ABC-9", URL: "https://x/browse/ABC-9"}
	rig.backlog.err = &jira.Error{Code: "jira_not_in_backlog", Msg: "ABC-9 was created but is not in the backlog"}
	s := rig.sealed()
	rec := rig.do(http.MethodPost, "/shapes/"+s.ID+"/jira", map[string]any{"workspace": rig.ws, "project": "ABC", "confirm": true})
	if rec.Code != http.StatusConflict || errCode(t, rec) != "jira_not_in_backlog" {
		t.Fatalf("jira %d %s", rec.Code, rec.Body)
	}
	stored, err := shape.Load(rig.ws, s.ID)
	if err != nil || stored.Jira == nil || stored.Jira.Key != "ABC-9" {
		t.Fatalf("stored %+v %v", stored, err)
	}
}

func TestShapeErrorStatusesFollowTheCode(t *testing.T) {
	cases := map[string]int{
		"shape_not_found": 404, "shape_not_ready": 409, "shape_not_sealed": 409, "shape_sealed": 409,
		"shape_repo_changed": 409, "jira_backlog_disabled": 409, "jira_board_ambiguous": 409,
		"jira_not_in_backlog": 409, "confirm_required": 400, "shape_not_git": 400, "jira_board_not_found": 400,
	}
	for code, want := range cases {
		rec := httptest.NewRecorder()
		writeErr(rec, &jira.Error{Code: code})
		if rec.Code != want {
			t.Errorf("%s: status %d, want %d", code, rec.Code, want)
		}
	}
}

func TestWriteErrKeepsTheExistingMapping(t *testing.T) {
	cases := map[string]int{"not_found": 404, "unauthorized": 401, "unreachable": 502, "idempotency_mismatch": 422, "other": 400}
	for code, want := range cases {
		rec := httptest.NewRecorder()
		writeErr(rec, &jira.Error{Code: code})
		if rec.Code != want {
			t.Errorf("%s: status %d, want %d", code, rec.Code, want)
		}
	}
}

func TestShapeTurnsOnOneSessionDoNotInterleave(t *testing.T) {
	rig := newShapeRig(t, openReply(t))
	s := rig.start()
	var active, peak int32
	release := make(chan struct{})
	entered := make(chan struct{}, 2)
	rig.run = func(context.Context, string, string) (string, error) {
		now := atomic.AddInt32(&active, 1)
		for {
			old := atomic.LoadInt32(&peak)
			if now <= old || atomic.CompareAndSwapInt32(&peak, old, now) {
				break
			}
		}
		entered <- struct{}{}
		<-release
		atomic.AddInt32(&active, -1)
		return agentReply(t, map[string]any{"story": "# S"}), nil
	}
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rig.do(http.MethodPost, "/shapes/"+s.ID+"/answers", map[string]any{"workspace": rig.ws, "assume": []string{}})
		}()
	}
	<-entered
	select {
	case <-entered:
		t.Error("second turn started while the first was running")
	default:
	}
	release <- struct{}{}
	<-entered
	release <- struct{}{}
	wg.Wait()
	if p := atomic.LoadInt32(&peak); p != 1 {
		t.Fatalf("peak concurrency %d", p)
	}
}

func TestShapePageRoutesServeTheIndex(t *testing.T) {
	rig := newShapeRig(t, openReply(t))
	for _, path := range []string{"/shaping", "/shaping/x"} {
		rec := rig.do(http.MethodGet, path, nil)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "index") {
			t.Fatalf("%s: %d %s", path, rec.Code, rec.Body)
		}
	}
}

func TestExistingFallbackStillServesTheIndex(t *testing.T) {
	rig := newShapeRig(t, openReply(t))
	rec := rig.do(http.MethodGet, "/desk", nil)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "index") {
		t.Fatalf("fallback %d %s", rec.Code, rec.Body)
	}
}

func TestJiraViewWithoutAccountIsNotConfigured(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	rec := httptest.NewRecorder()
	Handler(t.TempDir(), "").ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/jira?path="+t.TempDir(), nil))
	var view struct{ Configured bool }
	if err := json.Unmarshal(rec.Body.Bytes(), &view); err != nil || rec.Code != http.StatusOK || view.Configured {
		t.Fatalf("view %d %s %v", rec.Code, rec.Body, err)
	}
}

func TestShapeProjectIsRefusedWhenEmpty(t *testing.T) {
	rig := newShapeRig(t, readyReply(t))
	s := rig.sealed()
	rec := rig.do(http.MethodPost, "/shapes/"+s.ID+"/jira", map[string]any{"workspace": rig.ws, "project": " ", "confirm": true})
	if rec.Code != http.StatusBadRequest || errCode(t, rec) != "invalid" || rig.backlog.createCount() != 0 {
		t.Fatalf("jira %d %s", rec.Code, rec.Body)
	}
}
