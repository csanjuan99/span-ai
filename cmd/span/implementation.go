package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"span/internal/implementation"
)

// implementationUsage is what `span implementation` prints when called without a known mode.
const implementationUsage = `usage: ./span implementation <run> --files [--json]
       ./span implementation <run> --diff <path> [--commit <sha>] [--force] [--json]
       ./span implementation <run> --viewed <path> [--unview] [--json]
       ./span implementation <run> --commits [--json]
       ./span implementation <run> --commit <sha> [--json]
       ./span implementation <run> --reasoning [--json]
       ./span implementation <run> --check D1 D3 ... [--json]
       ./span implementation <run> --audit [--verify] [--json]
       (all take --spans .span/runs)
exit codes: 0 ok, 1 failed, 2 usage, 3 assumed decisions still unchecked, 4 not found, 12 audit chain broken`

// Exit codes the implementation views add to the usual 0, 1 and 2.
const (
	exitUnchecked   = 3
	exitNotFound    = 4
	exitChainBroken = 12
)

func implementationCmd(args []string) int {
	return implementationMain(args, os.Stdout, os.Stderr)
}

// implementationMain reads what the Implementation tab reads: the files, diffs, commits,
// reasoning and audit chain of a run. It calls the functions the page's routes call, so a
// remote operator sees exactly what the page shows. Marking a file viewed and checking
// assumed decisions are the two writes, and neither changes code.
func implementationMain(args []string, out, errw io.Writer) int {
	fs, v := verbSet("implementation", errw, false)
	files := fs.Bool("files", false, "list the changed files with status, counts and viewed marks")
	diff := fs.String("diff", "", "print the diff of one changed file")
	commit := fs.String("commit", "", "with --diff, limit it to one commit; alone, print that commit's diff")
	force := fs.Bool("force", false, "with --diff, print generated, binary and very large files too")
	viewed := fs.String("viewed", "", "mark one changed file viewed")
	unview := fs.Bool("unview", false, "with --viewed, take the mark off")
	commits := fs.Bool("commits", false, "list the commits of the branch, newest first")
	reasoning := fs.Bool("reasoning", false, "show the decisions the agent recorded and which are checked")
	check := fs.Bool("check", false, "mark the assumed decisions named after the run as checked")
	audit := fs.Bool("audit", false, "print the audit log with its hash chain")
	verify := fs.Bool("verify", false, "with --audit, only verify the chain")
	pos, ok := parseVerb(fs, args)
	if !ok || len(pos) < 1 {
		fmt.Fprintln(errw, implementationUsage)
		return 2
	}
	id, rest := pos[0], pos[1:]
	ctx := context.Background()
	modes := 0
	for _, on := range []bool{*files, *diff != "", *viewed != "", *commits, *reasoning, *check, *audit, *commit != "" && *diff == ""} {
		if on {
			modes++
		}
	}
	if modes != 1 || (len(rest) > 0 && !*check) || (*check && len(rest) == 0) {
		fmt.Fprintln(errw, implementationUsage)
		return 2
	}
	switch {
	case *files:
		list, err := implementation.ChangedFiles(ctx, v.spans, id)
		return finish(out, errw, v, err, map[string]any{"files": list}, func() {
			for _, f := range list {
				fmt.Fprintf(out, "%s %-40s %s%s\n", f.Status, f.Path, fileCounts(f.Added, f.Removed, f.Binary), fileNotes(f.Viewed, f.Generated, f.Large, f.Binary, f.From))
			}
		})
	case *diff != "":
		view, err := implementation.RunFileDiff(ctx, v.spans, id, *diff, *commit, *force)
		return finish(out, errw, v, err, view, func() {
			if view.Patch == "" {
				fmt.Fprintf(out, "%s: %s; not printed (--force prints it)\n", view.Path, fileNotes(false, view.Generated, view.Large, view.Binary, ""))
				return
			}
			fmt.Fprint(out, view.Patch)
		})
	case *viewed != "":
		err := implementation.SetViewed(ctx, v.spans, id, *viewed, !*unview)
		return finish(out, errw, v, err, map[string]any{"path": *viewed, "viewed": !*unview}, func() {
			fmt.Fprintf(out, "%s: viewed=%t\n", *viewed, !*unview)
		})
	case *commits:
		list, err := implementation.RunCommits(ctx, v.spans, id)
		return finish(out, errw, v, err, map[string]any{"commits": list}, func() {
			fmt.Fprintln(out, "SHA      TIME  FILES LINES        SUBJECT")
			for _, c := range list {
				lines := fmt.Sprintf("+%d -%d", c.Added, c.Removed)
				if c.RenameOnly {
					lines = "renamed"
				}
				fmt.Fprintf(out, "%-8s %-5s %-5d %-12s %s\n", c.SHA[:7], clock(c.Time), c.Files, lines, c.Subject)
			}
		})
	case *commit != "":
		view, err := implementation.RunCommitDiff(ctx, v.spans, id, *commit)
		return finish(out, errw, v, err, view, func() { fmt.Fprint(out, view.Patch) })
	case *reasoning:
		view, err := implementation.RunReasoning(v.spans, id)
		code := finish(out, errw, v, err, view, func() { printReasoning(out, view) })
		if code == 0 && view.Unchecked > 0 {
			fmt.Fprintf(errw, "%d assumed decisions are not checked\n", view.Unchecked)
			return exitUnchecked
		}
		return code
	case *check:
		err := implementation.CheckDecisions(v.spans, id, rest)
		return finish(out, errw, v, err, map[string]any{"checked": rest}, func() {
			fmt.Fprintf(out, "checked %s\n", strings.Join(rest, ", "))
		})
	}
	return auditView(out, errw, v, id, *verify)
}

// auditView prints the audit log, or only its verdict, and exits 12 when the chain is broken.
func auditView(out, errw io.Writer, v *verbFlags, id string, verifyOnly bool) int {
	view, err := implementation.RunAudit(v.spans, id)
	var payload any = view
	if verifyOnly {
		payload = map[string]any{"intact": view.Intact, "brokenAt": view.BrokenAt, "entries": len(view.Entries), "head": view.Head}
	}
	code := finish(out, errw, v, err, payload, func() {
		if !verifyOnly {
			fmt.Fprintln(out, "#  TIME  WHO          WHAT                                  HASH      PREV")
			for _, e := range view.Entries {
				fmt.Fprintf(out, "%-2d %-5s %-12s %-37s %-9s %s\n", e.Seq, clock(e.Time.Format(time.RFC3339)), e.Actor, e.What, short(e.Hash), dash(short(e.Prev)))
			}
		}
		if view.Intact {
			fmt.Fprintf(out, "chain: intact, %d of %d entries\n", len(view.Entries), len(view.Entries))
		} else {
			fmt.Fprintf(out, "chain: broken at entry %d of %d; entries %d to %d cannot be trusted\n", view.BrokenAt, len(view.Entries), view.BrokenAt, len(view.Entries))
		}
	})
	if code == 0 && !view.Intact {
		return exitChainBroken
	}
	return code
}

// finish prints a view as JSON or as text and turns an error into its exit code.
func finish(out, errw io.Writer, v *verbFlags, err error, payload any, text func()) int {
	if err != nil {
		return failWith(errw, v.asJSON, err)
	}
	if v.asJSON {
		return emit(out, errw, payload)
	}
	text()
	return 0
}

// failWith reports an error the way the page's routes name it: not found is exit 4, anything
// else exit 1. With --json the error is {error, message} on stderr.
func failWith(errw io.Writer, asJSON bool, err error) int {
	code, exit := "invalid", 1
	var request *implementation.RequestError
	if errors.As(err, &request) {
		code = request.Code
		if code == "not_found" {
			exit = exitNotFound
		}
	} else if errors.Is(err, os.ErrNotExist) {
		code, exit = "not_found", exitNotFound
	}
	if asJSON {
		raw, marshalErr := json.Marshal(map[string]string{"error": code, "message": err.Error()})
		if marshalErr == nil {
			fmt.Fprintln(errw, string(raw))
			return exit
		}
	}
	fmt.Fprintln(errw, err)
	return exit
}

func printReasoning(out io.Writer, view implementation.ReasoningView) {
	fmt.Fprintln(out, "ID  FROM     ASSUMED CHECKED DECISION")
	for _, d := range view.Decisions {
		from, checked := d.Source, "-"
		if from == "" {
			from = "none"
		}
		if d.Assumed {
			checked = "no"
			if d.Checked {
				checked = "yes"
			}
		}
		fmt.Fprintf(out, "%-3s %-8s %-7s %-7s %s\n", d.ID, from, yesNo(d.Assumed), checked, d.What)
	}
	if len(view.Decisions) == 0 {
		fmt.Fprintln(out, "no decisions recorded")
	}
}

func fileCounts(added, removed int, binary bool) string {
	if binary {
		return "binary"
	}
	return fmt.Sprintf("+%d -%d", added, removed)
}

func fileNotes(viewed, generated, large, binary bool, from string) string {
	var notes []string
	if from != "" {
		notes = append(notes, "renamed from "+from)
	}
	for name, on := range map[string]bool{"generated": generated, "large": large} {
		if on {
			notes = append(notes, name)
		}
	}
	if binary {
		notes = append(notes, "binary")
	}
	if viewed {
		notes = append(notes, "viewed")
	}
	if len(notes) == 0 {
		return ""
	}
	return "  [" + strings.Join(notes, ", ") + "]"
}

func yesNo(on bool) string {
	if on {
		return "yes"
	}
	return "no"
}

func short(hash string) string {
	if len(hash) > 4 {
		return hash[:4]
	}
	return hash
}

func dash(text string) string {
	if text == "" {
		return "-"
	}
	return text
}

// clock is the local HH:MM of an RFC 3339 time, or the text itself when it does not parse.
func clock(rfc string) string {
	t, err := time.Parse(time.RFC3339, rfc)
	if err != nil {
		return rfc
	}
	return t.Local().Format("15:04")
}
