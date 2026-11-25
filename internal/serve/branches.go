package serve

import (
	"net/http"

	"span/internal/start"
)

func handleBranches(w http.ResponseWriter, r *http.Request) {
	list, err := start.ListBranches(r.Context(), r.URL.Query().Get("path"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}
