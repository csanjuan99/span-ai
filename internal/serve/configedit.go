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
	"qa": true, "repair": true, "doneStatus": true,
}

// agentFieldPrefix starts the form field that sets the CLI of one stage, such as agents.build.
const agentFieldPrefix = "agents."

// editable says whether a form field may change span.yaml.
func editable(key string) bool {
	if stage, ok := strings.CutPrefix(key, agentFieldPrefix); ok {
		for _, name := range policy.AgentStageNames {
			if name == stage {
				return true
			}
		}
		return false
	}
	return editableKeys[key]
}

// effect is one thing a save would change for a person, read from the lines it writes.
type effect struct {
	Label string `json:"label"`
	Text  string `json:"text"`
}

// configEdit is the reply to a preview or a save: the lines that change, and, for a save,
// the settings as they read afterwards.
type configEdit struct {
	Lines   []policy.DiffLine `json:"lines"`
	Effects []effect          `json:"effects"`
	Config  *configPick       `json:"config,omitempty"`
}

// changesOf reads the fields of an edit form. A field that is present sets its value, a field
// that is absent is left alone. A list is one field with one check per line; blank lines are
// dropped, so an empty field is an empty list.
func changesOf(form url.Values) (policy.Changes, error) {
	var keys []string
	for key := range form {
		if !editable(key) {
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
	changes.DoneStatus = text("doneStatus")
	for _, stage := range policy.AgentStageNames {
		if value := text(agentFieldPrefix + stage); value != nil {
			if changes.Agents == nil {
				changes.Agents = map[string]string{}
			}
			changes.Agents[stage] = strings.TrimSpace(*value)
		}
	}
	if raw := text("repair"); raw != nil {
		attempts, err := strconv.Atoi(strings.TrimSpace(*raw))
		if err != nil {
			return policy.Changes{}, fmt.Errorf("repair %q is not a number", *raw)
		}
		changes.Repair = &attempts
	}
	return changes, nil
}

// editConfig previews or saves the form fields into the span.yaml of a project. A save
// validates the file before replacing it, so a refused change leaves it as it was.
func editConfig(project string, form url.Values, save bool) (configEdit, error) {
	path, err := policyFile(project)
	if err != nil {
		return configEdit{}, err
	}
	changes, err := changesOf(form)
	if err != nil {
		return configEdit{}, err
	}
	if !save {
		preview, err := policy.PreviewEdit(path, changes)
		if err != nil {
			return configEdit{}, err
		}
		return configEdit{Lines: preview.Lines, Effects: effectsOf(preview.Lines)}, nil
	}
	preview, err := policy.Edit(path, changes)
	if err != nil {
		return configEdit{}, err
	}
	view := configView(project)
	return configEdit{Lines: preview.Lines, Effects: effectsOf(preview.Lines), Config: &view}, nil
}

// settingsRoute answers POST /config/preview (save false) and POST /config (save true).
func settingsRoute(save bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			writeErr(w, err)
			return
		}
		edit, err := editConfig(r.PostForm.Get("project"), r.PostForm, save)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, edit)
	}
}

// effectsOf says what a save does to runs, from the lines it writes. It states only what the
// engine does: a run reads span.yaml again at every stage, and a CLI a run chose for a stage
// itself still wins over span.yaml.
func effectsOf(lines []policy.DiffLine) []effect {
	out := []effect{}
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line.Key, "agents."):
			stage := strings.TrimPrefix(line.Key, "agents.")
			who := line.After
			if who == "" {
				who = "the project agent"
			}
			out = append(out, effect{"Applies to", "The next " + stage + " stage of every run, running ones included, uses " + who + ". A stage a run was set to by hand keeps its choice."})
		case strings.HasPrefix(line.Key, "stages."):
			stage := strings.TrimPrefix(line.Key, "stages.")
			out = append(out, effect{"Next stage", stageEffect(stage, line.After)})
		case strings.HasPrefix(line.Key, "prove.") || line.Key == "qa.command":
			out = append(out, effect{"Next check", "Runs reach the new commands the next time they prove."})
		case line.Key == "repair.maxAttempts":
			out = append(out, effect{"Repairs", "A failing check may send a run back to build " + line.After + " times before it stops."})
		case line.Key == "jira.doneStatus":
			if line.After == "" {
				out = append(out, effect{"Jira", "A finished run leaves its story where it is."})
			} else {
				out = append(out, effect{"Jira", "A run that finishes moves its story to " + line.After + "."})
			}
		case line.Key == "base":
			out = append(out, effect{"New runs", "Start from " + baseText(line.After) + ". Runs already started keep their base."})
		}
	}
	return out
}

func stageEffect(stage, mode string) string {
	switch mode {
	case "hold":
		return "A run that reaches " + stage + " will stop and ask you."
	case "conditional":
		return "A run that reaches " + stage + " will ask you when the story is weighed deep."
	}
	return "A run that reaches " + stage + " will go on without asking."
}

func baseText(base string) string {
	if base == "" {
		return "the default base (dev, or master for a hotfix)"
	}
	return base
}
