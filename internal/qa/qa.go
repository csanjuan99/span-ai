// Package qa checks a story's acceptance lines against test declarations.
package qa

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

const marker = "Acceptance criteria:"

// The three results one criterion can carry.
const (
	Pass       = "pass"
	Fail       = "fail"
	Unverified = "unverified"
)

// Line is one acceptance criterion.
type Line struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

// Check is the result for one criterion.
type Check struct {
	ID     string `json:"id"`
	Result string `json:"result"`
}

// Gate is the acceptance result stored on the run. Verdict describes the
// criteria, not the process: an exit code alone never turns an id into a failure.
type Gate struct {
	Verdict  string  `json:"verdict"`
	Command  string  `json:"command,omitempty"`
	ExitCode int     `json:"exitCode"`
	Evidence string  `json:"evidence,omitempty"`
	Report   string  `json:"report,omitempty"`
	Reason   string  `json:"reason,omitempty"`
	Checks   []Check `json:"checks"`
}

// Body returns the text under the acceptance heading.
func Body(brief string) string {
	i := strings.LastIndex(brief, marker)
	if i < 0 {
		return ""
	}
	return strings.TrimSpace(brief[i+len(marker):])
}

// Lines numbers the non-empty acceptance lines.
func Lines(brief string) []Line {
	body := Body(brief)
	if body == "" {
		return nil
	}
	var out []Line
	for _, line := range strings.Split(body, "\n") {
		text := strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(line), "-*"))
		if text == "" {
			continue
		}
		id := "AC-" + itoa(len(out)+1)
		if number, ok := acceptanceNumber(text); ok {
			id = "AC-" + number
		}
		out = append(out, Line{ID: id, Text: text})
	}
	return out
}

// Install writes criteria the story already stated under the one standard heading.
// A brief that already has that section is left as it is.
func Install(brief string, found []string) string {
	if len(Lines(brief)) > 0 {
		return brief
	}
	var lines []string
	for _, line := range found {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	if len(lines) == 0 {
		return brief
	}
	var b strings.Builder
	b.WriteString(strings.TrimRight(brief, "\n"))
	b.WriteString("\n\nAcceptance criteria:\n")
	for _, line := range lines {
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}

// HasAcceptance reports whether the brief has at least one acceptance line.
func HasAcceptance(brief string) bool {
	return len(Lines(brief)) > 0
}

// Judge scores each id against the evidence that actually exists. A per-test
// report names which criteria ran and failed. Without one, a zero exit passes the
// declared criteria and a non-zero exit makes them unverified, never fail: the
// process failing is not evidence that a named criterion failed.
func Judge(lines []Line, exit int, covered map[string]bool, outcomes Outcomes) Gate {
	gate := Gate{ExitCode: exit, Checks: make([]Check, 0, len(lines)), Evidence: "exit-only"}
	if outcomes.Known() {
		gate.Evidence = outcomes.Source
	}
	pass, fail, unverified, uncovered := 0, 0, 0, 0
	for _, line := range lines {
		result := Unverified
		switch {
		case !covered[line.ID]:
			result = Unverified
			uncovered++
		case outcomes.Known():
			failed, ok := outcomes.forID(line.ID)
			switch {
			case !ok:
				result = Unverified
			case failed:
				result = Fail
			default:
				result = Pass
			}
		case exit == 0:
			result = Pass
		default:
			result = Unverified
		}
		switch result {
		case Pass:
			pass++
		case Fail:
			fail++
		default:
			unverified++
		}
		gate.Checks = append(gate.Checks, Check{ID: line.ID, Result: result})
	}
	switch {
	case fail > 0:
		gate.Verdict = Fail
		gate.Reason = fmt.Sprintf("%d of %d criteria failed", fail, len(lines))
	case !outcomes.Known() && exit != 0:
		gate.Verdict = Unverified
		gate.Reason = fmt.Sprintf("exit %d with no per-test report; none of %d criteria is attributable", exit, len(lines))
	case uncovered > 0:
		gate.Verdict = Unverified
		gate.Reason = fmt.Sprintf("%d of %d criteria have no test declaration", uncovered, len(lines))
	case unverified > 0:
		gate.Verdict = Unverified
		gate.Reason = fmt.Sprintf("%d of %d criteria were declared but the report did not run them", unverified, len(lines))
	default:
		gate.Verdict = Pass
		gate.Reason = fmt.Sprintf("all %d criteria passed", pass)
	}
	return gate
}

// Place is the test declaration that names an acceptance id.
type Place struct {
	File string
	Line int
}

// Case is one acceptance line prepared for a reader.
type Case struct {
	ID     string `json:"id"`
	Text   string `json:"text"`
	Result string `json:"result,omitempty"`
	File   string `json:"file,omitempty"`
	Line   int    `json:"line,omitempty"`
}

// Covered reads test files and reports which ids appear on a declaration line.
func Covered(root string, lines []Line) (map[string]bool, error) {
	places, err := Locate(root, lines)
	found := map[string]bool{}
	for id := range places {
		found[id] = true
	}
	return found, err
}

// Locate returns the first declaration for each id.
func Locate(root string, lines []Line) (map[string]Place, error) {
	found := map[string]Place{}
	if len(lines) == 0 || strings.TrimSpace(root) == "" {
		return found, nil
	}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry == nil {
			return nil
		}
		name := entry.Name()
		if entry.IsDir() {
			if path != root && skipDir(name) {
				return filepath.SkipDir
			}
			return nil
		}
		if !testFile(name) {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			rel = path
		}
		rel = filepath.ToSlash(rel)
		for n, line := range strings.Split(string(raw), "\n") {
			if !declaration(line) {
				continue
			}
			for _, item := range lines {
				if _, seen := found[item.ID]; seen {
					continue
				}
				if mentions(line, item.ID) {
					found[item.ID] = Place{File: rel, Line: n + 1}
				}
			}
		}
		return nil
	})
	return found, err
}

// Present joins acceptance lines, the gate, and the declaration that names each id.
func Present(root string, lines []Line, gate *Gate) ([]Case, error) {
	places, err := Locate(root, lines)
	if err != nil {
		return nil, err
	}
	out := make([]Case, 0, len(lines))
	for _, line := range lines {
		item := Case{ID: line.ID, Text: line.Text}
		if gate != nil {
			item.Result = checkResult(*gate, line.ID)
		}
		if item.Result != "unverified" {
			if place, ok := places[line.ID]; ok {
				item.File = place.File
				item.Line = place.Line
			}
		}
		out = append(out, item)
	}
	return out, nil
}

func checkResult(gate Gate, id string) string {
	for _, check := range gate.Checks {
		if check.ID == id {
			return check.Result
		}
	}
	return ""
}

func skipDir(name string) bool {
	return strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor"
}

func testFile(name string) bool {
	switch {
	case strings.HasSuffix(name, "_test.go"):
		return true
	case strings.HasSuffix(name, "_test.ts"), strings.HasSuffix(name, "_test.tsx"), strings.HasSuffix(name, "_test.js"), strings.HasSuffix(name, "_test.jsx"):
		return true
	case strings.HasSuffix(name, ".spec.ts"), strings.HasSuffix(name, ".spec.tsx"), strings.HasSuffix(name, ".spec.js"), strings.HasSuffix(name, ".spec.jsx"):
		return true
	case strings.HasSuffix(name, ".test.ts"), strings.HasSuffix(name, ".test.tsx"), strings.HasSuffix(name, ".test.js"), strings.HasSuffix(name, ".test.jsx"):
		return true
	case strings.HasPrefix(name, "test_") && strings.HasSuffix(name, ".py"):
		return true
	case strings.HasSuffix(name, "_test.py"):
		return true
	default:
		return false
	}
}

func declaration(line string) bool {
	s := strings.TrimSpace(line)
	if strings.Contains(s, "func Test") || strings.Contains(s, "def test_") {
		return true
	}
	return hasCall(s, "test(") || hasCall(s, "test.each(") || hasCall(s, "it(") || hasCall(s, "it.each(")
}

func hasCall(line, name string) bool {
	rest := line
	for {
		i := strings.Index(rest, name)
		if i < 0 {
			return false
		}
		if i == 0 || !isIdent(rune(rest[i-1])) {
			return true
		}
		rest = rest[i+1:]
	}
}

func isIdent(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

func mentions(line, id string) bool {
	number := strings.TrimPrefix(id, "AC-")
	for _, form := range []string{"AC-" + number, "AC" + number} {
		rest := line
		for {
			i := strings.Index(rest, form)
			if i < 0 {
				break
			}
			after := i + len(form)
			if after >= len(rest) || !unicode.IsDigit(rune(rest[after])) {
				return true
			}
			rest = rest[i+1:]
		}
	}
	return false
}

func acceptanceNumber(text string) (string, bool) {
	for i := 0; i+2 <= len(text); i++ {
		if (text[i] != 'A' && text[i] != 'a') || (text[i+1] != 'C' && text[i+1] != 'c') {
			continue
		}
		if i > 0 && isIdent(rune(text[i-1])) {
			continue
		}
		j := i + 2
		if j < len(text) && (text[j] == '-' || text[j] == ' ') {
			j++
		}
		start := j
		for j < len(text) && text[j] >= '0' && text[j] <= '9' {
			j++
		}
		if j == start || (j < len(text) && isIdent(rune(text[j]))) {
			continue
		}
		return text[start:j], true
	}
	return "", false
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

// Scope is the test files that declare these criteria, unique and sorted. It is the
// story's own test surface: what a check must run to answer for this story alone.
// A criterion with no declaration contributes nothing.
func Scope(root string, lines []Line) ([]string, error) {
	places, err := Locate(root, lines)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool, len(places))
	out := make([]string, 0, len(places))
	for _, place := range places {
		if place.File == "" || seen[place.File] {
			continue
		}
		seen[place.File] = true
		out = append(out, place.File)
	}
	sort.Strings(out)
	return out, nil
}
