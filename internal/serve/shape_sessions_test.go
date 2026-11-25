package serve

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"context"

	"span/internal/jira"
	"span/internal/shape"
)

func (r *shapeRig) views(body fmt.Stringer) []shape.View {
	r.t.Helper()
	var out []shape.View
	if err := json.Unmarshal([]byte(body.String()), &out); err != nil {
		r.t.Fatalf("views: %v", err)
	}
	return out
}

func TestShapeListHasTheDerivedFields(t *testing.T) {
	rig := newShapeRig(t, openReply(t))
	s := rig.start()
	rec := rig.do(http.MethodGet, "/shapes?workspace="+rig.ws, nil)
	got := rig.views(rec.Body)
	if len(got) != 1 || got[0].ID != s.ID {
		t.Fatalf("list %s", rec.Body)
	}
	row := got[0]
	if !strings.HasPrefix(row.Ref, "SES-") || row.Title != "a login" || row.OpenQuestions != 1 || row.UpdatedAt.IsZero() {
		t.Fatalf("derived fields %+v", row)
	}
	if one := rig.do(http.MethodGet, "/shapes/"+s.ID+"?workspace="+rig.ws, nil); !strings.Contains(one.Body.String(), `"openQuestions":1`) {
		t.Fatalf("get lacks derived fields: %s", one.Body)
	}
}

func TestShapeListFiltersByStatusQueryAndArchived(t *testing.T) {
	rig := newShapeRig(t, openReply(t))
	open := rig.start()
	rig.run = func(context.Context, string, string) (string, error) { return readyReply(t)(), nil }
	ready := rig.start()
	count := func(query string) int {
		rec := rig.do(http.MethodGet, "/shapes?workspace="+rig.ws+query, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: %d %s", query, rec.Code, rec.Body)
		}
		return len(rig.views(rec.Body))
	}
	if count("&status=ready") != 1 || count("&status=sealed") != 0 || count("&q=LOGIN") != 2 || count("&q=zzz") != 0 {
		t.Fatal("status and query filters do not narrow the list")
	}
	if rec := rig.do(http.MethodPost, "/shapes/"+open.ID+"/archive", map[string]any{"workspace": rig.ws}); rec.Code != http.StatusOK {
		t.Fatalf("archive %d %s", rec.Code, rec.Body)
	}
	if count("") != 1 || count("&archived=only") != 1 || count("&archived=include") != 2 {
		t.Fatal("archived sessions are not hidden by default")
	}
	if got := rig.views(rig.do(http.MethodGet, "/shapes?workspace="+rig.ws, nil).Body); got[0].ID != ready.ID {
		t.Fatalf("archived session was listed: %+v", got)
	}
	if rec := rig.do(http.MethodGet, "/shapes?workspace="+rig.ws+"&status=done", nil); rec.Code != http.StatusBadRequest || errCode(t, rec) != "invalid" {
		t.Fatalf("unknown status %d %s", rec.Code, rec.Body)
	}
}

func TestShapeArchiveCanBeUndone(t *testing.T) {
	rig := newShapeRig(t, openReply(t))
	s := rig.start()
	rig.do(http.MethodPost, "/shapes/"+s.ID+"/archive", map[string]any{"workspace": rig.ws})
	rec := rig.do(http.MethodPost, "/shapes/"+s.ID+"/archive", map[string]any{"workspace": rig.ws, "archived": false})
	if rec.Code != http.StatusOK || rig.session(rec).Archived {
		t.Fatalf("unarchive %d %s", rec.Code, rec.Body)
	}
}

func TestShapeDeleteRemovesOpenAndRefusesSealed(t *testing.T) {
	rig := newShapeRig(t, openReply(t))
	s := rig.start()
	if rec := rig.do(http.MethodDelete, "/shapes/"+s.ID+"?workspace="+rig.ws, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("delete %d %s", rec.Code, rec.Body)
	}
	if rec := rig.do(http.MethodGet, "/shapes/"+s.ID+"?workspace="+rig.ws, nil); rec.Code != http.StatusNotFound {
		t.Fatalf("get after delete %d", rec.Code)
	}
	ready := newShapeRig(t, readyReply(t))
	done := ready.sealed()
	rec := ready.do(http.MethodDelete, "/shapes/"+done.ID+"?workspace="+ready.ws, nil)
	if rec.Code != http.StatusConflict || errCode(t, rec) != "shape_sealed" {
		t.Fatalf("delete sealed %d %s", rec.Code, rec.Body)
	}
	if rec := ready.do(http.MethodPost, "/shapes/"+done.ID+"/archive", map[string]any{"workspace": ready.ws}); rec.Code != http.StatusOK {
		t.Fatalf("archive sealed %d %s", rec.Code, rec.Body)
	}
}

func TestShapeFailedTurnIsSavedAndResumedOnAnotherAgent(t *testing.T) {
	rig := newShapeRig(t, openReply(t))
	s := rig.start()
	rig.run = func(context.Context, string, string) (string, error) {
		return "", errors.New("claude-code hit a usage limit")
	}
	body := map[string]any{"workspace": rig.ws, "answers": []any{map[string]string{"question": "Q1", "text": "ops"}}}
	if rec := rig.do(http.MethodPost, "/shapes/"+s.ID+"/answers", body); rec.Code < 400 {
		t.Fatalf("the failed turn answered %d", rec.Code)
	}
	one := rig.session(rig.do(http.MethodGet, "/shapes/"+s.ID+"?workspace="+rig.ws, nil))
	if one.Failure == nil || one.Failure.Kind != shape.FailureUsageLimit || one.Map.Questions[0].Answer != "ops" {
		t.Fatalf("failure not kept: %+v", one)
	}
	rig.run = func(context.Context, string, string) (string, error) { return readyReply(t)(), nil }
	rec := rig.do(http.MethodPost, "/shapes/"+s.ID+"/resume", map[string]any{"workspace": rig.ws, "agent": "codex", "model": "o3"})
	got := rig.session(rec)
	if rec.Code != http.StatusOK || got.Failure != nil || got.Agent != "codex" || got.Model != "o3" {
		t.Fatalf("resume %d %s", rec.Code, rec.Body)
	}
}

func TestShapeSetAgentRunsNoTurn(t *testing.T) {
	rig := newShapeRig(t, openReply(t))
	s := rig.start()
	rig.run = func(context.Context, string, string) (string, error) { return "", errors.New("must not run") }
	rec := rig.do(http.MethodPost, "/shapes/"+s.ID+"/agent", map[string]any{"workspace": rig.ws, "agent": "codex"})
	if rec.Code != http.StatusOK || rig.session(rec).Agent != "codex" {
		t.Fatalf("set agent %d %s", rec.Code, rec.Body)
	}
}

func TestShapeStartFromAJiraIssue(t *testing.T) {
	rig := newShapeRig(t, openReply(t))
	rig.backlog.issue = jira.Story{Key: "BILL-2031", Summary: "Show tax", Brief: "Show tax per line on the PDF"}
	var prompt string
	rig.run = func(_ context.Context, _, p string) (string, error) { prompt = p; return openReply(t)(), nil }
	rec := rig.do(http.MethodPost, "/shapes", map[string]any{"workspace": rig.ws, "issue": "bill-2031", "idea": "only for EU"})
	var view shape.View
	if err := json.Unmarshal(rec.Body.Bytes(), &view); err != nil || rec.Code != http.StatusCreated {
		t.Fatalf("start %d %s", rec.Code, rec.Body)
	}
	if view.SourceIssue != "BILL-2031" || view.JiraKey != "BILL-2031" || !strings.Contains(view.Idea, "Show tax per line on the PDF") || !strings.Contains(view.Idea, "only for EU") {
		t.Fatalf("session %+v", view.Session)
	}
	if !strings.Contains(prompt, "Show tax per line") {
		t.Fatalf("the agent was not given the issue: %s", prompt)
	}
}

func TestShapeStartFromAMissingIssueStartsNothing(t *testing.T) {
	rig := newShapeRig(t, openReply(t))
	rig.backlog.err = &jira.Error{Code: "not_found", Msg: "no such issue"}
	rec := rig.do(http.MethodPost, "/shapes", map[string]any{"workspace": rig.ws, "issue": "BILL-1"})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("start %d %s", rec.Code, rec.Body)
	}
	list, err := shape.List(rig.ws)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("a session was left: %d", len(list))
	}
}
