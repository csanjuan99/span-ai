package transcript

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestWriteRedactsATokenSplitAcrossChunks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "transcript.log")
	log, err := Open(path, []string{"secret-token"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := log.Write([]byte("aa secret-")); err != nil {
		t.Fatal(err)
	}
	if _, err := log.Write([]byte("token bb\n")); err != nil {
		t.Fatal(err)
	}
	if err := log.Close(); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "secret-token") {
		t.Fatalf("token leaked: %s", raw)
	}
	if !strings.Contains(string(raw), "[redacted]") {
		t.Fatalf("body %s", raw)
	}
}

func TestTailKeepsTheLastLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "transcript.log")
	var b strings.Builder
	for i := 0; i < 1005; i++ {
		b.WriteString("line\n")
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	lines, err := Tail(path, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 1000 {
		t.Fatalf("lines %d", len(lines))
	}
}

func TestPublishFlushesOnClose(t *testing.T) {
	dir := t.TempDir()
	var mu sync.Mutex
	var got []string
	log, err := Open(filepath.Join(dir, "transcript.log"), nil, func(_ string, lines []string) {
		mu.Lock()
		got = append(got, lines...)
		mu.Unlock()
	})
	if err != nil {
		t.Fatal(err)
	}
	log.SetStage("build")
	if _, err := log.Write([]byte("wrote the charge\n")); err != nil {
		t.Fatal(err)
	}
	if err := log.Close(); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(got) != 1 || got[0] != "wrote the charge" {
		t.Fatalf("%q", got)
	}
}
