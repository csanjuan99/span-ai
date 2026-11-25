package shape

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func withProposals(body map[string]any, ps ...map[string]string) map[string]any {
	list := []any{}
	for _, p := range ps {
		list = append(list, p)
	}
	body["proposals"] = list
	return body
}

func TestAnswerIDsAreStableAndAnAssumptionCanBeReplaced(t *testing.T) {
	f := &fake{replies: []string{
		reply(t, turn([]map[string]any{q("Q1", "who", "anyone"), q("Q2", "how many", "3")}, "")),
		reply(t, turn(nil, "story")),
		reply(t, turn(nil, "story v2")),
	}}
	e, ws := engine(t, f)
	s := start(t, e, ws)
	s, err := e.Answer(context.Background(), ws, s.ID, []Answer{{Question: "Q2", Text: "5 per hour"}}, []string{"Q1"})
	if err != nil {
		t.Fatal(err)
	}
	if got := s.Map.Questions[1].AnswerID + "," + s.Map.Questions[0].AnswerID; got != "A1,A2" {
		t.Fatalf("answers are numbered in the order they are applied: %s", got)
	}
	s, err = e.Answer(context.Background(), ws, s.ID, []Answer{{Question: "Q1", Text: "admins"}}, nil)
	if err != nil {
		t.Fatalf("an assumption can be answered instead: %v", err)
	}
	q1 := s.Map.Questions[0]
	if q1.Status != QuestionAnswered || q1.Answer != "admins" || q1.AnswerID != "A2" {
		t.Fatalf("%+v", q1)
	}
	if _, err := e.Answer(context.Background(), ws, s.ID, []Answer{{Question: "Q2", Text: "again"}}, nil); err == nil {
		t.Fatal("an answered question is final")
	}
}

func TestSettleProposalsSkipsAndRestoresButNeverUndoesAWrite(t *testing.T) {
	body := withProposals(turn(nil, "story"),
		map[string]string{"id": "P1", "section": "Access", "text": "- Any user"},
		map[string]string{"id": "P2", "section": "Webhooks", "text": "- 30 days"})
	e, ws := engine(t, &fake{replies: []string{reply(t, body)}})
	s := start(t, e, ws)
	s, err := e.SettleProposals(ws, s.ID, []string{"P1"}, []string{"P2"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if s.Proposals[0].Status != ProposalAccepted || s.Proposals[1].Status != ProposalSkipped {
		t.Fatalf("%+v", s.Proposals)
	}
	if raw := mustRead(t, ContextPath(ws)); !strings.Contains(raw, "- Any user") || strings.Contains(raw, "30 days") {
		t.Fatalf("only the accepted note is written:\n%s", raw)
	}
	s, err = e.SettleProposals(ws, s.ID, nil, nil, []string{"P2"})
	if err != nil || s.Proposals[1].Status != ProposalProposed {
		t.Fatalf("restore: %v %+v", err, s.Proposals)
	}
	if _, err := e.SettleProposals(ws, s.ID, nil, []string{"P1"}, nil); codeOf(err) != "shape_context_written" {
		t.Fatalf("a written note cannot be skipped: %v", err)
	}
	if _, err := e.SettleProposals(ws, s.ID, []string{"P2", "P9"}, nil, nil); err == nil {
		t.Fatal("an unknown id changes nothing")
	}
	if got := mustLoad(t, ws, s.ID).Proposals[1].Status; got != ProposalProposed {
		t.Fatalf("a refused request wrote %s", got)
	}
}

func TestContextPreviewShowsExactlyWhatAcceptingWrites(t *testing.T) {
	body := withProposals(turn(nil, "story"), map[string]string{"id": "P1", "section": "Webhooks", "text": "- Retry 3 per hour"})
	e, ws := engine(t, &fake{replies: []string{reply(t, body)}})
	s := start(t, e, ws)
	writeContext(t, ws, "# Product\n\n## Webhooks\n\n- Signed payloads\n\n## Billing\n\n- Monthly\n")
	diff, err := ContextPreview(ws, s.ID, []string{"P1"})
	if err != nil {
		t.Fatal(err)
	}
	var added []string
	for _, l := range diff.Lines {
		if l.Kind == DiffAdd {
			added = append(added, l.Text)
		}
	}
	if len(added) != 1 || added[0] != "- Retry 3 per hour" {
		t.Fatalf("%+v", diff.Lines)
	}
	if before := mustRead(t, ContextPath(ws)); strings.Contains(before, "Retry") {
		t.Fatal("a preview writes nothing")
	}
	if _, err := e.SettleProposals(ws, s.ID, []string{"P1"}, nil, nil); err != nil {
		t.Fatal(err)
	}
	want := mustRead(t, ContextPath(ws))
	for _, l := range diff.Lines {
		if l.Kind == DiffAdd && !strings.Contains(want, l.Text) {
			t.Fatalf("preview line %q is not in the written file", l.Text)
		}
	}
	empty, err := ContextPreview(ws, s.ID, nil)
	if err != nil || len(empty.Lines) != 0 {
		t.Fatalf("nothing selected, nothing shown: %v %+v", err, empty)
	}
}

func splitTurn() map[string]any {
	body := turn(nil, "")
	body["split"] = []any{
		map[string]string{"story": "Show why a delivery failed", "leftOut": "The Retry button"},
		map[string]string{"story": "Retry a failed delivery", "leftOut": "Failure reasons"},
	}
	return body
}

func TestAcceptSplitCreatesLinkedChildrenThatShareTheMap(t *testing.T) {
	e, ws := engine(t, &fake{replies: []string{reply(t, splitTurn())}})
	s := start(t, e, ws)
	if s.Status != StatusSplit {
		t.Fatal(s.Status)
	}
	s, err := e.AcceptSplit(ws, s.ID)
	if err != nil || len(s.Children) != 2 {
		t.Fatalf("%v %+v", err, s.Children)
	}
	again, err := e.AcceptSplit(ws, s.ID)
	if err != nil || len(again.Children) != 2 || again.Children[0] != s.Children[0] {
		t.Fatalf("accepting twice creates nothing new: %v %+v", err, again.Children)
	}
	child := mustLoad(t, ws, s.Children[1])
	if child.Idea != "Retry a failed delivery" || child.Parent != s.ID || child.Status != StatusInterviewing || len(child.Map.Rules) != 1 || child.Story != "" {
		t.Fatalf("%+v", child)
	}
	if mustLoad(t, ws, s.ID).Status != StatusSplit {
		t.Fatal("the parent stays a split and cannot be sealed")
	}
}

func TestKeepWholeRunsTheTurnAgainWithoutProposingTheSplit(t *testing.T) {
	again := splitTurn()
	again["story"] = "# one story"
	f := &fake{replies: []string{reply(t, splitTurn()), reply(t, again)}}
	e, ws := engine(t, f)
	s := start(t, e, ws)
	s, err := e.KeepWhole(context.Background(), ws, s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if s.Status != StatusReady || len(s.Split) != 0 || !s.SplitKept || s.Story != "# one story" {
		t.Fatalf("%+v", s)
	}
	if !strings.Contains(f.prompts[1], "keep this as one story") {
		t.Fatalf("the agent is told:\n%s", f.prompts[1])
	}
	if _, err := e.AcceptSplit(ws, s.ID); codeOf(err) != "shape_no_split" {
		t.Fatalf("no split left to accept: %v", err)
	}
}

func TestForkReopensAssumptionsAndKeepsTheSealedStory(t *testing.T) {
	f := &fake{replies: []string{
		reply(t, turn([]map[string]any{q("Q1", "who", "anyone")}, "")),
		reply(t, turn(nil, "# story")),
	}}
	e, ws := engine(t, f)
	s := start(t, e, ws)
	s, err := e.Answer(context.Background(), ws, s.ID, nil, []string{"Q1"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.Fork(ws, s.ID); codeOf(err) != "shape_not_sealed" {
		t.Fatalf("only a sealed story is forked: %v", err)
	}
	sealed, err := e.Seal(ws, s.ID)
	if err != nil {
		t.Fatal(err)
	}
	child, err := e.Fork(ws, sealed.ID)
	if err != nil {
		t.Fatal(err)
	}
	if child.ID == sealed.ID || child.Parent != sealed.ID || child.Status != StatusInterviewing || child.Map.Questions[0].Status != QuestionOpen || child.Map.Questions[0].Assume != "anyone" || child.Story != "# story" || child.Digest != "" {
		t.Fatalf("%+v", child)
	}
	if mustLoad(t, ws, sealed.ID).Status != StatusSealed {
		t.Fatal("the sealed session stays")
	}
}

func TestSourceSnippetReadsTheLinesAroundARefAndStaysInsideTheRepo(t *testing.T) {
	e, ws := engine(t, &fake{replies: []string{reply(t, turn(nil, "story"))}})
	s := start(t, e, ws)
	lines := []string{}
	for i := 1; i <= 20; i++ {
		lines = append(lines, "line "+string(rune('a'+i-1)))
	}
	if err := os.MkdirAll(filepath.Join(ws, "internal"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ws, "internal", "a.go"), []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	snip, err := SourceSnippet(ws, s.ID, "internal/a.go:10")
	if err != nil {
		t.Fatal(err)
	}
	if snip.Path != "internal/a.go" || snip.Line != 10 || snip.Lines[0].N != 7 || snip.Lines[len(snip.Lines)-1].N != 13 {
		t.Fatalf("%+v", snip)
	}
	var hit []string
	for _, l := range snip.Lines {
		if l.Hit {
			hit = append(hit, l.Text)
		}
	}
	if len(hit) != 1 || hit[0] != "line j" {
		t.Fatalf("exactly the cited line is marked: %v", hit)
	}
	for _, bad := range []string{"../outside.go:1", "/etc/passwd:1", "internal/missing.go:1", "internal:1", ""} {
		if _, err := SourceSnippet(ws, s.ID, bad); err == nil {
			t.Fatalf("%q must be refused", bad)
		}
	}
	if snip, err := SourceSnippet(ws, s.ID, "internal/a.go:999"); err != nil || len(snip.Lines) != 0 {
		t.Fatalf("a line past the end shows nothing: %v %+v", err, snip)
	}
}

func TestSealStampsTheTime(t *testing.T) {
	e, ws := engine(t, &fake{replies: []string{reply(t, turn(nil, "story"))}})
	s := start(t, e, ws)
	sealed, err := e.Seal(ws, s.ID)
	if err != nil || sealed.SealedAt == nil || sealed.SealedAt.IsZero() {
		t.Fatalf("%v %+v", err, sealed.SealedAt)
	}
}
