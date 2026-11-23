package engine

import (
	"path/filepath"
	"strings"
	"testing"

	"span/internal/learn"
	"span/internal/statement"
)

// Two runs of one project, each in its own tree, learn different things at the same time.
// Neither may erase what the other wrote, and both must write to the project's book, not
// to a book inside a tree that is removed with the run.
func TestParallelRunsAddToOneLessonBookAndLoseNothing(t *testing.T) {
	project, treeA, treeB := t.TempDir(), t.TempDir(), t.TempDir()
	newRunner := func(tree string) *runner {
		return &runner{opt: Options{Workspace: tree, Project: project}, cur: Cursor{}}
	}
	a, b := newRunner(treeA), newRunner(treeB)
	// Both read the book before either saves, which is how two live runs overlap.
	a.lessonBook()
	b.lessonBook()
	a.noteLesson("build", statement.Hold, "the migration is missing an index")
	b.noteLesson("build", statement.Hold, "the handler ignores the context")

	book, err := learn.Load(filepath.Join(project, ".span", "lessons.json"))
	if err != nil {
		t.Fatal(err)
	}
	var all []string
	for _, lesson := range book.Lessons {
		all = append(all, lesson.Rule)
	}
	text := strings.Join(all, "|")
	if !strings.Contains(text, "migration is missing an index") || !strings.Contains(text, "handler ignores the context") {
		t.Fatalf("a parallel run erased the other's lesson: %q", text)
	}
}
