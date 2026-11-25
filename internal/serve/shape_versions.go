package serve

import (
	"net/http"
	"sort"
	"strings"

	"span/internal/shape"
)

// versionRoutes are the routes that compare: two revisions of one session, and two to five
// sessions of one repo. `span shape versions|diff|restore|compare|merge` call the same
// shape functions.
func versionRoutes(mux *http.ServeMux, c *shapeConfig) {
	mux.HandleFunc("GET /shapes/{id}/revisions", c.revisions)
	mux.HandleFunc("GET /shapes/{id}/diff", c.diff)
	mux.HandleFunc("POST /shapes/{id}/restore", c.restore)
	mux.HandleFunc("POST /shapes/{id}/revisions/{rev}/restore", c.restore)
	mux.HandleFunc("POST /shapes/{id}/revisions/{rev}/fork", c.forkRevision)
	mux.HandleFunc("GET /shapes/compare", c.compare)
	mux.HandleFunc("POST /shapes/compare/dismiss", c.dismiss)
	mux.HandleFunc("POST /shapes/merge", c.merge)
}

// versionBody is what the POST routes of this file read.
type versionBody struct {
	Workspace string            `json:"workspace"`
	Revision  string            `json:"revision"`
	IDs       []string          `json:"ids"`
	Into      string            `json:"into"`
	From      []string          `json:"from"`
	Against   []string          `json:"against"`
	Resolve   map[string]string `json:"resolve"`
}

// readVersionBody decodes a revisions or compare body and checks its workspace.
func readVersionBody(w http.ResponseWriter, r *http.Request) (versionBody, string, bool) {
	return decodeInWorkspace(w, r, func(b versionBody) string { return b.Workspace })
}

func queryWorkspace(w http.ResponseWriter, r *http.Request) (string, bool) {
	ws, err := workspaceDir(r.URL.Query().Get("workspace"))
	if err != nil {
		writeErr(w, err)
		return "", false
	}
	return ws, true
}

// revisions lists the revisions of a session, oldest first.
func (c *shapeConfig) revisions(w http.ResponseWriter, r *http.Request) {
	ws, ok := queryWorkspace(w, r)
	if !ok {
		return
	}
	list, err := shape.Revisions(ws, r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"revisions": list})
}

// diff answers what changed between two revisions, by kind of item, plus the story lines.
func (c *shapeConfig) diff(w http.ResponseWriter, r *http.Request) {
	ws, ok := queryWorkspace(w, r)
	if !ok {
		return
	}
	query := r.URL.Query()
	out, err := shape.Diff(ws, r.PathValue("id"), query.Get("from"), query.Get("to"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// restore adds a revision with the content of an earlier one; nothing is deleted.
func (c *shapeConfig) restore(w http.ResponseWriter, r *http.Request) {
	body, ws, ok := readVersionBody(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	defer c.locks.lock(sessionKey(ws, id))()
	rev := r.PathValue("rev")
	if rev == "" {
		rev = body.Revision
	}
	s, err := shape.Restore(ws, id, rev)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s.View())
}

// forkRevision starts a new session from a revision; the session it comes from stays.
func (c *shapeConfig) forkRevision(w http.ResponseWriter, r *http.Request) {
	_, ws, ok := readVersionBody(w, r)
	if !ok {
		return
	}
	s, err := shape.ForkRevision(ws, r.PathValue("id"), r.PathValue("rev"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, s.View())
}

// compare answers what two to five sessions of one repo share and where they disagree.
func (c *shapeConfig) compare(w http.ResponseWriter, r *http.Request) {
	ws, ok := queryWorkspace(w, r)
	if !ok {
		return
	}
	got, err := shape.Compare(ws, splitIDs(r.URL.Query().Get("ids")))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, got)
}

func splitIDs(raw string) []string {
	var out []string
	for _, id := range strings.Split(raw, ",") {
		if id = strings.TrimSpace(id); id != "" {
			out = append(out, id)
		}
	}
	return out
}

// dismiss records that the sessions stay separate.
func (c *shapeConfig) dismiss(w http.ResponseWriter, r *http.Request) {
	body, ws, ok := readVersionBody(w, r)
	if !ok {
		return
	}
	got, err := shape.Dismiss(ws, body.IDs)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, got)
}

// merge moves the unique items of the from sessions into one. Every session it touches is
// locked, in a fixed order so two merges cannot wait on each other.
func (c *shapeConfig) merge(w http.ResponseWriter, r *http.Request) {
	body, ws, ok := readVersionBody(w, r)
	if !ok {
		return
	}
	keys := []string{sessionKey(ws, body.Into)}
	for _, id := range body.From {
		keys = append(keys, sessionKey(ws, id))
	}
	sort.Strings(keys)
	for _, key := range keys {
		defer c.locks.lock(key)()
	}
	got, err := shape.Merge(ws, shape.MergeInput{Into: body.Into, From: body.From, Against: body.Against, Resolve: body.Resolve})
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"into": got.Into.View(), "moved": got.Moved, "answers": got.Answers, "replaced": got.Replaced, "merged": got.Merged,
	})
}
