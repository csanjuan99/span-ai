package serve

import (
	"net/http"

	"span/internal/shape"
)

// sessionRoutes are the routes of one session beyond answering and sealing: the example
// map's sources, the product-context preview, the split decision and the fork of a sealed
// story. `span shape` calls the same shape functions.
func sessionRoutes(mux *http.ServeMux, c *shapeConfig) {
	mux.HandleFunc("GET /shapes/{id}/source", c.source)
	mux.HandleFunc("GET /shapes/{id}/context/preview", c.contextPreview)
	mux.HandleFunc("POST /shapes/{id}/split", c.split)
	mux.HandleFunc("POST /shapes/{id}/fork", c.fork)
}

// source answers the code around a "path:line" reference of the session's repository.
func (c *shapeConfig) source(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	ws, ok := queryWorkspace(w, r)
	if !ok {
		return
	}
	snippet, err := shape.SourceSnippet(ws, r.PathValue("id"), query.Get("ref"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, snippet)
}

// contextPreview answers the lines accepting the listed proposals would add to the product
// context file. It writes nothing.
func (c *shapeConfig) contextPreview(w http.ResponseWriter, r *http.Request) {
	ws, ok := queryWorkspace(w, r)
	if !ok {
		return
	}
	diff, err := shape.ContextPreview(ws, r.PathValue("id"), splitIDs(r.URL.Query().Get("accept")))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, diff)
}

// split settles a proposed split: decision "split" creates one linked session per story,
// "keep" keeps the story whole and runs the turn again.
func (c *shapeConfig) split(w http.ResponseWriter, r *http.Request) {
	c.mutate(w, r, func(body shapeBody, ws string, s *shape.Session) (*shape.Session, error) {
		switch body.Decision {
		case "split":
			return c.engine(ws, s).AcceptSplit(ws, s.ID)
		case "keep":
			return c.engine(ws, s).KeepWhole(r.Context(), ws, s.ID)
		}
		return nil, &requestError{"invalid", `decision must be "split" or "keep"`}
	})
}

// fork starts a new session from a sealed one and answers the new session.
func (c *shapeConfig) fork(w http.ResponseWriter, r *http.Request) {
	c.mutate(w, r, func(_ shapeBody, ws string, s *shape.Session) (*shape.Session, error) {
		return c.engine(ws, s).Fork(ws, s.ID)
	})
}

// JiraBacklog is the Jira backlog of a workspace, read from the account saved for it. The
// command line uses it so it reaches Jira the way the server does.
func JiraBacklog(workspace string) (shape.Backlog, error) { return jiraBacklog(workspace) }
