package branch

import (
	"strings"
	"testing"
)

func TestPlanNewStoryFromDev(t *testing.T) {
	got, err := Derive("Add saved cards\n\nAs a shopper I save a card.", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "feat/add-saved-cards" || got.Base != "dev" || got.Kind != "feat" {
		t.Fatalf("%+v", got)
	}
}

func TestPlanLinkedStoryKeepsKey(t *testing.T) {
	got, err := Derive("Show the channel list", "AET-1478", "feat", "")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "feat/AET-1478-show-the-channel-list" || got.Base != "dev" {
		t.Fatalf("%+v", got)
	}
}

func TestPlanKeyInsideTheStory(t *testing.T) {
	got, err := Derive("AET-1478 Show channels", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "feat/AET-1478-show-channels" {
		t.Fatalf("%+v", got)
	}
}

func TestPlanFixFromDevAndHotfixFromMaster(t *testing.T) {
	fix, err := Derive("fix: ledger rounds the wrong way", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if fix.Name != "fix/ledger-rounds-the-wrong-way" || fix.Base != "dev" {
		t.Fatalf("%+v", fix)
	}
	hot, err := Derive("refund the double charge", "", "", "master")
	if err != nil {
		t.Fatal(err)
	}
	if hot.Name != "hotfix/refund-the-double-charge" || hot.Base != "master" || hot.Kind != "hotfix" {
		t.Fatalf("%+v", hot)
	}
}

func TestPlanRejectsOtherPrefixesAndWrongBase(t *testing.T) {
	if _, err := Derive("Add saved cards", "", "chore", ""); err == nil {
		t.Fatal("chore was accepted")
	}
	if _, err := Derive("Add saved cards", "", "feat", "master"); err == nil {
		t.Fatal("feat from master was accepted")
	}
	if _, err := Derive("Add saved cards", "", "hotfix", "dev"); err == nil {
		t.Fatal("hotfix from dev was accepted")
	}
}

func TestASpanishStoryStillProducesAValidBranch(t *testing.T) {
	story := "SAL-4 Retirar de Hermes los módulos que no se usan y dejar la navegación limpia\nType: Story\n"
	plan, err := Derive(story, "", "", "")
	if err != nil {
		t.Fatalf("a Spanish title was rejected: %v", err)
	}
	if !Valid(plan.Name, plan.Base) {
		t.Fatalf("derived name is not valid: %q", plan.Name)
	}
	for _, r := range plan.Name {
		if r > 127 {
			t.Fatalf("branch %q kept a non-ASCII letter", plan.Name)
		}
	}
	if strings.Contains(plan.Name, "modulos") == false {
		t.Fatalf("the accent was dropped instead of folded: %q", plan.Name)
	}
}

func TestTheBranchBudgetIsTheTitleNotTheWholeSlug(t *testing.T) {
	story := "SAL-4 Retirar de Hermes los módulos que no se usan y dejar la navegación limpia\nType: Story\n"
	plan, err := Derive(story, "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	title := strings.TrimPrefix(plan.Name, "feat/SAL-4-")
	if len([]rune(title)) > 48 {
		t.Fatalf("title is %d runes: %q", len([]rune(title)), title)
	}
	if strings.HasSuffix(title, "-") {
		t.Fatalf("the title ends with a dash: %q", title)
	}
	if plan.Feature != strings.ToLower("SAL-4-"+title) {
		t.Fatalf("feature = %q", plan.Feature)
	}
}

func TestTheFeatureNamePassesTheSameRuleAsThePolicy(t *testing.T) {
	plan, err := Derive("SAL-4 Módulos y navegación", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range plan.Feature {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
		default:
			t.Fatalf("feature %q has %q", plan.Feature, r)
		}
	}
}

// The exact story that failed: the accent in "módulos" made the derived name invalid.
func TestTheStoryThatFailedNowDerives(t *testing.T) {
	story := "SAL-4 Retirar de Hermes los módulos que no se usan\nType: Story\n"
	plan, err := Derive(story, "", "", "")
	if err != nil {
		t.Fatalf("derive: %v", err)
	}
	if plan.Name != "feat/SAL-4-retirar-de-hermes-los-modulos-que-no-se-usan" {
		t.Fatalf("name = %q", plan.Name)
	}
	if plan.Feature != "sal-4-retirar-de-hermes-los-modulos-que-no-se-usan" {
		t.Fatalf("feature = %q", plan.Feature)
	}
}
