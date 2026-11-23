package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestTheLabelNamesTheStoryTheSessionAndTheRepo(t *testing.T) {
	cur := Cursor{ID: "2026-11-19-0253-87f0", Issue: "SAL-3", Workspace: "/Volumes/External/dev/salute/salute-hermes"}
	if got := Label(cur); got != "SAL-3-2026-11-19-0253-87f0-salute-hermes" {
		t.Fatalf("label = %q", got)
	}
}

func TestTheLabelSkipsWhatTheRunDoesNotHave(t *testing.T) {
	cases := map[string]Cursor{
		"2026-11-19-0253-87f0-salute-hermes": {ID: "2026-11-19-0253-87f0", Workspace: "/x/salute-hermes"},
		"SAL-3-2026-11-19-0253-87f0":         {ID: "2026-11-19-0253-87f0", Issue: "SAL-3"},
		"SAL-3-salute-hermes":                {Issue: "SAL-3", Workspace: "/x/salute-hermes/"},
	}
	for want, cur := range cases {
		if got := Label(cur); got != want {
			t.Fatalf("label = %q, want %q", got, want)
		}
	}
}

func TestTheListCarriesTheLabelAndTheStage(t *testing.T) {
	root := t.TempDir()
	write := func(id string, cur Cursor) {
		dir := filepath.Join(root, id)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		cur.ID = id
		raw, err := json.Marshal(cur)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "cursor.json"), raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("2026-11-19-0253-87f0", Cursor{Issue: "SAL-3", Stage: "merge", Hold: true, Workspace: "/x/salute-hermes"})
	write("2026-11-19-0253-1a7d", Cursor{Issue: "SAL-3", Stage: "qa", Workspace: "/x/salute-infra/"})

	heads := Heads(root)
	if len(heads) != 2 {
		t.Fatalf("heads = %d", len(heads))
	}
	if heads[0].ID != "2026-11-19-0253-1a7d" {
		t.Fatalf("the list is not ordered: %+v", heads)
	}
	if heads[0].Label != "SAL-3-2026-11-19-0253-1a7d-salute-infra" {
		t.Fatalf("label = %q", heads[0].Label)
	}
	if heads[1].Verdict != "hold" || heads[1].Stage != "merge" {
		t.Fatalf("row = %+v", heads[1])
	}
	if heads[0].Repo != "salute-infra" {
		t.Fatalf("repo = %q", heads[0].Repo)
	}
}
