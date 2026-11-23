// Package branch names the only three branches Span is allowed to create.
// feat and fix start from dev. hotfix starts from master.
// The suffix is a linked story key plus a slug, or a descriptive slug.
package branch

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	Feat   = "feat"
	Fix    = "fix"
	Hotfix = "hotfix"
	Dev    = "dev"
	Master = "master"
)

var (
	keyPattern  = regexp.MustCompile(`\b[A-Z][A-Z0-9]+-\d+\b`)
	exactKey    = regexp.MustCompile(`^[A-Z][A-Z0-9]+-\d+$`)
	branchMatch = regexp.MustCompile(`^(feat|fix|hotfix)/[A-Za-z0-9][A-Za-z0-9._-]*$`)
)

// Plan is the branch for one story.
type Plan struct {
	Kind    string
	Base    string
	Name    string
	Feature string
}

// Derive infers the prefix from the kind and the base, and the suffix from a
// linked story key when one is present.
func Derive(story, issue, kind, base string) (Plan, error) {
	story = strings.TrimSpace(story)
	if story == "" {
		return Plan{}, fmt.Errorf("story is empty")
	}
	kind = strings.ToLower(strings.TrimSpace(kind))
	base = strings.TrimSpace(base)
	if kind == "" {
		kind = inferKind(story, base)
	}
	switch kind {
	case Feat, Fix:
		if base == "" {
			base = Dev
		}
		if base != Dev {
			return Plan{}, fmt.Errorf("%s starts from dev", kind)
		}
	case Hotfix:
		if base == "" {
			base = Master
		}
		if base != Master {
			return Plan{}, fmt.Errorf("hotfix starts from master")
		}
	default:
		return Plan{}, fmt.Errorf("branch kind %q is not feat, fix, or hotfix", kind)
	}
	key := strings.ToUpper(strings.TrimSpace(issue))
	if key == "" {
		key = keyPattern.FindString(story)
	}
	if key != "" && !exactKey.MatchString(key) {
		return Plan{}, fmt.Errorf("issue %q is not a story key", issue)
	}
	title := slug(titleOf(story))
	if key != "" {
		title = strings.Trim(strings.TrimPrefix(title, slug(key)+"-"), "-")
	}
	// The budget is for the title, after the story key is out of the way: capping the
	// joined string wasted the prefix and cut the last word in half.
	title = clip(title, maxSlug)
	if title == "" {
		title = "story"
	}
	suffix := title
	if key != "" {
		suffix = key + "-" + title
	}
	name := kind + "/" + suffix
	if !Valid(name, base) {
		return Plan{}, fmt.Errorf("branch %q is not a span branch", name)
	}
	return Plan{Kind: kind, Base: base, Name: name, Feature: strings.ToLower(suffix)}, nil
}

// Valid reports whether name is one of the three prefixes on the matching base.
func Valid(name, base string) bool {
	if !branchMatch.MatchString(name) {
		return false
	}
	kind := strings.SplitN(name, "/", 2)[0]
	switch kind {
	case Feat, Fix:
		return base == Dev
	case Hotfix:
		return base == Master
	default:
		return false
	}
}

func inferKind(story, base string) string {
	if base == Master {
		return Hotfix
	}
	line := strings.ToLower(firstLine(story))
	switch {
	case strings.HasPrefix(line, "hotfix:") || strings.HasPrefix(line, "hotfix "):
		return Hotfix
	case strings.HasPrefix(line, "fix:") || strings.HasPrefix(line, "fix ") || strings.HasPrefix(line, "bug:") || strings.HasPrefix(line, "bug "):
		return Fix
	default:
		return Feat
	}
}

func titleOf(story string) string {
	line := firstLine(story)
	lower := strings.ToLower(line)
	for _, prefix := range []string{"hotfix:", "feat:", "fix:", "bug:"} {
		if strings.HasPrefix(lower, prefix) {
			return strings.TrimSpace(line[len(prefix):])
		}
	}
	return line
}

func firstLine(story string) string {
	line, _, _ := strings.Cut(story, "\n")
	return strings.TrimSpace(line)
}

// maxSlug is how long the title part of a branch may be.
const maxSlug = 48

// fold turns the accented letters a Spanish story actually uses into ASCII. A branch
// name has to stay typeable, and it has to pass the same rule as a feature name.
var fold = map[rune]rune{
	'á': 'a', 'à': 'a', 'ä': 'a', 'â': 'a', 'ã': 'a', 'å': 'a',
	'é': 'e', 'è': 'e', 'ë': 'e', 'ê': 'e',
	'í': 'i', 'ì': 'i', 'ï': 'i', 'î': 'i',
	'ó': 'o', 'ò': 'o', 'ö': 'o', 'ô': 'o', 'õ': 'o',
	'ú': 'u', 'ù': 'u', 'ü': 'u', 'û': 'u',
	'ñ': 'n', 'ç': 'c',
}

// slug is ASCII by construction: an accented letter loses its accent, and anything that
// is still not a letter or a digit becomes a dash.
func slug(s string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(s) {
		if mapped, ok := fold[r]; ok {
			r = mapped
		}
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			dash = false
			continue
		}
		if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

// clip cuts to n runes, never through the middle of one, and never leaves a dash.
func clip(s string, n int) string {
	runes := []rune(s)
	if len(runes) > n {
		runes = runes[:n]
	}
	return strings.Trim(string(runes), "-")
}
