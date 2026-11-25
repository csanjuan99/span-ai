package serve

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"span/internal/shape"
)

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("body %q: %v", rec.Body.String(), err)
	}
	return out
}

func TestBoardGroupsByStatusAndCountsWhatNeedsYou(t *testing.T) {
	rig := newShapeRig(t, openReply(t))
	rig.start()
	rig.start()
	rec := rig.do(http.MethodGet, "/shapes?group=status&workspace="+rig.ws+"&limit=1", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	view := decode[shape.BoardView](t, rec)
	if view.InFlight != 2 || view.NeedYou != 2 || len(view.Columns) != 4 {
		t.Fatalf("%+v", view)
	}
	if col := view.Columns[0]; col.Key != shape.StatusInterviewing || col.Count != 2 || len(col.Cards) != 1 || col.Hidden != 1 {
		t.Fatalf("column %+v", col)
	}
	if bad := rig.do(http.MethodGet, "/shapes?group=status&workspace="+rig.ws+"&limit=-3", nil); bad.Code != http.StatusBadRequest {
		t.Fatalf("a bad limit is refused: %d", bad.Code)
	}
	if plain := rig.do(http.MethodGet, "/shapes?workspace="+rig.ws, nil); !strings.HasPrefix(plain.Body.String(), "[") {
		t.Fatalf("the plain list is unchanged: %s", plain.Body)
	}
}

func TestBulkAssumesAndAnswersPerSession(t *testing.T) {
	rig := newShapeRig(t, openReply(t))
	a, b := rig.start(), rig.start()
	dry := rig.do(http.MethodPost, "/shapes/bulk", map[string]any{"workspace": rig.ws, "action": "assume", "ids": []string{a.ID, b.ID}, "dryRun": true})
	if dry.Code != http.StatusOK {
		t.Fatalf("%d %s", dry.Code, dry.Body)
	}
	if res := decode[shape.BulkResult](t, dry); !res.DryRun || res.Done != 2 || len(res.Items[0].Changed) != 1 {
		t.Fatalf("dry run %+v", res)
	}
	if got := rig.do(http.MethodGet, "/shapes/"+a.ID+"?workspace="+rig.ws, nil); rig.session(got).Map.Questions[0].Status != shape.QuestionOpen {
		t.Fatal("a dry run changes nothing")
	}
	real := rig.do(http.MethodPost, "/shapes/bulk", map[string]any{"workspace": rig.ws, "action": "assume", "ids": []string{a.ID, "missing"}})
	res := decode[shape.BulkResult](t, real)
	if real.Code != http.StatusOK || res.Done != 1 || res.Failed != 1 || res.Items[1].Code != "shape_not_found" {
		t.Fatalf("%d %+v", real.Code, res)
	}
	if got := rig.do(http.MethodGet, "/shapes/"+a.ID+"?workspace="+rig.ws, nil); rig.session(got).Map.Questions[0].Status != shape.QuestionAssumed {
		t.Fatal("the assumption was accepted")
	}
	if bad := rig.do(http.MethodPost, "/shapes/bulk", map[string]any{"workspace": rig.ws, "action": "explode", "ids": []string{a.ID}}); bad.Code != http.StatusBadRequest {
		t.Fatalf("an unknown action is refused: %d", bad.Code)
	}
}

func TestActivityListsAgentStepsAndStreamsWhenFollowed(t *testing.T) {
	rig := newShapeRig(t, openReply(t))
	rig.start()
	rec := rig.do(http.MethodGet, "/shapes/activity?workspace="+rig.ws, nil)
	feed := decode[[]shape.Activity](t, rec)
	if rec.Code != http.StatusOK || len(feed) == 0 || feed[0].Actor == "you" {
		t.Fatalf("%d %+v", rec.Code, feed)
	}
	if bad := rig.do(http.MethodGet, "/shapes/activity?workspace="+rig.ws+"&since=yesterday", nil); bad.Code != http.StatusBadRequest {
		t.Fatalf("a bad time is refused: %d", bad.Code)
	}
	server := httptest.NewServer(rig.handler)
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/shapes/activity?follow=1&workspace="+rig.ws, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("content type %q", ct)
	}
	scanner := bufio.NewScanner(resp.Body)
	var event, data string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event: ") {
			event = strings.TrimPrefix(line, "event: ")
		}
		if strings.HasPrefix(line, "data: ") {
			data = strings.TrimPrefix(line, "data: ")
			break
		}
	}
	var step shape.Activity
	if err := json.Unmarshal([]byte(data), &step); err != nil || event != "step" || step.Text == "" {
		t.Fatalf("event %q data %q %v", event, data, err)
	}
}

func TestHistoryAndUndoOverHTTP(t *testing.T) {
	rig := newShapeRig(t, openReply(t))
	s := rig.start()
	assumed := rig.do(http.MethodPost, "/shapes/"+s.ID+"/answers", map[string]any{"workspace": rig.ws, "assume": []string{"Q1"}})
	if assumed.Code != http.StatusOK {
		t.Fatalf("%d %s", assumed.Code, assumed.Body)
	}
	rec := rig.do(http.MethodGet, "/shapes/"+s.ID+"/history?workspace="+rig.ws+"&actor=you&kind=assumptions", nil)
	view := decode[shape.HistoryView](t, rec)
	if rec.Code != http.StatusOK || len(view.Events) != 1 || !view.Events[0].Undoable || view.Counts.Everyone < 3 {
		t.Fatalf("%d %+v", rec.Code, view)
	}
	step := view.Events[0].ID
	plan := decode[shape.UndoPlan](t, rig.do(http.MethodGet, "/shapes/"+s.ID+"/history/"+step+"?workspace="+rig.ws, nil))
	if !plan.UndoOnlyThis || len(plan.Changes) != 1 {
		t.Fatalf("plan %+v", plan)
	}
	undone := rig.do(http.MethodPost, "/shapes/"+s.ID+"/history/"+step+"/undo", map[string]any{"workspace": rig.ws})
	if undone.Code != http.StatusOK {
		t.Fatalf("%d %s", undone.Code, undone.Body)
	}
	got := rig.do(http.MethodGet, "/shapes/"+s.ID+"?workspace="+rig.ws, nil)
	if rig.session(got).Map.Questions[0].Status != shape.QuestionOpen {
		t.Fatal("the question is open again")
	}
	if again := rig.do(http.MethodPost, "/shapes/"+s.ID+"/history/"+step+"/undo", map[string]any{"workspace": rig.ws}); again.Code != http.StatusConflict || errCode(t, again) != "shape_not_undoable" {
		t.Fatalf("%d %s", again.Code, again.Body)
	}
	if missing := rig.do(http.MethodGet, "/shapes/"+s.ID+"/history/h99?workspace="+rig.ws, nil); missing.Code != http.StatusNotFound {
		t.Fatalf("%d", missing.Code)
	}
	if bad := rig.do(http.MethodGet, "/shapes/"+s.ID+"/history?workspace="+rig.ws+"&actor=robots", nil); bad.Code != http.StatusBadRequest {
		t.Fatalf("%d", bad.Code)
	}
}

func TestSealedStoryRefusesUndoOverHTTP(t *testing.T) {
	rig := newShapeRig(t, readyReply(t))
	s := rig.sealed()
	rec := rig.do(http.MethodPost, "/shapes/"+s.ID+"/history/h1/undo", map[string]any{"workspace": rig.ws, "cascade": true})
	if rec.Code != http.StatusConflict || errCode(t, rec) != "shape_sealed" {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
}

const importedFile = "# Tax per line\n\n## Tax is shown on each line\n- 100 with 20% shows 20\n\n## Notes\nFrom finance.\n"

func TestImportMarkdownPreviewsThenStartsAndRunsTheInterview(t *testing.T) {
	rig := newShapeRig(t, openReply(t))
	dry := rig.do(http.MethodPost, "/shapes/import", map[string]any{"workspace": rig.ws, "markdown": importedFile, "name": "story.md", "dryRun": true})
	preview := decode[shape.MarkdownPreview](t, dry)
	if dry.Code != http.StatusOK || len(preview.Rules) != 1 || len(preview.Examples) != 1 || len(preview.Notes) != 1 {
		t.Fatalf("%d %+v", dry.Code, preview)
	}
	if list := rig.do(http.MethodGet, "/shapes?workspace="+rig.ws, nil); strings.Contains(list.Body.String(), "id") {
		t.Fatalf("a dry run starts nothing: %s", list.Body)
	}
	made := rig.do(http.MethodPost, "/shapes/import", map[string]any{"workspace": rig.ws, "markdown": importedFile, "name": "story.md", "run": false})
	if made.Code != http.StatusCreated {
		t.Fatalf("%d %s", made.Code, made.Body)
	}
	var out struct {
		Session shape.Session `json:"session"`
	}
	if err := json.Unmarshal(made.Body.Bytes(), &out); err != nil || len(out.Session.Map.Rules) != 1 || out.Session.Map.Rules[0].Source.Kind != shape.SourceImport {
		t.Fatalf("%s %v", made.Body, err)
	}
	if both := rig.do(http.MethodPost, "/shapes/import", map[string]any{"workspace": rig.ws, "markdown": importedFile, "jira": "BILL-1"}); both.Code != http.StatusBadRequest {
		t.Fatalf("both sources are refused: %d", both.Code)
	}
	if empty := rig.do(http.MethodPost, "/shapes/import", map[string]any{"workspace": rig.ws}); empty.Code != http.StatusBadRequest {
		t.Fatalf("neither source is refused: %d", empty.Code)
	}
}

func TestImportJiraReadsTheStoryAndStartsFromIt(t *testing.T) {
	rig := newShapeRig(t, openReply(t))
	rig.backlog.issue.Key, rig.backlog.issue.Summary, rig.backlog.issue.Status = "BILL-2031", "Tax per line", "Done"
	rig.backlog.issue.Brief, rig.backlog.issue.Acceptance = "One.\n\nTwo.", "- a\n- b"
	dry := rig.do(http.MethodPost, "/shapes/import", map[string]any{"workspace": rig.ws, "jira": "BILL-2031", "dryRun": true})
	read := decode[shape.StoryRead](t, dry)
	if dry.Code != http.StatusOK || read.Paragraphs != 2 || read.Criteria != 2 || read.CommentRead != "not read" {
		t.Fatalf("%d %+v", dry.Code, read)
	}
	made := rig.do(http.MethodPost, "/shapes/import", map[string]any{"workspace": rig.ws, "jira": "BILL-2031"})
	if made.Code != http.StatusCreated || rig.session(made).SourceIssue != "BILL-2031" {
		t.Fatalf("%d %s", made.Code, made.Body)
	}
	if rig.backlog.createCount() != 0 {
		t.Fatal("importing never writes to Jira")
	}
}

func TestExportAndShareNeedASealedStory(t *testing.T) {
	rig := newShapeRig(t, readyReply(t))
	draft := rig.start()
	if rec := rig.do(http.MethodGet, "/shapes/"+draft.ID+"/export?workspace="+rig.ws, nil); rec.Code != http.StatusConflict || errCode(t, rec) != "shape_not_sealed" {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	if rec := rig.do(http.MethodPost, "/shapes/"+draft.ID+"/shares", map[string]any{"workspace": rig.ws}); rec.Code != http.StatusConflict {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	sealed := rig.sealed()
	rec := rig.do(http.MethodGet, "/shapes/"+sealed.ID+"/export?workspace="+rig.ws+"&format=json&digest=1&sources=1", nil)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Header().Get("Content-Disposition"), "story.json") || !strings.Contains(rec.Body.String(), sealed.Digest) {
		t.Fatalf("%d %v %s", rec.Code, rec.Header(), rec.Body)
	}
	md := rig.do(http.MethodGet, "/shapes/"+sealed.ID+"/export?workspace="+rig.ws+"&format=md", nil)
	if !strings.HasPrefix(md.Body.String(), "# Log in") || !strings.HasPrefix(md.Header().Get("Content-Type"), "text/markdown") {
		t.Fatalf("%s %v", md.Body, md.Header())
	}
	if bad := rig.do(http.MethodGet, "/shapes/"+sealed.ID+"/export?workspace="+rig.ws+"&format=pdf", nil); bad.Code != http.StatusBadRequest {
		t.Fatalf("%d", bad.Code)
	}
}

func TestShareLinkOpensCountsAndRevokes(t *testing.T) {
	rig := newShapeRig(t, readyReply(t))
	sealed := rig.sealed()
	made := rig.do(http.MethodPost, "/shapes/"+sealed.ID+"/shares", map[string]any{"workspace": rig.ws, "days": 3})
	var created struct {
		Link shape.Share `json:"link"`
		Path string      `json:"path"`
	}
	if err := json.Unmarshal(made.Body.Bytes(), &created); err != nil || made.Code != http.StatusCreated || created.Path != "/shaping/"+sealed.ID+"/shared/"+created.Link.Token {
		t.Fatalf("%d %s %v", made.Code, made.Body, err)
	}
	open := rig.do(http.MethodGet, "/shapes/"+sealed.ID+"/shared/"+created.Link.Token+"?workspace="+rig.ws, nil)
	shown := decode[shape.SharedStory](t, open)
	if open.Code != http.StatusOK || shown.Story == "" || shown.Link.Opens != 1 || shown.Digest != sealed.Digest {
		t.Fatalf("%d %+v", open.Code, shown)
	}
	list := decode[[]shape.Share](t, rig.do(http.MethodGet, "/shapes/"+sealed.ID+"/shares?workspace="+rig.ws, nil))
	if len(list) != 1 || list[0].Opens != 1 {
		t.Fatalf("%+v", list)
	}
	if revoked := rig.do(http.MethodDelete, "/shapes/"+sealed.ID+"/shares/"+created.Link.Token+"?workspace="+rig.ws, nil); revoked.Code != http.StatusOK {
		t.Fatalf("%d %s", revoked.Code, revoked.Body)
	}
	gone := rig.do(http.MethodGet, "/shapes/"+sealed.ID+"/shared/"+created.Link.Token+"?workspace="+rig.ws, nil)
	if gone.Code != http.StatusGone || errCode(t, gone) != "shape_share_revoked" {
		t.Fatalf("%d %s", gone.Code, gone.Body)
	}
	if missing := rig.do(http.MethodGet, "/shapes/"+sealed.ID+"/shared/nope?workspace="+rig.ws, nil); missing.Code != http.StatusNotFound {
		t.Fatalf("%d", missing.Code)
	}
	if long := rig.do(http.MethodPost, "/shapes/"+sealed.ID+"/shares", map[string]any{"workspace": rig.ws, "days": 400}); long.Code != http.StatusBadRequest {
		t.Fatalf("%d", long.Code)
	}
}
