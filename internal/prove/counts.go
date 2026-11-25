package prove

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"

	"span/internal/statement"
)

var (
	// vitest and jest print the tests on one line: "Tests  2 failed | 338 passed (340)" or
	// "Tests:  2 failed, 41 passed, 43 total".
	jsSummary = regexp.MustCompile(`(?m)^\s*Tests:?\s+(.*)$`)
	// pytest closes with "== 2 failed, 41 passed, 1 skipped in 3.2s ==".
	pySummary = regexp.MustCompile(`(?m)^=+ (.*\b(?:passed|failed)\b.*) in [0-9.]+s.*=+$`)
	// One "<n> <word>" pair inside a summary line.
	pair = regexp.MustCompile(`(\d+) (passed|failed|skipped)`)
)

// CountTests reads how many tests passed, failed and were skipped from what a check printed.
// It understands the summary lines of vitest, jest and pytest, the events of go test -json, and the "--- PASS" lines of
// go test -v (top-level tests only, so a subtest is not counted twice). ok is false when the
// output holds no count, which is the common case for a build or a linter: nothing is guessed.
func CountTests(text string) (statement.TestCounts, bool) {
	if counts, ok := fromGoJSON(text); ok {
		return counts, true
	}
	if found := lastMatch(jsSummary, text); found != "" {
		return fromPairs(found)
	}
	if found := lastMatch(pySummary, text); found != "" {
		return fromPairs(found)
	}
	return fromGoVerbose(text)
}

func lastMatch(re *regexp.Regexp, text string) string {
	all := re.FindAllStringSubmatch(text, -1)
	if len(all) == 0 {
		return ""
	}
	return all[len(all)-1][1]
}

func fromPairs(summary string) (statement.TestCounts, bool) {
	var counts statement.TestCounts
	seen := false
	for _, match := range pair.FindAllStringSubmatch(summary, -1) {
		n, err := strconv.Atoi(match[1])
		if err != nil {
			continue
		}
		seen = true
		switch match[2] {
		case "passed":
			counts.Passed = n
		case "failed":
			counts.Failed = n
		case "skipped":
			counts.Skipped = n
		}
	}
	return counts, seen
}

// fromGoVerbose counts "--- PASS:" lines that start the line. Plain go test prints no pass
// lines, so its output has no count; a lone "--- FAIL" would be a count of one failure and
// zero passes, which is not what the run did.
func fromGoVerbose(text string) (statement.TestCounts, bool) {
	var counts statement.TestCounts
	ran := false
	for _, line := range strings.Split(text, "\n") {
		switch {
		case strings.HasPrefix(line, "--- PASS:"):
			counts.Passed++
			ran = true
		case strings.HasPrefix(line, "--- FAIL:"):
			counts.Failed++
		case strings.HasPrefix(line, "--- SKIP:"):
			counts.Skipped++
			ran = true
		}
	}
	return counts, ran
}

// goEvent is the part of a `go test -json` line that says how a test ended.
type goEvent struct {
	Action string `json:"Action"`
	Test   string `json:"Test"`
}

// fromGoJSON counts the final pass, fail and skip events of go test -json for top-level tests.
// A subtest is a name with a slash, and counting it too would count one test twice. Lines that
// are not JSON are ignored, so output mixed with a build log still reads.
func fromGoJSON(text string) (statement.TestCounts, bool) {
	var counts statement.TestCounts
	ran := false
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "{") {
			continue
		}
		var event goEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil || event.Test == "" || strings.Contains(event.Test, "/") {
			continue
		}
		switch event.Action {
		case "pass":
			counts.Passed++
			ran = true
		case "fail":
			counts.Failed++
			ran = true
		case "skip":
			counts.Skipped++
			ran = true
		}
	}
	return counts, ran
}
