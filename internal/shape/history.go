package shape

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// personActor is the actor of a step done by the person; steps of the agent carry its name.
const personActor = ActorPerson

// undoLogFile records which steps each undo put back, next to the revisions it belongs to.
const undoLogFile = "undo.jsonl"

// Kinds of a history step.
const (
	// KindIdea is the idea the session started from.
	KindIdea = "idea"
	// KindDraft is the agent adding rules and examples to the map.
	KindDraft = "draft"
	// KindAsk is the agent adding questions only the person can settle.
	KindAsk = "ask"
	// KindAnswer is the person answering a question.
	KindAnswer = "answer"
	// KindAssume is the person accepting the assumption of a question.
	KindAssume = "assume"
	// KindSplitKeep is the person keeping the story whole.
	KindSplitKeep = "split_keep"
	// KindSplitAccept is the person accepting a split into smaller stories.
	KindSplitAccept = "split_accept"
	// KindContextAccept is the person writing a note to the product context.
	KindContextAccept = "context_accept"
	// KindContextSkip is the person dropping a note.
	KindContextSkip = "context_skip"
	// KindContextRestore is the person bringing a skipped note back.
	KindContextRestore = "context_restore"
	// KindSeal is the story being sealed.
	KindSeal = "seal"
	// KindJira is the story being created in Jira.
	KindJira = "jira"
	// KindUndo is the person undoing earlier steps.
	KindUndo = "undo"
	// KindUpdate is a change the history can name but not put back, such as a restore.
	KindUpdate = "update"
)

// Undo operations: what putting a step back changes, kept with the step.
const (
	opRemoveRule      = "remove_rule"
	opRemoveExample   = "remove_example"
	opRemoveQuestion  = "remove_question"
	opSetQuestion     = "set_question"
	opSetSplitKept    = "set_split_kept"
	opSetProposal     = "set_proposal"
	opUnacceptContext = "unaccept_context"
)

// Op is one change that puts back what a step did. The fields it needs depend on Op.
type Op struct {
	Op       string `json:"op"`
	ID       string `json:"id"`
	Status   string `json:"status,omitempty"`
	Answer   string `json:"answer,omitempty"`
	AnswerID string `json:"answerId,omitempty"`
	Flag     bool   `json:"flag,omitempty"`
	Section  string `json:"section,omitempty"`
	Text     string `json:"text,omitempty"`
}

// Event is one step of a session: who did it, what it touched, and how to put it back.
type Event struct {
	ID      string    `json:"id"`
	At      time.Time `json:"at"`
	Actor   string    `json:"actor"`
	Kind    string    `json:"kind"`
	Summary string    `json:"summary"`
	// Ref says what the step was about: "Question 2", "Map", "Split".
	Ref string `json:"ref"`
	// Gives are the items the step made, so a later step can depend on them: an answer id, a
	// rule, example or question id.
	Gives []string `json:"gives,omitempty"`
	// Needs are the items the step was built from; it cannot stand once they are put back.
	Needs []string `json:"needs,omitempty"`
	Undo  []Op     `json:"undo,omitempty"`
	// Why the step cannot be undone, when it cannot.
	Locked string `json:"locked,omitempty"`
	// UndoneBy is the step that put this one back.
	UndoneBy string `json:"undoneBy,omitempty"`
	// Undoes lists the steps an undo step put back.
	Undoes []string `json:"undoes,omitempty"`
}

// undoEntry says which steps the undo that made a revision put back.
type undoEntry struct {
	// Rev is the number of the revision the undo made.
	Rev     int      `json:"rev"`
	Target  string   `json:"target"`
	Undoes  []string `json:"undoes"`
	Cascade bool     `json:"cascade"`
}

// historyMu keeps two undos of one process from writing at once.
var historyMu sync.Mutex

func undoLogPath(workspace, id string) (string, error) {
	path, err := sessionPath(workspace, id)
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(path), undoLogFile), nil
}

func readUndoLog(workspace, id string) ([]undoEntry, error) {
	path, err := undoLogPath(workspace, id)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read undo log: %w", err)
	}
	defer file.Close()
	var out []undoEntry
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var entry undoEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return out, fmt.Errorf("undo log line %d: %w", len(out)+1, err)
		}
		out = append(out, entry)
	}
	return out, scanner.Err()
}

func appendUndoLog(workspace, id string, entry undoEntry) error {
	path, err := undoLogPath(workspace, id)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("encode undo log: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("write undo log: %w", err)
	}
	if _, err := file.Write(append(raw, '\n')); err != nil {
		return errors.Join(fmt.Errorf("write undo log: %w", err), file.Close())
	}
	return file.Close()
}

func contains(in []string, id string) bool {
	for _, v := range in {
		if v == id {
			return true
		}
	}
	return false
}

// createdAt is when the session started, read from its id.
func createdAt(s *Session) time.Time {
	if len(s.ID) >= len(idTime) {
		if at, err := time.Parse(idTime, s.ID[:len(idTime)]); err == nil {
			return at
		}
	}
	return s.Activity()
}

func plural(n int, one, many string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, one)
	}
	return fmt.Sprintf("%d %s", n, many)
}

func quoted(text string) string {
	text = strings.TrimSpace(text)
	if len([]rune(text)) > 80 {
		text = string([]rune(text)[:79]) + "…"
	}
	return "“" + text + "”"
}

func joinIDs(ids []string) string {
	switch len(ids) {
	case 0:
		return ""
	case 1:
		return ids[0]
	}
	return strings.Join(ids[:len(ids)-1], ", ") + " and " + ids[len(ids)-1]
}

// before is what a revision's predecessor held, indexed for the diff.
type before struct {
	questions map[string]Question
	proposals map[string]Proposal
	rules     map[string]bool
	examples  map[string]bool
	splitKept bool
	sealed    bool
	children  int
	jira      bool
}

func beforeOf(snap Snapshot) before {
	b := before{
		questions: map[string]Question{}, proposals: map[string]Proposal{},
		rules: map[string]bool{}, examples: map[string]bool{}, splitKept: snap.SplitKept, sealed: snap.Status == StatusSealed,
		children: len(snap.Children), jira: snap.Jira != nil,
	}
	for _, q := range snap.Map.Questions {
		b.questions[q.ID] = q
	}
	for _, p := range snap.Proposals {
		b.proposals[p.ID] = p
	}
	for _, r := range snap.Map.Rules {
		b.rules[r.ID] = true
	}
	for _, e := range snap.Map.Examples {
		b.examples[e.ID] = true
	}
	return b
}

// changes lists the steps a revision made, in the order they must have happened: what
// the person settled first, then what the agent added after it.
func changes(prev, next Snapshot, agent string) []Event {
	was := beforeOf(prev)
	var out []Event
	var assumed []string
	var undo []Op
	var gives []string
	for _, q := range next.Map.Questions {
		old, existed := was.questions[q.ID]
		if !existed || (old.Status == q.Status && old.Answer == q.Answer) {
			continue
		}
		prior := Op{Op: opSetQuestion, ID: q.ID, Status: old.Status, Answer: old.Answer, AnswerID: old.AnswerID}
		switch q.Status {
		case QuestionAnswered:
			out = append(out, Event{
				Actor: personActor, Kind: KindAnswer, Ref: "Question " + strings.TrimPrefix(q.ID, "Q"),
				Summary: fmt.Sprintf("Answered %s: %s (%s)", q.ID, quoted(q.Answer), q.AnswerID),
				Gives:   []string{q.AnswerID}, Undo: []Op{prior},
			})
		case QuestionAssumed:
			assumed = append(assumed, q.ID)
			undo = append(undo, prior)
			gives = append(gives, q.AnswerID)
		}
	}
	if len(assumed) > 0 {
		out = append(out, Event{
			Actor: personActor, Kind: KindAssume, Ref: "Questions " + strings.Join(trimQ(assumed), ", "),
			Summary: "Used the assumption for " + joinIDs(assumed), Gives: gives, Undo: undo,
		})
	}
	if next.SplitKept && !was.splitKept {
		out = append(out, Event{
			Actor: personActor, Kind: KindSplitKeep, Ref: "Split", Summary: "Kept whole, no split",
			Undo: []Op{{Op: opSetSplitKept, Flag: false}},
		})
	}
	if len(next.Children) > was.children {
		out = append(out, Event{
			Actor: personActor, Kind: KindSplitAccept, Ref: "Split",
			Summary: "Split into " + plural(len(next.Children), "story", "stories"),
			Locked:  "the smaller stories already exist as their own sessions; archive or delete them there",
		})
	}
	out = append(out, proposalChanges(was, next)...)
	out = append(out, agentChanges(was, next, agent)...)
	if next.Status == StatusSealed && !was.sealed {
		digest := next.Digest
		if len(digest) > 8 {
			digest = digest[:8] + "…"
		}
		out = append(out, Event{
			Actor: personActor, Kind: KindSeal, Ref: "digest " + digest, Summary: "Sealed the story",
			Locked: "a sealed story cannot change",
		})
	}
	if next.Jira != nil && !was.jira {
		out = append(out, Event{
			Actor: personActor, Kind: KindJira, Ref: next.Jira.Key, Summary: "Created " + next.Jira.Key + " in Jira",
			Locked: "the issue already exists in Jira; Span never deletes it, so remove it there",
		})
	}
	return out
}

func proposalChanges(was before, next Snapshot) []Event {
	var out []Event
	for _, p := range next.Proposals {
		old, existed := was.proposals[p.ID]
		if !existed || old.Status == p.Status {
			continue
		}
		label := strings.TrimSpace(p.Section)
		switch {
		case p.Status == ProposalAccepted:
			out = append(out, Event{
				Actor: personActor, Kind: KindContextAccept, Ref: "Context note " + p.ID,
				Summary: "Accepted context note " + p.ID + ": " + label,
				Undo:    []Op{{Op: opUnacceptContext, ID: p.ID, Section: p.Section, Text: p.Text, Status: old.Status}},
			})
		case p.Status == ProposalSkipped:
			out = append(out, Event{
				Actor: personActor, Kind: KindContextSkip, Ref: "Context note " + p.ID,
				Summary: "Skipped context note " + p.ID + ": " + label,
				Undo:    []Op{{Op: opSetProposal, ID: p.ID, Status: old.Status}},
			})
		case old.Status == ProposalSkipped:
			out = append(out, Event{
				Actor: personActor, Kind: KindContextRestore, Ref: "Context note " + p.ID,
				Summary: "Brought back context note " + p.ID + ": " + label,
				Undo:    []Op{{Op: opSetProposal, ID: p.ID, Status: old.Status}},
			})
		}
	}
	return out
}

func trimQ(ids []string) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, strings.TrimPrefix(id, "Q"))
	}
	return out
}

// agentChanges are the rules, examples and questions the agent added since the earlier revision.
func agentChanges(was before, next Snapshot, agent string) []Event {
	var rules, examples, questions []string
	var undo, askUndo []Op
	needs := map[string]bool{}
	for _, r := range next.Map.Rules {
		if !was.rules[r.ID] {
			rules = append(rules, r.ID)
			undo = append(undo, Op{Op: opRemoveRule, ID: r.ID})
			noteNeed(needs, r.Source)
		}
	}
	for _, e := range next.Map.Examples {
		if !was.examples[e.ID] {
			examples = append(examples, e.ID)
			undo = append(undo, Op{Op: opRemoveExample, ID: e.ID})
			noteNeed(needs, e.Source)
		}
	}
	var out []Event
	if len(rules)+len(examples) > 0 {
		out = append(out, Event{
			Actor: agent, Kind: KindDraft, Ref: "Example map",
			Summary: "Drafted " + plural(len(rules), "rule", "rules") + " and " + plural(len(examples), "example", "examples"),
			Gives:   append(append([]string{}, rules...), examples...), Needs: sortedKeys(needs), Undo: undo,
		})
	}
	for _, q := range next.Map.Questions {
		if _, existed := was.questions[q.ID]; !existed {
			questions = append(questions, q.ID)
			askUndo = append(askUndo, Op{Op: opRemoveQuestion, ID: q.ID})
		}
	}
	if len(questions) > 0 {
		out = append(out, Event{
			Actor: agent, Kind: KindAsk, Ref: "Questions " + strings.Join(trimQ(questions), ", "),
			Summary: "Asked you " + plural(len(questions), "question", "questions") + " (" + strings.Join(questions, ", ") + ")",
			Gives:   questions, Undo: askUndo,
		})
	}
	return out
}

func noteNeed(needs map[string]bool, src Source) {
	if src.Kind == SourceAnswer && src.Ref != "" {
		needs[src.Ref] = true
	}
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// deriveEvents turns the revisions of a session into its steps. A revision made by an undo
// becomes the undo step the undo log names; any other revision becomes the steps its
// content change shows, or one plain step when only the label says what happened.
func deriveEvents(s *Session) ([]Event, error) {
	revs, err := loadRevisions(s.Workspace, s.ID)
	if err != nil {
		return nil, err
	}
	undos, err := readUndoLog(s.Workspace, s.ID)
	if err != nil {
		return nil, err
	}
	byRev := map[int]undoEntry{}
	for _, u := range undos {
		byRev[u.Rev] = u
	}
	agent := agentName(s)
	events := []Event{{
		ID: "h1", At: createdAt(s), Actor: personActor, Kind: KindIdea, Ref: "Session",
		Summary: "Saved the idea", Locked: "this is where the session starts",
	}}
	next := func(e Event) {
		e.ID = "h" + strconv.Itoa(len(events)+1)
		events = append(events, e)
	}
	prev := Snapshot{}
	for i, rev := range revs {
		if u, ok := byRev[rev.N]; ok {
			note := Event{At: rev.At, Actor: personActor, Kind: KindUndo, Ref: u.Target, Undoes: u.Undoes, Summary: rev.Label}
			next(note)
			for j := range events {
				if contains(u.Undoes, events[j].ID) {
					events[j].UndoneBy = "h" + strconv.Itoa(len(events))
				}
			}
			prev = rev.Snapshot
			continue
		}
		made := changes(prev, rev.Snapshot, agent)
		if len(made) == 0 && i > 0 && strings.TrimSpace(rev.Label) != "" {
			made = []Event{{Actor: rev.Actor, Kind: KindUpdate, Ref: "Session", Summary: rev.Label, Locked: "this step is not one an undo can put back"}}
		}
		for _, e := range made {
			e.At = rev.At
			next(e)
		}
		prev = rev.Snapshot
	}
	return events, nil
}

// EventView is a step with what the page and the CLI need to offer its undo.
type EventView struct {
	Event
	// Undoable says the step can be put back now.
	Undoable bool `json:"undoable"`
	// Why the step cannot be undone, when it cannot.
	Why string `json:"why,omitempty"`
	// Dependents are the later steps that would be put back with it.
	Dependents []string `json:"dependents,omitempty"`
}

// Counts is how many steps each filter of the history shows.
type Counts struct {
	Everyone int `json:"everyone"`
	You      int `json:"you"`
	Agents   int `json:"agents"`
}

// SessionSoFar is the short state of the session shown beside its history.
type SessionSoFar struct {
	Rules     int    `json:"rules"`
	Examples  int    `json:"examples"`
	Open      int    `json:"open"`
	Answered  int    `json:"answered"`
	Assumed   int    `json:"assumed"`
	Split     string `json:"split"`
	Context   int    `json:"context"`
	SealedAt  string `json:"sealedAt,omitempty"`
	Digest    string `json:"digest,omitempty"`
	Locked    bool   `json:"locked"`
	LockedWhy string `json:"lockedWhy,omitempty"`
}

// HistoryView is the history of a session, newest step first.
type HistoryView struct {
	ID      string       `json:"id"`
	Ref     string       `json:"ref"`
	Title   string       `json:"title"`
	Status  string       `json:"status"`
	Sealed  bool         `json:"sealed"`
	Counts  Counts       `json:"counts"`
	Events  []EventView  `json:"events"`
	Session SessionSoFar `json:"session"`
}

// HistoryFilter narrows a history: Actor is "you" or "agents", Kind one of "answers",
// "assumptions", "context" or "split".
type HistoryFilter struct {
	Actor string
	Kind  string
}

func (f HistoryFilter) check() error {
	switch f.Actor {
	case "", "everyone", "you", "agents":
	default:
		return fail("invalid", "actor must be everyone, you or agents, not %q", f.Actor)
	}
	switch f.Kind {
	case "", "all", "answers", "assumptions", "context", "split":
	default:
		return fail("invalid", "kind must be answers, assumptions, context or split, not %q", f.Kind)
	}
	return nil
}

func (f HistoryFilter) keeps(e Event) bool {
	switch f.Actor {
	case "you":
		if e.Actor != personActor {
			return false
		}
	case "agents":
		if e.Actor == personActor {
			return false
		}
	}
	switch f.Kind {
	case "answers":
		return e.Kind == KindAnswer
	case "assumptions":
		return e.Kind == KindAssume
	case "context":
		return strings.HasPrefix(e.Kind, "context_")
	case "split":
		return strings.HasPrefix(e.Kind, "split_")
	}
	return true
}

// dependents returns the later steps that need what the step gave, and the steps that
// need those, in the order they happened.
func dependents(events []Event, target Event) []Event {
	gone := map[string]bool{}
	for _, g := range target.Gives {
		gone[g] = true
	}
	var out []Event
	for _, e := range events {
		if e.ID == target.ID || e.UndoneBy != "" || e.Kind == KindUndo || !after(e.ID, target.ID) {
			continue
		}
		if needsAny(e, gone) {
			out = append(out, e)
			for _, g := range e.Gives {
				gone[g] = true
			}
		}
	}
	return out
}

func needsAny(e Event, gone map[string]bool) bool {
	for _, n := range e.Needs {
		if gone[n] {
			return true
		}
	}
	return false
}

// after says whether step a happened after step b, by their numbers.
func after(a, b string) bool {
	x, errA := strconv.Atoi(strings.TrimPrefix(a, "h"))
	y, errB := strconv.Atoi(strings.TrimPrefix(b, "h"))
	// A step that is not numbered is never after another one.
	return errA == nil && errB == nil && x > y
}

func (e Event) lockedWhy(sealed bool) string {
	switch {
	case e.UndoneBy != "":
		return "already undone by " + e.UndoneBy
	case e.Kind == KindUndo:
		return "an undo is not undone; do the step again instead"
	case e.Kind == KindJira:
		return e.Locked
	case sealed:
		return "a sealed story cannot change"
	case e.Locked != "":
		return e.Locked
	case len(e.Undo) == 0:
		return "nothing to put back"
	}
	return ""
}

func view(events []Event, e Event, sealed bool) EventView {
	v := EventView{Event: e, Why: e.lockedWhy(sealed)}
	v.Undoable = v.Why == ""
	for _, d := range dependents(events, e) {
		v.Dependents = append(v.Dependents, d.ID)
		if why := d.lockedWhy(sealed); why != "" && v.Undoable {
			v.Undoable, v.Why = false, d.ID+" depends on it and "+why
		}
	}
	return v
}

func soFar(s *Session) SessionSoFar {
	out := SessionSoFar{Rules: len(s.Map.Rules), Examples: len(s.Map.Examples), Split: "not needed"}
	for _, q := range s.Map.Questions {
		switch q.Status {
		case QuestionOpen:
			out.Open++
		case QuestionAnswered:
			out.Answered++
		case QuestionAssumed:
			out.Assumed++
		}
	}
	for _, p := range s.Proposals {
		if p.Status == ProposalAccepted {
			out.Context++
		}
	}
	switch {
	case len(s.Children) > 0:
		out.Split = "split into " + plural(len(s.Children), "story", "stories")
	case s.SplitKept:
		out.Split = "kept whole"
	case len(s.Split) > 0:
		out.Split = "proposed"
	}
	if s.Status == StatusSealed {
		out.Locked, out.LockedWhy = true, "a sealed story cannot change"
		if s.SealedAt != nil {
			out.SealedAt = s.SealedAt.UTC().Format(time.RFC3339)
		}
		out.Digest = s.Digest
	}
	return out
}

// History returns the steps of a session, newest first, derived from its revisions, so it
// shows changes made by any process: the page, the CLI or a resumed run.
func History(workspace, id string, f HistoryFilter) (HistoryView, error) {
	if err := f.check(); err != nil {
		return HistoryView{}, err
	}
	historyMu.Lock()
	defer historyMu.Unlock()
	s, err := Load(workspace, id)
	if err != nil {
		return HistoryView{}, err
	}
	events, err := deriveEvents(s)
	if err != nil {
		return HistoryView{}, err
	}
	sealed := s.Status == StatusSealed
	out := HistoryView{
		ID: s.ID, Ref: s.Ref(), Title: s.Title(), Status: s.Status, Sealed: sealed,
		Events: []EventView{}, Session: soFar(s),
	}
	for i := len(events) - 1; i >= 0; i-- {
		e := events[i]
		out.Counts.Everyone++
		if e.Actor == personActor {
			out.Counts.You++
		} else {
			out.Counts.Agents++
		}
		if f.keeps(e) {
			out.Events = append(out.Events, view(events, e, sealed))
		}
	}
	return out, nil
}

// UndoPlan says what undoing a step would change, before anything changes.
type UndoPlan struct {
	Target     EventView   `json:"target"`
	Dependents []EventView `json:"dependents"`
	// Now and After are the state of what the step touched, now and once it is undone.
	Now   string `json:"now"`
	After string `json:"after"`
	// Changes lists what putting the steps back does, one line each.
	Changes []string `json:"changes"`
	// Kept says what stays: the undo is a new step.
	Kept string `json:"kept"`
	// Next says what the person does afterwards.
	Next string `json:"next"`
	// Sealed and Undoable repeat what the page needs to enable its two buttons.
	Sealed bool `json:"sealed"`
	// UndoOnlyThis is false when later steps depend on this one, so only the cascade is offered.
	UndoOnlyThis bool `json:"undoOnlyThis"`
}

func findEvent(events []Event, hid string) (Event, error) {
	for _, e := range events {
		if e.ID == hid {
			return e, nil
		}
	}
	return Event{}, fail("shape_step_not_found", "no step %q in this session", hid)
}

func describeNow(s *Session, e Event) string {
	for _, op := range e.Undo {
		if op.Op != opSetQuestion {
			continue
		}
		for _, q := range s.Map.Questions {
			if q.ID != op.ID {
				continue
			}
			if q.Status == QuestionAnswered {
				return fmt.Sprintf("Answered by you: %s (%s)", q.Answer, q.AnswerID)
			}
			if q.Status == QuestionAssumed {
				return "Assumed: " + q.Answer
			}
		}
	}
	return e.Summary
}

func describeOp(s *Session, op Op) (after string, change string) {
	switch op.Op {
	case opSetQuestion:
		switch op.Status {
		case QuestionAssumed:
			return "Assumed by the agent: " + op.Answer, op.ID + " goes back to assumed"
		case QuestionAnswered:
			return "Answered by you: " + op.Answer, op.ID + " goes back to its earlier answer"
		}
		return op.ID + " is open again", op.ID + " goes back to open"
	case opRemoveRule:
		return "", op.ID + " leaves the map"
	case opRemoveExample:
		return "", op.ID + " leaves the map"
	case opRemoveQuestion:
		return "", op.ID + " leaves the map"
	case opSetSplitKept:
		return "", "The split can be proposed again"
	case opSetProposal:
		return "", "Context note " + op.ID + " goes back to " + op.Status
	case opUnacceptContext:
		return "", "Context note " + op.ID + " leaves the product context and goes back to proposed"
	}
	return "", op.Op
}

// PlanUndo says what undoing the step would put back, without changing anything.
func PlanUndo(workspace, id, hid string) (UndoPlan, error) {
	historyMu.Lock()
	defer historyMu.Unlock()
	s, err := Load(workspace, id)
	if err != nil {
		return UndoPlan{}, err
	}
	events, err := deriveEvents(s)
	if err != nil {
		return UndoPlan{}, err
	}
	target, err := findEvent(events, hid)
	if err != nil {
		return UndoPlan{}, err
	}
	sealed := s.Status == StatusSealed
	plan := UndoPlan{
		Target: view(events, target, sealed), Dependents: []EventView{}, Sealed: sealed,
		Now:   describeNow(s, target),
		Kept:  "The undo is a new step. The history is not erased.",
		Next:  "You settle it again, or keep what it goes back to.",
		After: "",
	}
	steps := append([]Event{target}, dependents(events, target)...)
	for i, e := range steps {
		if i > 0 {
			plan.Dependents = append(plan.Dependents, view(events, e, sealed))
		}
		for _, op := range e.Undo {
			after, change := describeOp(s, op)
			if i == 0 && plan.After == "" {
				plan.After = after
			}
			plan.Changes = append(plan.Changes, change)
		}
	}
	if plan.After == "" {
		plan.After = "Put back: " + strings.ToLower(target.Summary[:1]) + target.Summary[1:]
	}
	plan.UndoOnlyThis = len(plan.Dependents) == 0
	return plan, nil
}

// UndoResult is what an undo did.
type UndoResult struct {
	Session *Session `json:"-"`
	// Step is the new step that records the undo.
	Step Event `json:"step"`
	// Undone lists the steps that were put back, the one asked for first.
	Undone []string `json:"undone"`
}

// Undo puts a step back, and with cascade the later steps that depend on it. Without
// cascade a step that has dependents is refused. A sealed story never changes, so nothing
// is undone once it is sealed. The undo is a new step: the history is not erased.
func (e *Engine) Undo(workspace, id, hid string, cascade bool) (UndoResult, error) {
	historyMu.Lock()
	defer historyMu.Unlock()
	s, err := Load(workspace, id)
	if err != nil {
		return UndoResult{}, err
	}
	if s.Status == StatusSealed {
		return UndoResult{}, fail("shape_sealed", "session %s is sealed: a sealed story cannot change", id)
	}
	events, err := deriveEvents(s)
	if err != nil {
		return UndoResult{}, err
	}
	target, err := findEvent(events, hid)
	if err != nil {
		return UndoResult{}, err
	}
	chain := append([]Event{target}, dependents(events, target)...)
	for i, step := range chain {
		if why := step.lockedWhy(false); why != "" {
			if i == 0 {
				return UndoResult{}, fail("shape_not_undoable", "step %s cannot be undone: %s", step.ID, why)
			}
			return UndoResult{}, fail("shape_not_undoable", "step %s cannot be undone: %s depends on it and %s", target.ID, step.ID, why)
		}
	}
	if len(chain) > 1 && !cascade {
		ids := make([]string, 0, len(chain)-1)
		for _, d := range chain[1:] {
			ids = append(ids, d.ID)
		}
		return UndoResult{}, fail("shape_has_dependents", "step %s has steps that depend on it (%s): undo them with it", target.ID, strings.Join(ids, ", "))
	}
	for i := len(chain) - 1; i >= 0; i-- {
		for _, op := range chain[i].Undo {
			if err := applyOp(workspace, s, op); err != nil {
				return UndoResult{}, err
			}
		}
	}
	s.Status = computeStatus(s)
	path, err := revisionsPath(workspace, id)
	if err != nil {
		return UndoResult{}, err
	}
	undone := make([]string, 0, len(chain))
	for _, step := range chain {
		undone = append(undone, step.ID)
	}
	label := "Undid " + target.ID + ": " + target.Summary
	if len(chain) > 1 {
		label = "Undid " + target.ID + " and " + plural(len(chain)-1, "step", "steps") + " that depend on it"
	}
	// The undo is a revision of its own, so the log is only ever appended to.
	revisionHints.Store(path, revisionHint{label: label, actor: ActorPerson})
	defer revisionHints.Delete(path)
	if err := Save(s); err != nil {
		return UndoResult{}, err
	}
	revs, err := loadRevisions(workspace, id)
	if err != nil {
		return UndoResult{}, err
	}
	entry := undoEntry{Rev: len(revs), Target: target.ID, Undoes: undone, Cascade: len(chain) > 1}
	if err := appendUndoLog(workspace, id, entry); err != nil {
		return UndoResult{}, err
	}
	step := Event{
		ID: "h" + strconv.Itoa(len(events)+1), At: revs[len(revs)-1].At, Actor: personActor, Kind: KindUndo,
		Ref: target.ID, Summary: label, Undoes: undone,
	}
	return UndoResult{Session: s, Step: step, Undone: undone}, nil
}

func applyOp(workspace string, s *Session, op Op) error {
	switch op.Op {
	case opRemoveRule:
		s.Map.Rules = removeBy(s.Map.Rules, func(r Rule) bool { return r.ID == op.ID })
	case opRemoveExample:
		s.Map.Examples = removeBy(s.Map.Examples, func(x Example) bool { return x.ID == op.ID })
	case opRemoveQuestion:
		s.Map.Questions = removeBy(s.Map.Questions, func(q Question) bool { return q.ID == op.ID })
	case opSetQuestion:
		if q := findQuestion(s, op.ID); q != nil {
			q.Status, q.Answer, q.AnswerID = op.Status, op.Answer, op.AnswerID
		}
	case opSetSplitKept:
		s.SplitKept = op.Flag
	case opSetProposal:
		if p := findProposal(s, op.ID); p != nil {
			p.Status = op.Status
		}
	case opUnacceptContext:
		p := findProposal(s, op.ID)
		if p == nil {
			return nil
		}
		if err := removeContext(workspace, op.Section, op.Text); err != nil {
			return err
		}
		p.Status = ProposalProposed
		if op.Status == ProposalSkipped {
			p.Status = ProposalSkipped
		}
	}
	return nil
}

func removeBy[T any](in []T, drop func(T) bool) []T {
	out := make([]T, 0, len(in))
	for _, v := range in {
		if !drop(v) {
			out = append(out, v)
		}
	}
	return out
}

// removeContext takes out what mergeContext added: the entry under its section, and the
// section heading when the entry was all it held. A file that no longer has the entry
// is left alone and the undo is refused, so a note a person edited is never lost.
func removeContext(workspace, section, text string) error {
	path := ContextPath(workspace)
	raw, err := os.ReadFile(path)
	if err != nil {
		return fail("shape_context_changed", "the product context cannot be read, so the note cannot be taken out: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	entry := strings.Split(strings.TrimRight(text, "\n"), "\n")
	heading := "## " + strings.TrimSpace(section)
	at := -1
	for i := len(lines) - len(entry); i >= 0; i-- {
		if equalLines(lines[i:i+len(entry)], entry) {
			at = i
			break
		}
	}
	if at < 0 {
		return fail("shape_context_changed", "the note is not in the product context any more, so it cannot be taken out")
	}
	rest := append(append([]string{}, lines[:at]...), lines[at+len(entry):]...)
	rest = dropEmptySection(rest, heading)
	body := strings.TrimRight(strings.Join(rest, "\n"), "\n")
	if body != "" {
		body += "\n"
	}
	return writeAtomic(path, []byte(body))
}

func equalLines(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// dropEmptySection removes heading and its blank lines when nothing else is under it.
func dropEmptySection(lines []string, heading string) []string {
	for i, l := range lines {
		if strings.TrimSpace(l) != heading {
			continue
		}
		end := i + 1
		for end < len(lines) && strings.TrimSpace(lines[end]) == "" {
			end++
		}
		if end < len(lines) && !strings.HasPrefix(lines[end], "## ") {
			return lines
		}
		out := append([]string{}, lines[:i]...)
		out = append(out, lines[end:]...)
		for len(out) > 0 && strings.TrimSpace(out[len(out)-1]) == "" {
			out = out[:len(out)-1]
		}
		return out
	}
	return lines
}
