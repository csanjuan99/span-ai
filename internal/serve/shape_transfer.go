package serve

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"span/internal/shape"
)

// transferRoutes bring stories in and out: import from a Jira story or a markdown file,
// export of a sealed story, and read-only links to it. `span shape import`, `export` and
// `share` call the same shape functions.
func transferRoutes(mux *http.ServeMux, c *shapeConfig) {
	mux.HandleFunc("POST /shapes/import", c.importStory)
	mux.HandleFunc("GET /shapes/{id}/export", c.export)
	mux.HandleFunc("GET /shapes/{id}/shares", c.shares)
	mux.HandleFunc("POST /shapes/{id}/shares", c.createShare)
	mux.HandleFunc("DELETE /shapes/{id}/shares/{token}", c.revokeShare)
	mux.HandleFunc("GET /shapes/{id}/shared/{token}", c.openShare)
}

// importBody asks to start a session from a Jira story (jira) or a markdown file (markdown).
type importBody struct {
	Workspace string `json:"workspace"`
	Repo      string `json:"repo"`
	Agent     string `json:"agent"`
	Model     string `json:"model"`
	Jira      string `json:"jira"`
	Idea      string `json:"idea"`
	Markdown  string `json:"markdown"`
	Name      string `json:"name"`
	// DryRun answers what Span understood and starts nothing.
	DryRun bool `json:"dryRun"`
	// Run is false to create the session without running the first interview turn.
	Run *bool `json:"run"`
}

// importStory starts a session from a Jira story or a markdown file. A dry run answers what
// was read: the story's fields, or the rules and examples found in the file.
func (c *shapeConfig) importStory(w http.ResponseWriter, r *http.Request) {
	body, ws, ok := decodeInWorkspace(w, r, func(b importBody) string { return b.Workspace })
	if !ok {
		return
	}
	hasJira, hasFile := strings.TrimSpace(body.Jira) != "", strings.TrimSpace(body.Markdown) != ""
	switch {
	case hasJira == hasFile:
		writeErr(w, &requestError{"invalid", "give either jira (a story key) or markdown (the file text), not both and not neither"})
	case hasJira:
		c.importJira(w, r, ws, body)
	default:
		c.importMarkdown(w, r, ws, body)
	}
}

func (c *shapeConfig) importJira(w http.ResponseWriter, r *http.Request, ws string, body importBody) {
	if body.DryRun {
		backlog, err := c.backlog(ws)
		if err != nil {
			writeErr(w, err)
			return
		}
		reader, ok := backlog.(shape.IssueReader)
		if !ok {
			writeErr(w, &requestError{"invalid", "this Jira connection cannot read issues"})
			return
		}
		read, err := shape.ReadStory(r.Context(), reader, body.Jira)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, read)
		return
	}
	s, err := c.startSession(r.Context(), ws, body.Repo, body.Idea, body.Jira, body.Agent, body.Model)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, s.View())
}

func (c *shapeConfig) importMarkdown(w http.ResponseWriter, r *http.Request, ws string, body importBody) {
	if body.DryRun {
		preview, err := shape.ParseMarkdown(body.Name, body.Markdown)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, preview)
		return
	}
	s, preview, err := shape.ImportMarkdown(ws, shape.ImportInput{
		Repo: body.Repo, Agent: body.Agent, Model: body.Model, Name: body.Name, Markdown: body.Markdown,
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	if body.Run == nil || *body.Run {
		defer c.locks.lock(sessionKey(ws, s.ID))()
		if s, err = c.engine(ws, s).Resume(r.Context(), ws, s.ID, "", ""); err != nil {
			writeErr(w, err)
			return
		}
	}
	writeJSON(w, http.StatusCreated, map[string]any{"session": s.View(), "read": preview})
}

// export answers a sealed story as a file: format md or json, and sources, assumptions and
// digest set to 1 to include them. A story that is not sealed is refused with "seal first".
func (c *shapeConfig) export(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	ws, ok := queryWorkspace(w, r)
	if !ok {
		return
	}
	file, err := shape.Export(ws, r.PathValue("id"), shape.ExportOptions{
		Format: query.Get("format"), Sources: query.Get("sources") == "1",
		Assumptions: query.Get("assumptions") == "1", Digest: query.Get("digest") == "1",
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	w.Header().Set("Content-Type", file.MIME+"; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+file.Name+`"`)
	w.Header().Set("Content-Length", strconv.Itoa(len(file.Body)))
	writeText(w, string(file.Body))
}

func (c *shapeConfig) shares(w http.ResponseWriter, r *http.Request) {
	ws, ok := queryWorkspace(w, r)
	if !ok {
		return
	}
	list, err := shape.Shares(ws, r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// shareBody asks for a read-only link that lasts days days (7 when zero).
type shareBody struct {
	Workspace string `json:"workspace"`
	Days      int    `json:"days"`
}

// createShare makes a read-only link to a sealed story. The link path is
// /shaping/<id>/shared/<token>, to open with the same access as the rest of the server.
func (c *shapeConfig) createShare(w http.ResponseWriter, r *http.Request) {
	body, ws, ok := decodeInWorkspace(w, r, func(b shareBody) string { return b.Workspace })
	if !ok {
		return
	}
	id := r.PathValue("id")
	link, err := shape.CreateShare(ws, id, body.Days, time.Now())
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"link": link, "path": "/shaping/" + id + "/shared/" + link.Token})
}

func (c *shapeConfig) revokeShare(w http.ResponseWriter, r *http.Request) {
	ws, ok := queryWorkspace(w, r)
	if !ok {
		return
	}
	link, err := shape.RevokeShare(ws, r.PathValue("id"), r.PathValue("token"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, link)
}

// openShare answers what a link shows, and counts the opening.
func (c *shapeConfig) openShare(w http.ResponseWriter, r *http.Request) {
	ws, ok := queryWorkspace(w, r)
	if !ok {
		return
	}
	story, err := shape.OpenShare(ws, r.PathValue("id"), r.PathValue("token"), time.Now())
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, story)
}
