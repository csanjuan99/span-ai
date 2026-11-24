package serve

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"span/internal/agent"
	"span/internal/jira"
	"span/internal/policy"
	"span/internal/shape"
)

// maxShapeBody caps a shaping request body; answers are prose, never files.
const maxShapeBody = 1 << 20

// ShapeAgents builds the runner of the interview agent for a workspace, the agent the
// session asked for and its model. Empty agent and model mean the workspace policy, then
// the first ready agent, as the run stages choose.
type ShapeAgents func(workspace, agentName, model string) shape.Runner

// Backlog is the part of Jira that shaping uses: look at the backlog of a Kanban board and
// create a story in it.
type Backlog interface {
	Preview(ctx context.Context, project string, board int) (jira.Preview, error)
	CreateInBacklog(ctx context.Context, draft jira.Draft) (jira.Created, error)
}

// BacklogFor returns the Jira backlog of a workspace; the account is read per workspace.
type BacklogFor func(workspace string) (Backlog, error)

// Option changes how New wires the shaping endpoints. Without options they use the real
// agent CLIs and the Jira account saved for the workspace.
type Option func(*shapeConfig)

// WithShapeAgents replaces how the interview agent is run.
func WithShapeAgents(agents ShapeAgents) Option {
	return func(c *shapeConfig) { c.agents = agents }
}

// WithBacklog replaces how the Jira backlog of a workspace is reached.
func WithBacklog(backlog BacklogFor) Option {
	return func(c *shapeConfig) { c.backlog = backlog }
}

type shapeConfig struct {
	agents  ShapeAgents
	backlog BacklogFor
	locks   *keyedLock
}

func newShapeConfig(opts []Option) *shapeConfig {
	c := &shapeConfig{agents: cliShapeRunner, backlog: jiraBacklog, locks: &keyedLock{held: map[string]*heldLock{}}}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// keyedLock serializes work per key and forgets a key once nobody holds or waits for it.
type keyedLock struct {
	mu   sync.Mutex
	held map[string]*heldLock
}

type heldLock struct {
	mu   sync.Mutex
	refs int
}

// lock blocks until the key is free and returns the function that releases it.
func (k *keyedLock) lock(key string) func() {
	k.mu.Lock()
	h := k.held[key]
	if h == nil {
		h = &heldLock{}
		k.held[key] = h
	}
	h.refs++
	k.mu.Unlock()
	h.mu.Lock()
	return func() {
		h.mu.Unlock()
		k.mu.Lock()
		h.refs--
		if h.refs == 0 {
			delete(k.held, key)
		}
		k.mu.Unlock()
	}
}

// sessionKey names one session for the lock.
func sessionKey(ws, id string) string { return ws + "\x00" + id }

// projectKey normalizes the Jira project of a request; a refusal answers 400 invalid.
func projectKey(w http.ResponseWriter, raw string) (string, bool) {
	key, err := jira.ProjectKey(raw)
	if err != nil {
		writeErr(w, err)
		return "", false
	}
	return key, true
}

// requestError is a failure of the request itself, answered with its code.
type requestError struct{ code, msg string }

func (e *requestError) Error() string    { return e.msg }
func (e *requestError) SpanCode() string { return e.code }

// cliShapeRunner runs the interview on the agent CLI the session names, or the policy
// default, or the first ready one. The interview reads the repository and must not change
// it; the engine checks that after every turn.
func cliShapeRunner(workspace, agentName, model string) shape.Runner {
	return func(ctx context.Context, dir, prompt string) (string, error) {
		want, model := agentName, model
		file, err := policy.Load(filepath.Join(workspace, policy.Name))
		if err == nil {
			if agent.Auto(want) {
				want = file.Agent
			}
			if model == "" {
				model = file.ModelFor(want)
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("span.yaml: %w", err)
		}
		sel, err := agent.Select(want, agent.Home())
		if err != nil {
			return "", err
		}
		return agent.RunModel(ctx, sel.Build, model, dir, prompt, nil)
	}
}

// jiraBacklog reads the Jira account saved for the workspace.
func jiraBacklog(workspace string) (Backlog, error) {
	account, err := jira.Load(workspace)
	if err != nil {
		return nil, err
	}
	return jira.Client{Account: account}, nil
}

func shapeRoutes(mux *http.ServeMux, c *shapeConfig) {
	mux.HandleFunc("POST /shapes", c.start)
	mux.HandleFunc("GET /shapes", c.list)
	mux.HandleFunc("GET /shapes/{id}", c.get)
	mux.HandleFunc("POST /shapes/{id}/answers", c.answers)
	mux.HandleFunc("POST /shapes/{id}/context", c.context)
	mux.HandleFunc("POST /shapes/{id}/seal", c.seal)
	mux.HandleFunc("GET /shapes/{id}/jira/preview", c.preview)
	mux.HandleFunc("POST /shapes/{id}/jira", c.createJira)
}

// workspaceDir accepts only an absolute path to an existing folder.
func workspaceDir(raw string) (string, error) {
	path := strings.TrimSpace(raw)
	if path == "" || !filepath.IsAbs(path) {
		return "", &requestError{"invalid", "workspace must be an absolute path"}
	}
	path = filepath.Clean(path)
	if info, err := os.Stat(path); err != nil || !info.IsDir() {
		return "", &requestError{"invalid", fmt.Sprintf("workspace %q is not a folder", path)}
	}
	return path, nil
}

// shapeBody is every field a shaping POST can carry; each route reads its own.
type shapeBody struct {
	Workspace string         `json:"workspace"`
	Repo      string         `json:"repo"`
	Idea      string         `json:"idea"`
	Agent     string         `json:"agent"`
	Model     string         `json:"model"`
	Answers   []shape.Answer `json:"answers"`
	Assume    []string       `json:"assume"`
	Accept    []string       `json:"accept"`
	Project   string         `json:"project"`
	Board     int            `json:"board"`
	Confirm   bool           `json:"confirm"`
}

// readBody decodes the JSON body and checks the workspace.
func readBody(w http.ResponseWriter, r *http.Request) (shapeBody, string, bool) {
	var body shapeBody
	r.Body = http.MaxBytesReader(w, r.Body, maxShapeBody)
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, &requestError{"invalid", "the body is not valid JSON: " + err.Error()})
		return body, "", false
	}
	ws, err := workspaceDir(body.Workspace)
	if err != nil {
		writeErr(w, err)
		return body, "", false
	}
	return body, ws, true
}

func (c *shapeConfig) engine(ws string, s *shape.Session) *shape.Engine {
	return &shape.Engine{Run: c.agents(ws, s.Agent, s.Model)}
}

func (c *shapeConfig) start(w http.ResponseWriter, r *http.Request) {
	body, ws, ok := readBody(w, r)
	if !ok {
		return
	}
	eng := c.engine(ws, &shape.Session{Agent: body.Agent, Model: body.Model})
	s, err := eng.Start(r.Context(), shape.StartInput{Workspace: ws, Repo: body.Repo, Idea: body.Idea, Agent: body.Agent, Model: body.Model})
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, s)
}

// list answers the sessions that could be read. A session that could not be read does not
// hide the others: the list is returned and X-Span-Warning names what was skipped.
func (c *shapeConfig) list(w http.ResponseWriter, r *http.Request) {
	ws, err := workspaceDir(r.URL.Query().Get("workspace"))
	if err != nil {
		writeErr(w, err)
		return
	}
	sessions, err := shape.List(ws)
	if sessions == nil {
		writeErr(w, err)
		return
	}
	if err != nil {
		w.Header().Set("X-Span-Warning", strings.Join(strings.Fields(err.Error()), " "))
	}
	writeJSON(w, http.StatusOK, sessions)
}

func (c *shapeConfig) get(w http.ResponseWriter, r *http.Request) {
	ws, err := workspaceDir(r.URL.Query().Get("workspace"))
	if err != nil {
		writeErr(w, err)
		return
	}
	s, err := shape.Load(ws, r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s)
}

// mutate runs one change of a session while no other request changes the same session.
func (c *shapeConfig) mutate(w http.ResponseWriter, r *http.Request, change func(body shapeBody, ws string, s *shape.Session) (*shape.Session, error)) {
	body, ws, ok := readBody(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	defer c.locks.lock(sessionKey(ws, id))()
	current, err := shape.Load(ws, id)
	if err != nil {
		writeErr(w, err)
		return
	}
	updated, err := change(body, ws, current)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (c *shapeConfig) answers(w http.ResponseWriter, r *http.Request) {
	c.mutate(w, r, func(body shapeBody, ws string, s *shape.Session) (*shape.Session, error) {
		return c.engine(ws, s).Answer(r.Context(), ws, s.ID, body.Answers, body.Assume)
	})
}

func (c *shapeConfig) context(w http.ResponseWriter, r *http.Request) {
	c.mutate(w, r, func(body shapeBody, ws string, s *shape.Session) (*shape.Session, error) {
		return c.engine(ws, s).AcceptProposals(ws, s.ID, body.Accept)
	})
}

func (c *shapeConfig) seal(w http.ResponseWriter, r *http.Request) {
	c.mutate(w, r, func(_ shapeBody, ws string, s *shape.Session) (*shape.Session, error) {
		return c.engine(ws, s).Seal(ws, s.ID)
	})
}

// storySummary is the text of the first "# " heading of the story, or its first line.
func storySummary(story string) string {
	first := ""
	for _, line := range strings.Split(story, "\n") {
		line = strings.TrimSpace(line)
		if title, ok := strings.CutPrefix(line, "# "); ok && strings.TrimSpace(title) != "" {
			return strings.TrimSpace(title)
		}
		if first == "" {
			first = line
		}
	}
	return first
}

// sealedSession loads a session and refuses one that is not sealed.
func sealedSession(ws, id string) (*shape.Session, error) {
	s, err := shape.Load(ws, id)
	if err != nil {
		return nil, err
	}
	if s.Status != shape.StatusSealed {
		return nil, &requestError{"shape_not_sealed", "session " + id + " is not sealed"}
	}
	return s, nil
}

func (c *shapeConfig) preview(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	ws, err := workspaceDir(query.Get("workspace"))
	if err != nil {
		writeErr(w, err)
		return
	}
	board := 0
	if raw := query.Get("board"); raw != "" {
		if board, err = strconv.Atoi(raw); err != nil || board < 0 {
			writeErr(w, &requestError{"invalid", "board must be a board id"})
			return
		}
	}
	project, ok := projectKey(w, query.Get("project"))
	if !ok {
		return
	}
	s, err := sealedSession(ws, r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	backlog, err := c.backlog(ws)
	if err != nil {
		writeErr(w, err)
		return
	}
	view, err := backlog.Preview(r.Context(), project, board)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"project": project, "board": view.Board, "backlogStatuses": view.BacklogStatuses,
		"summary": storySummary(s.Story), "description": s.Story,
	})
}

// createJira creates the story of a sealed session in the backlog, once. The key is stored
// whenever Jira reports one, even when the issue did not reach the backlog.
func (c *shapeConfig) createJira(w http.ResponseWriter, r *http.Request) {
	body, ws, ok := readBody(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	defer c.locks.lock(sessionKey(ws, id))()
	s, err := sealedSession(ws, id)
	if err != nil {
		writeErr(w, err)
		return
	}
	if s.Jira != nil && s.Jira.Key != "" {
		writeJSON(w, http.StatusOK, s)
		return
	}
	if !body.Confirm {
		writeErr(w, &requestError{"confirm_required", "creating the story in Jira needs confirm: true"})
		return
	}
	project, ok := projectKey(w, body.Project)
	if !ok {
		return
	}
	backlog, err := c.backlog(ws)
	if err != nil {
		writeErr(w, err)
		return
	}
	created, createErr := backlog.CreateInBacklog(r.Context(), jira.Draft{
		Project: project, Board: body.Board, Summary: storySummary(s.Story), Description: s.Story,
	})
	if created.Key == "" {
		if createErr == nil {
			createErr = &requestError{"invalid", "jira returned no issue key"}
		}
		writeErr(w, createErr)
		return
	}
	saved, saveErr := (&shape.Engine{}).SetJira(ws, id, shape.JiraRef{Key: created.Key, URL: created.URL, Status: created.Status})
	if err := errors.Join(createErr, saveErr); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, saved)
}
