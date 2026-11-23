package adapt

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// scanLimit bounds the staged diff read for secret shapes. A larger one is not scanned past
// the cut, which the commit error cannot say; the path rules still cover the whole change.
const scanLimit = 8 << 20

var (
	// junkPath is build output and caches. They never belong in history, so the commit
	// leaves them in the working tree without staging them and does not stop.
	junkPath = regexp.MustCompile(`(^|/)(__pycache__|node_modules)/|\.(pyc|pyo)$|(^|/)\.DS_Store$`)
	// secretPath is a file that usually holds a secret. It stops the commit.
	secretPath = regexp.MustCompile(`(^|/)\.env(\.[^/]+)?$|\.(pem|key|p12|pfx)$|(^|/)id_(rsa|dsa|ecdsa|ed25519)$`)
	// templatePath is a secret-looking file that is a shareable template.
	templatePath = regexp.MustCompile(`\.(example|sample|template|dist)$`)
)

var secretShapes = []struct {
	why     string
	pattern *regexp.Regexp
}{
	{"an AWS access key id", regexp.MustCompile(`AKIA[0-9A-Z]{16}`)},
	{"a private key", regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----`)},
	{"a GitHub token", regexp.MustCompile(`\bgh[pousr]_[A-Za-z0-9]{30,}`)},
	{"a hard-coded credential", regexp.MustCompile(`(?i)(api[_-]?key|secret|passw(or)?d|token)\s*[:=]\s*["'][^"'\s]{20,}["']`)},
}

// finding is a staged file the commit will not take, and why. It never carries the match.
type finding struct {
	path string
	why  string
}

// allowed reports whether a person listed this path under commit.allow, as the exact path or
// as a folder written with a trailing slash.
func allowed(path string, allow []string) bool {
	for _, entry := range allow {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if path == entry || (strings.HasSuffix(entry, "/") && strings.HasPrefix(path, entry)) {
			return true
		}
	}
	return false
}

// byPath names the staged files that look like secrets from their name alone.
func byPath(paths, allow []string) []finding {
	var out []finding
	for _, path := range paths {
		if allowed(path, allow) || junkPath.MatchString(path) {
			continue
		}
		if secretPath.MatchString(path) && !templatePath.MatchString(path) {
			out = append(out, finding{path, "its name says it holds a secret"})
		}
	}
	return out
}

// byContent reads a staged diff and names the files whose added lines look like a secret.
// One finding per file, and never the matching text.
func byContent(diff string, allow []string, skip map[string]bool) []finding {
	var out []finding
	var path string
	done := map[string]bool{}
	for _, line := range strings.Split(diff, "\n") {
		switch {
		case strings.HasPrefix(line, "+++ b/"):
			path = strings.TrimPrefix(line, "+++ b/")
		case strings.HasPrefix(line, "+++ "):
			path = ""
		case strings.HasPrefix(line, "+") && path != "" && !done[path] && !skip[path] && !allowed(path, allow) && !junkPath.MatchString(path):
			for _, shape := range secretShapes {
				if shape.pattern.MatchString(line) {
					done[path] = true
					out = append(out, finding{path, "an added line looks like " + shape.why})
					break
				}
			}
		}
	}
	return out
}

// guardStaged runs after everything is staged. Junk is unstaged and left on disk. A file that
// looks like a secret stops the commit: the index is cleared and the error names the file and
// the way out, so a person decides instead of the agent's leftovers reaching history.
// It reports whether anything is left to commit.
func (g Git) guardStaged(ctx context.Context) (bool, error) {
	raw, err := g.output(ctx, "diff", "--cached", "--name-only", "-z")
	if err != nil {
		return false, err
	}
	var kept []string
	for _, path := range strings.Split(raw, "\x00") {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		if junkPath.MatchString(path) && !allowed(path, g.Allow) {
			if _, err := g.output(ctx, "reset", "-q", "--", path); err != nil {
				return false, err
			}
			continue
		}
		kept = append(kept, path)
	}
	found := byPath(kept, g.Allow)
	named := map[string]bool{}
	for _, item := range found {
		named[item.path] = true
	}
	diff, _, err := g.bounded(ctx, scanLimit, "diff", "--cached", "-U0", "--no-color", "--no-ext-diff")
	if err != nil {
		return false, err
	}
	found = append(found, byContent(diff, g.Allow, named)...)
	if len(found) > 0 {
		if _, err := g.output(ctx, "reset", "-q"); err != nil {
			return false, err
		}
		return false, blocked(found)
	}
	return len(kept) > 0, nil
}

func blocked(found []finding) error {
	var b strings.Builder
	b.WriteString("commit blocked, nothing was committed:")
	for _, item := range found {
		fmt.Fprintf(&b, " %s (%s);", item.path, item.why)
	}
	b.WriteString(" remove them, or after checking them list the path under commit.allow in span.yaml")
	return fmt.Errorf("%s", b.String())
}
