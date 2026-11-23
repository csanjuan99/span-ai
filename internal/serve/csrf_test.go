package serve

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"span/internal/setup"
)

// quickstartHandler returns a handler whose launcher records whether it ran. A launch is
// what starts an agent with write access, so it is the effect a forged request must not reach.
func quickstartHandler(t *testing.T, token string) (http.Handler, *int) {
	t.Helper()
	launched := new(int)
	h := New(t.TempDir(), token, func(context.Context, setup.Request) (Outcome, error) {
		*launched++
		return Outcome{ID: "span-1", Stage: "intake"}, nil
	}, nil, nil, nil)
	return h, launched
}

func postQuickstart(h http.Handler, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/quickstart", strings.NewReader("workspace=.&story=Charge"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestAForeignPageCannotStartARun(t *testing.T) {
	h, launched := quickstartHandler(t, "")
	rec := postQuickstart(h, map[string]string{"Origin": "https://evil.example"})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status %d, want 403", rec.Code)
	}
	if *launched != 0 {
		t.Fatal("a cross-origin POST reached the launcher")
	}
}

func TestACrossSiteFetchMetadataIsRefused(t *testing.T) {
	h, launched := quickstartHandler(t, "")
	rec := postQuickstart(h, map[string]string{"Sec-Fetch-Site": "cross-site"})
	if rec.Code != http.StatusForbidden || *launched != 0 {
		t.Fatalf("status %d launched %d", rec.Code, *launched)
	}
}

func TestEveryStateChangingRouteIsProtected(t *testing.T) {
	h, _ := quickstartHandler(t, "")
	for _, path := range []string{"/spans/span-1/stop", "/spans/span-1/rewind", "/jira/account"} {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader("stage=build"))
		req.Header.Set("Origin", "https://evil.example")
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s: status %d, want 403", path, rec.Code)
		}
	}
}

func TestTheOwnPageAndPlainClientsStillWork(t *testing.T) {
	h, launched := quickstartHandler(t, "")
	same := postQuickstart(h, map[string]string{"Origin": "http://example.com", "Idempotency-Key": "page"})
	if same.Code != http.StatusAccepted {
		t.Fatalf("same-origin page: status %d", same.Code)
	}
	plain := postQuickstart(h, map[string]string{"Idempotency-Key": "cli"})
	if plain.Code != http.StatusAccepted {
		t.Fatalf("a client that sends no Origin (curl, the CLI): status %d", plain.Code)
	}
	if *launched != 2 {
		t.Fatalf("launched %d, want 2", *launched)
	}
}

func TestReadsAreNotBlockedByTheOriginCheck(t *testing.T) {
	h, _ := quickstartHandler(t, "")
	req := httptest.NewRequest(http.MethodGet, "/spans", nil)
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code == http.StatusForbidden {
		t.Fatal("a GET was refused by the cross-origin check")
	}
}

func TestBearerTokenStillOpensAndAWrongOneDoesNot(t *testing.T) {
	h, launched := quickstartHandler(t, "secret")
	if rec := postQuickstart(h, map[string]string{"Authorization": "Bearer secret"}); rec.Code != http.StatusAccepted {
		t.Fatalf("right token: status %d", rec.Code)
	}
	for _, bad := range []string{"Bearer wrong", "Bearer secre", "Bearer secret ", "secret", ""} {
		if rec := postQuickstart(h, map[string]string{"Authorization": bad}); rec.Code != http.StatusUnauthorized {
			t.Errorf("token %q: status %d, want 401", bad, rec.Code)
		}
	}
	if *launched != 1 {
		t.Fatalf("launched %d, want 1", *launched)
	}
}

func TestLoopbackOnlyRefusesARebindingHost(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	cases := map[string]int{
		"127.0.0.1:8787":      http.StatusNoContent,
		"127.0.0.1":           http.StatusNoContent,
		"localhost:8787":      http.StatusNoContent,
		"[::1]:8787":          http.StatusNoContent,
		"localhost.":          http.StatusNoContent,
		"LOCALHOST:8787":      http.StatusNoContent,
		"evil.example":        http.StatusForbidden,
		"evil.example:8787":   http.StatusForbidden,
		"localhost.evil.test": http.StatusForbidden,
		"10.0.0.5:8787":       http.StatusForbidden,
		"":                    http.StatusForbidden,
	}
	for host, want := range cases {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Host = host
		rec := httptest.NewRecorder()
		loopbackOnly(ok).ServeHTTP(rec, req)
		if rec.Code != want {
			t.Errorf("host %q: status %d, want %d", host, rec.Code, want)
		}
	}
}
