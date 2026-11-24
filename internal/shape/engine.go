package shape

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"span/internal/agent"
	"span/internal/harness"
)

// maxNewQuestions is how many new open questions one turn may add. The rest are dropped.
const maxNewQuestions = 3

// Runner runs the interview agent in dir with prompt and returns its reply.
type Runner func(ctx context.Context, dir, prompt string) (string, error)

// Engine runs turns. Run is required. Harness is the harness directory (empty means look
// it up). Now defaults to time.Now.
type Engine struct {
	Run     Runner
	Harness string
	Now     func() time.Time
}

// Answer is the person's reply to one question.
type Answer struct {
	Question string `json:"question"`
	Text     string `json:"text"`
}

type agentReply struct {
	Map       Map        `json:"map"`
	Split     []Split    `json:"split"`
	Proposals []Proposal `json:"proposals"`
	Story     string     `json:"story"`
}

func (e *Engine) now() time.Time {
	if e.Now != nil {
		return e.Now()
	}
	return time.Now()
}

// Start creates a session and runs its first turn. A failed first turn leaves nothing on disk.
func (e *Engine) Start(ctx context.Context, in StartInput) (*Session, error) {
	if strings.TrimSpace(in.Idea) == "" {
		return nil, errors.New("idea is required")
	}
	s, err := newSession(in, e.now())
	if err != nil {
		return nil, err
	}
	if err := e.turn(ctx, s); err != nil {
		return nil, err
	}
	if err := Save(s); err != nil {
		return nil, err
	}
	return s, nil
}

// Answer records the person's answers and accepted assumptions, then runs the next turn.
// The answers are saved before the agent runs, so a failed turn does not lose them.
func (e *Engine) Answer(ctx context.Context, workspace, id string, answers []Answer, assume []string) (*Session, error) {
	s, err := loadOpen(workspace, id)
	if err != nil {
		return nil, err
	}
	if err := applyAnswers(s, answers, assume); err != nil {
		return nil, err
	}
	s.Status = computeStatus(s)
	if err := Save(s); err != nil {
		return nil, err
	}
	if err := e.turn(ctx, s); err != nil {
		return nil, err
	}
	if err := Save(s); err != nil {
		return nil, err
	}
	return s, nil
}

func loadOpen(workspace, id string) (*Session, error) {
	s, err := Load(workspace, id)
	if err != nil {
		return nil, err
	}
	if s.Status == StatusSealed {
		return nil, fail("shape_sealed", "session %s is sealed", id)
	}
	return s, nil
}

func findQuestion(s *Session, id string) *Question {
	for i := range s.Map.Questions {
		if s.Map.Questions[i].ID == id {
			return &s.Map.Questions[i]
		}
	}
	return nil
}

func openQuestion(s *Session, id string) (*Question, error) {
	q := findQuestion(s, id)
	if q == nil || q.Status != QuestionOpen {
		return nil, fmt.Errorf("question %q is not open", id)
	}
	return q, nil
}

func applyAnswers(s *Session, answers []Answer, assume []string) error {
	for _, a := range answers {
		q, err := openQuestion(s, a.Question)
		if err != nil {
			return err
		}
		if strings.TrimSpace(a.Text) == "" {
			return fmt.Errorf("answer to %s is empty", a.Question)
		}
		q.Status, q.Answer = QuestionAnswered, a.Text
	}
	for _, id := range assume {
		q, err := openQuestion(s, id)
		if err != nil {
			return err
		}
		if strings.TrimSpace(q.Assume) == "" {
			return fmt.Errorf("question %s has no assumption to accept", id)
		}
		q.Status, q.Answer = QuestionAssumed, q.Assume
	}
	return nil
}

func computeStatus(s *Session) string {
	if len(s.Split) > 0 {
		return StatusSplit
	}
	if len(openIDs(s)) == 0 && strings.TrimSpace(s.Story) != "" {
		return StatusReady
	}
	return StatusInterviewing
}

func openIDs(s *Session) []string {
	var ids []string
	for _, q := range s.Map.Questions {
		if q.Status == QuestionOpen {
			ids = append(ids, q.ID)
		}
	}
	return ids
}

func repoDir(s *Session) string {
	switch {
	case s.Repo == "":
		return s.Workspace
	case filepath.IsAbs(s.Repo):
		return s.Repo
	default:
		return filepath.Join(s.Workspace, s.Repo)
	}
}

// turn runs the agent once and merges its reply into s. s is untouched when it fails.
func (e *Engine) turn(ctx context.Context, s *Session) error {
	dir := repoDir(s)
	before, err := gitState(ctx, dir)
	if err != nil {
		return err
	}
	prompt, err := e.prompt(s)
	if err != nil {
		return err
	}
	raw, runErr := e.Run(ctx, dir, prompt)
	if runErr != nil {
		runErr = fmt.Errorf("interview agent: %w", runErr)
	}
	after, err := gitState(ctx, dir)
	if err != nil {
		return errors.Join(runErr, err)
	}
	if changed := diffState(before, after); len(changed) > 0 {
		return errors.Join(runErr, fail("shape_repo_changed", "the interview changed the repository: %s", strings.Join(changed, ", ")))
	}
	if runErr != nil {
		return runErr
	}
	var r agentReply
	if err := agent.ParseJSON(raw, &r); err != nil {
		return err
	}
	merged, err := merge(s, r)
	if err != nil {
		return err
	}
	merged.Turns = append(merged.Turns, Turn{N: len(s.Turns) + 1, At: e.now().UTC(), Asked: newQuestionIDs(s, merged)})
	merged.Status = computeStatus(merged)
	*s = *merged
	return nil
}

func newQuestionIDs(old, merged *Session) []string {
	asked := []string{}
	for _, q := range merged.Map.Questions {
		if findQuestion(old, q.ID) == nil {
			asked = append(asked, q.ID)
		}
	}
	return asked
}

func checkSource(kind, id string, src Source) error {
	switch src.Kind {
	case SourcePrompt, SourceAnswer, SourceCode, SourceContext:
		return nil
	}
	return fmt.Errorf("%s %s has no valid source (kind must be prompt, answer, code or context, got %q)", kind, id, src.Kind)
}

// merge applies the agent's reply over a copy of s. Span is authoritative: known questions
// are kept as they are, new ones arrive open and capped, and facts need a source.
func merge(s *Session, r agentReply) (*Session, error) {
	for _, rule := range r.Map.Rules {
		if err := checkSource("rule", rule.ID, rule.Source); err != nil {
			return nil, err
		}
	}
	for _, ex := range r.Map.Examples {
		if err := checkSource("example", ex.ID, ex.Source); err != nil {
			return nil, err
		}
	}
	out := *s
	out.Map = Map{Rules: nonNil(r.Map.Rules), Examples: nonNil(r.Map.Examples), Questions: append([]Question{}, s.Map.Questions...)}
	added := 0
	for _, q := range r.Map.Questions {
		if findQuestion(&out, q.ID) != nil || q.ID == "" || added >= maxNewQuestions {
			continue
		}
		q.Status, q.Answer = QuestionOpen, ""
		out.Map.Questions = append(out.Map.Questions, q)
		added++
	}
	out.Split = nonNil(r.Split)
	out.Story = r.Story
	out.Proposals = append([]Proposal{}, s.Proposals...)
	for _, p := range r.Proposals {
		if p.ID == "" || findProposal(&out, p.ID) != nil {
			continue
		}
		p.Status = ProposalProposed
		out.Proposals = append(out.Proposals, p)
	}
	out.Turns = append([]Turn{}, s.Turns...)
	return &out, nil
}

func nonNil[T any](in []T) []T {
	if in == nil {
		return []T{}
	}
	return in
}

func findProposal(s *Session, id string) *Proposal {
	for i := range s.Proposals {
		if s.Proposals[i].ID == id {
			return &s.Proposals[i]
		}
	}
	return nil
}

func (e *Engine) skillPaths() (string, string) {
	root := e.Harness
	if root == "" {
		root = harness.Root()
	}
	base := filepath.Join("harness", "skills", "user-story")
	if root != "" {
		base = filepath.Join(root, "skills", "user-story")
	}
	return filepath.Join(base, "SKILL.md"), filepath.Join(base, "references", "interview.md")
}

const replySchema = `{"map":{"rules":[],"examples":[],"questions":[]},"split":[],"proposals":[],"story":""}`

// elementShapes describes the items of the reply arrays.
const elementShapes = `rule {"id":"R1","text":"","source":{"kind":"prompt|answer|code|context","ref":""}}
example {"id":"E1","rule":"R1","text":"","source":{"kind":"prompt|answer|code|context","ref":""}}
question {"id":"Q1","text":"","blocks":"","assume":""}
split item {"story":"","leftOut":""}
proposal {"id":"P1","section":"","text":""}`

func (e *Engine) prompt(s *Session) (string, error) {
	skill, interview := e.skillPaths()
	state, err := json.MarshalIndent(s.Map, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode map: %w", err)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "You are the interviewer of a user-story shaping session (Example Mapping).\n")
	fmt.Fprintf(&b, "Read these files before answering and follow them; their contents are not repeated here:\n- %s\n- %s\n\n", skill, interview)
	fmt.Fprintf(&b, "Idea:\n%s\n\n", s.Idea)
	fmt.Fprintf(&b, "Current map (question statuses and answers are final; Span owns them):\n%s\n\n", state)
	ctxText, err := os.ReadFile(ContextPath(s.Workspace))
	switch {
	case err == nil:
		fmt.Fprintf(&b, "Product context (%s):\n%s\n\n", ContextPath(s.Workspace), ctxText)
	case !errors.Is(err, fs.ErrNotExist):
		return "", fmt.Errorf("read context: %w", err)
	}
	if len(s.Split) > 0 || strings.TrimSpace(s.Story) != "" {
		fmt.Fprintf(&b, "Current story draft:\n%s\n\n", s.Story)
	}
	b.WriteString("Rules:\n")
	b.WriteString("- You must not modify any file. Inspect the repository read-only; Span fails the turn if anything changes.\n")
	b.WriteString("- Every rule and example needs a source: prompt, answer, code (ref is the path) or context.\n")
	b.WriteString("- Ask at most 3 new questions. Never reopen or answer an existing question. Give each an assume text when a safe default exists.\n")
	b.WriteString("- Return the full current rules and examples every turn. Fill story only when no question is open; fill split instead when the map holds more than one outcome that could ship separately.\n")
	b.WriteString("- Propose product-context additions in proposals; they are written only if the person accepts.\n\n")
	fmt.Fprintf(&b, "Reply with one JSON object and nothing else, exactly this shape:\n%s\nItems of the arrays:\n%s\n", replySchema, elementShapes)
	return b.String(), nil
}

// ContextPath is the product context file of a workspace. Only AcceptProposals writes it.
func ContextPath(workspace string) string {
	return filepath.Join(workspace, "span", "context", "product.md")
}

// gitState fingerprints every path git reports as changed: its status code plus the
// sha256 of its bytes, so a second edit of an already-dirty file is seen.
func gitState(ctx context.Context, dir string) (map[string]string, error) {
	top, err := gitOutput(ctx, dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, err
	}
	out, err := gitOutput(ctx, dir, "status", "--porcelain", "--untracked-files=all")
	if err != nil {
		return nil, err
	}
	root := strings.TrimSpace(top)
	state := map[string]string{}
	for _, l := range strings.Split(out, "\n") {
		if strings.TrimSpace(l) == "" {
			continue
		}
		path := statusPath(l)
		state[path] = l[:2] + " " + fingerprint(filepath.Join(root, path))
	}
	return state, nil
}

func gitOutput(ctx context.Context, dir string, args ...string) (string, error) {
	out, err := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...).Output()
	if err != nil {
		return "", fail("shape_not_git", "%s is not a git repository, the interview cannot be checked as read-only: %v", dir, err)
	}
	return string(out), nil
}

// fingerprint is the sha256 of a file, or a marker when there is no regular file to hash.
func fingerprint(path string) string {
	info, err := os.Lstat(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return "deleted"
	case err != nil:
		return "unreadable: " + err.Error()
	case !info.Mode().IsRegular():
		return "not-a-file"
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return "unreadable: " + err.Error()
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func diffState(before, after map[string]string) []string {
	var paths []string
	for p, v := range before {
		if after[p] != v {
			paths = append(paths, p)
		}
	}
	for p := range after {
		if _, ok := before[p]; !ok {
			paths = append(paths, p)
		}
	}
	sort.Strings(paths)
	return paths
}

// statusPath is the path of a porcelain line: the new name of a rename, unquoted.
func statusPath(line string) string {
	if len(line) <= 3 {
		return line
	}
	p := line[3:]
	if _, to, ok := strings.Cut(p, " -> "); ok {
		p = to
	}
	if unq, err := strconv.Unquote(p); err == nil {
		return unq
	}
	return p
}

// AcceptProposals writes each accepted proposal under its section of the product context
// file and marks it accepted. It is the only code that writes that file.
func (e *Engine) AcceptProposals(workspace, id string, ids []string) (*Session, error) {
	s, err := loadOpen(workspace, id)
	if err != nil {
		return nil, err
	}
	for _, pid := range ids {
		p := findProposal(s, pid)
		if p == nil {
			return nil, fmt.Errorf("proposal %q does not exist", pid)
		}
	}
	for _, pid := range ids {
		p := findProposal(s, pid)
		if p.Status == ProposalAccepted {
			continue
		}
		if err := appendContext(workspace, p.Section, p.Text); err != nil {
			return nil, err
		}
		p.Status = ProposalAccepted
		if err := Save(s); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func appendContext(workspace, section, text string) error {
	path := ContextPath(workspace)
	raw, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("read context: %w", err)
	}
	lines := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	if len(raw) == 0 {
		lines = nil
	}
	heading := "## " + strings.TrimSpace(section)
	start := -1
	for i, l := range lines {
		if strings.TrimSpace(l) == heading {
			start = i
			break
		}
	}
	entry := strings.Split(strings.TrimRight(text, "\n"), "\n")
	var out []string
	if start < 0 {
		out = append(out, lines...)
		if len(out) > 0 {
			out = append(out, "")
		}
		out = append(out, heading, "")
		out = append(out, entry...)
	} else {
		end := len(lines)
		for i := start + 1; i < len(lines); i++ {
			if strings.HasPrefix(lines[i], "## ") {
				end = i
				break
			}
		}
		last := end
		for last > start+1 && strings.TrimSpace(lines[last-1]) == "" {
			last--
		}
		out = append(out, lines[:last]...)
		out = append(out, entry...)
		if end < len(lines) {
			out = append(out, "")
		}
		out = append(out, lines[end:]...)
	}
	return writeAtomic(path, []byte(strings.Join(out, "\n")+"\n"))
}

// Seal freezes a ready session: it stores the story and its sha256 digest.
func (e *Engine) Seal(workspace, id string) (*Session, error) {
	s, err := loadOpen(workspace, id)
	if err != nil {
		return nil, err
	}
	if s.Status != StatusReady {
		if open := openIDs(s); len(open) > 0 {
			return nil, fail("shape_not_ready", "open questions: %s", strings.Join(open, ", "))
		}
		return nil, fail("shape_not_ready", "status is %s, not ready", s.Status)
	}
	sum := sha256.Sum256([]byte(s.Story))
	s.Digest = hex.EncodeToString(sum[:])
	s.Status = StatusSealed
	if err := Save(s); err != nil {
		return nil, err
	}
	return s, nil
}

// SetJira stores the issue created for a sealed story. It is the only change a sealed
// session accepts.
func (e *Engine) SetJira(workspace, id string, ref JiraRef) (*Session, error) {
	s, err := Load(workspace, id)
	if err != nil {
		return nil, err
	}
	if s.Status != StatusSealed {
		return nil, fail("shape_not_sealed", "session %s is not sealed", id)
	}
	s.Jira = &ref
	if err := Save(s); err != nil {
		return nil, err
	}
	return s, nil
}
