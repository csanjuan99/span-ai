package serve

import (
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"span/internal/policy"
)

// editableKeys are the form fields that may change span.yaml, next to "project". Anything
// else is refused rather than ignored, so a typo never reads as a saved setting.
var editableKeys = map[string]bool{
	"project": true, "base": true,
	"open": true, "merge": true, "promote": true, "plan": true, "conflict": true,
	"focused": true, "fast": true, "acceptance": true,
	"qa": true, "repair": true,
}

// configEdit is the reply to a preview or a save: the lines that change, and, for a save,
// the settings as they read afterwards.
type configEdit struct {
	Lines  []policy.DiffLine `json:"lines"`
	Config *configPick       `json:"config,omitempty"`
}

// changesOf reads the fields of an edit form. A field that is present sets its value, a field
// that is absent is left alone. A list is one field with one check per line; blank lines are
// dropped, so an empty field is an empty list.
func changesOf(form url.Values) (policy.Changes, error) {
	var keys []string
	for key := range form {
		if !editableKeys[key] {
			keys = append(keys, key)
		}
	}
	if len(keys) > 0 {
		sort.Strings(keys)
		return policy.Changes{}, fmt.Errorf("%s cannot be changed from here", strings.Join(keys, ", "))
	}
	var changes policy.Changes
	text := func(key string) *string {
		if _, ok := form[key]; !ok {
			return nil
		}
		value := form.Get(key)
		return &value
	}
	items := func(key string) *[]string {
		if _, ok := form[key]; !ok {
			return nil
		}
		out := []string{}
		for _, line := range strings.Split(form.Get(key), "\n") {
			if clean := strings.TrimSpace(line); clean != "" {
				out = append(out, clean)
			}
		}
		return &out
	}
	changes.Base, changes.Open, changes.Merge = text("base"), text("open"), text("merge")
	changes.Promote, changes.Plan, changes.Conflict = text("promote"), text("plan"), text("conflict")
	changes.Focused, changes.Fast, changes.Acceptance = items("focused"), items("fast"), items("acceptance")
	changes.QA = text("qa")
	if raw := text("repair"); raw != nil {
		attempts, err := strconv.Atoi(strings.TrimSpace(*raw))
		if err != nil {
			return policy.Changes{}, fmt.Errorf("repair %q is not a number", *raw)
		}
		changes.Repair = &attempts
	}
	return changes, nil
}

// previewConfig says what a save would change in span.yaml and writes nothing.
func previewConfig(w http.ResponseWriter, r *http.Request) {
	_, path, ok := agentPolicyPath(w, r)
	if !ok {
		return
	}
	changes, err := changesOf(r.PostForm)
	if err != nil {
		writeErr(w, err)
		return
	}
	preview, err := policy.PreviewEdit(path, changes)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, configEdit{Lines: preview.Lines})
}

// saveConfig writes the changes into span.yaml and returns what changed with the settings as
// they now read. The file is validated before it is replaced, so a refused change leaves it
// as it was.
func saveConfig(w http.ResponseWriter, r *http.Request) {
	project, path, ok := agentPolicyPath(w, r)
	if !ok {
		return
	}
	changes, err := changesOf(r.PostForm)
	if err != nil {
		writeErr(w, err)
		return
	}
	preview, err := policy.Edit(path, changes)
	if err != nil {
		writeErr(w, err)
		return
	}
	view := configView(project)
	writeJSON(w, http.StatusOK, configEdit{Lines: preview.Lines, Config: &view})
}
