// Package harness locates the Span skill catalog and lists each skill by name.
// Skill bodies stay on disk. Callers pass paths, and the agent reads a skill
// only when its trigger matches the change.
package harness

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Root is the harness directory, or empty when this checkout has none.
func Root() string {
	if p := os.Getenv("SPAN_HARNESS"); isRoot(p) {
		return p
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		for _, candidate := range []string{
			filepath.Join(dir, "harness"),
			filepath.Join(dir, "..", "harness"),
		} {
			if isRoot(candidate) {
				return candidate
			}
		}
	}
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for i := 0; i < 8; i++ {
		candidate := filepath.Join(dir, "harness")
		if isRoot(candidate) {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func isRoot(path string) bool {
	if path == "" {
		return false
	}
	info, err := os.Stat(filepath.Join(path, "skills"))
	return err == nil && info.IsDir()
}

// Skill is one catalog entry. Path is the SKILL.md to read when needed.
type Skill struct {
	Name        string
	Description string
	Path        string
}

// Skills reads frontmatter only. Reference files are not loaded.
func Skills(root string) ([]Skill, error) {
	if root == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(filepath.Join(root, "skills"))
	if err != nil {
		return nil, err
	}
	var out []Skill
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(root, "skills", entry.Name(), "SKILL.md")
		name, desc, err := front(path)
		if err != nil {
			continue
		}
		if name == "" {
			name = entry.Name()
		}
		out = append(out, Skill{Name: name, Description: oneLine(desc, 220), Path: path})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// Catalog is the prompt block. It names skills and where to read them.
func Catalog(root string) string {
	skills, err := Skills(root)
	if err != nil || len(skills) == 0 {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Harness: %s\n", root)
	for _, skill := range skills {
		fmt.Fprintf(&b, "- %s: %s\n  read: %s\n", skill.Name, skill.Description, skill.Path)
	}
	return b.String()
}

func front(path string) (string, string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}
	text := string(raw)
	if !strings.HasPrefix(text, "---\n") {
		return "", "", fmt.Errorf("no frontmatter")
	}
	rest := text[4:]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return "", "", fmt.Errorf("no frontmatter end")
	}
	var name, desc string
	for _, line := range strings.Split(rest[:end], "\n") {
		switch {
		case strings.HasPrefix(line, "name:"):
			name = unquote(strings.TrimSpace(strings.TrimPrefix(line, "name:")))
		case strings.HasPrefix(line, "description:"):
			desc = unquote(strings.TrimSpace(strings.TrimPrefix(line, "description:")))
		}
	}
	return name, desc, nil
}

func unquote(s string) string {
	return strings.Trim(s, `"'`)
}

func oneLine(s string, max int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > max {
		return s[:max]
	}
	return s
}
