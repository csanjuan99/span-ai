package serve

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"span/internal/engine"
	"span/internal/fsutil"
	"span/internal/journal"
	"span/internal/qa"
	"span/internal/statement"
)

// The verdicts of a gate and of the whole QA reading.
const (
	qaPass    = "pass"
	qaFail    = "fail"
	qaHeld    = "held"
	qaNotRun  = "notrun"
	qaOK      = "ok"
	qaNoCases = "no_cases"
	qaNever   = "never_ran"
	qaUnseal  = "unsealed"
)

// QA exit codes: what a terminal reading of the gate answers, so a script can branch on it.
const (
	QAExitOK       = 0
	QAExitFailed   = 10
	QAExitNoProof  = 11
	qaRerunTimeout = 5 * time.Minute
	qaRerunLimit   = 50
)

// QAGate is one check of the proof, in the order it ran.
type QAGate struct {
	Name        string                `json:"name"`
	Band        string                `json:"band"`
	Command     string                `json:"command"`
	Verdict     string                `json:"verdict"`
	Ran         bool                  `json:"ran"`
	ExitCode    int                   `json:"exitCode"`
	DurationMS  int64                 `json:"durationMs,omitempty"`
	Lines       int                   `json:"lines,omitempty"`
	LogPath     string                `json:"logPath,omitempty"`
	Counts      *statement.TestCounts `json:"counts,omitempty"`
	Preexisting bool                  `json:"preexisting,omitempty"`
	EndedAt     time.Time             `json:"endedAt,omitempty"`
	// Detail says why a gate did not run or did not pass, in a short plain line.
	Detail string `json:"detail,omitempty"`
}

// QACounts is what the checks said about the tests they ran.
type QACounts struct {
	Passed  int     `json:"passed"`
	Failed  int     `json:"failed"`
	Skipped int     `json:"skipped"`
	Seconds float64 `json:"seconds"`
}

// QAEvidence says who observed the result. Span ran every command it shows, so the kind is
// always "span"; the note says what was read.
type QAEvidence struct {
	Kind string `json:"kind"`
	Note string `json:"note"`
}

// QAView is the QA reading of a run: the verdict, the command that produced it and the gates.
type QAView struct {
	Run      string      `json:"run"`
	State    string      `json:"state"`
	Verdict  string      `json:"verdict"`
	Headline string      `json:"headline"`
	Command  string      `json:"command,omitempty"`
	ExitCode int         `json:"exitCode"`
	Exit     int         `json:"exit"`
	Evidence *QAEvidence `json:"evidence,omitempty"`
	Counts   *QACounts   `json:"counts,omitempty"`
	Gates    []QAGate    `json:"gates"`
	// Cases is how many test cases the gates' output names; 0 when the output names none.
	Cases int `json:"cases"`
	// Reason, Kept and Next fill the honest states: what happened, what stays, what to do.
	Reason string `json:"reason,omitempty"`
	Kept   string `json:"kept,omitempty"`
	Next   string `json:"next,omitempty"`
	// Accepted is the risk a person accepted for this run, when one did.
	Accepted *QAAcceptance `json:"accepted,omitempty"`
}

// QAAcceptance is a person's recorded decision to go on with a gate that did not pass.
type QAAcceptance struct {
	Reason string    `json:"reason"`
	Time   time.Time `json:"time"`
	Gate   string    `json:"gate,omitempty"`
}

var noTestFiles = regexp.MustCompile(`(?m)\bno test files\b`)

// latestStatement is the statement of the highest iteration of a stage.
func latestStatement(list []statement.Statement, stage string) (statement.Statement, bool) {
	var best statement.Statement
	found := false
	for _, item := range list {
		if item.Stage == stage && (!found || item.Iteration >= best.Iteration) {
			best, found = item, true
		}
	}
	return best, found
}

// gateVerdict reads one command: a failure is a fail, a command that did not run is held, and a
// pass that skipped cases is held too, because skipped cases prove nothing.
func gateVerdict(c statement.Command) string {
	switch {
	case c.Error != "":
		return qaHeld
	case c.ExitCode != 0 && !c.Preexisting:
		return qaFail
	case c.Tests != nil && c.Tests.Skipped > 0 && c.Tests.Failed == 0:
		return qaHeld
	case c.ExitCode == 0 && c.Tests != nil && c.Tests.Passed+c.Tests.Failed+c.Tests.Skipped == 0:
		return qaHeld
	default:
		return qaPass
	}
}

// qaGates lists the commands of the latest prove and qa statements with a stable name for each:
// the band, numbered when a band holds several.
func qaGates(list []statement.Statement) []QAGate {
	var gates []QAGate
	used := map[string]int{}
	for _, stage := range []string{"prove", "qa"} {
		st, ok := latestStatement(list, stage)
		if !ok {
			continue
		}
		for _, c := range st.Commands {
			band := c.Band
			if band == "" {
				band = stage
			}
			used[band]++
			name := band
			if used[band] > 1 {
				name = fmt.Sprintf("%s-%d", band, used[band])
			}
			gate := QAGate{
				Name: name, Band: band, Command: c.Command, Ran: c.Error == "", ExitCode: c.ExitCode,
				DurationMS: c.DurationMS, Lines: c.Lines, LogPath: c.LogPath, Counts: c.Tests,
				Preexisting: c.Preexisting, EndedAt: st.Time, Verdict: gateVerdict(c),
			}
			switch {
			case c.Error != "":
				gate.Detail = c.Error
			case c.Preexisting:
				gate.Detail = "failed the same way before this change"
			}
			gates = append(gates, gate)
		}
	}
	// A band that renames itself on a repeat keeps its first command under the plain name.
	for i := range gates {
		if used[gates[i].Band] > 1 && gates[i].Name == gates[i].Band {
			gates[i].Name = gates[i].Band + "-1"
		}
	}
	return gates
}

// decisive is the gate the verdict speaks about: the first one that failed, else the last one
// that counted tests, else the last one that ran.
func decisive(gates []QAGate) (QAGate, bool) {
	for _, g := range gates {
		if g.Verdict == qaFail {
			return g, true
		}
	}
	for i := len(gates) - 1; i >= 0; i-- {
		if gates[i].Counts != nil {
			return gates[i], true
		}
	}
	if len(gates) == 0 {
		return QAGate{}, false
	}
	return gates[len(gates)-1], true
}

// ReadQA reads the QA view of a run.
func ReadQA(root, id string) (QAView, error) {
	snap, err := loadSnapshot(root, id)
	if err != nil {
		return QAView{}, err
	}
	gates := qaGates(snap.Statements)
	view := QAView{Run: id, Gates: gates, Verdict: qaNotRun}
	if accepted, ok := readQAAccepted(root, id); ok {
		view.Accepted = &accepted
	}
	if len(gates) == 0 {
		return neverRan(view, snap), nil
	}
	g, _ := decisive(gates)
	view.Command, view.ExitCode, view.Verdict = g.Command, g.ExitCode, g.Verdict
	view.Evidence = &QAEvidence{Kind: "span", Note: evidenceNote(g)}
	view.Counts = totalCounts(gates)
	view.Cases = len(casesOf(root, id, snap, gates, false))
	view.Headline = headline(g, view.Counts)
	if g.Verdict == qaPass || g.Verdict == qaHeld {
		completeGaps(&view, g, snap)
	}
	if view.State == "" {
		view.State = qaOK
	}
	notRun(&view, snap)
	view.Exit = exitFor(view.Verdict)
	return view, nil
}

// loadSnapshot reads a run once the id is known to name a folder.
func loadSnapshot(root, id string) (engine.Snapshot, error) {
	if !safe(id) {
		return engine.Snapshot{}, os.ErrNotExist
	}
	return engine.Load(root, id)
}

func exitFor(verdict string) int {
	switch verdict {
	case qaPass:
		return QAExitOK
	case qaFail:
		return QAExitFailed
	default:
		return QAExitNoProof
	}
}

func evidenceNote(g QAGate) string {
	note := fmt.Sprintf("Span ran the command and read its exit code, %d.", g.ExitCode)
	if g.Lines > 0 {
		note += fmt.Sprintf(" Output saved, %d lines.", g.Lines)
	}
	return note
}

func totalCounts(gates []QAGate) *QACounts {
	var total QACounts
	seen := false
	for _, g := range gates {
		if g.Counts == nil {
			continue
		}
		seen = true
		total.Passed += g.Counts.Passed
		total.Failed += g.Counts.Failed
		total.Skipped += g.Counts.Skipped
		total.Seconds += float64(g.DurationMS) / 1000
	}
	if !seen {
		return nil
	}
	return &total
}

func headline(g QAGate, counts *QACounts) string {
	name := strings.ToUpper(g.Name[:1]) + g.Name[1:]
	switch g.Verdict {
	case qaFail:
		if counts != nil && counts.Failed > 0 {
			return fmt.Sprintf("%s failed: %d of %d cases", name, counts.Failed, counts.Passed+counts.Failed+counts.Skipped)
		}
		return fmt.Sprintf("%s failed with exit %d", name, g.ExitCode)
	case qaHeld:
		if counts != nil && counts.Skipped > 0 {
			return fmt.Sprintf("Exit %d, but %d cases were skipped. Span does not call this a pass.", g.ExitCode, counts.Skipped)
		}
		return fmt.Sprintf("Exit %d, but the command did not show proof. Span does not call this a pass.", g.ExitCode)
	}
	if counts != nil {
		return fmt.Sprintf("%s passed: %d of %d cases", name, counts.Passed, counts.Passed+counts.Failed+counts.Skipped)
	}
	return name + " passed"
}

// completeGaps turns a passing or held reading into the honest states of the design: the
// command found no test, or no sealed build exists to tie the result to.
func completeGaps(view *QAView, g QAGate, snap engine.Snapshot) {
	if noCases(g, snap) {
		view.State, view.Verdict = qaNoCases, qaHeld
		view.Headline = "The " + g.Name + " command ran, but it found 0 cases"
		view.Reason = fmt.Sprintf("%s printed \"no test files\" or counted 0 tests and exited %d. 0 passed, 0 failed, 0 skipped. Span does not call that a pass.", g.Command, g.ExitCode)
		view.Kept = "The command, its exit code and its output. The code changes."
		view.Next = "Ask the agent to add tests. Or, if this change needs none, accept it and say why."
		return
	}
	if !sealedBuild(snap.Statements) {
		view.State, view.Verdict = qaUnseal, qaFail
		view.Headline = "Tests passed, but Span cannot say what code they ran against"
		view.Reason = fmt.Sprintf("%s exited %d, but build never sealed, so there is no commit to tie the result to. A pass without a build is not shown.", g.Command, g.ExitCode)
		view.Kept = "The test output and the branch as they are."
		view.Next = "Seal build again, then run prove on that build. Or rewind to build."
		return
	}
	view.State = qaOK
}

// noCases reports a gate that exited 0 and proved nothing: it counted zero tests, or its output
// says no test file exists.
func noCases(g QAGate, snap engine.Snapshot) bool {
	if g.ExitCode != 0 || !g.Ran {
		return false
	}
	if g.Counts != nil {
		return g.Counts.Passed+g.Counts.Failed+g.Counts.Skipped == 0
	}
	if text, ok := gateText(snap, g); ok {
		return noTestFiles.MatchString(text) && !strings.Contains(text, "\nok ")
	}
	return false
}

// sealedBuild reports whether the build stage ever advanced.
func sealedBuild(list []statement.Statement) bool {
	for _, item := range list {
		if item.Stage == "build" && item.Verdict == statement.Advance {
			return true
		}
	}
	return false
}

// notRun names the gates that wait behind a failure, so the table can say "not run".
func notRun(view *QAView, snap engine.Snapshot) {
	if view.Verdict != qaFail {
		return
	}
	configured := configuredAcceptance(snap)
	for _, g := range view.Gates {
		if g.Band == "acceptance" || g.Band == "qa" {
			return
		}
	}
	if configured == "" {
		return
	}
	last := view.Gates[len(view.Gates)-1].Name
	view.Gates = append(view.Gates, QAGate{
		Name: "acceptance", Band: "acceptance", Command: configured, Verdict: qaNotRun, Detail: "waits for " + last,
	})
}

func neverRan(view QAView, snap engine.Snapshot) QAView {
	view.State, view.Verdict, view.Exit = qaNever, qaNotRun, QAExitNoProof
	view.Headline = "Prove has not run, so there is no verdict"
	stage := snap.Cursor.Stage
	if build, ok := latestStatement(snap.Statements, "build"); ok && build.Verdict == statement.Advance {
		view.Reason = fmt.Sprintf("build sealed at %s. The run stopped before prove started. No test command was recorded for this run.", build.Time.Local().Format("15:04"))
		view.Kept = "Build, sealed, with its commits."
	} else {
		view.Reason = fmt.Sprintf("The run is at %s. No test command was recorded for this run.", stage)
		view.Kept = "Everything the run has done so far."
	}
	view.Next = "Resume the run. Prove starts first, and its result shows here."
	return view
}

// configuredAcceptance is the acceptance command of the workspace policy, when there is one.
func configuredAcceptance(snap engine.Snapshot) string {
	if snap.Gate != nil && snap.Gate.Command != "" {
		return snap.Gate.Command
	}
	return ""
}

// gateText is what a gate printed: the whole log when it was kept, else the tail the
// statement carries.
func gateText(snap engine.Snapshot, g QAGate) (string, bool) {
	for _, stage := range []string{"prove", "qa"} {
		st, ok := latestStatement(snap.Statements, stage)
		if !ok {
			continue
		}
		for _, c := range st.Commands {
			if c.Command == g.Command && c.LogPath == g.LogPath && c.Band == g.Band {
				return c.Output, true
			}
		}
	}
	return "", false
}

// QACase is a test case of the proof with the gate that ran it and what happened to it.
type QACase struct {
	qa.TestCase
	Gate    string          `json:"gate"`
	History []QACaseHistory `json:"history,omitempty"`
}

// QACaseHistory is one result of a case: the first run of the gate and each rerun after it.
type QACaseHistory struct {
	Time    time.Time `json:"time"`
	Kind    string    `json:"kind"`
	Result  string    `json:"result"`
	Message string    `json:"message,omitempty"`
}

// QACases is the case list the page and the terminal read.
type QACases struct {
	Run    string   `json:"run"`
	Cases  []QACase `json:"cases"`
	Total  int      `json:"total"`
	Passed int      `json:"passed"`
	Failed int      `json:"failed"`
	New    int      `json:"new"`
	// Known is false when the gates' output names no case. Then Note says how to get cases.
	Known bool   `json:"known"`
	Note  string `json:"note,omitempty"`
	// Rerun is the command a rerun of the failing cases would run.
	Rerun string `json:"rerun,omitempty"`
}

// QACaseFilter narrows the cases. Status is "", "failed", "passed", "skipped" or "new".
type QACaseFilter struct {
	Status string
	Group  string
	Query  string
}

// casesOf parses every gate's output. withNew asks git for the tests the run added, which costs
// a diff, so only the case list pays for it.
func casesOf(root, id string, snap engine.Snapshot, gates []QAGate, withNew bool) []QACase {
	var out []QACase
	var added map[string]bool
	if withNew {
		added = addedTests(root, id, snap)
	}
	history := rerunHistory(root, id)
	for _, g := range gates {
		if !g.Ran {
			continue
		}
		text := gateOutput(root, id, snap, g)
		parsed, ok := qa.ParseGoTests(text)
		if !ok {
			continue
		}
		parsed = qa.PlaceTests(snap.Cursor.Workspace, parsed)
		if added != nil {
			parsed = qa.MarkNew(parsed, added)
		}
		for _, c := range parsed {
			item := QACase{TestCase: c, Gate: g.Name}
			item.History = historyOf(g, c, history)
			out = append(out, item)
		}
	}
	return out
}

// gateOutput is the whole log of a gate when it was kept and still readable, else the tail.
func gateOutput(root, id string, snap engine.Snapshot, g QAGate) string {
	if g.LogPath != "" {
		if view, err := readRunLog(root, id, g.LogPath); err == nil {
			return view.Text
		}
	}
	text, _ := gateText(snap, g)
	return text
}

func addedTests(root, id string, snap engine.Snapshot) map[string]bool {
	if snap.Cursor.Workspace == "" || snap.Cursor.Base == "" || snap.Cursor.GitSHA == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	diff, err := readImplementation(ctx, root, id)
	if err != nil {
		return nil
	}
	return qa.AddedTests(diff.Patch)
}

// ReadQACases lists the cases of a run, filtered.
func ReadQACases(root, id string, filter QACaseFilter) (QACases, error) {
	snap, err := loadSnapshot(root, id)
	if err != nil {
		return QACases{}, err
	}
	gates := qaGates(snap.Statements)
	all := casesOf(root, id, snap, gates, true)
	body := QACases{Run: id, Known: len(all) > 0, Cases: []QACase{}}
	for _, c := range all {
		body.Total++
		switch c.Result {
		case qa.CaseFail:
			body.Failed++
		case qa.CasePass:
			body.Passed++
		}
		if c.New {
			body.New++
		}
	}
	body.Rerun = rerunCommand(gates, failedNames(all))
	if !body.Known {
		body.Note = "The output of the checks names no case. Run the tests with go test -json, or -v, and Span lists each case."
	}
	for _, c := range all {
		if keepCase(c, filter) {
			body.Cases = append(body.Cases, c)
		}
	}
	sortCases(body.Cases, filter.Group == "file")
	return body, nil
}

func keepCase(c QACase, f QACaseFilter) bool {
	switch f.Status {
	case "failed":
		if c.Result != qa.CaseFail {
			return false
		}
	case "passed":
		if c.Result != qa.CasePass {
			return false
		}
	case "skipped":
		if c.Result != qa.CaseSkip {
			return false
		}
	case "new":
		if !c.New {
			return false
		}
	}
	return f.Query == "" || strings.Contains(strings.ToLower(c.Name+" "+c.Path), strings.ToLower(f.Query))
}

// sortCases puts failures first, within each file when the list is grouped by file.
func sortCases(cases []QACase, byFile bool) {
	rank := func(c QACase) int {
		switch c.Result {
		case qa.CaseFail:
			return 0
		case qa.CaseSkip:
			return 1
		default:
			return 2
		}
	}
	sort.SliceStable(cases, func(i, j int) bool {
		if byFile && cases[i].Path != cases[j].Path {
			// Cases the declarations could not place group last.
			if cases[i].Path == "" || cases[j].Path == "" {
				return cases[j].Path == ""
			}
			return cases[i].Path < cases[j].Path
		}
		return rank(cases[i]) < rank(cases[j])
	})
}

func failedNames(cases []QACase) []string {
	var names []string
	for _, c := range cases {
		if c.Result == qa.CaseFail {
			names = append(names, c.Name)
		}
	}
	return names
}

// ReadQACase reads one case by name, with its history.
func ReadQACase(root, id, name string) (QACase, error) {
	snap, err := loadSnapshot(root, id)
	if err != nil {
		return QACase{}, err
	}
	for _, c := range casesOf(root, id, snap, qaGates(snap.Statements), true) {
		if c.Name == name {
			return c, nil
		}
	}
	return QACase{}, &requestError{"not_found", "case " + name + " is not in this run"}
}

// rerun is one rerun the page asked for, stored beside the run so the case history shows it.
type rerun struct {
	Time     time.Time   `json:"time"`
	Gate     string      `json:"gate"`
	Command  string      `json:"command"`
	ExitCode int         `json:"exitCode"`
	Cases    []rerunCase `json:"cases"`
}

// rerunCase is what one case did in a rerun.
type rerunCase struct {
	Name      string `json:"name"`
	Result    string `json:"result"`
	ElapsedMS int64  `json:"elapsedMs"`
	Message   string `json:"message,omitempty"`
}

var rerunMu sync.Mutex

func rerunHistory(root, id string) []rerun {
	raw, err := os.ReadFile(filepath.Join(root, id, "qa-reruns.json"))
	if err != nil {
		return nil
	}
	var out []rerun
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	return out
}

// historyOf lists the first result of a case and each rerun of it.
func historyOf(g QAGate, c qa.TestCase, reruns []rerun) []QACaseHistory {
	list := []QACaseHistory{{Time: g.EndedAt, Kind: "first run", Result: c.Result, Message: c.Message}}
	for _, r := range reruns {
		for _, rc := range r.Cases {
			if rc.Name == c.Name {
				list = append(list, QACaseHistory{Time: r.Time, Kind: "rerun", Result: rc.Result, Message: rc.Message})
			}
		}
	}
	return list
}

var (
	caseName   = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	valuedFlag = map[string]bool{
		"-tags": true, "-timeout": true, "-p": true, "-parallel": true, "-cpu": true,
		"-bench": true, "-coverprofile": true, "-covermode": true, "-run": true, "-count": true,
		"-coverpkg": true, "-ldflags": true, "-gcflags": true, "-race": false,
	}
)

// rerunArgs builds the argv that runs only the named cases of a `go test` command: its packages
// and its flags are kept, -run and -count are replaced, and -json is added to read the result.
// A command that is not `go test` cannot be rerun case by case.
func rerunArgs(command string, names []string) ([]string, error) {
	fields := strings.Fields(command)
	if len(fields) < 2 || fields[0] != "go" || fields[1] != "test" {
		return nil, &requestError{"invalid", "only a go test command can be rerun case by case; this gate runs " + command}
	}
	if len(names) == 0 || len(names) > qaRerunLimit {
		return nil, &requestError{"invalid", fmt.Sprintf("name between 1 and %d cases to rerun", qaRerunLimit)}
	}
	for _, name := range names {
		if !caseName.MatchString(name) {
			return nil, &requestError{"invalid", fmt.Sprintf("%q is not a test name", name)}
		}
	}
	args := []string{"test"}
	rest := fields[2:]
	for i := 0; i < len(rest); i++ {
		arg := rest[i]
		switch {
		case arg == "-json" || arg == "-v":
		case strings.HasPrefix(arg, "-run=") || strings.HasPrefix(arg, "-count="):
		case arg == "-run" || arg == "-count":
			i++
		case strings.HasPrefix(arg, "-"):
			args = append(args, arg)
			if valuedFlag[arg] && i+1 < len(rest) {
				i++
				args = append(args, rest[i])
			}
		default:
			args = append(args, arg)
		}
	}
	pattern := "^(" + strings.Join(names, "|") + ")$"
	return append(args, "-run", pattern, "-count=1", "-json"), nil
}

func rerunCommand(gates []QAGate, names []string) string {
	for _, g := range gates {
		if !g.Ran || g.Verdict != qaFail {
			continue
		}
		if args, err := rerunArgs(g.Command, names); err == nil {
			return "go " + strings.Join(args, " ")
		}
	}
	return ""
}

// goTest runs `go test` in a folder and returns what it printed and its exit code. A variable so
// a test can answer without compiling anything.
var goTest = func(ctx context.Context, dir string, args []string) (string, int, error) {
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = dir
	raw, err := cmd.CombinedOutput()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return string(raw), exit.ExitCode(), nil
	}
	return string(raw), 0, err
}

// QARerun is what a rerun answers: the cases it ran and where each one stands now.
type QARerun struct {
	Run      string   `json:"run"`
	Command  string   `json:"command"`
	ExitCode int      `json:"exitCode"`
	Cases    []QACase `json:"cases"`
}

// RerunQA runs the named cases again on the run's checkout, with no shell: the argv comes from
// the failing `go test` command and names validated by pattern. The result is added to the
// run's history and the first result stays. A run that is going on is refused.
func RerunQA(ctx context.Context, root, id string, names []string) (QARerun, error) {
	snap, err := loadSnapshot(root, id)
	if err != nil {
		return QARerun{}, err
	}
	if _, busy := engine.RunBusy(root, id); busy {
		return QARerun{}, &requestError{"busy", "this run is going on; wait for it to stop before rerunning cases"}
	}
	dir := snap.Cursor.Workspace
	if info, err := os.Stat(dir); dir == "" || err != nil || !info.IsDir() {
		return QARerun{}, &requestError{"invalid", "the run's checkout is not on this machine"}
	}
	gates := qaGates(snap.Statements)
	var args []string
	var gate QAGate
	for _, g := range gates {
		if !g.Ran {
			continue
		}
		if a, err := rerunArgs(g.Command, names); err == nil && hasCases(root, id, snap, g, names) {
			args, gate = a, g
			break
		}
	}
	if args == nil {
		return QARerun{}, &requestError{"invalid", "none of these cases belongs to a go test gate of this run"}
	}
	ctx, cancel := context.WithTimeout(ctx, qaRerunTimeout)
	defer cancel()
	text, code, err := goTest(ctx, dir, args)
	if err != nil {
		return QARerun{}, err
	}
	parsed, _ := qa.ParseGoTests(text)
	parsed = qa.PlaceTests(dir, parsed)
	entry := rerun{Time: time.Now().UTC(), Gate: gate.Name, Command: "go " + strings.Join(args, " "), ExitCode: code}
	out := QARerun{Run: id, Command: entry.Command, ExitCode: code, Cases: []QACase{}}
	for _, c := range parsed {
		entry.Cases = append(entry.Cases, rerunCase{c.Name, c.Result, c.ElapsedMS, c.Message})
		out.Cases = append(out.Cases, QACase{TestCase: c, Gate: gate.Name})
	}
	if err := saveRerun(root, id, entry); err != nil {
		return QARerun{}, err
	}
	detail := map[string]string{"gate": gate.Name, "cases": fmt.Sprint(len(names)), "exit": fmt.Sprint(code)}
	if err := journal.Append(filepath.Join(root, id), journal.Person, "qa.rerun", detail); err != nil {
		return QARerun{}, err
	}
	return out, nil
}

// hasCases reports whether every name is a case of the gate's output.
func hasCases(root, id string, snap engine.Snapshot, g QAGate, names []string) bool {
	have := map[string]bool{}
	for _, c := range casesOf(root, id, snap, []QAGate{g}, false) {
		have[c.Name] = true
	}
	for _, name := range names {
		if !have[name] {
			return false
		}
	}
	return true
}

// maxQABody caps a QA request body; the bodies are a few names or one sentence.
const maxQABody = 1 << 16

// saveRerun appends one rerun to the run's history, atomically.
func saveRerun(root, id string, entry rerun) error {
	rerunMu.Lock()
	defer rerunMu.Unlock()
	list := append(rerunHistory(root, id), entry)
	raw, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return fsutil.WriteAtomic(filepath.Join(root, id, "qa-reruns.json"), append(raw, '\n'))
}

// readQAAccepted reads the risk a person accepted for a run, if any.
func readQAAccepted(root, id string) (QAAcceptance, bool) {
	raw, err := os.ReadFile(filepath.Join(root, id, "qa-accept.json"))
	if err != nil {
		return QAAcceptance{}, false
	}
	var a QAAcceptance
	if err := json.Unmarshal(raw, &a); err != nil {
		return QAAcceptance{}, false
	}
	return a, true
}

// AcceptQA records that a person goes on with a gate that did not prove anything, with the
// reason they gave. It records the risk with the time and the journal's actor and changes nothing
// else: the run still waits for a resume, which reads this decision.
func AcceptQA(root, id, reason string) (QAAcceptance, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return QAAcceptance{}, &requestError{"invalid", "accepting a gate without proof needs a reason"}
	}
	view, err := ReadQA(root, id)
	if err != nil {
		return QAAcceptance{}, err
	}
	if view.Verdict == qaPass && view.State == qaOK {
		return QAAcceptance{}, &requestError{"invalid", "the gate passed; there is nothing to accept"}
	}
	gate := ""
	if g, ok := decisive(view.Gates); ok {
		gate = g.Name
	}
	accepted := QAAcceptance{Reason: reason, Time: time.Now().UTC(), Gate: gate}
	raw, err := json.MarshalIndent(accepted, "", "  ")
	if err != nil {
		return QAAcceptance{}, err
	}
	if err := os.WriteFile(filepath.Join(root, id, "qa-accept.json"), append(raw, '\n'), 0o644); err != nil {
		return QAAcceptance{}, err
	}
	detail := map[string]string{"gate": gate, "reasonLength": fmt.Sprint(len(reason))}
	if err := journal.Append(filepath.Join(root, id), journal.Person, "qa.accepted", detail); err != nil {
		return QAAcceptance{}, err
	}
	return accepted, nil
}

// writeQAReadErr answers a failed QA reading: a run that is not on this desk is a JSON 404 like
// every other QA route, anything else keeps the status its own code carries.
func writeQAReadErr(w http.ResponseWriter, id string, err error) {
	if errors.Is(err, os.ErrNotExist) {
		writeErr(w, &requestError{"not_found", "run " + id + " is not on this desk"})
		return
	}
	writeErr(w, err)
}

// qaRoutes mounts the QA readings and actions.
func qaRoutes(mux *http.ServeMux, root string) {
	mux.HandleFunc("GET /spans/{id}/qa", func(w http.ResponseWriter, r *http.Request) {
		view, err := ReadQA(root, r.PathValue("id"))
		if err != nil {
			writeQAReadErr(w, r.PathValue("id"), err)
			return
		}
		writeJSON(w, http.StatusOK, view)
	})
	mux.HandleFunc("GET /spans/{id}/qa/cases", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		body, err := ReadQACases(root, r.PathValue("id"), QACaseFilter{
			Status: query.Get("status"), Group: query.Get("group"), Query: query.Get("q"),
		})
		if err != nil {
			writeQAReadErr(w, r.PathValue("id"), err)
			return
		}
		writeJSON(w, http.StatusOK, body)
	})
	mux.HandleFunc("GET /spans/{id}/qa/cases/{case}", func(w http.ResponseWriter, r *http.Request) {
		body, err := ReadQACase(root, r.PathValue("id"), r.PathValue("case"))
		if err != nil {
			writeQAReadErr(w, r.PathValue("id"), err)
			return
		}
		writeJSON(w, http.StatusOK, body)
	})
	mux.HandleFunc("POST /spans/{id}/qa/rerun", func(w http.ResponseWriter, r *http.Request) {
		body, ok := decodeJSON[struct {
			Cases []string `json:"cases"`
		}](w, r, maxQABody)
		if !ok {
			return
		}
		rerunReply(w, r, root, body.Cases)
	})
	mux.HandleFunc("POST /spans/{id}/qa/cases/{case}/rerun", func(w http.ResponseWriter, r *http.Request) {
		rerunReply(w, r, root, []string{r.PathValue("case")})
	})
	mux.HandleFunc("POST /spans/{id}/qa/accept", func(w http.ResponseWriter, r *http.Request) {
		body, ok := decodeJSON[struct {
			Reason string `json:"reason"`
		}](w, r, maxQABody)
		if !ok {
			return
		}
		accepted, err := AcceptQA(root, r.PathValue("id"), body.Reason)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, accepted)
	})
	mux.HandleFunc("GET /spans/{id}/logs/{command}", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		view, err := ReadGateLog(root, r.PathValue("id"), r.PathValue("command"), LogQuery{
			From: atoi(query.Get("from")), To: atoi(query.Get("to")), Q: query.Get("q"),
			FirstError: query.Get("first_error") == "1",
		})
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, view)
	})
}

func rerunReply(w http.ResponseWriter, r *http.Request, root string, names []string) {
	body, err := RerunQA(r.Context(), root, r.PathValue("id"), names)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, body)
}

// atoi reads a query number; anything that is not a positive number is 0, which means unset.
func atoi(raw string) int {
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return 0
	}
	return n
}
