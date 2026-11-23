package branch

import "testing"

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
