package serve

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMetricsAreServedAsJSONWithEveryListPresent(t *testing.T) {
	h := Handler(t.TempDir(), "")
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"runs", "stages", "agents", "recent"} {
		raw, ok := body[key]
		if !ok || string(raw) == "null" {
			t.Fatalf("%s is %q: the page cannot map over it", key, raw)
		}
	}
}

func TestMetricsNeedTheTokenLikeEveryOtherRoute(t *testing.T) {
	h := Handler(t.TempDir(), "secret")
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d", rec.Code)
	}
}
