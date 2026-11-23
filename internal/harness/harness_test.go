package harness

import (
	"strings"
	"testing"
)

func TestCatalogNamesSkillsWithoutInliningBodies(t *testing.T) {
	root := Root()
	if root == "" {
		t.Fatal("harness root not found")
	}
	got := Catalog(root)
	for _, name := range []string{"clean-code", "solid-principles", "api-first"} {
		if !strings.Contains(got, name) {
			t.Fatalf("missing %s\n%s", name, got)
		}
	}
	if strings.Contains(got, "Never swallow errors") {
		t.Fatal("skill body was inlined")
	}
	if !strings.Contains(got, "SKILL.md") {
		t.Fatal("no skill path")
	}
}
