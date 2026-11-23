// Package engine walks a span from brief to deploy.
// Stages write a statement before the next one starts. External effects are
// retried only when the port can show they did not happen.
package engine

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"span/internal/agent"
	"span/internal/branch"
	"span/internal/feature"
	"span/internal/handoff"
	"span/internal/harness"
	"span/internal/learn"
	"span/internal/policy"
	"span/internal/prompt"
	"span/internal/prove"
	"span/internal/qa"
	"span/internal/statement"
	"span/internal/transcript"
)

const (
	intake   = "intake"
	frame    = "frame"
	build    = "build"
	proveSt  = "prove"
	qaSt     = "qa"
	inspect  = "inspect"
	open     = "open"
	merge    = "merge"
	promote  = "promote"
	note     = "note"
	maxIter  = 3
	maxSteps = 40

	missingAcceptance = "Acceptance criteria are missing."
)

var order = []string{intake, frame, build, proveSt, qaSt, inspect, open, merge, promote, note}

// Source is the workspace git boundary.
type Source interface {
	Head(ctx context.Context) (string, error)
	Dirty(ctx context.Context) (bool, error)
	Commit(ctx context.Context, message string) (string, error)
	Push(ctx context.Context) error
	Branch(ctx context.Context) (string, error)
	EnsureBranch(ctx context.Context, base, branch string) error
}

// Review opens and merges a change request.
type Review interface {
	Find(ctx context.Context, spanID, branch string) (url string, found bool, err error)
	Open(ctx context.Context, spanID, branch, title, body string) (string, error)
	Merged(ctx context.Context, url string) (bool, error)
	Merge(ctx context.Context, url string) error
}

// Releaser runs one environment command. A returned error means the command
// finished and failed. A killed process never returns.
type Releaser interface {
	Run(ctx context.Context, command, dir, sha string) error
}

// FrameDoc is the frame agent's answer.
type FrameDoc struct {
	Claims      []string `json:"claims"`
	Assumptions []string `json:"assumptions"`
	Questions   []string `json:"questions"`
	Acceptance  []string `json:"acceptance"`
	OutOfScope  bool     `json:"outOfScope"`
}

// Answer is one reply to a question the run is waiting on.
type Answer struct {
	Question string `json:"question"`
	Body     string `json:"body"`
}

// InspectDoc is the inspect agent's answer.
type InspectDoc struct {
	Blocking bool   `json:"blocking"`
	Reason   string `json:"reason"`
}

// IssueBoard comments on one issue and moves it to a named status.
type IssueBoard interface {
	Comment(ctx context.Context, key, text string) error
	Transition(ctx context.Context, key, statusName string) error
}

// Agents runs the selected CLIs. Select fails when Gentle AI is missing.
type Agents interface {
	Select(want string) (agent.Selection, error)
	Frame(ctx context.Context, name, workspace, text string) (FrameDoc, error)
	Build(ctx context.Context, name, workspace, text string) error
	Inspect(ctx context.Context, name, workspace, text string) (InspectDoc, error)
}

// Options configures one run. Root is the directory that holds span ids.
type Options struct {
	Workspace string
	Root      string
	Policy    policy.File
	Brief     string
	Issue     string
	Branch    string
	Base      string
	Files     []handoff.File
	Issues    IssueBoard
	Now       func() time.Time
	Agents    Agents
	Check     prove.Checker
	Source    Source
	Review    Review
	Release   Releaser
	Watch     Publisher
	Secrets   []string
}

// Result is how far the span got.
type Result struct {
	ID        string
	Stage     string
	Verdict   string
	Reason    string
	ChangeURL string
}

// Cursor is the resume point. It is written before the next stage starts.
type Cursor struct {
	ID              string     `json:"id"`
	Iteration       int        `json:"iteration"`
	Stage           string     `json:"stage"`
	Hold            bool       `json:"hold"`
	Stop            bool       `json:"stop"`
	Done            bool       `json:"done"`
	Reason          string     `json:"reason,omitempty"`
	GitSHA          string     `json:"gitSHA,omitempty"`
	ChangeURL       string     `json:"changeURL,omitempty"`
	Pending         *Pending   `json:"pending,omitempty"`
	InspectBlocks   int        `json:"inspectBlocks"`
	LastDigest      string     `json:"lastDigest,omitempty"`
	FrameAdvancedAt *time.Time `json:"frameAdvancedAt,omitempty"`
	StartedAt       time.Time  `json:"startedAt"`
	Branch          string     `json:"branch,omitempty"`
	Base            string     `json:"base,omitempty"`
	Workspace       string     `json:"workspace,omitempty"`
	Issue           string     `json:"issue,omitempty"`
	Questions       []string   `json:"questions,omitempty"`
}

// Pending is an external effect that was started and may not have finished.
type Pending struct {
	Kind        string `json:"kind"`
	Environment string `json:"environment,omitempty"`
	SHA         string `json:"sha,omitempty"`
	Branch      string `json:"branch,omitempty"`
}

// Publisher receives run events. A slow receiver must not fail the run.
type Publisher interface {
	Publish(id string, event Event)
}

// Event is one JSON frame on the live socket.
type Event struct {
	Type       string               `json:"type"`
	Cursor     *Cursor              `json:"cursor,omitempty"`
	Statement  *statement.Statement `json:"statement,omitempty"`
	Gate       *qa.Gate             `json:"gate,omitempty"`
	Stage      string               `json:"stage,omitempty"`
	Lines      []string             `json:"lines,omitempty"`
	Transcript []string             `json:"transcript,omitempty"`
	Span       *Snapshot            `json:"span,omitempty"`
}

// Snapshot is the durable reading of one run.
type Snapshot struct {
	Cursor     Cursor                `json:"cursor"`
	Statements []statement.Statement `json:"statements"`
	Acceptance []qa.Line             `json:"acceptance"`
	Gate       *qa.Gate              `json:"gate,omitempty"`
}

// Start creates a span and walks until it holds, stops, or finishes.
func Start(ctx context.Context, opt Options) (Result, error) {
	id, err := Open(opt)
	if err != nil {
		return Result{}, err
	}
	return Resume(ctx, opt, id)
}

// Open writes the cursor and returns once the run directory exists.
func Open(opt Options) (string, error) {
	if strings.TrimSpace(opt.Brief) == "" {
		return "", errors.New("brief is empty")
	}
	if err := opt.Policy.Validate(); err != nil {
		return "", err
	}
	now := opt.now()
	id, err := newID(now)
	if err != nil {
		return "", err
	}
	cur := Cursor{
		ID: id, Iteration: 1, Stage: intake, StartedAt: now,
		Branch: opt.Branch, Base: opt.Base, Workspace: opt.Workspace,
		Issue: strings.TrimSpace(opt.Issue),
	}
	r := &runner{opt: opt, cur: cur}
	if err := os.MkdirAll(r.spanDir(), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(r.spanDir(), "brief.md"), []byte(opt.Brief), 0o644); err != nil {
		return "", err
	}
	if err := handoff.Write(filepath.Join(r.spanDir(), "handoffs"), opt.Files); err != nil {
		return "", err
	}
	if err := r.save(); err != nil {
		return "", err
	}
	return id, nil
}

// Resume continues a span from its cursor. A stopped or finished span stays there.
func Resume(ctx context.Context, opt Options, id string) (Result, error) {
	r, err := load(opt, id)
	if err != nil {
		return Result{}, err
	}
	if r.cur.Stop {
		return r.finish(statement.Stop), nil
	}
	if r.cur.Done {
		return r.finish(statement.Advance), nil
	}
	r.cur.Hold = false
	if err := r.save(); err != nil {
		return Result{}, err
	}
	return r.loop(ctx)
}

// Waiting reports a run that stopped or held for questions and can continue.
func Waiting(cur Cursor) bool {
	if cur.Done {
		return false
	}
	if len(cur.Questions) > 0 {
		return cur.Hold || cur.Stop
	}
	return strings.HasPrefix(cur.Reason, "open questions:") && (cur.Hold || cur.Stop)
}

// Continue records the answers and runs the waiting span again.
func Continue(ctx context.Context, opt Options, id string, answers []Answer) (Result, error) {
	r, err := load(opt, id)
	if err != nil {
		return Result{}, err
	}
	if !Waiting(r.cur) {
		if r.cur.Stop {
			return r.finish(statement.Stop), nil
		}
		return r.finish(statement.Hold), nil
	}
	if err := writeAnswers(r.spanDir(), answers); err != nil {
		return Result{}, err
	}
	for _, answer := range answers {
		if strings.TrimSpace(answer.Question) != missingAcceptance {
			continue
		}
		r.opt.Brief = qa.Install(r.opt.Brief, splitAnswer(answer.Body))
	}
	if err := os.WriteFile(filepath.Join(r.spanDir(), "brief.md"), []byte(r.opt.Brief), 0o644); err != nil {
		return Result{}, err
	}
	r.clearWait()
	if err := r.save(); err != nil {
		return Result{}, err
	}
	return r.loop(ctx)
}

// Reload replaces the brief with the story and runs the waiting span again.
func Reload(ctx context.Context, opt Options, id, brief string) (Result, error) {
	r, err := load(opt, id)
	if err != nil {
		return Result{}, err
	}
	if !Waiting(r.cur) {
		if r.cur.Stop {
			return r.finish(statement.Stop), nil
		}
		return r.finish(statement.Hold), nil
	}
	if strings.TrimSpace(brief) == "" {
		return Result{}, errors.New("brief is empty")
	}
	r.opt.Brief = brief
	if err := os.WriteFile(filepath.Join(r.spanDir(), "brief.md"), []byte(brief), 0o644); err != nil {
		return Result{}, err
	}
	if key := IssueKey(brief); key != "" {
		r.cur.Issue = key
		r.opt.Issue = key
	}
	r.clearWait()
	if err := r.save(); err != nil {
		return Result{}, err
	}
	return r.loop(ctx)
}

// IssueKey reads a KEY-123 token from the first line of a brief.
func IssueKey(brief string) string {
	line := brief
	if i := strings.IndexByte(brief, '\n'); i >= 0 {
		line = brief[:i]
	}
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return ""
	}
	project, number, ok := strings.Cut(fields[0], "-")
	if !ok || project == "" || number == "" {
		return ""
	}
	for _, r := range project {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return ""
		}
	}
	for _, r := range number {
		if !unicode.IsDigit(r) {
			return ""
		}
	}
	return fields[0]
}

func load(opt Options, id string) (*runner, error) {
	if err := opt.Policy.Validate(); err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(filepath.Join(opt.Root, id, "cursor.json"))
	if err != nil {
		return nil, err
	}
	var cur Cursor
	if err := json.Unmarshal(raw, &cur); err != nil {
		return nil, err
	}
	if strings.TrimSpace(opt.Brief) == "" {
		rawBrief, err := os.ReadFile(filepath.Join(opt.Root, id, "brief.md"))
		if err != nil {
			return nil, err
		}
		opt.Brief = string(rawBrief)
	}
	if opt.Branch == "" {
		opt.Branch = cur.Branch
		opt.Base = cur.Base
	}
	if strings.TrimSpace(opt.Issue) == "" {
		opt.Issue = cur.Issue
	}
	if cur.Issue == "" {
		cur.Issue = IssueKey(opt.Brief)
	}
	return &runner{opt: opt, cur: cur}, nil
}

func (r *runner) clearWait() {
	r.cur.Stop = false
	r.cur.Hold = false
	r.cur.Questions = nil
	r.cur.Reason = ""
}

type runner struct {
	opt      Options
	cur      Cursor
	log      *transcript.Log
	book     *learn.Book
	bookPath string
	shown    map[string][]string
}

func (r *runner) loop(ctx context.Context) (Result, error) {
	ctx = agent.WithSpan(ctx, r.cur.ID)
	log, err := transcript.Open(filepath.Join(r.spanDir(), "transcript.log"), r.opt.Secrets, func(stage string, lines []string) {
		r.publish(Event{Type: "transcript", Stage: stage, Lines: lines})
	})
	if err != nil {
		return Result{}, err
	}
	defer log.Close()
	r.log = log
	if binder, ok := r.opt.Agents.(interface{ Bind(io.Writer) }); ok {
		binder.Bind(log)
	}
	if err := r.ensureBranch(ctx); err != nil {
		_, reason, recErr := r.record(r.cur.Stage, statement.Hold, err.Error(), nil)
		if recErr != nil {
			return Result{}, recErr
		}
		r.cur.Hold = true
		r.cur.Reason = reason
		if err := r.save(); err != nil {
			return Result{}, err
		}
		return r.finish(statement.Hold), nil
	}
	for step := 0; step < maxSteps; step++ {
		if r.cur.Done {
			return r.finish(statement.Advance), nil
		}
		stage := r.cur.Stage
		if advanced, err := r.already(stage); err != nil {
			return Result{}, err
		} else if advanced {
			r.cur.Stage = next(stage)
			if err := r.save(); err != nil {
				return Result{}, err
			}
			continue
		}
		verdict, reason, err := r.exec(ctx, stage)
		if err != nil {
			return Result{}, err
		}
		switch verdict {
		case statement.Advance:
			if stage == note {
				r.cur.Done = true
			} else {
				r.cur.Stage = next(stage)
			}
			r.cur.Questions = nil
			r.cur.Reason = reason
		case statement.Hold:
			r.cur.Hold = true
			r.cur.Reason = reason
		case statement.Stop:
			r.cur.Stop = true
			r.cur.Reason = reason
		case "iterate":
			r.cur.Iteration++
			r.cur.Stage = frame
			r.cur.InspectBlocks = 0
			r.cur.Reason = reason
		case "rebuild":
			r.cur.Stage = build
			r.cur.Reason = reason
		default:
			return Result{}, fmt.Errorf("unknown verdict %s", verdict)
		}
		if err := r.save(); err != nil {
			return Result{}, err
		}
		if r.cur.Hold || r.cur.Stop || r.cur.Done {
			return r.finish(verdict), nil
		}
	}
	return Result{}, errors.New("span exceeded the stage budget")
}

func (r *runner) finish(verdict string) Result {
	if verdict == "iterate" || verdict == "rebuild" {
		verdict = statement.Hold
	}
	return Result{ID: r.cur.ID, Stage: r.cur.Stage, Verdict: verdict, Reason: r.cur.Reason, ChangeURL: r.cur.ChangeURL}
}

func (r *runner) exec(ctx context.Context, stage string) (string, string, error) {
	if r.log != nil {
		r.log.SetStage(stage)
	}
	switch stage {
	case intake:
		return r.doIntake()
	case frame:
		return r.doFrame(ctx)
	case build:
		return r.doBuild(ctx)
	case proveSt:
		return r.doProve(ctx)
	case qaSt:
		return r.doQA(ctx)
	case inspect:
		return r.doInspect(ctx)
	case open:
		return r.doOpen(ctx)
	case merge:
		return r.doMerge(ctx)
	case promote:
		return r.doPromote(ctx)
	case note:
		return r.doNote()
	default:
		return "", "", fmt.Errorf("unknown stage %s", stage)
	}
}

func (r *runner) doIntake() (string, string, error) {
	sum := sha256.Sum256([]byte(r.opt.Brief))
	reason := "brief accepted"
	return r.record(intake, statement.Advance, reason, func(s *statement.Statement) {
		s.BriefDigest = hex.EncodeToString(sum[:])
	})
}

func (r *runner) doFrame(ctx context.Context) (string, string, error) {
	sel, err := r.opt.Agents.Select(r.opt.Policy.Agent)
	if err != nil {
		return r.record(frame, statement.Hold, err.Error(), nil)
	}
	prev := ""
	if r.cur.Iteration > 1 {
		raw, err := os.ReadFile(filepath.Join(r.spanDir(), "iter", strconv.Itoa(r.cur.Iteration-1), "prove.md"))
		if err == nil {
			prev = string(raw)
		}
	}
	in := r.promptInput(frame, prev)
	doc, err := r.opt.Agents.Frame(ctx, sel.Build, r.opt.Workspace, prompt.Frame(in))
	if err != nil {
		return r.record(frame, statement.Hold, err.Error(), nil)
	}
	if doc.OutOfScope {
		return r.record(frame, statement.Stop, "out of scope", nil)
	}
	if err := r.adopt(doc.Acceptance); err != nil {
		return "", "", err
	}
	questions := append([]string{}, doc.Questions...)
	if strings.TrimSpace(r.opt.Issue) != "" && !qa.HasAcceptance(r.opt.Brief) {
		questions = append(questions, missingAcceptance)
	}
	if len(questions) > 0 {
		return r.block(ctx, questions)
	}
	if len(doc.Claims) == 0 {
		return r.record(frame, statement.Hold, "frame produced no claims", nil)
	}
	now := r.opt.now()
	r.cur.FrameAdvancedAt = &now
	return r.record(frame, statement.Advance, fmt.Sprintf("%d claims", len(doc.Claims)), nil)
}

func (r *runner) doBuild(ctx context.Context) (string, string, error) {
	sel, err := r.opt.Agents.Select(r.opt.Policy.Agent)
	if err != nil {
		return r.record(build, statement.Hold, err.Error(), nil)
	}
	in := r.promptInput(build, "")
	if err := r.opt.Agents.Build(ctx, sel.Build, r.opt.Workspace, prompt.Build(in)); err != nil {
		return r.record(build, statement.Hold, err.Error(), nil)
	}
	path := featurePath(r.opt.Workspace, r.opt.Policy.Feature)
	if err := feature.Valid(path); err != nil {
		return r.record(build, statement.Stop, err.Error(), func(s *statement.Statement) {
			s.FeatureDoc = path
		})
	}
	sha, err := r.commit(ctx)
	if err != nil {
		return r.record(build, statement.Hold, err.Error(), nil)
	}
	r.cur.GitSHA = sha
	return r.record(build, statement.Advance, "feature document present", func(s *statement.Statement) {
		s.FeatureDoc = path
		s.GitSHA = sha
	})
}

func (r *runner) doProve(ctx context.Context) (string, string, error) {
	rep, err := prove.Bands(ctx, r.opt.Check, r.opt.Workspace, r.opt.Policy.Prove.Fast, r.opt.Policy.Prove.Acceptance)
	if err != nil {
		return "", "", err
	}
	verdict := rep.Outcome
	if verdict == "iterate" && r.cur.Iteration >= maxIter {
		verdict = statement.Hold
		rep.Reason = "iteration cap: " + rep.Reason
	}
	return r.record(proveSt, verdict, rep.Reason, func(s *statement.Statement) {
		s.Commands = rep.Commands
		s.GitSHA = r.cur.GitSHA
		s.Attributable = rep.Outcome == "iterate"
	})
}

func (r *runner) block(ctx context.Context, questions []string) (string, string, error) {
	r.cur.Questions = append([]string{}, questions...)
	reason := "open questions: " + strings.Join(questions, "; ")
	issue := strings.TrimSpace(r.opt.Issue)
	if issue != "" && r.opt.Issues != nil {
		if err := r.opt.Issues.Comment(ctx, issue, waitComment(questions)); err != nil {
			reason += "; the Jira comment was not posted"
		}
	}
	return r.record(frame, statement.Hold, reason, nil)
}

func waitComment(questions []string) string {
	var b strings.Builder
	b.WriteString("Span is waiting for an answer before it writes code.\n")
	for i, question := range questions {
		fmt.Fprintf(&b, "%d. %s\n", i+1, question)
	}
	b.WriteString("Answer on the span, or update this issue and read the story again.\n")
	return b.String()
}

func (r *runner) adopt(found []string) error {
	next := qa.Install(r.opt.Brief, found)
	if next == r.opt.Brief {
		return nil
	}
	r.opt.Brief = next
	return os.WriteFile(filepath.Join(r.spanDir(), "brief.md"), []byte(next), 0o644)
}

func (r *runner) doQA(ctx context.Context) (string, string, error) {
	lines := qa.Lines(r.opt.Brief)
	if len(lines) == 0 {
		return r.record(qaSt, statement.Advance, "no acceptance criteria", nil)
	}
	if err := writeJSON(filepath.Join(r.spanDir(), "acceptance.json"), lines); err != nil {
		return "", "", err
	}
	command := r.opt.Policy.QACommand(r.browserQA())
	gate := qa.Gate{Command: command, Checks: make([]qa.Check, 0, len(lines))}
	if command == "" || prove.Prose(command) {
		for _, line := range lines {
			gate.Checks = append(gate.Checks, qa.Check{ID: line.ID, Result: "unverified"})
		}
		gate.Verdict = "unverified"
		if err := r.writeGate(gate); err != nil {
			return "", "", err
		}
		reason := "qa command is missing"
		if prove.Prose(command) {
			reason = command + " is not a command"
		}
		return r.record(qaSt, statement.Hold, reason, func(s *statement.Statement) {
			s.Commands = []statement.Command{{Command: command, Band: "qa", Error: reason}}
		})
	}
	code, err := r.opt.Check.Run(ctx, command, r.opt.Workspace)
	ran := err == nil
	if !ran {
		code = 0
	}
	covered, coverErr := qa.Covered(r.opt.Workspace, lines)
	if coverErr != nil {
		return "", "", coverErr
	}
	gate = qa.Judge(lines, code, ran, covered)
	gate.Command = command
	if err := r.writeGate(gate); err != nil {
		return "", "", err
	}
	item := statement.Command{Command: command, ExitCode: code, Band: "qa"}
	if err != nil {
		item.Error = err.Error()
		return r.record(qaSt, statement.Hold, command+" did not run: "+err.Error(), func(s *statement.Statement) {
			s.Commands = []statement.Command{item}
		})
	}
	if gate.Verdict != "pass" {
		return r.record(qaSt, statement.Hold, "acceptance gate "+gate.Verdict, func(s *statement.Statement) {
			s.Commands = []statement.Command{item}
		})
	}
	return r.record(qaSt, statement.Advance, "acceptance gate passed", func(s *statement.Statement) {
		s.Commands = []statement.Command{item}
		s.GitSHA = r.cur.GitSHA
	})
}

func (r *runner) doInspect(ctx context.Context) (string, string, error) {
	sel, err := r.opt.Agents.Select(r.opt.Policy.Agent)
	if err != nil {
		return r.record(inspect, statement.Hold, err.Error(), nil)
	}
	doc, err := r.opt.Agents.Inspect(ctx, sel.Inspect, r.opt.Workspace, prompt.Inspect(r.promptInput(inspect, "")))
	if err != nil {
		return r.record(inspect, statement.Hold, err.Error(), nil)
	}
	if !doc.Blocking {
		return r.record(inspect, statement.Advance, doc.Reason, nil)
	}
	r.cur.InspectBlocks++
	if r.cur.InspectBlocks >= 3 {
		return r.record(inspect, statement.Hold, "inspect cap: "+doc.Reason, nil)
	}
	dir := r.iterDir()
	suffix := fmt.Sprintf("block-%d", r.cur.InspectBlocks)
	for _, stage := range []string{build, proveSt, qaSt} {
		if err := statement.Archive(dir, stage, suffix); err != nil {
			return "", "", err
		}
	}
	if _, _, err := r.record(inspect, statement.Stop, doc.Reason, nil); err != nil {
		return "", "", err
	}
	if err := statement.Archive(dir, inspect, suffix); err != nil {
		return "", "", err
	}
	return "rebuild", doc.Reason, nil
}

func (r *runner) doOpen(ctx context.Context) (string, string, error) {
	if r.opt.Policy.OpenMode() == statement.Hold {
		return r.record(open, statement.Hold, "open is hold", nil)
	}
	branch, err := r.opt.Source.Branch(ctx)
	if err != nil {
		return r.record(open, statement.Hold, err.Error(), nil)
	}
	r.cur.Pending = &Pending{Kind: open, Branch: branch, SHA: r.cur.GitSHA}
	if err := r.save(); err != nil {
		return "", "", err
	}
	url, found, err := r.opt.Review.Find(ctx, r.cur.ID, branch)
	if err != nil {
		return r.record(open, statement.Hold, "change lookup failed: "+err.Error(), nil)
	}
	if !found {
		if err := r.opt.Source.Push(ctx); err != nil {
			return r.record(open, statement.Hold, err.Error(), nil)
		}
		url, err = r.opt.Review.Open(ctx, r.cur.ID, branch, r.opt.Policy.Feature, r.opt.Brief)
		if err != nil {
			return r.record(open, statement.Hold, "open outcome unknown: "+err.Error(), nil)
		}
	}
	r.cur.ChangeURL = url
	r.cur.Pending = nil
	return r.record(open, statement.Advance, "change open", func(s *statement.Statement) {
		s.ChangeURL = url
		s.GitSHA = r.cur.GitSHA
	})
}

func (r *runner) doMerge(ctx context.Context) (string, string, error) {
	if r.cur.ChangeURL == "" {
		return r.record(merge, statement.Hold, "no change url", nil)
	}
	if r.opt.Policy.MergeMode() == statement.Hold {
		return r.record(merge, statement.Hold, "merge is hold", nil)
	}
	r.cur.Pending = &Pending{Kind: merge, SHA: r.cur.GitSHA}
	if err := r.save(); err != nil {
		return "", "", err
	}
	merged, err := r.opt.Review.Merged(ctx, r.cur.ChangeURL)
	if err != nil {
		return r.record(merge, statement.Hold, "merge lookup failed: "+err.Error(), nil)
	}
	if !merged {
		if err := r.opt.Review.Merge(ctx, r.cur.ChangeURL); err != nil {
			return r.record(merge, statement.Hold, "merge outcome unknown: "+err.Error(), nil)
		}
	}
	r.cur.Pending = nil
	return r.record(merge, statement.Advance, "merged", func(s *statement.Statement) {
		s.ChangeURL = r.cur.ChangeURL
	})
}

func (r *runner) doPromote(ctx context.Context) (string, string, error) {
	var done []string
	for _, env := range r.opt.Policy.Promote.Environments {
		mode := env.Mode
		if mode == "" {
			mode = "auto"
		}
		if mode == statement.Hold {
			return r.record(promote, statement.Hold, env.Name+" is hold", func(s *statement.Statement) {
				s.Environment = env.Name
			})
		}
		if r.receiptExists(env.Name, r.cur.GitSHA) {
			done = append(done, env.Name)
			continue
		}
		if r.cur.Pending != nil && r.cur.Pending.Kind == promote && r.cur.Pending.Environment == env.Name {
			return r.record(promote, statement.Hold, env.Name+" outcome unknown", func(s *statement.Statement) {
				s.Environment = env.Name
				s.GitSHA = r.cur.GitSHA
			})
		}
		r.cur.Pending = &Pending{Kind: promote, Environment: env.Name, SHA: r.cur.GitSHA}
		if err := r.save(); err != nil {
			return "", "", err
		}
		if err := r.opt.Release.Run(ctx, env.Command, r.opt.Workspace, r.cur.GitSHA); err != nil {
			r.cur.Pending = nil
			return r.record(promote, statement.Stop, env.Name+": "+err.Error(), func(s *statement.Statement) {
				s.Environment = env.Name
				s.GitSHA = r.cur.GitSHA
			})
		}
		if err := r.writeReceipt(env.Name, r.cur.GitSHA); err != nil {
			return "", "", err
		}
		r.cur.Pending = nil
		if err := r.save(); err != nil {
			return "", "", err
		}
		done = append(done, env.Name)
	}
	return r.record(promote, statement.Advance, "promoted "+strings.Join(done, ","), func(s *statement.Statement) {
		s.Environment = strings.Join(done, ",")
		s.GitSHA = r.cur.GitSHA
	})
}

func (r *runner) doNote() (string, string, error) {
	reason := "no frame time"
	if r.cur.FrameAdvancedAt != nil {
		reason = "lead time " + r.opt.now().Sub(*r.cur.FrameAdvancedAt).String()
	}
	return r.record(note, statement.Advance, reason, nil)
}

func (r *runner) commit(ctx context.Context) (string, error) {
	dirty, err := r.opt.Source.Dirty(ctx)
	if err != nil {
		return "", err
	}
	if !dirty {
		return r.opt.Source.Head(ctx)
	}
	return r.opt.Source.Commit(ctx, fmt.Sprintf("feat(%s): span %s", r.opt.Policy.Feature, r.cur.ID))
}

func (r *runner) record(stage, verdict, reason string, fill func(*statement.Statement)) (string, string, error) {
	stored := verdict
	if stored == "iterate" {
		stored = statement.Hold
	}
	text, digest, err := readInstructions(r.opt.Workspace, r.spanDir(), stage)
	if err != nil {
		return "", "", err
	}
	handoffs, err := handoffDigests(r.spanDir())
	if err != nil {
		return "", "", err
	}
	s := statement.Statement{
		SpanID: r.cur.ID, Stage: stage, Iteration: r.cur.Iteration,
		Verdict: stored, ParentDigest: r.cur.LastDigest, Reason: reason,
		InstructionsDigest: digest, ManifestDigest: manifestDigest(r.opt.Policy, stage),
		HandoffDigests: handoffs, Time: r.opt.now(),
	}
	if fill != nil {
		fill(&s)
	}
	_ = text
	if err := statement.Write(r.iterDir(), &s); err != nil {
		return "", "", err
	}
	sealed := s
	r.publish(Event{Type: "statement", Statement: &sealed})
	r.cur.LastDigest = s.Digest
	r.noteLesson(stage, stored, reason)
	return verdict, reason, nil
}

func (r *runner) ensureBranch(ctx context.Context) error {
	if r.opt.Branch == "" {
		return nil
	}
	if !branch.Valid(r.opt.Branch, r.opt.Base) {
		return fmt.Errorf("branch %q on %q is not feat or fix from dev, or hotfix from master", r.opt.Branch, r.opt.Base)
	}
	r.cur.Branch = r.opt.Branch
	r.cur.Base = r.opt.Base
	return r.opt.Source.EnsureBranch(ctx, r.opt.Base, r.opt.Branch)
}

func (r *runner) promptInput(stage, proveReport string) prompt.Input {
	text, _, _ := readInstructions(r.opt.Workspace, r.spanDir(), stage)
	names, _ := handoffNames(r.spanDir())
	root := r.opt.Policy.Harness
	if root == "" {
		root = harness.Root()
	}
	manifest := manifestText(r.opt.Policy, stage)
	browser := stage == build && r.browserQA()
	if browser && !strings.Contains(manifest, "@playwright/mcp") {
		if manifest != "" {
			manifest += "\n"
		}
		manifest += "mcp: @playwright/mcp --caps=testing"
	}
	var ids []string
	if stage == build {
		for _, line := range qa.Lines(r.opt.Brief) {
			ids = append(ids, line.ID)
		}
	}
	lessons := ""
	if learn.Applies(stage) {
		var sigs []string
		lessons, sigs = r.lessonBook().Select(stage)
		if r.shown == nil {
			r.shown = map[string][]string{}
		}
		r.shown[stage] = sigs
	}
	return prompt.Input{
		Stage: stage, FeaturePath: featurePath(r.opt.Workspace, r.opt.Policy.Feature),
		Brief: r.opt.Brief, Instructions: text, Answers: readAnswers(r.spanDir()),
		Handoffs:    handoff.Paths(r.cur.ID, names),
		ProveReport: proveReport, Manifest: manifest,
		Standards: harness.Catalog(root), Acceptance: ids, Browser: browser,
		Lessons: lessons,
	}
}

// lessonBook reads the project ledger once. A broken file stays untouched.
func (r *runner) lessonBook() *learn.Book {
	if r.book != nil {
		return r.book
	}
	r.bookPath = filepath.Join(r.opt.Workspace, ".span", "lessons.json")
	book, err := learn.Load(r.bookPath)
	if err != nil {
		r.book = &learn.Book{}
		r.bookPath = ""
		return r.book
	}
	r.book = book
	return r.book
}

// noteLesson updates the ledger after the statement is sealed.
// A ledger error does not change the verdict.
func (r *runner) noteLesson(stage, verdict, reason string) {
	if !learn.Applies(stage) {
		return
	}
	book := r.lessonBook()
	var shown []string
	if r.shown != nil {
		shown = r.shown[stage]
		delete(r.shown, stage)
	}
	if book.Observe(stage, verdict, reason, shown) && r.bookPath != "" {
		_ = learn.Save(r.bookPath, book)
	}
}

func (r *runner) browserQA() bool {
	if r.opt.Policy.QA.Browser {
		return true
	}
	matches, err := filepath.Glob(filepath.Join(r.opt.Workspace, "playwright.config.*"))
	return err == nil && len(matches) > 0
}

func (r *runner) already(stage string) (bool, error) {
	s, ok, err := statement.Read(r.iterDir(), stage)
	if err != nil || !ok {
		return false, err
	}
	return s.Verdict == statement.Advance, nil
}

func readAnswers(dir string) string {
	raw, err := os.ReadFile(filepath.Join(dir, "answers.md"))
	if err != nil {
		return ""
	}
	return string(raw)
}

func writeAnswers(dir string, answers []Answer) error {
	var b strings.Builder
	if raw, err := os.ReadFile(filepath.Join(dir, "answers.md")); err == nil {
		b.Write(raw)
		if len(raw) > 0 && raw[len(raw)-1] != '\n' {
			b.WriteByte('\n')
		}
	}
	for _, answer := range answers {
		question := strings.TrimSpace(answer.Question)
		body := strings.TrimSpace(answer.Body)
		if question == "" || body == "" {
			return errors.New("an answer is empty")
		}
		fmt.Fprintf(&b, "%s\n%s\n\n", question, body)
	}
	return os.WriteFile(filepath.Join(dir, "answers.md"), []byte(b.String()), 0o644)
}

func splitAnswer(body string) []string {
	var out []string
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(line), "-*"))
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func (r *runner) spanDir() string { return filepath.Join(r.opt.Root, r.cur.ID) }

func (r *runner) iterDir() string {
	return filepath.Join(r.spanDir(), "iter", strconv.Itoa(r.cur.Iteration))
}

func (r *runner) save() error {
	if err := os.MkdirAll(r.spanDir(), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(r.cur, "", "  ")
	if err != nil {
		return err
	}
	if err := writeAtomic(filepath.Join(r.spanDir(), "cursor.json"), append(raw, '\n')); err != nil {
		return err
	}
	cur := r.cur
	r.publish(Event{Type: "cursor", Cursor: &cur})
	return nil
}

func (r *runner) writeGate(gate qa.Gate) error {
	if err := writeJSON(filepath.Join(r.spanDir(), "qa-gate.json"), gate); err != nil {
		return err
	}
	sealed := gate
	r.publish(Event{Type: "gate", Gate: &sealed})
	return nil
}

func (r *runner) publish(event Event) {
	if r.opt.Watch == nil {
		return
	}
	r.opt.Watch.Publish(r.cur.ID, event)
}

func (r *runner) receiptExists(env, sha string) bool {
	_, err := os.Stat(r.receiptPath(env, sha))
	return err == nil
}

func (r *runner) writeReceipt(env, sha string) error {
	return writeAtomic(r.receiptPath(env, sha), []byte(sha+"\n"))
}

func (r *runner) receiptPath(env, sha string) string {
	return filepath.Join(r.spanDir(), "receipts", env+"-"+sha)
}

func (o Options) now() time.Time {
	if o.Now != nil {
		return o.Now()
	}
	return time.Now().UTC()
}

func next(stage string) string {
	for i, name := range order {
		if name == stage && i+1 < len(order) {
			return order[i+1]
		}
	}
	return note
}

func newID(now time.Time) (string, error) {
	var b [2]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return now.UTC().Format("2006-01-02-1504") + "-" + hex.EncodeToString(b[:]), nil
}

func featurePath(workspace, feature string) string {
	return filepath.Join(workspace, "odd", "tasks", feature+".md")
}

func readInstructions(workspace, spanDir, stage string) (string, string, error) {
	parts := []string{
		filepath.Join(workspace, "span", "stages", stage+".md"),
		filepath.Join(spanDir, "instructions", stage+".md"),
	}
	var b strings.Builder
	for _, path := range parts {
		raw, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return "", "", err
		}
		b.Write(raw)
		if !strings.HasSuffix(string(raw), "\n") {
			b.WriteByte('\n')
		}
	}
	sum := sha256.Sum256([]byte(b.String()))
	if b.Len() == 0 {
		return "", "", nil
	}
	return b.String(), hex.EncodeToString(sum[:]), nil
}

func handoffDigests(spanDir string) ([]string, error) {
	dir := filepath.Join(spanDir, "handoffs")
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		sum := sha256.Sum256(raw)
		out = append(out, entry.Name()+":"+hex.EncodeToString(sum[:]))
	}
	sort.Strings(out)
	return out, nil
}

func handoffNames(spanDir string) ([]string, error) {
	digests, err := handoffDigests(spanDir)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(digests))
	for _, item := range digests {
		names = append(names, strings.SplitN(item, ":", 2)[0])
	}
	return names, nil
}

func manifestDigest(p policy.File, stage string) string {
	raw, err := json.Marshal(p.Manifest[stage])
	if err != nil || string(raw) == "null" {
		return ""
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func manifestText(p policy.File, stage string) string {
	m := p.Manifest[stage]
	if len(m.Skills) == 0 && len(m.Rules) == 0 && len(m.MCP) == 0 {
		return ""
	}
	return fmt.Sprintf("skills: %s\nrules: %s\nmcp: %s", strings.Join(m.Skills, ","), strings.Join(m.Rules, ","), strings.Join(m.MCP, ","))
}

func writeJSON(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(path, append(raw, '\n'))
}

func writeAtomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer func() { _ = os.Remove(tmp) }()
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
