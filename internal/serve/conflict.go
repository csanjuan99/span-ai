package serve

import (
	"errors"
	"net/http"

	"span/internal/engine"
)

// conflictRoutes answers the three decisions a person makes on a merge conflict. Each one
// writes a file beside the cursor, which the runner owns, and wakes the run to act on it.
func conflictRoutes(mux *http.ServeMux, root string, wake Wake) {
	mux.HandleFunc("POST /spans/{id}/conflict/choose", func(w http.ResponseWriter, r *http.Request) {
		decideConflict(wake, root, w, r, func(id string) error {
			choices, err := formChoices(r)
			if err != nil {
				return err
			}
			return engine.ChooseConflict(root, id, choices)
		})
	})
	mux.HandleFunc("POST /spans/{id}/conflict/hint", func(w http.ResponseWriter, r *http.Request) {
		decideConflict(wake, root, w, r, func(id string) error {
			return engine.HintConflict(root, id, r.PostForm.Get("text"))
		})
	})
	mux.HandleFunc("POST /spans/{id}/conflict/abort", func(w http.ResponseWriter, r *http.Request) {
		decideConflict(wake, root, w, r, func(id string) error {
			return engine.AbortConflict(root, id)
		})
	})
}

func decideConflict(wake Wake, root string, w http.ResponseWriter, r *http.Request, decide func(id string) error) {
	id := r.PathValue("id")
	if wake == nil || !safe(id) {
		http.NotFound(w, r)
		return
	}
	snap, err := engine.Load(root, id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		writeErr(w, err)
		return
	}
	if err := decide(id); err != nil {
		if errors.Is(err, engine.ErrNoConflict) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "invalid", "message": err.Error()})
			return
		}
		writeErr(w, err)
		return
	}
	if err := wake(r.Context(), id, Signal{Resume: true}); err != nil {
		writeBusy(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"id": id, "stage": snap.Cursor.Stage})
}

// formChoices pairs each group=<id> with the option=<id> at the same position.
func formChoices(r *http.Request) (map[string]string, error) {
	groups, options := r.PostForm["group"], r.PostForm["option"]
	if len(groups) == 0 || len(groups) != len(options) {
		return nil, errors.New("send one group and one option for every group")
	}
	choices := make(map[string]string, len(groups))
	for i, group := range groups {
		if _, seen := choices[group]; seen {
			return nil, errors.New("group " + group + " is chosen twice")
		}
		choices[group] = options[i]
	}
	return choices, nil
}
