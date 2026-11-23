package qa

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func criteria(n int) []Line {
	out := make([]Line, 0, n)
	for i := 1; i <= n; i++ {
		out = append(out, Line{ID: "AC-" + itoa(100*i), Text: "criterion"})
	}
	return out
}

func covered(lines []Line) map[string]bool {
	out := map[string]bool{}
	for _, line := range lines {
		out[line.ID] = true
	}
	return out
}

func count(gate Gate, result string) int {
	n := 0
	for _, check := range gate.Checks {
		if check.Result == result {
			n++
		}
	}
	return n
}

// The incident: prove passed the fast suite on the same commit, QA ran it again,
// the process exited 1, and every declared criterion was painted fail.
func TestANonZeroExitDoesNotMarkEveryCoveredIDFail(t *testing.T) {
	lines := criteria(22)
	gate := Judge(lines, 1, covered(lines), Outcomes{})
	if got := count(gate, "fail"); got != 0 {
		t.Fatalf("%d criteria were marked fail without a per-test report", got)
	}
	if got := count(gate, "unverified"); got != 22 {
		t.Fatalf("unverified = %d, want 22", got)
	}
	if gate.Verdict != "unverified" {
		t.Fatalf("verdict = %s", gate.Verdict)
	}
	if gate.Evidence != "exit-only" {
		t.Fatalf("evidence = %s", gate.Evidence)
	}
	if gate.Reason == "" {
		t.Fatal("the gate has no reason a reader can act on")
	}
}

func TestAReportAttributesEachCriterion(t *testing.T) {
	lines := []Line{{ID: "AC-100"}, {ID: "AC-200"}, {ID: "AC-300"}}
	out := Outcomes{Source: "junit", Tests: []Outcome{
		{Name: "totals sums the lines AC-100", Failed: false},
		{Name: "totals rounds up AC-200", Failed: true},
		{Name: "totals ignores empty AC-300", Failed: false},
	}}
	gate := Judge(lines, 1, covered(lines), out)
	if gate.Verdict != "fail" {
		t.Fatalf("verdict = %s", gate.Verdict)
	}
	want := map[string]string{"AC-100": "pass", "AC-200": "fail", "AC-300": "pass"}
	for _, check := range gate.Checks {
		if check.Result != want[check.ID] {
			t.Fatalf("%s = %s, want %s", check.ID, check.Result, want[check.ID])
		}
	}
	if gate.Evidence != "junit" {
		t.Fatalf("evidence = %s", gate.Evidence)
	}
}

func TestADeclaredCriterionTheReportNeverRanIsUnverified(t *testing.T) {
	lines := []Line{{ID: "AC-100"}, {ID: "AC-200"}}
	out := Outcomes{Source: "junit", Tests: []Outcome{
		{Name: "totals sums the lines AC-100", Failed: false},
	}}
	gate := Judge(lines, 0, covered(lines), out)
	if gate.Verdict != "unverified" {
		t.Fatalf("verdict = %s", gate.Verdict)
	}
	for _, check := range gate.Checks {
		if check.ID == "AC-200" && check.Result != "unverified" {
			t.Fatalf("a criterion that never ran is %s", check.Result)
		}
	}
}

func TestExitZeroWithoutAReportPassesDeclaredCriteria(t *testing.T) {
	lines := []Line{{ID: "AC-100"}}
	gate := Judge(lines, 0, covered(lines), Outcomes{})
	if gate.Verdict != "pass" {
		t.Fatalf("verdict = %s", gate.Verdict)
	}
	if gate.Evidence != "exit-only" {
		t.Fatalf("evidence = %s", gate.Evidence)
	}
}

func TestACriterionWithoutADeclarationIsUnverified(t *testing.T) {
	lines := []Line{{ID: "AC-100"}, {ID: "AC-200"}}
	gate := Judge(lines, 0, map[string]bool{"AC-100": true}, Outcomes{})
	if gate.Verdict != "unverified" {
		t.Fatalf("verdict = %s", gate.Verdict)
	}
	if count(gate, "unverified") != 1 {
		t.Fatalf("unverified = %d", count(gate, "unverified"))
	}
}

func TestParseJUnitReadsFailures(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "junit.xml")
	body := `<?xml version="1.0"?>
<testsuites><testsuite name="cart">
  <testcase classname="cart.test.ts" name="sums the lines AC-100"/>
  <testcase classname="cart.test.ts" name="rounds up AC-200"><failure message="expected 2 to be 3"/></testcase>
</testsuite></testsuites>`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	if out.Source != "junit" || len(out.Tests) != 2 {
		t.Fatalf("source=%s tests=%d", out.Source, len(out.Tests))
	}
	failed := 0
	for _, test := range out.Tests {
		if test.Failed {
			failed++
		}
	}
	if failed != 1 {
		t.Fatalf("failed = %d, want 1", failed)
	}
}

func TestParseVitestJSONReadsFailures(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "results.json")
	body := `{"testResults":[{"name":"src/cart.test.ts","assertionResults":[
	  {"fullName":"totals sums the lines AC-100","title":"sums","status":"passed"},
	  {"fullName":"totals rounds up AC-200","title":"rounds","status":"failed"}
	]}]}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	if out.Source != "vitest-json" || len(out.Tests) != 2 {
		t.Fatalf("source=%s tests=%d", out.Source, len(out.Tests))
	}
	if !out.Tests[1].Failed {
		t.Fatal("the failing test was reported as a pass")
	}
}

func TestFindIgnoresAReportOlderThanTheRun(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "test-results")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "junit.xml")
	body := `<testsuites><testcase classname="a" name="b AC-100"/></testsuites>`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
	out, _, err := Find(root, "", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if out.Known() {
		t.Fatal("a stale report was accepted as this run's evidence")
	}
}

func TestScopeIsTheDeclaringFilesForThisStory(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"src/cartera/statement.test.ts":   "it('AC-100 sums the lines', () => {})\n",
		"src/cartera/sidebar.test.ts":     "it('AC-101 shows the badge', () => {})\n",
		"src/persons/plan-detail.test.ts": "it('AC-900 other story', () => {})\n",
	}
	for name, body := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	lines := []Line{{ID: "AC-100"}, {ID: "AC-101"}, {ID: "AC-200"}}
	got, err := Scope(root, lines)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"src/cartera/sidebar.test.ts", "src/cartera/statement.test.ts"}
	if len(got) != len(want) {
		t.Fatalf("scope = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("scope = %v, want %v", got, want)
		}
	}
}

func TestScopeIsEmptyWhenNothingIsDeclared(t *testing.T) {
	got, err := Scope(t.TempDir(), []Line{{ID: "AC-100"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("scope = %v", got)
	}
}
