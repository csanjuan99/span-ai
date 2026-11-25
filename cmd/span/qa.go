package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"span/internal/serve"
)

// qaUsage is what the QA verbs print when one is called wrongly. Each verb reads or does what the
// matching route of `span serve` does, with the same code.
const qaUsage = `usage: ./span qa show <run> [--json]
       ./span qa cases <run> [--failed|--passed|--skipped|--new] [--group file] [--q text] [--json]
       ./span qa case <run> <name> [--json]
       ./span qa rerun <run> --failed | <name>... [--json]
       ./span qa accept <run> --reason "..." [--json]
       ./span qa log <run> <gate> [--lines from:to] [--grep text] [--first-error] [--json]
       ./span transcript <run> [--stage name] [--from n] [--limit n] [--json]
       (all take --spans .span/runs; exit 0 ok, 1 failed, 2 usage, 4 not found,
        10 a gate failed, 11 no proof, 13 a log is cut)`

// qaExitLogCut is the exit code of a log whose start was cut. A missing run, case, gate or stage
// exits with exitNotFound (4). Conflict and decision keep exit 1 for a missing run, as the run
// page verbs always have.
const qaExitLogCut = 13

func qaCmd(args []string) int         { return qaMain(args, os.Stdout, os.Stderr) }
func transcriptCmd(args []string) int { return transcriptMain(args, os.Stdout, os.Stderr) }

func qaUsageError(errw io.Writer) int {
	fmt.Fprintln(errw, qaUsage)
	return 2
}

// qaFailure prints an error and gives the exit code its kind earns: a missing run, case, gate or
// stage is 4, anything else 1. With --json the object is the one writeErr sends
// ({"error","message"}) so a script can read the same fields as the route.
func qaFailure(out, errw io.Writer, asJSON bool, err error) int {
	codeName := "error"
	var coded interface{ SpanCode() string }
	if errors.As(err, &coded) {
		codeName = coded.SpanCode()
	}
	exit := 1
	if codeName == "not_found" || errors.Is(err, os.ErrNotExist) {
		if codeName == "error" {
			codeName = "not_found"
		}
		exit = exitNotFound
	}
	if asJSON {
		raw, marshalErr := json.Marshal(map[string]string{"error": codeName, "message": err.Error()})
		if marshalErr != nil {
			fmt.Fprintln(errw, marshalErr)
			return 1
		}
		fmt.Fprintln(out, string(raw))
		return exit
	}
	fmt.Fprintln(errw, err)
	return exit
}

func qaMain(args []string, out, errw io.Writer) int {
	if len(args) == 0 {
		return qaUsageError(errw)
	}
	verb := args[0]
	fs, v := verbSet("qa "+verb, errw, false)
	failed := fs.Bool("failed", false, "cases, rerun: only the failing cases")
	passed := fs.Bool("passed", false, "cases: only the passing cases")
	skipped := fs.Bool("skipped", false, "cases: only the skipped cases")
	fresh := fs.Bool("new", false, "cases: only the cases this run added")
	group := fs.String("group", "", "cases: file groups the cases by file")
	query := fs.String("q", "", "cases: only cases whose name or file holds this text")
	reason := fs.String("reason", "", "accept: why the risk is accepted")
	lines := fs.String("lines", "", "log: a range of lines, from:to")
	grep := fs.String("grep", "", "log: search the whole log, case-insensitive")
	first := fs.Bool("first-error", false, "log: the lines around the first error")
	pos, ok := parseVerb(fs, args[1:])
	if !ok || len(pos) == 0 {
		return qaUsageError(errw)
	}
	id, rest := pos[0], pos[1:]
	switch verb {
	case "show":
		if len(rest) != 0 {
			return qaUsageError(errw)
		}
		return qaShow(out, errw, v, id)
	case "cases":
		if len(rest) != 0 {
			return qaUsageError(errw)
		}
		status := ""
		switch {
		case *failed:
			status = "failed"
		case *passed:
			status = "passed"
		case *skipped:
			status = "skipped"
		case *fresh:
			status = "new"
		}
		return qaCases(out, errw, v, id, serve.QACaseFilter{Status: status, Group: *group, Query: *query})
	case "case":
		if len(rest) != 1 {
			return qaUsageError(errw)
		}
		return qaCase(out, errw, v, id, rest[0])
	case "rerun":
		return qaRerun(out, errw, v, id, rest, *failed)
	case "accept":
		if len(rest) != 0 {
			return qaUsageError(errw)
		}
		return qaAccept(out, errw, v, id, *reason)
	case "log":
		if len(rest) != 1 {
			return qaUsageError(errw)
		}
		from, to, ok := parseRange(*lines)
		if !ok {
			return qaUsageError(errw)
		}
		return qaLog(out, errw, v, id, rest[0], serve.LogQuery{From: from, To: to, Q: *grep, FirstError: *first})
	}
	return qaUsageError(errw)
}

// parseRange reads "from:to"; an empty text is no range.
func parseRange(text string) (from, to int, ok bool) {
	if text == "" {
		return 0, 0, true
	}
	a, b, found := strings.Cut(text, ":")
	if !found {
		return 0, 0, false
	}
	var err error
	if from, err = strconv.Atoi(a); err != nil || from < 1 {
		return 0, 0, false
	}
	if to, err = strconv.Atoi(b); err != nil || to < from {
		return 0, 0, false
	}
	return from, to, true
}

func seconds(ms int64) string {
	if ms <= 0 {
		return "-"
	}
	return fmt.Sprintf("%.1fs", float64(ms)/1000)
}

func qaShow(out, errw io.Writer, v *verbFlags, id string) int {
	view, err := serve.ReadQA(v.spans, id)
	if err != nil {
		return qaFailure(out, errw, v.asJSON, err)
	}
	if v.asJSON {
		if code := emit(out, errw, view); code != 0 {
			return code
		}
		return view.Exit
	}
	fmt.Fprintf(out, "%s\n", view.Headline)
	tw := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "GATE\tVERDICT\tEXIT\tTIME\tEVIDENCE")
	for _, g := range view.Gates {
		exit, evidence := strconv.Itoa(g.ExitCode), "run by span"
		if !g.Ran {
			exit, evidence = "-", g.Detail
		}
		if g.Counts != nil {
			evidence += fmt.Sprintf(" · %d passed %d failed %d skipped", g.Counts.Passed, g.Counts.Failed, g.Counts.Skipped)
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", g.Name, g.Verdict, exit, seconds(g.DurationMS), evidence)
	}
	if err := tw.Flush(); err != nil {
		fmt.Fprintln(errw, err)
		return 1
	}
	if view.Reason != "" {
		fmt.Fprintf(out, "reason: %s\n", view.Reason)
	}
	if view.Accepted != nil {
		fmt.Fprintf(out, "accepted: %s\n", view.Accepted.Reason)
	}
	fmt.Fprintf(out, "exit %d: %s\n", view.Exit, map[int]string{0: "ok", 10: "a gate failed", 11: "no proof"}[view.Exit])
	return view.Exit
}

func qaCases(out, errw io.Writer, v *verbFlags, id string, filter serve.QACaseFilter) int {
	body, err := serve.ReadQACases(v.spans, id, filter)
	if err != nil {
		return qaFailure(out, errw, v.asJSON, err)
	}
	if v.asJSON {
		return emit(out, errw, body)
	}
	if !body.Known {
		fmt.Fprintln(out, body.Note)
		return 0
	}
	tw := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "FILE\tRESULT\tCASE\tLINE\tTIME\tNEW")
	for _, c := range body.Cases {
		file, line, fresh := c.File, "-", "-"
		if file == "" {
			file = "-"
		}
		if c.Line > 0 {
			line = strconv.Itoa(c.Line)
		}
		if c.New {
			fresh = "new"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%.2fs\t%s\n", file, strings.ToUpper(c.Result), c.Name, line, float64(c.ElapsedMS)/1000, fresh)
	}
	if err := tw.Flush(); err != nil {
		fmt.Fprintln(errw, err)
		return 1
	}
	return 0
}

func qaCase(out, errw io.Writer, v *verbFlags, id, name string) int {
	c, err := serve.ReadQACase(v.spans, id, name)
	if err != nil {
		return qaFailure(out, errw, v.asJSON, err)
	}
	if v.asJSON {
		return emit(out, errw, c)
	}
	fmt.Fprintf(out, "case: %s\nresult: %s\n", c.Name, c.Result)
	if c.Path != "" {
		fmt.Fprintf(out, "where: %s:%d\n", c.Path, c.Line)
	}
	if c.Message != "" {
		fmt.Fprintf(out, "assertion: %s\n", c.Message)
	}
	fmt.Fprintf(out, "time: %.2fs\nruns: %d\n", float64(c.ElapsedMS)/1000, len(c.History))
	for _, line := range c.Output {
		fmt.Fprintf(out, "  %s\n", line)
	}
	return 0
}

// qaRerun runs cases again, the page's "Rerun failed cases". It exits 10 when a case still
// fails, so a script can branch on it.
func qaRerun(out, errw io.Writer, v *verbFlags, id string, names []string, failed bool) int {
	if failed {
		body, err := serve.ReadQACases(v.spans, id, serve.QACaseFilter{Status: "failed"})
		if err != nil {
			return qaFailure(out, errw, v.asJSON, err)
		}
		for _, c := range body.Cases {
			names = append(names, c.Name)
		}
	}
	if len(names) == 0 {
		fmt.Fprintln(errw, "no case to rerun: name the cases or pass --failed with a failing case")
		return 2
	}
	body, err := serve.RerunQA(context.Background(), v.spans, id, names)
	if err != nil {
		return qaFailure(out, errw, v.asJSON, err)
	}
	if v.asJSON {
		if code := emit(out, errw, body); code != 0 {
			return code
		}
	} else {
		for _, c := range body.Cases {
			fmt.Fprintf(out, "%s %s\n", strings.ToUpper(c.Result), c.Name)
		}
	}
	for _, c := range body.Cases {
		if c.Result == "fail" {
			return serve.QAExitFailed
		}
	}
	return 0
}

func qaAccept(out, errw io.Writer, v *verbFlags, id, reason string) int {
	accepted, err := serve.AcceptQA(v.spans, id, reason)
	if err != nil {
		return qaFailure(out, errw, v.asJSON, err)
	}
	if v.asJSON {
		return emit(out, errw, accepted)
	}
	fmt.Fprintf(out, "%s accepted without proof at %s: %s\n", id, accepted.Gate, accepted.Reason)
	return 0
}

// qaLog prints a gate's log with the line numbers of the whole log. It exits 13 when the start of
// the log was cut, after printing what is kept.
func qaLog(out, errw io.Writer, v *verbFlags, id, gate string, q serve.LogQuery) int {
	view, err := serve.ReadGateLog(v.spans, id, gate, q)
	if err != nil {
		return qaFailure(out, errw, v.asJSON, err)
	}
	if v.asJSON {
		if code := emit(out, errw, view); code != 0 {
			return code
		}
	} else {
		for _, line := range view.Lines {
			fmt.Fprintf(out, "%d: %s\n", line.N, line.Text)
		}
		if q.Q != "" {
			fmt.Fprintf(out, "%d matches\n", len(view.Matches))
		}
		fmt.Fprintf(out, "lines %d of %d (%s)\n", view.Kept, view.TotalLines, map[bool]string{true: "cut", false: "complete"}[view.Truncated])
	}
	if view.Truncated {
		return qaExitLogCut
	}
	return 0
}

func transcriptMain(args []string, out, errw io.Writer) int {
	fs, v := verbSet("transcript", errw, false)
	stage := fs.String("stage", "", "the stage to read; the last one that printed by default")
	from := fs.Int("from", 0, "the first turn to print, from 0")
	limit := fs.Int("limit", 0, "how many turns to print; 10 by default")
	pos, ok := parseVerb(fs, args)
	if !ok || len(pos) != 1 {
		return qaUsageError(errw)
	}
	view, err := serve.ReadTranscript(v.spans, pos[0], *stage, *from, *limit)
	if err != nil {
		return qaFailure(out, errw, v.asJSON, err)
	}
	if v.asJSON {
		return emit(out, errw, view)
	}
	fmt.Fprintf(out, "stage %s · %d turns · %d lines · %d hidden values\n", view.Stage, view.Total, view.Lines, view.Redacted)
	for _, turn := range view.Turns {
		fmt.Fprintf(out, "%d  %s\n", turn.N, strings.ReplaceAll(turn.Text, "\n", "\n   "))
	}
	if end := view.From + len(view.Turns); end < view.Total {
		fmt.Fprintf(out, "turns %d to %d of %d; --from %d reads on\n", view.From+1, end, view.Total, end)
	}
	return 0
}
