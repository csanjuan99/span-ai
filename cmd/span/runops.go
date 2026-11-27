package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"span/internal/adapt"
	"span/internal/agent"
	"span/internal/engine"
	"span/internal/jira"
	"span/internal/runlock"
	"span/internal/serve"
)

// runopsUsage is what the run-page verbs print when one is called wrongly. Each verb does what
// the matching route of `span serve` does, with the same engine code, so a run can be driven
// from a terminal on the machine that holds it.
const runopsUsage = `usage: ./span runs list|show|implementation|feature|acceptance|events <run> [--json] [--patch]
       ./span log <run> <path under iter/, ending .log> [--json]
       ./span pause <run> [--json]
       ./span stop <run> [--json]
       ./span rewind <run> <stage> [--note "..."] [--no-resume] [--json]
       ./span note <run> <stage> --text "..." | --file path [--json]
       ./span stage-agent <run> <stage> <agent|any> [--json]
       ./span approve <run> <stage> [--environment name] [--no-resume] [--json]
       ./span answer <run> --answer "question=text"... | --file answers.json [--json]
       ./span story <run> [--json]
       ./span delete <run> --yes [--json]
       (all take --spans .span/runs; exit 0 done, 1 refused or failed, 2 usage; stop also 3 run finished, 4 run not found)`

func runsCmd(args []string) int       { return runsMain(args, os.Stdout, os.Stderr) }
func logCmd(args []string) int        { return logMain(args, os.Stdout, os.Stderr) }
func pauseCmd(args []string) int      { return pauseMain(args, os.Stdout, os.Stderr) }
func stopCmd(args []string) int       { return stopMain(args, os.Stdout, os.Stderr) }
func rewindCmd(args []string) int     { return rewindMain(args, os.Stdout, os.Stderr) }
func noteCmd(args []string) int       { return noteMain(args, os.Stdout, os.Stderr) }
func stageAgentCmd(args []string) int { return stageAgentMain(args, os.Stdout, os.Stderr) }
func approveCmd(args []string) int    { return approveMain(args, os.Stdout, os.Stderr) }
func answerCmd(args []string) int     { return answerMain(args, os.Stdout, os.Stderr) }
func deleteCmd(args []string) int     { return deleteMain(args, os.Stdout, os.Stderr) }
func storyCmd(args []string) int      { return storyMain(args, os.Stdout, os.Stderr) }

// usageError prints the usage and answers the exit code of a wrong call.
func usageError(errw io.Writer) int {
	fmt.Fprintln(errw, runopsUsage)
	return 2
}

// emit prints a value as indented JSON, the shape the routes answer with.
func emit(out, errw io.Writer, value any) int {
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		fmt.Fprintln(errw, err)
		return 1
	}
	fmt.Fprintln(out, string(body))
	return 0
}

// result is what an acting verb reports: the run, what was done, and a plain reason.
type result struct {
	ID     string `json:"id"`
	Action string `json:"action"`
	Done   bool   `json:"done"`
	Reason string `json:"reason"`
}

// report prints an acting verb's result as JSON or as one line.
func report(out, errw io.Writer, v *verbFlags, res result) int {
	if v.asJSON {
		return emit(out, errw, res)
	}
	fmt.Fprintf(out, "%s %s: %s\n", res.ID, res.Action, res.Reason)
	return 0
}

// loadRun reads a run, which every verb needs before it acts on it.
func loadRun(errw io.Writer, v *verbFlags, id string) (engine.Snapshot, bool) {
	snap, err := engine.Load(v.spans, id)
	if err != nil {
		fmt.Fprintln(errw, err)
		return engine.Snapshot{}, false
	}
	return snap, true
}

func runsMain(args []string, out, errw io.Writer) int {
	if len(args) == 0 {
		return usageError(errw)
	}
	verb := args[0]
	fs, v := verbSet("runs "+verb, errw, false)
	patch := fs.Bool("patch", false, "implementation: print the whole patch, not only the stat")
	pos, ok := parseVerb(fs, args[1:])
	if !ok {
		return usageError(errw)
	}
	if verb == "list" {
		if len(pos) != 0 {
			return usageError(errw)
		}
		return listRuns(out, errw, v)
	}
	if len(pos) != 1 {
		return usageError(errw)
	}
	id := pos[0]
	switch verb {
	case "show":
		return showRun(out, errw, v, id)
	case "implementation":
		return showImplementation(out, errw, v, id, *patch)
	case "feature":
		return showFeature(out, errw, v, id)
	case "acceptance":
		return showAcceptance(out, errw, v, id)
	case "events":
		return showEvents(out, errw, v, id)
	}
	return usageError(errw)
}

// listRuns prints what GET /spans lists, the heads the desk reads.
func listRuns(out, errw io.Writer, v *verbFlags) int {
	heads := engine.Heads(v.spans)
	if v.asJSON {
		return emit(out, errw, map[string]any{"spans": heads})
	}
	for _, head := range heads {
		need := ""
		if head.Need != "" {
			need = fmt.Sprintf("  needs %s (%d)", head.Need, head.Count)
		}
		fmt.Fprintf(out, "%-24s %-8s %-8s %s%s\n", head.ID, head.Stage, head.Verdict, head.Title, need)
	}
	return 0
}

// view prints what a read route answers: the error as one line and exit 1, the value as JSON
// with --json, or whatever text prints and returns as the exit code.
func view[T any](out, errw io.Writer, v *verbFlags, value T, err error, text func(T) int) int {
	if err != nil {
		fmt.Fprintln(errw, err)
		return 1
	}
	if v.asJSON {
		return emit(out, errw, value)
	}
	return text(value)
}

// showRun prints what GET /spans/{id} answers: the cursor, the statements and the gate.
func showRun(out, errw io.Writer, v *verbFlags, id string) int {
	snap, ok := loadRun(errw, v, id)
	if !ok {
		return 1
	}
	return view(out, errw, v, snap, nil, func(snap engine.Snapshot) int {
		cur := snap.Cursor
		fmt.Fprintf(out, "run %s\nstage %s\nbranch %s\nheld %t  stopped %t  done %t\n", id, cur.Stage, cur.Branch, cur.Hold, cur.Stop, cur.Done)
		if snap.Gate != nil {
			fmt.Fprintf(out, "gate %s  %s\n", snap.Gate.Verdict, snap.Gate.Command)
		}
		return 0
	})
}

// showImplementation prints the commits and the stat of the run's branch, and its patch on request.
func showImplementation(out, errw io.Writer, v *verbFlags, id string, patch bool) int {
	ctx, stop := interruptContext()
	defer stop()
	diff, err := serve.ReadImplementation(ctx, v.spans, id)
	return view(out, errw, v, diff, err, func(diff adapt.Diff) int {
		fmt.Fprintf(out, "%d commits\n", diff.Commits)
		for _, commit := range diff.CommitLog {
			fmt.Fprintf(out, "  %.8s  %s  %s\n", commit.SHA, commit.Time, commit.Subject)
		}
		fmt.Fprintf(out, "\n%s\n", strings.TrimRight(diff.Stat, "\n"))
		if patch {
			fmt.Fprintf(out, "\n%s\n", diff.Patch)
		}
		return 0
	})
}

// showFeature prints the feature document of the run.
func showFeature(out, errw io.Writer, v *verbFlags, id string) int {
	feature, err := serve.ReadFeature(v.spans, id)
	return view(out, errw, v, feature, err, func(feature serve.Feature) int {
		fmt.Fprint(out, feature.Markdown)
		return 0
	})
}

// showAcceptance prints the gate and the acceptance cases of the run.
func showAcceptance(out, errw io.Writer, v *verbFlags, id string) int {
	body, err := serve.ReadAcceptance(v.spans, id)
	return view(out, errw, v, body, err, func(body serve.Acceptance) int {
		if body.Skipped {
			fmt.Fprintln(out, "no acceptance cases")
			return 0
		}
		fmt.Fprintf(out, "gate %s  exit %d  %s\n", body.Verdict, body.ExitCode, body.Command)
		for _, line := range body.Lines {
			place := ""
			if line.File != "" {
				place = fmt.Sprintf("  %s:%d", line.File, line.Line)
			}
			fmt.Fprintf(out, "  %-8s %s  %s%s\n", line.Result, line.ID, line.Text, place)
		}
		return 0
	})
}

// showEvents prints the audit chain of the run and exits 1 when it is broken.
func showEvents(out, errw io.Writer, v *verbFlags, id string) int {
	events, err := serve.ReadEvents(v.spans, id)
	return view(out, errw, v, events, err, func(events serve.Events) int {
		for _, event := range events.Events {
			fmt.Fprintf(out, "%4d  %s  %-8s %s\n", event.Seq, event.Time.Format(time.RFC3339), event.Actor, event.Type)
		}
		if !events.Intact {
			fmt.Fprintf(out, "the chain is broken at event %d\n", events.BrokenAt)
			return 1
		}
		return 0
	})
}

// logMain prints the full log of one command, the page's "Open full log". The path is the one a
// statement stores; the limits are the route's: a .log under iter/, the last 512 KiB.
func logMain(args []string, out, errw io.Writer) int {
	fs, v := verbSet("log", errw, false)
	pos, ok := parseVerb(fs, args)
	if !ok || len(pos) != 2 {
		return usageError(errw)
	}
	log, err := serve.ReadRunLog(v.spans, pos[0], pos[1])
	return view(out, errw, v, log, err, func(log serve.RunLog) int {
		fmt.Fprint(out, log.Text)
		if log.Truncated {
			fmt.Fprintln(errw, "the log is longer than 512 KiB: this is its end")
		}
		return 0
	})
}

// pauseMain asks the run to hold before its next stage, as the page's Pause does. The request
// is a file the running process reads at the stage boundary, so it also reaches a run driven
// by `span serve`. A request made while nothing runs waits until the next resume discards it.
func pauseMain(args []string, out, errw io.Writer) int {
	fs, v := verbSet("pause", errw, false)
	pos, ok := parseVerb(fs, args)
	if !ok || len(pos) != 1 {
		return usageError(errw)
	}
	id := pos[0]
	snap, found := loadRun(errw, v, id)
	if !found {
		return 1
	}
	if snap.Cursor.Done || snap.Cursor.Stop {
		fmt.Fprintln(errw, "this run is finished; a pause only applies to a stage in flight")
		return 1
	}
	if err := engine.RequestPause(v.spans, id); err != nil {
		fmt.Fprintln(errw, err)
		return 1
	}
	return report(out, errw, v, result{ID: id, Action: "pause", Done: true, Reason: "the run holds before the next stage"})
}

// stopMain asks the process driving the run to cancel the stage in flight, as the page's
// "Stop now" does. The request is a file that process reads, so it reaches a run driven by
// `span serve`, `span run` or `span resume` alike. When nothing drives the run there is no
// stage to cancel: that is said plainly and no request is left to cancel a later resume.
// Exit 0 asked or nothing running, 3 the run is finished, 4 no such run.
func stopMain(args []string, out, errw io.Writer) int {
	fs, v := verbSet("stop", errw, false)
	pos, ok := parseVerb(fs, args)
	if !ok || len(pos) != 1 {
		return usageError(errw)
	}
	id := pos[0]
	snap, err := engine.Load(v.spans, id)
	if errors.Is(err, os.ErrNotExist) {
		fmt.Fprintln(errw, err)
		return 4
	}
	if err != nil {
		fmt.Fprintln(errw, err)
		return 1
	}
	if snap.Cursor.Done || snap.Cursor.Stop {
		fmt.Fprintln(errw, "this run is finished; a stop only applies to a stage in flight")
		return 3
	}
	if _, driven := engine.RunBusy(v.spans, id); !driven {
		return report(out, errw, v, result{ID: id, Action: "stop", Done: false, Reason: "nothing is running this run"})
	}
	if err := engine.RequestStop(v.spans, id); err != nil {
		fmt.Fprintln(errw, err)
		return 1
	}
	return report(out, errw, v, result{ID: id, Action: "stop", Done: true, Reason: "the stage in flight is cancelled; the run holds with the error"})
}

// rewindMain restarts a run at a stage, as the page's "Restart from a stage" does, with an
// optional note the stage reads, and then lets the run go on unless asked not to.
func rewindMain(args []string, out, errw io.Writer) int {
	fs, v := verbSet("rewind", errw, true)
	note := fs.String("note", "", "an instruction the restarted stage reads")
	pos, ok := parseVerb(fs, args)
	if !ok || len(pos) != 2 {
		return usageError(errw)
	}
	id, stage := pos[0], pos[1]
	if _, found := loadRun(errw, v, id); !found {
		return 1
	}
	if text := strings.TrimSpace(*note); text != "" {
		if err := serve.SaveInstruction(v.spans, id, stage, []byte(text)); err != nil {
			fmt.Fprintln(errw, err)
			return 1
		}
	}
	if err := engine.Rewind(v.spans, id, stage); err != nil {
		fmt.Fprintln(errw, err)
		return 1
	}
	return answered(out, errw, v, id, "rewind to "+stage, nil)
}

// noteMain leaves an instruction for a stage, the page's "Leave a note for <stage>". A note is
// read by the stage when it next runs; it does not wake the run.
func noteMain(args []string, out, errw io.Writer) int {
	fs, v := verbSet("note", errw, false)
	text := fs.String("text", "", "the instruction")
	file := fs.String("file", "", "read the instruction from a file")
	pos, ok := parseVerb(fs, args)
	if !ok || len(pos) != 2 || (*text == "") == (*file == "") {
		return usageError(errw)
	}
	id, stage := pos[0], pos[1]
	body := []byte(*text)
	if *file != "" {
		raw, err := os.ReadFile(*file)
		if err != nil {
			fmt.Fprintln(errw, err)
			return 1
		}
		body = raw
	}
	if _, found := loadRun(errw, v, id); !found {
		return 1
	}
	if err := serve.SaveInstruction(v.spans, id, stage, body); err != nil {
		fmt.Fprintln(errw, err)
		return 1
	}
	return report(out, errw, v, result{ID: id, Action: "note for " + stage, Done: true, Reason: "saved; the stage reads it when it next runs"})
}

// stageAgentMain chooses which agent CLI one stage of one run uses, the page's "Agent for
// <stage>". "any" clears the choice and falls back to the policy.
func stageAgentMain(args []string, out, errw io.Writer) int {
	fs, v := verbSet("stage-agent", errw, false)
	pos, ok := parseVerb(fs, args)
	if !ok || len(pos) != 3 {
		return usageError(errw)
	}
	id, stage, name := pos[0], pos[1], pos[2]
	if _, found := loadRun(errw, v, id); !found {
		return 1
	}
	if err := engine.SetStageAgent(filepath.Join(v.spans, id), stage, name, agent.Home()); err != nil {
		fmt.Fprintln(errw, err)
		return 1
	}
	return report(out, errw, v, result{ID: id, Action: "agent for " + stage, Done: true, Reason: name})
}

// approveMain releases a stage that waits for a yes, as the page's Approve does, then lets the
// run go on unless asked not to.
func approveMain(args []string, out, errw io.Writer) int {
	fs, v := verbSet("approve", errw, true)
	environment := fs.String("environment", "", "the environment the approval is for, when the stage asks")
	pos, ok := parseVerb(fs, args)
	if !ok || len(pos) != 2 {
		return usageError(errw)
	}
	id, stage := pos[0], pos[1]
	if _, found := loadRun(errw, v, id); !found {
		return 1
	}
	if err := engine.Approve(v.spans, id, stage, *environment); err != nil {
		fmt.Fprintln(errw, err)
		return 1
	}
	return answered(out, errw, v, id, "approve "+stage, nil)
}

// continueAnswers is how the answers to a run's questions go on: in this process, holding the
// run's lock, as the page's wake does. A variable so a test can observe it without an agent.
var continueAnswers = func(ctx context.Context, spans, id string, answers []engine.Answer) (engine.Result, error) {
	opt, _, err := existingRunOptions(spans, id)
	if err != nil {
		return engine.Result{}, err
	}
	return engine.Continue(ctx, opt, id, answers)
}

// answerMain answers the questions a run waits on, as the page's "Needs you" does, and lets
// the run go on. An answer is `question=text`; --file reads the page's JSON, a list of
// {"question","body"}.
func answerMain(args []string, out, errw io.Writer) int {
	fs, v := verbSet("answer", errw, false)
	var given repeated
	fs.Var(&given, "answer", `one answer as "question=text" (repeat for each question)`)
	file := fs.String("file", "", "read the answers from a JSON file")
	pos, ok := parseVerb(fs, args)
	if !ok || len(pos) != 1 || (len(given) == 0) == (*file == "") {
		return usageError(errw)
	}
	id := pos[0]
	answers, err := readAnswers(given, *file)
	if err != nil {
		fmt.Fprintln(errw, err)
		return 2
	}
	snap, found := loadRun(errw, v, id)
	if !found {
		return 1
	}
	if !engine.Waiting(snap.Cursor) {
		fmt.Fprintln(errw, "this run is not waiting for an answer")
		return 1
	}
	ctx, stop := interruptContext()
	defer stop()
	res, err := continueAnswers(ctx, v.spans, id, answers)
	if err != nil {
		fmt.Fprintln(errw, err)
		return 1
	}
	return reportContinued(out, errw, v.asJSON, id, "answer", res)
}

// reportContinued reports a run that a verb woke and drove again.
func reportContinued(out, errw io.Writer, asJSON bool, id, action string, res engine.Result) int {
	done := outcome{ID: id, Action: action, Stage: res.Stage, Outcome: outcomeContinued, Resumed: true, Verdict: res.Verdict, Reason: res.Reason}
	return reportOutcome(out, errw, asJSON, done)
}

// readStory reads the story the run came from out of Jira again, with the login saved for the
// run's folder. A variable so a test can run the verb without a Jira site.
var readStory = func(ctx context.Context, spans, id string) (string, error) {
	snap, err := engine.Load(spans, id)
	if err != nil {
		return "", err
	}
	key := snap.Cursor.Issue
	if key == "" {
		raw, err := os.ReadFile(filepath.Join(spans, id, "brief.md"))
		if err != nil {
			return "", err
		}
		key = engine.IssueKey(string(raw))
	}
	account, err := jira.Load(snap.Cursor.Workspace)
	if err != nil {
		return "", err
	}
	story, err := (jira.Client{Account: account}).Issue(ctx, key)
	if err != nil {
		return "", err
	}
	return story.Brief, nil
}

// reloadRun replaces the brief of a waiting run with the story and runs it again, in this
// process and holding the run's lock. A variable so a test can observe it without an agent.
var reloadRun = func(ctx context.Context, spans, id, brief string) (engine.Result, error) {
	opt, _, err := existingRunOptions(spans, id)
	if err != nil {
		return engine.Result{}, err
	}
	if holder, busy := engine.RunBusy(spans, id); busy {
		return engine.Result{}, &runlock.BusyError{Scope: runlock.Run, Holder: holder}
	}
	return engine.Reload(ctx, opt, id, brief)
}

// storyMain is the page's "Read the story again": it fetches the Jira story with the folder's
// login, replaces the brief and runs the waiting run again. Like the route, it only applies
// to a run that waits.
func storyMain(args []string, out, errw io.Writer) int {
	fs, v := verbSet("story", errw, false)
	pos, ok := parseVerb(fs, args)
	if !ok || len(pos) != 1 {
		return usageError(errw)
	}
	id := pos[0]
	snap, found := loadRun(errw, v, id)
	if !found {
		return 1
	}
	if !engine.Waiting(snap.Cursor) {
		fmt.Fprintln(errw, "this run is not waiting")
		return 1
	}
	ctx, stop := interruptContext()
	defer stop()
	brief, err := readStory(ctx, v.spans, id)
	if err != nil {
		fmt.Fprintln(errw, err)
		return 1
	}
	res, err := reloadRun(ctx, v.spans, id, brief)
	if err != nil {
		fmt.Fprintln(errw, err)
		return 1
	}
	return reportContinued(out, errw, v.asJSON, id, "story", res)
}

// readAnswers builds the answers of a call from its flags. An answer without words is refused,
// as the route refuses it.
func readAnswers(given []string, file string) ([]engine.Answer, error) {
	var answers []engine.Answer
	if file != "" {
		raw, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(raw, &answers); err != nil {
			return nil, fmt.Errorf("%s is not a list of answers: %w", file, err)
		}
	}
	for _, one := range given {
		question, body, found := strings.Cut(one, "=")
		if !found {
			return nil, fmt.Errorf("answer %q is not question=text", one)
		}
		answers = append(answers, engine.Answer{Question: question, Body: body})
	}
	if err := engine.CheckAnswers(answers); err != nil {
		return nil, err
	}
	return answers, nil
}

// deleteMain removes a run for good, the page's delete. The engine refuses a run another
// process is driving; --yes stands for the confirmation the page asks for.
func deleteMain(args []string, out, errw io.Writer) int {
	fs, v := verbSet("delete", errw, false)
	yes := fs.Bool("yes", false, "confirm that the run is removed for good")
	pos, ok := parseVerb(fs, args)
	if !ok || len(pos) != 1 || !*yes {
		return usageError(errw)
	}
	id := pos[0]
	if _, found := loadRun(errw, v, id); !found {
		return 1
	}
	if err := adapt.DeleteRun(context.Background(), v.spans, id); err != nil {
		fmt.Fprintln(errw, err)
		return 1
	}
	return report(out, errw, v, result{ID: id, Action: "delete", Done: true, Reason: "removed"})
}
