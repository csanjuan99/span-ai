package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProbeHealthRejectsAMalformedAnswer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if _, err := w.Write([]byte("<html>not json</html>")); err != nil {
			t.Error(err)
		}
	}))
	defer srv.Close()
	if _, _, err := probeHealth(srv.URL, ""); err == nil {
		t.Fatal("a malformed body must be an error, not a connected server")
	}
}

func TestProbeHealthReportsARefusedToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()
	body, refused, err := probeHealth(srv.URL, "wrong")
	if err != nil || !refused || body.Status != "refused" {
		t.Fatalf("body = %+v refused = %v err = %v", body, refused, err)
	}
}
