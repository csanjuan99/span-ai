package serve

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAMissingRunAnswersNotFoundJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/spans/missing-run", nil)
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()
	Handler(t.TempDir(), "").ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status %d %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body %s: %v", rec.Body.String(), err)
	}
	if body.Error != "not_found" || !strings.Contains(body.Message, "missing-run") {
		t.Fatalf("%+v", body)
	}
}
