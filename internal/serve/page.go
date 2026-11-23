package serve

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"

	"span/internal/adapt"
	"span/internal/engine"
	"span/internal/live"
	"span/internal/qa"
)

func wantsJSON(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept"), "application/json")
}

func bearerProtocol(token string) string {
	return "span." + base64.RawURLEncoding.EncodeToString([]byte(token))
}

func offeredProtocols(r *http.Request) []string {
	var offered []string
	for _, part := range strings.Split(r.Header.Get("Sec-WebSocket-Protocol"), ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			offered = append(offered, part)
		}
	}
	return offered
}

func subprotocol(r *http.Request, token string) bool {
	want := bearerProtocol(token)
	for _, part := range strings.Split(r.Header.Get("Sec-WebSocket-Protocol"), ",") {
		if subtle.ConstantTimeCompare([]byte(strings.TrimSpace(part)), []byte(want)) == 1 {
			return true
		}
	}
	return false
}

func servePage(pages fs.FS, w http.ResponseWriter, r *http.Request) {
	if pages == nil {
		http.NotFound(w, r)
		return
	}
	name := strings.TrimPrefix(r.URL.Path, "/")
	if name == "" {
		name = "index.html"
	}
	if _, err := fs.Stat(pages, name); err != nil {
		name = "index.html"
	}
	http.ServeFileFS(w, r, pages, name)
}

type acceptanceBody struct {
	Skipped  bool      `json:"skipped"`
	Verdict  string    `json:"verdict,omitempty"`
	Command  string    `json:"command,omitempty"`
	ExitCode int       `json:"exitCode"`
	Lines    []qa.Case `json:"lines"`
}

func readAcceptance(root, id string) (acceptanceBody, error) {
	snap, err := engine.Load(root, id)
	if err != nil {
		return acceptanceBody{}, err
	}
	lines := snap.Acceptance
	if len(lines) == 0 {
		raw, readErr := os.ReadFile(filepath.Join(root, id, "brief.md"))
		if readErr == nil {
			lines = qa.Lines(string(raw))
		}
	}
	if len(lines) == 0 {
		return acceptanceBody{Skipped: true, Lines: []qa.Case{}}, nil
	}
	cases, err := qa.Present(snap.Cursor.Workspace, lines, snap.Gate)
	if err != nil {
		return acceptanceBody{}, err
	}
	body := acceptanceBody{Lines: cases}
	if snap.Gate != nil {
		body.Verdict = snap.Gate.Verdict
		body.Command = snap.Gate.Command
		body.ExitCode = snap.Gate.ExitCode
	}
	return body, nil
}

func readImplementation(ctx context.Context, root, id string) (adapt.Diff, error) {
	snap, err := engine.Load(root, id)
	if err != nil {
		return adapt.Diff{}, err
	}
	if snap.Cursor.Workspace == "" || snap.Cursor.Base == "" || snap.Cursor.GitSHA == "" {
		return adapt.Diff{}, nil
	}
	return (adapt.Git{Dir: snap.Cursor.Workspace}).Diff(ctx, snap.Cursor.Base, snap.Cursor.GitSHA)
}

// liveWriteTimeout bounds one socket write, in nanoseconds; tests shorten it.
var liveWriteTimeout atomic.Int64

func init() { liveWriteTimeout.Store(int64(10 * time.Second)) }

func serveLive(hub *live.Hub, root, token, id string, w http.ResponseWriter, r *http.Request) {
	if !safe(id) {
		http.NotFound(w, r)
		return
	}
	opts := &websocket.AcceptOptions{OriginPatterns: []string{"localhost:*", "127.0.0.1:*"}}
	if token != "" {
		opts.Subprotocols = []string{bearerProtocol(token)}
	} else if offered := offeredProtocols(r); len(offered) > 0 {
		// Chrome drops the handshake when the page offers a protocol and the
		// response does not name one. No SPAN_TOKEN means the value is not a secret.
		opts.Subprotocols = offered
	}
	conn, err := websocket.Accept(w, r, opts)
	if err != nil {
		return
	}
	defer conn.CloseNow()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Subscribe, then read the seq, then load: every frame published after the
	// subscription carries a larger seq than the replay, and the page drops the ones at
	// or below it as already covered by the snapshot.
	frames, unsub := hub.Subscribe(id)
	defer unsub()
	resume := hub.Seq(id)
	// The replay is sent even when the run cannot be loaded: it is the baseline the page
	// compares seq against, and without it a reconnecting page would skip or gap forever.
	// The transcript does not depend on the snapshot, so it is replayed either way.
	replay := engine.Event{Type: "replay", Seq: resume}
	replay.Transcript, replay.Offset = transcriptTail(filepath.Join(root, id, "transcript.log"), 1000)
	if snap, err := engine.Load(root, id); err == nil {
		replay.Span = &snap
	}
	if raw, err := json.Marshal(replay); err == nil {
		if raw, ok := redactFrame(raw, token); ok {
			if err := writeLive(ctx, conn, raw); err != nil {
				return
			}
		}
	}
	go func() {
		for {
			if _, _, err := conn.Read(ctx); err != nil {
				cancel()
				return
			}
		}
	}()
	for {
		select {
		case <-ctx.Done():
			conn.Close(websocket.StatusNormalClosure, "")
			return
		case raw, ok := <-frames:
			if !ok {
				return
			}
			// Cut before redacting: the cut counts the bytes the log wrote.
			clean, valid := redactFrame(trimCovered(raw, replay.Offset), token)
			if !valid {
				continue
			}
			if err := writeLive(ctx, conn, clean); err != nil {
				return
			}
		}
	}
}

// trimCovered drops, from one encoded transcript frame, the lines this connection's
// replay tail already holds. A batch flushed after the replay can begin before it; its
// frame still goes out, with fewer or no lines, so the page sees no seq hole.
func trimCovered(raw []byte, replayOffset int64) []byte {
	if replayOffset <= 0 {
		return raw
	}
	var event engine.Event
	if err := json.Unmarshal(raw, &event); err != nil || event.Type != "transcript" {
		return raw
	}
	cut := cutCovered(event, replayOffset)
	if len(cut.Lines) == len(event.Lines) {
		return raw
	}
	out, err := json.Marshal(cut)
	if err != nil {
		return raw
	}
	return out
}

// cutCovered keeps the lines of a transcript frame that end past replayOffset. The log
// writes each line as its bytes plus one newline from Start, so line i ends at Start plus
// the widths of lines 0..i. A frame without Start is left alone.
func cutCovered(event engine.Event, replayOffset int64) engine.Event {
	if event.Type != "transcript" || event.Start == nil || replayOffset <= *event.Start {
		return event
	}
	end := *event.Start
	var kept []string
	for _, line := range event.Lines {
		end += int64(len(line)) + 1
		if end > replayOffset {
			kept = append(kept, line)
		}
	}
	event.Lines = kept
	return event
}

// writeLive sends one frame with a deadline, so a client that stops reading cannot hold
// this goroutine until TCP gives up.
func writeLive(ctx context.Context, conn *websocket.Conn, raw []byte) error {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(liveWriteTimeout.Load()))
	defer cancel()
	return conn.Write(ctx, websocket.MessageText, raw)
}

// transcriptTail reads the last n lines of the transcript and the byte offset they end
// at. Both come from one read, cut at the last newline, so the tail never holds a byte
// past the offset; a later transcript frame at or below it repeats lines the page has.
func transcriptTail(path string, n int) ([]string, int64) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, 0
	}
	end := strings.LastIndexByte(string(raw), '\n') + 1
	raw = raw[:end]
	offset := int64(end)
	const maxBytes = 1 << 20
	if len(raw) > maxBytes {
		raw = raw[len(raw)-maxBytes:]
		if i := strings.IndexByte(string(raw), '\n'); i >= 0 {
			raw = raw[i+1:]
		}
	}
	text := strings.TrimRight(string(raw), "\n")
	if text == "" {
		return nil, offset
	}
	lines := strings.Split(text, "\n")
	if n > 0 && len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines, offset
}

// minRedactLen is the shortest serve token the socket redacts. A shorter token is too
// likely to be an ordinary word ("type", "done") to replace in run output; such a token
// still guards the socket, it is just not scrubbed from the frames.
const minRedactLen = 8

// redactFrame replaces the serve token inside the string values of one JSON frame. It
// works on decoded values, never on the encoded bytes: keys stay intact, and a token
// that JSON escapes (a quote, a backslash, `<`, `&`) is still found. ok is false when
// the frame cannot be decoded; the caller drops it rather than send it unredacted.
func redactFrame(raw []byte, token string) ([]byte, bool) {
	if len(token) < minRedactLen {
		return raw, true
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, false
	}
	out, err := json.Marshal(redactValue(value, token))
	if err != nil {
		return nil, false
	}
	return out, true
}

// protocolKeys hold the closed values the page routes on (frame type, stage name,
// verdict). A token that happens to equal one of them must not turn it into
// "[redacted]"; these fields never carry run output.
var protocolKeys = map[string]bool{"type": true, "stage": true, "verdict": true}

func redactValue(value any, token string) any {
	switch typed := value.(type) {
	case string:
		return strings.ReplaceAll(typed, token, "[redacted]")
	case []any:
		for i, item := range typed {
			typed[i] = redactValue(item, token)
		}
		return typed
	case map[string]any:
		for key, item := range typed {
			if protocolKeys[key] {
				continue
			}
			typed[key] = redactValue(item, token)
		}
		return typed
	default:
		return value
	}
}
