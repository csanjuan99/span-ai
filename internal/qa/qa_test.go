package qa

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLinesNumberTheAcceptanceSection(t *testing.T) {
	brief := "AET-12 Charge\n\nAcceptance criteria:\n- Charges the card.\n* Keeps a receipt.\n\n"
	lines := Lines(brief)
	if len(lines) != 2 || lines[0].ID != "AC-1" || lines[1].ID != "AC-2" || lines[0].Text != "Charges the card." {
		t.Fatalf("%+v", lines)
	}
	if HasAcceptance("Charge the card.\n") {
		t.Fatal("plain story looked like acceptance")
	}
}

func TestInstallCarriesFoundCriteriaIntoTheStandardSection(t *testing.T) {
	story := "SAL-3 Estado\n\nQuien tiene el permiso ve la entrada.\n"
	if HasAcceptance(story) {
		t.Fatal("the story was treated as the standard section before it was carried in")
	}
	got := Install(story, []string{"Quien tiene el permiso ve la entrada."})
	lines := Lines(got)
	if len(lines) != 1 || lines[0].ID != "AC-1" || lines[0].Text != "Quien tiene el permiso ve la entrada." {
		t.Fatalf("%+v\n%s", lines, got)
	}
	if !strings.Contains(got, "\n\nAcceptance criteria:\n") {
		t.Fatalf("%s", got)
	}
}

func TestLinesKeepAnIdentifierTheStoryAlreadyUses(t *testing.T) {
	brief := "Acceptance criteria:\nAC 100 - Entrada\nAC-101 Llegada\n"
	lines := Lines(brief)
	if len(lines) != 2 || lines[0].ID != "AC-100" || lines[1].ID != "AC-101" {
		t.Fatalf("%+v", lines)
	}
}

func TestCoveredIgnoresCommentsAndLongerIDs(t *testing.T) {
	root := t.TempDir()
	body := "package pay\n\n// AC-1 is only a comment\nfunc TestAC1_Charges(t *testing.T) {}\nfunc TestAC10_Other(t *testing.T) {}\n"
	if err := os.WriteFile(filepath.Join(root, "pay_test.go"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	lines := []Line{{ID: "AC-1", Text: "Charges"}, {ID: "AC-2", Text: "Receipt"}, {ID: "AC-10", Text: "Other"}}
	got, err := Covered(root, lines)
	if err != nil {
		t.Fatal(err)
	}
	if !got["AC-1"] || got["AC-2"] || !got["AC-10"] {
		t.Fatalf("%v", got)
	}
}

func TestJudgeFailsABrokenCommandAndLeavesAMissingIDUnverified(t *testing.T) {
	lines := []Line{{ID: "AC-1"}, {ID: "AC-2"}}
	failed := Judge(lines, 1, true, map[string]bool{"AC-1": true})
	if failed.Verdict != "fail" || failed.Checks[0].Result != "fail" || failed.Checks[1].Result != "unverified" {
		t.Fatalf("%+v", failed)
	}
	passed := Judge(lines, 0, true, map[string]bool{"AC-1": true})
	if passed.Verdict != "unverified" || passed.Checks[0].Result != "pass" {
		t.Fatalf("%+v", passed)
	}
	ok := Judge(lines, 0, true, map[string]bool{"AC-1": true, "AC-2": true})
	if ok.Verdict != "pass" {
		t.Fatalf("%+v", ok)
	}
}

func TestPresentNamesTheDeclarationAndLeavesAMissingIDUnverified(t *testing.T) {
	root := t.TempDir()
	body := "package pay\n\nfunc TestAC1_Charges(t *testing.T) {}\n"
	if err := os.WriteFile(filepath.Join(root, "pay_test.go"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	lines := []Line{{ID: "AC-1", Text: "Charges"}, {ID: "AC-2", Text: "Receipt"}}
	gate := Judge(lines, 0, true, map[string]bool{"AC-1": true})
	got, err := Present(root, lines, &gate)
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Result != "pass" || got[0].File != "pay_test.go" || got[0].Line != 3 {
		t.Fatalf("covered %+v", got[0])
	}
	if got[1].Result != "unverified" || got[1].File != "" {
		t.Fatalf("missing %+v", got[1])
	}
}
