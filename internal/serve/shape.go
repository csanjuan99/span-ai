package serve

import (
	"context"
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

// Backlog is the part of Jira that shaping uses; the interface lives with the code that calls it.
type Backlog = shape.Backlog

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
	c := &shapeConfig{agents: CLIShapeRunner, backlog: jiraBacklog, locks: &keyedLock{held: map[string]*heldLock{}}}
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

// Error is the plain sentence the person reads.
func (e *requestError) Error() string { return e.msg }

// SpanCode is the machine name of the failure, the `error` of the JSON answer.
func (e *requestError) SpanCode() string { return e.code }

// CLIShapeRunner is the default ShapeAgents: it runs the interview on the agent CLI the session names, or the policy
// default, or the first ready one. The interview reads the repository and must not change
// it; the engine checks that after every turn.
func CLIShapeRunner(workspace, agentName, model string) shape.Runner {
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
	sessionRoutes(mux, c)
	boardRoutes(mux, c)
	historyRoutes(mux, c)
	transferRoutes(mux, c)
	versionRoutes(mux, c)
	mux.HandleFunc("POST /shapes/{id}/resume", c.resume)
	mux.HandleFunc("POST /shapes/{id}/agent", c.setAgent)
	mux.HandleFunc("POST /shapes/{id}/archive", c.archive)
	mux.HandleFunc("DELETE /shapes/{id}", c.remove)
	mux.HandleFunc("GET /shapes/{id}/jira/preview", c.preview)
	mux.HandleFunc("POST /shapes/{id}/jira", c.createJira)
}

// WorkspaceDir is workspaceDir for the CLI, which reads the same folders the API does.
func WorkspaceDir(raw string) (string, error) { return workspaceDir(raw) }

// workspaceDir accepts only an absolute path to an existing folder.
func workspaceDir(raw string) (string, error) {
	path := strings.TrimSpace(raw)
	if path == "" {
		return "", &requestError{"invalid", "workspace is missing"}
	}
	// A relative folder is read from the server's folder, as /quickstart reads it, and kept
	// absolute so the session names the same folder whatever the server's folder later is.
	path, err := filepath.Abs(path)
	if err != nil {
		return "", &requestError{"invalid", fmt.Sprintf("workspace %q cannot be resolved", raw)}
	}
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
	Issue     string         `json:"issue"`
	Archived  *bool          `json:"archived"`
	// Skip and Restore settle product-context notes beside Accept; Decision answers a split.
	Skip     []string `json:"skip"`
	Restore  []string `json:"restore"`
	Decision string   `json:"decision"`
}

// readBody decodes the JSON body and checks the workspace.
func readBody(w http.ResponseWriter, r *http.Request) (shapeBody, string, bool) {
	return decodeInWorkspace(w, r, func(b shapeBody) string { return b.Workspace })
}

func (c *shapeConfig) engine(ws string, s *shape.Session) *shape.Engine {
	return &shape.Engine{Run: c.agents(ws, s.Agent, s.Model)}
}

func (c *shapeConfig) start(w http.ResponseWriter, r *http.Request) {
	body, ws, ok := readBody(w, r)
	if !ok {
		return
	}
	s, err := c.startSession(r.Context(), ws, body.Repo, body.Idea, body.Issue, body.Agent, body.Model)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, s.View())
}

// startSession starts an interview from an idea, or from a Jira issue when issue names one.
// The page, the import route and the CLI share it, so they start sessions the same way.
func (c *shapeConfig) startSession(ctx context.Context, ws, repo, idea, issue, agentName, model string) (*shape.Session, error) {
	source := ""
	if strings.TrimSpace(issue) != "" {
		var err error
		if idea, source, err = c.ideaFromIssue(ctx, ws, issue, idea); err != nil {
			return nil, err
		}
	}
	eng := c.engine(ws, &shape.Session{Agent: agentName, Model: model})
	return eng.Start(ctx, shape.StartInput{Workspace: ws, Repo: repo, Idea: idea, Agent: agentName, Model: model, SourceIssue: source})
}

// ideaFromIssue reads the Jira issue of a workspace and builds the idea of a session from it.
func (c *shapeConfig) ideaFromIssue(ctx context.Context, ws, key, extra string) (string, string, error) {
	backlog, err := c.backlog(ws)
	if err != nil {
		return "", "", err
	}
	reader, ok := backlog.(shape.IssueReader)
	if !ok {
		return "", "", &requestError{"invalid", "this Jira connection cannot read issues"}
	}
	return shape.IdeaFromIssue(ctx, reader, key, extra)
}

// list answers the sessions that could be read, filtered and ordered by the query:
// status, q, repo, agent, sort (activity or created) and archived (include or only). A
// session that could not be read does not hide the others: the list is returned and
// X-Span-Warning names what was skipped.
func (c *shapeConfig) list(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	if query.Get("group") == "status" {
		c.board(w, r)
		return
	}
	ws, ok := queryWorkspace(w, r)
	if !ok {
		return
	}
	sessions, err := shape.List(ws)
	if sessions == nil {
		writeErr(w, err)
		return
	}
	kept, ferr := shape.Filter{
		Status: query.Get("status"), Query: query.Get("q"), Repo: query.Get("repo"),
		Agent: query.Get("agent"), Sort: query.Get("sort"), Archived: query.Get("archived"),
	}.Apply(sessions)
	if ferr != nil {
		writeErr(w, ferr)
		return
	}
	if err != nil {
		w.Header().Set("X-Span-Warning", strings.Join(strings.Fields(err.Error()), " "))
	}
	writeJSON(w, http.StatusOK, shape.Views(kept))
}

func (c *shapeConfig) get(w http.ResponseWriter, r *http.Request) {
	ws, ok := queryWorkspace(w, r)
	if !ok {
		return
	}
	s, err := shape.Load(ws, r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s.View())
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
	writeJSON(w, http.StatusOK, updated.View())
}

func (c *shapeConfig) answers(w http.ResponseWriter, r *http.Request) {
	c.mutate(w, r, func(body shapeBody, ws string, s *shape.Session) (*shape.Session, error) {
		return c.engine(ws, s).Answer(r.Context(), ws, s.ID, body.Answers, body.Assume)
	})
}

func (c *shapeConfig) context(w http.ResponseWriter, r *http.Request) {
	c.mutate(w, r, func(body shapeBody, ws string, s *shape.Session) (*shape.Session, error) {
		return c.engine(ws, s).SettleProposals(ws, s.ID, body.Accept, body.Skip, body.Restore)
	})
}

func (c *shapeConfig) seal(w http.ResponseWriter, r *http.Request) {
	c.mutate(w, r, func(_ shapeBody, ws string, s *shape.Session) (*shape.Session, error) {
		return c.engine(ws, s).Seal(ws, s.ID)
	})
}

// resume runs the next turn from the saved map, on the agent and model of the body when it
// names one. A failure of the turn is saved on the session and answered as an error.
func (c *shapeConfig) resume(w http.ResponseWriter, r *http.Request) {
	c.mutate(w, r, func(body shapeBody, ws string, s *shape.Session) (*shape.Session, error) {
		if body.Agent != "" {
			s.Agent, s.Model = body.Agent, body.Model
		}
		return c.engine(ws, s).Resume(r.Context(), ws, s.ID, body.Agent, body.Model)
	})
}

// setAgent changes the agent and model of the session without running a turn.
func (c *shapeConfig) setAgent(w http.ResponseWriter, r *http.Request) {
	c.mutate(w, r, func(body shapeBody, ws string, s *shape.Session) (*shape.Session, error) {
		return c.engine(ws, s).SetAgent(ws, s.ID, body.Agent, body.Model)
	})
}

// archive hides a session from the default list, or shows it again with archived: false.
func (c *shapeConfig) archive(w http.ResponseWriter, r *http.Request) {
	c.mutate(w, r, func(body shapeBody, ws string, s *shape.Session) (*shape.Session, error) {
		return shape.SetArchived(ws, s.ID, body.Archived == nil || *body.Archived)
	})
}

// remove deletes a session that is not sealed.
func (c *shapeConfig) remove(w http.ResponseWriter, r *http.Request) {
	ws, ok := queryWorkspace(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	defer c.locks.lock(sessionKey(ws, id))()
	if err := shape.Delete(ws, id); err != nil {
		writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (c *shapeConfig) preview(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	ws, ok := queryWorkspace(w, r)
	if !ok {
		return
	}
	board := 0
	if raw := query.Get("board"); raw != "" {
		var err error
		if board, err = strconv.Atoi(raw); err != nil {
			writeErr(w, &requestError{"invalid", "board must be a board id"})
			return
		}
	}
	backlog, err := c.backlog(ws)
	if err != nil {
		writeErr(w, err)
		return
	}
	view, err := shape.PreviewIssue(r.Context(), backlog, ws, r.PathValue("id"), query.Get("project"), board)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
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
	saved, err := (&shape.Engine{}).CreateIssue(r.Context(), func() (shape.Backlog, error) { return c.backlog(ws) }, ws, id, body.Project, body.Board, body.Confirm)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, saved.View())
}
