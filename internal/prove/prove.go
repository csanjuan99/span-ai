// Package prove runs the commands a project declared and classifies the result.
// A line with shell punctuation is not executed.
package prove

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"span/internal/statement"
)

// ErrProse means the configured line is not an argv command.
var ErrProse = errors.New("not a command")

// Output is what a check produced: the exit code and the captured text.
// A non-nil error from Run means the command never started.
type Output struct {
	ExitCode int
	Text     string
	// Truncated means the start of the output was dropped to stay under a size cap.
	Truncated bool
}

// Checker starts a command.
type Checker interface {
	Run(ctx context.Context, command, dir string) (Output, error)
}

// Report is the band outcome the engine turns into a verdict.
type Report struct {
	Outcome     string // advance, hold, iterate
	Reason      string
	Commands    []statement.Command
	Fingerprint string
}

// Plan is the ordered work of one prove pass. Focused is the story's own tests, Fast is
// the wide suite, and Acceptance is the extra band.
type Plan struct {
	Focused    []string
	Fast       []string
	Acceptance []string
	// Excuse, when set, lets a command that already failed the same way on the base
	// commit report without blocking. nil keeps every failure blocking.
	Excuse Excuse
	// WideBlocks makes a wide failure that Excuse does not excuse fatal even when a focused
	// band exists. It is set when the focused band was narrowed from the diff and a baseline
	// says which wide failures are new: a new one is this change's, wherever it shows up.
	// false keeps the wide band report-only.
	WideBlocks bool
	// FocusLabel, when set, names the focused band in its failure fingerprints instead of
	// the focused command. A band narrowed from the diff names different packages after
	// each repair; the same failure must still read as the same one.
	FocusLabel string
}

// Runs tells whether a focused band is configured. When it is, the wide band reports
// instead of deciding: a failure outside this story's files is not this story's to fix.
func (p Plan) Runs() bool { return len(p.Focused) > 0 }

// Bands runs focused, then fast, then acceptance. A failure in a blocking band stops the
// pass; the wide band only reports when a focused band exists.
func Bands(ctx context.Context, check Checker, dir string, plan Plan) (Report, error) {
	rep, err := bands(ctx, check, dir, plan)
	if err != nil || rep.Outcome != statement.Advance {
		return rep, err
	}
	switch n := countPreexisting(rep.Commands); {
	case n == 1:
		rep.Reason += "; 1 failure already present on the base commit was not counted"
	case n > 1:
		rep.Reason += fmt.Sprintf("; %d failures already present on the base commit were not counted", n)
	}
	return rep, nil
}

func countPreexisting(commands []statement.Command) int {
	n := 0
	for _, item := range commands {
		if item.Preexisting {
			n++
		}
	}
	return n
}

func bands(ctx context.Context, check Checker, dir string, plan Plan) (Report, error) {
	var collected []statement.Command
	if plan.Runs() {
		focused, err := band(ctx, check, dir, plan.Excuse, "focused", plan.FocusLabel, plan.Focused)
		if err != nil {
			return Report{}, err
		}
		collected = append(collected, focused.Commands...)
		if focused.Outcome != statement.Advance {
			focused.Commands = collected
			return focused, nil
		}
		if len(plan.Fast) == 0 {
			acc, err := band(ctx, check, dir, plan.Excuse, "acceptance", "", plan.Acceptance)
			if err != nil {
				return Report{}, err
			}
			collected = append(collected, acc.Commands...)
			acc.Commands = collected
			if acc.Outcome == statement.Advance {
				acc.Reason = "focused passed"
			}
			return acc, nil
		}
		wide, err := band(ctx, check, dir, plan.Excuse, "wide", "", plan.Fast)
		if err != nil {
			return Report{}, err
		}
		collected = append(collected, wide.Commands...)
		if wide.Outcome != statement.Advance && plan.WideBlocks {
			// Not on the base commit, so this change caused it: it decides like a focused one.
			wide.Commands = collected
			return wide, nil
		}
		if wide.Outcome != statement.Advance {
			// Reported, not fatal: the reason names it so the page and the log can be read.
			return Report{Outcome: statement.Advance, Reason: "focused passed; " + wide.Reason + " outside this story", Commands: collected}, nil
		}
		acc, err := band(ctx, check, dir, plan.Excuse, "acceptance", "", plan.Acceptance)
		if err != nil {
			return Report{}, err
		}
		collected = append(collected, acc.Commands...)
		acc.Commands = collected
		if acc.Outcome == statement.Advance {
			acc.Reason = "focused, wide, and acceptance passed"
		}
		return acc, nil
	}
	fastReport, err := band(ctx, check, dir, plan.Excuse, "fast", "", plan.Fast)
	if err != nil {
		return Report{}, err
	}
	if fastReport.Outcome != statement.Advance {
		return fastReport, nil
	}
	acc, err := band(ctx, check, dir, plan.Excuse, "acceptance", "", plan.Acceptance)
	if err != nil {
		return Report{}, err
	}
	acc.Commands = append(fastReport.Commands, acc.Commands...)
	if acc.Outcome == statement.Advance && fastReport.Reason != "" {
		acc.Reason = "fast and acceptance passed"
	}
	return acc, nil
}

// band runs one band in order. label, when set, stands for each command in the fingerprint.
func band(ctx context.Context, check Checker, dir string, excuse Excuse, name, label string, commands []string) (Report, error) {
	rep := Report{Outcome: statement.Advance, Reason: name + " passed"}
	if len(commands) == 0 {
		rep.Reason = name + " had no commands"
		return rep, nil
	}
	for _, command := range commands {
		if Prose(command) {
			rep.Outcome = statement.Hold
			rep.Reason = fmt.Sprintf("%s is not a command", command)
			rep.Commands = append(rep.Commands, statement.Command{Command: command, Band: name, Error: ErrProse.Error()})
			return rep, nil
		}
		began := time.Now()
		code, err := check.Run(ctx, command, dir)
		item := statement.Command{Command: command, ExitCode: code.ExitCode, Band: name,
			Output: code.Text, DurationMS: time.Since(began).Milliseconds()}
		if counts, ok := CountTests(code.Text); ok {
			item.Tests = &counts
		}
		if err != nil {
			item.Error = err.Error()
			rep.Commands = append(rep.Commands, item)
			rep.Outcome = statement.Hold
			rep.Reason = fmt.Sprintf("%s did not run: %s", command, err.Error())
			return rep, nil
		}
		rep.Commands = append(rep.Commands, item)
		if code.ExitCode != 0 {
			named := command
			if label != "" {
				named = label
			}
			item.Fingerprint = Fingerprint(named, code.Text)
			if excuse != nil && excuse(command, code) {
				// Already failing this way on the base commit: kept and shown, not counted.
				item.Preexisting = true
				rep.Commands[len(rep.Commands)-1] = item
				continue
			}
			rep.Commands[len(rep.Commands)-1] = item
			rep.Fingerprint = item.Fingerprint
			rep.Outcome = "iterate"
			rep.Reason = fmt.Sprintf("%s exited %d", command, code.ExitCode)
			return rep, nil
		}
	}
	return rep, nil
}

// Prose reports shell punctuation. Those lines are never executed.
func Prose(command string) bool {
	return strings.ContainsAny(command, "()<>;`|&")
}

// markers are the words that name what broke in a failing run. They are matched
// case-insensitively against the kept output, with no model in the path.
var markers = []string{
	"fail", "error", "✗", "×", "not ok", "panic", "assert",
	"expected", "received", "timeout", "timed out", "segmentation", "econnrefused",
}

// Fingerprint is the stable identity of one failure: the command plus the lines
// that name what broke. The repair loop compares it across iterations, so the
// same failure twice on a different commit stops instead of spending another run.
func Fingerprint(command, output string) string {
	parts := []string{command}
	count := 0
	for _, line := range strings.Split(output, "\n") {
		clean := strings.TrimSpace(line)
		if clean == "" {
			continue
		}
		low := strings.ToLower(clean)
		for _, marker := range markers {
			if strings.Contains(low, marker) {
				parts = append(parts, clean)
				count++
				break
			}
		}
		if count >= 12 {
			break
		}
	}
	return statement.Signature(parts...)
}

// focusable maps a wide command to the way that runner takes explicit test files.
// The runner has to be told which files; the file list already exists, the syntax is
// the only thing Span does not know, so this is the one small table it needs.
var focusable = []struct {
	prefix string
	tail   string
}{
	{"npm test", " -- "},
	{"pnpm test", " -- "},
	{"yarn test", " -- "},
	{"npx vitest", " "},
	{"vitest", " "},
	{"npx jest", " "},
	{"jest", " "},
	{"pytest", " "},
	{"python -m pytest", " "},
	{"npx playwright test", " "},
}

// DeriveFocus turns the wide command into a band scoped to these files, so a project
// that never configured one still gets its story's tests instead of the whole repo.
// nil means the command shape is not known and the wide band keeps deciding.
func DeriveFocus(commands, files []string) []string {
	if len(files) == 0 {
		return nil
	}
	for _, command := range commands {
		clean := strings.TrimSpace(command)
		if clean == "" || Prose(clean) {
			continue
		}
		for _, row := range focusable {
			switch {
			case clean == row.prefix:
				return []string{clean + row.tail + strings.Join(files, " ")}
			case strings.HasPrefix(clean, row.prefix+" "):
				return []string{clean + row.tail + strings.Join(files, " ")}
			case strings.HasPrefix(clean, "npm run "), strings.HasPrefix(clean, "pnpm run "), strings.HasPrefix(clean, "yarn run "):
				return []string{clean + " -- " + strings.Join(files, " ")}
			}
		}
	}
	return nil
}
