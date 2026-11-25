package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"span/internal/engine"
)

// conflictUsage is what `span conflict` prints when it is called without a known verb.
const conflictUsage = `usage: ./span conflict show <run> [--options] [--json]
       ./span conflict choose <run> <group>=<option>... | <group> <option> [--no-resume] [--json]
       ./span conflict hint <run> --text "..." [--no-resume] [--json]
       ./span conflict abort <run> [--no-resume] [--json]
       (all take --spans .span/runs)`

func conflictCmd(args []string) int {
	return conflictMain(args, os.Stdout, os.Stderr)
}

// conflictMain runs one `span conflict` verb. Show reads what the page reads: the open merge
// conflict of a run, with each option's diff computed by Span. Choose, hint and abort call
// the engine functions the page's routes call, then continue the run as `span resume` does.
func conflictMain(args []string, out, errw io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(errw, conflictUsage)
		return 2
	}
	verb := args[0]
	switch verb {
	case "show", "choose", "hint", "abort":
	default:
		fmt.Fprintln(errw, conflictUsage)
		return 2
	}
	fs, v := verbSet("conflict "+verb, errw, verb != "show")
	options := fs.Bool("options", false, "show: list every option with the diff it would write")
	text := fs.String("text", "", "hint: the words for the agent")
	pos, ok := parseVerb(fs, args[1:])
	if !ok || len(pos) < 1 {
		fmt.Fprintln(errw, conflictUsage)
		return 2
	}
	id, rest := pos[0], pos[1:]
	switch verb {
	case "show":
		if len(rest) != 0 {
			fmt.Fprintln(errw, conflictUsage)
			return 2
		}
		return showConflict(out, errw, v, id, *options)
	case "choose":
		choices, err := conflictChoices(rest)
		if err != nil {
			fmt.Fprintln(errw, err)
			return 2
		}
		return act(out, errw, v, id, verb, func() error { return engine.ChooseConflict(v.spans, id, choices) })
	case "hint":
		if len(rest) != 0 || *text == "" {
			fmt.Fprintln(errw, conflictUsage)
			return 2
		}
		return act(out, errw, v, id, verb, func() error { return engine.HintConflict(v.spans, id, *text) })
	}
	if len(rest) != 0 {
		fmt.Fprintln(errw, conflictUsage)
		return 2
	}
	return act(out, errw, v, id, verb, func() error { return engine.AbortConflict(v.spans, id) })
}

// conflictChoices reads `group=option` pairs, or the two-word form `group option` for a
// conflict with one group. Every group is chosen at once, as the page does.
func conflictChoices(args []string) (map[string]string, error) {
	choices := map[string]string{}
	if len(args) == 2 && !strings.Contains(args[0], "=") && !strings.Contains(args[1], "=") {
		return map[string]string{args[0]: args[1]}, nil
	}
	for _, arg := range args {
		group, option, found := strings.Cut(arg, "=")
		if !found || group == "" || option == "" {
			return nil, errors.New("choose names each group as <group>=<option>")
		}
		if _, seen := choices[group]; seen {
			return nil, errors.New("group " + group + " is chosen twice")
		}
		choices[group] = option
	}
	if len(choices) == 0 {
		return nil, errors.New("choose names each group as <group>=<option>")
	}
	return choices, nil
}

// act records one conflict decision, then lets the run go on. The engine refuses a decision
// the conflict's state cannot take, which exits 1 with its reason.
func act(out, errw io.Writer, v *verbFlags, id, verb string, record func() error) int {
	if _, err := engine.Load(v.spans, id); err != nil {
		fmt.Fprintln(errw, err)
		return 1
	}
	if err := record(); err != nil {
		fmt.Fprintln(errw, err)
		return 1
	}
	return answered(out, errw, v, id, "conflict "+verb, nil)
}

func showConflict(out, errw io.Writer, v *verbFlags, id string, options bool) int {
	snap, err := engine.Load(v.spans, id)
	if err != nil {
		fmt.Fprintln(errw, err)
		return 1
	}
	conflict := snap.Cursor.Conflict
	if conflict == nil {
		fmt.Fprintln(errw, engine.ErrNoConflict)
		return 1
	}
	view := *conflict
	if !options {
		view.Groups = withoutDiffs(view.Groups)
	}
	if v.asJSON {
		return emit(out, errw, view)
	}
	printConflict(out, id, view, options)
	return 0
}

// withoutDiffs copies the groups with every option's diff left out.
func withoutDiffs(groups []engine.ConflictGroup) []engine.ConflictGroup {
	plain := make([]engine.ConflictGroup, len(groups))
	for i, group := range groups {
		plain[i] = group
		plain[i].Options = make([]engine.ConflictOption, len(group.Options))
		for j, option := range group.Options {
			option.Diff = ""
			plain[i].Options[j] = option
		}
	}
	return plain
}

func printConflict(out io.Writer, id string, c engine.Conflict, options bool) {
	fmt.Fprintf(out, "run %s\nbase %s\nstate %s\n", id, c.Base, c.State)
	if c.Note != "" {
		fmt.Fprintf(out, "note %s\n", c.Note)
	}
	for i, group := range c.Groups {
		fmt.Fprintf(out, "\ngroup %d of %d  %s  (%s)\n  %s\n", i+1, len(c.Groups), group.ID, group.Kind, strings.Join(group.Files, ", "))
		fmt.Fprintf(out, "  %s\n", group.Summary)
		if !options {
			continue
		}
		for j, option := range group.Options {
			mark := ""
			if option.ID == group.Chosen {
				mark = "  [chosen]"
			}
			fmt.Fprintf(out, "\n  %c  %s  %s  (%s risk, %s)%s\n", 'A'+j, option.ID, option.Title, option.Risk, option.Action, mark)
			if option.Summary != "" {
				fmt.Fprintf(out, "     %s\n", option.Summary)
			}
			for _, line := range strings.Split(strings.TrimRight(option.Diff, "\n"), "\n") {
				if option.Diff != "" {
					fmt.Fprintf(out, "     | %s\n", line)
				}
			}
		}
	}
}
