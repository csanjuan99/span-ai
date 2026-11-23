package policy

import "testing"

func TestWeightIsDerivedFromTheStory(t *testing.T) {
	cases := []struct {
		name     string
		criteria int
		handoffs []string
		want     string
	}{
		{"a fix with no criteria", 0, nil, Light},
		{"a small story", 3, nil, Light},
		{"a normal story", 8, nil, Standard},
		{"a story with many criteria", 21, nil, Deep},
		{"a story with a design handoff", 3, []string{"backoffice.dc.html"}, Deep},
		{"a story with a picture", 2, []string{"screen.png"}, Deep},
		{"a handoff that is only prose", 8, []string{"notes.md"}, Standard},
	}
	for _, row := range cases {
		if got := Derive(row.criteria, row.handoffs); got != row.want {
			t.Fatalf("%s: got %s, want %s", row.name, got, row.want)
		}
	}
}

func TestADeclaredWeightWins(t *testing.T) {
	file := File{Run: Run{Weight: "light"}}
	if got := file.Weight(21, []string{"screen.png"}); got != Light {
		t.Fatalf("declared weight ignored: %s", got)
	}
}

func TestEmptyWeightIsDerived(t *testing.T) {
	file := File{}
	if got := file.Weight(21, nil); got != Deep {
		t.Fatalf("got %s, want deep", got)
	}
}

func TestTheCapFollowsTheWeight(t *testing.T) {
	file := File{}
	for weight, want := range map[string]int{Light: 2, Standard: 3, Deep: 6} {
		if got := file.RepairCapFor(weight); got != want {
			t.Fatalf("%s: cap %d, want %d", weight, got, want)
		}
	}
	if got := (File{Repair: Repair{MaxAttempts: 4}}).RepairCapFor(Deep); got != 4 {
		t.Fatalf("an explicit cap was ignored: %d", got)
	}
}

func TestAFixDoesNotNeedTheWholeSuite(t *testing.T) {
	if WideOn(Light) {
		t.Fatal("a fix was asked for the whole suite")
	}
	for _, weight := range []string{Standard, Deep} {
		if !WideOn(weight) {
			t.Fatalf("%s skipped the whole suite", weight)
		}
	}
}

func TestAnUnknownWeightIsRefused(t *testing.T) {
	if err := (File{Feature: "payments", Run: Run{Weight: "enormous"}}).Validate(); err == nil {
		t.Fatal("an unknown weight passed validation")
	}
}
