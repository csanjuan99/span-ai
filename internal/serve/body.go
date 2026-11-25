package serve

import (
	"encoding/json"
	"net/http"
)

// decodeJSON reads a JSON body of at most limit bytes into T. A body that is not valid JSON
// answers 400 invalid and reports false.
func decodeJSON[T any](w http.ResponseWriter, r *http.Request, limit int64) (T, bool) {
	var body T
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, &requestError{"invalid", "the body is not valid JSON: " + err.Error()})
		return body, false
	}
	return body, true
}

// decodeInWorkspace is decodeJSON for a shaping body that names its workspace: it also
// checks the folder and returns it, answering 400 invalid when it is not one.
func decodeInWorkspace[T any](w http.ResponseWriter, r *http.Request, workspace func(T) string) (T, string, bool) {
	body, ok := decodeJSON[T](w, r, maxShapeBody)
	if !ok {
		return body, "", false
	}
	ws, err := workspaceDir(workspace(body))
	if err != nil {
		writeErr(w, err)
		return body, "", false
	}
	return body, ws, true
}
