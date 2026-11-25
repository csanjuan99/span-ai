package shape

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"span/internal/jira"
)

const storyFile = `# Show tax per line on the invoice PDF

## Tax is shown on each line
- A line of 100 with 20% tax shows 20
- A zero rated line shows 0

## Rounding
Each line rounds half up.
- 10.005 shows 10.01

## Notes
Finance asked for this in October.
`

func TestParseMarkdownReadsHeadingsAsRulesAndBulletsAsExamples(t *testing.T) {
	p, err := ParseMarkdown("story.md", storyFile)
	if err != nil {
		t.Fatal(err)
	}
	if p.Title != "Show tax per line on the invoice PDF" || len(p.Rules) != 2 || len(p.Examples) != 3 || len(p.Notes) != 1 {
		t.Fatalf("%+v", p)
	}
	if p.Rules[1].Text != "Rounding. Each line rounds half up." || p.Examples[2].Rule != "R2" {
		t.Fatalf("rules %+v examples %+v", p.Rules, p.Examples)
	}
	if got := p.Examples[0].Source; got.Kind != SourceImport || got.Ref != "story.md:4" {
		t.Fatalf("an imported item names its line: %+v", got)
	}
	if p.Notes[0].Heading != "Notes" || !strings.Contains(p.Notes[0].Text, "October") {
		t.Fatalf("notes %+v", p.Notes)
	}
	want := []string{
		"2 headings read as rules (R1, R2)",
		"3 bullets read as examples, each with its line",
		"section “Notes” (line 11) could not be read as a rule; it is kept as context, not as a rule",
	}
	if strings.Join(p.Reads, "|") != strings.Join(want, "|") {
		t.Fatalf("reads %q", p.Reads)
	}
}

func TestParseMarkdownRefusesWhatItCannotRead(t *testing.T) {
	for name, text := range map[string]string{
		"empty": "  \n", "binary": "# t\n## r\x00", "no rules": "# Title only\n- a bullet\n",
		"too big": "# t\n## r\n" + strings.Repeat("x", maxImportBytes),
	} {
		if _, err := ParseMarkdown("a.md", text); codeOf(err) != "invalid" {
			t.Errorf("%s: got %v", name, err)
		}
	}
	if p, err := ParseMarkdown("../../etc/passwd", "# t\n## r\n"); err != nil || p.Name != "passwd" {
		t.Fatalf("only the base name is kept: %+v %v", p.Name, err)
	}
}

func TestImportMarkdownStartsASessionWithTheMapFilled(t *testing.T) {
	ws := gitRepo(t)
	s, p, err := ImportMarkdown(ws, ImportInput{Name: "story.md", Markdown: storyFile, Agent: "claude"})
	if err != nil {
		t.Fatal(err)
	}
	got := mustLoad(t, ws, s.ID)
	if len(got.Map.Rules) != 2 || len(got.Map.Examples) != 3 || got.Status != StatusInterviewing || got.Map.Rules[0].Source.Kind != SourceImport {
		t.Fatalf("%+v", got.Map)
	}
	if !strings.Contains(got.Idea, "Show tax per line") || len(p.Rules) != 2 {
		t.Fatalf("idea %q", got.Idea)
	}
}

type issues map[string]jira.Story

func (i issues) Issue(_ context.Context, key string) (jira.Story, error) {
	story, ok := i[key]
	if !ok {
		return jira.Story{}, fail("not_found", "no issue %s", key)
	}
	return story, nil
}

func TestReadStoryCountsWhatItReadAndSaysCommentsAreNotRead(t *testing.T) {
	reader := issues{"BILL-2031": {Key: "BILL-2031", Summary: "Show tax per line", Status: "Done",
		Brief: "First paragraph.\n\nSecond paragraph.", Acceptance: "- one\n- two\n* three\nplain"}}
	got, err := ReadStory(context.Background(), reader, "BILL-2031")
	if err != nil {
		t.Fatal(err)
	}
	if got.Paragraphs != 2 || got.Criteria != 3 || got.CommentRead != "not read" || got.Status != "Done" {
		t.Fatalf("%+v", got)
	}
	if _, err := ReadStory(context.Background(), reader, "NOPE-1"); err == nil {
		t.Fatal("an unknown key is an error")
	}
}

func sealedStory(t *testing.T) (string, *Session) {
	t.Helper()
	qs := []map[string]any{q("Q1", "limit?", "5 per hour")}
	body := exampleFrom("A1", qs, "# Retry a delivery\n\nThe customer retries.")
	f := &fake{replies: []string{reply(t, turn(qs, "")), reply(t, body)}}
	e, ws := engine(t, f)
	s := start(t, e, ws)
	if _, err := e.Answer(context.Background(), ws, s.ID, nil, []string{"Q1"}); err != nil {
		t.Fatal(err)
	}
	sealed, err := e.Seal(ws, s.ID)
	if err != nil {
		t.Fatal(err)
	}
	return ws, sealed
}

func TestExportNeedsASealedStoryAndHonoursItsOptions(t *testing.T) {
	ws, s := sealedStory(t)
	file, err := Export(ws, s.ID, ExportOptions{Format: FormatJSON, Sources: true, Assumptions: true, Digest: true})
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(file.Body, &doc); err != nil {
		t.Fatal(err)
	}
	if file.Name != "story.json" || doc["digest"] != s.Digest || doc["status"] != "sealed" || doc["assumptions"] == nil {
		t.Fatalf("%s %v", file.Name, doc)
	}
	if rules := doc["rules"].([]any); rules[0].(map[string]any)["source"] == nil {
		t.Fatalf("sources are included: %v", rules)
	}
	bare, err := Export(ws, s.ID, ExportOptions{Format: FormatJSON})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(bare.Body), "digest") || strings.Contains(string(bare.Body), "assumptions") || strings.Contains(string(bare.Body), `"source"`) {
		t.Fatalf("what was not asked for is left out:\n%s", bare.Body)
	}
	md, err := Export(ws, s.ID, ExportOptions{Format: FormatMarkdown, Sources: true, Assumptions: true, Digest: true})
	if err != nil {
		t.Fatal(err)
	}
	text := string(md.Body)
	if !strings.HasPrefix(text, "# Retry a delivery") || !strings.Contains(text, "## Sources") || !strings.Contains(text, "limit? → 5 per hour") || !strings.Contains(text, "sha256:"+s.Digest) {
		t.Fatalf("markdown:\n%s", text)
	}
	if _, err := Export(ws, s.ID, ExportOptions{Format: "pdf"}); codeOf(err) != "invalid" {
		t.Fatalf("got %v", err)
	}
	draft, err := ImportedDraft(t, ws)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Export(ws, draft, ExportOptions{Format: FormatJSON}); codeOf(err) != "shape_not_sealed" {
		t.Fatalf("a draft says to seal first, got %v", err)
	}
}

// ImportedDraft makes a session that is not sealed.
func ImportedDraft(t *testing.T, ws string) (string, error) {
	t.Helper()
	s, _, err := ImportMarkdown(ws, ImportInput{Name: "a.md", Markdown: storyFile})
	if err != nil {
		return "", err
	}
	return s.ID, nil
}

func TestShareIsReadOnlyExpiresAndRevokes(t *testing.T) {
	ws, s := sealedStory(t)
	now := time.Date(2026, 11, 25, 9, 52, 0, 0, time.UTC)
	link, err := CreateShare(ws, s.ID, 0, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(link.Token) < 30 || !link.ExpiresAt.Equal(now.Add(7*24*time.Hour)) || link.Opens != 0 {
		t.Fatalf("%+v", link)
	}
	shown, err := OpenShare(ws, s.ID, link.Token, now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if shown.Story == "" || shown.Digest != s.Digest || shown.Link.Opens != 1 || len(shown.Rules) == 0 {
		t.Fatalf("%+v", shown)
	}
	body, err := json.Marshal(shown)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "5 per hour") || strings.Contains(string(body), "limit?") {
		t.Fatalf("the answers and questions are not shared:\n%s", body)
	}
	if _, err := OpenShare(ws, s.ID, link.Token, now.Add(8*24*time.Hour)); codeOf(err) != "shape_share_expired" {
		t.Fatalf("got %v", err)
	}
	if _, err := RevokeShare(ws, s.ID, link.Token); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenShare(ws, s.ID, link.Token, now.Add(time.Hour)); codeOf(err) != "shape_share_revoked" {
		t.Fatalf("got %v", err)
	}
	if _, err := OpenShare(ws, s.ID, "nope", now); codeOf(err) != "shape_share_not_found" {
		t.Fatalf("got %v", err)
	}
	list, err := Shares(ws, s.ID)
	if err != nil || len(list) != 1 || !list[0].Revoked || list[0].Opens != 1 {
		t.Fatalf("%+v %v", list, err)
	}
	if _, err := CreateShare(ws, s.ID, 400, now); codeOf(err) != "invalid" {
		t.Fatalf("got %v", err)
	}
	draft, err := ImportedDraft(t, ws)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CreateShare(ws, draft, 1, now); codeOf(err) != "shape_not_sealed" {
		t.Fatalf("got %v", err)
	}
}
