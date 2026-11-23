package scope

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverInsideOneRepo(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := Discover(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got.Mode != "repo" || len(got.Repos) != 1 || got.Repos[0].Path != dir {
		t.Fatalf("%+v", got)
	}
	chosen, err := Select(got, nil)
	if err != nil || len(chosen) != 1 {
		t.Fatalf("%v %+v", err, chosen)
	}
}

func TestDiscoverFolderRequiresAChoice(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"salute-apollo", "salute-infra", "salute-hermes"} {
		if err := os.MkdirAll(filepath.Join(dir, name, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(dir, "notes"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := Discover(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got.Mode != "folder" || len(got.Repos) != 3 {
		t.Fatalf("%+v", got)
	}
	if _, err := Select(got, nil); err == nil || err.(*Error).Code != "selection_required" {
		t.Fatalf("%v", err)
	}
	chosen, err := Select(got, []string{"salute-infra", "salute-apollo"})
	if err != nil {
		t.Fatal(err)
	}
	if len(chosen) != 2 || chosen[0].Name != "salute-infra" || chosen[1].Name != "salute-apollo" {
		t.Fatalf("%+v", chosen)
	}
	if _, err := Select(got, []string{"missing"}); err == nil || err.(*Error).Code != "unknown_repo" {
		t.Fatalf("%v", err)
	}
}

func TestParentAndChildrenAreBothSelectable(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "salute-infra", ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := Discover(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got.Mode != "folder" || len(got.Repos) != 2 {
		t.Fatalf("%+v", got)
	}
}
