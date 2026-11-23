// Package branch names the only three branches Span is allowed to create: feat, fix and
// hotfix. Without an explicit base, feat and fix start from dev and hotfix from master. An
// explicit base (from the run or from span.yaml) is the person's choice and may be any
// valid branch name, so stacked work can start from another feature branch.
// The suffix is a linked story key plus a slug, or a descriptive slug.
package branch

import (
	"fmt"
	"regexp"
	"strings"
)

// The three kinds and the two default bases.
const (
	Feat   = "feat"
	Fix    = "fix"
	Hotfix = "hotfix"
	Dev    = "dev"
	Master = "master"
)

// MissingBase is the base branch a run would start from, or open its change into, that
// exists neither locally nor on origin. Cause is what the lookup saw, such as a failed
// fetch; it is kept for callers and left out of the message a person reads.
type MissingBase struct {
	Name  string
	Cause error
}

func (e MissingBase) Error() string {
	return fmt.Sprintf("base branch %s does not exist on origin; push it or choose another base", e.Name)
}

// Unwrap returns the lookup failure behind the missing base, if any.
func (e MissingBase) Unwrap() error { return e.Cause }

var (
	keyPattern  = regexp.MustCompile(`\b[A-Z][A-Z0-9]+-\d+\b`)
	exactKey    = regexp.MustCompile(`^[A-Z][A-Z0-9]+-\d+$`)
	branchMatch = regexp.MustCompile(`^(feat|fix|hotfix)/[A-Za-z0-9][A-Za-z0-9._-]*$`)
	shaLike     = regexp.MustCompile(`^[0-9a-f]{7,40}$`)
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
	if strings.TrimSpace(base) == "" {
		base = ""
	}
	if kind == "" {
		kind = inferKind(story, base)
	}
	switch kind {
	case Feat, Fix, Hotfix:
	default:
		return Plan{}, fmt.Errorf("branch kind %q is not feat, fix, or hotfix", kind)
	}
	if base == "" {
		base = DefaultBase(kind)
	}
	if err := CheckBase(base); err != nil {
		return Plan{}, err
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
	if name == base {
		return Plan{}, fmt.Errorf("branch %q cannot start from itself", name)
	}
	if !Valid(name, base) {
		return Plan{}, fmt.Errorf("branch %q is not a span branch", name)
	}
	return Plan{Kind: kind, Base: base, Name: name, Feature: strings.ToLower(suffix)}, nil
}

// FeatureOf is the run identity a branch gives: the part after "<kind>/" (the whole name
// when there is no slash), lowercased. A suffix that breaks the feature rule is slugged, so
// "a/b" becomes "a-b". It is "" when nothing usable is left; the caller picks a fallback.
func FeatureOf(name string) string {
	name = strings.TrimSpace(name)
	if _, suffix, ok := strings.Cut(name, "/"); ok {
		name = suffix
	}
	name = strings.ToLower(name)
	if ValidFeature(name) {
		return name
	}
	return slug(name)
}

// ValidFeature reports whether name may name a feature document: letters, digits, dots,
// dashes, or underscores, and never "." or "..".
func ValidFeature(name string) bool {
	if name == "" || name == "." || name == ".." {
		return false
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-' || r == '_' || r == '.':
		default:
			return false
		}
	}
	return true
}

// Valid reports whether name is one of the three prefixes on a base that is a valid branch
// name other than name itself.
func Valid(name, base string) bool {
	return branchMatch.MatchString(name) && ValidBase(base) && base != name
}

// ValidBase reports whether name may be a base: a valid branch name that git cannot read as
// an option, HEAD, or a commit. A base reaches git fetch, git worktree add and gh pr create,
// so "--upload-pack=..." must never get that far.
func ValidBase(name string) bool {
	return name == strings.TrimSpace(name) && ValidRef(name) && name != "HEAD" && !shaLike.MatchString(name)
}

// CheckBase is ValidBase as the error a person reads.
func CheckBase(name string) error {
	if ValidBase(name) {
		return nil
	}
	return fmt.Errorf("invalid base branch %q: use a branch name, not an option, HEAD, or a commit", name)
}

// Choose settles the kind and the base before Derive. A base the run names (explicit) is
// the person's choice for any kind and is returned as is. Otherwise a project base (the
// base: line of span.yaml) applies to feat and fix only: the kind is inferred without it, so
// "base: master" never turns a story into a hotfix, and a hotfix keeps master. The kind is
// returned unchanged unless the project base applied, when it is the inferred one, so
// Derive cannot infer it again from that base. An empty base leaves the default to Derive.
func Choose(story, kind, explicit, project string) (string, string, error) {
	// A blank field is no choice; anything else is checked as typed.
	if strings.TrimSpace(explicit) != "" {
		return kind, explicit, CheckBase(explicit)
	}
	if project == "" {
		return kind, "", nil
	}
	if err := CheckBase(project); err != nil {
		return "", "", err
	}
	settled := strings.ToLower(strings.TrimSpace(kind))
	if settled == "" {
		settled = inferKind(story, "")
	}
	if settled == Hotfix {
		return kind, "", nil
	}
	return settled, project, nil
}

// DefaultBase is where a kind starts when nobody chose a base: hotfix from master, feat and
// fix from dev.
func DefaultBase(kind string) string {
	if kind == Hotfix {
		return Master
	}
	return Dev
}

// ValidRef reports whether name is a branch name git accepts, by the rules of
// git check-ref-format --branch, and never the bare name HEAD: no control characters, spaces or any of ~^:?*[\, no "..",
// no "@{", not "@", no leading "-", no empty, leading-dot or ".lock" component, and no
// leading or trailing "/" or trailing ".".
func ValidRef(name string) bool {
	if name == "" || name == "@" || name == "HEAD" || strings.HasPrefix(name, "-") {
		return false
	}
	if strings.Contains(name, "..") || strings.Contains(name, "@{") || strings.HasSuffix(name, ".") {
		return false
	}
	for _, r := range name {
		if r < 0x20 || r == 0x7f || strings.ContainsRune(" ~^:?*[\\", r) {
			return false
		}
	}
	for _, part := range strings.Split(name, "/") {
		if part == "" || strings.HasPrefix(part, ".") || strings.HasSuffix(part, ".lock") {
			return false
		}
	}
	return true
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
