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
	"log"
	"os"
	"path/filepath"
	"regexp"
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
	"span/internal/journal"
	"span/internal/learn"
	"span/internal/policy"
	"span/internal/proc"
	"span/internal/prompt"
	"span/internal/prove"
	"span/internal/qa"
	"span/internal/runlock"
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

// Change is one open change request and the branch it targets. An empty Base means the
// host cannot tell.
type Change struct {
	URL  string
	Base string
}

// Review opens and merges a change request.
type Review interface {
	// Find returns every open change for branch.
	Find(ctx context.Context, spanID, branch string) ([]Change, error)
	// Open creates the change from branch into base.
	Open(ctx context.Context, spanID, branch, base, title, body string) (string, error)
	Merged(ctx context.Context, url string) (bool, error)
	Merge(ctx context.Context, url string) error
}

// baseReporter is a Review that can tell which branch a change targets. A review without it
// is trusted: the merge stage cannot compare what it cannot read.
type baseReporter interface {
	ChangeBase(ctx context.Context, url string) (string, error)
}

// baseFinder is a Source that can tell whether a base branch exists. adapt.Git is one; a
// source without it is trusted to have the base.
type baseFinder interface {
	BaseExists(ctx context.Context, base string) (bool, error)
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
	// HeldWorkspace is a checkout claim the caller already holds. The run uses it instead of
	// taking the checkout again and releases it when the work ends. A launcher takes the
	// claim before it writes anything into the project, so a second launch cannot rewrite
	// span.yaml or the brief under a run that is starting.
	HeldWorkspace *runlock.Lock
	Workspace     string
	Root          string
	Policy        policy.File
	Brief         string
	Issue         string
	Branch        string
	Base          string
	Files         []handoff.File
	Issues        IssueBoard
	Now           func() time.Time
	Agents        Agents
	Check         prove.Checker
	Source        Source
	Review        Review
	Release       Releaser
	Watch         Publisher
	Secrets       []string
	// Worktree is true when the workspace is a linked git worktree, so parallel runs do
	// not share a tree. The prompt says so and the agent may not switch branches.
	Worktree bool
	// HostSettle is how long to wait, between retries, for the host to recompute whether a
	// change merges after a resolved merge commit was pushed to it. Empty means no retry.
	// The host answers "conflict" from its old view for a few seconds after a push.
	HostSettle []time.Duration
	// Project is the checkout the run belongs to. The policy and the Jira account live
	// there, while the code lives in the worktree.
	Project string
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
	ID            string   `json:"id"`
	Iteration     int      `json:"iteration"`
	Stage         string   `json:"stage"`
	Hold          bool     `json:"hold"`
	Stop          bool     `json:"stop"`
	Done          bool     `json:"done"`
	Reason        string   `json:"reason,omitempty"`
	GitSHA        string   `json:"gitSHA,omitempty"`
	ChangeURL     string   `json:"changeURL,omitempty"`
	Pending       *Pending `json:"pending,omitempty"`
	InspectBlocks int      `json:"inspectBlocks"`
	// InspectFeedback is what inspect found wrong when it sent the work back to build. Build
	// reads it on its next round and it is cleared when inspect passes.
	InspectFeedback string `json:"inspectFeedback,omitempty"`
	// Rewound marks a run a person restarted at a stage. The branch may have moved since,
	// so the run reads its commit again before the first stage and then clears this.
	Rewound         bool        `json:"rewound,omitempty"`
	LastDigest      string      `json:"lastDigest,omitempty"`
	FrameAdvancedAt *time.Time  `json:"frameAdvancedAt,omitempty"`
	StartedAt       time.Time   `json:"startedAt"`
	Branch          string      `json:"branch,omitempty"`
	Base            string      `json:"base,omitempty"`
	Workspace       string      `json:"workspace,omitempty"`
	Issue           string      `json:"issue,omitempty"`
	Questions       []string    `json:"questions,omitempty"`
	Repairs         []Repair    `json:"repairs,omitempty"`
	WideSHA         string      `json:"wideSHA,omitempty"`
	Worktree        bool        `json:"worktree,omitempty"`
	Project         string      `json:"project,omitempty"`
	Weight          string      `json:"weight,omitempty"`
	AgentBlock      *AgentBlock `json:"agentBlock,omitempty"`
	// Feature is this run's identity: its feature document odd/tasks/<feature>.md and its
	// commit scope. It comes from the branch, never from span.yaml, so parallel runs in one
	// project never write the same document.
	Feature string `json:"feature,omitempty"`
	// Conflict is a merge of the base into the branch waiting on a person or in progress.
	Conflict *Conflict `json:"conflict,omitempty"`
	// Resolutions counts the conflicts this run resolved; the base cannot move forever.
	Resolutions int `json:"resolutions,omitempty"`
	// Unpushed is a merge commit that is only local. It is pushed right before the host is
	// asked to merge, once prove, QA and inspect passed on it.
	Unpushed bool `json:"unpushed,omitempty"`
}

// AgentBlock is an agent CLI that cannot work until a person acts. It is a hold,
// not a stop: the run continues once the account can run again.
type AgentBlock struct {
	Kind   string    `json:"kind"`
	Agent  string    `json:"agent"`
	Detail string    `json:"detail,omitempty"`
	Time   time.Time `json:"time"`
}

// Repair is one failing check that was sent back to build. The ledger is written
// before the next build starts, so a resume reads the attempts that were already
// spent instead of starting over.
type Repair struct {
	Iteration   int       `json:"iteration"`
	Stage       string    `json:"stage"`
	Fingerprint string    `json:"fingerprint,omitempty"`
	GitSHA      string    `json:"gitSHA,omitempty"`
	Reason      string    `json:"reason,omitempty"`
	Time        time.Time `json:"time"`
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
	Conflict   *Conflict            `json:"conflict,omitempty"`
	// Seq orders the live frames of one run so the page can tell a duplicate or a gap
	// from the next frame. Zero means the event was not published through the hub.
	Seq uint64 `json:"seq,omitempty"`
	// Offset is the size of transcript.log once the lines of this frame are in it, and on
	// a replay the size its transcript tail was read up to. The page drops a transcript
	// frame at or below the replay's offset: its lines are already in the tail.
	Offset int64 `json:"offset,omitempty"`
	// Start is the size of transcript.log before the first line of a transcript frame. It
	// is a pointer so a batch at the very start of the file still says 0. The socket uses
	// it to cut the lines a connection's replay tail already holds.
	Start *int64 `json:"start,omitempty"`
}

// Snapshot is the durable reading of one run.
type Snapshot struct {
	Cursor      Cursor                `json:"cursor"`
	Statements  []statement.Statement `json:"statements"`
	Acceptance  []qa.Line             `json:"acceptance"`
	Gate        *qa.Gate              `json:"gate,omitempty"`
	StageAgents map[string]string     `json:"stageAgents,omitempty"`
	Label       string                `json:"label,omitempty"`
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
		Issue: strings.TrimSpace(opt.Issue), Worktree: opt.Worktree, Project: opt.Project,
		Feature: branch.FeatureOf(opt.Branch),
	}
	r := &runner{opt: opt, cur: cur}
	if err := os.MkdirAll(r.spanDir(), 0o755); err != nil {
		return "", err
	}
	if err := ignoreStateRoot(opt.Root); err != nil {
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
	r, release, err := acquire(opt, id)
	if err != nil {
		return Result{}, err
	}
	defer release()
	if r.cur.Stop {
		return r.finish(statement.Stop), nil
	}
	if r.cur.Done {
		return r.finish(statement.Advance), nil
	}
	clearBusyHold(&r.cur)
	r.takePause()
	if err := r.save(); err != nil {
		return Result{}, err
	}
	return r.run(ctx)
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
	r, release, err := acquire(opt, id)
	if err != nil {
		return Result{}, err
	}
	defer release()
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
	return r.run(ctx)
}

// Reload replaces the brief with the story and runs the waiting span again.
func Reload(ctx context.Context, opt Options, id, brief string) (Result, error) {
	r, release, err := acquire(opt, id)
	if err != nil {
		return Result{}, err
	}
	defer release()
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
	return r.run(ctx)
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
		if cur.Base != "" {
			opt.Base = cur.Base
		}
	}
	if strings.TrimSpace(opt.Issue) == "" {
		opt.Issue = cur.Issue
	}
	if cur.Issue == "" {
		cur.Issue = IssueKey(opt.Brief)
	}
	// A cursor written before runs carried their feature, or one edited into a path, takes
	// it from its branch. A run with no recorded branch settles it when it runs.
	if !branch.ValidFeature(cur.Feature) {
		cur.Feature = branch.FeatureOf(cur.Branch)
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
	last     failure
}

// failure is what the stage that just ran learned about why it failed. It is set
// by prove and qa and read once by repair.
type failure struct {
	Fingerprint  string
	Inconclusive bool
}

// repair decides whether a failing check starts another build. The loop keeps
// going while each failure is new. The same failure twice means the build did not
// fix it, so it stops instead of spending another run, and the cap stays as the
// ceiling while new failures keep appearing.
func (r *runner) repair(stage, reason string) (string, string) {
	prev := r.lastRepair()
	r.noteRepair(stage, reason)
	if r.last.Inconclusive {
		return statement.Hold, "inconclusive, the same command passed and failed on this commit: " + reason
	}
	if prev != nil && prev.Fingerprint != "" && prev.Fingerprint == r.last.Fingerprint {
		return statement.Hold, "no progress, the same failure came back: " + reason
	}
	// The cap counts repairs, not iterations: a deliberate restart clears the ledger and
	// therefore gets its own budget. Three repairs means the fourth failure holds.
	if cap := r.opt.Policy.RepairCapFor(r.weight()); len(r.cur.Repairs) > cap {
		return statement.Hold, fmt.Sprintf("iteration cap %d: %s", cap, reason)
	}
	return "iterate", reason
}

func (r *runner) noteRepair(stage, reason string) {
	r.cur.Repairs = append(r.cur.Repairs, Repair{
		Iteration: r.cur.Iteration, Stage: stage, Fingerprint: r.last.Fingerprint,
		GitSHA: r.cur.GitSHA, Reason: reason, Time: r.opt.now(),
	})
}

func (r *runner) lastRepair() *Repair {
	if len(r.cur.Repairs) == 0 {
		return nil
	}
	return &r.cur.Repairs[len(r.cur.Repairs)-1]
}

// stamp writes each command's full output to the run log and keeps a short tail on
// the statement. The log is what a reader opens; the tail is what a prompt carries.
func (r *runner) stamp(stage string, commands []statement.Command) []statement.Command {
	out := make([]statement.Command, len(commands))
	for i, item := range commands {
		if item.Output != "" {
			if path, err := r.writeLog(stage, i, item.Output); err == nil {
				item.LogPath = path
			}
			item.Output = statement.Tail(item.Output)
		}
		out[i] = item
	}
	return out
}

func (r *runner) writeLog(stage string, index int, text string) (string, error) {
	dir := filepath.Join(r.iterDir(), "logs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	name := fmt.Sprintf("%s-%d.log", stage, index+1)
	if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o644); err != nil {
		return "", err
	}
	return filepath.ToSlash(filepath.Join("iter", strconv.Itoa(r.cur.Iteration), "logs", name)), nil
}

// reapOrphan ends an agent a killed Span left running in this tree. The tree is locked, so
// nothing alive owns it, and it would keep editing files under the agent this run starts.
func (r *runner) reapOrphan() {
	tree := r.cur.Workspace
	if tree == "" {
		tree = r.opt.Workspace
	}
	if tree != "" && proc.Reap(filepath.Join(tree, ".span")) {
		r.journal("agent.reaped", nil)
		log.Printf("run %s: ended an agent left running by a Span that was killed", r.cur.ID)
	}
}

// run walks the stages and, when the loop fails, leaves the run held with the reason.
// A run that died mid-stage with hold=false looks identical to a running one on the
// page, so the person is offered nothing to do. A hold with the error is honest.
func (r *runner) run(ctx context.Context) (Result, error) {
	release, busy := r.lockWorkspace()
	if busy != nil {
		return r.holdBusy(busy), nil
	}
	defer release()
	r.reapOrphan()
	res, err := r.loop(ctx)
	if err != nil {
		r.cur.Hold = true
		r.cur.Reason = "the run stopped: " + err.Error()
		_ = r.save()
	}
	return res, err
}

func (r *runner) loop(ctx context.Context) (Result, error) {
	ctx = agent.WithSpan(ctx, r.cur.ID)
	log, err := OpenTranscript(filepath.Join(r.spanDir(), "transcript.log"), r.opt.Secrets, r.publish)
	if err != nil {
		return Result{}, err
	}
	defer log.Close()
	r.log = log
	if binder, ok := r.opt.Agents.(interface{ Bind(io.Writer) }); ok {
		binder.Bind(log)
	}
	err = r.startBase(ctx)
	if err == nil {
		err = r.ensureBranch(ctx)
	}
	if err == nil {
		err = r.settleFeature(ctx)
	}
	if err != nil {
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
	if err := r.settleRewind(ctx); err != nil {
		return Result{}, err
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
		if r.takePause() && stage != note {
			r.cur.Hold = true
			r.cur.Reason = "paused before " + stage + ". Pick the agent for this stage, then resume."
			if err := r.save(); err != nil {
				return Result{}, err
			}
			return r.finish(statement.Hold), nil
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
			r.cur.AgentBlock = nil
			r.cur.Reason = reason
		case statement.Hold:
			r.cur.Hold = true
			r.cur.Reason = reason
		case statement.Stop:
			r.cur.Stop = true
			r.cur.Reason = reason
		case "iterate":
			decision, why := r.repair(stage, reason)
			r.cur.Reason = why
			if decision == "iterate" {
				r.cur.Iteration++
				r.cur.Stage = build
				r.cur.InspectBlocks = 0
				break
			}
			r.cur.Hold = true
		case "rebuild":
			r.cur.Stage = build
			r.cur.Reason = reason
		case "reprove":
			// The merge changed the head. A new iteration keeps the old statements as history
			// and proves the bytes that will be merged.
			r.cur.Iteration++
			r.cur.Stage = proveSt
			r.cur.InspectBlocks = 0
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
	if verdict == "iterate" || verdict == "rebuild" || verdict == "reprove" {
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
	target, err := r.pick(frame)
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
	call, cancel := r.agentCall(ctx)
	defer cancel()
	doc, err := r.opt.Agents.Frame(call, target, r.opt.Workspace, prompt.Frame(in))
	if err != nil {
		return r.agentBlocked(frame, err)
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
		return r.record(frame, statement.Hold, "frame produced no claims: the agent returned the story with nothing to check. Resume to ask again, or make the story say what must be true when it is done.", nil)
	}
	now := r.opt.now()
	r.cur.FrameAdvancedAt = &now
	return r.record(frame, statement.Advance, fmt.Sprintf("%d claims", len(doc.Claims)), nil)
}

func (r *runner) doBuild(ctx context.Context) (string, string, error) {
	if err := r.recordBaseline(ctx); err != nil {
		return "", "", err
	}
	target, err := r.pick(build)
	if err != nil {
		return r.record(build, statement.Hold, err.Error(), nil)
	}
	in := r.promptInput(build, r.failurePacket())
	call, cancel := r.agentCall(ctx)
	defer cancel()
	if err := r.opt.Agents.Build(call, target, r.opt.Workspace, prompt.Build(in)); err != nil {
		return r.agentBlocked(build, err)
	}
	path := featurePath(r.opt.Workspace, r.cur.Feature)
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

// weight reads the story as it stands now. It is not frozen at intake: frame can install
// criteria, and a brief that grows into twenty of them earns more checks.
func (r *runner) weight() string {
	names, _ := handoffNames(r.spanDir())
	return r.opt.Policy.Weight(len(qa.Lines(r.opt.Brief)), names)
}

// provePlan scopes the focused band to the files that declare this story's criteria.
// The map already exists for QA, so the tool answers what belongs to this story instead
// of asking a person to write a path filter. No declaration means no scope, and the wide
// band keeps deciding.
func (r *runner) provePlan() prove.Plan {
	weight := r.weight()
	plan := prove.Plan{Fast: r.opt.Policy.Prove.Fast, Acceptance: r.opt.Policy.Prove.Acceptance}
	if !policy.WideOn(weight) {
		plan.Fast = nil
	}
	templates := r.opt.Policy.Prove.Focused
	if r.cur.WideSHA != "" && r.cur.WideSHA == r.cur.GitSHA {
		// The whole suite already passed on these exact bytes. Running it again buys
		// nothing and costs its full duration.
		plan.Fast = nil
	}
	files, err := qa.Scope(r.opt.Workspace, qa.Lines(r.opt.Brief))
	if err != nil || len(files) == 0 {
		// Nothing to focus on. A prove pass must never be left with no command at all:
		// the wide band decides again, whatever the weight said.
		plan.Fast = r.opt.Policy.Prove.Fast
		return plan
	}
	joined := strings.Join(files, " ")
	if len(templates) == 0 {
		// Nothing configured: derive the scoped command from the band that was declared
		// for this weight, so a light story still has something to run.
		wide := plan.Fast
		if len(wide) == 0 {
			wide = r.opt.Policy.Prove.Fast
		}
		plan.Focused = prove.DeriveFocus(wide, files)
		return plan
	}
	for _, template := range templates {
		plan.Focused = append(plan.Focused, strings.ReplaceAll(template, "{files}", joined))
	}
	return plan
}

func (r *runner) doProve(ctx context.Context) (string, string, error) {
	plan := r.provePlan()
	plan.Excuse = r.excuse()
	plan = r.focusOnDiff(ctx, plan)
	r.cur.Weight = r.weight()
	rep, err := prove.Bands(ctx, r.opt.Check, r.opt.Workspace, plan)
	if err != nil {
		return "", "", err
	}
	if plan.Runs() && len(plan.Fast) > 0 && rep.Outcome == statement.Advance && r.cur.GitSHA != "" {
		r.cur.WideSHA = r.cur.GitSHA
	}
	r.last = failure{Fingerprint: rep.Fingerprint}
	return r.record(proveSt, rep.Outcome, rep.Reason, func(s *statement.Statement) {
		s.Commands = r.stamp(proveSt, rep.Commands)
		s.GitSHA = r.cur.GitSHA
		s.Attributable = rep.Outcome == "iterate"
		s.Fingerprint = rep.Fingerprint
		s.Weight = r.cur.Weight
		s.Cap = r.opt.Policy.RepairCapFor(r.cur.Weight)
	})
}

// agentCall bounds one agent call by limits.stageTimeout. Without a limit the call keeps the
// run's own context.
func (r *runner) agentCall(ctx context.Context) (context.Context, context.CancelFunc) {
	if limit := r.opt.Policy.StageTimeout(); limit > 0 {
		return context.WithTimeout(ctx, limit)
	}
	return ctx, func() {}
}

// agentBlocked records an agent that ran out of quota or credentials. The reason
// names the action a person has to take, because nothing in the project can fix it.
func (r *runner) agentBlocked(stage string, err error) (string, string, error) {
	if limit := r.opt.Policy.StageTimeout(); limit > 0 && errors.Is(err, context.DeadlineExceeded) {
		// Not an account problem and not a failing check: the agent did not finish, so no
		// repair is spent. A person decides between more time and a smaller story.
		return r.record(stage, statement.Hold, fmt.Sprintf(
			"%s ran past limits.stageTimeout (%s) and was stopped. Raise the limit or split the story, then resume.", stage, limit), nil)
	}
	blocked, ok := agent.AsBlocked(err)
	if !ok {
		return r.record(stage, statement.Hold, err.Error(), nil)
	}
	r.cur.AgentBlock = &AgentBlock{
		Kind: blocked.Kind, Agent: blocked.Agent, Detail: blocked.Detail, Time: r.opt.now(),
	}
	var why string
	switch blocked.Kind {
	case agent.Credentials:
		why = fmt.Sprintf("agent credentials missing (%s): %s. Log in to that CLI, then resume.", blocked.Agent, blocked.Detail)
	default:
		why = fmt.Sprintf("agent quota exhausted (%s): %s. Resume after the limit resets, or set agent: to another CLI.", blocked.Agent, blocked.Detail)
	}
	return r.record(stage, statement.Hold, why, func(s *statement.Statement) {
		s.Fingerprint = statement.Signature(blocked.Agent, blocked.Kind, blocked.Detail)
		s.Cap = r.opt.Policy.RepairCap()
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
	covered, coverErr := qa.Covered(r.opt.Workspace, lines)
	if coverErr != nil {
		return "", "", coverErr
	}
	if command == "" || prove.Prose(command) {
		gate := qa.Gate{Command: command, Verdict: qa.Unverified, Checks: make([]qa.Check, 0, len(lines))}
		for _, line := range lines {
			gate.Checks = append(gate.Checks, qa.Check{ID: line.ID, Result: qa.Unverified})
		}
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
	prev, hadProve, _ := statement.Read(r.iterDir(), proveSt)
	if hadProve && r.reusableQA(prev, command, ctx) {
		gate := qa.Judge(lines, 0, covered, qa.Outcomes{})
		gate.Command = command
		gate.Evidence = "prove"
		// Only a pass is explained by the reuse. When the gate does not pass, its own reason is
		// the one a person can act on (which criteria no test declares), so it stays.
		if gate.Verdict == qa.Pass {
			gate.Reason = command + " already passed on this commit in a clean tree; it was not run again"
		} else {
			gate.Reason += " (" + command + " already passed on this commit and was not run again)"
		}
		if err := r.writeGate(gate); err != nil {
			return "", "", err
		}
		fill := func(s *statement.Statement) {
			s.Commands = []statement.Command{{Command: command, Band: "qa"}}
			s.GitSHA = r.cur.GitSHA
			s.Fingerprint = statement.Signature(command, gate.Reason)
			s.Cap = r.opt.Policy.RepairCap()
		}
		if gate.Verdict == qa.Pass {
			return r.record(qaSt, statement.Advance, "acceptance gate passed; "+command+" was not run again", fill)
		}
		r.last = failure{Fingerprint: statement.Signature(command, gate.Reason)}
		return r.record(qaSt, "iterate", gate.Reason, fill)
	}
	started := r.opt.now()
	out, runErr := r.opt.Check.Run(ctx, command, r.opt.Workspace)
	item := statement.Command{Command: command, Band: "qa", ExitCode: out.ExitCode, Output: out.Text}
	if runErr != nil {
		item.Error = runErr.Error()
		return r.record(qaSt, statement.Hold, command+" did not run: "+runErr.Error(), func(s *statement.Statement) {
			s.Commands = r.stamp(qaSt, []statement.Command{item})
		})
	}
	outcomes, reportPath, findErr := qa.Find(r.opt.Workspace, r.opt.Policy.QA.Report, started)
	if findErr != nil {
		return "", "", findErr
	}
	greenBefore := hadProve && prev.GitSHA != "" && prev.GitSHA == r.cur.GitSHA && provePassed(prev, command)
	if out.ExitCode != 0 && !greenBefore {
		if verdict, reason, ok, err := r.excusedQA(lines, covered, outcomes, reportPath, item, out); ok || err != nil {
			return verdict, reason, err
		}
	}
	gate := qa.Judge(lines, out.ExitCode, covered, outcomes)
	gate.Command = command
	gate.Report = reportPath
	if err := r.writeGate(gate); err != nil {
		return "", "", err
	}
	r.last = failure{
		Fingerprint:  statement.Signature(command, gate.Verdict, failingIDs(gate), prove.Fingerprint(command, out.Text)),
		Inconclusive: out.ExitCode != 0 && greenBefore,
	}
	fill := func(s *statement.Statement) {
		s.Commands = r.stamp(qaSt, []statement.Command{item})
		s.GitSHA = r.cur.GitSHA
		s.Fingerprint = r.last.Fingerprint
		s.Inconclusive = r.last.Inconclusive
		s.Cap = r.opt.Policy.RepairCap()
	}
	if out.ExitCode == 0 && gate.Verdict == qa.Pass {
		return r.record(qaSt, statement.Advance, "acceptance gate passed", fill)
	}
	return r.record(qaSt, "iterate", "acceptance gate "+gate.Verdict+": "+gate.Reason, fill)
}

// reusableQA reports whether this exact command already passed on this commit in a
// clean tree. Re-running it would verify the same bytes a second time and turn a
// flaky test into a red gate.
func (r *runner) reusableQA(prev statement.Statement, command string, ctx context.Context) bool {
	if prev.Verdict != statement.Advance || prev.GitSHA == "" || prev.GitSHA != r.cur.GitSHA {
		return false
	}
	if !provePassed(prev, command) {
		return false
	}
	dirty, err := r.opt.Source.Dirty(ctx)
	return err == nil && !dirty
}

func provePassed(prev statement.Statement, command string) bool {
	for _, item := range prev.Commands {
		if item.Command == command && item.ExitCode == 0 && item.Error == "" {
			return true
		}
	}
	return false
}

func failingIDs(gate qa.Gate) string {
	var ids []string
	for _, check := range gate.Checks {
		if check.Result == qa.Fail {
			ids = append(ids, check.ID)
		}
	}
	return strings.Join(ids, " ")
}

func (r *runner) doInspect(ctx context.Context) (string, string, error) {
	if r.cur.InspectBlocks >= maxInspectBlocks {
		// The run held at the cap and a person resumed it. That is a request for another round:
		// inspect already said what is wrong, so build gets that instead of the same verdict
		// again, and the count starts over. Each round still needs a person to ask for it.
		reason := strings.TrimPrefix(r.cur.Reason, inspectCap)
		verdict, why, err := r.sendBack(reason)
		if err == nil {
			r.cur.InspectBlocks = 0
		}
		return verdict, why, err
	}
	target, err := r.pick(inspect)
	if err != nil {
		return r.record(inspect, statement.Hold, err.Error(), nil)
	}
	call, cancel := r.agentCall(ctx)
	defer cancel()
	doc, err := r.opt.Agents.Inspect(call, target, r.opt.Workspace, prompt.Inspect(r.promptInput(inspect, "")))
	if err != nil {
		return r.agentBlocked(inspect, err)
	}
	if !doc.Blocking {
		r.cur.InspectFeedback = ""
		return r.record(inspect, statement.Advance, doc.Reason, nil)
	}
	r.cur.InspectBlocks++
	if r.cur.InspectBlocks >= maxInspectBlocks {
		return r.record(inspect, statement.Hold, inspectCap+doc.Reason, nil)
	}
	return r.sendBack(doc.Reason)
}

// maxInspectBlocks is how many times inspect may send the work back to build before the run
// holds for a person.
const maxInspectBlocks = 3

// inspectCap starts the hold reason of a run that reached maxInspectBlocks.
const inspectCap = "inspect cap: "

// sendBack archives this round's build, prove, QA and inspect statements as history and
// returns the verdict that makes build try again with the inspector's reason.
func (r *runner) sendBack(reason string) (string, string, error) {
	r.cur.InspectFeedback = reason
	dir := r.iterDir()
	suffix := fmt.Sprintf("block-%d", r.cur.InspectBlocks)
	for _, stage := range []string{build, proveSt, qaSt} {
		if err := statement.Archive(dir, stage, suffix); err != nil {
			return "", "", err
		}
	}
	if _, _, err := r.record(inspect, statement.Stop, reason, nil); err != nil {
		return "", "", err
	}
	if err := statement.Archive(dir, inspect, suffix); err != nil {
		return "", "", err
	}
	return "rebuild", reason, nil
}

func (r *runner) doOpen(ctx context.Context) (string, string, error) {
	if r.gated(r.opt.Policy.OpenMode(), open, "") {
		return r.record(open, statement.Hold, "open is hold", nil)
	}
	branch, err := r.opt.Source.Branch(ctx)
	if err != nil {
		return r.record(open, statement.Hold, err.Error(), nil)
	}
	base := r.runBase()
	// Without a base the host picks the repository's default branch, which is how a change
	// meant for a feature branch lands on main.
	if strings.TrimSpace(base) == "" {
		return r.record(open, statement.Hold, noBase, nil)
	}
	// Checked again here, before the push: the base may have gone since the run started.
	if err := r.checkBase(ctx, branch); err != nil {
		return r.record(open, statement.Hold, err.Error(), nil)
	}
	r.cur.Pending = &Pending{Kind: open, Branch: branch, SHA: r.cur.GitSHA}
	if err := r.save(); err != nil {
		return "", "", err
	}
	changes, err := r.opt.Review.Find(ctx, r.cur.ID, branch)
	if err != nil {
		return r.record(open, statement.Hold, "change lookup failed: "+err.Error(), nil)
	}
	url, reuse := onBase(changes, base)
	if !reuse && len(changes) > 0 {
		// The changes are someone's decision now; retargeting or closing them is theirs too.
		r.cur.Pending = nil
		return r.record(open, statement.Hold, wrongBase(changes, base), nil)
	}
	if !reuse {
		if err := r.opt.Source.Push(ctx); err != nil {
			return r.record(open, statement.Hold, err.Error(), nil)
		}
		url, err = r.opt.Review.Open(ctx, r.cur.ID, branch, base, storyTitle(r.opt.Brief, r.cur.Feature), r.opt.Brief)
		if err != nil {
			return r.record(open, statement.Hold, "open outcome unknown: "+err.Error(), nil)
		}
	}
	r.cur.ChangeURL = url
	r.cur.Pending = nil
	r.spendApproval(open, "")
	return r.record(open, statement.Advance, "change open", func(s *statement.Statement) {
		s.ChangeURL = url
		s.GitSHA = r.cur.GitSHA
	})
}

func (r *runner) doMerge(ctx context.Context) (string, string, error) {
	if r.cur.ChangeURL == "" {
		return r.record(merge, statement.Hold, "no change url", nil)
	}
	if r.cur.Conflict != nil {
		return r.continueConflict(ctx)
	}
	if r.gated(r.opt.Policy.MergeMode(), merge, "") {
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
	if reason, got := r.otherBase(ctx, merged); reason != "" {
		if !r.approved(merge, "") {
			r.cur.Pending = nil
			return r.record(merge, statement.Hold, reason, nil)
		}
		// The person approved merging where the change is, so the conflict check that
		// follows must read that branch, not the base the host moved the change off.
		if !merged && branch.CheckBase(got) == nil {
			r.cur.Base = got
			if err := r.save(); err != nil {
				return "", "", err
			}
		}
	}
	note := ""
	if !merged {
		// A conflict with the base is found here, without touching the checkout, instead of
		// being a refusal from the host. A clean base is left to the host, as before.
		verdict, reason, handled, why, err := r.preflightBase(ctx)
		if err != nil || handled {
			return verdict, reason, err
		}
		note = why
		pushed := false
		if r.cur.Unpushed {
			// A resolved merge commit is published only now, once it passed prove, QA and
			// inspect, so a failing check never reaches the change.
			if err := r.opt.Source.Push(ctx); err != nil {
				return r.record(merge, statement.Hold, "the merge commit was not pushed: "+err.Error()+". Resume to push it again.", nil)
			}
			r.cur.Unpushed = false
			pushed = true
			if err := r.save(); err != nil {
				return "", "", err
			}
		}
		if err := r.mergeSettling(ctx, pushed); err != nil {
			if errors.Is(err, ErrConflict) {
				// The base moved after the check above. Nothing was merged, so this is known.
				return r.refused(ctx, err)
			}
			if errors.Is(err, ErrBehind) {
				// Nothing was merged: the host wants the change updated first.
				r.cur.Pending = nil
				return r.record(merge, statement.Hold, BehindReason, nil)
			}
			return r.record(merge, statement.Hold, "merge outcome unknown: "+err.Error(), nil)
		}
	}
	r.cur.Pending = nil
	r.spendApproval(merge, "")
	reason := "merged"
	if note != "" {
		reason += " (" + note + ")"
	}
	return r.record(merge, statement.Advance, reason, func(s *statement.Statement) {
		s.ChangeURL = r.cur.ChangeURL
	})
}

// DefaultHostSettle is how the launcher waits for a host to catch up with a push: about
// half a minute in four steps, which is longer than GitHub took in practice.
var DefaultHostSettle = []time.Duration{2 * time.Second, 4 * time.Second, 8 * time.Second, 12 * time.Second}

// mergeSettling asks the host to merge. Right after a resolved merge commit was pushed, the
// host may still answer from the branch as it was and call the change conflicting. Only then
// is that answer retried, a few times, so a refusal that is really stale does not become a
// hold that says the base merges cleanly. A conflict that does not clear is returned as it
// is; so is any other refusal, and a stop ends the wait.
func (r *runner) mergeSettling(ctx context.Context, pushed bool) error {
	err := r.opt.Review.Merge(ctx, r.cur.ChangeURL)
	if !pushed || !errors.Is(err, ErrConflict) {
		return err
	}
	for _, wait := range r.opt.HostSettle {
		select {
		case <-ctx.Done():
			return err
		case <-time.After(wait):
		}
		err = r.opt.Review.Merge(ctx, r.cur.ChangeURL)
		if !errors.Is(err, ErrConflict) {
			return err
		}
	}
	return err
}

// otherBase is the hold reason when the change targets a branch other than this run's base,
// or "" when it matches or the host cannot tell. A merged change cannot be retargeted, so
// the person is asked to look at the branch it landed on and approve the stage to go on.
func (r *runner) otherBase(ctx context.Context, merged bool) (string, string) {
	reporter, ok := r.opt.Review.(baseReporter)
	expected := r.runBase()
	if !ok || strings.TrimSpace(expected) == "" {
		return "", ""
	}
	got, err := reporter.ChangeBase(ctx, r.cur.ChangeURL)
	got = strings.TrimSpace(got)
	if err != nil || got == "" || got == expected {
		return "", ""
	}
	if merged {
		return fmt.Sprintf("the change %s was merged into %s, not into this run's base %s. Check %s for changes it should not have, then approve this stage to close the run", strings.TrimSpace(r.cur.ChangeURL), got, expected, got), got
	}
	return fmt.Sprintf("the change %s targets %s but this run's base is %s. Retarget it (gh pr edit %s --base %s), or approve this stage to merge it into %s", strings.TrimSpace(r.cur.ChangeURL), got, expected, prRef(r.cur.ChangeURL), expected, got), got
}

func (r *runner) doPromote(ctx context.Context) (string, string, error) {
	var done []string
	for _, env := range r.opt.Policy.Promote.Environments {
		mode := env.Mode
		if mode == "" {
			mode = "auto"
		}
		if r.gated(mode, promote, env.Name) {
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
	for _, env := range r.opt.Policy.Promote.Environments {
		r.spendApproval(promote, env.Name)
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

// settleRewind reads the commit the branch is on after a person restarted the run. Only build
// writes the cursor's commit, so a restart at prove or later would keep checking and recording
// the commit from before the person's own edits. A restart at build commits on its own.
func (r *runner) settleRewind(ctx context.Context) error {
	if !r.cur.Rewound {
		return nil
	}
	if r.cur.Stage != build {
		sha, err := r.opt.Source.Head(ctx)
		if err != nil {
			return err
		}
		r.cur.GitSHA = sha
	}
	r.cur.Rewound = false
	return r.save()
}

func (r *runner) commit(ctx context.Context) (string, error) {
	dirty, err := r.opt.Source.Dirty(ctx)
	if err != nil {
		return "", err
	}
	if !dirty {
		return r.opt.Source.Head(ctx)
	}
	return r.opt.Source.Commit(ctx, fmt.Sprintf("feat(%s): span %s", r.cur.Feature, r.cur.ID))
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
	r.journal("stage", map[string]string{
		"stage": stage, "verdict": stored, "iteration": strconv.Itoa(r.cur.Iteration), "digest": s.Digest,
	})
	r.cur.LastDigest = s.Digest
	r.noteLesson(stage, stored, reason)
	return verdict, reason, nil
}

// journal adds one line to the run's event journal. A journal that cannot be written is
// logged and does not stop the run: the statement it mirrors is already sealed.
func (r *runner) journal(kind string, detail map[string]string) {
	if err := journal.Append(r.spanDir(), journal.Span, kind, detail); err != nil {
		log.Printf("journal %s: %v", r.cur.ID, err)
	}
}

func (r *runner) ensureBranch(ctx context.Context) error {
	if r.opt.Branch == "" {
		return nil
	}
	if !branch.Valid(r.opt.Branch, r.opt.Base) {
		return fmt.Errorf("branch %q on %q is not a feat, fix or hotfix branch on a valid base", r.opt.Branch, r.opt.Base)
	}
	r.cur.Branch = r.opt.Branch
	r.cur.Base = r.opt.Base
	return r.opt.Source.EnsureBranch(ctx, r.opt.Base, r.opt.Branch)
}

// baseExists fails with branch.MissingBase when the source can tell the base is gone.
func (r *runner) baseExists(ctx context.Context, base string) error {
	finder, ok := r.opt.Source.(baseFinder)
	if !ok {
		return nil
	}
	found, err := finder.BaseExists(ctx, base)
	if err != nil {
		return fmt.Errorf("base branch %s could not be checked: %w", base, err)
	}
	if !found {
		return branch.MissingBase{Name: base}
	}
	return nil
}

// noBase is the hold for a run that has no base to open its change into.
const noBase = "this run has no base branch, so its change would target the repository's default branch; start it again with a base"

// runBase is the base this run opens its change into: the cursor's, else the options'.
func (r *runner) runBase() string {
	if r.cur.Base != "" {
		return r.cur.Base
	}
	return r.opt.Base
}

// checkBase is every check a base must pass before work starts and again before the change
// is opened: a valid branch name (never an option, HEAD or a commit, so it is safe to hand
// to git), not the branch the run works on, and present where the change will be opened.
// The error is the hold reason.
func (r *runner) checkBase(ctx context.Context, head string) error {
	base := r.runBase()
	if err := branch.CheckBase(base); err != nil {
		return err
	}
	if head != "" && base == head {
		return fmt.Errorf("the base branch %s is the branch itself; a change cannot target its own head", base)
	}
	return r.baseExists(ctx, base)
}

// startBase checks the base before the first stage, so a missing base or a run started on
// its own base fails fast instead of after the build. A run with no base at all still
// works and holds at open; a run whose change is already open is past this question.
func (r *runner) startBase(ctx context.Context) error {
	if r.cur.Done || r.cur.ChangeURL != "" || strings.TrimSpace(r.runBase()) == "" {
		return nil
	}
	head := r.opt.Branch
	if head == "" {
		current, err := r.opt.Source.Branch(ctx)
		if err != nil {
			return err
		}
		head = current
	}
	return r.checkBase(ctx, head)
}

// onBase picks the open change to reuse: one that targets base, or one whose base the host
// cannot tell (a local record).
func onBase(changes []Change, base string) (string, bool) {
	for _, change := range changes {
		if change.Base == base || change.Base == "" {
			return change.URL, true
		}
	}
	return "", false
}

var pullNumber = regexp.MustCompile(`/pull/(\d+)`)

// prRef is what gh pr edit takes for url: the pull request number when the url has one.
func prRef(url string) string {
	if match := pullNumber.FindStringSubmatch(url); match != nil {
		return match[1]
	}
	return url
}

// wrongBase is the hold for changes that already exist but target other branches. It names
// each one and the command that retargets the first.
func wrongBase(changes []Change, expected string) string {
	first := changes[0]
	if len(changes) == 1 {
		return fmt.Sprintf("the open change %s targets %s but this run's base is %s; retarget it (gh pr edit %s --base %s) or close it, then resume", first.URL, first.Base, expected, prRef(first.URL), expected)
	}
	named := make([]string, 0, len(changes))
	for _, change := range changes {
		named = append(named, fmt.Sprintf("%s (targets %s)", change.URL, change.Base))
	}
	return fmt.Sprintf("the open changes %s do not target this run's base %s; retarget one (gh pr edit %s --base %s) and close the others, then resume", strings.Join(named, ", "), expected, prRef(first.URL), expected)
}

// failurePacket is what build needs to fix the check that failed. It is built from
// the statement already on disk: no model, no second suite, no extra command.
func (r *runner) failurePacket() string {
	var b strings.Builder
	if r.cur.InspectFeedback != "" {
		fmt.Fprintf(&b, "Inspect sent this change back.\nReason: %s\n", r.cur.InspectFeedback)
	}
	if r.cur.Iteration <= 1 {
		return closePacket(&b)
	}
	dir := filepath.Join(r.spanDir(), "iter", strconv.Itoa(r.cur.Iteration-1))
	for _, stage := range []string{proveSt, qaSt} {
		prev, ok, err := statement.Read(dir, stage)
		if err != nil || !ok || prev.Verdict == statement.Advance {
			continue
		}
		fmt.Fprintf(&b, "The %s check failed.\nReason: %s\n", stage, prev.Reason)
		if prev.Cap > 0 {
			fmt.Fprintf(&b, "Attempt %d of %d.\n", prev.Iteration, prev.Cap)
		}
		for _, item := range prev.Commands {
			if (item.ExitCode == 0 && item.Error == "") || item.Preexisting {
				continue
			}
			fmt.Fprintf(&b, "Command: %s\nExit: %d\n", item.Command, item.ExitCode)
			if item.Error != "" {
				fmt.Fprintf(&b, "Error: %s\n", item.Error)
			}
			if item.LogPath != "" {
				fmt.Fprintf(&b, "Full log: %s\n", filepath.Join(r.spanDir(), filepath.FromSlash(item.LogPath)))
			}
			if item.Output != "" {
				fmt.Fprintf(&b, "Last lines:\n%s\n", strings.TrimRight(item.Output, "\n"))
			}
		}
	}
	return closePacket(&b)
}

// closePacket ends a failure packet with the instruction to fix only that, or returns "" when
// there was nothing to report.
func closePacket(b *strings.Builder) string {
	if b.Len() == 0 {
		return ""
	}
	b.WriteString("Fix this failure. Do not restart the feature, and do not widen the change beyond what the failure needs.\n")
	return b.String()
}

func (r *runner) promptInput(stage, failure string) prompt.Input {
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
		Stage: stage, FeaturePath: featurePath(r.opt.Workspace, r.cur.Feature),
		Brief: r.opt.Brief, Instructions: text, Answers: readAnswers(r.spanDir()),
		Handoffs: handoff.Paths(r.cur.ID, names),
		Failure:  failure, Manifest: manifest,
		Standards: harness.Catalog(root), Acceptance: ids, Browser: browser,
		Lessons: lessons,
	}
}

// lessonBook reads the project ledger once. A broken file stays untouched.
func (r *runner) lessonBook() *learn.Book {
	if r.book != nil {
		return r.book
	}
	r.bookPath = r.lessonPath()
	book, err := learn.Load(r.bookPath)
	if err != nil {
		r.book = &learn.Book{}
		r.bookPath = ""
		return r.book
	}
	r.book = book
	return r.book
}

// lessonPath is the project's book. Runs in their own worktrees share it, so it lives in
// the project and not in a tree that goes away with its run.
func (r *runner) lessonPath() string {
	home := r.opt.Project
	if home == "" {
		home = r.opt.Workspace
	}
	return filepath.Join(home, ".span", "lessons.json")
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
	if r.bookPath == "" {
		book.Observe(stage, verdict, reason, shown)
		return
	}
	r.saveLesson(book, stage, verdict, reason, shown)
}

// saveLesson applies one observation to the book on disk, not to the copy this run read when
// it started. Other runs of the project write the same file; replaying the observation on a
// fresh read under a short lock keeps what they added. When the lock or the read fails the
// lesson is dropped: it is a hint, and a verdict never waits on it.
func (r *runner) saveLesson(own *learn.Book, stage, verdict, reason string, shown []string) {
	lock, err := runlock.Acquire(runlock.Lessons, r.bookPath+".lock", "lessons "+r.cur.ID)
	for attempt := 0; err != nil && attempt < 200; attempt++ {
		var busy *runlock.BusyError
		if !errors.As(err, &busy) {
			break
		}
		time.Sleep(5 * time.Millisecond)
		lock, err = runlock.Acquire(runlock.Lessons, r.bookPath+".lock", "lessons "+r.cur.ID)
	}
	if err != nil {
		own.Observe(stage, verdict, reason, shown)
		return
	}
	defer lock.Release()
	fresh, err := learn.Load(r.bookPath)
	if err != nil {
		return
	}
	if fresh.Observe(stage, verdict, reason, shown) {
		_ = learn.Save(r.bookPath, fresh)
	}
	r.book = fresh
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

// OpenTranscript opens a run's transcript and publishes each batch as a transcript
// event carrying its byte range: Start before its first line, Offset after its last.
// The run uses it, and the socket tests drive the same wiring.
func OpenTranscript(path string, secrets []string, publish func(Event)) (*transcript.Log, error) {
	return transcript.Open(path, secrets, func(batch transcript.Batch) {
		start := batch.Start
		publish(Event{Type: "transcript", Stage: batch.Stage, Lines: batch.Lines, Offset: batch.End, Start: &start})
	})
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

// settleFeature gives a run that has no feature yet the one its branch names: the recorded
// branch, else the checked-out one. A branch that names nothing usable falls back to the run
// id, which is unique too. It is saved, so the run keeps it on every resume.
func (r *runner) settleFeature(ctx context.Context) error {
	if branch.ValidFeature(r.cur.Feature) {
		return nil
	}
	name := r.cur.Branch
	if name == "" {
		current, err := r.opt.Source.Branch(ctx)
		if err != nil {
			return fmt.Errorf("the run's branch could not be read to name its feature document: %w", err)
		}
		name = current
	}
	r.cur.Feature = branch.FeatureOf(name)
	if r.cur.Feature == "" {
		r.cur.Feature = r.cur.ID
	}
	return r.save()
}

// maxTitle is how long a pull request title may be.
const maxTitle = 100

// storyTitle is the pull request title: the first non-empty line of the brief without its
// Markdown heading marks or control characters, cut to maxTitle runes. A first line that is
// only a story key (SAL-12) is joined with the next one, so the title says what the story is.
// A brief with no such line uses fallback.
func storyTitle(brief, fallback string) string {
	title := ""
	for _, line := range strings.Split(brief, "\n") {
		line = titleLine(line)
		if line == "" {
			continue
		}
		if title != "" {
			title += " " + line
			break
		}
		title = line
		if !storyKey.MatchString(line) {
			break
		}
	}
	if title == "" {
		return fallback
	}
	if runes := []rune(title); len(runes) > maxTitle {
		title = strings.TrimSpace(string(runes[:maxTitle]))
	}
	return title
}

// storyKey is a line that is only a story key.
var storyKey = regexp.MustCompile(`^[A-Z][A-Z0-9]+-\d+$`)

// titleLine is one brief line as a title: a tab is a space, other control characters go,
// and the Markdown heading marks are trimmed.
func titleLine(line string) string {
	line = strings.Map(func(r rune) rune {
		switch {
		case r == '\t':
			return ' '
		case unicode.IsControl(r):
			return -1
		}
		return r
	}, line)
	return strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(line), "#"))
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
