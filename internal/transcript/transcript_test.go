package transcript

import (
	"errors"
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
	log, err := Open(filepath.Join(dir, "transcript.log"), nil, func(batch Batch) {
		mu.Lock()
		got = append(got, batch.Lines...)
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

// recorder keeps every published batch in publication order.
type recorder struct {
	mu  sync.Mutex
	got []Batch
}

func (r *recorder) publish(batch Batch) {
	r.mu.Lock()
	r.got = append(r.got, batch)
	r.mu.Unlock()
}

func (r *recorder) batches() []Batch {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]Batch(nil), r.got...)
}

func openAt(t *testing.T, before string) (string, *Log, *recorder) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "transcript.log")
	if before != "" {
		if err := os.WriteFile(path, []byte(before), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	rec := &recorder{}
	log, err := Open(path, nil, rec.publish)
	if err != nil {
		t.Fatal(err)
	}
	return path, log, rec
}

func TestABatchCarriesWhereItStartsAndEndsInTheFile(t *testing.T) {
	_, log, rec := openAt(t, "")
	log.SetStage("build")
	if _, err := log.Write([]byte("h\u00e9llo\n\nz\n")); err != nil {
		t.Fatal(err)
	}
	log.SetStage("qa")
	if _, err := log.Write([]byte("next\n")); err != nil {
		t.Fatal(err)
	}
	if err := log.Close(); err != nil {
		t.Fatal(err)
	}
	got := rec.batches()
	if len(got) != 2 {
		t.Fatalf("batches %+v", got)
	}
	// "héllo" is 6 bytes, the empty line 1, "z" 2: the first batch is [0, 10).
	if got[0].Start != 0 || got[0].End != 10 || len(got[0].Lines) != 3 || got[0].Stage != "build" {
		t.Fatalf("first %+v", got[0])
	}
	if got[1].Start != 10 || got[1].End != 15 || got[1].Stage != "qa" {
		t.Fatalf("second %+v", got[1])
	}
}

// The verifier's interleaving: one send is paused right after it took its batch, and a
// newer batch is taken meanwhile. Publication must follow take order.
func TestBatchesArePublishedInTheOrderTheyWereTaken(t *testing.T) {
	path, log, rec := openAt(t, "old\n")
	entered := make(chan struct{})
	release := make(chan struct{})
	tookSecond := make(chan struct{})
	takes := 0
	pauses := 0
	log.mu.Lock()
	log.afterTake = func(lines []string) {
		takes++
		if takes == 2 {
			close(tookSecond)
		}
	}
	log.beforePublish = func() {
		pauses++
		if pauses == 1 {
			close(entered)
			<-release
		}
	}
	log.mu.Unlock()
	log.SetStage("build")
	if _, err := log.Write([]byte("AAAAAAAAAA\n")); err != nil {
		t.Fatal(err)
	}
	firstDone := make(chan struct{})
	go func() {
		defer close(firstDone)
		log.SetStage("qa")
	}()
	<-entered
	if _, err := log.Write([]byte("B\n")); err != nil {
		t.Fatal(err)
	}
	secondDone := make(chan struct{})
	go func() {
		defer close(secondDone)
		log.SetStage("done")
	}()
	<-tookSecond
	close(release)
	<-firstDone
	<-secondDone
	if err := log.Close(); err != nil {
		t.Fatal(err)
	}
	got := rec.batches()
	if len(got) != 2 || got[0].Lines[0] != "AAAAAAAAAA" || got[1].Lines[0] != "B" {
		t.Fatalf("publication order %+v", got)
	}
	if got[0].Start != 4 || got[0].End != 15 || got[1].Start != 15 || got[1].End != 17 {
		t.Fatalf("offsets %+v", got)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() != 17 {
		t.Fatalf("file size %d", info.Size())
	}
}

// shortWriter accepts the first limit bytes and then fails, like a full disk.
type shortWriter struct {
	limit int
	wrote []byte
}

func (w *shortWriter) Write(p []byte) (int, error) {
	n := len(p)
	if n > w.limit {
		n = w.limit
	}
	w.limit -= n
	w.wrote = append(w.wrote, p[:n]...)
	if n < len(p) {
		return n, errors.New("disk full")
	}
	return n, nil
}

func TestAShortWritePublishesOnlyTheLinesThatReachedTheFile(t *testing.T) {
	_, log, rec := openAt(t, "")
	out := &shortWriter{limit: 6}
	log.mu.Lock()
	log.out = out
	log.mu.Unlock()
	log.SetStage("build")
	n, err := log.Write([]byte("one\ntwo\n"))
	if err != nil || n != len("one\ntwo\n") {
		t.Fatalf("Write must not fail the agent: %d %v", n, err)
	}
	closeErr := log.Close()
	if closeErr == nil || !strings.Contains(closeErr.Error(), "disk full") {
		t.Fatalf("close error %v", closeErr)
	}
	got := rec.batches()
	if len(got) != 1 || len(got[0].Lines) != 1 || got[0].Lines[0] != "one" || got[0].End != 4 {
		t.Fatalf("published %+v", got)
	}
	if string(out.wrote) != "one\ntw" {
		t.Fatalf("wrote %q", out.wrote)
	}
}

func TestStagesSplitTheTranscriptWhereEachBegan(t *testing.T) {
	path := filepath.Join(t.TempDir(), "transcript.log")
	log, err := Open(path, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	log.SetStage("frame")
	if _, err := log.Write([]byte("one\ntwo\n")); err != nil {
		t.Fatal(err)
	}
	log.SetStage("build")
	if _, err := log.Write([]byte("three\n")); err != nil {
		t.Fatal(err)
	}
	if err := log.Close(); err != nil {
		t.Fatal(err)
	}
	segs, err := Segments(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(segs) != 2 || segs[0].Stage != "frame" || len(segs[0].Lines) != 2 || segs[1].Stage != "build" || segs[1].Lines[0] != "three" {
		t.Fatalf("segments %#v", segs)
	}
	if segs[1].FirstLine != 3 || segs[0].EndedAt.IsZero() {
		t.Fatalf("second segment %#v", segs[1])
	}
}

func TestATranscriptWithoutMarkersIsOneSegment(t *testing.T) {
	path := filepath.Join(t.TempDir(), "transcript.log")
	if err := os.WriteFile(path, []byte("a\nb\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	segs, err := Segments(path)
	if err != nil || len(segs) != 1 || segs[0].Stage != "" || len(segs[0].Lines) != 2 {
		t.Fatalf("segments %#v err %v", segs, err)
	}
	if none, err := Segments(filepath.Join(t.TempDir(), "missing.log")); err != nil || none != nil {
		t.Fatalf("a missing transcript has no segment: %#v %v", none, err)
	}
}
