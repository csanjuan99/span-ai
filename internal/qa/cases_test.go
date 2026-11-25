package qa

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const goJSON = `{"Action":"run","Package":"acme/api/internal/webhook","Test":"TestRetryBackoff"}
{"Action":"output","Package":"acme/api/internal/webhook","Test":"TestRetryBackoff","Output":"=== RUN   TestRetryBackoff\n"}
{"Action":"output","Package":"acme/api/internal/webhook","Test":"TestRetryBackoff","Output":"    retry_test.go:58: delay 2s, want 4s\n"}
{"Action":"output","Package":"acme/api/internal/webhook","Test":"TestRetryBackoff","Output":"--- FAIL: TestRetryBackoff (0.02s)\n"}
{"Action":"fail","Package":"acme/api/internal/webhook","Test":"TestRetryBackoff","Elapsed":0.02}
{"Action":"pass","Package":"acme/api/internal/webhook","Test":"TestSendOK","Elapsed":0.01}
{"Action":"output","Package":"acme/api/internal/webhook","Test":"TestSendOK/sub","Output":"    sub ran\n"}
{"Action":"pass","Package":"acme/api/internal/webhook","Test":"TestSendOK/sub","Elapsed":0}
{"Action":"skip","Package":"acme/api/internal/webhook","Test":"TestSlow","Elapsed":0}
{"Action":"fail","Package":"acme/api/internal/webhook","Elapsed":0.4}
`

func TestParseGoJSONReadsEachTopLevelCase(t *testing.T) {
	got, ok := ParseGoTests(goJSON)
	if !ok || len(got) != 3 {
		t.Fatalf("ok %v cases %#v", ok, got)
	}
	fail := got[0]
	if fail.Name != "TestRetryBackoff" || fail.Result != CaseFail || fail.ElapsedMS != 20 {
		t.Fatalf("fail case %#v", fail)
	}
	if fail.File != "retry_test.go" || fail.Line != 58 || fail.Message != "delay 2s, want 4s" {
		t.Fatalf("where the case failed: %#v", fail)
	}
	if got[1].Result != CasePass || got[2].Result != CaseSkip {
		t.Fatalf("results %#v", got)
	}
	if !strings.Contains(strings.Join(got[1].Output, "\n"), "sub ran") {
		t.Fatalf("a subtest's output joins its parent: %#v", got[1].Output)
	}
}

func TestParseGoVerboseReadsCasesAndTheirPackage(t *testing.T) {
	text := "=== RUN   TestA\n    a_test.go:10: boom\n--- FAIL: TestA (0.03s)\n=== RUN   TestB\n--- PASS: TestB (0.00s)\nFAIL\nFAIL\tacme/api/internal/x\t0.4s\n"
	got, ok := ParseGoTests(text)
	if !ok || len(got) != 2 {
		t.Fatalf("ok %v cases %#v", ok, got)
	}
	if got[0].Package != "acme/api/internal/x" || got[0].Line != 10 || got[0].ElapsedMS != 30 {
		t.Fatalf("case %#v", got[0])
	}
}

func TestParseGoTestsFindsNothingInQuietOutput(t *testing.T) {
	if cases, ok := ParseGoTests("ok  \tacme/api\t0.4s\n"); ok || len(cases) != 0 {
		t.Fatalf("a quiet go test names no case: %#v", cases)
	}
}

func TestPlaceTestsPlacesPassingCasesAtTheirDeclaration(t *testing.T) {
	dir := t.TempDir()
	pkg := filepath.Join(dir, "internal", "webhook")
	if err := os.MkdirAll(pkg, 0o755); err != nil {
		t.Fatal(err)
	}
	must := func(path, text string) {
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	must(filepath.Join(dir, "go.mod"), "module acme/api\n\ngo 1.22\n")
	must(filepath.Join(pkg, "client_test.go"), "package webhook\n\nimport \"testing\"\n\nfunc TestSendOK(t *testing.T) {}\n")
	cases := []TestCase{
		{Name: "TestSendOK", Package: "acme/api/internal/webhook", Result: CasePass},
		{Name: "TestRetryBackoff", Package: "acme/api/internal/webhook", Result: CaseFail, File: "retry_test.go", Line: 58},
		{Name: "TestElsewhere", Package: "other/mod/x", Result: CasePass},
	}
	got := PlaceTests(dir, cases)
	if got[0].Path != "internal/webhook/client_test.go" || got[0].Line != 5 || got[0].File != "client_test.go" {
		t.Fatalf("passing case %#v", got[0])
	}
	if got[1].Path != "internal/webhook/retry_test.go" || got[1].Line != 58 {
		t.Fatalf("failing case keeps its assertion line: %#v", got[1])
	}
	if got[2].Path != "" {
		t.Fatalf("a package outside the module stays unplaced: %#v", got[2])
	}
}

func TestAddedTestsReadsTheTestsAPatchAdds(t *testing.T) {
	patch := "+func TestNew(t *testing.T) {\n-func TestGone(t *testing.T) {\n func TestKept(t *testing.T) {\n"
	added := AddedTests(patch)
	if !added["TestNew"] || added["TestGone"] || added["TestKept"] {
		t.Fatalf("added %#v", added)
	}
	cases := MarkNew([]TestCase{{Name: "TestNew"}, {Name: "TestKept"}}, added)
	if !cases[0].New || cases[1].New {
		t.Fatalf("cases %#v", cases)
	}
}
