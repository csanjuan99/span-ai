package serve

import (
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"span/internal/transcript"
)

const (
	// turnTextLimit bounds the text one turn carries; Cut says it was shortened.
	turnTextLimit = 4000
	// turnPage is how many turns one answer carries unless the caller asks for another number.
	turnPage = 10
)

// TranscriptStage is a stage that printed something, with how many passes it made.
type TranscriptStage struct {
	Stage string `json:"stage"`
	Runs  int    `json:"runs"`
	Lines int    `json:"lines"`
}

// TranscriptTurn is one block of what the agent printed: consecutive lines up to a blank line,
// with a fenced block kept whole. The CLIs Span runs print plain text, so this is the structure
// that exists; no tool call or role is recorded, and none is shown.
type TranscriptTurn struct {
	N         int    `json:"n"`
	Run       int    `json:"run"`
	Text      string `json:"text"`
	Cut       bool   `json:"cut,omitempty"`
	FirstLine int    `json:"firstLine"`
	Lines     int    `json:"lines"`
}

// TranscriptView is what the agent printed in one stage, turn by turn.
type TranscriptView struct {
	Run    string            `json:"run"`
	Stage  string            `json:"stage"`
	Agent  string            `json:"agent,omitempty"`
	Stages []TranscriptStage `json:"stages"`
	// StartedAt and EndedAt bound the stage's passes; zero when the transcript has no markers.
	StartedAt time.Time `json:"startedAt,omitempty"`
	EndedAt   time.Time `json:"endedAt,omitempty"`
	Total     int       `json:"total"`
	From      int       `json:"from"`
	Lines     int       `json:"lines"`
	// Redacted counts the values Span hid because they look like secrets.
	Redacted int              `json:"redacted"`
	Turns    []TranscriptTurn `json:"turns"`
	// Structured is false: the agent's output has no recorded roles or tool calls.
	Structured bool   `json:"structured"`
	Note       string `json:"note"`
}

// ReadTranscript reads the transcript of a stage. An empty stage means the last one that
// printed. from and limit page the turns; a limit of 0 means the default page.
func ReadTranscript(root, id, stage string, from, limit int) (TranscriptView, error) {
	snap, err := loadSnapshot(root, id)
	if err != nil {
		return TranscriptView{}, &requestError{"not_found", "run " + id + " is not on this desk"}
	}
	segs, err := transcript.Segments(filepath.Join(root, id, "transcript.log"))
	if err != nil {
		return TranscriptView{}, err
	}
	view := TranscriptView{
		Run: id, Stages: []TranscriptStage{}, Turns: []TranscriptTurn{},
		Note: "The agent prints plain text. Turns are the blocks it printed, in order; tool calls are not recorded.",
	}
	order := []string{}
	seen := map[string]*TranscriptStage{}
	for _, seg := range segs {
		if seg.Stage == "" && len(segs) > 1 {
			continue
		}
		entry, ok := seen[seg.Stage]
		if !ok {
			entry = &TranscriptStage{Stage: seg.Stage}
			seen[seg.Stage] = entry
			order = append(order, seg.Stage)
		}
		entry.Runs++
		entry.Lines += len(seg.Lines)
	}
	for _, name := range order {
		view.Stages = append(view.Stages, *seen[name])
	}
	if stage == "" && len(order) > 0 {
		stage = order[len(order)-1]
	}
	if stage != "" && seen[stage] == nil {
		return TranscriptView{}, &requestError{"not_found", "stage " + stage + " printed nothing in this run"}
	}
	view.Stage = stage
	view.Agent = snap.StageAgents[stage]
	if limit <= 0 {
		limit = turnPage
	}
	var turns []TranscriptTurn
	run := 0
	for _, seg := range segs {
		if seg.Stage != stage {
			continue
		}
		run++
		if view.StartedAt.IsZero() || seg.StartedAt.Before(view.StartedAt) {
			view.StartedAt = seg.StartedAt
		}
		if seg.EndedAt.After(view.EndedAt) {
			view.EndedAt = seg.EndedAt
		}
		view.Lines += len(seg.Lines)
		for _, turn := range splitTurns(seg.Lines, seg.FirstLine) {
			turn.Run = run
			view.Redacted += strings.Count(turn.Text, "[redacted]")
			turns = append(turns, turn)
		}
	}
	for i := range turns {
		turns[i].N = i + 1
	}
	view.Total = len(turns)
	if from < 0 {
		from = 0
	}
	if from > len(turns) {
		from = len(turns)
	}
	end := from + limit
	if end > len(turns) {
		end = len(turns)
	}
	view.From = from
	view.Turns = turns[from:end]
	return view, nil
}

// splitTurns cuts lines into blocks at blank lines, keeping a fenced block whole.
func splitTurns(lines []string, first int) []TranscriptTurn {
	var out []TranscriptTurn
	var block []string
	start := first
	fenced := false
	flush := func(next int) {
		if len(block) > 0 {
			text := strings.Join(block, "\n")
			turn := TranscriptTurn{Text: text, FirstLine: start, Lines: len(block)}
			if len(text) > turnTextLimit {
				turn.Text, turn.Cut = text[:turnTextLimit], true
			}
			out = append(out, turn)
		}
		block = nil
		start = next
	}
	for i, line := range lines {
		n := first + i
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			fenced = !fenced
		}
		if strings.TrimSpace(line) == "" && !fenced {
			flush(n + 1)
			continue
		}
		block = append(block, line)
	}
	flush(0)
	sort.SliceStable(out, func(i, j int) bool { return out[i].FirstLine < out[j].FirstLine })
	return out
}

func transcriptRoutes(mux *http.ServeMux, root string) {
	mux.HandleFunc("GET /spans/{id}/transcript", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		view, err := ReadTranscript(root, r.PathValue("id"), query.Get("stage"), atoi(query.Get("from")), atoi(query.Get("limit")))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, view)
	})
}
