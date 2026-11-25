package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"span/internal/jira"
	"span/internal/serve"
	"span/internal/shape"
)

// Exit codes of `span shape`. They are the same for every verb, so a script can tell a
// missing session from a refused one without reading the message.
const (
	shapeExitFailed   = 1
	shapeExitUsage    = 2
	shapeExitNotFound = 3
	shapeExitRefused  = 4
	shapeExitAgent    = 5
)

const shapeUsage = `usage: ./span shape list|show|start|resume|agent|archive|delete|answer|context|split|seal|source|preview|jira|story|fork|log|diff|restore|compare|merge|board|activity|bulk|history|undo|import|export|share [--workspace .] [--json]
  list    [--status s] [--query q] [--repo r] [--agent a] [--sort activity|created] [--archived include|only]
  show    <id>
  start   (--idea text | --idea-file f | --issue KEY) [--repo r] [--agent a] [--model m]
  resume  <id> [--agent a] [--model m]
  agent   <id> --agent a [--model m]
  archive <id> [--undo]
  delete  <id> --yes
` + shapeSessionUsage + "\n" + shapeVersionUsage + "\n" + shapeBoardUsage + "\n" + shapeHistoryUsage + "\n" + shapeTransferUsage

// shapeAgents builds the interview runner; tests replace it.
var shapeAgents serve.ShapeAgents = serve.CLIShapeRunner

// shapeIssues reads Jira issues for `start --issue`; tests replace it.
var shapeIssues = func(workspace string) (shape.IssueReader, error) {
	account, err := jira.Load(workspace)
	if err != nil {
		return nil, err
	}
	return jira.Client{Account: account}, nil
}

func shapeMain(args []string) int { return shapeCmd(context.Background(), args, os.Stdout, os.Stderr) }

// shapeCmd runs one `span shape` verb. It calls the same code as the HTTP routes, so the
// page and the command line are two clients of one behavior.
func shapeCmd(ctx context.Context, args []string, out, errw io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(errw, shapeUsage)
		return shapeExitUsage
	}
	verb, rest := args[0], args[1:]
	fs := flag.NewFlagSet("shape "+verb, flag.ContinueOnError)
	fs.SetOutput(errw)
	o := &shapeOpts{}
	fs.StringVar(&o.workspace, "workspace", ".", "folder that holds the sessions")
	fs.BoolVar(&o.json, "json", false, "print JSON")
	switch verb {
	case "list":
		fs.StringVar(&o.filter.Status, "status", "", "interviewing, ready, split or sealed")
		fs.StringVar(&o.filter.Query, "query", "", "text in the idea, id, repo or Jira key")
		fs.StringVar(&o.filter.Repo, "repo", "", "repo of the session")
		fs.StringVar(&o.filter.Agent, "agent", "", "agent of the session; auto for none")
		fs.StringVar(&o.filter.Sort, "sort", "", "activity (default) or created")
		fs.StringVar(&o.filter.Archived, "archived", "", "include or only")
	case "start":
		fs.StringVar(&o.idea, "idea", "", "the idea, in your words")
		fs.StringVar(&o.ideaFile, "idea-file", "", "file with the idea")
		fs.StringVar(&o.issue, "issue", "", "Jira issue key to start from")
		fs.StringVar(&o.repo, "repo", "", "repo inside the workspace")
		fs.StringVar(&o.agent, "agent", "", "interview agent; empty uses the project default")
		fs.StringVar(&o.model, "model", "", "model of the agent")
	case "resume", "agent":
		fs.StringVar(&o.agent, "agent", "", "interview agent")
		fs.StringVar(&o.model, "model", "", "model of the agent")
	case "archive":
		fs.BoolVar(&o.undo, "undo", false, "show the session in the list again")
	case "delete":
		fs.BoolVar(&o.yes, "yes", false, "confirm the deletion")
	case "show":
	default:
		if sessionVerbs[verb] {
			o.sessionFlags(fs, verb)
			break
		}
		if versionVerbs[verb] {
			o.versionFlags(fs, verb)
			break
		}
		if boardVerbs[verb] {
			o.boardFlags(fs, verb)
			break
		}
		if historyVerbs[verb] {
			o.historyFlags(fs, verb)
			break
		}
		if transferVerbs[verb] {
			o.transferFlags(fs, verb)
			break
		}
		fmt.Fprintln(errw, shapeUsage)
		return shapeExitUsage
	}
	// Flags may come before or after the session id, as they do on every other verb.
	pos, ok := parseVerb(fs, rest)
	if !ok {
		return shapeExitUsage
	}
	o.args = pos
	ws, err := serve.WorkspaceDir(o.workspace)
	if err != nil {
		fmt.Fprintln(errw, err)
		return shapeExitFailed
	}
	o.ws, o.out, o.err = ws, out, errw
	return o.run(ctx, verb)
}

type shapeOpts struct {
	workspace, ws         string
	json, undo, yes       bool
	filter                shape.Filter
	idea, ideaFile, issue string
	repo, agent, model    string
	args                  []string
	out, err              io.Writer
	sessionOpts
	ver  versionOpts
	brd  boardOpts
	hist historyOpts
	xfer transferOpts
}

func (o *shapeOpts) id() (string, bool) {
	if len(o.args) != 1 {
		fmt.Fprintln(o.err, shapeUsage)
		return "", false
	}
	return o.args[0], true
}

func (o *shapeOpts) run(ctx context.Context, verb string) int {
	if verb == "list" {
		return o.list()
	}
	if verb == "start" {
		return o.start(ctx)
	}
	if versionVerbs[verb] {
		return o.version(verb)
	}
	if boardVerbs[verb] {
		fctx, stop := followContext(ctx)
		defer stop()
		return o.boardVerb(fctx, verb)
	}
	if historyVerbs[verb] {
		return o.historyVerb(verb)
	}
	if transferVerbs[verb] {
		return o.transferVerb(ctx, verb)
	}
	id, ok := o.id()
	if !ok {
		return shapeExitUsage
	}
	if sessionVerbs[verb] {
		return o.session(ctx, verb, id)
	}
	switch verb {
	case "show":
		s, err := shape.Load(o.ws, id)
		return o.show(s, err)
	case "resume":
		eng := &shape.Engine{Run: shapeAgents(o.ws, o.agent, o.model)}
		s, err := eng.Resume(ctx, o.ws, id, o.agent, o.model)
		return o.turn(s, err)
	case "agent":
		if o.agent == "" {
			fmt.Fprintln(o.err, "agent needs --agent")
			return shapeExitUsage
		}
		s, err := (&shape.Engine{}).SetAgent(o.ws, id, o.agent, o.model)
		return o.show(s, err)
	case "archive":
		s, err := shape.SetArchived(o.ws, id, !o.undo)
		return o.show(s, err)
	default:
		if !o.yes {
			fmt.Fprintln(o.err, "delete needs --yes")
			return shapeExitUsage
		}
		if err := shape.Delete(o.ws, id); err != nil {
			return o.fail(err, false)
		}
		fmt.Fprintf(o.out, "deleted %s\n", id)
		return 0
	}
}

func (o *shapeOpts) list() int {
	sessions, err := shape.List(o.ws)
	if sessions == nil {
		return o.fail(err, false)
	}
	if err != nil {
		fmt.Fprintln(o.err, "skipped:", strings.Join(strings.Fields(err.Error()), " "))
	}
	kept, ferr := o.filter.Apply(sessions)
	if ferr != nil {
		return o.fail(ferr, false)
	}
	if o.json {
		return o.encode(shape.Views(kept))
	}
	for _, s := range kept {
		v := s.View()
		fmt.Fprintf(o.out, "%s\t%s\t%s\t%d open\t%s\t%s\n", v.Ref, v.Status, agentLabel(s), v.OpenQuestions, v.JiraKey, v.Title)
	}
	return 0
}

func agentLabel(s *shape.Session) string {
	name, model := s.Agent, s.Model
	if name == "" {
		name = "auto"
	}
	if model == "" {
		model = "auto"
	}
	return name + "/" + model
}

func (o *shapeOpts) start(ctx context.Context) int {
	idea := strings.TrimSpace(o.idea)
	if o.ideaFile != "" {
		raw, err := os.ReadFile(o.ideaFile)
		if err != nil {
			return o.fail(err, false)
		}
		idea = strings.TrimSpace(string(raw))
	}
	source := ""
	if o.issue != "" {
		reader, err := shapeIssues(o.ws)
		if err != nil {
			return o.fail(err, false)
		}
		if idea, source, err = shape.IdeaFromIssue(ctx, reader, o.issue, idea); err != nil {
			return o.fail(err, false)
		}
	}
	if idea == "" {
		fmt.Fprintln(o.err, "start needs --idea, --idea-file or --issue")
		return shapeExitUsage
	}
	eng := &shape.Engine{Run: shapeAgents(o.ws, o.agent, o.model)}
	s, err := eng.Start(ctx, shape.StartInput{Workspace: o.ws, Repo: o.repo, Idea: idea, Agent: o.agent, Model: o.model, SourceIssue: source})
	return o.turn(s, err)
}

// turn reports a verb that ran the interview. A failed turn leaves the session as it was,
// with the failure saved on it, and exits 5.
func (o *shapeOpts) turn(s *shape.Session, err error) int {
	if err != nil {
		return o.fail(err, true)
	}
	return o.show(s, nil)
}

func (o *shapeOpts) show(s *shape.Session, err error) int {
	if err != nil {
		return o.fail(err, false)
	}
	if o.json {
		return o.encode(s.View())
	}
	v := s.View()
	fmt.Fprintf(o.out, "%s %s\nstatus %s\nagent %s\nrepo %s\nopen questions %d\nrules %d\nexamples %d\njira %s\nupdated %s\narchived %t\n",
		v.Ref, v.Title, v.Status, agentLabel(s), s.Repo, v.OpenQuestions, len(s.Map.Rules), len(s.Map.Examples), v.JiraKey, v.UpdatedAt.Format("2006-01-02 15:04:05"), s.Archived)
	if f := s.Failure; f != nil {
		fmt.Fprintf(o.out, "failure %s at %s: %s\n", f.Kind, f.At.Format("15:04"), f.Message)
	}
	return 0
}

func (o *shapeOpts) encode(v any) int {
	return emit(o.out, o.err, v)
}

// fail prints the error and picks the exit code: not found, refused (the session is in a
// state that does not allow it), the agent failing, or any other failure.
func (o *shapeOpts) fail(err error, ran bool) int {
	if o.json {
		code := "error"
		var coded interface{ SpanCode() string }
		if errors.As(err, &coded) {
			code = coded.SpanCode()
		}
		raw, marshalErr := json.Marshal(map[string]string{"error": code, "message": err.Error()})
		if marshalErr != nil {
			raw = []byte(`{"error":"error","message":"the error could not be encoded"}`)
		}
		fmt.Fprintln(o.out, string(raw))
	} else {
		fmt.Fprintln(o.err, err)
	}
	var coded interface{ SpanCode() string }
	if errors.As(err, &coded) {
		switch coded.SpanCode() {
		case "shape_not_found", "not_found", "shape_source_not_found", "shape_revision_not_found", "shape_step_not_found", "shape_share_not_found":
			return shapeExitNotFound
		case "shape_sealed", "shape_not_ready", "shape_not_sealed", "shape_repo_changed", "shape_context_written", "shape_no_split",
			"shape_conflicts_open", "shape_not_undoable", "shape_has_dependents", "shape_context_changed", "shape_share_expired", "shape_share_revoked", "confirm_required", "jira_backlog_disabled", "jira_board_ambiguous", "jira_not_in_backlog":
			return shapeExitRefused
		case "invalid", "shape_repo_outside", "shape_not_git":
			return shapeExitFailed
		}
	}
	if ran {
		return shapeExitAgent
	}
	return shapeExitFailed
}
