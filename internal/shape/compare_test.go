package shape

import (
	"errors"
	"strings"
	"testing"
)

func cmpRule(id, text string) Rule {
	return Rule{ID: id, Text: text, Source: Source{Kind: SourcePrompt, Ref: "idea"}}
}

func answered(id, text, answer, answerID string) Question {
	return Question{ID: id, Text: text, Status: QuestionAnswered, Answer: answer, AnswerID: answerID}
}

// three builds the sessions of the design: 0142 (interviewing), 0138 (ready), 0129 (sealed).
func three(t *testing.T) (ws string, a, b, c *Session) {
	t.Helper()
	ws = gitRepo(t)
	mk := func(idea string, change func(s *Session)) *Session {
		s, err := Create(ws, StartInput{Idea: idea})
		if err != nil {
			t.Fatal(err)
		}
		return saved(t, ws, s.ID, change)
	}
	a = mk("retry a failed delivery", func(s *Session) {
		s.Map = Map{
			Rules: []Rule{
				cmpRule("R1", "A customer sees the status and reason of each delivery attempt."),
				cmpRule("R4", "A customer sees only deliveries from their own account."),
				cmpRule("R5", "A customer can retry a delivery from the list."),
			},
			Examples: []Example{{ID: "E8", Rule: "R1", Text: "Deliveries stay visible for 30 days."}},
			Questions: []Question{
				{ID: "Q1", Text: "Who can see the delivery list?", Status: QuestionAssumed, Answer: "Any user on the account", AnswerID: "A1"},
				{ID: "Q2", Text: "Should a failed delivery notify by email?", Status: QuestionOpen},
			},
		}
	})
	b = mk("webhook delivery log page", func(s *Session) {
		s.Status = StatusReady
		s.Map = Map{
			Rules: []Rule{
				cmpRule("R2", "A customer sees the status and the reason of each delivery attempt."),
				cmpRule("R4", "A customer sees only deliveries from their own account."),
			},
			Examples:  []Example{{ID: "E5", Rule: "R2", Text: "Deliveries stay visible for 60 days."}},
			Questions: []Question{answered("Q1", "Who can see the delivery list?", "Admins only", "A1")},
		}
	})
	c = mk("keep delivery history", func(s *Session) {
		s.Status = StatusSealed
		s.Digest = "d"
		s.Map = Map{
			Rules: []Rule{
				cmpRule("R1", "A customer sees the status and reason of each delivery attempt."),
				cmpRule("R2", "A customer sees only deliveries from their own account."),
				cmpRule("R3", "Deliveries stay visible for 90 days."),
			},
			Examples:  []Example{},
			Questions: []Question{},
		}
	})
	return ws, a, b, c
}

func TestCompareFindsSameConflictsAndDuplicates(t *testing.T) {
	ws, a, b, c := three(t)
	got, err := Compare(ws, []string{a.ID, b.ID, c.ID})
	if err != nil {
		t.Fatal(err)
	}
	if !got.Estimate || len(got.Sessions) != 3 {
		t.Fatalf("header %+v", got)
	}
	var wide []string
	for _, g := range got.Same {
		if len(g.Members) == 3 {
			wide = append(wide, g.Text)
		}
	}
	if len(wide) != 2 {
		t.Fatalf("rules in 3 sessions: %v (same %d)", wide, len(got.Same))
	}
	if len(got.Conflicts) != 2 {
		t.Fatalf("conflicts %+v", got.Conflicts)
	}
	byTitle := map[string]Conflict{}
	for _, c := range got.Conflicts {
		byTitle[c.Title] = c
	}
	who := byTitle["Who can see the delivery list?"]
	if len(who.Options) != 2 || who.Options[0].Members[0].Origin != originAssumed || !strings.HasPrefix(who.Options[1].Members[0].Origin, "answered by you") {
		t.Fatalf("who %+v", who)
	}
	var days Conflict
	for _, c := range got.Conflicts {
		if strings.Contains(strings.ToLower(c.Title), "days") {
			days = c
		}
	}
	if len(days.Options) != 3 {
		t.Fatalf("days %+v", days)
	}
	sealed := 0
	for _, o := range days.Options {
		if o.Members[0].Sealed && o.Members[0].Origin == originSealed {
			sealed++
		}
	}
	if sealed != 1 {
		t.Fatalf("sealed option %+v", days)
	}
	if got.Conflicts[0].ID != "C1" || got.Conflicts[1].ID != "C2" {
		t.Fatalf("ids %s %s", got.Conflicts[0].ID, got.Conflicts[1].ID)
	}
}

func TestCompareCountsDuplicatesAndUniqueItems(t *testing.T) {
	ws, a, b, c := three(t)
	got, err := Compare(ws, []string{c.ID, a.ID, b.ID})
	if err != nil {
		t.Fatal(err)
	}
	// 0138 has two rules and both are also in 0142: the same story twice.
	var dup *Duplicate
	for i := range got.Duplicates {
		d := got.Duplicates[i]
		if (d.A == a.ID && d.B == b.ID) || (d.A == b.ID && d.B == a.ID) {
			dup = &got.Duplicates[i]
		}
	}
	if dup == nil || dup.Percent < duplicatePercent || len(dup.AItems) != 2 || dup.Dismissed {
		t.Fatalf("duplicates %+v", got.Duplicates)
	}
	if got.Unique == 0 {
		t.Fatalf("unique %d", got.Unique)
	}
	if _, err := Dismiss(ws, []string{a.ID, b.ID}); err != nil {
		t.Fatal(err)
	}
	again, err := Compare(ws, []string{a.ID, b.ID, c.ID})
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range again.Duplicates {
		if (d.A == a.ID && d.B == b.ID) && !d.Dismissed {
			t.Fatalf("not dismissed: %+v", d)
		}
	}
}

func TestCompareRefusesWhatItCannotTake(t *testing.T) {
	ws, a, b, _ := three(t)
	if _, err := Compare(ws, []string{a.ID}); err == nil {
		t.Fatal("one session compared")
	}
	if _, err := Compare(ws, []string{a.ID, a.ID}); err == nil {
		t.Fatal("same session twice")
	}
	var coded *Error
	if _, err := Compare(ws, []string{a.ID, "missing"}); !errors.As(err, &coded) || coded.Code != "shape_not_found" {
		t.Fatalf("missing: %v", err)
	}
	saved(t, ws, b.ID, func(s *Session) { s.Repo = "other" })
	if _, err := Compare(ws, []string{a.ID, b.ID}); err == nil || !strings.Contains(err.Error(), "share one repo") {
		t.Fatalf("repo: %v", err)
	}
}

func TestCompareAcceptsTheSessionReference(t *testing.T) {
	ws, a, b, _ := three(t)
	if _, err := Compare(ws, []string{a.Ref(), strings.ToLower(b.Ref())}); err != nil {
		t.Fatal(err)
	}
}

func TestMergeNeedsEveryConflictDecidedAndNeverTouchesASealedSession(t *testing.T) {
	ws, a, b, c := three(t)
	var coded *Error
	_, err := Merge(ws, MergeInput{Into: b.ID, From: []string{a.ID}, Against: []string{c.ID}})
	if !errors.As(err, &coded) || coded.Code != "shape_conflicts_open" || !strings.Contains(err.Error(), "C1") {
		t.Fatalf("open conflicts: %v", err)
	}
	_, err = Merge(ws, MergeInput{Into: b.ID, From: []string{c.ID}})
	if !errors.As(err, &coded) || coded.Code != "shape_sealed" {
		t.Fatalf("sealed: %v", err)
	}
	if after := mustLoad(t, ws, b.ID); len(after.Map.Rules) != 2 {
		t.Fatalf("refused merge changed the target: %+v", after.Map)
	}
}

func TestMergeMovesUniqueItemsKeepsSameOnceAndAppliesTheDecisions(t *testing.T) {
	ws, a, b, c := three(t)
	cmp, err := Compare(ws, []string{a.ID, b.ID, c.ID})
	if err != nil {
		t.Fatal(err)
	}
	resolve := map[string]string{}
	for _, conf := range cmp.Conflicts {
		resolve[conf.ID] = b.Ref()
	}
	res, err := Merge(ws, MergeInput{Into: b.Ref(), From: []string{a.Ref()}, Against: []string{c.Ref()}, Resolve: resolve})
	if err != nil {
		t.Fatal(err)
	}
	target := mustLoad(t, ws, b.ID)
	texts := map[string]int{}
	for _, r := range target.Map.Rules {
		texts[r.Text]++
	}
	if texts["A customer sees only deliveries from their own account."] != 1 {
		t.Fatalf("same rule kept more than once: %v", texts)
	}
	if texts["A customer can retry a delivery from the list."] != 1 {
		t.Fatalf("unique rule not moved: %v", texts)
	}
	ids := map[string]bool{}
	for _, r := range target.Map.Rules {
		if ids[r.ID] {
			t.Fatalf("duplicate rule id %s", r.ID)
		}
		ids[r.ID] = true
	}
	var open, answer string
	for _, q := range target.Map.Questions {
		if q.Status == QuestionOpen {
			open = q.ID
		}
		if q.Text == "Who can see the delivery list?" {
			answer = q.Answer
		}
	}
	if open == "" || target.Status != StatusInterviewing {
		t.Fatalf("open question not moved or status %s", target.Status)
	}
	if answer != "Admins only" {
		t.Fatalf("decision not applied: %q", answer)
	}
	if res.Moved == 0 || len(res.Merged) != 1 {
		t.Fatalf("result %+v", res)
	}
	from := mustLoad(t, ws, a.ID)
	if from.MergedInto != b.ID {
		t.Fatalf("source not linked: %q", from.MergedInto)
	}
	if sealed := mustLoad(t, ws, c.ID); sealed.MergedInto != "" || len(sealed.Map.Rules) != 3 {
		t.Fatalf("sealed session changed: %+v", sealed)
	}
	got := strings.Join(labels(t, ws, b.ID), "|")
	if !strings.Contains(got, "Merged "+a.Ref()) {
		t.Fatalf("merge not in history: %s", got)
	}
}

func TestMergeTakesTheAnswerOfAnotherSessionWhenItWins(t *testing.T) {
	ws, a, b, _ := three(t)
	resolve := map[string]string{}
	cmp, err := Compare(ws, []string{a.ID, b.ID})
	if err != nil {
		t.Fatal(err)
	}
	for _, conf := range cmp.Conflicts {
		resolve[conf.ID] = a.ID
	}
	var coded *Error
	_, err = Merge(ws, MergeInput{Into: b.ID, From: []string{a.ID}, Resolve: map[string]string{"C1": "nobody", "C2": "nobody"}})
	if !errors.As(err, &coded) {
		t.Fatalf("bad winner: %v", err)
	}
	if _, err := Merge(ws, MergeInput{Into: b.ID, From: []string{a.ID}, Resolve: resolve}); err != nil {
		t.Fatal(err)
	}
	target := mustLoad(t, ws, b.ID)
	for _, q := range target.Map.Questions {
		if q.Text == "Who can see the delivery list?" && q.Answer != "Any user on the account" {
			t.Fatalf("winner's answer not applied: %q", q.Answer)
		}
	}
}
