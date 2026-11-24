package shape

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fake struct {
	replies []string
	prompts []string
	dirs    []string
	effect  func(dir string)
	err     error
}

func (f *fake) run(_ context.Context, dir, prompt string) (string, error) {
	f.prompts = append(f.prompts, prompt)
	f.dirs = append(f.dirs, dir)
	if f.effect != nil {
		f.effect(dir)
	}
	if f.err != nil {
		return "", f.err
	}
	i := len(f.prompts) - 1
	if i >= len(f.replies) {
		i = len(f.replies) - 1
	}
	return f.replies[i], nil
}

func reply(t *testing.T, v any) string {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return "here you go\n" + string(raw)
}

func initGit(t *testing.T, dir string) {
	t.Helper()
	cmd := exec.Command("git", "init", "-q")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, out)
	}
}

func gitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	initGit(t, dir)
	return dir
}

func mustLoad(t *testing.T, ws, id string) *Session {
	t.Helper()
	s, err := Load(ws, id)
	if err != nil {
		t.Fatalf("load %s: %v", id, err)
	}
	return s
}

func writeContext(t *testing.T, ws, body string) string {
	t.Helper()
	file := ContextPath(ws)
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return file
}

func mustRead(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func codeOf(err error) string {
	var coded interface{ SpanCode() string }
	if errors.As(err, &coded) {
		return coded.SpanCode()
	}
	return ""
}

func newEngine(f *fake) *Engine {
	n := 0
	return &Engine{
		Run:     f.run,
		Harness: "/h",
		Now: func() time.Time {
			n++
			return time.Date(2026, 11, 24, 10, n, 0, 0, time.UTC)
		},
	}
}

func engine(t *testing.T, f *fake) (*Engine, string) {
	t.Helper()
	return newEngine(f), gitRepo(t)
}

func q(id, text, assume string) map[string]any {
	return map[string]any{"id": id, "text": text, "assume": assume, "blocks": "R1", "status": "open"}
}

func rule(id string) map[string]any {
	return map[string]any{"id": id, "text": "rule " + id, "source": map[string]string{"kind": "prompt", "ref": "idea"}}
}

func turn(questions []map[string]any, story string) map[string]any {
	return map[string]any{
		"map":       map[string]any{"rules": []any{rule("R1")}, "examples": []any{}, "questions": questions},
		"split":     []any{},
		"proposals": []any{},
		"story":     story,
	}
}

func start(t *testing.T, e *Engine, ws string) *Session {
	t.Helper()
	s, err := e.Start(context.Background(), StartInput{Workspace: ws, Idea: "let users export", Agent: "claude", Model: "m"})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	return s
}

func TestCreateLoadList(t *testing.T) {
	ws := t.TempDir()
	a, err := Create(ws, StartInput{Workspace: ws, Idea: "one", Agent: "claude"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := Create(ws, StartInput{Workspace: ws, Idea: "two"})
	if err != nil {
		t.Fatal(err)
	}
	if a.ID == b.ID || a.Status != StatusInterviewing || a.Idea != "one" || a.Workspace != ws {
		t.Fatalf("bad sessions %+v %+v", a, b)
	}
	if _, err := os.Stat(filepath.Join(ws, ".span", "shapes", a.ID, "session.json")); err != nil {
		t.Fatalf("session file: %v", err)
	}
	got, err := Load(ws, a.ID)
	if err != nil || got.Idea != "one" || got.Agent != "claude" {
		t.Fatalf("load %+v %v", got, err)
	}
	list, err := List(ws)
	if err != nil || len(list) != 2 || list[0].ID != b.ID {
		t.Fatalf("list newest first: %+v %v", list, err)
	}
}

func TestLoadRejectsTraversalAndMissing(t *testing.T) {
	ws := t.TempDir()
	for _, id := range []string{"../x", "a/b", "", "..", `a\b`} {
		if _, err := Load(ws, id); codeOf(err) != "shape_not_found" {
			t.Errorf("id %q: code %q err %v", id, codeOf(err), err)
		}
	}
	if _, err := Load(ws, "nope"); codeOf(err) != "shape_not_found" {
		t.Fatalf("missing: %v", err)
	}
	list, err := List(ws)
	if err != nil || len(list) != 0 {
		t.Fatalf("empty list %v %v", list, err)
	}
}

func TestFirstTurnBuildsPromptAndMap(t *testing.T) {
	f := &fake{replies: []string{reply(t, turn([]map[string]any{q("Q1", "who?", "admins"), q("Q2", "when?", "daily")}, ""))}}
	e, ws := engine(t, f)
	writeContext(t, ws, "# Product\nPersona: Ana\n")
	s := start(t, e, ws)
	if s.Status != StatusInterviewing || len(s.Map.Questions) != 2 || len(s.Map.Rules) != 1 {
		t.Fatalf("session %+v", s)
	}
	if len(s.Turns) != 1 || s.Turns[0].N != 1 || len(s.Turns[0].Asked) != 2 || s.Turns[0].At.IsZero() {
		t.Fatalf("turns %+v", s.Turns)
	}
	if f.dirs[0] != ws {
		t.Fatalf("runner dir %q", f.dirs[0])
	}
	p := f.prompts[0]
	for _, want := range []string{
		"/h/skills/user-story/SKILL.md", "/h/skills/user-story/references/interview.md",
		"let users export", "Persona: Ana", `"proposals":[]`, `"story":""`, "must not modify",
	} {
		if !strings.Contains(p, want) {
			t.Errorf("prompt lacks %q", want)
		}
	}
	disk, err := Load(ws, s.ID)
	if err != nil || len(disk.Map.Questions) != 2 {
		t.Fatalf("persisted %+v %v", disk, err)
	}
}

func TestRepoSubfolderIsRunnerDir(t *testing.T) {
	f := &fake{replies: []string{reply(t, turn(nil, "story"))}}
	e, ws := engine(t, f)
	repo := filepath.Join(ws, "api")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	initGit(t, repo)
	if _, err := e.Start(context.Background(), StartInput{Workspace: ws, Repo: "api", Idea: "x"}); err != nil {
		t.Fatal(err)
	}
	if f.dirs[0] != repo {
		t.Fatalf("dir %q want %q", f.dirs[0], repo)
	}
}

func TestMergeKeepsKnownQuestionsAndCapsNew(t *testing.T) {
	first := reply(t, turn([]map[string]any{q("Q1", "who?", "admins")}, ""))
	reopen := q("Q1", "who?", "admins")
	reopen["status"] = "answered"
	reopen["answer"] = "agent made this up"
	second := reply(t, turn([]map[string]any{reopen, q("Q2", "a", ""), q("Q3", "b", ""), q("Q4", "c", ""), q("Q5", "d", "")}, ""))
	f := &fake{replies: []string{first, second}}
	e, ws := engine(t, f)
	s := start(t, e, ws)
	s, err := e.Answer(context.Background(), ws, s.ID, []Answer{{Question: "Q1", Text: "only admins"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Map.Questions) != 4 {
		t.Fatalf("want Q1 + 3 new, got %+v", s.Map.Questions)
	}
	q1 := s.Map.Questions[0]
	if q1.Status != QuestionAnswered || q1.Answer != "only admins" {
		t.Fatalf("Q1 must keep the person's answer: %+v", q1)
	}
	for _, x := range s.Map.Questions[1:] {
		if x.Status != QuestionOpen || x.Answer != "" {
			t.Fatalf("new question must be open: %+v", x)
		}
	}
	if got := s.Turns[1].Asked; len(got) != 3 || got[0] != "Q2" {
		t.Fatalf("asked %v", got)
	}
	if !strings.Contains(f.prompts[1], "only admins") || !strings.Contains(f.prompts[1], "answered") {
		t.Fatalf("second prompt lacks the answers")
	}
}

func TestMissingOrBadSourceFailsTurn(t *testing.T) {
	for name, src := range map[string]any{
		"missing": nil,
		"bad":     map[string]string{"kind": "guess", "ref": "x"},
		"empty":   map[string]string{"kind": "", "ref": ""},
	} {
		t.Run(name, func(t *testing.T) {
			r := map[string]any{"id": "R1", "text": "t"}
			if src != nil {
				r["source"] = src
			}
			body := turn(nil, "")
			body["map"] = map[string]any{"rules": []any{r}, "examples": []any{}, "questions": []any{}}
			f := &fake{replies: []string{reply(t, body)}}
			e, ws := engine(t, f)
			_, err := e.Start(context.Background(), StartInput{Workspace: ws, Idea: "x"})
			if err == nil || !strings.Contains(err.Error(), "R1") || !strings.Contains(err.Error(), "source") {
				t.Fatalf("want source error naming R1, got %v", err)
			}
			list, lerr := List(ws)
			if lerr != nil || len(list) != 0 {
				t.Fatalf("a failed first turn must not leave a session: %v %v", list, lerr)
			}
		})
	}
}

func TestExampleNeedsSource(t *testing.T) {
	body := turn(nil, "")
	body["map"] = map[string]any{"rules": []any{rule("R1")}, "examples": []any{map[string]any{"id": "E1", "rule": "R1", "text": "t"}}, "questions": []any{}}
	f := &fake{replies: []string{reply(t, body)}}
	e, ws := engine(t, f)
	if _, err := e.Start(context.Background(), StartInput{Workspace: ws, Idea: "x"}); err == nil || !strings.Contains(err.Error(), "E1") {
		t.Fatalf("want E1 error, got %v", err)
	}
}

func TestStatusComputed(t *testing.T) {
	cases := map[string]struct {
		body map[string]any
		want string
	}{
		"open question": {turn([]map[string]any{q("Q1", "x", "")}, "story"), StatusInterviewing},
		"no story":      {turn(nil, ""), StatusInterviewing},
		"ready":         {turn(nil, "# story"), StatusReady},
		"split": {func() map[string]any {
			b := turn(nil, "story")
			b["split"] = []any{map[string]string{"story": "A", "leftOut": "B"}}
			return b
		}(), StatusSplit},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			f := &fake{replies: []string{reply(t, c.body)}}
			e, ws := engine(t, f)
			if s := start(t, e, ws); s.Status != c.want {
				t.Fatalf("status %q want %q", s.Status, c.want)
			}
		})
	}
}

func TestAnswerAndAssume(t *testing.T) {
	f := &fake{replies: []string{
		reply(t, turn([]map[string]any{q("Q1", "who?", "admins"), q("Q2", "when?", "daily")}, "")),
		reply(t, turn(nil, "# story")),
	}}
	e, ws := engine(t, f)
	s := start(t, e, ws)
	s, err := e.Answer(context.Background(), ws, s.ID, []Answer{{Question: "Q1", Text: "all staff"}}, []string{"Q2"})
	if err != nil {
		t.Fatal(err)
	}
	if s.Map.Questions[0].Status != QuestionAnswered || s.Map.Questions[0].Answer != "all staff" {
		t.Fatalf("Q1 %+v", s.Map.Questions[0])
	}
	if s.Map.Questions[1].Status != QuestionAssumed || s.Map.Questions[1].Answer != "daily" {
		t.Fatalf("Q2 %+v", s.Map.Questions[1])
	}
	if s.Status != StatusReady || s.Story != "# story" {
		t.Fatalf("status %q", s.Status)
	}
	if _, err := e.Answer(context.Background(), ws, s.ID, []Answer{{Question: "Q9", Text: "x"}}, nil); err == nil {
		t.Fatal("unknown question must fail")
	}
}

func TestAssumeWithoutTextFails(t *testing.T) {
	f := &fake{replies: []string{reply(t, turn([]map[string]any{q("Q1", "who?", "")}, ""))}}
	e, ws := engine(t, f)
	s := start(t, e, ws)
	if _, err := e.Answer(context.Background(), ws, s.ID, nil, []string{"Q1"}); err == nil {
		t.Fatal("assuming a question with no assumption must fail")
	}
}

func TestGitGuard(t *testing.T) {
	f := &fake{replies: []string{reply(t, turn(nil, "s"))}}
	e, ws := engine(t, f)
	f.effect = func(dir string) {
		if err := os.WriteFile(filepath.Join(dir, "touched.txt"), []byte("x"), 0o644); err != nil {
			t.Error(err)
		}
	}
	_, err := e.Start(context.Background(), StartInput{Workspace: ws, Idea: "x"})
	if codeOf(err) != "shape_repo_changed" || !strings.Contains(err.Error(), "touched.txt") {
		t.Fatalf("want shape_repo_changed naming the path, got %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(ws, "touched.txt")); statErr != nil {
		t.Fatal("the file must not be reverted")
	}
}

func TestGitGuardIgnoresPreExistingChanges(t *testing.T) {
	f := &fake{replies: []string{reply(t, turn(nil, "s"))}}
	e, ws := engine(t, f)
	if err := os.WriteFile(filepath.Join(ws, "dirty.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Start(context.Background(), StartInput{Workspace: ws, Idea: "x"}); err != nil {
		t.Fatalf("pre-existing dirt is not the agent's: %v", err)
	}
}

func TestNotAGitRepo(t *testing.T) {
	f := &fake{replies: []string{reply(t, turn(nil, "s"))}}
	e := newEngine(f)
	plain := t.TempDir()
	_, err := e.Start(context.Background(), StartInput{Workspace: plain, Idea: "x"})
	if err == nil || !strings.Contains(err.Error(), "git") {
		t.Fatalf("want clear git error, got %v", err)
	}
	if len(f.prompts) != 0 {
		t.Fatal("the agent must not run outside a git repo")
	}
}

func TestAcceptProposals(t *testing.T) {
	body := turn(nil, "")
	body["proposals"] = []any{
		map[string]string{"id": "P1", "section": "Personas", "text": "- Ana, admin"},
		map[string]string{"id": "P2", "section": "Glossary", "text": "- Export: CSV file"},
	}
	again := turn(nil, "")
	again["proposals"] = []any{map[string]string{"id": "P1", "section": "Personas", "text": "changed"}}
	f := &fake{replies: []string{reply(t, body), reply(t, again)}}
	e, ws := engine(t, f)
	s := start(t, e, ws)
	file := ContextPath(ws)
	if _, err := os.Stat(file); err == nil {
		t.Fatal("nothing writes the file before acceptance")
	}
	if s.Proposals[0].Status != ProposalProposed {
		t.Fatalf("%+v", s.Proposals)
	}
	s, err := e.AcceptProposals(ws, s.ID, []string{"P1"})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "## Personas") || !strings.Contains(string(raw), "- Ana, admin") || strings.Contains(string(raw), "Glossary") {
		t.Fatalf("file:\n%s", raw)
	}
	if s.Proposals[0].Status != ProposalAccepted || s.Proposals[1].Status != ProposalProposed {
		t.Fatalf("%+v", s.Proposals)
	}
	if _, err := e.AcceptProposals(ws, s.ID, []string{"P2"}); err != nil {
		t.Fatal(err)
	}
	raw, err = os.ReadFile(file)
	if err != nil || strings.Count(string(raw), "- Ana, admin") != 1 || !strings.Contains(string(raw), "## Glossary") {
		t.Fatalf("file:\n%s %v", raw, err)
	}
	if _, err := e.AcceptProposals(ws, s.ID, []string{"P9"}); err == nil {
		t.Fatal("unknown proposal must fail")
	}
	// a later turn proposing the same id does not duplicate or reset it
	s, err = e.Answer(context.Background(), ws, s.ID, nil, nil)
	if err != nil || len(s.Proposals) != 2 || s.Proposals[0].Status != ProposalAccepted || s.Proposals[0].Text != "- Ana, admin" {
		t.Fatalf("proposals %+v %v", s.Proposals, err)
	}
}

func TestAcceptAppendsUnderExistingSection(t *testing.T) {
	body := turn(nil, "")
	body["proposals"] = []any{map[string]string{"id": "P1", "section": "Personas", "text": "- Bo"}}
	f := &fake{replies: []string{reply(t, body)}}
	e, ws := engine(t, f)
	file := writeContext(t, ws, "## Personas\n- Ana\n\n## Glossary\n- X\n")
	s := start(t, e, ws)
	if _, err := e.AcceptProposals(ws, s.ID, []string{"P1"}); err != nil {
		t.Fatal(err)
	}
	got := mustRead(t, file)
	if strings.Count(got, "## Personas") != 1 || strings.Index(got, "- Bo") > strings.Index(got, "## Glossary") || !strings.Contains(got, "- X") {
		t.Fatalf("file:\n%s", got)
	}
}

func TestSealGateAndDigest(t *testing.T) {
	f := &fake{replies: []string{
		reply(t, turn([]map[string]any{q("Q1", "who?", "a"), q("Q2", "when?", "b")}, "# story")),
		reply(t, turn(nil, "# story")),
	}}
	e, ws := engine(t, f)
	s := start(t, e, ws)
	_, err := e.Seal(ws, s.ID)
	if codeOf(err) != "shape_not_ready" || !strings.Contains(err.Error(), "Q1") || !strings.Contains(err.Error(), "Q2") {
		t.Fatalf("want shape_not_ready listing Q1,Q2, got %v", err)
	}
	s, err = e.Answer(context.Background(), ws, s.ID, []Answer{{Question: "Q1", Text: "x"}}, []string{"Q2"})
	if err != nil {
		t.Fatal(err)
	}
	s, err = e.Seal(ws, s.ID)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte("# story"))
	if s.Status != StatusSealed || s.Story != "# story" || s.Digest != hex.EncodeToString(sum[:]) {
		t.Fatalf("sealed %+v", s)
	}
	disk := mustLoad(t, ws, s.ID)
	if disk.Status != StatusSealed || disk.Digest != s.Digest {
		t.Fatalf("not persisted %+v", disk)
	}
}

func TestSealedImmutable(t *testing.T) {
	body := turn(nil, "# story")
	body["proposals"] = []any{map[string]string{"id": "P1", "section": "S", "text": "t"}}
	f := &fake{replies: []string{reply(t, body)}}
	e, ws := engine(t, f)
	s := start(t, e, ws)
	if _, err := e.SetJira(ws, s.ID, JiraRef{Key: "A-1"}); codeOf(err) != "shape_not_sealed" {
		t.Fatalf("SetJira before seal: %v", err)
	}
	if _, err := e.Seal(ws, s.ID); err != nil {
		t.Fatal(err)
	}
	turns := len(f.prompts)
	if _, err := e.Answer(context.Background(), ws, s.ID, nil, nil); codeOf(err) != "shape_sealed" {
		t.Fatalf("answer: %v", err)
	}
	if _, err := e.AcceptProposals(ws, s.ID, []string{"P1"}); codeOf(err) != "shape_sealed" {
		t.Fatalf("accept: %v", err)
	}
	if _, err := e.Seal(ws, s.ID); codeOf(err) != "shape_sealed" {
		t.Fatalf("seal: %v", err)
	}
	if len(f.prompts) != turns {
		t.Fatal("a sealed session must not run the agent")
	}
	before := mustLoad(t, ws, s.ID)
	got, err := e.SetJira(ws, s.ID, JiraRef{Key: "A-1", URL: "http://j/A-1", Status: "Backlog"})
	if err != nil || got.Jira == nil || got.Jira.Key != "A-1" || got.Jira.Status != "Backlog" {
		t.Fatalf("SetJira %+v %v", got, err)
	}
	after := mustLoad(t, ws, s.ID)
	if after.Digest != before.Digest || after.Story != before.Story || after.Status != StatusSealed || after.Jira.URL != "http://j/A-1" {
		t.Fatalf("after %+v", after)
	}
}

func TestRunnerErrorFailsStartAndLeavesNothing(t *testing.T) {
	f := &fake{err: errors.New("agent crashed")}
	e, ws := engine(t, f)
	_, err := e.Start(context.Background(), StartInput{Workspace: ws, Idea: "x"})
	if err == nil || !strings.Contains(err.Error(), "agent crashed") {
		t.Fatalf("want runner error, got %v", err)
	}
	list, lerr := List(ws)
	if lerr != nil || len(list) != 0 {
		t.Fatalf("nothing on disk: %v %v", list, lerr)
	}
}

func TestRunnerErrorKeepsAnswersWithoutNewTurn(t *testing.T) {
	f := &fake{replies: []string{reply(t, turn([]map[string]any{q("Q1", "who?", "a")}, ""))}}
	e, ws := engine(t, f)
	s := start(t, e, ws)
	f.err = errors.New("agent crashed")
	if _, err := e.Answer(context.Background(), ws, s.ID, []Answer{{Question: "Q1", Text: "ops"}}, nil); err == nil {
		t.Fatal("want runner error")
	}
	disk := mustLoad(t, ws, s.ID)
	if disk.Map.Questions[0].Status != QuestionAnswered || disk.Map.Questions[0].Answer != "ops" {
		t.Fatalf("answers must be saved: %+v", disk.Map.Questions)
	}
	if len(disk.Turns) != 1 {
		t.Fatalf("no turn appended: %+v", disk.Turns)
	}
}

func TestSplitStatusBlocksSeal(t *testing.T) {
	body := turn(nil, "story")
	body["split"] = []any{map[string]string{"story": "A", "leftOut": "B"}}
	f := &fake{replies: []string{reply(t, body)}}
	e, ws := engine(t, f)
	s := start(t, e, ws)
	if s.Status != StatusSplit || len(s.Split) != 1 || s.Split[0].LeftOut != "B" {
		t.Fatalf("session %+v", s)
	}
	if _, err := e.Seal(ws, s.ID); codeOf(err) != "shape_not_ready" {
		t.Fatalf("seal on split: %v", err)
	}
	if got := mustLoad(t, ws, s.ID); got.Status != StatusSplit || got.Digest != "" {
		t.Fatalf("persisted %+v", got)
	}
}

func TestPromptNamesInterviewReferenceThatExists(t *testing.T) {
	f := &fake{replies: []string{reply(t, turn(nil, ""))}}
	e, ws := engine(t, f)
	e.Harness = filepath.Join("..", "..", "harness")
	start(t, e, ws)
	rel := filepath.Join("harness", "skills", "user-story", "references", "interview.md")
	want := filepath.Join(e.Harness, "skills", "user-story", "references", "interview.md")
	if !strings.Contains(f.prompts[0], want) {
		t.Fatalf("prompt must name %s", want)
	}
	if _, err := os.Stat(filepath.Join("..", "..", rel)); err != nil {
		t.Fatalf("reference file missing: %v", err)
	}
}

func TestListReportsCorruptSessionAndKeepsValid(t *testing.T) {
	ws := t.TempDir()
	good, err := Create(ws, StartInput{Workspace: ws, Idea: "ok"})
	if err != nil {
		t.Fatal(err)
	}
	badDir := filepath.Join(Dir(ws), "broken")
	if err := os.MkdirAll(badDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(badDir, "session.json"), []byte("{nope"), 0o644); err != nil {
		t.Fatal(err)
	}
	list, err := List(ws)
	if err == nil || !strings.Contains(err.Error(), "broken") {
		t.Fatalf("want error naming broken, got %v", err)
	}
	if len(list) != 1 || list[0].ID != good.ID {
		t.Fatalf("valid sessions must still be returned: %+v", list)
	}
}

func TestUnreadableContextFailsTurn(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads everything")
	}
	f := &fake{replies: []string{reply(t, turn(nil, ""))}}
	e, ws := engine(t, f)
	file := writeContext(t, ws, "x")
	if err := os.Chmod(file, 0); err != nil {
		t.Fatal(err)
	}
	_, err := e.Start(context.Background(), StartInput{Workspace: ws, Idea: "x"})
	if err == nil || !strings.Contains(err.Error(), "context") {
		t.Fatalf("want context read error, got %v", err)
	}
	if len(f.prompts) != 0 {
		t.Fatal("the agent must not run with a broken context")
	}
}

func TestRunnerErrorAndRepoChangeBothSurface(t *testing.T) {
	f := &fake{err: errors.New("agent crashed")}
	e, ws := engine(t, f)
	f.effect = func(dir string) {
		if err := os.WriteFile(filepath.Join(dir, "touched.txt"), []byte("x"), 0o644); err != nil {
			t.Error(err)
		}
	}
	_, err := e.Start(context.Background(), StartInput{Workspace: ws, Idea: "x"})
	if codeOf(err) != "shape_repo_changed" || !strings.Contains(err.Error(), "agent crashed") || !strings.Contains(err.Error(), "touched.txt") {
		t.Fatalf("both failures must surface with the code, got %v", err)
	}
}

func TestRunnerErrorAndGitFailureBothSurface(t *testing.T) {
	f := &fake{err: errors.New("agent crashed")}
	e, ws := engine(t, f)
	f.effect = func(dir string) {
		if err := os.RemoveAll(filepath.Join(dir, ".git")); err != nil {
			t.Error(err)
		}
	}
	_, err := e.Start(context.Background(), StartInput{Workspace: ws, Idea: "x"})
	if codeOf(err) != "shape_not_git" || !strings.Contains(err.Error(), "agent crashed") {
		t.Fatalf("both failures must surface, got %v", err)
	}
}

func TestGuardSeesSecondEditOfDirtyFile(t *testing.T) {
	f := &fake{replies: []string{reply(t, turn(nil, "s"))}}
	e, ws := engine(t, f)
	dirty := filepath.Join(ws, "dirty.txt")
	if err := os.WriteFile(dirty, []byte("one"), 0o644); err != nil {
		t.Fatal(err)
	}
	f.effect = func(string) {
		if err := os.WriteFile(dirty, []byte("two"), 0o644); err != nil {
			t.Error(err)
		}
	}
	_, err := e.Start(context.Background(), StartInput{Workspace: ws, Idea: "x"})
	if codeOf(err) != "shape_repo_changed" || !strings.Contains(err.Error(), "dirty.txt") {
		t.Fatalf("want shape_repo_changed naming dirty.txt, got %v", err)
	}
}

func TestGuardAllowsUntouchedDirtyFile(t *testing.T) {
	f := &fake{replies: []string{reply(t, turn(nil, "s"))}}
	e, ws := engine(t, f)
	if err := os.WriteFile(filepath.Join(ws, "dirty.txt"), []byte("one"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Start(context.Background(), StartInput{Workspace: ws, Idea: "x"}); err != nil {
		t.Fatalf("untouched dirty file must pass: %v", err)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.email=a@b", "-c", "user.name=a"}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v %s", args, err, out)
	}
}

func committedEngine(t *testing.T) (*Engine, *fake, string) {
	t.Helper()
	f := &fake{replies: []string{reply(t, turn(nil, "s"))}}
	e, ws := engine(t, f)
	if err := os.WriteFile(filepath.Join(ws, "a.txt"), []byte("1"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, ws, "add", "-A")
	runGit(t, ws, "commit", "-qm", "init")
	return e, f, ws
}

func TestGitGuardSeesHistoryAndRefsMovesOnACleanTree(t *testing.T) {
	cases := map[string][]string{
		"empty commit": {"commit", "-q", "--allow-empty", "-m", "x"},
		"new branch":   {"branch", "other"},
		"new tag":      {"tag", "v1"},
		"checkout":     {"checkout", "-q", "-b", "elsewhere"},
	}
	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			e, f, ws := committedEngine(t)
			f.effect = func(dir string) { runGit(t, dir, args...) }
			_, err := e.Start(context.Background(), StartInput{Workspace: ws, Idea: "x"})
			if codeOf(err) != "shape_repo_changed" {
				t.Fatalf("want shape_repo_changed, got %v", err)
			}
		})
	}
}

func TestGitGuardAcceptsATurnThatMovesNothing(t *testing.T) {
	e, _, ws := committedEngine(t)
	if _, err := e.Start(context.Background(), StartInput{Workspace: ws, Idea: "x"}); err != nil {
		t.Fatal(err)
	}
}

func TestRepoMustStayInsideTheWorkspace(t *testing.T) {
	ws := gitRepo(t)
	other := gitRepo(t)
	rel, err := filepath.Rel(ws, other)
	if err != nil {
		t.Fatal(err)
	}
	for _, repo := range []string{rel, other, "../x"} {
		f := &fake{replies: []string{reply(t, turn(nil, "s"))}}
		_, err := newEngine(f).Start(context.Background(), StartInput{Workspace: ws, Repo: repo, Idea: "x"})
		if codeOf(err) != "shape_repo_outside" || len(f.prompts) != 0 {
			t.Fatalf("repo %q: want shape_repo_outside and no agent run, got %v", repo, err)
		}
	}
	inner := filepath.Join(ws, "inner")
	if err := os.MkdirAll(inner, 0o755); err != nil {
		t.Fatal(err)
	}
	initGit(t, inner)
	f := &fake{replies: []string{reply(t, turn(nil, "s"))}}
	if _, err := newEngine(f).Start(context.Background(), StartInput{Workspace: ws, Repo: "inner", Idea: "x"}); err != nil {
		t.Fatalf("a repo inside the workspace must be accepted: %v", err)
	}
}
