package shape

import (
	"errors"
	"strings"
	"testing"
)

// saved changes the session on disk the way an operation does: mutate, then Save.
func saved(t *testing.T, ws, id string, change func(s *Session)) *Session {
	t.Helper()
	s := mustLoad(t, ws, id)
	change(s)
	if err := Save(s); err != nil {
		t.Fatal(err)
	}
	return s
}

func labels(t *testing.T, ws, id string) []string {
	t.Helper()
	revs, err := Revisions(ws, id)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, r := range revs {
		out = append(out, r.ID+":"+r.Label+":"+r.Actor)
	}
	return out
}

func TestEverySavedChangeMakesARevisionAndBookkeepingDoesNot(t *testing.T) {
	ws := gitRepo(t)
	s, err := Create(ws, StartInput{Idea: "Retry a failed delivery", Agent: "claude-code"})
	if err != nil {
		t.Fatal(err)
	}
	saved(t, ws, s.ID, func(s *Session) {
		s.Map = Map{
			Rules:     []Rule{{ID: "R3", Text: "A manual retry is one new attempt.", Source: Source{Kind: SourcePrompt, Ref: "idea"}}},
			Examples:  []Example{},
			Questions: []Question{{ID: "Q1", Text: "How many per hour?", Assume: "5", Status: QuestionOpen}},
		}
	})
	saved(t, ws, s.ID, func(s *Session) {
		s.Map.Questions[0].Status = QuestionAnswered
		s.Map.Questions[0].Answer = "3 per hour"
		s.Map.Questions[0].AnswerID = "A1"
	})
	// Archiving and turn bookkeeping change the file but not the story.
	saved(t, ws, s.ID, func(s *Session) { s.Archived = true; s.Turns = append(s.Turns, Turn{N: 1}) })

	want := []string{"r1:Idea saved:you", "r2:Map drafted:claude-code", "r3:Answered Q1 (A1):you"}
	got := labels(t, ws, s.ID)
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("revisions %v, want %v", got, want)
	}
	infos, err := Revisions(ws, s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !infos[2].Current || infos[1].Current || infos[0].Digest == infos[2].Digest {
		t.Fatalf("current/digest wrong: %+v", infos)
	}
}

func TestAssumptionAcceptContextAndSealAreNamed(t *testing.T) {
	ws := gitRepo(t)
	s, err := Create(ws, StartInput{Idea: "x"})
	if err != nil {
		t.Fatal(err)
	}
	saved(t, ws, s.ID, func(s *Session) {
		s.Map.Questions = []Question{{ID: "Q2", Status: QuestionOpen}}
		s.Proposals = []Proposal{{ID: "P1", Status: ProposalProposed}}
	})
	saved(t, ws, s.ID, func(s *Session) {
		s.Map.Questions[0].Status = QuestionAssumed
		s.Map.Questions[0].AnswerID = "A1"
	})
	saved(t, ws, s.ID, func(s *Session) { s.Proposals[0].Status = ProposalAccepted })
	saved(t, ws, s.ID, func(s *Session) { s.Status = StatusSealed })
	got := strings.Join(labels(t, ws, s.ID), "|")
	for _, want := range []string{"Used the assumption for Q2 (A1)", "Context note accepted", "Sealed"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in %s", want, got)
		}
	}
}

func TestDiffByKindOfItemAndStoryLines(t *testing.T) {
	ws := gitRepo(t)
	s, err := Create(ws, StartInput{Idea: "x"})
	if err != nil {
		t.Fatal(err)
	}
	saved(t, ws, s.ID, func(s *Session) {
		s.Map = Map{
			Rules:     []Rule{{ID: "R3", Text: "A manual retry counts as one new attempt and sends the same payload."}},
			Examples:  []Example{{ID: "E5", Rule: "R3", Text: "Retry sends the original payload."}},
			Questions: []Question{{ID: "Q2", Text: "Retries per hour?", Assume: "5", Status: QuestionOpen}},
		}
		s.Story = "# Retry\n### R3 one attempt\n- E5 same payload\n- Limited to 5 per hour (Q2, assumed)\n"
	})
	saved(t, ws, s.ID, func(s *Session) {
		s.Map.Rules[0].Text = "A manual retry counts as one new attempt, up to 3 per hour, and sends the same payload."
		s.Map.Rules[0].Source = Source{Kind: SourceAnswer, Ref: "A2"}
		s.Map.Examples = append(s.Map.Examples, Example{ID: "E6", Rule: "R3", Text: "A fourth retry is refused.", Source: Source{Kind: SourceAnswer, Ref: "A2"}})
		s.Map.Questions[0].Status = QuestionAnswered
		s.Map.Questions[0].Answer = "3 per hour"
		s.Map.Questions[0].AnswerID = "A2"
		s.Story = "# Retry\n### R3 one attempt, up to 3 per hour\n- E5 same payload\n- E6 a fourth is refused\n"
	})
	d, err := Diff(ws, s.ID, "r2", "3")
	if err != nil {
		t.Fatal(err)
	}
	if d.Rules != (Tally{Changed: 1}) || d.Examples != (Tally{Added: 1}) || d.Questions != (Tally{Changed: 1}) {
		t.Fatalf("tallies %+v %+v %+v", d.Rules, d.Examples, d.Questions)
	}
	if d.StoryAdded != 2 || d.StoryRemoved != 2 {
		t.Fatalf("story +%d -%d", d.StoryAdded, d.StoryRemoved)
	}
	byID := map[string]ItemChange{}
	for _, it := range d.Items {
		byID[it.ID] = it
	}
	if byID["R3"].Source == nil || byID["R3"].Source.Ref != "A2" || !strings.Contains(byID["R3"].After, "up to 3 per hour") {
		t.Fatalf("R3 %+v", byID["R3"])
	}
	if byID["E6"].Change != ChangeAdded || byID["E6"].Before != "" {
		t.Fatalf("E6 %+v", byID["E6"])
	}
	if !strings.HasPrefix(byID["Q2"].After, "Answered by you: 3 per hour (A2)") || !strings.HasPrefix(byID["Q2"].Before, "Open:") {
		t.Fatalf("Q2 %+v", byID["Q2"])
	}
	var del, add int
	for _, l := range d.Story {
		switch l.Kind {
		case LineDel:
			del++
			if l.New != 0 || l.Old == 0 {
				t.Fatalf("del numbers %+v", l)
			}
		case LineAdd:
			add++
			if l.Old != 0 || l.New == 0 {
				t.Fatalf("add numbers %+v", l)
			}
		}
	}
	if del != 2 || add != 2 {
		t.Fatalf("story lines del %d add %d: %+v", del, add, d.Story)
	}
	same, err := Diff(ws, s.ID, "r3", "r3")
	if err != nil || len(same.Items) != 0 || len(same.Story) != 0 {
		t.Fatalf("same revision: %+v %v", same, err)
	}
}

func TestDiffRefusesAnUnknownRevision(t *testing.T) {
	ws := gitRepo(t)
	s, err := Create(ws, StartInput{Idea: "x"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = Diff(ws, s.ID, "r1", "r9")
	var coded *Error
	if !errors.As(err, &coded) || coded.Code != "shape_revision_not_found" {
		t.Fatalf("err %v", err)
	}
}

func TestRestoreAddsARevisionAndKeepsTheHistory(t *testing.T) {
	ws := gitRepo(t)
	s, err := Create(ws, StartInput{Idea: "x"})
	if err != nil {
		t.Fatal(err)
	}
	saved(t, ws, s.ID, func(s *Session) {
		s.Map.Questions = []Question{{ID: "Q2", Text: "per hour?", Assume: "5", Status: QuestionAssumed, Answer: "5", AnswerID: "A1"}}
		s.Status = StatusReady
		s.Story = "five per hour\n"
	})
	saved(t, ws, s.ID, func(s *Session) {
		s.Map.Questions[0].Status = QuestionAnswered
		s.Map.Questions[0].Answer = "3"
		s.Story = "three per hour\n"
	})
	restored, err := Restore(ws, s.ID, "r2")
	if err != nil {
		t.Fatal(err)
	}
	if restored.Story != "five per hour\n" || restored.Map.Questions[0].Answer != "5" || restored.Status != StatusReady {
		t.Fatalf("restored %+v", restored)
	}
	got := labels(t, ws, s.ID)
	if len(got) != 4 || got[3] != "r4:Restored r2:you" {
		t.Fatalf("revisions %v", got)
	}
	// Restoring the content that is already current adds nothing and leaves no stale hint.
	if _, err := Restore(ws, s.ID, "r4"); err != nil || len(labels(t, ws, s.ID)) != 4 {
		t.Fatalf("restore current: %v %v", err, labels(t, ws, s.ID))
	}
	saved(t, ws, s.ID, func(s *Session) { s.Story = "again\n" })
	if got := labels(t, ws, s.ID); got[4] != "r5:Story drafted:"+ActorAgent {
		t.Fatalf("stale hint leaked: %v", got)
	}
}

func TestRestoreRefusesASealedSession(t *testing.T) {
	ws := gitRepo(t)
	s, err := Create(ws, StartInput{Idea: "x"})
	if err != nil {
		t.Fatal(err)
	}
	saved(t, ws, s.ID, func(s *Session) { s.Status = StatusSealed; s.Digest = "d" })
	_, err = Restore(ws, s.ID, "r1")
	var coded *Error
	if !errors.As(err, &coded) || coded.Code != "shape_sealed" {
		t.Fatalf("err %v", err)
	}
}

func TestForkRevisionStartsANewSessionFromAnEarlierRevisionEvenWhenSealed(t *testing.T) {
	ws := gitRepo(t)
	s, err := Create(ws, StartInput{Idea: "x"})
	if err != nil {
		t.Fatal(err)
	}
	saved(t, ws, s.ID, func(s *Session) {
		s.Map.Questions = []Question{{ID: "Q1", Text: "per hour?", Assume: "5", Status: QuestionAssumed, Answer: "5", AnswerID: "A1"}}
		s.Story = "five per hour\n"
	})
	saved(t, ws, s.ID, func(s *Session) { s.Story = "sealed text\n"; s.Status = StatusSealed; s.Digest = "d" })
	child, err := ForkRevision(ws, s.ID, "r2")
	if err != nil {
		t.Fatal(err)
	}
	if child.Parent != s.ID || child.Story != "five per hour\n" || child.Map.Questions[0].Status != QuestionOpen || child.Status != StatusInterviewing {
		t.Fatalf("child %+v", child)
	}
	if kept := mustLoad(t, ws, s.ID); kept.Status != StatusSealed || kept.Story != "sealed text\n" {
		t.Fatalf("the sealed session changed: %+v", kept)
	}
}
