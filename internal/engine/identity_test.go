package engine

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"span/internal/statement"
)

// Two runs in one workspace must not share a feature document, a pull request title, or a
// commit scope: that sharing is what made parallel runs conflict.
func TestRunsOnDifferentBranchesKeepTheirOwnIdentity(t *testing.T) {
	opt, _, source, review, _, _ := baseOpt(t)
	runs := []struct{ branch, brief, feature, title string }{
		{"feat/SAL-1-cards", "# Add saved cards\n\nCharge the card.", "sal-1-cards", "Add saved cards"},
		{"feat/SAL-2-wallet", "\n## Pay with the wallet\n\nCharge the wallet.", "sal-2-wallet", "Pay with the wallet"},
	}
	for i, run := range runs {
		opt.Branch, opt.Brief = run.branch, run.brief
		review.found = false
		source.dirty = true
		got, err := Start(context.Background(), opt)
		if err != nil {
			t.Fatal(err)
		}
		if got.Verdict != statement.Advance {
			t.Fatalf("%s: %+v", run.branch, got)
		}
		snap, err := Load(opt.Root, got.ID)
		if err != nil {
			t.Fatal(err)
		}
		if snap.Cursor.Feature != run.feature {
			t.Fatalf("cursor feature %q, want %q", snap.Cursor.Feature, run.feature)
		}
		built, ok, err := statement.Read(filepath.Join(opt.Root, got.ID, "iter", "1"), build)
		if err != nil || !ok {
			t.Fatalf("build statement ok=%v err=%v", ok, err)
		}
		if want := filepath.Join(opt.Workspace, "odd", "tasks", run.feature+".md"); built.FeatureDoc != want {
			t.Fatalf("feature doc %q, want %q", built.FeatureDoc, want)
		}
		if review.titles[i] != run.title {
			t.Fatalf("pull request title %q, want %q", review.titles[i], run.title)
		}
		if want := "feat(" + run.feature + "): span " + got.ID; source.messages[len(source.messages)-1] != want {
			t.Fatalf("commit %q, want %q", source.messages[len(source.messages)-1], want)
		}
	}
}

func TestTheProjectFeatureLineNoLongerNamesTheRun(t *testing.T) {
	opt, agents, _, _, _, _ := baseOpt(t)
	opt.Policy.Feature = "hermes"
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != statement.Advance {
		t.Fatalf("%+v", got)
	}
	// fakeSource reports feat/payments as the checked-out branch.
	if contains(agents.prompts, "hermes.md") || !contains(agents.prompts, filepath.Join("odd", "tasks", "payments.md")) {
		t.Fatal("the build prompt did not name the branch's feature document")
	}
}

func TestAnOldCursorTakesItsFeatureFromItsBranch(t *testing.T) {
	opt, _, _, _, _, _ := baseOpt(t)
	opt.Policy.Feature = "hermes"
	id, err := Open(opt)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(opt.Root, id, "cursor.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(err)
	}
	delete(fields, "feature")
	fields["branch"] = "fix/SAL-9-Refund"
	fields["base"] = "dev"
	raw, err = json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	r, err := load(opt, id)
	if err != nil {
		t.Fatal(err)
	}
	if r.cur.Feature != "sal-9-refund" {
		t.Fatalf("feature %q, want sal-9-refund", r.cur.Feature)
	}
}

func TestAStoredFeatureThatIsAPathIsNotUsed(t *testing.T) {
	opt, _, _, _, _, _ := baseOpt(t)
	opt.Branch = "feat/SAL-5-safe"
	id, err := Open(opt)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(opt.Root, id, "cursor.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(err)
	}
	if fields["feature"] != "sal-5-safe" {
		t.Fatalf("Open stored feature %v", fields["feature"])
	}
	fields["feature"] = "../escape"
	raw, err = json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	r, err := load(opt, id)
	if err != nil {
		t.Fatal(err)
	}
	if r.cur.Feature != "sal-5-safe" {
		t.Fatalf("feature %q, want sal-5-safe", r.cur.Feature)
	}
}

func TestStoryTitleIsTheFirstLineOfTheBrief(t *testing.T) {
	long := strings.Repeat("a", 130)
	for brief, want := range map[string]string{
		"# Add saved cards\n\nbody": "Add saved cards",
		"\n\n  ##   Pay  \nmore":    "Pay",
		"#\n\nReal title":           "Real title",
		long:                        strings.Repeat("a", 100),
		"  \n#  \n":                 "fallback",
	} {
		if got := storyTitle(brief, "fallback"); got != want {
			t.Fatalf("storyTitle(%q) = %q, want %q", brief, got, want)
		}
	}
}
