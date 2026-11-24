package jira

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIssueBuildsTheBriefAndKeepsTheTokenOutOfTheRepo(t *testing.T) {
	var auth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		if r.URL.Path != "/rest/api/3/issue/AET-1478" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		reply(t, w, `{
			"key":"AET-1478",
			"names":{"customfield_10034":"Acceptance Criteria"},
			"fields":{
				"summary":"Show channels",
				"issuetype":{"name":"Story"},
				"status":{"name":"To Do"},
				"description":{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"Show the channel list."}]}]},
				"customfield_10034":{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"Includes archived channels."}]}]}
			}
		}`)
	}))
	defer server.Close()
	client := Client{Account: Account{BaseURL: server.URL, Email: "dev@example.com", Token: "secret-token"}, HTTP: server.Client()}
	got, err := client.Issue(context.Background(), "aet-1478")
	if err != nil {
		t.Fatal(err)
	}
	want := "Basic " + base64.StdEncoding.EncodeToString([]byte("dev@example.com:secret-token"))
	if auth != want {
		t.Fatalf("auth %q", auth)
	}
	if got.Key != "AET-1478" || !strings.Contains(got.Brief, "Show the channel list.") || !strings.Contains(got.Brief, "Includes archived channels.") {
		t.Fatalf("%+v", got)
	}
	if strings.Contains(got.Brief, "secret-token") {
		t.Fatal("token leaked into the brief")
	}
}

func TestIssueUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusUnauthorized)
	}))
	defer server.Close()
	client := Client{Account: Account{BaseURL: server.URL, Email: "dev@example.com", Token: "bad"}, HTTP: server.Client()}
	_, err := client.Issue(context.Background(), "AET-1")
	coded, ok := err.(*Error)
	if !ok || coded.Code != "unauthorized" {
		t.Fatalf("%v", err)
	}
}

func TestSaveIsPerProjectAndViewOmitsTheToken(t *testing.T) {
	first := t.TempDir()
	second := t.TempDir()
	if err := Save(first, Account{BaseURL: "https://example.atlassian.net", Email: "dev@example.com", Token: "secret-token"}); err != nil {
		t.Fatal(err)
	}
	if err := Save(second, Account{BaseURL: "https://other.atlassian.net", Email: "other@example.com", Token: "other-token"}); err != nil {
		t.Fatal(err)
	}
	path, err := Path(first)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode %o", info.Mode().Perm())
	}
	loaded, err := Load(first)
	if err != nil || loaded.Token != "secret-token" {
		t.Fatalf("%v %+v", err, loaded)
	}
	other, err := Load(second)
	if err != nil || other.Token != "other-token" {
		t.Fatalf("%v %+v", err, other)
	}
	view := View(loaded, true)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(view.Email, "secret") || view.BaseURL == "" {
		t.Fatalf("%+v", view)
	}
	if !strings.Contains(string(raw), "secret-token") {
		t.Fatal("token was not stored")
	}
	ignore, err := os.ReadFile(filepath.Join(filepath.Dir(path), ".gitignore"))
	if err != nil || string(ignore) != "*\n" {
		t.Fatalf("gitignore %q %v", ignore, err)
	}
	signedOut := Logout(first)
	if signedOut.Configured {
		t.Fatal("session stayed open")
	}
	if _, err := Load(first); err == nil || err.(*Error).Code != "not_configured" {
		t.Fatalf("load after logout %v", err)
	}
	if still, err := Load(second); err != nil || still.Token != "other-token" {
		t.Fatalf("other project %v %+v", err, still)
	}
}

func TestSearchUsesTheJQLEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/search/jql" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		reply(t, w, `{"issues":[{"key":"AET-2","fields":{"summary":"Refund","status":{"name":"Open"},"issuetype":{"name":"Bug"}}}]}`)
	}))
	defer server.Close()
	client := Client{Account: Account{BaseURL: server.URL, Token: "pat"}, HTTP: server.Client()}
	got, err := client.Search(context.Background(), "project = AET")
	if err != nil || len(got) != 1 || got[0].Key != "AET-2" || got[0].Summary != "Refund" {
		t.Fatalf("%v %+v", err, got)
	}
}

func TestCommentPostsADFAndTransitionUsesTheStatusName(t *testing.T) {
	var commentBody []byte
	var transitionID string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/rest/api/3/issue/AET-12/comment":
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("read comment: %v", err)
				http.Error(w, "bad body", http.StatusBadRequest)
				return
			}
			commentBody = body
			w.WriteHeader(http.StatusCreated)
			reply(t, w, `{"id":"10001"}`)
		case r.Method == http.MethodGet && r.URL.Path == "/rest/api/3/issue/AET-12/transitions":
			reply(t, w, `{"transitions":[{"id":"11","name":"Start","to":{"name":"In Progress"}},{"id":"31","name":"Block","to":{"name":"Block"}}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/rest/api/3/issue/AET-12/transitions":
			var payload transitionPayload
			if !readJSON(t, w, r, &payload) {
				return
			}
			transitionID = payload.Transition.ID
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client := Client{Account: Account{BaseURL: server.URL, Email: "dev@example.com", Token: "secret-token"}, HTTP: server.Client()}
	if err := client.Comment(context.Background(), "aet-12", "Span stopped before writing code.\nWhich currency?"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(commentBody), `"doc"`) || !strings.Contains(string(commentBody), "Which currency?") || strings.Contains(string(commentBody), "secret-token") {
		t.Fatalf("comment %s", commentBody)
	}
	if err := client.Transition(context.Background(), "AET-12", "block"); err != nil {
		t.Fatal(err)
	}
	if transitionID != "31" {
		t.Fatalf("transition %q", transitionID)
	}
}

func TestCommentErrorOmitsTheToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "secret-token rejected", http.StatusBadRequest)
	}))
	defer server.Close()
	client := Client{Account: Account{BaseURL: server.URL, Email: "dev@example.com", Token: "secret-token"}, HTTP: server.Client()}
	err := client.Comment(context.Background(), "AET-12", "hello")
	if err == nil || strings.Contains(err.Error(), "secret-token") || !strings.Contains(err.Error(), "[redacted]") {
		t.Fatalf("%v", err)
	}
}

func TestTransitionErrorOmitsTheToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			reply(t, w, `{"transitions":[{"id":"11","name":"Start","to":{"name":"In Progress"}}]}`)
			return
		}
		http.Error(w, "secret-token rejected", http.StatusBadRequest)
	}))
	defer server.Close()
	client := Client{Account: Account{BaseURL: server.URL, Email: "dev@example.com", Token: "secret-token"}, HTTP: server.Client()}
	err := client.Transition(context.Background(), "AET-12", "Block")
	if err == nil || strings.Contains(err.Error(), "secret-token") {
		t.Fatalf("%v", err)
	}
	coded, ok := err.(*Error)
	if !ok || coded.Code != "invalid" || !strings.Contains(coded.Msg, "In Progress") {
		t.Fatalf("%v", err)
	}
}
