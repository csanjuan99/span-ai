package serve

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"span/internal/adapt"
	"span/internal/engine"
	"span/internal/statement"
)

// runLogLimit bounds what one request reads of a command log. A longer log keeps its end,
// which is where a check says why it failed.
const runLogLimit = 512 * 1024

// runLogView is the full output of one command, as the run page's "Open full log" shows it.
type runLogView struct {
	Path      string `json:"path"`
	Text      string `json:"text"`
	Lines     int    `json:"lines"`
	Truncated bool   `json:"truncated"`
	// Bytes is the size of the whole log. TotalLines counts every line of it, FirstLine is the
	// number of the first line Text holds (1 unless the start was cut), and Partial says that
	// line is itself cut. The log viewer says "N of M lines" from these.
	Bytes      int64 `json:"bytes,omitempty"`
	TotalLines int   `json:"totalLines,omitempty"`
	FirstLine  int   `json:"firstLine,omitempty"`
	Partial    bool  `json:"partial,omitempty"`
}

// errRunLog is every reason a log is not served; the route answers all of them with 404 so a
// caller cannot tell a missing file from one that is not theirs to read.
var errRunLog = errors.New("the log is not available")

// readRunLog reads a command log of a run. The path is the one a statement stores
// ("iter/1/logs/prove-1.log"): relative, under the run's iterations, a .log file, and still
// inside the run after links are resolved.
func readRunLog(root, id, rel string) (runLogView, error) {
	if !safe(id) || !validRunLogPath(rel) {
		return runLogView{}, errRunLog
	}
	if _, err := engine.Load(root, id); err != nil {
		return runLogView{}, errRunLog
	}
	dir, err := filepath.EvalSymlinks(filepath.Join(root, id))
	if err != nil {
		return runLogView{}, errRunLog
	}
	full, err := filepath.EvalSymlinks(filepath.Join(dir, filepath.FromSlash(rel)))
	if err != nil || !strings.HasPrefix(full, dir+string(filepath.Separator)) {
		return runLogView{}, errRunLog
	}
	file, err := os.Open(full)
	if err != nil {
		return runLogView{}, errRunLog
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return runLogView{}, errRunLog
	}
	truncated := info.Size() > runLogLimit
	cutLines := 0
	partial := false
	if truncated {
		cutAt := info.Size() - runLogLimit
		cutLines, partial = linesBefore(file, cutAt)
		if _, err := file.Seek(cutAt, io.SeekStart); err != nil {
			return runLogView{}, errRunLog
		}
	}
	raw, err := io.ReadAll(io.LimitReader(file, runLogLimit))
	if err != nil {
		return runLogView{}, errRunLog
	}
	text := string(raw)
	for truncated && text != "" && !utf8.RuneStart(text[0]) {
		text = text[1:]
	}
	lines := statement.CountLines(text)
	return runLogView{
		Path: rel, Text: text, Lines: lines, Truncated: truncated,
		Bytes: info.Size(), TotalLines: cutLines + lines, FirstLine: cutLines + 1, Partial: partial,
	}, nil
}

// linesBefore counts the line breaks in the first n bytes of a file, so the line numbers of a
// log that was cut at its start still match the whole log. partial is true when the cut falls
// inside a line. A read error counts what was read.
func linesBefore(file *os.File, n int64) (lines int, partial bool) {
	buf := make([]byte, 64*1024)
	var last byte = '\n'
	for read := int64(0); read < n; {
		want := int64(len(buf))
		if n-read < want {
			want = n - read
		}
		got, err := file.ReadAt(buf[:want], read)
		lines += bytes.Count(buf[:got], []byte{'\n'})
		if got > 0 {
			last = buf[got-1]
		}
		read += int64(got)
		if err != nil || got == 0 {
			break
		}
	}
	return lines, last != '\n'
}

// validRunLogPath accepts only a clean, relative path to a .log file under iter/.
func validRunLogPath(rel string) bool {
	if rel == "" || strings.ContainsAny(rel, `\`) || path.IsAbs(rel) || path.Clean(rel) != rel {
		return false
	}
	return strings.HasPrefix(rel, "iter/") && strings.HasSuffix(rel, ".log")
}

func serveRunLog(root string, w http.ResponseWriter, r *http.Request) {
	view, err := readRunLog(root, r.PathValue("id"), r.URL.Query().Get("path"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

// The run page's readers and its instruction writer, exported so `span` answers with the
// code the routes use. Each keeps the check its route makes before it reads the folder.

// RunLog is the full output of one command, as GET /spans/{id}/log returns it.
type RunLog = runLogView

// ReadRunLog reads the log at a statement's stored path, within the limits of the route.
func ReadRunLog(root, id, rel string) (RunLog, error) { return readRunLog(root, id, rel) }

// Feature is the report of the feature document, as GET /spans/{id}/feature returns it.
type Feature = featureView

// ReadFeature reads the feature document of a run.
func ReadFeature(root, id string) (Feature, error) { return readFeature(root, id) }

// Acceptance is the acceptance cases and the gate, as GET /spans/{id}/acceptance returns them.
type Acceptance = acceptanceBody

// ReadAcceptance reads the acceptance cases of a run with the gate verdict.
func ReadAcceptance(root, id string) (Acceptance, error) { return readAcceptance(root, id) }

// Events is the audit journal of a run with its chain check, as GET /spans/{id}/events returns it.
type Events = eventsView

// ReadEvents reads the journal of a run.
func ReadEvents(root, id string) (Events, error) {
	if !safe(id) {
		return Events{}, os.ErrNotExist
	}
	return readEvents(root, id)
}

// ReadImplementation reads what the run changed: the files, the commits and the diff.
func ReadImplementation(ctx context.Context, root, id string) (adapt.Diff, error) {
	if !safe(id) {
		return adapt.Diff{}, os.ErrNotExist
	}
	return readImplementation(ctx, root, id)
}

// SaveInstruction stores the note a stage reads and keeps every version in the decision log.
func SaveInstruction(root, id, stage string, raw []byte) error {
	if !safe(id) || !safe(stage) {
		return errors.New("invalid span or stage")
	}
	return saveInstruction(root, id, stage, raw)
}

// LogQuery narrows a gate's log: a range of lines, a search, or the first error.
type LogQuery struct {
	From, To   int
	Q          string
	FirstError bool
}

// LogLine is one line of a log with its number in the whole log and what it is.
type LogLine struct {
	N    int    `json:"n"`
	Text string `json:"text"`
	// Kind is "err" for a line that names a failure and "ctx" for any other line.
	Kind string `json:"kind"`
}

// GateLog is the log of one gate with what the viewer needs: the line numbers of the whole log,
// where the first error is, and the lines that match a search, read over the whole log.
type GateLog struct {
	Run        string    `json:"run"`
	Gate       string    `json:"gate"`
	Command    string    `json:"command"`
	ExitCode   int       `json:"exitCode"`
	DurationMS int64     `json:"durationMs,omitempty"`
	EndedAt    time.Time `json:"endedAt,omitempty"`
	Bytes      int64     `json:"bytes,omitempty"`
	TotalLines int       `json:"totalLines"`
	FirstLine  int       `json:"firstLine"`
	Kept       int       `json:"kept"`
	Truncated  bool      `json:"truncated"`
	// Windowed is true when Lines is a range of what is kept, because From/To asked for one or
	// the log is too long to send whole.
	Windowed   bool      `json:"windowed,omitempty"`
	FirstError int       `json:"firstError,omitempty"`
	Q          string    `json:"q,omitempty"`
	Matches    []int     `json:"matches"`
	Lines      []LogLine `json:"lines"`
}

// gateLogLines bounds how many lines one answer carries.
const gateLogLines = 20000

var errorLine = regexp.MustCompile(`(^|\s)(--- FAIL|FAIL\b|panic:|fatal error)|_test\.go:\d+:|^\s*(×|✗|not ok)`)

// ReadGateLog reads the log of a gate by its name in the QA view ("fast", "acceptance-2").
func ReadGateLog(root, id, gateName string, q LogQuery) (GateLog, error) {
	snap, err := loadSnapshot(root, id)
	if err != nil {
		return GateLog{}, &requestError{"not_found", "run " + id + " is not on this desk"}
	}
	var gate QAGate
	found := false
	for _, g := range qaGates(snap.Statements) {
		if g.Name == gateName || (g.Band == gateName && !found) {
			gate, found = g, true
		}
	}
	if !found || !gate.Ran {
		return GateLog{}, &requestError{"not_found", "gate " + gateName + " has no log in this run"}
	}
	view, err := gateLogView(root, id, snap, gate)
	if err != nil {
		return GateLog{}, &requestError{"not_found", "the log of " + gateName + " was not kept"}
	}
	return buildGateLog(id, gate, view, q), nil
}

// gateLogView is the kept log of a gate, or the tail its statement carries when the file is gone.
func gateLogView(root, id string, snap engine.Snapshot, gate QAGate) (runLogView, error) {
	if gate.LogPath != "" {
		if view, err := readRunLog(root, id, gate.LogPath); err == nil {
			return view, nil
		}
	}
	text, ok := gateText(snap, gate)
	if !ok || text == "" {
		return runLogView{}, errRunLog
	}
	kept := statement.CountLines(text)
	total := kept
	if gate.Lines > kept {
		total = gate.Lines
	}
	return runLogView{Path: gate.LogPath, Text: text, Lines: kept, Truncated: total > kept,
		Bytes: int64(len(text)), TotalLines: total, FirstLine: total - kept + 1}, nil
}

func buildGateLog(id string, gate QAGate, view runLogView, q LogQuery) GateLog {
	out := GateLog{
		Run: id, Gate: gate.Name, Command: gate.Command, ExitCode: gate.ExitCode, DurationMS: gate.DurationMS,
		EndedAt: gate.EndedAt, Bytes: view.Bytes, TotalLines: view.TotalLines, FirstLine: view.FirstLine,
		Truncated: view.Truncated, Q: q.Q, Matches: []int{}, Lines: []LogLine{},
	}
	if out.FirstLine < 1 {
		out.FirstLine = 1
	}
	needle := strings.ToLower(q.Q)
	text := strings.TrimSuffix(view.Text, "\n")
	var all []LogLine
	if view.Text != "" {
		for i, line := range strings.Split(text, "\n") {
			n := out.FirstLine + i
			kind := "ctx"
			if errorLine.MatchString(line) {
				kind = "err"
				if out.FirstError == 0 {
					out.FirstError = n
				}
			}
			if needle != "" && len(out.Matches) < gateLogLines && strings.Contains(strings.ToLower(line), needle) {
				out.Matches = append(out.Matches, n)
			}
			all = append(all, LogLine{N: n, Text: line, Kind: kind})
		}
	}
	out.Kept = len(all)
	from, to := q.From, q.To
	if q.FirstError && out.FirstError > 0 {
		from, to = out.FirstError-3, out.FirstError+3
	}
	for _, line := range all {
		if (from > 0 && line.N < from) || (to > 0 && line.N > to) {
			continue
		}
		if len(out.Lines) >= gateLogLines {
			break
		}
		out.Lines = append(out.Lines, line)
	}
	out.Windowed = len(out.Lines) < len(all)
	return out
}
