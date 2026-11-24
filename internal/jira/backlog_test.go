package jira

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// transitionPayload is the body Jira receives to apply a transition.
type transitionPayload struct {
	Transition struct {
		ID string `json:"id"`
	} `json:"transition"`
}

// reply writes a JSON body, a test error when the write fails.
func reply(t *testing.T, w http.ResponseWriter, body string) {
	t.Helper()
	if _, err := w.Write([]byte(body)); err != nil {
		t.Errorf("write response: %v", err)
	}
}

// readJSON decodes the request body, answering 400 when it is not valid.
func readJSON(t *testing.T, w http.ResponseWriter, r *http.Request, into any) bool {
	t.Helper()
	raw, err := io.ReadAll(r.Body)
	if err == nil {
		err = json.Unmarshal(raw, into)
	}
	if err != nil {
		t.Errorf("bad request body: %v", err)
		http.Error(w, "bad body", http.StatusBadRequest)
		return false
	}
	return true
}

// fakeJira is a Jira Cloud stand-in that records the writes it receives.
type fakeJira struct {
	boards       string // JSON "values" array of the board search
	config       string // JSON of the board configuration
	createStatus string // status id the new issue starts in
	transitions  string // JSON of the issue transitions
	mu           sync.Mutex
	createBody   map[string]any
	creates      int
	transitionID string
	statusID     string // current status id of the created issue
}

var fakeStatuses = map[string]string{"1": "Backlog", "2": "To Do", "3": "In Progress", "4": "Selected"}

func (f *fakeJira) handler(t *testing.T) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/rest/agile/1.0/board":
			if r.URL.Query().Get("projectKeyOrId") != "AET" || r.URL.Query().Get("type") != "kanban" {
				http.Error(w, "bad query", http.StatusBadRequest)
				return
			}
			reply(t, w, `{"isLast":true,"values":`+f.boards+`}`)
		case r.Method == http.MethodGet && r.URL.Path == "/rest/agile/1.0/board/7":
			reply(t, w, `{"id":7,"name":"AET board","type":"kanban"}`)
		case r.Method == http.MethodGet && r.URL.Path == "/rest/agile/1.0/board/7/configuration":
			reply(t, w, f.config)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/rest/api/3/status/"):
			id := strings.TrimPrefix(r.URL.Path, "/rest/api/3/status/")
			name, ok := fakeStatuses[id]
			if !ok {
				http.NotFound(w, r)
				return
			}
			reply(t, w, `{"id":"`+id+`","name":"`+name+`"}`)
		case r.Method == http.MethodPost && r.URL.Path == "/rest/api/3/issue":
			if !readJSON(t, w, r, &f.createBody) {
				return
			}
			f.creates++
			f.statusID = f.createStatus
			w.WriteHeader(http.StatusCreated)
			reply(t, w, `{"id":"10","key":"AET-99"}`)
		case r.Method == http.MethodGet && r.URL.Path == "/rest/api/3/issue/AET-99":
			reply(t, w, `{"key":"AET-99","fields":{"status":{"id":"`+f.statusID+`","name":"`+fakeStatuses[f.statusID]+`"}}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/rest/api/3/issue/AET-99/transitions":
			reply(t, w, f.transitions)
		case r.Method == http.MethodPost && r.URL.Path == "/rest/api/3/issue/AET-99/transitions":
			var payload transitionPayload
			if !readJSON(t, w, r, &payload) {
				return
			}
			f.transitionID = payload.Transition.ID
			f.statusID = "1"
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	})
}

const backlogConfig = `{"columnConfig":{"columns":[
	{"name":"Backlog","statuses":[{"id":"1"}]},
	{"name":"To Do","statuses":[{"id":"2"}]}]}}`

const oneBoard = `[{"id":7,"name":"AET board","type":"kanban"}]`

func newFake(t *testing.T, f *fakeJira) Client {
	t.Helper()
	server := httptest.NewServer(f.handler(t))
	t.Cleanup(server.Close)
	return Client{Account: Account{BaseURL: server.URL, Email: "dev@example.com", Token: "secret-token"}, HTTP: server.Client()}
}

func asCode(t *testing.T, err error) *Error {
	t.Helper()
	coded, ok := err.(*Error)
	if !ok {
		t.Fatalf("not a coded error: %v", err)
	}
	return coded
}

func TestPreviewUsesTheSingleKanbanBoardAndNamesTheBacklogStatuses(t *testing.T) {
	client := newFake(t, &fakeJira{boards: oneBoard, config: backlogConfig})
	got, err := client.Preview(context.Background(), "AET", 0)
	if err != nil {
		t.Fatal(err)
	}
	if got.Board.ID != 7 || got.Board.Name != "AET board" || len(got.BacklogStatuses) != 1 || got.BacklogStatuses[0] != "Backlog" {
		t.Fatalf("%+v", got)
	}
}

func TestSeveralKanbanBoardsWithoutAnIDAreAmbiguous(t *testing.T) {
	client := newFake(t, &fakeJira{boards: `[{"id":7,"name":"One"},{"id":8,"name":"Two"}]`, config: backlogConfig})
	_, err := client.Preview(context.Background(), "AET", 0)
	coded := asCode(t, err)
	if coded.Code != "jira_board_ambiguous" || !strings.Contains(coded.Msg, "7 One") || !strings.Contains(coded.Msg, "8 Two") {
		t.Fatalf("%v", err)
	}
}

func TestNoKanbanBoardIsNotFound(t *testing.T) {
	client := newFake(t, &fakeJira{boards: `[]`, config: backlogConfig})
	_, err := client.Preview(context.Background(), "AET", 0)
	if coded := asCode(t, err); coded.Code != "jira_board_not_found" {
		t.Fatalf("%v", err)
	}
}

func TestAnExplicitBoardIDIsVerifiedAndUsed(t *testing.T) {
	// Two boards would be ambiguous; the explicit id skips discovery.
	client := newFake(t, &fakeJira{boards: `[{"id":7,"name":"One"},{"id":8,"name":"Two"}]`, config: backlogConfig})
	got, err := client.Preview(context.Background(), "AET", 7)
	if err != nil || got.Board.ID != 7 || got.Board.Name != "AET board" {
		t.Fatalf("%v %+v", err, got)
	}
	_, err = client.Preview(context.Background(), "AET", 99)
	if coded := asCode(t, err); coded.Code != "jira_board_not_found" {
		t.Fatalf("%v", err)
	}
}

func TestAnEmptyBacklogColumnIsDisabledAndCreatesNothing(t *testing.T) {
	fake := &fakeJira{boards: oneBoard, config: `{"columnConfig":{"columns":[{"name":"Backlog","statuses":[]},{"name":"To Do","statuses":[{"id":"2"}]}]}}`}
	client := newFake(t, fake)
	_, err := client.CreateInBacklog(context.Background(), Draft{Project: "AET", Summary: "S", Description: "D"})
	if coded := asCode(t, err); coded.Code != "jira_backlog_disabled" {
		t.Fatalf("%v", err)
	}
	if fake.creates != 0 {
		t.Fatalf("created %d issues", fake.creates)
	}
}

func TestCreatedAlreadyInTheBacklogNeedsNoTransition(t *testing.T) {
	fake := &fakeJira{boards: oneBoard, config: backlogConfig, createStatus: "1"}
	client := newFake(t, fake)
	got, err := client.CreateInBacklog(context.Background(), Draft{Project: "AET", Summary: "Show channels", Description: "Line one\nLine two"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Key != "AET-99" || got.Status != "Backlog" || !strings.HasSuffix(got.URL, "/browse/AET-99") {
		t.Fatalf("%+v", got)
	}
	if fake.transitionID != "" {
		t.Fatalf("unexpected transition %q", fake.transitionID)
	}
}

func TestCreateSendsStoryLabelADFAndNoAssignee(t *testing.T) {
	fake := &fakeJira{boards: oneBoard, config: backlogConfig, createStatus: "1"}
	client := newFake(t, fake)
	if _, err := client.CreateInBacklog(context.Background(), Draft{Project: "AET", Summary: "Show channels", Description: "Line one\nLine two"}); err != nil {
		t.Fatal(err)
	}
	fields, ok := fake.createBody["fields"].(map[string]any)
	if !ok {
		t.Fatalf("no fields in %v", fake.createBody)
	}
	project, okProject := fields["project"].(map[string]any)
	issuetype, okType := fields["issuetype"].(map[string]any)
	description, okDescription := fields["description"].(map[string]any)
	labels, okLabels := fields["labels"].([]any)
	if !okProject || !okType || !okDescription || !okLabels {
		t.Fatalf("fields have the wrong shape: %v", fields)
	}
	if project["key"] != "AET" || issuetype["name"] != "Story" || fields["summary"] != "Show channels" {
		t.Fatalf("%v", fields)
	}
	if description["type"] != "doc" || len(labels) != 1 || labels[0] != "span" {
		t.Fatalf("%v", fields)
	}
	if _, has := fields["assignee"]; has {
		t.Fatalf("assignee was set: %v", fields)
	}
}

func TestCreatedElsewhereIsTransitionedIntoTheBacklog(t *testing.T) {
	fake := &fakeJira{
		boards: oneBoard, config: backlogConfig, createStatus: "2",
		transitions: `{"transitions":[{"id":"21","name":"Start","to":{"id":"3","name":"In Progress"}},{"id":"41","name":"Back","to":{"id":"1","name":"Backlog"}}]}`,
	}
	client := newFake(t, fake)
	got, err := client.CreateInBacklog(context.Background(), Draft{Project: "AET", Summary: "S", Description: "D"})
	if err != nil {
		t.Fatal(err)
	}
	if fake.transitionID != "41" || got.Status != "Backlog" || got.Key != "AET-99" {
		t.Fatalf("%q %+v", fake.transitionID, got)
	}
}

func TestNoTransitionIntoTheBacklogReportsTheCreatedKey(t *testing.T) {
	fake := &fakeJira{
		boards: oneBoard, config: backlogConfig, createStatus: "2",
		transitions: `{"transitions":[{"id":"21","name":"Start","to":{"id":"3","name":"In Progress"}}]}`,
	}
	client := newFake(t, fake)
	_, err := client.CreateInBacklog(context.Background(), Draft{Project: "AET", Summary: "S", Description: "D"})
	coded := asCode(t, err)
	if coded.Code != "jira_not_in_backlog" || !strings.Contains(coded.Msg, "AET-99") || strings.Contains(err.Error(), "secret-token") {
		t.Fatalf("%v", err)
	}
	if fake.transitionID != "" || fake.creates != 1 {
		t.Fatalf("transition %q creates %d", fake.transitionID, fake.creates)
	}
}

func TestTransitionByNameStillPicksTheTargetStatusOverTheTransitionName(t *testing.T) {
	var sent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			reply(t, w, `{"transitions":[{"id":"5","name":"Done","to":{"id":"9","name":"Review"}},{"id":"6","name":"Finish","to":{"id":"10","name":"Done"}}]}`)
			return
		}
		var payload transitionPayload
		if !readJSON(t, w, r, &payload) {
			return
		}
		sent = payload.Transition.ID
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	client := Client{Account: Account{BaseURL: server.URL, Token: "pat"}, HTTP: server.Client()}
	if err := client.Transition(context.Background(), "AET-1", "done"); err != nil || sent != "6" {
		t.Fatalf("%v %q", err, sent)
	}
}

func TestLogoutKeepsTheAccountWhenTheFileCannotBeRemoved(t *testing.T) {
	account := Account{BaseURL: "https://example.atlassian.net", Token: "secret-token"}
	signedOut := t.TempDir()
	if err := Save(signedOut, account); err != nil {
		t.Fatal(err)
	}
	if got := Logout(signedOut); got.Configured {
		t.Fatalf("normal logout stayed configured: %+v", got)
	}

	locked := t.TempDir()
	if err := Save(locked, account); err != nil {
		t.Fatal(err)
	}
	path, err := Path(locked)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Dir(path)
	// A read-only directory makes os.Remove fail while the file stays readable.
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(dir, 0o700); err != nil {
			t.Errorf("restore permissions: %v", err)
		}
	})
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	if got := Logout(locked); !got.Configured {
		t.Fatalf("a failed remove must keep the account configured: %+v", got)
	}
}
