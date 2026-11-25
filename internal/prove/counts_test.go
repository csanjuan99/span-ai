package prove

import (
	"context"
	"testing"

	"span/internal/statement"
)

func TestCountTestsReadsTheSummaryOfEachRunner(t *testing.T) {
	cases := []struct {
		name string
		text string
		want statement.TestCounts
	}{
		{"vitest with a failure", "\n Test Files  1 failed | 52 passed (53)\n      Tests  2 failed | 338 passed (340)\n", statement.TestCounts{Passed: 338, Failed: 2}},
		{"vitest all green", "      Tests  340 passed (340)\n", statement.TestCounts{Passed: 340}},
		{"vitest with skipped", "      Tests  1 failed | 4 skipped | 8 passed (13)\n", statement.TestCounts{Passed: 8, Failed: 1, Skipped: 4}},
		{"jest", "Tests:       2 failed, 41 passed, 43 total\nSnapshots:   0 total\n", statement.TestCounts{Passed: 41, Failed: 2}},
		{"pytest", "=========== 2 failed, 41 passed, 1 skipped in 3.20s ===========\n", statement.TestCounts{Passed: 41, Failed: 2, Skipped: 1}},
		{"pytest all green", "============ 41 passed in 0.52s ============\n", statement.TestCounts{Passed: 41}},
		{"go test -v counts the top-level tests only", "=== RUN   TestA\n--- PASS: TestA (0.00s)\n=== RUN   TestB\n--- FAIL: TestB (0.00s)\n    --- FAIL: TestB/sub (0.00s)\n=== RUN   TestC\n--- SKIP: TestC (0.00s)\nFAIL\n", statement.TestCounts{Passed: 1, Failed: 1, Skipped: 1}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := CountTests(c.text)
			if !ok || got != c.want {
				t.Fatalf("got %+v, %v; want %+v", got, ok, c.want)
			}
		})
	}
}

func TestCountTestsSaysNothingItCannotKnow(t *testing.T) {
	for _, text := range []string{
		"",
		"ok  \tspan/internal/prove\t0.412s\n",
		"--- FAIL: TestB (0.00s)\nFAIL\n",
		"lint passed\n",
	} {
		if got, ok := CountTests(text); ok {
			t.Fatalf("%q gave %+v, want no count", text, got)
		}
	}
}

func TestABandKeepsTheCountsOfACommandThatPrintsThem(t *testing.T) {
	s := &script{output: map[string]string{"npm test": "      Tests  2 failed | 3 passed (5)\n"}, fail: map[string]int{"npm test": 1}}
	got, err := Bands(context.Background(), s, t.TempDir(), Plan{Fast: []string{"npm test"}})
	if err != nil {
		t.Fatal(err)
	}
	counts := got.Commands[0].Tests
	if counts == nil || counts.Passed != 3 || counts.Failed != 2 {
		t.Fatalf("counts = %+v", counts)
	}
}

func TestABandLeavesTheCountsOutWhenTheOutputHasNone(t *testing.T) {
	s := &script{output: map[string]string{"go build ./...": ""}}
	got, err := Bands(context.Background(), s, t.TempDir(), Plan{Fast: []string{"go build ./..."}})
	if err != nil {
		t.Fatal(err)
	}
	if got.Commands[0].Tests != nil {
		t.Fatalf("counts = %+v, want none", got.Commands[0].Tests)
	}
}
