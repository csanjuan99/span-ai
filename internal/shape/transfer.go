package shape

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// SourceImport marks a rule or example that came from a file a person brought, with the
// line it was read from as its ref.
const SourceImport = "import"

const (
	// maxImportBytes caps a markdown file; a story is prose, never a data dump.
	maxImportBytes = 256 << 10
	// shareFile holds the read-only links of one session, next to session.json.
	shareFile = "shares.json"
	// defaultShareDays is how long a link lives when the person does not say.
	defaultShareDays = 7
	maxShareDays     = 90
)

// Export formats.
const (
	// FormatMarkdown is the story as written, with optional appendices.
	FormatMarkdown = "md"
	// FormatJSON is the story with its rules, examples, sources and digest, for tools.
	FormatJSON = "json"
)

// ContextNote is a section of an imported file that is kept as context and not read as a rule.
type ContextNote struct {
	Heading string `json:"heading"`
	Line    int    `json:"line"`
	Text    string `json:"text"`
}

// MarkdownPreview is what Span understood from a markdown file, before a session starts.
type MarkdownPreview struct {
	Name     string        `json:"name"`
	Bytes    int           `json:"bytes"`
	Title    string        `json:"title"`
	Rules    []Rule        `json:"rules"`
	Examples []Example     `json:"examples"`
	Notes    []ContextNote `json:"notes"`
	// Reads says in plain words how each part of the file was read, one line each.
	Reads []string `json:"reads"`
}

var (
	headingPattern = regexp.MustCompile(`^(#{1,6})\s+(.+?)\s*#*$`)
	bulletPattern  = regexp.MustCompile(`^\s*(?:[-*+]|\d+[.)])\s+(.+)$`)
	// contextHeadings are sections that describe the story and do not state a rule.
	contextHeadings = map[string]bool{
		"notes": true, "note": true, "context": true, "background": true, "references": true,
		"links": true, "out of scope": true, "open questions": true, "glossary": true,
	}
)

// ParseMarkdown reads a markdown story: each heading below the title is a rule, each bullet
// under it an example with its line, and a section such as Notes is kept as context. It
// changes nothing; the preview is what a session would start from.
func ParseMarkdown(name, text string) (MarkdownPreview, error) {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	switch {
	case strings.TrimSpace(text) == "":
		return MarkdownPreview{}, fail("invalid", "the file is empty")
	case len(text) > maxImportBytes:
		return MarkdownPreview{}, fail("invalid", "the file is %d KB; at most %d KB", len(text)/1024, maxImportBytes/1024)
	case strings.ContainsRune(text, 0):
		return MarkdownPreview{}, fail("invalid", "the file is not text; markdown only")
	}
	name = filepath.Base(strings.TrimSpace(name))
	if name == "" || name == "." {
		name = "story.md"
	}
	out := MarkdownPreview{Name: name, Bytes: len(text), Rules: []Rule{}, Examples: []Example{}, Notes: []ContextNote{}}
	var (
		current  *Rule
		inNotes  *ContextNote
		titleSet bool
		fence    bool
	)
	for i, raw := range strings.Split(text, "\n") {
		line := i + 1
		ref := fmt.Sprintf("%s:%d", name, line)
		trimmed := strings.TrimSpace(raw)
		if strings.HasPrefix(trimmed, "```") {
			fence = !fence
			continue
		}
		if fence {
			continue
		}
		if m := headingPattern.FindStringSubmatch(trimmed); m != nil {
			heading := strings.TrimSpace(m[2])
			if !titleSet && len(m[1]) == 1 {
				out.Title, titleSet = heading, true
				current, inNotes = nil, nil
				continue
			}
			if contextHeadings[strings.ToLower(heading)] {
				out.Notes = append(out.Notes, ContextNote{Heading: heading, Line: line})
				inNotes, current = &out.Notes[len(out.Notes)-1], nil
				continue
			}
			out.Rules = append(out.Rules, Rule{
				ID: fmt.Sprintf("R%d", len(out.Rules)+1), Text: heading, Source: Source{Kind: SourceImport, Ref: ref},
			})
			current, inNotes = &out.Rules[len(out.Rules)-1], nil
			continue
		}
		if trimmed == "" {
			continue
		}
		switch {
		case inNotes != nil:
			inNotes.Text = strings.TrimSpace(inNotes.Text + "\n" + trimmed)
		case current != nil:
			if m := bulletPattern.FindStringSubmatch(raw); m != nil {
				out.Examples = append(out.Examples, Example{
					ID: fmt.Sprintf("E%d", len(out.Examples)+1), Rule: current.ID, Text: strings.TrimSpace(m[1]),
					Source: Source{Kind: SourceImport, Ref: ref},
				})
			} else {
				current.Text = strings.TrimSpace(current.Text + ". " + trimmed)
			}
		}
	}
	if len(out.Rules) == 0 {
		return MarkdownPreview{}, fail("invalid", "no heading below the title to read as a rule; write each rule as a heading")
	}
	out.Reads = readLines(out)
	return out, nil
}

func readLines(p MarkdownPreview) []string {
	ids := func(n int, prefix string) string {
		switch {
		case n == 1:
			return prefix + "1"
		case n == 2:
			return prefix + "1, " + prefix + "2"
		}
		return prefix + "1.." + prefix + fmt.Sprint(n)
	}
	out := []string{
		fmt.Sprintf("%s read as rules (%s)", plural(len(p.Rules), "heading", "headings"), ids(len(p.Rules), "R")),
	}
	if len(p.Examples) > 0 {
		out = append(out, fmt.Sprintf("%s read as examples, each with its line", plural(len(p.Examples), "bullet", "bullets")))
	}
	for _, n := range p.Notes {
		out = append(out, fmt.Sprintf("section “%s” (line %d) could not be read as a rule; it is kept as context, not as a rule", n.Heading, n.Line))
	}
	return out
}

// ImportInput starts a session from a markdown file.
type ImportInput struct {
	Repo     string
	Agent    string
	Model    string
	Name     string
	Markdown string
}

// ImportMarkdown creates a session whose map already holds what the file said, with the
// sources marked "import". No agent runs: the caller resumes the session to start the
// interview about what the file leaves open.
func ImportMarkdown(workspace string, in ImportInput) (*Session, MarkdownPreview, error) {
	preview, err := ParseMarkdown(in.Name, in.Markdown)
	if err != nil {
		return nil, preview, err
	}
	idea := strings.TrimSpace(in.Markdown)
	if first, rest, ok := strings.Cut(idea, "\n"); strings.HasPrefix(first, "#") {
		idea = strings.TrimSpace(strings.TrimLeft(first, "# "))
		if ok && strings.TrimSpace(rest) != "" {
			idea += "\n\n" + strings.TrimSpace(rest)
		}
	}
	s, err := Create(workspace, StartInput{Workspace: workspace, Repo: in.Repo, Idea: idea, Agent: in.Agent, Model: in.Model})
	if err != nil {
		return nil, preview, err
	}
	s.Map = Map{Rules: preview.Rules, Examples: preview.Examples, Questions: []Question{}}
	if err := Save(s); err != nil {
		return nil, preview, err
	}
	return s, preview, nil
}

// StoryRead is what Span reads from a Jira story before it starts a session from it.
type StoryRead struct {
	Key         string `json:"key"`
	Summary     string `json:"summary"`
	Status      string `json:"status"`
	Paragraphs  int    `json:"paragraphs"`
	Criteria    int    `json:"criteria"`
	CommentRead string `json:"comments"`
	URL         string `json:"url"`
}

// ReadStory reads one Jira story and says what a session would start from. It only
// reads: nothing is written to Jira.
func ReadStory(ctx context.Context, reader IssueReader, key string) (StoryRead, error) {
	story, err := reader.Issue(ctx, key)
	if err != nil {
		return StoryRead{}, err
	}
	out := StoryRead{Key: story.Key, Summary: story.Summary, Status: story.Status, URL: story.URL, CommentRead: "not read"}
	for _, block := range strings.Split(strings.ReplaceAll(story.Brief, "\r\n", "\n"), "\n\n") {
		if strings.TrimSpace(block) != "" {
			out.Paragraphs++
		}
	}
	for _, line := range strings.Split(story.Acceptance, "\n") {
		if bulletPattern.MatchString(line) {
			out.Criteria++
		}
	}
	return out, nil
}

// ExportOptions say what an export includes beyond the story itself.
type ExportOptions struct {
	// Format is FormatMarkdown or FormatJSON.
	Format string
	// Sources adds where each rule and example comes from.
	Sources bool
	// Assumptions adds the assumptions the story was sealed with.
	Assumptions bool
	// Digest adds the story's digest.
	Digest bool
}

// ExportFile is a file an export produced.
type ExportFile struct {
	Name string `json:"name"`
	MIME string `json:"mime"`
	Body []byte `json:"-"`
}

type exportRule struct {
	ID     string  `json:"id"`
	Text   string  `json:"text"`
	Source *Source `json:"source,omitempty"`
}

type exportExample struct {
	ID     string  `json:"id"`
	Rule   string  `json:"rule"`
	Text   string  `json:"text"`
	Source *Source `json:"source,omitempty"`
}

type exportAssumption struct {
	Question string `json:"question"`
	Text     string `json:"text"`
	Assumed  string `json:"assumed"`
}

type exportDoc struct {
	ID          string             `json:"id"`
	Ref         string             `json:"ref"`
	Status      string             `json:"status"`
	Repo        string             `json:"repo"`
	Digest      string             `json:"digest,omitempty"`
	SealedAt    *time.Time         `json:"sealedAt,omitempty"`
	Jira        string             `json:"jira,omitempty"`
	Rules       []exportRule       `json:"rules"`
	Examples    []exportExample    `json:"examples"`
	Assumptions []exportAssumption `json:"assumptions,omitempty"`
	Story       string             `json:"story"`
}

func maybeSource(src Source, on bool) *Source {
	if !on {
		return nil
	}
	return &src
}

// Export writes a sealed story as a file. Only a sealed story exports, so the file always
// matches the digest the session was sealed with; a draft says to seal first.
func Export(workspace, id string, opt ExportOptions) (ExportFile, error) {
	s, err := Load(workspace, id)
	if err != nil {
		return ExportFile{}, err
	}
	if s.Status != StatusSealed {
		return ExportFile{}, fail("shape_not_sealed", "seal first: only a sealed story can be exported (status is %s)", s.Status)
	}
	doc := exportDoc{
		ID: s.ID, Ref: s.Ref(), Status: s.Status, Repo: s.Repo, SealedAt: s.SealedAt, Jira: s.JiraKey(),
		Rules: []exportRule{}, Examples: []exportExample{}, Story: s.Story,
	}
	if opt.Digest {
		doc.Digest = s.Digest
	}
	for _, r := range s.Map.Rules {
		doc.Rules = append(doc.Rules, exportRule{ID: r.ID, Text: r.Text, Source: maybeSource(r.Source, opt.Sources)})
	}
	for _, e := range s.Map.Examples {
		doc.Examples = append(doc.Examples, exportExample{ID: e.ID, Rule: e.Rule, Text: e.Text, Source: maybeSource(e.Source, opt.Sources)})
	}
	if opt.Assumptions {
		for _, q := range s.Map.Questions {
			if q.Status == QuestionAssumed {
				doc.Assumptions = append(doc.Assumptions, exportAssumption{Question: q.ID, Text: q.Text, Assumed: q.Answer})
			}
		}
	}
	switch opt.Format {
	case FormatJSON, "":
		raw, err := json.MarshalIndent(doc, "", "  ")
		if err != nil {
			return ExportFile{}, fmt.Errorf("encode export: %w", err)
		}
		return ExportFile{Name: "story.json", MIME: "application/json", Body: append(raw, '\n')}, nil
	case FormatMarkdown:
		return ExportFile{Name: "story.md", MIME: "text/markdown", Body: []byte(exportMarkdown(doc, opt))}, nil
	}
	return ExportFile{}, fail("invalid", "format must be md or json, not %q", opt.Format)
}

func exportMarkdown(doc exportDoc, opt ExportOptions) string {
	var b strings.Builder
	b.WriteString(strings.TrimRight(doc.Story, "\n"))
	b.WriteString("\n")
	if opt.Sources {
		b.WriteString("\n## Sources\n\n")
		for _, r := range doc.Rules {
			fmt.Fprintf(&b, "- %s: %s", r.ID, r.Text)
			writeSource(&b, r.Source)
		}
		for _, e := range doc.Examples {
			fmt.Fprintf(&b, "- %s (%s): %s", e.ID, e.Rule, e.Text)
			writeSource(&b, e.Source)
		}
	}
	if opt.Assumptions && len(doc.Assumptions) > 0 {
		b.WriteString("\n## Assumptions sealed in\n\n")
		for _, a := range doc.Assumptions {
			fmt.Fprintf(&b, "- %s %s → %s\n", a.Question, a.Text, a.Assumed)
		}
	}
	if opt.Digest && doc.Digest != "" {
		fmt.Fprintf(&b, "\nDigest of the story above: sha256:%s\n", doc.Digest)
	}
	return b.String()
}

func writeSource(b *strings.Builder, src *Source) {
	if src != nil {
		fmt.Fprintf(b, " — %s %s", src.Kind, src.Ref)
	}
	b.WriteString("\n")
}

// Share is a read-only link to a sealed story. Anyone who can reach the server and has its
// access can open it; it never lets them change the story.
type Share struct {
	Token     string    `json:"token"`
	Session   string    `json:"session"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
	CreatedBy string    `json:"createdBy"`
	Opens     int       `json:"opens"`
	Revoked   bool      `json:"revoked"`
}

// Live says the link can still be opened at the given time.
func (s Share) Live(at time.Time) bool { return !s.Revoked && at.Before(s.ExpiresAt) }

// shareMu serializes changes to the share files of this process.
var shareMu sync.Mutex

func sharePath(workspace, id string) (string, error) {
	path, err := sessionPath(workspace, id)
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(path), shareFile), nil
}

func readShares(workspace, id string) ([]Share, error) {
	path, err := sharePath(workspace, id)
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read shares: %w", err)
	}
	var out []Share
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("shares %s: %w", id, err)
	}
	return out, nil
}

func writeShares(workspace, id string, shares []Share) error {
	path, err := sharePath(workspace, id)
	if err != nil {
		return err
	}
	raw, err := json.MarshalIndent(shares, "", "  ")
	if err != nil {
		return fmt.Errorf("encode shares: %w", err)
	}
	return writeAtomic(path, append(raw, '\n'))
}

// CreateShare makes a read-only link to a sealed story that lasts days days (7 when zero).
func CreateShare(workspace, id string, days int, now time.Time) (Share, error) {
	if days == 0 {
		days = defaultShareDays
	}
	if days < 1 || days > maxShareDays {
		return Share{}, fail("invalid", "a link lasts 1 to %d days, not %d", maxShareDays, days)
	}
	shareMu.Lock()
	defer shareMu.Unlock()
	s, err := Load(workspace, id)
	if err != nil {
		return Share{}, err
	}
	if s.Status != StatusSealed {
		return Share{}, fail("shape_not_sealed", "seal first: only a sealed story can be shared (status is %s)", s.Status)
	}
	var raw [24]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return Share{}, fmt.Errorf("share token: %w", err)
	}
	shares, err := readShares(workspace, id)
	if err != nil {
		return Share{}, err
	}
	share := Share{
		Token: base64.RawURLEncoding.EncodeToString(raw[:]), Session: id, CreatedAt: now.UTC(),
		ExpiresAt: now.UTC().Add(time.Duration(days) * 24 * time.Hour), CreatedBy: personActor,
	}
	return share, writeShares(workspace, id, append(shares, share))
}

// Shares lists the links of a session, newest first.
func Shares(workspace, id string) ([]Share, error) {
	if _, err := Load(workspace, id); err != nil {
		return nil, err
	}
	shares, err := readShares(workspace, id)
	if err != nil {
		return nil, err
	}
	out := make([]Share, 0, len(shares))
	for i := len(shares) - 1; i >= 0; i-- {
		out = append(out, shares[i])
	}
	return out, nil
}

// RevokeShare closes a link at once. The story stays sealed and unchanged.
func RevokeShare(workspace, id, token string) (Share, error) {
	shareMu.Lock()
	defer shareMu.Unlock()
	shares, err := readShares(workspace, id)
	if err != nil {
		return Share{}, err
	}
	for i := range shares {
		if shares[i].Token == token {
			shares[i].Revoked = true
			return shares[i], writeShares(workspace, id, shares)
		}
	}
	return Share{}, fail("shape_share_not_found", "no link %q for session %s", token, id)
}

// SharedStory is what a link shows: the story, the map and where each fact comes from. It
// leaves out code snippets, run logs and the text of the person's answers.
type SharedStory struct {
	Ref      string          `json:"ref"`
	Title    string          `json:"title"`
	Repo     string          `json:"repo"`
	Story    string          `json:"story"`
	Digest   string          `json:"digest"`
	SealedAt *time.Time      `json:"sealedAt,omitempty"`
	Rules    []exportRule    `json:"rules"`
	Examples []exportExample `json:"examples"`
	Link     Share           `json:"link"`
}

// OpenShare returns what a link shows and counts the opening. An expired or revoked link
// is refused.
func OpenShare(workspace, id, token string, now time.Time) (SharedStory, error) {
	shareMu.Lock()
	defer shareMu.Unlock()
	session, err := Load(workspace, id)
	if err != nil {
		return SharedStory{}, err
	}
	shares, err := readShares(workspace, id)
	if err != nil {
		return SharedStory{}, err
	}
	for i := range shares {
		if shares[i].Token != token {
			continue
		}
		if shares[i].Revoked {
			return SharedStory{}, fail("shape_share_revoked", "this link was revoked")
		}
		if !shares[i].Live(now) {
			return SharedStory{}, fail("shape_share_expired", "this link expired on %s", shares[i].ExpiresAt.Format("2 Jan"))
		}
		shares[i].Opens++
		if err := writeShares(workspace, id, shares); err != nil {
			return SharedStory{}, err
		}
		return sharedView(session, shares[i]), nil
	}
	return SharedStory{}, fail("shape_share_not_found", "no such link")
}

func sharedView(s *Session, link Share) SharedStory {
	out := SharedStory{
		Ref: s.Ref(), Title: s.Title(), Repo: s.Repo, Story: s.Story, Digest: s.Digest, SealedAt: s.SealedAt,
		Rules: []exportRule{}, Examples: []exportExample{}, Link: link,
	}
	for _, r := range s.Map.Rules {
		out.Rules = append(out.Rules, exportRule{ID: r.ID, Text: r.Text, Source: maybeSource(r.Source, true)})
	}
	for _, e := range s.Map.Examples {
		out.Examples = append(out.Examples, exportExample{ID: e.ID, Rule: e.Rule, Text: e.Text, Source: maybeSource(e.Source, true)})
	}
	return out
}
