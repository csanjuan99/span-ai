package serve

import (
	"context"
	"encoding/base64"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"span/internal/adapt"
	"span/internal/engine"
	"span/internal/live"
	"span/internal/qa"
	"span/internal/transcript"
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
		if strings.TrimSpace(part) == want {
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
	frames, unsub := hub.Subscribe(id)
	defer unsub()
	snap, err := engine.Load(root, id)
	if err == nil {
		lines, _ := transcript.Tail(filepath.Join(root, id, "transcript.log"), 1000)
		_ = wsjson.Write(ctx, conn, engine.Event{Type: "replay", Span: &snap, Transcript: lines})
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
			if token != "" {
				raw = scrubToken(raw, token)
			}
			if err := conn.Write(ctx, websocket.MessageText, raw); err != nil {
				return
			}
		}
	}
}

func scrubToken(raw []byte, token string) []byte {
	if token == "" {
		return raw
	}
	return []byte(strings.ReplaceAll(string(raw), token, "[redacted]"))
}
