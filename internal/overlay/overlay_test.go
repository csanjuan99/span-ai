package overlay

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func decide(t *testing.T, dir, dedupe, text string, epoch int) Entry {
	t.Helper()
	e, err := Append(dir, Entry{Kind: Decision, Stage: "build", Title: "AC 400", Text: text, Dedupe: dedupe, Epoch: epoch, Time: time.Unix(60, 0)})
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func TestTheLogOnlyGrowsAndKeepsEveryVersion(t *testing.T) {
	dir := t.TempDir()
	first := decide(t, dir, "k", "follow AGENTS.md", 0)
	second := decide(t, dir, "k", "follow the brief", 0)
	if first.Seq != 1 || second.Seq != 2 || len(second.Supersedes) != 1 || second.Supersedes[0] != 1 {
		t.Fatalf("first %+v second %+v", first, second)
	}
	raw, err := os.ReadFile(filepath.Join(dir, File))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "follow AGENTS.md") || !strings.Contains(string(raw), "follow the brief") {
		t.Fatalf("a superseded decision was lost:\n%s", raw)
	}
	active := Active(mustRead(t, dir))
	if len(active) != 1 || active[0].Text != "follow the brief" {
		t.Fatalf("active = %+v", active)
	}
}

func TestADecisionWithoutAKeyNeverSupersedesAnother(t *testing.T) {
	dir := t.TempDir()
	decide(t, dir, "", "a", 0)
	decide(t, dir, "", "b", 0)
	if got := Active(mustRead(t, dir)); len(got) != 2 {
		t.Fatalf("active = %+v", got)
	}
}

func TestStaleAnchorAndSupersede(t *testing.T) {
	dir := t.TempDir()
	one := decide(t, dir, "a", "one", 0)
	two := decide(t, dir, "b", "two", 0)
	if got := Stale(mustRead(t, dir), 0); len(got) != 0 {
		t.Fatalf("stale against the same story: %+v", got)
	}
	if got := Stale(mustRead(t, dir), 1); len(got) != 2 {
		t.Fatalf("stale after the story changed: %+v", got)
	}
	if _, err := Append(dir, Entry{Kind: Anchor, Ref: one.Seq, Epoch: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := Append(dir, Entry{Kind: Supersede, Supersedes: []int{two.Seq}}); err != nil {
		t.Fatal(err)
	}
	entries := mustRead(t, dir)
	if got := Stale(entries, 1); len(got) != 0 {
		t.Fatalf("an anchored or dropped decision is still stale: %+v", got)
	}
	if got := Active(entries); len(got) != 1 || got[0].Text != "one" {
		t.Fatalf("active = %+v", got)
	}
}

func TestRenderShowsWhoWhenAndWhereAndOnlyWhatStands(t *testing.T) {
	dir := t.TempDir()
	decide(t, dir, "k", "old", 0)
	decide(t, dir, "k", "follow AGENTS.md", 0)
	out := Render(mustRead(t, dir), 0)
	if !strings.Contains(out, "[D2]") || !strings.Contains(out, "follow AGENTS.md") || strings.Contains(out, "old") || !strings.Contains(out, "never override a rule the repository enforces") {
		t.Fatalf("render:\n%s", out)
	}
	if strings.Contains(out, "earlier version") {
		t.Fatalf("a current decision was marked stale:\n%s", out)
	}
	if !strings.Contains(Render(mustRead(t, dir), 3), "earlier version of the story") {
		t.Fatal("a stale decision was not marked")
	}
	if Render(nil, 0) != "" {
		t.Fatal("an empty log rendered text")
	}
}

func TestAppendNeedsAKindAndReadRejectsACorruptLine(t *testing.T) {
	dir := t.TempDir()
	if _, err := Append(dir, Entry{}); err == nil {
		t.Fatal("a line without a kind was written")
	}
	if err := os.WriteFile(filepath.Join(dir, File), []byte("{not json\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(dir); err == nil {
		t.Fatal("a corrupt line was read as empty")
	}
}

func mustRead(t *testing.T, dir string) []Entry {
	t.Helper()
	got, err := Read(dir)
	if err != nil {
		t.Fatal(err)
	}
	return got
}
