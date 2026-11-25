package serve

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"span/internal/shape"
)

func proposalReply(t *testing.T) func() string {
	return func() string {
		return agentReply(t, map[string]any{
			"story": "# Log in",
			"proposals": []any{
				map[string]string{"id": "P1", "section": "Access", "text": "- Any user"},
				map[string]string{"id": "P2", "section": "Webhooks", "text": "- 30 days"},
			},
		})
	}
}

func TestShapeContextSkipsRestoresAndPreviewsWithoutWriting(t *testing.T) {
	rig := newShapeRig(t, proposalReply(t))
	s := rig.start()
	query := url.Values{"workspace": {rig.ws}, "accept": {"P1,P2"}}
	rec := rig.do(http.MethodGet, "/shapes/"+s.ID+"/context/preview?"+query.Encode(), nil)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"- Any user"`) || !strings.Contains(rec.Body.String(), `"add"`) {
		t.Fatalf("preview %d %s", rec.Code, rec.Body)
	}
	if _, err := os.Stat(shape.ContextPath(rig.ws)); err == nil {
		t.Fatal("a preview writes nothing")
	}
	rec = rig.do(http.MethodPost, "/shapes/"+s.ID+"/context", map[string]any{"workspace": rig.ws, "accept": []string{"P1"}, "skip": []string{"P2"}})
	got := rig.session(rec)
	if rec.Code != http.StatusOK || got.Proposals[0].Status != shape.ProposalAccepted || got.Proposals[1].Status != shape.ProposalSkipped {
		t.Fatalf("settle %d %s", rec.Code, rec.Body)
	}
	rec = rig.do(http.MethodPost, "/shapes/"+s.ID+"/context", map[string]any{"workspace": rig.ws, "skip": []string{"P1"}})
	if rec.Code != http.StatusConflict || errCode(t, rec) != "shape_context_written" {
		t.Fatalf("a written note cannot be skipped: %d %s", rec.Code, rec.Body)
	}
	rec = rig.do(http.MethodPost, "/shapes/"+s.ID+"/context", map[string]any{"workspace": rig.ws, "restore": []string{"P2"}})
	if rig.session(rec).Proposals[1].Status != shape.ProposalProposed {
		t.Fatalf("restore %s", rec.Body)
	}
}

func splitReply(t *testing.T) func() string {
	return func() string {
		return agentReply(t, map[string]any{"split": []any{
			map[string]string{"story": "Show why", "leftOut": "Retry"},
			map[string]string{"story": "Retry it", "leftOut": "Reasons"},
		}})
	}
}

func TestShapeSplitDecisions(t *testing.T) {
	rig := newShapeRig(t, splitReply(t))
	s := rig.start()
	rec := rig.do(http.MethodPost, "/shapes/"+s.ID+"/split", map[string]any{"workspace": rig.ws, "decision": "maybe"})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unknown decision: %d %s", rec.Code, rec.Body)
	}
	rec = rig.do(http.MethodPost, "/shapes/"+s.ID+"/split", map[string]any{"workspace": rig.ws, "decision": "split"})
	got := rig.session(rec)
	if rec.Code != http.StatusOK || len(got.Children) != 2 {
		t.Fatalf("split %d %s", rec.Code, rec.Body)
	}
	child := rig.do(http.MethodGet, "/shapes/"+got.Children[0]+"?workspace="+url.QueryEscape(rig.ws), nil)
	if rig.session(child).Parent != s.ID {
		t.Fatalf("child %s", child.Body)
	}
	other := newShapeRig(t, readyReply(t))
	plain := other.start()
	rec = other.do(http.MethodPost, "/shapes/"+plain.ID+"/split", map[string]any{"workspace": other.ws, "decision": "keep"})
	if rec.Code != http.StatusConflict || errCode(t, rec) != "shape_no_split" {
		t.Fatalf("nothing to keep: %d %s", rec.Code, rec.Body)
	}
}

func TestShapeKeepWholeWritesTheStory(t *testing.T) {
	calls := 0
	rig := newShapeRig(t, nil)
	rig.run = func(ctx context.Context, _, _ string) (string, error) {
		calls++
		if calls == 1 {
			return splitReply(t)(), nil
		}
		return readyReply(t)(), nil
	}
	s := rig.start()
	rec := rig.do(http.MethodPost, "/shapes/"+s.ID+"/split", map[string]any{"workspace": rig.ws, "decision": "keep"})
	got := rig.session(rec)
	if rec.Code != http.StatusOK || got.Status != shape.StatusReady || !got.SplitKept {
		t.Fatalf("keep %d %s", rec.Code, rec.Body)
	}
}

func TestShapeForkAndSource(t *testing.T) {
	rig := newShapeRig(t, readyReply(t))
	sealed := rig.sealed()
	rec := rig.do(http.MethodPost, "/shapes/"+sealed.ID+"/fork", map[string]any{"workspace": rig.ws})
	forked := rig.session(rec)
	if rec.Code != http.StatusOK || forked.ID == sealed.ID || forked.Parent != sealed.ID || forked.Status == shape.StatusSealed {
		t.Fatalf("fork %d %s", rec.Code, rec.Body)
	}
	other := newShapeRig(t, readyReply(t))
	open := other.start()
	rec = other.do(http.MethodPost, "/shapes/"+open.ID+"/fork", map[string]any{"workspace": other.ws})
	if rec.Code != http.StatusConflict || errCode(t, rec) != "shape_not_sealed" {
		t.Fatalf("an open session cannot be forked: %d %s", rec.Code, rec.Body)
	}
	if err := os.WriteFile(filepath.Join(rig.ws, "a.go"), []byte("one\ntwo\nthree\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	get := func(ref string) *httptest.ResponseRecorder {
		q := url.Values{"workspace": {rig.ws}, "ref": {ref}}
		return rig.do(http.MethodGet, "/shapes/"+sealed.ID+"/source?"+q.Encode(), nil)
	}
	if rec := get("a.go:2"); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"hit":true`) {
		t.Fatalf("source %d %s", rec.Code, rec.Body)
	}
	if rec := get("../x:1"); rec.Code != http.StatusBadRequest {
		t.Fatalf("escape: %d %s", rec.Code, rec.Body)
	}
	if rec := get("missing.go:1"); rec.Code != http.StatusNotFound {
		t.Fatalf("missing: %d %s", rec.Code, rec.Body)
	}
}
