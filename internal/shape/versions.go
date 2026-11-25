package shape

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// revisionsFile holds the revisions of one session, one JSON object per line, next to
// session.json. A line is only ever appended.
const revisionsFile = "revisions.jsonl"

// Revision actors: who made a change.
const (
	// ActorPerson is a change the person made: an answer, an accept, a seal.
	ActorPerson = "you"
	// ActorAgent is a change the interview agent made: a drafted map, a proposed split.
	ActorAgent = "agent"
)

// Snapshot is the content of a session that revisions track: what the story says and what
// settles it. Bookkeeping such as timestamps, turns, failures or the archive flag is not
// part of it, so saving them never makes a revision.
type Snapshot struct {
	Idea      string     `json:"idea"`
	Status    string     `json:"status"`
	Map       Map        `json:"map"`
	Split     []Split    `json:"split"`
	SplitKept bool       `json:"splitKept"`
	Proposals []Proposal `json:"proposals"`
	Story     string     `json:"story"`
	Digest    string     `json:"digest"`
	// Children are the sessions an accepted split made.
	Children []string `json:"children,omitempty"`
	// Jira is the issue created for the sealed story.
	Jira *JiraRef `json:"jira,omitempty"`
}

// Revision is one saved state of a session, with what made it.
type Revision struct {
	// N counts from 1; the id of a revision is "r" and N.
	N      int       `json:"n"`
	ID     string    `json:"id"`
	At     time.Time `json:"at"`
	Actor  string    `json:"actor"`
	Label  string    `json:"label"`
	Digest string    `json:"digest"`
	// Snapshot is the full content at that revision, so any two can be compared.
	Snapshot Snapshot `json:"snapshot"`
}

// RevisionInfo is a revision without its content, for lists.
type RevisionInfo struct {
	N      int       `json:"n"`
	ID     string    `json:"id"`
	At     time.Time `json:"at"`
	Actor  string    `json:"actor"`
	Label  string    `json:"label"`
	Digest string    `json:"digest"`
	Status string    `json:"status"`
	// Current marks the last revision.
	Current bool `json:"current"`
}

func (r Revision) info(current bool) RevisionInfo {
	return RevisionInfo{N: r.N, ID: r.ID, At: r.At, Actor: r.Actor, Label: r.Label, Digest: r.Digest, Status: r.Snapshot.Status, Current: current}
}

func snapshotOf(s *Session) Snapshot {
	snap := Snapshot{
		Idea: s.Idea, Status: s.Status, Map: s.Map, Split: s.Split, SplitKept: s.SplitKept,
		Proposals: s.Proposals, Story: s.Story, Digest: s.Digest, Children: s.Children, Jira: s.Jira,
	}
	// Nil and empty slices mean the same thing; one spelling keeps digests stable.
	if snap.Map.Rules == nil {
		snap.Map.Rules = []Rule{}
	}
	if snap.Map.Examples == nil {
		snap.Map.Examples = []Example{}
	}
	if snap.Map.Questions == nil {
		snap.Map.Questions = []Question{}
	}
	if snap.Split == nil {
		snap.Split = []Split{}
	}
	if snap.Proposals == nil {
		snap.Proposals = []Proposal{}
	}
	return snap
}

func (n Snapshot) digest() (string, error) {
	raw, err := json.Marshal(n)
	if err != nil {
		return "", fmt.Errorf("encode snapshot: %w", err)
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func revisionsPath(workspace, id string) (string, error) {
	path, err := sessionPath(workspace, id)
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(path), revisionsFile), nil
}

func readRevisions(path string) ([]Revision, error) {
	file, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read revisions: %w", err)
	}
	defer file.Close()
	var out []Revision
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 16<<20)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var rev Revision
		if err := json.Unmarshal([]byte(line), &rev); err != nil {
			return out, fmt.Errorf("revisions line %d: %w", len(out)+1, err)
		}
		out = append(out, rev)
	}
	if err := scanner.Err(); err != nil {
		return out, fmt.Errorf("read revisions: %w", err)
	}
	return out, nil
}

// revisionHints carry the label and actor a caller wants on the revision its next Save makes,
// keyed by session path. Callers hold the session's lock, so a hint is used by that Save.
var revisionHints sync.Map

type revisionHint struct{ label, actor string }

// recordRevision appends a revision when the saved content differs from the last revision.
// Save calls it after the session file is written; a failure is returned to the caller,
// because a history that silently skips a change is worse than a failed save.
func recordRevision(s *Session) error {
	path, err := revisionsPath(s.Workspace, s.ID)
	if err != nil {
		return err
	}
	revs, err := readRevisions(path)
	if err != nil {
		return err
	}
	next := snapshotOf(s)
	digest, err := next.digest()
	if err != nil {
		return err
	}
	var prev *Snapshot
	if n := len(revs); n > 0 {
		if revs[n-1].Digest == digest {
			return nil
		}
		prev = &revs[n-1].Snapshot
	}
	if prev == nil && firstTurnSave(s) {
		// Start saves a session only after its first turn, so the idea itself never had a save.
		// Record it first, so the history reads: idea saved, then what the agent drafted.
		idea := Snapshot{Idea: s.Idea, Status: StatusInterviewing, Map: Map{Rules: []Rule{}, Examples: []Example{}, Questions: []Question{}}, Split: []Split{}, Proposals: []Proposal{}}
		if revs, err = appendRevision(path, revs, idea, "Idea saved", ActorPerson); err != nil {
			return err
		}
		prev = &revs[len(revs)-1].Snapshot
	}
	label, actor := describeChange(prev, next, s, len(revs) == 0)
	if hint, ok := revisionHints.LoadAndDelete(path); ok {
		h := hint.(revisionHint)
		label, actor = h.label, h.actor
	}
	_, err = appendRevision(path, revs, next, label, actor)
	return err
}

// appendRevision writes one revision line and returns the list with it added.
func appendRevision(path string, revs []Revision, snap Snapshot, label, actor string) ([]Revision, error) {
	digest, err := snap.digest()
	if err != nil {
		return revs, err
	}
	rev := Revision{N: len(revs) + 1, At: time.Now().UTC(), Actor: actor, Label: label, Digest: digest, Snapshot: snap}
	rev.ID = "r" + strconv.Itoa(rev.N)
	raw, err := json.Marshal(rev)
	if err != nil {
		return revs, fmt.Errorf("encode revision: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return revs, fmt.Errorf("write revisions: %w", err)
	}
	if _, err := file.Write(append(raw, '\n')); err != nil {
		return revs, errors.Join(fmt.Errorf("write revisions: %w", err), file.Close())
	}
	if err := file.Close(); err != nil {
		return revs, fmt.Errorf("write revisions: %w", err)
	}
	return append(revs, rev), nil
}

// firstTurnSave is true for a session saved for the first time right after its first turn:
// one turn, created in the last minutes. An older session saved for the first time since
// revisions began is not new, and its history says it starts there.
func firstTurnSave(s *Session) bool {
	if len(s.Turns) != 1 || len(s.ID) < len(idTime) {
		return false
	}
	created, err := time.Parse(idTime, s.ID[:len(idTime)])
	return err == nil && time.Since(created) < 5*time.Minute
}

func agentName(s *Session) string {
	name := strings.TrimSpace(s.Agent)
	if name == "" || strings.EqualFold(name, "auto") || strings.EqualFold(name, "any") {
		return ActorAgent
	}
	return name
}

// describeChange names what changed between two snapshots and who did it. The first
// revision of a session that already has content (saved before revisions existed) says so.
func describeChange(prev *Snapshot, next Snapshot, s *Session, first bool) (label, actor string) {
	if prev == nil {
		if first && (len(next.Map.Rules) > 0 || next.Story != "") {
			return "History starts here", ActorPerson
		}
		return "Idea saved", ActorPerson
	}
	if next.Status == StatusSealed && prev.Status != StatusSealed {
		return "Sealed", ActorPerson
	}
	if len(next.Children) > len(prev.Children) {
		return "Split accepted", ActorPerson
	}
	if next.Jira != nil && prev.Jira == nil {
		return "Created in Jira", ActorPerson
	}
	if label := questionChange(prev.Map.Questions, next.Map.Questions); label != "" {
		return label, ActorPerson
	}
	if label := proposalChange(prev.Proposals, next.Proposals); label != "" {
		return label, ActorPerson
	}
	if next.SplitKept && !prev.SplitKept {
		return "Kept as one story", ActorPerson
	}
	if len(prev.Split) == 0 && len(next.Split) > 0 {
		return "Split proposed", agentName(s)
	}
	if len(prev.Map.Rules) == 0 && len(next.Map.Rules) > 0 {
		return "Map drafted", agentName(s)
	}
	if mapChanged(prev.Map, next.Map) {
		return "Map updated", agentName(s)
	}
	if prev.Story != next.Story {
		return "Story drafted", agentName(s)
	}
	return "Updated", ActorPerson
}

func mapChanged(a, b Map) bool {
	x, errA := json.Marshal(a)
	y, errB := json.Marshal(b)
	return errA != nil || errB != nil || string(x) != string(y)
}

// questionChange says which questions the person answered or settled with their assumption.
func questionChange(prev, next []Question) string {
	before := map[string]Question{}
	for _, q := range prev {
		before[q.ID] = q
	}
	var answered, assumed []string
	for _, q := range next {
		old, seen := before[q.ID]
		if seen && old.Status == q.Status && old.Answer == q.Answer {
			continue
		}
		switch q.Status {
		case QuestionAnswered:
			answered = append(answered, withAnswerID(q))
		case QuestionAssumed:
			assumed = append(assumed, withAnswerID(q))
		}
	}
	switch {
	case len(answered) > 0:
		return "Answered " + strings.Join(answered, ", ")
	case len(assumed) > 0:
		return "Used the assumption for " + strings.Join(assumed, ", ")
	}
	return ""
}

func withAnswerID(q Question) string {
	if q.AnswerID == "" {
		return q.ID
	}
	return q.ID + " (" + q.AnswerID + ")"
}

func proposalChange(prev, next []Proposal) string {
	before := map[string]string{}
	for _, p := range prev {
		before[p.ID] = p.Status
	}
	for _, p := range next {
		old, seen := before[p.ID]
		if !seen || old == p.Status {
			continue
		}
		switch p.Status {
		case ProposalAccepted:
			return "Context note accepted"
		case ProposalSkipped:
			return "Context note skipped"
		case ProposalProposed:
			return "Context note restored"
		}
	}
	return ""
}

// Revisions lists the revisions of a session, oldest first, without their content. A session
// saved before revisions existed has none until its next change.
func Revisions(workspace, id string) ([]RevisionInfo, error) {
	revs, err := loadRevisions(workspace, id)
	if err != nil {
		return nil, err
	}
	out := make([]RevisionInfo, 0, len(revs))
	for i, r := range revs {
		out = append(out, r.info(i == len(revs)-1))
	}
	return out, nil
}

func loadRevisions(workspace, id string) ([]Revision, error) {
	if _, err := Load(workspace, id); err != nil {
		return nil, err
	}
	path, err := revisionsPath(workspace, id)
	if err != nil {
		return nil, err
	}
	return readRevisions(path)
}

// findRevision resolves "r3" or "3" to a revision.
func findRevision(revs []Revision, ref string) (Revision, error) {
	text := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(ref)), "r")
	n, err := strconv.Atoi(text)
	if err != nil || n < 1 || n > len(revs) {
		return Revision{}, fail("shape_revision_not_found", "revision %q does not exist; the session has %d", ref, len(revs))
	}
	return revs[n-1], nil
}

// Restore makes a new revision with the content of an earlier one. Nothing is deleted: the
// earlier revisions stay in the history. A sealed story cannot change, so it refuses. The
// product context file is not touched, so proposals keep the status they have now.
func Restore(workspace, id, ref string) (*Session, error) {
	s, err := Load(workspace, id)
	if err != nil {
		return nil, err
	}
	if s.Status == StatusSealed {
		return nil, fail("shape_sealed", "session %s is sealed; a sealed story cannot change", id)
	}
	revs, err := loadRevisions(workspace, id)
	if err != nil {
		return nil, err
	}
	rev, err := findRevision(revs, ref)
	if err != nil {
		return nil, err
	}
	if rev.Snapshot.Status == StatusSealed {
		return nil, fail("shape_sealed", "revision %s is the sealed story; restore an earlier one", rev.ID)
	}
	snap := rev.Snapshot
	s.Idea, s.Map, s.Split, s.SplitKept, s.Story, s.Digest = snap.Idea, snap.Map, snap.Split, snap.SplitKept, snap.Story, ""
	s.Status = snap.Status
	if s.OpenQuestions() > 0 {
		s.Status = StatusInterviewing
	}
	path, err := revisionsPath(workspace, id)
	if err != nil {
		return nil, err
	}
	revisionHints.Store(path, revisionHint{label: "Restored " + rev.ID, actor: ActorPerson})
	// A restore of the content already current makes no revision, so the hint must not outlive it.
	defer revisionHints.Delete(path)
	if err := Save(s); err != nil {
		return nil, err
	}
	return s, nil
}

// ForkRevision starts a new session from the content of a revision of another, which stays
// as it is. It works on a sealed session too: that is how a sealed story is reopened. As in
// Fork, assumptions reopen so the person can answer them this time.
func ForkRevision(workspace, id, ref string) (*Session, error) {
	parent, err := Load(workspace, id)
	if err != nil {
		return nil, err
	}
	revs, err := loadRevisions(workspace, id)
	if err != nil {
		return nil, err
	}
	rev, err := findRevision(revs, ref)
	if err != nil {
		return nil, err
	}
	snap := rev.Snapshot
	return forkSession(parent, snap.Idea, snap.Map, snap.Story, time.Now())
}

// Change kinds of a revision diff.
const (
	// ChangeAdded is an item present only in the later revision.
	ChangeAdded = "added"
	// ChangeRemoved is an item present only in the earlier revision.
	ChangeRemoved = "removed"
	// ChangeChanged is an item in both whose content differs.
	ChangeChanged = "changed"
)

// Item kinds of a revision diff.
const (
	// ItemRule is a rule of the example map.
	ItemRule = "rule"
	// ItemExample is an example of the example map.
	ItemExample = "example"
	// ItemQuestion is a question of the example map.
	ItemQuestion = "question"
)

// ItemChange is one rule, example or question that differs between two revisions.
type ItemChange struct {
	Kind   string `json:"kind"`
	ID     string `json:"id"`
	Change string `json:"change"`
	// Before and After say the item the way a person reads it; empty on the side where it is absent.
	Before string `json:"before"`
	After  string `json:"after"`
	// Source is where the later (or, for a removal, the earlier) version comes from.
	Source *Source `json:"source,omitempty"`
}

// Tally counts what changed for one kind of item.
type Tally struct {
	Added   int `json:"added"`
	Removed int `json:"removed"`
	Changed int `json:"changed"`
}

// StoryLine is one line of the story text diff. Old and New are 1-based line numbers in the
// earlier and later story; 0 on the side where the line is absent.
type StoryLine struct {
	Kind string `json:"kind"`
	Old  int    `json:"old"`
	New  int    `json:"new"`
	Text string `json:"text"`
}

// Story line kinds.
const (
	// LineContext is a line present in both stories.
	LineContext = "ctx"
	// LineAdd is a line present only in the later story.
	LineAdd = "add"
	// LineDel is a line present only in the earlier story.
	LineDel = "del"
)

// RevisionDiff is what changed between two revisions of one session, by kind of item.
type RevisionDiff struct {
	From      RevisionInfo `json:"from"`
	To        RevisionInfo `json:"to"`
	Rules     Tally        `json:"rules"`
	Examples  Tally        `json:"examples"`
	Questions Tally        `json:"questions"`
	// StoryAdded and StoryRemoved count the lines of story text.
	StoryAdded   int          `json:"storyAdded"`
	StoryRemoved int          `json:"storyRemoved"`
	Items        []ItemChange `json:"items"`
	Story        []StoryLine  `json:"story"`
	// Sealed is true when the session is sealed, so the page says nothing can be restored.
	Sealed bool `json:"sealed"`
}

// Diff compares two revisions of a session. from and to are "r3" or "3"; they may be given
// in either order of time, the diff always goes from the first to the second.
func Diff(workspace, id, from, to string) (RevisionDiff, error) {
	s, err := Load(workspace, id)
	if err != nil {
		return RevisionDiff{}, err
	}
	revs, err := loadRevisions(workspace, id)
	if err != nil {
		return RevisionDiff{}, err
	}
	a, err := findRevision(revs, from)
	if err != nil {
		return RevisionDiff{}, err
	}
	b, err := findRevision(revs, to)
	if err != nil {
		return RevisionDiff{}, err
	}
	out := RevisionDiff{From: a.info(a.N == len(revs)), To: b.info(b.N == len(revs)), Items: []ItemChange{}, Sealed: s.Status == StatusSealed}
	diffMaps(&out, a.Snapshot.Map, b.Snapshot.Map)
	out.Story, out.StoryAdded, out.StoryRemoved = diffStory(a.Snapshot.Story, b.Snapshot.Story)
	return out, nil
}

func tally(t *Tally, change string) {
	switch change {
	case ChangeAdded:
		t.Added++
	case ChangeRemoved:
		t.Removed++
	default:
		t.Changed++
	}
}

func diffMaps(out *RevisionDiff, a, b Map) {
	oldRules := map[string]Rule{}
	for _, r := range a.Rules {
		oldRules[r.ID] = r
	}
	seen := map[string]bool{}
	for _, r := range b.Rules {
		seen[r.ID] = true
		old, had := oldRules[r.ID]
		src := r.Source
		switch {
		case !had:
			out.add(ItemRule, r.ID, ChangeAdded, "", r.Text, &src, &out.Rules)
		case old.Text != r.Text:
			out.add(ItemRule, r.ID, ChangeChanged, old.Text, r.Text, &src, &out.Rules)
		}
	}
	for _, r := range a.Rules {
		if !seen[r.ID] {
			src := r.Source
			out.add(ItemRule, r.ID, ChangeRemoved, r.Text, "", &src, &out.Rules)
		}
	}
	oldExamples := map[string]Example{}
	for _, e := range a.Examples {
		oldExamples[e.ID] = e
	}
	seen = map[string]bool{}
	for _, e := range b.Examples {
		seen[e.ID] = true
		old, had := oldExamples[e.ID]
		src := e.Source
		switch {
		case !had:
			out.add(ItemExample, e.ID, ChangeAdded, "", e.Text, &src, &out.Examples)
		case old.Text != e.Text || old.Rule != e.Rule:
			out.add(ItemExample, e.ID, ChangeChanged, old.Text, e.Text, &src, &out.Examples)
		}
	}
	for _, e := range a.Examples {
		if !seen[e.ID] {
			src := e.Source
			out.add(ItemExample, e.ID, ChangeRemoved, e.Text, "", &src, &out.Examples)
		}
	}
	oldQuestions := map[string]Question{}
	for _, q := range a.Questions {
		oldQuestions[q.ID] = q
	}
	seen = map[string]bool{}
	for _, q := range b.Questions {
		seen[q.ID] = true
		old, had := oldQuestions[q.ID]
		switch {
		case !had:
			out.add(ItemQuestion, q.ID, ChangeAdded, "", describeQuestion(q), nil, &out.Questions)
		case old.Status != q.Status || old.Answer != q.Answer || old.Text != q.Text || old.Assume != q.Assume:
			out.add(ItemQuestion, q.ID, ChangeChanged, describeQuestion(old), describeQuestion(q), nil, &out.Questions)
		}
	}
	for _, q := range a.Questions {
		if !seen[q.ID] {
			out.add(ItemQuestion, q.ID, ChangeRemoved, describeQuestion(q), "", nil, &out.Questions)
		}
	}
}

func (d *RevisionDiff) add(kind, id, change, before, after string, src *Source, count *Tally) {
	tally(count, change)
	d.Items = append(d.Items, ItemChange{Kind: kind, ID: id, Change: change, Before: before, After: after, Source: src})
}

// describeQuestion says how a question stands: open, assumed by the agent, or answered.
func describeQuestion(q Question) string {
	switch q.Status {
	case QuestionAnswered:
		return "Answered by you: " + q.Answer + answerSuffix(q)
	case QuestionAssumed:
		return "Assumed by the agent: " + q.Answer + answerSuffix(q)
	}
	if q.Assume != "" {
		return "Open: " + q.Text + " (assumption if skipped: " + q.Assume + ")"
	}
	return "Open: " + q.Text
}

func answerSuffix(q Question) string {
	if q.AnswerID == "" {
		return ""
	}
	return " (" + q.AnswerID + ")"
}

// maxStoryCells bounds the line-by-line comparison; beyond it the whole story is replaced.
const maxStoryCells = 4_000_000

// storyContext is how many unchanged lines stay around a change.
const storyContext = 1

// diffStory compares two stories line by line and keeps a line of context around each change.
func diffStory(before, after string) (lines []StoryLine, added, removed int) {
	a, b := splitLines(before), splitLines(after)
	all := lcsDiff(a, b)
	for _, l := range all {
		switch l.Kind {
		case LineAdd:
			added++
		case LineDel:
			removed++
		}
	}
	keep := make([]bool, len(all))
	for i, l := range all {
		if l.Kind == LineContext {
			continue
		}
		for near := max(i-storyContext, 0); near <= min(i+storyContext, len(all)-1); near++ {
			keep[near] = true
		}
	}
	lines = []StoryLine{}
	for i, l := range all {
		if keep[i] {
			lines = append(lines, l)
		}
	}
	return lines, added, removed
}

func lcsDiff(a, b []string) []StoryLine {
	if len(a)*len(b) > maxStoryCells {
		out := make([]StoryLine, 0, len(a)+len(b))
		for i, t := range a {
			out = append(out, StoryLine{Kind: LineDel, Old: i + 1, Text: t})
		}
		for j, t := range b {
			out = append(out, StoryLine{Kind: LineAdd, New: j + 1, Text: t})
		}
		return out
	}
	table := make([][]int, len(a)+1)
	for i := range table {
		table[i] = make([]int, len(b)+1)
	}
	for i := len(a) - 1; i >= 0; i-- {
		for j := len(b) - 1; j >= 0; j-- {
			if a[i] == b[j] {
				table[i][j] = table[i+1][j+1] + 1
			} else {
				table[i][j] = max(table[i+1][j], table[i][j+1])
			}
		}
	}
	var out []StoryLine
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		switch {
		case a[i] == b[j]:
			out = append(out, StoryLine{Kind: LineContext, Old: i + 1, New: j + 1, Text: a[i]})
			i++
			j++
		case table[i+1][j] >= table[i][j+1]:
			out = append(out, StoryLine{Kind: LineDel, Old: i + 1, Text: a[i]})
			i++
		default:
			out = append(out, StoryLine{Kind: LineAdd, New: j + 1, Text: b[j]})
			j++
		}
	}
	for ; i < len(a); i++ {
		out = append(out, StoryLine{Kind: LineDel, Old: i + 1, Text: a[i]})
	}
	for ; j < len(b); j++ {
		out = append(out, StoryLine{Kind: LineAdd, New: j + 1, Text: b[j]})
	}
	return out
}
