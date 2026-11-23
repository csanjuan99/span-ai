package launch

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"span/internal/jira"
	"span/internal/scope"
)

func TestTwoReposNeedASelectionAndOneStoryStaysInEach(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"salute-infra", "salute-apollo"} {
		if err := os.MkdirAll(filepath.Join(dir, name, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	_, err := Resolve(context.Background(), Request{Path: dir, Story: "Add saved cards"}, nil)
	if err == nil || err.(*scope.Error).Code != "selection_required" {
		t.Fatalf("%v", err)
	}
	got, err := Resolve(context.Background(), Request{
		Path: dir, Names: []string{"salute-infra", "salute-apollo"}, Story: "Add saved cards",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Name != "salute-infra" {
		t.Fatalf("%+v", got)
	}
	if !strings.Contains(got[0].Story, "Write only in repo salute-infra.") || !strings.Contains(got[0].Story, "salute-apollo") {
		t.Fatalf("%s", got[0].Story)
	}
}

func TestEmptyStoryLoadsTheJiraIssue(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := Resolve(context.Background(), Request{Path: dir, Issue: "AET-12"}, func(context.Context, string) (jira.Story, error) {
		return jira.Story{Key: "AET-12", Brief: "AET-12 Charge the card\n"}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !strings.Contains(got[0].Story, "AET-12 Charge the card") || !strings.Contains(got[0].Story, "Write only in repo") {
		t.Fatalf("%+v", got)
	}
}
