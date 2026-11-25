package main

import (
	"flag"
	"fmt"
	"strings"

	"span/internal/shape"
)

// shapeVersionUsage lists the verbs that compare: two revisions of one session, and two to
// five sessions of one repo. Each calls the same shape function as the HTTP route named
// beside it in USAGE.md.
const shapeVersionUsage = `  log      <id>              (also: versions)
  diff     <id> <rev-a> <rev-b> [--format inline|side]
  restore  <id> <rev> [--new]
  compare  <id> <id>... [--dismiss]
  merge    <into> <from>... --yes [--against id,id] [--resolve C1=id,C2=id]`

// versionVerbs are the verbs of shapeVersionUsage.
var versionVerbs = map[string]bool{"log": true, "versions": true, "diff": true, "restore": true, "compare": true, "merge": true}

// versionOpts are the flags of the version verbs.
type versionOpts struct {
	dismiss, yes     bool
	asNew            bool
	against, resolve string
	format           string
}

func (o *shapeOpts) versionFlags(fs *flag.FlagSet, verb string) {
	switch verb {
	case "diff":
		fs.StringVar(&o.ver.format, "format", "inline", "how to print the story text: inline or side")
	case "restore":
		fs.BoolVar(&o.ver.asNew, "new", false, "start a new session from the revision instead of adding a revision to this one")
	case "compare":
		fs.BoolVar(&o.ver.dismiss, "dismiss", false, "record that these sessions stay separate")
	case "merge":
		fs.BoolVar(&o.ver.yes, "yes", false, "confirm the merge; without it nothing changes")
		fs.StringVar(&o.ver.against, "against", "", "sessions that were in the comparison you decided on, as id,id; they never change")
		fs.StringVar(&o.ver.resolve, "resolve", "", "the session whose answer wins each conflict, as C1=id,C2=id")
	}
}

// version runs one version verb.
func (o *shapeOpts) version(verb string) int {
	switch verb {
	case "compare":
		return o.compare()
	case "merge":
		return o.merge()
	}
	if verb == "log" {
		verb = "versions"
	}
	want := map[string]int{"versions": 1, "diff": 3, "restore": 2}[verb]
	if len(o.args) != want {
		fmt.Fprintln(o.err, shapeUsage)
		return shapeExitUsage
	}
	id := o.args[0]
	switch verb {
	case "versions":
		return o.versions(id)
	case "diff":
		out, err := shape.Diff(o.ws, id, o.args[1], o.args[2])
		if err != nil {
			return o.fail(err, false)
		}
		if o.json {
			return o.encode(out)
		}
		if o.ver.format != "inline" && o.ver.format != "side" {
			fmt.Fprintln(o.err, "--format must be inline or side")
			return shapeExitUsage
		}
		o.printDiff(out)
		return 0
	}
	if o.ver.asNew {
		child, err := shape.ForkRevision(o.ws, id, o.args[1])
		if err != nil {
			return o.fail(err, false)
		}
		if o.json {
			return o.encode(child.View())
		}
		fmt.Fprintf(o.out, "started %s from %s of %s\n", child.Ref(), o.args[1], id)
		return 0
	}
	s, err := shape.Restore(o.ws, id, o.args[1])
	if err != nil {
		return o.fail(err, false)
	}
	if o.json {
		return o.encode(s.View())
	}
	fmt.Fprintf(o.out, "restored %s into a new revision of %s\n", o.args[1], s.Ref())
	return 0
}

func (o *shapeOpts) versions(id string) int {
	list, err := shape.Revisions(o.ws, id)
	if err != nil {
		return o.fail(err, false)
	}
	if o.json {
		return o.encode(map[string]any{"revisions": list})
	}
	for _, r := range list {
		mark := " "
		if r.Current {
			mark = "*"
		}
		fmt.Fprintf(o.out, "%s %-4s %s  %-12s %s\n", mark, r.ID, r.At.Local().Format("15:04"), r.Actor, r.Label)
	}
	return 0
}

func tallyText(name string, t shape.Tally) string {
	var parts []string
	if t.Added > 0 {
		parts = append(parts, fmt.Sprintf("%d added", t.Added))
	}
	if t.Removed > 0 {
		parts = append(parts, fmt.Sprintf("%d removed", t.Removed))
	}
	if t.Changed > 0 {
		parts = append(parts, fmt.Sprintf("%d changed", t.Changed))
	}
	if len(parts) == 0 {
		parts = append(parts, "no change")
	}
	return name + " " + strings.Join(parts, ", ")
}

func (o *shapeOpts) printDiff(d shape.RevisionDiff) {
	fmt.Fprintf(o.out, "%s -> %s\n", d.From.ID, d.To.ID)
	fmt.Fprintln(o.out, tallyText("rules    ", d.Rules))
	fmt.Fprintln(o.out, tallyText("examples ", d.Examples))
	fmt.Fprintln(o.out, tallyText("questions", d.Questions))
	fmt.Fprintf(o.out, "story     +%d -%d lines\n", d.StoryAdded, d.StoryRemoved)
	marks := map[string]string{shape.ChangeAdded: "+", shape.ChangeRemoved: "-", shape.ChangeChanged: "~"}
	for _, it := range d.Items {
		fmt.Fprintf(o.out, "%s %s %s\n", marks[it.Change], it.Kind, it.ID)
		if it.Before != "" {
			fmt.Fprintf(o.out, "    %s: %s\n", d.From.ID, it.Before)
		}
		if it.After != "" {
			fmt.Fprintf(o.out, "    %s: %s\n", d.To.ID, it.After)
		}
	}
	for _, l := range d.Story {
		switch {
		case o.ver.format == "side" && l.Kind == shape.LineDel:
			fmt.Fprintf(o.out, "%4d - %s\n", l.Old, l.Text)
		case o.ver.format == "side" && l.Kind == shape.LineAdd:
			fmt.Fprintf(o.out, "     | %4d + %s\n", l.New, l.Text)
		case l.Kind == shape.LineDel:
			fmt.Fprintf(o.out, "- %s\n", l.Text)
		case l.Kind == shape.LineAdd:
			fmt.Fprintf(o.out, "+ %s\n", l.Text)
		default:
			fmt.Fprintf(o.out, "  %s\n", l.Text)
		}
	}
}
