package serve

import (
	"encoding/json"
	"net/http"
	"strings"

	"span/internal/implementation"
)

// maxCheckBody bounds the body of a check request: a short list of decision ids.
const maxCheckBody = 4 << 10

// registerImplementation adds the routes the Implementation tab reads and writes: the files,
// their diffs and viewed marks, the commits, the reasoning and the audit chain. Each has a
// `span implementation` command that calls the same functions.
func registerImplementation(mux *http.ServeMux, root string) {
	mux.HandleFunc("GET /spans/{id}/files", func(w http.ResponseWriter, r *http.Request) {
		files, err := implementation.ChangedFiles(r.Context(), root, r.PathValue("id"))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"files": files})
	})
	mux.HandleFunc("GET /spans/{id}/files/{path...}", func(w http.ResponseWriter, r *http.Request) {
		file, ok := strings.CutSuffix(r.PathValue("path"), "/diff")
		if !ok {
			http.NotFound(w, r)
			return
		}
		query := r.URL.Query()
		view, err := implementation.RunFileDiff(r.Context(), root, r.PathValue("id"), file, query.Get("commit"), query.Get("force") == "true")
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, view)
	})
	mux.HandleFunc("POST /spans/{id}/files/{path...}", func(w http.ResponseWriter, r *http.Request) {
		file, ok := strings.CutSuffix(r.PathValue("path"), "/viewed")
		if !ok {
			http.NotFound(w, r)
			return
		}
		// Marking is the default; ?viewed=false takes the mark off.
		viewed := r.URL.Query().Get("viewed") != "false"
		if err := implementation.SetViewed(r.Context(), root, r.PathValue("id"), file, viewed); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"path": file, "viewed": viewed})
	})
	mux.HandleFunc("GET /spans/{id}/commits", func(w http.ResponseWriter, r *http.Request) {
		commits, err := implementation.RunCommits(r.Context(), root, r.PathValue("id"))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"commits": commits})
	})
	mux.HandleFunc("GET /spans/{id}/commits/{sha}/diff", func(w http.ResponseWriter, r *http.Request) {
		diff, err := implementation.RunCommitDiff(r.Context(), root, r.PathValue("id"), r.PathValue("sha"))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, diff)
	})
	mux.HandleFunc("GET /spans/{id}/reasoning", func(w http.ResponseWriter, r *http.Request) {
		view, err := implementation.RunReasoning(root, r.PathValue("id"))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, view)
	})
	mux.HandleFunc("POST /spans/{id}/decisions/check", func(w http.ResponseWriter, r *http.Request) {
		ids, err := checkIDs(w, r)
		if err != nil {
			writeErr(w, err)
			return
		}
		id := r.PathValue("id")
		if err := implementation.CheckDecisions(root, id, ids); err != nil {
			writeErr(w, err)
			return
		}
		view, err := implementation.RunReasoning(root, id)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, view)
	})
	mux.HandleFunc("GET /spans/{id}/audit", func(w http.ResponseWriter, r *http.Request) {
		view, err := implementation.RunAudit(root, r.PathValue("id"))
		if err != nil {
			writeErr(w, err)
			return
		}
		if r.URL.Query().Get("format") == "json" {
			w.Header().Set("Content-Disposition", `attachment; filename="`+r.PathValue("id")+`-audit.json"`)
		}
		writeJSON(w, http.StatusOK, view)
	})
	mux.HandleFunc("GET /spans/{id}/audit/verify", func(w http.ResponseWriter, r *http.Request) {
		view, err := implementation.RunAudit(root, r.PathValue("id"))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"intact": view.Intact, "brokenAt": view.BrokenAt, "entries": len(view.Entries), "head": view.Head})
	})
}

// checkIDs reads the decisions to check from a JSON body {"ids": [...]} or from a form with
// one or more ids fields, each of which may hold a comma separated list.
func checkIDs(w http.ResponseWriter, r *http.Request) ([]string, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxCheckBody)
	var raw []string
	if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		var body struct {
			IDs []string `json:"ids"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			return nil, &requestError{"invalid", "the body is not valid JSON: " + err.Error()}
		}
		raw = body.IDs
	} else {
		if err := r.ParseForm(); err != nil {
			return nil, &requestError{"invalid", "the form could not be read: " + err.Error()}
		}
		raw = r.PostForm["ids"]
	}
	var ids []string
	for _, field := range raw {
		for _, id := range strings.Split(field, ",") {
			if id = strings.TrimSpace(id); id != "" {
				ids = append(ids, id)
			}
		}
	}
	return ids, nil
}
