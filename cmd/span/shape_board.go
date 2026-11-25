package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"span/internal/shape"
)

// shapeBoardUsage lists the verbs for many sessions at once. They call the same shape
// functions as GET /shapes?group=status, GET /shapes/activity and POST /shapes/bulk.
const shapeBoardUsage = `  board    [--repo r] [--archived include|only] [--limit n]
  activity [--who agents|you|everyone] [--since RFC3339] [--limit n] [--follow]
  bulk     assume|seal|archive|unarchive <id|SES-ref>... [--ready] [--dry-run] [--yes]`

// boardPoll is how often --follow looks for new steps.
const boardPoll = time.Second

// boardVerbs are the verbs of shapeBoardUsage.
var boardVerbs = map[string]bool{"board": true, "activity": true, "bulk": true}

// boardOpts are the flags of the board verbs.
type boardOpts struct {
	archived       string
	limit          int
	who, since     string
	follow         bool
	ready, dry, ok bool
}

func (o *shapeOpts) boardFlags(fs *flag.FlagSet, verb string) {
	switch verb {
	case "board":
		fs.StringVar(&o.filter.Repo, "repo", "", "only the sessions of this repo")
		fs.StringVar(&o.brd.archived, "archived", "", "include or only")
		fs.IntVar(&o.brd.limit, "limit", 0, "most cards per column; 0 shows all")
	case "activity":
		fs.StringVar(&o.brd.who, "who", "", "agents (default), you or everyone")
		fs.StringVar(&o.brd.since, "since", "", "only steps after this time, RFC 3339")
		fs.IntVar(&o.brd.limit, "limit", 0, "most steps; 0 means 30")
		fs.BoolVar(&o.brd.follow, "follow", false, "keep running and print new steps until interrupted")
	case "bulk":
		fs.BoolVar(&o.brd.ready, "ready", false, "select every session that is ready (for seal)")
		fs.BoolVar(&o.brd.dry, "dry-run", false, "say what would change and change nothing")
		fs.BoolVar(&o.brd.ok, "yes", false, "confirm the change; without it, a run only says what would change")
	}
}

// board runs one board verb.
func (o *shapeOpts) boardVerb(ctx context.Context, verb string) int {
	switch verb {
	case "board":
		return o.showBoard()
	case "activity":
		return o.activity(ctx)
	}
	return o.bulk(ctx)
}

func (o *shapeOpts) showBoard() int {
	f := o.filter
	f.Archived = o.brd.archived
	view, err := shape.Board(o.ws, f, o.brd.limit)
	if err != nil {
		return o.fail(err, false)
	}
	if o.json {
		return o.encode(view)
	}
	fmt.Fprintf(o.out, "%d in flight · %d questions need you\n", view.InFlight, view.NeedYou)
	for _, col := range view.Columns {
		fmt.Fprintf(o.out, "\n%s (%d)\n", strings.ToUpper(col.Name), col.Count)
		for _, c := range col.Cards {
			fmt.Fprintf(o.out, "  %-11s %s\n", c.Ref, c.Title)
			fmt.Fprintf(o.out, "  %-11s %s\n", "", cardLine(c))
		}
		if col.Hidden > 0 {
			fmt.Fprintf(o.out, "  … %d more\n", col.Hidden)
		}
	}
	return 0
}

func cardLine(c shape.BoardCard) string {
	agent := c.Agent
	if agent == "" {
		agent = "auto"
	}
	model := c.Model
	if model == "" {
		model = "auto"
	}
	parts := []string{agent + " · " + model}
	switch c.Status {
	case shape.StatusInterviewing:
		parts = append(parts, fmt.Sprintf("%d open", c.OpenQuestions))
	case shape.StatusReady:
		parts = append(parts, fmt.Sprintf("%d assumed · %d open", c.Assumed, c.OpenQuestions))
	case shape.StatusSplit:
		parts = append(parts, fmt.Sprintf("%d stories proposed", c.Stories))
	case shape.StatusSealed:
		if c.JiraKey != "" {
			parts = append(parts, c.JiraKey)
		}
	}
	if c.Doing != "" {
		parts = append(parts, c.Doing)
	}
	return strings.Join(parts, " · ")
}

func (o *shapeOpts) activity(ctx context.Context) int {
	var since time.Time
	if o.brd.since != "" {
		var err error
		if since, err = time.Parse(time.RFC3339Nano, o.brd.since); err != nil {
			fmt.Fprintln(o.err, "--since must be a time like 2026-11-25T09:30:00Z")
			return shapeExitUsage
		}
	}
	// One JSON array, newest first, the body GET /shapes/activity returns. --follow keeps
	// printing each new step on its own line because the route then streams events.
	if o.json && !o.brd.follow {
		feed, err := shape.RecentActivity(o.ws, o.brd.who, since, o.brd.limit)
		if err != nil {
			return o.fail(err, false)
		}
		return o.encode(feed)
	}
	if !o.json {
		fmt.Fprintln(o.out, "TIME      SESSION      ACTOR        DOING")
	}
	for {
		feed, err := shape.RecentActivity(o.ws, o.brd.who, since, o.brd.limit)
		if err != nil {
			return o.fail(err, false)
		}
		for i := len(feed) - 1; i >= 0; i-- {
			a := feed[i]
			if o.json {
				if code := o.encode(a); code != 0 {
					return code
				}
			} else {
				fmt.Fprintf(o.out, "%-9s %-12s %-12s %s\n", a.At.Local().Format("15:04:05"), a.Ref, a.Actor, a.Text)
			}
			if a.At.After(since) {
				since = a.At
			}
		}
		if !o.brd.follow {
			return 0
		}
		select {
		case <-ctx.Done():
			return 0
		case <-time.After(boardPoll):
		}
	}
}

// resolveSessions turns ids and SES- refs into session ids.
func (o *shapeOpts) resolveSessions(names []string) ([]string, error) {
	all, err := shape.List(o.ws)
	if all == nil {
		return nil, err
	}
	var out []string
	for _, name := range names {
		found := ""
		for _, s := range all {
			if s.ID == name || strings.EqualFold(s.Ref(), name) {
				found = s.ID
				break
			}
		}
		if found == "" {
			found = name
		}
		out = append(out, found)
	}
	return out, nil
}

func (o *shapeOpts) bulk(ctx context.Context) int {
	if len(o.args) == 0 {
		fmt.Fprintln(o.err, shapeUsage)
		return shapeExitUsage
	}
	action, names := o.args[0], o.args[1:]
	if o.brd.ready {
		all, err := shape.List(o.ws)
		if all == nil {
			return o.fail(err, false)
		}
		for _, s := range all {
			if s.Status == shape.StatusReady && !s.Archived {
				names = append(names, s.ID)
			}
		}
	}
	ids, err := o.resolveSessions(names)
	if err != nil {
		return o.fail(err, false)
	}
	dry := o.brd.dry || !o.brd.ok
	engineFor := func(s *shape.Session) *shape.Engine { return &shape.Engine{Run: shapeAgents(o.ws, s.Agent, s.Model)} }
	res, err := shape.Bulk(ctx, o.ws, action, ids, dry, engineFor)
	if err != nil {
		return o.fail(err, false)
	}
	if o.json {
		if code := o.encode(res); code != 0 {
			return code
		}
	} else {
		o.printBulk(res, o.brd.dry, o.brd.ok)
	}
	if res.Failed > 0 {
		return shapeExitFailed
	}
	return 0
}

func (o *shapeOpts) printBulk(res shape.BulkResult, asked, confirmed bool) {
	if res.DryRun {
		why := "dry run"
		if !asked && !confirmed {
			why = "nothing changed: add --yes to apply"
		}
		fmt.Fprintf(o.out, "%s (%s)\n", res.Action, why)
	}
	for _, item := range res.Items {
		switch {
		case item.Code != "":
			fmt.Fprintf(o.out, "  %-11s failed: %s\n", orID(item), item.Message)
		case item.Skipped:
			fmt.Fprintf(o.out, "  %-11s skipped: %s\n", orID(item), item.Message)
		default:
			fmt.Fprintf(o.out, "  %-11s ok\n", orID(item))
			for _, line := range item.Changed {
				fmt.Fprintf(o.out, "              %s\n", line)
			}
		}
	}
	fmt.Fprintf(o.out, "%d done · %d skipped · %d failed\n", res.Done, res.Skipped, res.Failed)
}

func orID(item shape.BulkItem) string {
	if item.Ref != "" {
		return item.Ref
	}
	return item.ID
}

// followContext is ctx that ends when the person interrupts the command.
func followContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(ctx, os.Interrupt)
}
