package main

import (
	"flag"
	"fmt"
	"strings"

	"span/internal/shape"
)

// shapeHistoryUsage lists the verbs for the history of one session. They call the same shape
// functions as GET /shapes/{id}/history and POST /shapes/{id}/history/{step}/undo.
const shapeHistoryUsage = `  history  <id|SES-ref> [--actor you|agents] [--kind answers|assumptions|context|split]
  undo     <id|SES-ref> <step> [--cascade] [--plan] [--yes]`

// historyVerbs are the verbs of shapeHistoryUsage.
var historyVerbs = map[string]bool{"history": true, "undo": true}

// historyOpts are the flags of the history verbs.
type historyOpts struct {
	actor, kind string
	cascade     bool
	plan        bool
}

func (o *shapeOpts) historyFlags(fs *flag.FlagSet, verb string) {
	if verb == "history" {
		fs.StringVar(&o.hist.actor, "actor", "", "everyone (default), you or agents")
		fs.StringVar(&o.hist.kind, "kind", "", "answers, assumptions, context or split")
		return
	}
	fs.BoolVar(&o.hist.cascade, "cascade", false, "also undo the steps that depend on this one")
	fs.BoolVar(&o.hist.plan, "plan", false, "say what undoing would change and change nothing")
	fs.BoolVar(&o.yes, "yes", false, "confirm the undo")
}

func (o *shapeOpts) historyVerb(verb string) int {
	if len(o.args) == 0 || (verb == "undo" && len(o.args) != 2) || (verb == "history" && len(o.args) != 1) {
		fmt.Fprintln(o.err, shapeUsage)
		return shapeExitUsage
	}
	ids, err := o.resolveSessions(o.args[:1])
	if err != nil {
		return o.fail(err, false)
	}
	if verb == "history" {
		return o.showHistory(ids[0])
	}
	return o.undoStep(ids[0], o.args[1])
}

func (o *shapeOpts) showHistory(id string) int {
	view, err := shape.History(o.ws, id, shape.HistoryFilter{Actor: o.hist.actor, Kind: o.hist.kind})
	if err != nil {
		return o.fail(err, false)
	}
	if o.json {
		return o.encode(view)
	}
	fmt.Fprintf(o.out, "%s %s · %d steps (%d you, %d agents)\n", view.Ref, view.Status, view.Counts.Everyone, view.Counts.You, view.Counts.Agents)
	fmt.Fprintln(o.out, "ID    TIME   WHO          WHAT                                           UNDO")
	for _, e := range view.Events {
		fmt.Fprintf(o.out, "%-5s %-6s %-12s %-46s %s\n", e.ID, e.At.Local().Format("15:04"), e.Actor, e.Summary, undoWord(e))
	}
	return 0
}

func undoWord(e shape.EventView) string {
	switch {
	case e.UndoneBy != "":
		return "undone by " + e.UndoneBy
	case !e.Undoable:
		return "no: " + e.Why
	case len(e.Dependents) > 0:
		return fmt.Sprintf("yes, +%d", len(e.Dependents))
	}
	return "yes"
}

func (o *shapeOpts) undoStep(id, step string) int {
	plan, err := shape.PlanUndo(o.ws, id, step)
	if err != nil {
		return o.fail(err, false)
	}
	if o.hist.plan || !o.yes {
		if o.json {
			return o.encode(plan)
		}
		fmt.Fprintf(o.out, "undo %s: %s\n  now:   %s\n  after: %s\n", plan.Target.ID, plan.Target.Summary, plan.Now, plan.After)
		for _, c := range plan.Changes {
			fmt.Fprintf(o.out, "  - %s\n", c)
		}
		if len(plan.Dependents) > 0 {
			ids := make([]string, 0, len(plan.Dependents))
			for _, d := range plan.Dependents {
				ids = append(ids, d.ID)
			}
			fmt.Fprintf(o.out, "  depends on it: %s (add --cascade)\n", strings.Join(ids, ", "))
		}
		if !o.hist.plan {
			fmt.Fprintln(o.err, "nothing changed: add --yes to undo")
			return shapeExitUsage
		}
		return 0
	}
	res, err := (&shape.Engine{}).Undo(o.ws, id, step, o.hist.cascade)
	if err != nil {
		return o.fail(err, false)
	}
	if o.json {
		return o.encode(map[string]any{"step": res.Step, "undone": res.Undone, "session": res.Session.View()})
	}
	fmt.Fprintf(o.out, "%s: undid %s\n", res.Step.ID, strings.Join(res.Undone, ", "))
	return 0
}
