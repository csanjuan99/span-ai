package serve

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"span/internal/shape"
)

func ruleReply(t *testing.T) func() string {
	return func() string {
		return agentReply(t, map[string]any{"map": map[string]any{
			"rules":     []any{map[string]any{"id": "R1", "text": "A customer sees why a delivery failed.", "source": map[string]string{"kind": "prompt", "ref": "idea"}}},
			"questions": []any{question("Q1")},
		}})
	}
}

func TestShapeRevisionsDiffAndRestore(t *testing.T) {
	rig := newShapeRig(t, ruleReply(t))
	s := rig.start()
	rec := rig.do(http.MethodPost, "/shapes/"+s.ID+"/answers", map[string]any{"workspace": rig.ws, "answers": []map[string]string{{"question": "Q1", "text": "admins"}}})
	if rec.Code != http.StatusOK {
		t.Fatalf("answer %d %s", rec.Code, rec.Body)
	}
	base := "/shapes/" + s.ID
	ws := url.Values{"workspace": {rig.ws}}.Encode()
	rec = rig.do(http.MethodGet, base+"/revisions?"+ws, nil)
	var list struct {
		Revisions []shape.RevisionInfo `json:"revisions"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil || rec.Code != http.StatusOK || len(list.Revisions) < 3 {
		t.Fatalf("revisions %d %s", rec.Code, rec.Body)
	}
	last := list.Revisions[len(list.Revisions)-1]
	if !last.Current || !strings.HasPrefix(last.Label, "Answered Q1") {
		t.Fatalf("last %+v", last)
	}
	rec = rig.do(http.MethodGet, base+"/diff?"+ws+"&from=r1&to="+last.ID, nil)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"kind":"question"`) {
		t.Fatalf("diff %d %s", rec.Code, rec.Body)
	}
	rec = rig.do(http.MethodGet, base+"/diff?"+ws+"&from=r1&to=r99", nil)
	if rec.Code != http.StatusNotFound || errCode(t, rec) != "shape_revision_not_found" {
		t.Fatalf("unknown revision %d %s", rec.Code, rec.Body)
	}
	before := len(list.Revisions)
	rec = rig.do(http.MethodPost, base+"/restore", map[string]any{"workspace": rig.ws, "revision": "r2"})
	if rec.Code != http.StatusOK || rig.session(rec).Map.Questions[0].Status != shape.QuestionOpen {
		t.Fatalf("restore %d %s", rec.Code, rec.Body)
	}
	rec = rig.do(http.MethodGet, base+"/revisions?"+ws, nil)
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil || len(list.Revisions) != before+1 || list.Revisions[before].Label != "Restored r2" {
		t.Fatalf("history after restore %s", rec.Body)
	}
}

func TestShapeCompareMergeAndDismiss(t *testing.T) {
	rig := newShapeRig(t, ruleReply(t))
	a, b := rig.start(), rig.start()
	ids := url.Values{"workspace": {rig.ws}, "ids": {a.ID + "," + b.ID}}.Encode()
	rec := rig.do(http.MethodGet, "/shapes/compare?"+ids, nil)
	var got shape.Comparison
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || rec.Code != http.StatusOK || len(got.Same) != 1 || len(got.Same[0].Members) != 2 || !got.Estimate {
		t.Fatalf("compare %d %s", rec.Code, rec.Body)
	}
	rec = rig.do(http.MethodGet, "/shapes/compare?workspace="+rig.ws+"&ids="+a.ID, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("one session: %d %s", rec.Code, rec.Body)
	}
	rec = rig.do(http.MethodPost, "/shapes/compare/dismiss", map[string]any{"workspace": rig.ws, "ids": []string{a.ID, b.ID}})
	if rec.Code != http.StatusOK {
		t.Fatalf("dismiss %d %s", rec.Code, rec.Body)
	}
	rec = rig.do(http.MethodPost, "/shapes/merge", map[string]any{"workspace": rig.ws, "into": a.ID, "from": []string{b.ID}})
	var merged struct {
		Merged []string `json:"merged"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &merged); err != nil || rec.Code != http.StatusOK || len(merged.Merged) != 1 {
		t.Fatalf("merge %d %s", rec.Code, rec.Body)
	}
	if after := rig.do(http.MethodGet, "/shapes/"+b.ID+"?workspace="+rig.ws, nil); rig.session(after).MergedInto != a.ID {
		t.Fatalf("source not linked: %s", after.Body)
	}
}

func TestShapeMergeRefusesASealedSession(t *testing.T) {
	rig := newShapeRig(t, readyReply(t))
	sealed := rig.sealed()
	open := rig.start()
	rec := rig.do(http.MethodPost, "/shapes/merge", map[string]any{"workspace": rig.ws, "into": open.ID, "from": []string{sealed.ID}})
	if rec.Code != http.StatusConflict || errCode(t, rec) != "shape_sealed" {
		t.Fatalf("sealed merge %d %s", rec.Code, rec.Body)
	}
}
