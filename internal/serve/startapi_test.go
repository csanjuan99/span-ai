package serve

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"span/internal/setup"
	"span/internal/start"
)

func TestHealthAnswersConnected(t *testing.T) {
	mux := http.NewServeMux()
	startRoutes(mux, ".", nil, newRecent())
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestPrecheckRouteOnEmptyGitlessPath(t *testing.T) {
	mux := http.NewServeMux()
	startRoutes(mux, ".", nil, newRecent())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/start/precheck?path=/no/such/path/for/span", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict && rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body %s", rec.Code, rec.Body.String())
	}
}

func TestAPartialStartNamesTheRepositoriesThatFailed(t *testing.T) {
	h := New(t.TempDir(), "", func(context.Context, setup.Request) (Outcome, error) {
		return Outcome{ID: "span-1", Stage: "intake", Partial: true,
			Runs:   []RunRef{{ID: "span-1", Stage: "intake", Repo: "api"}},
			Failed: []RepoFailure{{Repo: "web", Reason: "no base branch"}}}, nil
	}, nil, nil, nil)
	if body := submit(h, "workspace=.&story=Charge", nil).Body.String(); !strings.Contains(body, `"failed":[{"repo":"web","reason":"no base branch"}]`) {
		t.Fatalf("quickstart does not name the failed repository: %s", body)
	}
	req := httptest.NewRequest(http.MethodPost, "/spans", strings.NewReader("workspace=.&story=Charge"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var body struct {
		OK      bool               `json:"ok"`
		Results []start.RepoResult `json:"results"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !body.OK || len(body.Results) != 2 || body.Results[1].OK || body.Results[1].Repo != "web" ||
		body.Results[1].Refusal == nil || body.Results[1].Refusal.Message != "no base branch" {
		t.Fatalf("results = %s", rec.Body)
	}
}
