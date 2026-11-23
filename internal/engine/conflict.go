package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"span/internal/prompt"
	"span/internal/statement"
)

// ErrConflict is a change the host refused to merge because it conflicts with its base.
// Nothing was merged, so the outcome is known and the run can resolve it.
var ErrConflict = errors.New("the change conflicts with its base")

// ErrBehind is a change the host refused to merge because it is behind its base. Nothing was
// merged; the change has to be updated first, and span does not update it on its own.
var ErrBehind = errors.New("the change is behind its base")

// BehindReason is the hold reason for ErrBehind.
const BehindReason = "the change is behind its base; resume to merge again after it is updated"

// ErrNoConflict means the run has no conflict in the state the request needs.
var ErrNoConflict = errors.New("this run has no conflict waiting for a choice")

// Conflict states. The run holds while a conflict is choosing or failed.
const (
	ConflictProposing = "proposing"
	ConflictChoosing  = "choosing"
	ConflictApplying  = "applying"
	ConflictFailed    = "failed"
)

// Hold reasons the page reads.
const (
	ConflictChooseReason  = "conflict: choose how to resolve"
	ConflictAbortedReason = "conflict: aborted"
	ConflictTooManyReason = "conflict: too many resolutions; the base keeps moving"
)

// The page writes the person's decision beside the cursor, which the runner owns; the
// plan is the runner's own record of what each option writes.
const (
	conflictChoiceFile = "conflict-choice.json"
	conflictHintFile   = "conflict-hint"
	conflictAbortFile  = "conflict-abort"
	conflictPlanFile   = "conflict-plan.json"

	maxConflictProposals = 3
	maxResolutions       = 3
	conflictDiffCap      = 64 * 1024
	conflictPromptBudget = 1 << 20
	maxHint              = 4000
)

// Conflict is a merge of the base into the branch that git could not finish on its own.
type Conflict struct {
	Base    string          `json:"base"`
	BaseSHA string          `json:"baseSHA"`
	State   string          `json:"state"`
	Note    string          `json:"note"`
	Groups  []ConflictGroup `json:"groups"`
}

// ConflictGroup is a set of files one decision resolves.
type ConflictGroup struct {
	ID      string           `json:"id"`
	Kind    string           `json:"kind"`
	Files   []string         `json:"files"`
	Summary string           `json:"summary"`
	Options []ConflictOption `json:"options"`
	Chosen  string           `json:"chosen,omitempty"`
}

// ConflictOption is one way to resolve a group. Diff is what it would write, computed by
// span from the bytes, never taken from the agent.
type ConflictOption struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Summary string `json:"summary"`
	Risk    string `json:"risk"`
	Action  string `json:"action"`
	Command string `json:"command,omitempty"`
	Diff    string `json:"diff"`
}

// Integration is the result of merging the base into the branch without committing.
type Integration struct {
	// Clean is true when nothing conflicts. Merged is true when the clean merge brought
	// new commits and is staged, waiting for FinishMerge.
	Clean   bool
	Merged  bool
	Base    string
	BaseSHA string
	Files   []ConflictFile
}

// ConflictFile is one path git could not merge, with its three sides. Temps are the
// names git parked a side under (f~HEAD, f~<sha>) when the two sides had different types;
// they are removed by every resolution.
type ConflictFile struct {
	Path   string   `json:"path"`
	Kind   string   `json:"kind"`
	Ours   FileSide `json:"ours"`
	Theirs FileSide `json:"theirs"`
	Base   FileSide `json:"base"`
	Temps  []string `json:"temps,omitempty"`
}

// Preflight is a merge of the base computed without touching the checkout. Checked is
// false when it could not be computed; Note says why, and the host decides the merge.
type Preflight struct {
	Checked   bool
	Conflicts bool
	Base      string
	BaseSHA   string
	Note      string
}

// TreeState is what the index and the tracked tree held at one moment, so whatever an agent
// or a command changed outside the conflict can be put back.
type TreeState struct {
	Index     map[string]string
	Dirty     map[string]bool
	Untracked map[string]bool
	Ignored   map[string]bool
	// Head, Symref, Refs and Merge are the repository around the tree: the commit checked
	// out, the branch HEAD names, every local branch, and the merge state files.
	Head   string
	Symref string
	Refs   map[string]string
	Merge  map[string][]byte
	// Git are files under the git directory an agent could use to act later (hooks, config),
	// by path, with their bytes and mode.
	Git map[string]GitFile
}

// GitFile is one file under the git directory, at Path, keyed by a name like .git/config.
type GitFile struct {
	Path string
	Data []byte
	Mode os.FileMode
}

// Restored is what Restore put back, and the ignored files that were left where they are.
type Restored struct {
	Paths   []string
	Ignored []string
}

// FileSide is one version of a conflicted file. Text is empty for binary content and cut
// at the source's cap when Truncated.
type FileSide struct {
	Present   bool   `json:"present,omitempty"`
	SHA       string `json:"sha,omitempty"`
	Mode      string `json:"mode,omitempty"`
	Text      string `json:"text,omitempty"`
	Truncated bool   `json:"truncated,omitempty"`
	Binary    bool   `json:"binary,omitempty"`
}

// FileResolution is what one path becomes: a blob, a written text, nothing, or what is in
// the tree after a regenerate command.
type FileResolution struct {
	Delete  bool   `json:"delete,omitempty"`
	Blob    string `json:"blob,omitempty"`
	Mode    string `json:"mode,omitempty"`
	Write   bool   `json:"write,omitempty"`
	Content []byte `json:"content,omitempty"`
	Stage   bool   `json:"stage,omitempty"`
	// Remove are extra paths (a type conflict's temporary names) dropped from the index and
	// the tree.
	Remove []string `json:"remove,omitempty"`
}

// ConflictReply is the agent's proposal.
type ConflictReply struct {
	Groups []ConflictReplyGroup `json:"groups"`
}

// ConflictReplyGroup is one group the agent proposed.
type ConflictReplyGroup struct {
	ID      string                `json:"id"`
	Kind    string                `json:"kind"`
	Files   []string              `json:"files"`
	Summary string                `json:"summary"`
	Options []ConflictReplyOption `json:"options"`
}

// ConflictReplyOption is one option the agent proposed. Content (one file) or Contents
// (by path) carry the whole resolved text of a combine.
type ConflictReplyOption struct {
	ID       string            `json:"id"`
	Title    string            `json:"title"`
	Summary  string            `json:"summary"`
	Risk     string            `json:"risk"`
	Action   string            `json:"action"`
	Command  string            `json:"command"`
	Content  string            `json:"content"`
	Contents map[string]string `json:"contents"`
}

// integrator is a source that can merge the base into the branch and resolve what does
// not merge. It is optional: a source without it merges exactly as before.
type integrator interface {
	Integrate(ctx context.Context, base string) (Integration, error)
	Merging(ctx context.Context) (bool, error)
	Abort(ctx context.Context) error
	Resolve(ctx context.Context, files map[string]FileResolution) error
	FinishMerge(ctx context.Context, message string) (string, error)
	Preview(ctx context.Context, path string, before FileSide, res FileResolution) (string, error)
	Preflight(ctx context.Context, base string) (Preflight, error)
	Snapshot(ctx context.Context) (TreeState, error)
	Restore(ctx context.Context, before TreeState, keep []string) (Restored, error)
	Parents(ctx context.Context) ([]string, error)
}

// conflictAgent is an agent port that can read a conflict and propose options. Without it
// the options are the mechanical ones.
type conflictAgent interface {
	Propose(ctx context.Context, name, workspace, text string) (ConflictReply, error)
}

// conflictPlan is the runner's record of the conflict: the files and what each option
// writes. The page never writes it, so a chosen command is the one the card showed.
type conflictPlan struct {
	Files     []ConflictFile                   `json:"files"`
	Proposals int                              `json:"proposals"`
	Options   map[string]map[string]planOption `json:"options"`
}

type planOption struct {
	Command string                    `json:"command,omitempty"`
	Files   map[string]FileResolution `json:"files"`
}

var (
	conflictKinds   = []string{"content", "add-add", "modify-delete", "rename", "mode", "binary", "lockfile", "generated", "other"}
	conflictRisks   = []string{"low", "medium", "high"}
	conflictActions = []string{"ours", "theirs", "combine", "regenerate", "delete", "keep"}
	conflictID      = regexp.MustCompile(`^[A-Za-z0-9_-]{1,40}$`)
)

// checkBase asks, without touching the checkout, whether the base merges cleanly into the
// branch. A clean merge, or a check that could not run, leaves the merge to the host exactly
// as before; note says why nothing was checked. Only a conflict starts the conflict flow.
func (r *runner) checkBase(ctx context.Context) (string, string, bool, string, error) {
	src, ok := r.opt.Source.(integrator)
	if !ok || strings.TrimSpace(r.cur.Base) == "" {
		return "", "", false, "", nil
	}
	pre, err := src.Preflight(ctx, r.cur.Base)
	if err != nil {
		return "", "", false, "no conflict check: " + err.Error(), nil
	}
	if !pre.Checked {
		return "", "", false, "no conflict check: " + pre.Note, nil
	}
	if !pre.Conflicts {
		return "", "", false, "", nil
	}
	verdict, reason, handled, err := r.integrate(ctx, src, nil)
	return verdict, reason, handled, "", err
}

// integrate merges the base into the branch to resolve what conflicts. refusal is the
// host's conflict error when the base moved after the check. handled is false when the base
// turned out to merge cleanly and the host can merge as before.
func (r *runner) integrate(ctx context.Context, src integrator, refusal error) (string, string, bool, error) {
	r.cur.Pending = nil
	if r.cur.Resolutions >= maxResolutions {
		verdict, reason, err := r.record(merge, statement.Hold, ConflictTooManyReason, nil)
		return verdict, reason, true, err
	}
	integ, err := src.Integrate(ctx, r.cur.Base)
	if err != nil {
		verdict, reason, err := r.record(merge, statement.Hold, "could not merge "+r.cur.Base+" into the branch: "+err.Error(), nil)
		return verdict, reason, true, err
	}
	if len(integ.Files) == 0 {
		// No merge commit is made for a clean base: the host merges it, as before.
		if err := src.Abort(ctx); err != nil {
			verdict, reason, recErr := r.record(merge, statement.Hold, "the trial merge of "+integ.Base+" could not be undone: "+err.Error(), nil)
			return verdict, reason, true, recErr
		}
		if refusal != nil {
			verdict, reason, err := r.record(merge, statement.Hold, "merge conflict reported by the host, but "+integ.Base+" merges cleanly here: "+refusal.Error(), nil)
			return verdict, reason, true, err
		}
		return "", "", false, nil
	}
	r.cur.Resolutions++
	r.cur.Conflict = &Conflict{Base: integ.Base, BaseSHA: integ.BaseSHA, State: ConflictProposing, Groups: []ConflictGroup{}}
	plan := &conflictPlan{Files: integ.Files}
	if err := r.writePlan(plan); err != nil {
		return "", "", true, err
	}
	if err := r.saveConflict(); err != nil {
		return "", "", true, err
	}
	r.logLine(fmt.Sprintf("merge: %s does not merge cleanly: %d conflicted files", integ.Base, len(integ.Files)))
	verdict, reason, err := r.propose(ctx, src, plan, "")
	return verdict, reason, true, err
}

// refused handles a host that would not merge because of conflicts.
func (r *runner) refused(ctx context.Context, refusal error) (string, string, error) {
	src, ok := r.opt.Source.(integrator)
	if !ok || strings.TrimSpace(r.cur.Base) == "" {
		r.cur.Pending = nil
		return r.record(merge, statement.Hold, "merge conflict: "+refusal.Error(), nil)
	}
	verdict, reason, handled, err := r.integrate(ctx, src, refusal)
	if err != nil || handled {
		return verdict, reason, err
	}
	return r.record(merge, statement.Hold, "merge conflict: "+refusal.Error(), nil)
}

// continueConflict picks the conflict up where the person or a crash left it.
func (r *runner) continueConflict(ctx context.Context) (string, string, error) {
	src, ok := r.opt.Source.(integrator)
	if !ok {
		return r.record(merge, statement.Hold, "conflict: this source cannot resolve conflicts", nil)
	}
	if r.takeSide(conflictAbortFile) != nil {
		return r.abortConflict(ctx, src)
	}
	plan, err := r.readPlan()
	if err != nil {
		return r.failConflict("the conflict record is unreadable (" + err.Error() + "); abort it and merge again")
	}
	switch r.cur.Conflict.State {
	case ConflictApplying:
		choices := map[string]string{}
		for _, group := range r.cur.Conflict.Groups {
			choices[group.ID] = group.Chosen
		}
		return r.applyConflict(ctx, src, plan, choices)
	case ConflictProposing:
		return r.propose(ctx, src, plan, "")
	case ConflictFailed:
		return r.record(merge, statement.Hold, "conflict: "+r.cur.Conflict.Note, nil)
	}
	if raw := r.takeSide(conflictChoiceFile); raw != nil {
		var choices map[string]string
		if err := json.Unmarshal(raw, &choices); err != nil {
			r.cur.Conflict.Note = "the choice could not be read: " + err.Error()
			return r.holdChoosing()
		}
		if err := ValidateChoices(r.cur.Conflict, choices); err != nil {
			r.cur.Conflict.Note = err.Error()
			return r.holdChoosing()
		}
		return r.applyConflict(ctx, src, plan, choices)
	}
	if raw := r.takeSide(conflictHintFile); raw != nil {
		return r.propose(ctx, src, plan, strings.TrimSpace(string(raw)))
	}
	return r.holdChoosing()
}

// propose asks the build agent for options, falls back to the mechanical ones, and holds
// for a choice unless the policy may apply a low-risk proposal on its own.
func (r *runner) propose(ctx context.Context, src integrator, plan *conflictPlan, hint string) (string, string, error) {
	c := r.cur.Conflict
	c.State = ConflictProposing
	if err := r.saveConflict(); err != nil {
		return "", "", err
	}
	var groups []ConflictGroup
	var options map[string]map[string]planOption
	note := ""
	if plan.Proposals < maxConflictProposals {
		plan.Proposals++
		var fatal error
		groups, options, note, fatal = r.askAgent(ctx, src, plan, hint)
		if fatal != nil {
			if err := r.writePlan(plan); err != nil {
				return "", "", err
			}
			return r.failConflict(fatal.Error())
		}
	} else {
		note = fmt.Sprintf("the agent already proposed %d times; these are the mechanical options", maxConflictProposals)
	}
	if groups == nil {
		groups, options = mechanicalOptions(plan.Files, c.Base)
	}
	r.previewOptions(ctx, src, plan.Files, groups, options)
	plan.Options = options
	if err := r.writePlan(plan); err != nil {
		return "", "", err
	}
	c.Groups = groups
	c.Note = note
	if r.opt.Policy.ConflictMode() == "auto" {
		if choices, ok := autoChoices(groups); ok {
			c.Note = strings.TrimSpace("applied the first option of every group automatically: every one is low risk. " + note)
			return r.applyConflict(ctx, src, plan, choices)
		}
	}
	return r.holdChoosing()
}

func (r *runner) holdChoosing() (string, string, error) {
	r.cur.Conflict.State = ConflictChoosing
	if err := r.saveConflict(); err != nil {
		return "", "", err
	}
	return r.record(merge, statement.Hold, ConflictChooseReason, nil)
}

func (r *runner) failConflict(note string) (string, string, error) {
	r.cur.Conflict.State = ConflictFailed
	r.cur.Conflict.Note = note
	if err := r.saveConflict(); err != nil {
		return "", "", err
	}
	return r.record(merge, statement.Hold, "conflict: "+note, nil)
}

// autoChoices picks the first option of every group when all of them are low risk. A
// regenerate option is never applied without a person: its command must be read first.
func autoChoices(groups []ConflictGroup) (map[string]string, bool) {
	choices := map[string]string{}
	for _, group := range groups {
		if len(group.Options) == 0 {
			return nil, false
		}
		first := group.Options[0]
		if first.Risk != "low" || first.Action == "regenerate" {
			return nil, false
		}
		choices[group.ID] = first.ID
	}
	return choices, len(choices) > 0
}

// askAgent returns nil groups when the options must be the mechanical ones, with the
// reason in note. The agent runs in the checkout, so whatever it changes there is put back
// before its proposal is used: nothing reaches the merge commit that no card showed.
func (r *runner) askAgent(ctx context.Context, src integrator, plan *conflictPlan, hint string) ([]ConflictGroup, map[string]map[string]planOption, string, error) {
	proposer, ok := r.opt.Agents.(conflictAgent)
	if !ok {
		return nil, nil, "no agent can propose options here; these are the mechanical options", nil
	}
	target, err := r.pick(build)
	if err != nil {
		return nil, nil, "no agent is available (" + err.Error() + "); these are the mechanical options", nil
	}
	input := prompt.ConflictInput{Base: r.cur.Conflict.Base, Branch: r.cur.Branch, Files: promptFiles(plan.Files), Hint: hint}
	rejected := ""
	var stray, ignored []string
	for attempt := 0; attempt < 2; attempt++ {
		input.Rejected = rejected
		before, err := src.Snapshot(ctx)
		if err != nil {
			return nil, nil, "the checkout could not be read before asking the agent (" + err.Error() + "); these are the mechanical options", nil
		}
		r.logLine(fmt.Sprintf("merge: asking %s for options on %d conflicted files", target, len(plan.Files)))
		reply, err := proposer.Propose(ctx, target, r.opt.Workspace, prompt.Conflict(r.promptInput(build, ""), input))
		restored, restoreErr := src.Restore(ctx, before, nil)
		if restoreErr != nil {
			return nil, nil, "", fmt.Errorf("what the agent changed in the checkout could not be put back (%s); abort the conflict", restoreErr.Error())
		}
		stray = append(stray, restored.Paths...)
		ignored = append(ignored, restored.Ignored...)
		if err != nil {
			return nil, nil, joinNotes(strayNote(stray, ignored), "the agent did not answer ("+err.Error()+"); these are the mechanical options"), nil
		}
		groups, options, err := validateReply(reply, plan.Files)
		if err == nil {
			return groups, options, strayNote(stray, ignored), nil
		}
		rejected = err.Error()
		r.logLine("merge: the agent's options were refused: " + rejected)
	}
	return nil, nil, joinNotes(strayNote(stray, ignored), "the agent's options were refused twice ("+rejected+"); these are the mechanical options"), nil
}

func strayNote(paths, ignored []string) string {
	left := ""
	if len(ignored) > 0 {
		left = "the agent left ignored files, which span does not delete: " + strings.Join(ignored, ", ")
	}
	if len(paths) == 0 {
		return left
	}
	seen := map[string]bool{}
	var unique []string
	for _, path := range paths {
		if !seen[path] {
			seen[path] = true
			unique = append(unique, path)
		}
	}
	return joinNotes("the agent changed files outside the conflict ("+strings.Join(unique, ", ")+"); they were restored", left)
}

func joinNotes(notes ...string) string {
	var kept []string
	for _, note := range notes {
		if note != "" {
			kept = append(kept, note)
		}
	}
	return strings.Join(kept, ". ")
}

// promptFiles copies the sides for the prompt, leaving out text past the prompt budget so
// one huge file cannot crowd out the rest. A file left out cannot be combined.
func promptFiles(files []ConflictFile) []prompt.ConflictFile {
	budget := conflictPromptBudget
	side := func(s FileSide) prompt.ConflictSide {
		out := prompt.ConflictSide{Present: s.Present, Binary: s.Binary, Truncated: s.Truncated, Text: s.Text}
		if len(out.Text) > budget {
			out.Text, out.Truncated = "", true
		}
		budget -= len(out.Text)
		return out
	}
	out := make([]prompt.ConflictFile, 0, len(files))
	for _, file := range files {
		out = append(out, prompt.ConflictFile{
			Path: file.Path, Kind: file.Kind,
			Base: side(file.Base), Ours: side(file.Ours), Theirs: side(file.Theirs),
		})
	}
	return out
}

// validateReply checks the agent's proposal against the files git reported and turns it
// into groups for the page and resolutions for the plan.
func validateReply(reply ConflictReply, files []ConflictFile) ([]ConflictGroup, map[string]map[string]planOption, error) {
	byPath := map[string]ConflictFile{}
	for _, file := range files {
		byPath[file.Path] = file
	}
	if len(reply.Groups) == 0 {
		return nil, nil, errors.New("no groups")
	}
	seenFile := map[string]bool{}
	seenGroup := map[string]bool{}
	var groups []ConflictGroup
	options := map[string]map[string]planOption{}
	for _, rg := range reply.Groups {
		if !conflictID.MatchString(rg.ID) || seenGroup[rg.ID] {
			return nil, nil, fmt.Errorf("group id %q is missing, repeated, or not letters, digits and dashes", rg.ID)
		}
		seenGroup[rg.ID] = true
		if len(rg.Files) == 0 {
			return nil, nil, fmt.Errorf("group %s names no files", rg.ID)
		}
		var members []ConflictFile
		for _, path := range rg.Files {
			file, ok := byPath[path]
			if !ok {
				return nil, nil, fmt.Errorf("group %s names %q, which is not a conflicted file", rg.ID, path)
			}
			if seenFile[path] {
				return nil, nil, fmt.Errorf("%s is in more than one group", path)
			}
			seenFile[path] = true
			members = append(members, file)
		}
		if len(rg.Options) < 2 || len(rg.Options) > 4 {
			return nil, nil, fmt.Errorf("group %s has %d options; it needs 2 to 4", rg.ID, len(rg.Options))
		}
		group := ConflictGroup{
			ID: rg.ID, Kind: groupKind(members, rg.Kind), Files: append([]string{}, rg.Files...),
			Summary: clip(rg.Summary, 2000),
		}
		options[rg.ID] = map[string]planOption{}
		for _, ro := range rg.Options {
			if !conflictID.MatchString(ro.ID) {
				return nil, nil, fmt.Errorf("group %s: option id %q is not letters, digits and dashes", rg.ID, ro.ID)
			}
			if _, dup := options[rg.ID][ro.ID]; dup {
				return nil, nil, fmt.Errorf("group %s repeats option %s", rg.ID, ro.ID)
			}
			if !oneOf(ro.Risk, conflictRisks) {
				return nil, nil, fmt.Errorf("group %s option %s: risk %q is not low, medium or high", rg.ID, ro.ID, ro.Risk)
			}
			if strings.TrimSpace(ro.Title) == "" {
				return nil, nil, fmt.Errorf("group %s option %s has no title", rg.ID, ro.ID)
			}
			planned, err := resolveOption(ro, members, group.Kind)
			if err != nil {
				return nil, nil, fmt.Errorf("group %s option %s: %w", rg.ID, ro.ID, err)
			}
			options[rg.ID][ro.ID] = planned
			group.Options = append(group.Options, ConflictOption{
				ID: ro.ID, Title: clip(ro.Title, 120), Summary: clip(ro.Summary, 2000),
				Risk: effectiveRisk(ro.Action, ro.Risk), Action: ro.Action, Command: planned.Command,
			})
		}
		groups = append(groups, group)
	}
	for _, file := range files {
		if !seenFile[file.Path] {
			return nil, nil, fmt.Errorf("%s is in no group", file.Path)
		}
	}
	return groups, options, nil
}

// resolveOption turns one option into what it writes for each file of its group.
func resolveOption(ro ConflictReplyOption, members []ConflictFile, kind string) (planOption, error) {
	out := planOption{Files: map[string]FileResolution{}}
	for _, file := range members {
		if typeConflict(file) && ro.Action != "ours" && ro.Action != "theirs" {
			return planOption{}, fmt.Errorf("%s has a different type on each side; only ours or theirs can resolve it", file.Path)
		}
	}
	switch ro.Action {
	case "ours", "theirs", "delete", "keep":
		for _, file := range members {
			out.Files[file.Path] = sideResolution(file, ro.Action)
		}
	case "combine":
		for _, file := range members {
			if file.Ours.Binary || file.Theirs.Binary || file.Ours.Truncated || file.Theirs.Truncated {
				return planOption{}, fmt.Errorf("%s is binary or too large to combine", file.Path)
			}
			text, ok := ro.Contents[file.Path]
			if !ok && len(members) == 1 && ro.Content != "" {
				text, ok = ro.Content, true
			}
			if !ok {
				return planOption{}, fmt.Errorf("combine has no content for %s", file.Path)
			}
			if HasConflictMarkers([]byte(text)) {
				return planOption{}, fmt.Errorf("the content for %s still has conflict markers", file.Path)
			}
			mode := file.Ours.Mode
			if mode == "" {
				mode = file.Theirs.Mode
			}
			out.Files[file.Path] = FileResolution{Write: true, Content: []byte(text), Mode: mode, Remove: file.Temps}
		}
	case "regenerate":
		if kind != "lockfile" && kind != "generated" {
			return planOption{}, errors.New("regenerate is only for lockfile or generated files")
		}
		command, err := regenerateCommand(ro.Command)
		if err != nil {
			return planOption{}, err
		}
		out.Command = command
		for _, file := range members {
			out.Files[file.Path] = FileResolution{Stage: true, Remove: file.Temps}
		}
	default:
		return planOption{}, fmt.Errorf("action %q is not one of %s", ro.Action, strings.Join(conflictActions, ", "))
	}
	return out, nil
}

// sideResolution writes one side's blob, or removes the path when that side has none.
func sideResolution(file ConflictFile, action string) FileResolution {
	pickSide := func(side FileSide) FileResolution {
		if !side.Present {
			return FileResolution{Delete: true, Remove: file.Temps}
		}
		return FileResolution{Blob: side.SHA, Mode: side.Mode, Remove: file.Temps}
	}
	switch action {
	case "ours":
		return pickSide(file.Ours)
	case "theirs":
		return pickSide(file.Theirs)
	case "keep":
		if file.Ours.Present {
			return pickSide(file.Ours)
		}
		return pickSide(file.Theirs)
	default:
		return FileResolution{Delete: true, Remove: file.Temps}
	}
}

// typeConflict is a path whose sides are different kinds of thing (a file and a symlink or
// a directory, or a submodule): only one side or the other can win.
func typeConflict(file ConflictFile) bool {
	return len(file.Temps) > 0 || file.Ours.Mode == "160000" || file.Theirs.Mode == "160000" || file.Base.Mode == "160000"
}

// riskFloor is the least risk each action can carry. The agent's label can raise it, never
// lower it: an option that drops one side's work is never low risk, whatever it says.
var riskFloor = map[string]string{"combine": "low", "keep": "low", "ours": "medium"}

func effectiveRisk(action, said string) string {
	floor, ok := riskFloor[action]
	if !ok {
		floor = "high"
	}
	rank := map[string]int{"low": 0, "medium": 1, "high": 2}
	if rank[said] > rank[floor] {
		return said
	}
	return floor
}

// regenerators are the programs a regenerate option may run. A lockfile or a generated
// file is rebuilt by its package manager or build tool, not by anything else.
var regenerators = map[string]bool{
	"npm": true, "pnpm": true, "yarn": true, "bun": true, "npx": true, "go": true, "cargo": true,
	"poetry": true, "pip": true, "pip3": true, "uv": true, "bundle": true, "composer": true,
	"dotnet": true, "gradle": true, "mvn": true, "make": true,
}

// regenerateCommand checks a regenerate command: one allowed program by name, its arguments,
// and no shell syntax, because it runs as argv.
func regenerateCommand(command string) (string, error) {
	command = strings.TrimSpace(command)
	if command == "" || strings.ContainsAny(command, ";&|<>$`\\\n\r\"'*?(){}[]~!#") {
		return "", fmt.Errorf("command %q is empty or has shell syntax", command)
	}
	program := strings.Fields(command)[0]
	if !regenerators[program] {
		return "", fmt.Errorf("command %q does not start with a package manager or build tool span runs (npm, pnpm, yarn, bun, npx, go, cargo, poetry, pip, pip3, uv, bundle, composer, dotnet, gradle, mvn, make)", command)
	}
	return command, nil
}

// mechanicalOptions are the options span can offer without an agent: one side or the
// other, and for a deletion, keeping or deleting. Renamed paths share one group, because
// the old and new names are one decision.
func mechanicalOptions(files []ConflictFile, base string) ([]ConflictGroup, map[string]map[string]planOption) {
	if base == "" {
		base = "the base"
	}
	var buckets [][]ConflictFile
	renameAt := -1
	for _, file := range files {
		if file.Kind == "rename" {
			if renameAt < 0 {
				renameAt = len(buckets)
				buckets = append(buckets, nil)
			}
			buckets[renameAt] = append(buckets[renameAt], file)
			continue
		}
		buckets = append(buckets, []ConflictFile{file})
	}
	var groups []ConflictGroup
	options := map[string]map[string]planOption{}
	for i, members := range buckets {
		id := fmt.Sprintf("g%d", i+1)
		kind := groupKind(members, "")
		group := ConflictGroup{ID: id, Kind: kind, Summary: mechanicalSummary(members, kind, base)}
		for _, file := range members {
			group.Files = append(group.Files, file.Path)
		}
		type choice struct{ id, title, summary string }
		var choices []choice
		if kind == "modify-delete" {
			choices = []choice{
				{"keep", "Keep the file", "Keep the side that changed the file; the deletion is dropped."},
				{"delete", "Delete the file", "Accept the deletion; the other side's changes to it are dropped."},
			}
		} else {
			choices = []choice{
				{"ours", "Keep this branch's version", "Every file takes this branch's version; what " + base + " changed in them is dropped."},
				{"theirs", "Take " + base + "'s version", "Every file takes " + base + "'s version; what this branch changed in them is dropped."},
			}
			if kind == "rename" {
				choices = append(choices, choice{"keep", "Keep every name", "Keep each file under every name either side gave it."})
			}
		}
		options[id] = map[string]planOption{}
		for _, ch := range choices {
			planned := planOption{Files: map[string]FileResolution{}}
			for _, file := range members {
				planned.Files[file.Path] = sideResolution(file, ch.id)
			}
			options[id][ch.id] = planned
			group.Options = append(group.Options, ConflictOption{
				ID: ch.id, Title: ch.title, Summary: ch.summary, Risk: effectiveRisk(ch.id, "medium"), Action: ch.id,
			})
		}
		groups = append(groups, group)
	}
	return groups, options
}

func mechanicalSummary(members []ConflictFile, kind, base string) string {
	names := make([]string, 0, len(members))
	for _, file := range members {
		names = append(names, file.Path)
	}
	return fmt.Sprintf("%s: a %s conflict between this branch and %s.", strings.Join(names, ", "), kind, base)
}

// groupKind is the kind every file of the group shares, the agent's word when they
// differ and it names a known kind, and other otherwise.
func groupKind(members []ConflictFile, said string) string {
	kind := ""
	for _, file := range members {
		if kind == "" {
			kind = file.Kind
		} else if kind != file.Kind {
			kind = "mixed"
		}
	}
	if kind != "mixed" && oneOf(kind, conflictKinds) {
		return kind
	}
	if oneOf(said, conflictKinds) {
		return said
	}
	return "other"
}

// previewOptions fills each option's diff from the bytes it would write against the
// branch's version, capped so one card cannot carry a megabyte.
func (r *runner) previewOptions(ctx context.Context, src integrator, files []ConflictFile, groups []ConflictGroup, options map[string]map[string]planOption) {
	byPath := map[string]ConflictFile{}
	for _, file := range files {
		byPath[file.Path] = file
	}
	for gi := range groups {
		group := &groups[gi]
		for oi := range group.Options {
			option := &group.Options[oi]
			planned := options[group.ID][option.ID]
			var b strings.Builder
			for _, path := range group.Files {
				res := planned.Files[path]
				if res.Stage {
					fmt.Fprintf(&b, "%s is rewritten by `%s` after you choose this option.\n", path, planned.Command)
					continue
				}
				diff, err := src.Preview(ctx, path, byPath[path].Ours, res)
				if err != nil {
					fmt.Fprintf(&b, "%s: no preview (%s)\n", path, err.Error())
					continue
				}
				b.WriteString(diff)
				if res.Mode != "" && byPath[path].Ours.Present && res.Mode != byPath[path].Ours.Mode {
					fmt.Fprintf(&b, "%s: mode %s becomes %s\n", path, byPath[path].Ours.Mode, res.Mode)
				}
			}
			option.Diff = capDiff(b.String())
		}
	}
}

func capDiff(diff string) string {
	if len(diff) <= conflictDiffCap {
		return diff
	}
	return diff[:conflictDiffCap] + "\n[diff cut at 64 KiB]\n"
}

// applyConflict writes the chosen options, commits the merge and sends the run back to
// prove. Nothing is pushed here: the merge commit is published right before the host merge,
// once prove, QA and inspect passed on it. It is safe to run again: a merge already committed
// by an earlier wake is not committed twice.
func (r *runner) applyConflict(ctx context.Context, src integrator, plan *conflictPlan, choices map[string]string) (string, string, error) {
	c := r.cur.Conflict
	for i := range c.Groups {
		c.Groups[i].Chosen = choices[c.Groups[i].ID]
	}
	c.State = ConflictApplying
	if err := r.saveConflict(); err != nil {
		return "", "", err
	}
	summary := describeChoices(c)
	merging, err := src.Merging(ctx)
	if err != nil {
		return r.retryChoice("the merge state could not be read: " + err.Error())
	}
	if !merging {
		head, headErr := r.opt.Source.Head(ctx)
		if headErr == nil && head != "" && head != r.cur.GitSHA {
			if note := r.foreignMerge(ctx, src); note != "" {
				return r.failConflict(note)
			}
			return r.resolved(head, "resolved the conflict with "+c.Base+" ("+summary+")")
		}
		return r.failConflict("the merge is no longer in progress in the checkout; abort the conflict and run the merge again")
	}
	files := map[string]FileResolution{}
	for _, group := range c.Groups {
		planned, ok := plan.Options[group.ID][group.Chosen]
		if !ok {
			return r.retryChoice("group " + group.ID + " has no option " + group.Chosen)
		}
		if planned.Command != "" {
			if err := r.regenerate(ctx, src, plan, planned); err != nil {
				return r.retryChoice(err.Error())
			}
		}
		for path, res := range planned.Files {
			files[path] = res
		}
	}
	if err := src.Resolve(ctx, files); err != nil {
		return r.retryChoice("the choice could not be applied: " + err.Error())
	}
	sha, err := src.FinishMerge(ctx, r.mergeMessage(c.Base, "resolve conflicts"))
	if err != nil {
		return r.retryChoice("the merge could not be committed: " + err.Error())
	}
	if note := r.foreignMerge(ctx, src); note != "" {
		return r.failConflict(note)
	}
	return r.resolved(sha, "resolved the conflict with "+c.Base+" ("+summary+")")
}

// regenerate puts the branch's version of its files back, runs the command the person read
// on the card as argv (never a shell) in the checkout, and leaves its output to be staged. A
// command that changed anything outside its own files is undone and refused.
func (r *runner) regenerate(ctx context.Context, src integrator, plan *conflictPlan, planned planOption) error {
	reset := map[string]FileResolution{}
	var own []string
	for _, file := range plan.Files {
		if _, ok := planned.Files[file.Path]; ok {
			reset[file.Path] = sideResolution(file, "keep")
			own = append(own, file.Path)
		}
	}
	sort.Strings(own)
	if err := src.Resolve(ctx, reset); err != nil {
		return fmt.Errorf("the files could not be reset before %s: %w", planned.Command, err)
	}
	before, err := src.Snapshot(ctx)
	if err != nil {
		return fmt.Errorf("the checkout could not be read before %s: %w", planned.Command, err)
	}
	r.logLine("merge: running " + planned.Command + " to regenerate the conflicted files")
	out, runErr := r.opt.Check.Run(ctx, planned.Command, r.opt.Workspace)
	restored, err := src.Restore(ctx, before, own)
	if err != nil {
		return fmt.Errorf("what %s changed could not be put back: %w", planned.Command, err)
	}
	changed := restored.Paths
	if len(changed) > 0 {
		return fmt.Errorf("%s changed files outside its group (%s); they were restored", planned.Command, strings.Join(changed, ", "))
	}
	if runErr != nil {
		return fmt.Errorf("%s did not run: %w", planned.Command, runErr)
	}
	if out.ExitCode != 0 {
		return fmt.Errorf("%s exited %d", planned.Command, out.ExitCode)
	}
	return nil
}

// retryChoice puts the conflict back in front of the person with what went wrong.
func (r *runner) retryChoice(note string) (string, string, error) {
	c := r.cur.Conflict
	for i := range c.Groups {
		c.Groups[i].Chosen = ""
	}
	c.Note = note
	return r.holdChoosing()
}

// resolved records the merge commit and sends the run back to prove: the statements already
// sealed describe the head before the merge. The commit stays local until doMerge pushes it.
func (r *runner) resolved(sha, what string) (string, string, error) {
	r.cur.GitSHA = sha
	r.cur.Unpushed = true
	r.cur.Conflict = nil
	r.clearConflictFiles()
	reason := what + "; prove runs again on " + short(sha)
	if _, _, err := r.record(merge, statement.Hold, reason, func(s *statement.Statement) {
		s.GitSHA = sha
		s.ChangeURL = r.cur.ChangeURL
	}); err != nil {
		return "", "", err
	}
	return "reprove", reason, nil
}

// foreignMerge is empty when HEAD is the merge span prepared: its parents are exactly the
// head the run was on and the base it merged. Anything else (an agent's commit, an amend, a
// reset) is never pushed.
func (r *runner) foreignMerge(ctx context.Context, src integrator) string {
	parents, err := src.Parents(ctx)
	if err != nil {
		return "the merge commit could not be read: " + err.Error()
	}
	if len(parents) != 2 || parents[0] != r.cur.GitSHA || parents[1] != r.cur.Conflict.BaseSHA {
		return "the branch has a commit that is not the merge span prepared; abort the conflict and look at the branch"
	}
	return ""
}

func (r *runner) abortConflict(ctx context.Context, src integrator) (string, string, error) {
	if err := src.Abort(ctx); err != nil {
		return r.failConflict("git merge --abort failed: " + err.Error())
	}
	if head, err := r.opt.Source.Head(ctx); err == nil && r.cur.GitSHA != "" && head != r.cur.GitSHA {
		return r.failConflict("the merge was aborted but HEAD is " + short(head) + ", not the branch head " + short(r.cur.GitSHA) + " the conflict started from")
	}
	r.cur.Conflict = nil
	r.clearConflictFiles()
	if err := r.save(); err != nil {
		return "", "", err
	}
	return r.record(merge, statement.Hold, ConflictAbortedReason, nil)
}

func (r *runner) mergeMessage(base, suffix string) string {
	branch := r.cur.Branch
	if branch == "" {
		if name, err := r.opt.Source.Branch(context.Background()); err == nil {
			branch = name
		}
	}
	message := "merge " + base + " into " + branch
	if suffix != "" {
		message += ": " + suffix
	}
	return message
}

func describeChoices(c *Conflict) string {
	parts := make([]string, 0, len(c.Groups))
	for _, group := range c.Groups {
		title := group.Chosen
		for _, option := range group.Options {
			if option.ID == group.Chosen {
				title = option.Title
			}
		}
		parts = append(parts, group.ID+": "+title)
	}
	return strings.Join(parts, "; ")
}

func (r *runner) saveConflict() error {
	if err := r.save(); err != nil {
		return err
	}
	if r.cur.Conflict != nil {
		snapshot := cloneConflict(r.cur.Conflict)
		r.publish(Event{Type: "conflict", Conflict: snapshot})
	}
	return nil
}

func cloneConflict(c *Conflict) *Conflict {
	out := *c
	out.Groups = make([]ConflictGroup, len(c.Groups))
	for i, group := range c.Groups {
		group.Files = append([]string{}, group.Files...)
		group.Options = append([]ConflictOption{}, group.Options...)
		out.Groups[i] = group
	}
	return &out
}

func (r *runner) writePlan(plan *conflictPlan) error {
	raw, err := json.Marshal(plan)
	if err != nil {
		return err
	}
	path := filepath.Join(r.spanDir(), conflictPlanFile)
	if err := writeAtomic(path, raw); err != nil {
		return err
	}
	// The plan holds file contents; only this user reads it.
	return os.Chmod(path, 0o600)
}

func (r *runner) readPlan() (*conflictPlan, error) {
	raw, err := os.ReadFile(filepath.Join(r.spanDir(), conflictPlanFile))
	if err != nil {
		return nil, err
	}
	var plan conflictPlan
	if err := json.Unmarshal(raw, &plan); err != nil {
		return nil, err
	}
	return &plan, nil
}

// takeSide reads and removes one file the page wrote. nil means it was not there.
func (r *runner) takeSide(name string) []byte {
	path := filepath.Join(r.spanDir(), name)
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	_ = os.Remove(path)
	return raw
}

func (r *runner) clearConflictFiles() {
	for _, name := range []string{conflictChoiceFile, conflictHintFile, conflictAbortFile, conflictPlanFile} {
		_ = os.Remove(filepath.Join(r.spanDir(), name))
	}
}

// ValidateChoices checks that a choice names every group of the conflict once, with an
// option that group offers.
func ValidateChoices(c *Conflict, choices map[string]string) error {
	if c == nil {
		return ErrNoConflict
	}
	known := map[string]bool{}
	for _, group := range c.Groups {
		known[group.ID] = true
		chosen, ok := choices[group.ID]
		if !ok || chosen == "" {
			return fmt.Errorf("group %s has no choice", group.ID)
		}
		found := false
		for _, option := range group.Options {
			if option.ID == chosen {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("group %s has no option %q", group.ID, chosen)
		}
	}
	for id := range choices {
		if !known[id] {
			return fmt.Errorf("there is no group %q", id)
		}
	}
	return nil
}

// ChooseConflict records the person's choice for every group. The run applies it when it
// wakes.
func ChooseConflict(root, id string, choices map[string]string) error {
	cur, dir, err := choosingConflict(root, id, ConflictChoosing)
	if err != nil {
		return err
	}
	if err := ValidateChoices(cur.Conflict, choices); err != nil {
		return err
	}
	raw, err := json.Marshal(choices)
	if err != nil {
		return err
	}
	return writeAtomic(filepath.Join(dir, conflictChoiceFile), raw)
}

// HintConflict records the person's words for the agent's next proposal.
func HintConflict(root, id, text string) error {
	text = strings.TrimSpace(text)
	if text == "" {
		return errors.New("the hint is empty")
	}
	if len(text) > maxHint {
		return fmt.Errorf("the hint is longer than %d characters", maxHint)
	}
	_, dir, err := choosingConflict(root, id, ConflictChoosing)
	if err != nil {
		return err
	}
	return writeAtomic(filepath.Join(dir, conflictHintFile), []byte(text+"\n"))
}

// AbortConflict asks the run to undo the merge and hold.
func AbortConflict(root, id string) error {
	_, dir, err := choosingConflict(root, id, ConflictChoosing, ConflictFailed)
	if err != nil {
		return err
	}
	return writeAtomic(filepath.Join(dir, conflictAbortFile), []byte("aborted by the page\n"))
}

func choosingConflict(root, id string, states ...string) (Cursor, string, error) {
	if !validID(id) {
		return Cursor{}, "", os.ErrNotExist
	}
	dir := filepath.Join(root, id)
	cur, err := readCursorFile(dir)
	if err != nil {
		return Cursor{}, "", err
	}
	if cur.Conflict == nil || !oneOf(cur.Conflict.State, states) || cur.Done || cur.Stop {
		return Cursor{}, "", ErrNoConflict
	}
	return cur, dir, nil
}

// HasConflictMarkers reports a text that still carries a git conflict: a start marker line,
// then a separator line, then an end marker line, in that order. A document that quotes one
// of them is not a conflict. Binary content is never checked: its bytes are not lines.
func HasConflictMarkers(data []byte) bool {
	if bytes.IndexByte(data, 0) >= 0 {
		return false
	}
	state := 0
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSuffix(line, "\r")
		switch {
		case state == 0 && (line == "<<<<<<<" || strings.HasPrefix(line, "<<<<<<< ")):
			state = 1
		case state == 1 && line == "=======":
			state = 2
		case state == 2 && (line == ">>>>>>>" || strings.HasPrefix(line, ">>>>>>> ")):
			return true
		}
	}
	return false
}

func oneOf(value string, set []string) bool {
	for _, item := range set {
		if item == value {
			return true
		}
	}
	return false
}

func clip(text string, max int) string {
	text = strings.TrimSpace(text)
	if len(text) <= max {
		return text
	}
	return strings.ToValidUTF8(text[:max], "") + "…"
}

func short(sha string) string {
	if len(sha) > 12 {
		return sha[:12]
	}
	return sha
}
