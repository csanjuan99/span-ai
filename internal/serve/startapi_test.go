package serve

import (
	"net/http"
	"net/http/httptest"
	"testing"
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
