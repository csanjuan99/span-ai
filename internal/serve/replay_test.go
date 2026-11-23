package serve

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"span/internal/engine"
	"span/internal/live"
)

func liveRun(t *testing.T) (root, id string) {
	t.Helper()
	root = t.TempDir()
	id = "span-1"
	dir := filepath.Join(root, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(engine.Cursor{ID: id, Stage: "build"})
	if err := os.WriteFile(filepath.Join(dir, "cursor.json"), append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	return root, id
}

func dialLive(t *testing.T, ctx context.Context, srv *httptest.Server, id string) *websocket.Conn {
	t.Helper()
	conn, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"/spans/"+id+"/live", nil)
	if err != nil {
		t.Fatal(err)
	}
	conn.SetReadLimit(-1)
	t.Cleanup(func() { conn.CloseNow() })
	return conn
}

func TestLiveReplayCarriesTheSeqTheStreamResumesFrom(t *testing.T) {
	root, id := liveRun(t)
	hub := live.New()
	hub.Publish(id, engine.Event{Type: "stage", Stage: "frame"})
	hub.Publish(id, engine.Event{Type: "stage", Stage: "build"})
	srv := httptest.NewServer(New(root, "", nil, nil, hub, nil))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn := dialLive(t, ctx, srv, id)
	var replay engine.Event
	if err := wsjson.Read(ctx, conn, &replay); err != nil {
		t.Fatal(err)
	}
	if replay.Type != "replay" || replay.Seq != 2 {
		t.Fatalf("replay %s seq %d, want seq 2", replay.Type, replay.Seq)
	}
	hub.Publish(id, engine.Event{Type: "stage", Stage: "qa"})
	var next engine.Event
	if err := wsjson.Read(ctx, conn, &next); err != nil {
		t.Fatal(err)
	}
	if next.Seq != 3 || next.Stage != "qa" {
		t.Fatalf("next %+v", next)
	}
}

func TestASlowClientIsCutAndResyncsFromAFreshReplay(t *testing.T) {
	root, id := liveRun(t)
	hub := live.New()
	srv := httptest.NewServer(New(root, "", nil, nil, hub, nil))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	slow := dialLive(t, ctx, srv, id)
	var replay engine.Event
	if err := wsjson.Read(ctx, slow, &replay); err != nil {
		t.Fatal(err)
	}
	// The client stops reading. Large frames fill the socket buffers, the server stops
	// draining the hub channel, and the hub cuts the subscriber.
	line := strings.Repeat("x", 64*1024)
	const published = 2000
	for i := 0; i < published; i++ {
		hub.Publish(id, engine.Event{Type: "lines", Lines: []string{line}})
	}
	read := 0
	for {
		var frame engine.Event
		if err := wsjson.Read(ctx, slow, &frame); err != nil {
			break
		}
		read++
	}
	if read >= published {
		t.Fatalf("the slow client read every frame (%d); it was never cut", read)
	}
	again := dialLive(t, ctx, srv, id)
	var fresh engine.Event
	if err := wsjson.Read(ctx, again, &fresh); err != nil {
		t.Fatal(err)
	}
	if fresh.Type != "replay" || fresh.Seq != published {
		t.Fatalf("fresh replay %s seq %d, want %d", fresh.Type, fresh.Seq, published)
	}
}

func TestALineWrittenJustBeforeConnectArrivesOnce(t *testing.T) {
	root, id := liveRun(t)
	path := filepath.Join(root, id, "transcript.log")
	if err := os.WriteFile(path, []byte("compiling pkg A\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	hub := live.New()
	srv := httptest.NewServer(New(root, "", nil, nil, hub, nil))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn := dialLive(t, ctx, srv, id)
	var replay engine.Event
	if err := wsjson.Read(ctx, conn, &replay); err != nil {
		t.Fatal(err)
	}
	if len(replay.Transcript) != 1 || replay.Transcript[0] != "compiling pkg A" {
		t.Fatalf("replay transcript %q", replay.Transcript)
	}
	if replay.Offset != int64(len("compiling pkg A\n")) {
		t.Fatalf("replay offset %d, want the file size %d", replay.Offset, len("compiling pkg A\n"))
	}
	// The delayed flush of the same line: its offset is not past the replay's, so the
	// page drops it instead of showing the line twice.
	hub.Publish(id, engine.Event{Type: "transcript", Stage: "build", Lines: []string{"compiling pkg A"}, Offset: replay.Offset})
	var late engine.Event
	if err := wsjson.Read(ctx, conn, &late); err != nil {
		t.Fatal(err)
	}
	if late.Offset > replay.Offset {
		t.Fatalf("late offset %d is past the replay offset %d", late.Offset, replay.Offset)
	}
}

func TestLiveSendsABaselineReplayWhenTheRunCannotBeLoaded(t *testing.T) {
	root := t.TempDir()
	id := "span-missing"
	hub := live.New()
	for i := 0; i < 3; i++ {
		hub.Publish(id, engine.Event{Type: "stage"})
	}
	srv := httptest.NewServer(New(root, "", nil, nil, hub, nil))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn := dialLive(t, ctx, srv, id)
	var replay engine.Event
	if err := wsjson.Read(ctx, conn, &replay); err != nil {
		t.Fatal(err)
	}
	if replay.Type != "replay" || replay.Span != nil || replay.Seq != 3 || replay.Offset != 0 {
		t.Fatalf("baseline replay %+v", replay)
	}
}

func TestTheReplayIsScrubbedOfTheServeToken(t *testing.T) {
	root, id := liveRun(t)
	const token = "tok-live-123"
	if err := os.WriteFile(filepath.Join(root, id, "transcript.log"), []byte("export SPAN_TOKEN="+token+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(New(root, token, nil, nil, live.New(), nil))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"/spans/"+id+"/live", &websocket.DialOptions{Subprotocols: []string{bearerProtocol(token)}})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.CloseNow()
	_, raw, err := conn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), token) || !strings.Contains(string(raw), "[redacted]") {
		t.Fatalf("replay %s", raw)
	}
}

func TestAStalledReaderIsDroppedByTheWriteDeadline(t *testing.T) {
	previous := liveWriteTimeout.Swap(int64(200 * time.Millisecond))
	t.Cleanup(func() { liveWriteTimeout.Store(previous) })
	root, id := liveRun(t)
	hub := live.New()
	srv := httptest.NewServer(New(root, "", nil, nil, hub, nil))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	conn := dialLive(t, ctx, srv, id)
	var replay engine.Event
	if err := wsjson.Read(ctx, conn, &replay); err != nil {
		t.Fatal(err)
	}
	// Fewer frames than the hub buffer, so only the write deadline can end the socket.
	const published = 20
	line := strings.Repeat("y", 2<<20)
	for i := 0; i < published; i++ {
		hub.Publish(id, engine.Event{Type: "lines", Lines: []string{line}})
	}
	time.Sleep(1500 * time.Millisecond)
	read := 0
	for {
		var frame engine.Event
		if err := wsjson.Read(ctx, conn, &frame); err != nil {
			break
		}
		read++
	}
	if read >= published {
		t.Fatalf("the stalled reader got all %d frames; the write never timed out", read)
	}
}

func readRawFrame(t *testing.T, ctx context.Context, conn *websocket.Conn) map[string]any {
	t.Helper()
	_, raw, err := conn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var frame map[string]any
	if err := json.Unmarshal(raw, &frame); err != nil {
		t.Fatalf("the frame is not valid JSON: %v %s", err, raw)
	}
	return frame
}

func firstString(t *testing.T, frame map[string]any, key string) string {
	t.Helper()
	list, ok := frame[key].([]any)
	if !ok || len(list) == 0 {
		t.Fatalf("%s missing in %v", key, frame)
	}
	text, ok := list[0].(string)
	if !ok {
		t.Fatalf("%s[0] is not a string: %v", key, list[0])
	}
	return text
}

func TestRedactionCoversTokensJSONEscapes(t *testing.T) {
	tokens := []string{`quote"token-1`, `back\slash-token`, `x<y&z-token-3`, "tökén-ünicode-4"}
	for _, token := range tokens {
		t.Run(token, func(t *testing.T) {
			root, id := liveRun(t)
			if err := os.WriteFile(filepath.Join(root, id, "transcript.log"), []byte("saw "+token+" here\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			hub := live.New()
			srv := httptest.NewServer(New(root, token, nil, nil, hub, nil))
			defer srv.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			conn, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"/spans/"+id+"/live", &websocket.DialOptions{Subprotocols: []string{bearerProtocol(token)}})
			if err != nil {
				t.Fatal(err)
			}
			defer conn.CloseNow()
			replay := readRawFrame(t, ctx, conn)
			if replay["type"] != "replay" {
				t.Fatalf("type %v", replay["type"])
			}
			if line := firstString(t, replay, "transcript"); line != "saw [redacted] here" {
				t.Fatalf("replay line %q", line)
			}
			hub.Publish(id, engine.Event{Type: "transcript", Stage: "build", Lines: []string{"again " + token}})
			next := readRawFrame(t, ctx, conn)
			if line := firstString(t, next, "lines"); line != "again [redacted]" || next["type"] != "transcript" {
				t.Fatalf("live frame %v", next)
			}
		})
	}
}

func TestAShortTokenNeverRewritesTheFrame(t *testing.T) {
	const token = "type"
	root, id := liveRun(t)
	if err := os.WriteFile(filepath.Join(root, id, "transcript.log"), []byte("the type checker ran\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(New(root, token, nil, nil, live.New(), nil))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"/spans/"+id+"/live", &websocket.DialOptions{Subprotocols: []string{bearerProtocol(token)}})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.CloseNow()
	replay := readRawFrame(t, ctx, conn)
	if replay["type"] != "replay" {
		t.Fatalf("keys were rewritten: %v", replay)
	}
	if line := firstString(t, replay, "transcript"); line != "the type checker ran" {
		t.Fatalf("a short token redacted ordinary text: %q", line)
	}
}

func transcriptFrame(start int64, lines ...string) engine.Event {
	end := start
	for _, line := range lines {
		end += int64(len(line)) + 1
	}
	return engine.Event{Type: "transcript", Stage: "build", Lines: lines, Start: &start, Offset: end}
}

func TestCutCoveredKeepsOnlyTheLinesPastTheReplay(t *testing.T) {
	cases := []struct {
		name   string
		frame  engine.Event
		replay int64
		want   []string
	}{
		{"straddles the replay", transcriptFrame(0, "a", "b"), 2, []string{"b"}},
		{"multi-byte lines", transcriptFrame(0, "hé", "日本", "z"), int64(len("hé\n日本\n")), []string{"z"}},
		{"an empty line counts its newline", transcriptFrame(0, "", "x"), 1, []string{"x"}},
		{"entirely before the replay", transcriptFrame(0, "a", "b"), 10, nil},
		{"entirely after the replay", transcriptFrame(5, "c", "d"), 5, []string{"c", "d"}},
		{"no replay offset", transcriptFrame(0, "a"), 0, []string{"a"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := cutCovered(tc.frame, tc.replay)
			if strings.Join(got.Lines, "|") != strings.Join(tc.want, "|") || len(got.Lines) != len(tc.want) {
				t.Fatalf("lines %q, want %q", got.Lines, tc.want)
			}
			if got.Seq != tc.frame.Seq || got.Offset != tc.frame.Offset || got.Type != "transcript" {
				t.Fatalf("frame identity changed: %+v", got)
			}
		})
	}
	noStart := engine.Event{Type: "transcript", Lines: []string{"a"}, Offset: 2}
	if got := cutCovered(noStart, 2); len(got.Lines) != 1 {
		t.Fatalf("a frame without start was cut: %+v", got)
	}
}

func TestABatchThatStraddlesTheReplayShowsEachLineOnce(t *testing.T) {
	root, id := liveRun(t)
	hub := live.New()
	log, err := engine.OpenTranscript(filepath.Join(root, id, "transcript.log"), nil, func(event engine.Event) { hub.Publish(id, event) })
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	log.SetStage("build")
	if _, err := log.Write([]byte("a\n")); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(New(root, "", nil, nil, hub, nil))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn := dialLive(t, ctx, srv, id)
	var replay engine.Event
	if err := wsjson.Read(ctx, conn, &replay); err != nil {
		t.Fatal(err)
	}
	if replay.Offset != 2 || strings.Join(replay.Transcript, "|") != "a" {
		t.Fatalf("replay offset %d tail %q", replay.Offset, replay.Transcript)
	}
	if _, err := log.Write([]byte("b\n")); err != nil {
		t.Fatal(err)
	}
	log.SetStage("qa")
	shown := append([]string(nil), replay.Transcript...)
	for !(len(shown) > 0 && shown[len(shown)-1] == "b") {
		var frame engine.Event
		if err := wsjson.Read(ctx, conn, &frame); err != nil {
			t.Fatalf("after %q: %v", shown, err)
		}
		if frame.Type == "transcript" && frame.Offset > replay.Offset {
			shown = append(shown, frame.Lines...)
		}
	}
	if strings.Join(shown, "|") != "a|b" {
		t.Fatalf("the page would show %q", shown)
	}
}

func TestTheReplayCarriesTheTranscriptEvenWhenTheRunCannotBeLoaded(t *testing.T) {
	root := t.TempDir()
	id := "span-1"
	if err := os.MkdirAll(filepath.Join(root, id), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, id, "transcript.log"), []byte("a\nb\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(New(root, "", nil, nil, live.New(), nil))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn := dialLive(t, ctx, srv, id)
	var replay engine.Event
	if err := wsjson.Read(ctx, conn, &replay); err != nil {
		t.Fatal(err)
	}
	if replay.Span != nil || strings.Join(replay.Transcript, "|") != "a|b" || replay.Offset != 4 {
		t.Fatalf("replay %+v", replay)
	}
}

func TestATokenEqualToAProtocolValueNeverRewritesIt(t *testing.T) {
	const token = "transcript"
	root, id := liveRun(t)
	hub := live.New()
	srv := httptest.NewServer(New(root, token, nil, nil, hub, nil))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"/spans/"+id+"/live", &websocket.DialOptions{Subprotocols: []string{bearerProtocol(token)}})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.CloseNow()
	readRawFrame(t, ctx, conn)
	hub.Publish(id, engine.Event{Type: "transcript", Stage: "build", Lines: []string{"the transcript is here"}})
	next := readRawFrame(t, ctx, conn)
	if next["type"] != "transcript" || next["stage"] != "build" {
		t.Fatalf("protocol fields were redacted: %v", next)
	}
	if line := firstString(t, next, "lines"); line != "the [redacted] is here" {
		t.Fatalf("line %q", line)
	}
}
