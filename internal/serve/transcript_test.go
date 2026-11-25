package serve

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func writeTranscript(t *testing.T, root, id, text, stages string) {
	t.Helper()
	path := filepath.Join(root, id, "transcript.log")
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	if stages != "" {
		if err := os.WriteFile(path+".stages", []byte(stages), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestTheTranscriptOfAStageIsItsBlocksInOrder(t *testing.T) {
	root, id := featureRun(t, "pay", "")
	first := "I will read the client.\n\nAdded the scheduler. key [redacted] hidden.\n\n```go\nx := 1\n\ny := 2\n```\n"
	text := first + "Now verifying.\n"
	stages := `{"stage":"build","offset":0,"at":"2026-11-25T14:25:00Z"}` + "\n" +
		`{"stage":"prove","offset":` + itoa(len(first)) + `,"at":"2026-11-25T14:47:00Z"}` + "\n"
	writeTranscript(t, root, id, text, stages)
	var view TranscriptView
	if code := getJSON(t, root, "/spans/"+id+"/transcript?stage=build", &view); code != http.StatusOK {
		t.Fatalf("status %d", code)
	}
	if view.Total != 3 || view.Turns[0].Text != "I will read the client." || view.Redacted != 1 {
		t.Fatalf("view %+v", view)
	}
	if !strings.Contains(view.Turns[2].Text, "x := 1\n\ny := 2") {
		t.Fatalf("a fenced block stays whole: %q", view.Turns[2].Text)
	}
	if len(view.Stages) != 2 || view.Structured || view.EndedAt.IsZero() {
		t.Fatalf("stages %+v structured %v", view.Stages, view.Structured)
	}
	var last TranscriptView
	getJSON(t, root, "/spans/"+id+"/transcript", &last)
	if last.Stage != "prove" || last.Total != 1 {
		t.Fatalf("with no stage the last one that printed is read: %+v", last)
	}
	var page TranscriptView
	getJSON(t, root, "/spans/"+id+"/transcript?stage=build&from=1&limit=1", &page)
	if page.From != 1 || len(page.Turns) != 1 || page.Turns[0].N != 2 {
		t.Fatalf("page %+v", page)
	}
	if code := getJSON(t, root, "/spans/"+id+"/transcript?stage=deploy", &page); code != http.StatusNotFound {
		t.Fatalf("a stage that printed nothing answers 404, got %d", code)
	}
}

func itoa(n int) string { return strconv.Itoa(n) }
