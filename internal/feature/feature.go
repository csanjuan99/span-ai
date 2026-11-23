// Package feature checks the ODD document Build must leave behind.
package feature

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Valid reports whether the feature document has a title and the three
// sections ODD requires.
func Valid(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("feature document: %w", err)
	}
	defer f.Close()
	var title bool
	sections := map[string]bool{"## Specs": false, "## Tasks": false, "## Log": false}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "# ") {
			title = true
		}
		if _, ok := sections[line]; ok {
			sections[line] = true
		}
	}
	if err := sc.Err(); err != nil {
		return err
	}
	if !title {
		return fmt.Errorf("feature document has no heading")
	}
	for name, ok := range sections {
		if !ok {
			return fmt.Errorf("feature document missing %s", name)
		}
	}
	return nil
}
