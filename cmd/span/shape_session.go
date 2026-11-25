package main

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"span/internal/serve"
	"span/internal/shape"
)

// shapeSessionUsage lists the verbs that work on one session after it started. Each calls the
// same shape function as the HTTP route named beside it in USAGE.md.
const shapeSessionUsage = `  answer  <id> [--answer Q1=text]... [--assume Q1,Q2]
  context <id> [--accept P1,P2] [--skip P3] [--restore P3] [--preview]
  split   <id> (--split | --keep)
  seal    <id>
  source  <id> --ref path:line
  preview <id> --project KEY [--board N]
  jira    <id> --project KEY [--board N] --confirm
  story   <id>
  fork    <id>`

// shapeBacklog reads the Jira backlog of a workspace; tests replace it.
var shapeBacklog = func(workspace string) (shape.Backlog, error) { return serve.JiraBacklog(workspace) }

// sessionVerbs are the verbs of shapeSessionUsage.
var sessionVerbs = map[string]bool{
	"answer": true, "context": true, "split": true, "seal": true, "source": true,
	"preview": true, "jira": true, "story": true, "fork": true,
}

// sessionOpts are the flags of the session verbs.
type sessionOpts struct {
	answers                       repeated
	assume, accept, skip, restore string
	preview, doSplit, keep        bool
	ref, project                  string
	board                         int
	confirm                       bool
}

func (o *shapeOpts) sessionFlags(fs *flag.FlagSet, verb string) {
	switch verb {
	case "answer":
		fs.Var(&o.answers, "answer", "an answer as Q1=text; repeat for more")
		fs.StringVar(&o.assume, "assume", "", "questions to close with their assumption, as Q1,Q2")
	case "context":
		fs.StringVar(&o.accept, "accept", "", "notes to write to the product context, as P1,P2")
		fs.StringVar(&o.skip, "skip", "", "notes to drop")
		fs.StringVar(&o.restore, "restore", "", "skipped notes to bring back")
		fs.BoolVar(&o.preview, "preview", false, "show what --accept would add and write nothing")
	case "split":
		fs.BoolVar(&o.doSplit, "split", false, "replace the story with one session per proposed story")
		fs.BoolVar(&o.keep, "keep", false, "keep one story; the agent writes it")
	case "source":
		fs.StringVar(&o.ref, "ref", "", "path:line inside the session's repository")
	case "preview", "jira":
		fs.StringVar(&o.project, "project", "", "Jira project key")
		fs.IntVar(&o.board, "board", 0, "Kanban board id; 0 finds the project's only board")
		if verb == "jira" {
			fs.BoolVar(&o.confirm, "confirm", false, "create the issue; without it nothing is sent")
		}
	}
}

func splitList(raw string) []string {
	var out []string
	for _, item := range strings.Split(raw, ",") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}

func parseAnswers(raw repeated) ([]shape.Answer, error) {
	var out []shape.Answer
	for _, item := range raw {
		question, text, ok := strings.Cut(item, "=")
		if !ok || strings.TrimSpace(question) == "" {
			return nil, fmt.Errorf("answer %q must look like Q1=text", item)
		}
		out = append(out, shape.Answer{Question: strings.TrimSpace(question), Text: text})
	}
	return out, nil
}

// session runs one session verb on session id.
func (o *shapeOpts) session(ctx context.Context, verb, id string) int {
	switch verb {
	case "source":
		snippet, err := shape.SourceSnippet(o.ws, id, o.ref)
		if err != nil {
			return o.fail(err, false)
		}
		if o.json {
			return o.encode(snippet)
		}
		for _, l := range snippet.Lines {
			mark := " "
			if l.Hit {
				mark = ">"
			}
			fmt.Fprintf(o.out, "%s%5d  %s\n", mark, l.N, l.Text)
		}
		return 0
	case "preview":
		return o.issuePreview(ctx, id)
	case "story":
		s, err := shape.Load(o.ws, id)
		if err != nil {
			return o.fail(err, false)
		}
		if s.Status != shape.StatusSealed {
			return o.fail(&shape.Error{Code: "shape_not_sealed", Msg: "session " + id + " is not sealed"}, false)
		}
		if o.json {
			return o.encode(map[string]string{"story": s.Story, "digest": s.Digest, "issue": s.JiraKey()})
		}
		fmt.Fprint(o.out, s.Story)
		return 0
	}
	s, err := shape.Load(o.ws, id)
	if err != nil {
		return o.fail(err, false)
	}
	eng := &shape.Engine{Run: shapeAgents(o.ws, s.Agent, s.Model)}
	switch verb {
	case "answer":
		answers, err := parseAnswers(o.answers)
		if err != nil {
			fmt.Fprintln(o.err, err)
			return shapeExitUsage
		}
		next, err := eng.Answer(ctx, o.ws, id, answers, splitList(o.assume))
		return o.turn(next, err)
	case "context":
		return o.contextVerb(eng, id)
	case "split":
		if o.doSplit == o.keep {
			fmt.Fprintln(o.err, "split needs exactly one of --split and --keep")
			return shapeExitUsage
		}
		if o.doSplit {
			next, err := eng.AcceptSplit(o.ws, id)
			return o.show(next, err)
		}
		next, err := eng.KeepWhole(ctx, o.ws, id)
		return o.turn(next, err)
	case "seal":
		next, err := eng.Seal(o.ws, id)
		return o.show(next, err)
	case "fork":
		next, err := eng.Fork(o.ws, id)
		return o.show(next, err)
	default:
		next, err := eng.CreateIssue(ctx, func() (shape.Backlog, error) { return shapeBacklog(o.ws) }, o.ws, id, o.project, o.board, o.confirm)
		return o.show(next, err)
	}
}

func (o *shapeOpts) contextVerb(eng *shape.Engine, id string) int {
	if o.preview {
		diff, err := shape.ContextPreview(o.ws, id, splitList(o.accept))
		if err != nil {
			return o.fail(err, false)
		}
		if o.json {
			return o.encode(diff)
		}
		for _, l := range diff.Lines {
			sign := " "
			if l.Kind == shape.DiffAdd {
				sign = "+"
			}
			fmt.Fprintf(o.out, "%s%4d  %s\n", sign, l.N, l.Text)
		}
		return 0
	}
	next, err := eng.SettleProposals(o.ws, id, splitList(o.accept), splitList(o.skip), splitList(o.restore))
	return o.show(next, err)
}

func (o *shapeOpts) issuePreview(ctx context.Context, id string) int {
	backlog, err := shapeBacklog(o.ws)
	if err != nil {
		return o.fail(err, false)
	}
	view, err := shape.PreviewIssue(ctx, backlog, o.ws, id, o.project, o.board)
	if err != nil {
		return o.fail(err, false)
	}
	if o.json {
		return o.encode(view)
	}
	fmt.Fprintf(o.out, "project %s\nboard %d %s\nbacklog %s\nsummary %s\n", view.Project, view.Board.ID, view.Board.Name, strings.Join(view.BacklogStatuses, ", "), view.Summary)
	return 0
}
