package prove

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// maxFailureLines bounds what a baseline keeps per command. An output with more failure
// lines than this is never excused, because the part that was cut could hold a new one.
const maxFailureLines = 400

// Baseline is what the base commit already failed, recorded once before the first build.
// A failure that was already there is not this story's, so it reports but never blocks.
type Baseline struct {
	SHA      string                     `json:"sha"`
	Commands map[string]BaselineCommand `json:"commands"`
}

// BaselineCommand is one command as it ran on the base commit. Truncated means the
// output or its failure lines were cut; such an entry never excuses anything.
type BaselineCommand struct {
	ExitCode  int      `json:"exitCode"`
	Lines     []string `json:"lines"`
	Truncated bool     `json:"truncated,omitempty"`
	// BuildFailed means a package did not compile or set up; such an entry never excuses.
	BuildFailed bool `json:"buildFailed,omitempty"`
}

// buildFailed reports a go package that could not run its tests at all. Its compile
// errors carry no marker word, so they cannot be compared, and nothing about it is known.
func buildFailed(text string) bool {
	return strings.Contains(text, "[build failed]") || strings.Contains(text, "[setup failed]")
}

// Excuse reports whether a failing command failed the same way on the base commit.
// A nil Excuse excuses nothing.
type Excuse func(command string, out Output) bool

var (
	ansiCode = regexp.MustCompile(`\x1b\[[0-9;?]*[A-Za-z]`)
	// A whole parenthesised duration, as go test and jest print after a test name.
	parenDuration = regexp.MustCompile(`\(\d+(?:\.\d+)? ?(?:ms|s)\)`)
	// A go test package line: FAIL or ok, a tab, the package path.
	goPackage = regexp.MustCompile(`^(?:FAIL|ok)\s*\t(\S+)`)
	// The timing column at the end of a go test package line.
	goTiming = regexp.MustCompile(`\s+(?:\d+(?:\.\d+)?s|\(cached\))$`)
)

// FailureLines is the sorted multiset of lines that name what broke. Only colors, a
// parenthesised duration, and the timing column of a go test package line are removed;
// every other character, values inside messages included, is kept, so a failure that
// differs by anything else is a new one. Repeats are kept: one more copy of an old
// failure is a new failure.
//
// go test output is read per test and per package. Every line a test prints, between its
// "=== RUN", "=== CONT" or "=== NAME" line and the next one, and the indented lines under
// its "--- FAIL:" line, belong to that test; they are kept, marker word or not, only when
// the test did not pass, prefixed with the test name. A test's lines come before its
// package line, so each kept line is prefixed with the package of the next "FAIL\t<pkg>"
// or "ok\t<pkg>" line, and the same test name failing in a second package is a new line.
// Output from other runners has none of these lines: only lines with a marker word are
// kept, compared as a plain multiset.
func FailureLines(text string) []string {
	lines, _ := failureLines(text)
	return lines
}

var (
	goTestStart   = regexp.MustCompile(`^=== (?:RUN|CONT|NAME)\s+(\S+)`)
	goTestPause   = regexp.MustCompile(`^=== PAUSE\s`)
	goTestVerdict = regexp.MustCompile(`^--- (FAIL|PASS|SKIP):\s+(\S+)`)
)

// goTest is what one test printed and how it ended.
type goTest struct {
	lines  []string
	passed bool
}

func failureLines(text string) ([]string, bool) {
	var kept, loose []string
	truncated := false
	add := func(line string) {
		if len(kept) == maxFailureLines {
			truncated = true
			return
		}
		kept = append(kept, line)
	}
	tests := map[string]*goTest{}
	var order []string
	current := ""
	test := func(name string) *goTest {
		if tests[name] == nil {
			tests[name] = &goTest{}
			order = append(order, name)
		}
		return tests[name]
	}
	flush := func(prefix string) {
		for _, name := range order {
			// A test with no verdict line (a panic, a timeout) did not pass.
			if t := tests[name]; !t.passed {
				for _, line := range t.lines {
					add(prefix + line)
				}
			}
		}
		for _, line := range loose {
			add(prefix + line)
		}
		tests, order, loose, current = map[string]*goTest{}, nil, nil, ""
	}
	for _, raw := range strings.Split(text, "\n") {
		clean := strings.TrimRight(ansiCode.ReplaceAllString(raw, ""), " \t\r")
		trimmed := strings.TrimSpace(clean)
		if trimmed == "" {
			continue
		}
		if m := goPackage.FindStringSubmatch(trimmed); m != nil {
			flush("[" + m[1] + "] ")
			if header := collapse(goTiming.ReplaceAllString(trimmed, "")); marked(header) {
				add(header)
			}
			continue
		}
		line := collapse(parenDuration.ReplaceAllString(trimmed, ""))
		switch m := goTestStart.FindStringSubmatch(trimmed); {
		case m != nil:
			current = m[1]
			test(current)
			continue
		case goTestPause.MatchString(trimmed):
			current = ""
			continue
		}
		if m := goTestVerdict.FindStringSubmatch(trimmed); m != nil {
			current = m[2]
			t := test(current)
			if m[1] == "FAIL" {
				t.lines = append(t.lines, line)
			} else {
				t.passed = true
			}
			continue
		}
		if trimmed == "FAIL" || trimmed == "PASS" || strings.HasPrefix(trimmed, "ok ") {
			current = ""
		}
		if current != "" {
			t := test(current)
			t.lines = append(t.lines, current+": "+line)
			continue
		}
		if marked(trimmed) {
			loose = append(loose, line)
		}
	}
	flush("")
	sort.Strings(kept)
	if kept == nil {
		kept = []string{}
	}
	return kept, truncated
}

func collapse(line string) string { return strings.Join(strings.Fields(line), " ") }

func marked(line string) bool {
	low := strings.ToLower(line)
	for _, marker := range markers {
		if strings.Contains(low, marker) {
			return true
		}
	}
	return false
}

// Excuses decides whether a failure was already present on the base commit. Only an
// exact command that failed on the base, failing now with no failure line it did not
// already print there as many times, is excused. Anything that cannot be compared,
// including a cut output on either side, is not.
func (b Baseline) Excuses(command string, out Output) (bool, string) {
	if out.ExitCode == 0 {
		return false, "the command passed"
	}
	if out.Truncated {
		return false, "the output was cut; the dropped part cannot be compared"
	}
	old, ok := b.Commands[command]
	if !ok {
		return false, "the base commit has no record of " + command
	}
	if old.ExitCode == 0 {
		return false, command + " passed on the base commit"
	}
	if old.Truncated {
		return false, "the base commit's output was cut; it cannot be compared"
	}
	if buildFailed(out.Text) || old.BuildFailed {
		return false, "a package did not build; its tests did not run, so nothing about it is known"
	}
	current, truncated := failureLines(out.Text)
	if truncated {
		return false, fmt.Sprintf("more than %d failure lines; they cannot all be compared", maxFailureLines)
	}
	if len(current) == 0 {
		return false, "no failure lines to compare"
	}
	known := make(map[string]int, len(old.Lines))
	for _, line := range old.Lines {
		known[line]++
	}
	for _, line := range current {
		if known[line] == 0 {
			return false, "new failure: " + line
		}
		known[line]--
	}
	return true, fmt.Sprintf("all %d failure lines were already present on the base commit", len(current))
}

// Record runs each distinct command once on the base commit. A prose line or a command
// that does not start is skipped and named in the notes: it has no record, so nothing it
// prints later is excused. A cancelled context returns an error and no baseline.
func Record(ctx context.Context, check Checker, dir, sha string, commands []string) (Baseline, []string, error) {
	base := Baseline{SHA: sha, Commands: map[string]BaselineCommand{}}
	var notes []string
	done := map[string]bool{}
	for _, command := range commands {
		command = strings.TrimSpace(command)
		if command == "" || done[command] {
			continue
		}
		done[command] = true
		if err := ctx.Err(); err != nil {
			return Baseline{}, notes, err
		}
		if Prose(command) {
			notes = append(notes, command+" is not a command; not recorded")
			continue
		}
		out, err := check.Run(ctx, command, dir)
		if ctxErr := ctx.Err(); ctxErr != nil {
			return Baseline{}, notes, ctxErr
		}
		if err != nil {
			notes = append(notes, command+" did not run: "+err.Error()+"; not recorded")
			continue
		}
		entry := BaselineCommand{ExitCode: out.ExitCode, Lines: []string{}, Truncated: out.Truncated, BuildFailed: buildFailed(out.Text)}
		if out.ExitCode != 0 {
			lines, cut := failureLines(out.Text)
			entry.Lines = lines
			entry.Truncated = entry.Truncated || cut
		}
		base.Commands[command] = entry
		notes = append(notes, fmt.Sprintf("%s exited %d (%d failing lines)", command, out.ExitCode, len(entry.Lines)))
	}
	return base, notes, nil
}
