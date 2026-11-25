package serve

import (
	"net/http"

	"span/internal/shape"
)

// historyRoutes are the routes of a session's history: every step with who did it, what
// undoing one would change, and the undo. `span shape history` and `undo` call the same
// shape functions.
func historyRoutes(mux *http.ServeMux, c *shapeConfig) {
	mux.HandleFunc("GET /shapes/{id}/history", c.history)
	mux.HandleFunc("GET /shapes/{id}/history/{step}", c.undoPlan)
	mux.HandleFunc("POST /shapes/{id}/history/{step}/undo", c.undo)
}

// history answers the steps of a session, newest first, filtered by actor (everyone, you or
// agents) and kind (answers, assumptions, context or split).
func (c *shapeConfig) history(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	ws, ok := queryWorkspace(w, r)
	if !ok {
		return
	}
	view, err := shape.History(ws, r.PathValue("id"), shape.HistoryFilter{Actor: query.Get("actor"), Kind: query.Get("kind")})
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

// undoPlan answers what undoing a step would change, and changes nothing.
func (c *shapeConfig) undoPlan(w http.ResponseWriter, r *http.Request) {
	ws, ok := queryWorkspace(w, r)
	if !ok {
		return
	}
	plan, err := shape.PlanUndo(ws, r.PathValue("id"), r.PathValue("step"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, plan)
}

// undoBody asks to undo a step; cascade undoes the steps that depend on it too.
type undoBody struct {
	Workspace string `json:"workspace"`
	Cascade   bool   `json:"cascade"`
}

// undo puts a step back. A step with dependents is refused unless cascade is true, and a
// sealed story never changes.
func (c *shapeConfig) undo(w http.ResponseWriter, r *http.Request) {
	body, ws, ok := decodeInWorkspace(w, r, func(b undoBody) string { return b.Workspace })
	if !ok {
		return
	}
	id := r.PathValue("id")
	defer c.locks.lock(sessionKey(ws, id))()
	res, err := (&shape.Engine{}).Undo(ws, id, r.PathValue("step"), body.Cascade)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"step": res.Step, "undone": res.Undone, "session": res.Session.View()})
}
