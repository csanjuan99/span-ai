package qa

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// The results one test case can carry.
const (
	CasePass = "pass"
	CaseFail = "fail"
	CaseSkip = "skip"
)

const (
	// caseOutputLines and caseLineWidth bound what one case keeps of its own output.
	caseOutputLines = 40
	caseLineWidth   = 400
)

// TestCase is one test a check ran, read from what the check printed.
type TestCase struct {
	Name      string `json:"name"`
	Package   string `json:"package,omitempty"`
	Result    string `json:"result"`
	ElapsedMS int64  `json:"elapsedMs"`
	// Path is the test file relative to the workspace, File its base name, and Line where the
	// case is: the failing assertion when it printed one, else the function declaration.
	Path string `json:"path,omitempty"`
	File string `json:"file,omitempty"`
	Line int    `json:"line,omitempty"`
	// Message is the first line of output that names why the case failed or was skipped.
	Message string `json:"message,omitempty"`
	// Output is the case's own output, bounded.
	Output []string `json:"output,omitempty"`
	// New is set when the run's patch added this test.
	New bool `json:"new,omitempty"`
}

// goEvent is one line of `go test -json`.
type goEvent struct {
	Action  string  `json:"Action"`
	Package string  `json:"Package"`
	Test    string  `json:"Test"`
	Elapsed float64 `json:"Elapsed"`
	Output  string  `json:"Output"`
}

var (
	runLine    = regexp.MustCompile(`^=== (?:RUN|CONT|PAUSE)\s+(\S+)`)
	resultLine = regexp.MustCompile(`^\s*--- (PASS|FAIL|SKIP): (\S+) \(([0-9.]+)s\)`)
	pkgLine    = regexp.MustCompile(`^(?:ok|FAIL|---\s+FAIL)\s+(\S+)\s`)
	assertLine = regexp.MustCompile(`^\s*([A-Za-z0-9_.\-/]+_test\.go):(\d+):\s*(.*)$`)
)

// ParseGoTests reads the cases a go test run printed, from `go test -json` or from `go test -v`.
// Only top-level tests are cases, as the counts are; the output of a subtest joins its parent's.
// ok is false when the text holds no case, which is what a run without -v or -json prints.
func ParseGoTests(text string) ([]TestCase, bool) {
	if cases, ok := fromJSON(text); ok {
		return cases, true
	}
	return fromVerbose(text)
}

type caseBuilder struct {
	order []string
	byKey map[string]*TestCase
}

func newBuilder() *caseBuilder { return &caseBuilder{byKey: map[string]*TestCase{}} }

func (b *caseBuilder) get(pkg, name string) *TestCase {
	key := pkg + "\x00" + name
	if c, ok := b.byKey[key]; ok {
		return c
	}
	c := &TestCase{Name: name, Package: pkg}
	b.byKey[key] = c
	b.order = append(b.order, key)
	return c
}

func (b *caseBuilder) done() ([]TestCase, bool) {
	out := make([]TestCase, 0, len(b.order))
	for _, key := range b.order {
		c := b.byKey[key]
		if c.Result == "" {
			continue
		}
		finish(c)
		out = append(out, *c)
	}
	return out, len(out) > 0
}

// top is the top-level test a name belongs to: "TestA/sub" is part of "TestA".
func top(name string) string {
	if i := strings.IndexByte(name, '/'); i > 0 {
		return name[:i]
	}
	return name
}

func (b *caseBuilder) addOutput(pkg, name, line string) {
	c := b.get(pkg, top(name))
	line = strings.TrimRight(line, "\r\n")
	if strings.HasPrefix(line, "=== ") {
		return
	}
	if len(c.Output) < caseOutputLines {
		if len(line) > caseLineWidth {
			line = line[:caseLineWidth]
		}
		c.Output = append(c.Output, line)
	}
}

func fromJSON(text string) ([]TestCase, bool) {
	b := newBuilder()
	for _, raw := range strings.Split(text, "\n") {
		raw = strings.TrimSpace(raw)
		if !strings.HasPrefix(raw, "{") {
			continue
		}
		var ev goEvent
		if err := json.Unmarshal([]byte(raw), &ev); err != nil || ev.Test == "" {
			continue
		}
		switch ev.Action {
		case "output":
			b.addOutput(ev.Package, ev.Test, ev.Output)
		case "pass", "fail", "skip":
			if strings.Contains(ev.Test, "/") {
				continue
			}
			c := b.get(ev.Package, ev.Test)
			c.Result = map[string]string{"pass": CasePass, "fail": CaseFail, "skip": CaseSkip}[ev.Action]
			c.ElapsedMS = int64(ev.Elapsed * 1000)
		}
	}
	return b.done()
}

func fromVerbose(text string) ([]TestCase, bool) {
	b := newBuilder()
	current := ""
	var pending []string
	for _, line := range strings.Split(text, "\n") {
		if m := runLine.FindStringSubmatch(line); m != nil {
			current = top(m[1])
			b.get("", current)
			continue
		}
		if m := resultLine.FindStringSubmatch(line); m != nil && !strings.HasPrefix(line, " ") {
			c := b.get("", m[2])
			c.Result = map[string]string{"PASS": CasePass, "FAIL": CaseFail, "SKIP": CaseSkip}[m[1]]
			if secs, err := strconv.ParseFloat(m[3], 64); err == nil {
				c.ElapsedMS = int64(secs * 1000)
			}
			pending = append(pending, c.Name)
			continue
		}
		if m := pkgLine.FindStringSubmatch(line); m != nil {
			for _, name := range pending {
				if c, ok := b.byKey["\x00"+name]; ok {
					c.Package = m[1]
				}
			}
			pending = nil
			continue
		}
		if current != "" && line != "" && !strings.HasPrefix(line, "--- ") {
			b.addOutput("", current, line)
		}
	}
	// A case stored under an empty package key is renamed once its package is known; the
	// builder key is only the lookup, so nothing else changes.
	return b.done()
}

// finish sets Message, File and Line from the case's own output.
func finish(c *TestCase) {
	for _, line := range c.Output {
		if m := assertLine.FindStringSubmatch(line); m != nil {
			c.File = filepath.Base(m[1])
			c.Path = m[1]
			if n, err := strconv.Atoi(m[2]); err == nil {
				c.Line = n
			}
			c.Message = strings.TrimSpace(m[3])
			return
		}
	}
	for _, line := range c.Output {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "--- ") || strings.HasPrefix(trim, "=== ") || trim == "FAIL" || trim == "PASS" {
			continue
		}
		if c.Result != CasePass {
			c.Message = trim
			return
		}
	}
}

var funcDecl = regexp.MustCompile(`^func (Test\w+|Example\w*|Fuzz\w+)\(`)

// PlaceTests fills Path, File and Line for the cases of a Go workspace. The package's folder is its
// import path without the module's own path; a case whose assertion already named a line keeps
// that line and gains the folder. Cases it cannot place are left as they are.
func PlaceTests(workspace string, cases []TestCase) []TestCase {
	module := goModule(workspace)
	if module == "" {
		return cases
	}
	out := make([]TestCase, len(cases))
	copy(out, cases)
	cache := map[string]map[string][2]string{}
	for i := range out {
		c := &out[i]
		rel, ok := packageDir(module, c.Package)
		if !ok {
			continue
		}
		funcs, seen := cache[rel]
		if !seen {
			funcs = declarations(filepath.Join(workspace, filepath.FromSlash(rel)))
			cache[rel] = funcs
		}
		if c.File != "" && c.Line > 0 {
			c.Path = filepath.ToSlash(filepath.Join(rel, c.File))
			continue
		}
		if where, ok := funcs[c.Name]; ok {
			c.File = filepath.Base(where[0])
			c.Path = filepath.ToSlash(filepath.Join(rel, where[0]))
			if n, err := strconv.Atoi(where[1]); err == nil {
				c.Line = n
			}
		}
	}
	return out
}

func goModule(workspace string) string {
	file, err := os.Open(filepath.Join(workspace, "go.mod"))
	if err != nil {
		return ""
	}
	defer file.Close()
	scan := bufio.NewScanner(file)
	for scan.Scan() {
		if rest, ok := strings.CutPrefix(strings.TrimSpace(scan.Text()), "module "); ok {
			return strings.Trim(strings.TrimSpace(rest), `"`)
		}
	}
	return ""
}

func packageDir(module, pkg string) (string, bool) {
	if pkg == module {
		return ".", true
	}
	rest, ok := strings.CutPrefix(pkg, module+"/")
	if !ok || strings.Contains(rest, "..") {
		return "", false
	}
	return rest, true
}

// declarations maps a test name to its file and line, for the *_test.go files of one folder.
func declarations(dir string) map[string][2]string {
	out := map[string][2]string{}
	files, err := filepath.Glob(filepath.Join(dir, "*_test.go"))
	if err != nil {
		return out
	}
	sort.Strings(files)
	for _, path := range files {
		file, err := os.Open(path)
		if err != nil {
			continue
		}
		scan := bufio.NewScanner(file)
		scan.Buffer(make([]byte, 0, 64*1024), 1<<20)
		for n := 1; scan.Scan(); n++ {
			if m := funcDecl.FindStringSubmatch(scan.Text()); m != nil {
				if _, taken := out[m[1]]; !taken {
					out[m[1]] = [2]string{filepath.Base(path), strconv.Itoa(n)}
				}
			}
		}
		file.Close()
	}
	return out
}

var addedTest = regexp.MustCompile(`^\+func (Test\w+|Example\w*|Fuzz\w+)\(`)

// AddedTests names the tests a patch adds, so a case can say it is new in this run.
func AddedTests(patch string) map[string]bool {
	out := map[string]bool{}
	for _, line := range strings.Split(patch, "\n") {
		if m := addedTest.FindStringSubmatch(line); m != nil {
			out[m[1]] = true
		}
	}
	return out
}

// MarkNew flags the cases the patch added.
func MarkNew(cases []TestCase, added map[string]bool) []TestCase {
	for i := range cases {
		cases[i].New = added[cases[i].Name]
	}
	return cases
}
