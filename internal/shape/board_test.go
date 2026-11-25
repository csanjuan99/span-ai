package shape

import (
	"context"
	"strings"
	"testing"
	"time"
)

func boardFixture(t *testing.T) (*Engine, string, map[string]*Session) {
	t.Helper()
	two := []map[string]any{q("Q1", "who?", "admins"), q("Q2", "when?", "daily")}
	f := &fake{replies: []string{reply(t, turn(two, "")), reply(t, turn(two, "# story"))}}
	e, ws := engine(t, f)
	out := map[string]*Session{}
	out["open"] = start(t, e, ws)
	ready := start(t, e, ws)
	if _, err := e.Answer(context.Background(), ws, ready.ID, nil, []string{"Q1", "Q2"}); err != nil {
		t.Fatal(err)
	}
	out["ready"] = mustLoad(t, ws, ready.ID)
	return e, ws, out
}

func TestBoardGroupsSessionsByStatus(t *testing.T) {
	_, ws, made := boardFixture(t)
	view, err := Board(ws, Filter{}, 0)
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, col := range view.Columns {
		counts[col.Key] = col.Count
	}
	if len(view.Columns) != 4 || counts[StatusInterviewing] != 1 || counts[StatusReady] != 1 || counts[StatusSplit] != 0 || counts[StatusSealed] != 0 {
		t.Fatalf("columns %+v", counts)
	}
	if view.InFlight != 2 || view.NeedYou != 2 {
		t.Fatalf("in flight %d need you %d", view.InFlight, view.NeedYou)
	}
	card := view.Columns[0].Cards[0]
	if card.ID != made["open"].ID || card.OpenQuestions != 2 || !strings.Contains(card.Doing, "asked you 2 questions") {
		t.Fatalf("card %+v", card)
	}
	ready := view.Columns[1].Cards[0]
	if ready.Assumed != 2 || ready.OpenQuestions != 0 {
		t.Fatalf("ready card %+v", ready)
	}
	if _, err := Board(ws, Filter{}, -1); codeOf(err) != "invalid" {
		t.Fatalf("a negative limit is invalid, got %v", err)
	}
}

func TestBoardLimitSaysWhatItHid(t *testing.T) {
	e, ws, _ := boardFixture(t)
	for i := 0; i < 2; i++ {
		start(t, e, ws)
	}
	view, err := Board(ws, Filter{}, 2)
	if err != nil {
		t.Fatal(err)
	}
	col := view.Columns[0]
	if col.Count != 3 || len(col.Cards) != 2 || col.Hidden != 1 {
		t.Fatalf("column %+v", col)
	}
}

func TestBulkAssumeDryRunChangesNothingAndRealRunSettlesEach(t *testing.T) {
	e, ws, made := boardFixture(t)
	engineFor := func(*Session) *Engine { return e }
	dry, err := Bulk(context.Background(), ws, BulkAssume, []string{made["open"].ID, made["ready"].ID}, true, engineFor)
	if err != nil {
		t.Fatal(err)
	}
	if !dry.DryRun || dry.Done != 1 || len(dry.Items[0].Changed) != 2 || !dry.Items[1].Skipped {
		t.Fatalf("dry run %+v", dry)
	}
	if mustLoad(t, ws, made["open"].ID).Map.Questions[0].Status != QuestionOpen {
		t.Fatal("a dry run changes nothing")
	}
	res, err := Bulk(context.Background(), ws, BulkAssume, []string{made["open"].ID, "nope"}, false, engineFor)
	if err != nil {
		t.Fatal(err)
	}
	if res.Done != 1 || res.Failed != 1 || res.Items[1].Code != "shape_not_found" {
		t.Fatalf("one failure must not stop the others: %+v", res)
	}
	if mustLoad(t, ws, made["open"].ID).Map.Questions[0].Status != QuestionAssumed {
		t.Fatal("the assumption was accepted")
	}
}

func TestBulkSealReportsPerSession(t *testing.T) {
	e, ws, made := boardFixture(t)
	engineFor := func(*Session) *Engine { return e }
	dry, err := Bulk(context.Background(), ws, BulkSeal, []string{made["ready"].ID, made["open"].ID}, true, engineFor)
	if err != nil {
		t.Fatal(err)
	}
	if dry.Items[0].OK != true || len(dry.Items[0].Changed) != 2 || dry.Items[1].Code != "shape_not_ready" {
		t.Fatalf("dry run %+v", dry)
	}
	res, err := Bulk(context.Background(), ws, BulkSeal, []string{made["ready"].ID, made["open"].ID}, false, engineFor)
	if err != nil || res.Done != 1 || res.Failed != 1 {
		t.Fatalf("%+v %v", res, err)
	}
	if mustLoad(t, ws, made["ready"].ID).Status != StatusSealed || mustLoad(t, ws, made["open"].ID).Status == StatusSealed {
		t.Fatal("only the ready one is sealed")
	}
	again, err := Bulk(context.Background(), ws, BulkAssume, []string{made["ready"].ID}, false, engineFor)
	if err != nil {
		t.Fatal(err)
	}
	if again.Items[0].Code != "shape_sealed" {
		t.Fatalf("a sealed story cannot change: %+v", again)
	}
}

func TestBulkArchiveAndValidation(t *testing.T) {
	e, ws, made := boardFixture(t)
	engineFor := func(*Session) *Engine { return e }
	id := made["open"].ID
	res, err := Bulk(context.Background(), ws, BulkArchive, []string{id, id}, false, engineFor)
	if err != nil || len(res.Items) != 1 || !mustLoad(t, ws, id).Archived {
		t.Fatalf("%+v %v", res, err)
	}
	again, err := Bulk(context.Background(), ws, BulkArchive, []string{id}, false, engineFor)
	if err != nil {
		t.Fatal(err)
	}
	if !again.Items[0].Skipped {
		t.Fatalf("archiving twice is skipped: %+v", again)
	}
	if _, err := Bulk(context.Background(), ws, "explode", []string{id}, false, engineFor); codeOf(err) != "invalid" {
		t.Fatalf("got %v", err)
	}
	if _, err := Bulk(context.Background(), ws, BulkArchive, nil, false, engineFor); codeOf(err) != "invalid" {
		t.Fatalf("got %v", err)
	}
}

func TestRecentActivityShowsAgentsFirstAndHonoursSince(t *testing.T) {
	_, ws, _ := boardFixture(t)
	feed, err := RecentActivity(ws, "", time.Time{}, 0)
	if err != nil || len(feed) == 0 {
		t.Fatalf("%v %v", feed, err)
	}
	for _, a := range feed {
		if a.Actor == personActor {
			t.Fatalf("the default feed is the agents: %+v", a)
		}
	}
	all, err := RecentActivity(ws, "everyone", time.Time{}, 0)
	if err != nil || len(all) <= len(feed) {
		t.Fatalf("everyone shows the person too: %d vs %d %v", len(all), len(feed), err)
	}
	newer, err := RecentActivity(ws, "everyone", time.Now().Add(time.Hour), 0)
	if err != nil || len(newer) != 0 {
		t.Fatalf("nothing is newer than the future: %v %v", newer, err)
	}
	if _, err := RecentActivity(ws, "robots", time.Time{}, 0); codeOf(err) != "invalid" {
		t.Fatalf("got %v", err)
	}
}
