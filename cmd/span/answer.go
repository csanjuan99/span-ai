package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"span/internal/decision"
	"span/internal/engine"
	"span/internal/runlock"
)

// verbFlags are the flags every verb that reads or answers a waiting run takes.
type verbFlags struct {
	spans    string
	asJSON   bool
	noResume bool
}

// verbSet declares the shared flags on a verb's flag set. Answering verbs also take
// --no-resume; reading ones do not.
func verbSet(name string, errw io.Writer, answers bool) (*flag.FlagSet, *verbFlags) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(errw)
	v := &verbFlags{}
	fs.StringVar(&v.spans, "spans", ".span/runs", "state directory the runs live in")
	fs.BoolVar(&v.asJSON, "json", false, "print the outcome as JSON")
	if answers {
		fs.BoolVar(&v.noResume, "no-resume", false, "only record the answer; continue the run later with span resume or from the page")
	}
	return fs, v
}

// parseVerb reads the arguments of a verb in any order and returns the positional ones. The
// first is the run id.
func parseVerb(fs *flag.FlagSet, args []string) ([]string, bool) {
	var positional []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, false
		}
		if fs.NArg() == 0 {
			return positional, true
		}
		positional = append(positional, fs.Arg(0))
		args = fs.Args()[1:]
	}
}

// continueRun is how an answered run goes on: in this process, the way `span resume` does,
// holding the run's lock. A variable so a test can observe it without driving an agent.
var continueRun = func(ctx context.Context, spans, id string, answer *decision.Answer) (engine.Result, error) {
	opt, _, err := existingRunOptions(spans, id)
	if err != nil {
		return engine.Result{}, err
	}
	if holder, busy := engine.RunBusy(spans, id); busy {
		return engine.Result{}, &runlock.BusyError{Scope: runlock.Run, Holder: holder}
	}
	if answer != nil {
		return engine.Decide(ctx, opt, id, *answer)
	}
	return engine.Resume(ctx, opt, id)
}

// outcome is what an answering verb reports. Exit 0 means the answer was recorded and, unless
// --no-resume, the run went on until it next stopped; the verdict says where it stopped.
type outcome struct {
	ID      string `json:"id"`
	Action  string `json:"action"`
	Stage   string `json:"stage"`
	Resumed bool   `json:"resumed"`
	Verdict string `json:"verdict,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

// answered records nothing itself: the caller already did. It continues the run unless asked
// not to and prints what happened.
func answered(out, errw io.Writer, v *verbFlags, id, action string, answer *decision.Answer) int {
	snap, err := engine.Load(v.spans, id)
	if err != nil {
		fmt.Fprintln(errw, err)
		return 1
	}
	res := outcome{ID: id, Action: action, Stage: snap.Cursor.Stage}
	if !v.noResume {
		ctx, stop := interruptContext()
		defer stop()
		result, err := continueRun(ctx, v.spans, id, answer)
		if err != nil {
			fmt.Fprintln(errw, err)
			return 1
		}
		res.Resumed, res.Stage, res.Verdict, res.Reason = true, result.Stage, result.Verdict, result.Reason
	}
	return reportOutcome(out, errw, v.asJSON, res)
}

// reportOutcome prints what an acting verb did, as JSON or as one plain sentence.
func reportOutcome(out, errw io.Writer, asJSON bool, res outcome) int {
	if asJSON {
		return emit(out, errw, res)
	}
	if res.Resumed {
		fmt.Fprintf(out, "%s %s: recorded, the run went on: %s %s\n%s\n", res.ID, res.Action, res.Stage, res.Verdict, res.Reason)
	} else {
		fmt.Fprintf(out, "%s %s: recorded. Continue with ./span resume %s\n", res.ID, res.Action, res.ID)
	}
	return 0
}

func decisionCmd(args []string) int { return decisionMain(args, os.Stdout, os.Stderr) }

const decisionUsage = "usage: ./span decision show <run> [--json] | choose <run> <option> [--text \"...\"] [--decision <id>] [--no-resume] [--json] [--spans .span/runs]"

// decisionMain runs one `span decision` verb: the same two things the decision panel does.
func decisionMain(args []string, out, errw io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(errw, decisionUsage)
		return 2
	}
	switch args[0] {
	case "show", "choose":
	default:
		fmt.Fprintln(errw, decisionUsage)
		return 2
	}
	fs, v := verbSet("decision "+args[0], errw, args[0] == "choose")
	text := fs.String("text", "", "your words for the agent")
	which := fs.String("decision", "", "the decision this answer is for (default: the open one)")
	pos, ok := parseVerb(fs, args[1:])
	if !ok || len(pos) < 1 {
		fmt.Fprintln(errw, decisionUsage)
		return 2
	}
	id := pos[0]
	snap, err := engine.Load(v.spans, id)
	if err != nil {
		fmt.Fprintln(errw, err)
		return 1
	}
	req := snap.Cursor.Decision
	if req == nil {
		fmt.Fprintln(errw, "this run has no open decision")
		return 1
	}
	if args[0] == "show" {
		if len(pos) != 1 {
			fmt.Fprintln(errw, decisionUsage)
			return 2
		}
		return printDecision(out, errw, v.asJSON, *req)
	}
	if len(pos) != 2 {
		fmt.Fprintln(errw, decisionUsage)
		return 2
	}
	answer := decision.Answer{ID: *which, Option: pos[1], Text: *text}
	if answer.ID == "" {
		answer.ID = req.ID
	}
	if _, err := req.Choose(answer); err != nil {
		fmt.Fprintln(errw, err)
		return 1
	}
	return answered(out, errw, v, id, "decision", &answer)
}

func printDecision(out, errw io.Writer, asJSON bool, req decision.Request) int {
	if asJSON {
		return emit(out, errw, req)
	}
	fmt.Fprintf(out, "decision %s (%s, stage %s)\n%s\n%s\n", req.ID, req.Class, req.Stage, req.Title, req.Happened)
	for _, e := range req.Evidence {
		fmt.Fprintf(out, "\n  %s\n    %s\n", e.Ref, e.Excerpt)
	}
	for i, o := range req.Options {
		mark := ""
		if o.ID == req.Recommendation {
			mark = "  [recommended]"
		}
		fmt.Fprintf(out, "\n  %c  %s  %s%s\n     %s\n", 'A'+i, o.ID, o.Label, mark, o.Consequence)
		if o.Gain != "" || o.Cost != "" {
			fmt.Fprintf(out, "     gain: %s | cost: %s\n", o.Gain, o.Cost)
		}
		if !o.Reversible {
			fmt.Fprintln(out, "     This cannot be undone from here.")
		}
	}
	return 0
}
