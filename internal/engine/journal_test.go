package engine

import (
	"context"
	"path/filepath"
	"testing"

	"span/internal/journal"
)

func TestEveryStatementLeavesAStageEventInAnIntactJournal(t *testing.T) {
	opt, _, _, _, _, _ := baseOpt(t)
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(opt.Root, got.ID)
	events, err := journal.Read(dir)
	if err != nil {
		t.Fatal(err)
	}
	snap, err := Load(opt.Root, got.ID)
	if err != nil {
		t.Fatal(err)
	}
	// Other lines, such as the agent a stage ran with, sit between the stage events.
	var stages []journal.Event
	for _, event := range events {
		if event.Type == "stage" {
			stages = append(stages, event)
		}
	}
	events = stages
	if len(events) == 0 || len(events) != len(snap.Statements) {
		t.Fatalf("%d events for %d statements", len(events), len(snap.Statements))
	}
	for index, event := range events {
		item := snap.Statements[index]
		if event.Actor != journal.Span || event.Type != "stage" || event.Detail["stage"] != item.Stage || event.Detail["digest"] != item.Digest {
			t.Fatalf("event %d does not match its statement: %+v vs %s %s", index, event, item.Stage, item.Digest)
		}
	}
	if bad, err := journal.Verify(dir); err != nil || bad != 0 {
		t.Fatalf("the journal chain broke: bad=%d err=%v", bad, err)
	}
}
