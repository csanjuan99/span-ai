// Package handoff stores story files where the frame can open them.
// A file is optional. Images and notes use the same folder.
package handoff

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

const (
	// MaxFiles is how many files one story accepts.
	MaxFiles = 20
	// MaxBytes is the size of one file.
	MaxBytes = 32 << 20
)

// File is one attachment. Name may include a directory; only the base name is kept.
type File struct {
	Name string
	Body []byte
}

// Write copies files into dir. Names are made safe and unique.
func Write(dir string, files []File) error {
	if len(files) == 0 {
		return nil
	}
	if len(files) > MaxFiles {
		return fmt.Errorf("invalid: a story accepts at most %d files", MaxFiles)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	used := map[string]bool{}
	for _, file := range files {
		name, err := clean(file.Name)
		if err != nil {
			return err
		}
		if len(file.Body) > MaxBytes {
			return fmt.Errorf("invalid: %s is larger than 32MiB", name)
		}
		name = unique(name, used)
		if err := os.WriteFile(filepath.Join(dir, name), file.Body, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// Paths are workspace-relative locations the agent can open.
func Paths(spanID string, names []string) []string {
	out := make([]string, 0, len(names))
	for _, name := range names {
		if name == "" {
			continue
		}
		out = append(out, ".span/runs/"+spanID+"/handoffs/"+name)
	}
	return out
}

func clean(name string) (string, error) {
	name = filepath.Base(strings.ReplaceAll(name, "\\", "/"))
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." {
		return "", fmt.Errorf("invalid: file name is missing")
	}
	var b strings.Builder
	for _, r := range name {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '.' || r == '-' || r == '_':
			b.WriteRune(r)
		case unicode.IsSpace(r):
			b.WriteByte('-')
		}
	}
	out := strings.Trim(b.String(), ".-")
	if out == "" {
		return "", fmt.Errorf("invalid: file name is missing")
	}
	return out, nil
}

func unique(name string, used map[string]bool) string {
	if !used[name] {
		used[name] = true
		return name
	}
	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	for i := 2; ; i++ {
		next := fmt.Sprintf("%s-%d%s", stem, i, ext)
		if !used[next] {
			used[next] = true
			return next
		}
	}
}
