package shape

import (
	"context"
	"os"
	"strings"
	"testing"
)

// exampleFrom is a turn that adds one example built from an answer.
func exampleFrom(answer string, questions []map[string]any, story string) map[string]any {
	body := turn(questions, story)
	m := body["map"].(map[string]any)
	m["examples"] = []any{map[string]any{
		"id": "E1", "rule": "R1", "text": "example from " + answer,
		"source": map[string]string{"kind": "answer", "ref": answer},
	}}
	return body
}

func historyIDs(v HistoryView) []string {
	ids := make([]string, 0, len(v.Events))
	for _, e := range v.Events {
		ids = append(ids, e.ID+":"+e.Kind)
	}
	return ids
}

func eventOf(t *testing.T, v HistoryView, kind string) EventView {
	t.Helper()
	for _, e := range v.Events {
		if e.Kind == kind {
			return e
		}
	}
	t.Fatalf("no %s step in %v", kind, historyIDs(v))
	return EventView{}
}

func TestHistoryNamesWhoDidEachStep(t *testing.T) {
	qs := []map[string]any{q("Q1", "who?", "admins"), q("Q2", "when?", "daily")}
	f := &fake{replies: []string{
		reply(t, turn(qs, "")),
		reply(t, exampleFrom("A1", qs, "# story")),
	}}
	e, ws := engine(t, f)
	s := start(t, e, ws)
	if _, err := e.Answer(context.Background(), ws, s.ID, []Answer{{Question: "Q1", Text: "all staff"}}, []string{"Q2"}); err != nil {
		t.Fatal(err)
	}
	v, err := History(ws, s.ID, HistoryFilter{})
	if err != nil {
		t.Fatal(err)
	}
	var kinds []string
	for i := len(v.Events) - 1; i >= 0; i-- {
		kinds = append(kinds, v.Events[i].Kind+"/"+v.Events[i].Actor)
	}
	want := "idea/you,draft/claude,ask/claude,answer/you,assume/you,draft/claude"
	if got := strings.Join(kinds, ","); got != want {
		t.Fatalf("steps\n got %s\nwant %s", got, want)
	}
	if v.Events[0].ID != "h6" || v.Counts.Everyone != 6 || v.Counts.You != 3 || v.Counts.Agents != 3 {
		t.Fatalf("ids and counts: %v %+v", historyIDs(v), v.Counts)
	}
	answer := eventOf(t, v, KindAnswer)
	if !strings.Contains(answer.Summary, "Answered Q1") || answer.Gives[0] != "A1" {
		t.Fatalf("answer step %+v", answer.Event)
	}
	you, err := History(ws, s.ID, HistoryFilter{Actor: "you", Kind: "answers"})
	if err != nil || len(you.Events) != 1 || you.Counts.Everyone != 6 {
		t.Fatalf("filtered %v %v", historyIDs(you), err)
	}
	if _, err := History(ws, s.ID, HistoryFilter{Actor: "robots"}); codeOf(err) != "invalid" {
		t.Fatalf("a bad filter is invalid, got %v", err)
	}
}

func TestUndoFollowsWhatDependsOnTheStep(t *testing.T) {
	qs := []map[string]any{q("Q1", "who?", "admins"), q("Q2", "when?", "daily")}
	f := &fake{replies: []string{
		reply(t, turn(qs, "")),
		reply(t, exampleFrom("A1", qs, "# story")),
	}}
	e, ws := engine(t, f)
	s := start(t, e, ws)
	if _, err := e.Answer(context.Background(), ws, s.ID, []Answer{{Question: "Q1", Text: "all staff"}}, []string{"Q2"}); err != nil {
		t.Fatal(err)
	}
	v, err := History(ws, s.ID, HistoryFilter{})
	if err != nil {
		t.Fatal(err)
	}
	answer := eventOf(t, v, KindAnswer)
	if !answer.Undoable || len(answer.Dependents) != 1 {
		t.Fatalf("the answer feeds the example: %+v", answer)
	}
	plan, err := PlanUndo(ws, s.ID, answer.ID)
	if err != nil {
		t.Fatal(err)
	}
	if plan.UndoOnlyThis || len(plan.Dependents) != 1 || !strings.Contains(strings.Join(plan.Changes, "|"), "E1 leaves the map") {
		t.Fatalf("plan %+v", plan)
	}
	if !strings.Contains(plan.Now, "all staff") {
		t.Fatalf("now %q", plan.Now)
	}
	if _, err := e.Undo(ws, s.ID, answer.ID, false); codeOf(err) != "shape_has_dependents" {
		t.Fatalf("a step with dependents needs the cascade, got %v", err)
	}
	if got := mustLoad(t, ws, s.ID); got.Map.Questions[0].Status != QuestionAnswered {
		t.Fatal("a refused undo changes nothing")
	}
	res, err := e.Undo(ws, s.ID, answer.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Undone) != 2 || res.Undone[0] != answer.ID {
		t.Fatalf("undone %v", res.Undone)
	}
	after := mustLoad(t, ws, s.ID)
	if after.Map.Questions[0].Status != QuestionOpen || after.Map.Questions[0].Answer != "" || len(after.Map.Examples) != 0 {
		t.Fatalf("map after undo %+v", after.Map)
	}
	if after.Map.Questions[1].Status != QuestionAssumed {
		t.Fatalf("the other assumption is untouched: %+v", after.Map.Questions[1])
	}
	again, err := History(ws, s.ID, HistoryFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if again.Events[0].Kind != KindUndo || again.Events[0].ID != res.Step.ID || len(again.Events[0].Undoes) != 2 {
		t.Fatalf("the undo is a step of its own: %+v", again.Events[0])
	}
	undone := eventOf(t, again, KindAnswer)
	if undone.UndoneBy != res.Step.ID || undone.Undoable {
		t.Fatalf("an undone step cannot be undone again: %+v", undone)
	}
	revs, err := Revisions(ws, s.ID)
	if err != nil || revs[len(revs)-1].Actor != ActorPerson || !strings.HasPrefix(revs[len(revs)-1].Label, "Undid h") {
		t.Fatalf("the undo is appended to the revision log: %+v %v", revs, err)
	}
	if _, err := e.Undo(ws, s.ID, answer.ID, true); codeOf(err) != "shape_not_undoable" {
		t.Fatalf("undoing twice is refused, got %v", err)
	}
}

func TestUndoPutsBackAReplacedAssumption(t *testing.T) {
	qs := []map[string]any{q("Q1", "limit?", "5 per hour")}
	f := &fake{replies: []string{reply(t, turn(qs, "")), reply(t, turn(qs, "")), reply(t, turn(qs, ""))}}
	e, ws := engine(t, f)
	s := start(t, e, ws)
	if _, err := e.Answer(context.Background(), ws, s.ID, nil, []string{"Q1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Answer(context.Background(), ws, s.ID, []Answer{{Question: "Q1", Text: "3 per hour"}}, nil); err != nil {
		t.Fatal(err)
	}
	v, err := History(ws, s.ID, HistoryFilter{Kind: "answers"})
	if err != nil || len(v.Events) != 1 {
		t.Fatalf("%v %v", historyIDs(v), err)
	}
	plan, err := PlanUndo(ws, s.ID, v.Events[0].ID)
	if err != nil || plan.After != "Assumed by the agent: 5 per hour" {
		t.Fatalf("plan %+v %v", plan, err)
	}
	if _, err := e.Undo(ws, s.ID, v.Events[0].ID, false); err != nil {
		t.Fatal(err)
	}
	got := mustLoad(t, ws, s.ID).Map.Questions[0]
	if got.Status != QuestionAssumed || got.Answer != "5 per hour" || got.AnswerID != "A1" {
		t.Fatalf("question %+v", got)
	}
}

func TestSealedStoryCannotBeUndone(t *testing.T) {
	f := &fake{replies: []string{reply(t, turn([]map[string]any{q("Q1", "who?", "admins")}, "")), reply(t, turn([]map[string]any{q("Q1", "who?", "admins")}, "# story"))}}
	e, ws := engine(t, f)
	s := start(t, e, ws)
	if _, err := e.Answer(context.Background(), ws, s.ID, nil, []string{"Q1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Seal(ws, s.ID); err != nil {
		t.Fatal(err)
	}
	v, err := History(ws, s.ID, HistoryFilter{})
	if err != nil || !v.Sealed || v.Events[0].Kind != KindSeal {
		t.Fatalf("%v %v", historyIDs(v), err)
	}
	for _, step := range v.Events {
		if step.Undoable || step.Why == "" {
			t.Fatalf("%s must be locked: %+v", step.ID, step)
		}
	}
	if _, err := e.Undo(ws, s.ID, "h2", true); codeOf(err) != "shape_sealed" {
		t.Fatalf("got %v", err)
	}
	if _, err := PlanUndo(ws, s.ID, "h99"); codeOf(err) != "shape_step_not_found" {
		t.Fatalf("got %v", err)
	}
}

func TestUndoTakesAnAcceptedNoteOutOfTheContext(t *testing.T) {
	body := turn(nil, "")
	body["proposals"] = []any{
		map[string]string{"id": "P1", "section": "Personas", "text": "- Ana, admin"},
		map[string]string{"id": "P2", "section": "Glossary", "text": "- Export: CSV file"},
	}
	f := &fake{replies: []string{reply(t, body)}}
	e, ws := engine(t, f)
	writeContext(t, ws, "# Product\n\n## Glossary\n\n- Order: a purchase\n")
	s := start(t, e, ws)
	if _, err := e.AcceptProposals(ws, s.ID, []string{"P1", "P2"}); err != nil {
		t.Fatal(err)
	}
	v, err := History(ws, s.ID, HistoryFilter{Kind: "context"})
	if err != nil || len(v.Events) != 2 {
		t.Fatalf("%v %v", historyIDs(v), err)
	}
	for _, step := range v.Events {
		if !step.Undoable {
			t.Fatalf("an accepted note can be taken out: %+v", step)
		}
		if _, err := e.Undo(ws, s.ID, step.ID, false); err != nil {
			t.Fatal(err)
		}
	}
	raw := mustRead(t, ContextPath(ws))
	if strings.Contains(raw, "Ana") || strings.Contains(raw, "Personas") || strings.Contains(raw, "CSV") {
		t.Fatalf("the context keeps only what was there:\n%s", raw)
	}
	if !strings.Contains(raw, "- Order: a purchase") {
		t.Fatalf("what was there stays:\n%s", raw)
	}
	for _, p := range mustLoad(t, ws, s.ID).Proposals {
		if p.Status != ProposalProposed {
			t.Fatalf("%+v", p)
		}
	}
}

func TestUndoRefusesWhenTheContextNoteWasEdited(t *testing.T) {
	body := turn(nil, "")
	body["proposals"] = []any{map[string]string{"id": "P1", "section": "Personas", "text": "- Ana, admin"}}
	f := &fake{replies: []string{reply(t, body)}}
	e, ws := engine(t, f)
	s := start(t, e, ws)
	if _, err := e.AcceptProposals(ws, s.ID, []string{"P1"}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ContextPath(ws), []byte("# Product\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	v, err := History(ws, s.ID, HistoryFilter{Kind: "context"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.Undo(ws, s.ID, v.Events[0].ID, false); codeOf(err) != "shape_context_changed" {
		t.Fatalf("got %v", err)
	}
	if mustLoad(t, ws, s.ID).Proposals[0].Status != ProposalAccepted {
		t.Fatal("a refused undo changes nothing")
	}
}

func TestKeepWholeCanBeUndone(t *testing.T) {
	body := turn(nil, "# story")
	body["split"] = []any{map[string]string{"story": "A", "leftOut": "B"}, map[string]string{"story": "B", "leftOut": "A"}}
	f := &fake{replies: []string{reply(t, body), reply(t, turn(nil, "# story"))}}
	e, ws := engine(t, f)
	s := start(t, e, ws)
	if _, err := e.KeepWhole(context.Background(), ws, s.ID); err != nil {
		t.Fatal(err)
	}
	v, err := History(ws, s.ID, HistoryFilter{Kind: "split"})
	if err != nil || len(v.Events) != 1 || !v.Events[0].Undoable {
		t.Fatalf("%v %v", historyIDs(v), err)
	}
	if _, err := e.Undo(ws, s.ID, v.Events[0].ID, false); err != nil {
		t.Fatal(err)
	}
	if mustLoad(t, ws, s.ID).SplitKept {
		t.Fatal("the split can be proposed again")
	}
}

func TestAcceptedSplitIsAStepThatSaysWhyItStays(t *testing.T) {
	f := &fake{replies: []string{reply(t, splitTurn())}}
	e, ws := engine(t, f)
	s := start(t, e, ws)
	if _, err := e.AcceptSplit(ws, s.ID); err != nil {
		t.Fatal(err)
	}
	v, err := History(ws, s.ID, HistoryFilter{Kind: "split"})
	if err != nil || len(v.Events) != 1 {
		t.Fatalf("%v %v", historyIDs(v), err)
	}
	step := v.Events[0]
	if step.Kind != KindSplitAccept || step.Actor != personActor || step.Undoable || !strings.Contains(step.Why, "own sessions") {
		t.Fatalf("split step %+v", step)
	}
	if _, err := e.AcceptSplit(ws, s.ID); err != nil {
		t.Fatal(err)
	}
	again, err := History(ws, s.ID, HistoryFilter{Kind: "split"})
	if err != nil || len(again.Events) != 1 {
		t.Fatalf("accepting twice adds no step: %v %v", historyIDs(again), err)
	}
}

func TestJiraCreateIsAStepThatSaysWhyItStays(t *testing.T) {
	f := &fake{replies: []string{reply(t, turn(nil, "# story"))}}
	e, ws := engine(t, f)
	s := start(t, e, ws)
	if _, err := e.Seal(ws, s.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.SetJira(ws, s.ID, JiraRef{Key: "A-1", URL: "http://j/A-1", Status: "Backlog"}); err != nil {
		t.Fatal(err)
	}
	v, err := History(ws, s.ID, HistoryFilter{})
	if err != nil {
		t.Fatal(err)
	}
	step := eventOf(t, v, KindJira)
	if step.Actor != personActor || step.Undoable || !strings.Contains(step.Summary, "A-1") || !strings.Contains(step.Why, "Jira") {
		t.Fatalf("jira step %+v", step)
	}
}
