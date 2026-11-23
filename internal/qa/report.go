package qa

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Outcomes are the per-test results a report named. An empty Source means no
// report was found, and the gate falls back to the exit code alone.
type Outcomes struct {
	Source string    // junit or vitest-json
	Tests  []Outcome // one entry per test the report named
}

// Outcome is one test the report ran.
type Outcome struct {
	Name   string
	Failed bool
}

// Known reports whether a report was read. Without one, no acceptance id can be
// attributed to a pass or a failure.
func (o Outcomes) Known() bool { return o.Source != "" }

// forID answers whether the report named this criterion. A failure wins over a
// pass when several tests mention the same id.
func (o Outcomes) forID(id string) (failed bool, ok bool) {
	for _, test := range o.Tests {
		if !mentions(test.Name, id) {
			continue
		}
		if test.Failed {
			return true, true
		}
		ok = true
	}
	return false, ok
}

// DefaultReports are looked for when qa.report is not set. A project that
// configures its reporter writes where these paths already point.
var DefaultReports = []string{
	"test-results/junit.xml",
	"junit.xml",
	"test-results/vitest.json",
	"test-results/results.json",
	"test-results.json",
	"report.json",
}

// Find returns the newest report matching pattern, relative to root. A report
// older than since is ignored: a stale report describes a different run.
func Find(root, pattern string, since time.Time) (Outcomes, string, error) {
	patterns := DefaultReports
	if strings.TrimSpace(pattern) != "" {
		patterns = []string{strings.TrimSpace(pattern)}
	}
	for _, candidate := range patterns {
		if !filepath.IsAbs(candidate) {
			candidate = filepath.Join(root, candidate)
		}
		matches, err := filepath.Glob(candidate)
		if err != nil {
			return Outcomes{}, "", err
		}
		for _, path := range matches {
			info, err := os.Stat(path)
			if err != nil || info.IsDir() {
				continue
			}
			if !since.IsZero() && info.ModTime().Before(since) {
				continue
			}
			out, err := Parse(path)
			if err != nil {
				continue
			}
			if out.Known() {
				return out, path, nil
			}
		}
	}
	return Outcomes{}, "", nil
}

// Parse reads a JUnit XML or a Vitest/Jest JSON report.
func Parse(path string) (Outcomes, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Outcomes{}, err
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".xml":
		return parseJUnit(raw)
	case ".json":
		return parseJSON(raw)
	default:
		return Outcomes{}, fmt.Errorf("report %s is not xml or json", path)
	}
}

type junitRoot struct {
	XMLName xml.Name     `xml:"testsuites"`
	Cases   []junitCase  `xml:"testcase"`
	Suites  []junitSuite `xml:"testsuite"`
}

type junitSuite struct {
	Name  string      `xml:"name,attr"`
	Cases []junitCase `xml:"testcase"`
}

type junitCase struct {
	Class   string    `xml:"classname,attr"`
	Name    string    `xml:"name,attr"`
	Failure *junitBad `xml:"failure"`
	Error   *junitBad `xml:"error"`
	Skipped *junitBad `xml:"skipped"`
}

type junitBad struct{}

func parseJUnit(raw []byte) (Outcomes, error) {
	var root junitRoot
	if err := xml.Unmarshal(raw, &root); err != nil {
		return Outcomes{}, err
	}
	out := Outcomes{Source: "junit"}
	add := func(c junitCase) {
		name := strings.TrimSpace(strings.Join([]string{c.Class, c.Name}, " "))
		if name == "" {
			return
		}
		out.Tests = append(out.Tests, Outcome{Name: name, Failed: c.Failure != nil || c.Error != nil})
	}
	for _, c := range root.Cases {
		add(c)
	}
	for _, suite := range root.Suites {
		for _, c := range suite.Cases {
			if c.Class == "" {
				c.Class = suite.Name
			}
			add(c)
		}
	}
	if len(out.Tests) == 0 {
		return Outcomes{}, fmt.Errorf("junit report has no test cases")
	}
	return out, nil
}

type jsonReport struct {
	TestResults []struct {
		Name    string `json:"name"`
		Results []struct {
			FullName string `json:"fullName"`
			Title    string `json:"title"`
			Status   string `json:"status"`
		} `json:"assertionResults"`
	} `json:"testResults"`
}

func parseJSON(raw []byte) (Outcomes, error) {
	var report jsonReport
	if err := json.Unmarshal(raw, &report); err != nil {
		return Outcomes{}, err
	}
	out := Outcomes{Source: "vitest-json"}
	for _, file := range report.TestResults {
		for _, result := range file.Results {
			name := strings.TrimSpace(result.FullName)
			if name == "" {
				name = strings.TrimSpace(file.Name + " " + result.Title)
			}
			if name == "" {
				continue
			}
			out.Tests = append(out.Tests, Outcome{
				Name:   name,
				Failed: result.Status == "failed",
			})
		}
	}
	if len(out.Tests) == 0 {
		return Outcomes{}, fmt.Errorf("json report has no assertion results")
	}
	return out, nil
}
