// Package serve exposes span statements and accepts stage instructions.
// It is the same binary as the engine, bound to localhost or to a token-protected address.
package serve

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"span/internal/agent"
	"span/internal/engine"
	"span/internal/handoff"
	"span/internal/jira"
	"span/internal/launch"
	"span/internal/live"
	"span/internal/scope"
	"span/internal/setup"
)

// ErrToken means a non-loopback listener was asked to start without SPAN_TOKEN.
var ErrToken = errors.New("non-loopback serve requires SPAN_TOKEN")

// Outcome is the quickstart reply. An id means the run is already on disk.
type Outcome struct {
	ID    string
	Stage string
	Text  string
}

// Launch configures a project and, when the request includes a story, starts a run.
type Launch func(ctx context.Context, req setup.Request) (Outcome, error)

// Signal is how a person continues a run from the page.
type Signal struct {
	Answers []engine.Answer
	Reload  bool
	Resume  bool
}

// Wake continues a waiting or paused run. A nil wake leaves those routes absent.
type Wake func(ctx context.Context, id string, signal Signal) error

// Handler serves the runs under root. A non-empty token is required on every request.
func Handler(root, token string) http.Handler {
	return New(root, token, nil, nil, nil, nil)
}

// New is Handler plus an optional quickstart launcher and the static page.
func New(root, token string, launch Launch, wake Wake, hub *live.Hub, pages fs.FS) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /spans", func(w http.ResponseWriter, r *http.Request) {
		if !wantsJSON(r) {
			servePage(pages, w, r)
			return
		}
		ids, _ := list(root)
		writeJSON(w, http.StatusOK, map[string]any{"spans": ids})
	})
	mux.HandleFunc("POST /quickstart", func(w http.ResponseWriter, r *http.Request) {
		req, err := storyRequest(r)
		if err != nil {
			writeErr(w, err)
			return
		}
		if launch != nil {
			out, err := launch(context.Background(), req)
			if err != nil {
				writeErr(w, err)
				return
			}
			if out.ID != "" {
				writeJSON(w, http.StatusAccepted, map[string]string{"id": out.ID, "stage": out.Stage})
				return
			}
			_, _ = io.WriteString(w, out.Text)
			return
		}
		text, err := configureOnly(r.Context(), req)
		if err != nil {
			writeErr(w, err)
			return
		}
		_, _ = io.WriteString(w, text)
	})
	mux.HandleFunc("GET /repos", func(w http.ResponseWriter, r *http.Request) {
		set, err := scope.Discover(r.URL.Query().Get("path"))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, set)
	})
	mux.HandleFunc("GET /jira", func(w http.ResponseWriter, r *http.Request) {
		project := r.URL.Query().Get("path")
		account, err := jira.Load(project)
		writeJSON(w, http.StatusOK, jira.View(account, err == nil))
	})
	mux.HandleFunc("POST /jira/account", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			writeErr(w, err)
			return
		}
		project := r.PostForm.Get("project")
		err := jira.Save(project, jira.Account{
			BaseURL: r.PostForm.Get("url"),
			Email:   r.PostForm.Get("email"),
			Token:   r.PostForm.Get("token"),
		})
		if err != nil {
			writeErr(w, err)
			return
		}
		account, _ := jira.Load(project)
		writeJSON(w, http.StatusOK, jira.View(account, true))
	})
	mux.HandleFunc("POST /jira/logout", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, jira.Logout(r.PostForm.Get("project")))
	})
	mux.HandleFunc("GET /jira/issues/{key}", func(w http.ResponseWriter, r *http.Request) {
		account, err := jira.Load(r.URL.Query().Get("path"))
		if err != nil {
			writeErr(w, err)
			return
		}
		story, err := (jira.Client{Account: account}).Issue(r.Context(), r.PathValue("key"))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, story)
	})
	mux.HandleFunc("GET /jira/search", func(w http.ResponseWriter, r *http.Request) {
		account, err := jira.Load(r.URL.Query().Get("path"))
		if err != nil {
			writeErr(w, err)
			return
		}
		stories, err := (jira.Client{Account: account}).Search(r.Context(), r.URL.Query().Get("jql"))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"issues": stories})
	})
	mux.HandleFunc("GET /spans/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if !wantsJSON(r) {
			servePage(pages, w, r)
			return
		}
		snap, err := engine.Load(root, id)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, snap)
	})
	mux.HandleFunc("GET /spans/{id}/implementation", func(w http.ResponseWriter, r *http.Request) {
		diff, err := readImplementation(r.Context(), root, r.PathValue("id"))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, diff)
	})
	mux.HandleFunc("GET /spans/{id}/acceptance", func(w http.ResponseWriter, r *http.Request) {
		body, err := readAcceptance(root, r.PathValue("id"))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, body)
	})
	mux.HandleFunc("GET /spans/{id}/live", func(w http.ResponseWriter, r *http.Request) {
		if hub == nil {
			http.NotFound(w, r)
			return
		}
		serveLive(hub, root, token, r.PathValue("id"), w, r)
	})
	mux.HandleFunc("POST /spans/{id}/answer", func(w http.ResponseWriter, r *http.Request) {
		continueRun(wake, root, r.PathValue("id"), w, r, false, false)
	})
	mux.HandleFunc("POST /spans/{id}/story", func(w http.ResponseWriter, r *http.Request) {
		continueRun(wake, root, r.PathValue("id"), w, r, true, false)
	})
	mux.HandleFunc("POST /spans/{id}/resume", func(w http.ResponseWriter, r *http.Request) {
		continueRun(wake, root, r.PathValue("id"), w, r, false, true)
	})
	mux.HandleFunc("POST /spans/{id}/relaunch", func(w http.ResponseWriter, r *http.Request) {
		relaunch(wake, hub, root, r.PathValue("id"), w, r)
	})
	mux.HandleFunc("GET /spans/{id}/agent", func(w http.ResponseWriter, r *http.Request) {
		body, err := agentStatus(hub, root, r.PathValue("id"))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, body)
	})
	mux.HandleFunc("GET /agents", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"agents": agentList(hub, root)})
	})
	mux.HandleFunc("POST /spans/{id}/instructions", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		stage := r.PostForm.Get("stage")
		if !safe(id) || !safe(stage) {
			http.Error(w, "bad path", http.StatusBadRequest)
			return
		}
		if err := saveInstruction(root, id, stage, []byte(r.PostForm.Get("body"))); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/spans/"+id, http.StatusSeeOther)
	})
	mux.HandleFunc("POST /spans/{id}/instructions/{stage}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		stage := r.PathValue("stage")
		if !safe(id) || !safe(stage) {
			http.Error(w, "bad path", http.StatusBadRequest)
			return
		}
		raw, err := instructionBody(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := saveInstruction(root, id, stage, raw); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/spans/"+id, http.StatusSeeOther)
	})
	mux.HandleFunc("GET /{path...}", func(w http.ResponseWriter, r *http.Request) {
		servePage(pages, w, r)
	})
	return protect(token, mux)
}

// Listen checks the address policy and serves.
func Listen(addr, root, token string, launch Launch, wake Wake, hub *live.Hub, pages fs.FS) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}
	ip := net.ParseIP(host)
	if (ip == nil || !ip.IsLoopback()) && token == "" {
		return ErrToken
	}
	return http.ListenAndServe(addr, New(root, token, launch, wake, hub, pages))
}

func continueRun(wake Wake, root, id string, w http.ResponseWriter, r *http.Request, reload, resume bool) {
	if wake == nil || !safe(id) {
		http.NotFound(w, r)
		return
	}
	snap, err := engine.Load(root, id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	signal := Signal{Reload: reload, Resume: resume}
	if !reload && !resume {
		var body struct {
			Answers []engine.Answer `json:"answers"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeErr(w, err)
			return
		}
		if len(body.Answers) == 0 {
			writeErr(w, errors.New("an answer is empty"))
			return
		}
		for _, answer := range body.Answers {
			if strings.TrimSpace(answer.Question) == "" || strings.TrimSpace(answer.Body) == "" {
				writeErr(w, errors.New("an answer is empty"))
				return
			}
		}
		signal.Answers = body.Answers
	}
	if resume {
		if engine.Waiting(snap.Cursor) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "invalid", "message": "this run is waiting for an answer"})
			return
		}
		if snap.Cursor.Stop || snap.Cursor.Done || !snap.Cursor.Hold {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "invalid", "message": "this run is not paused"})
			return
		}
	} else if !engine.Waiting(snap.Cursor) {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "invalid", "message": "this run is not waiting"})
		return
	}
	if err := wake(r.Context(), id, signal); err != nil {
		writeBusy(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"id": id, "stage": snap.Cursor.Stage})
}

func relaunch(wake Wake, hub *live.Hub, root, id string, w http.ResponseWriter, r *http.Request) {
	if wake == nil || !safe(id) {
		http.NotFound(w, r)
		return
	}
	snap, err := engine.Load(root, id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if snap.Cursor.Stop || snap.Cursor.Done {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "invalid", "message": "this run will not continue"})
		return
	}
	if engine.Waiting(snap.Cursor) {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "invalid", "message": "this run is waiting for an answer"})
		return
	}
	if snap.Cursor.Hold {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "invalid", "message": "this run is paused"})
		return
	}
	if _, ok := hub.Flight(id); ok {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "invalid", "message": "an agent is already running"})
		return
	}
	if err := wake(r.Context(), id, Signal{Resume: true}); err != nil {
		writeBusy(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"id": id, "stage": snap.Cursor.Stage})
}

func writeBusy(w http.ResponseWriter, err error) {
	if errors.Is(err, live.ErrBusy) {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "invalid", "message": err.Error()})
		return
	}
	writeErr(w, err)
}

type agentBody struct {
	ID       string `json:"id,omitempty"`
	Running  bool   `json:"running"`
	Relaunch bool   `json:"relaunch"`
	Name     string `json:"name,omitempty"`
	PID      int    `json:"pid,omitempty"`
	Stage    string `json:"stage,omitempty"`
	Elapsed  string `json:"elapsed,omitempty"`
	Summary  string `json:"summary"`
}

func agentStatus(hub *live.Hub, root, id string) (agentBody, error) {
	if !safe(id) {
		return agentBody{}, os.ErrNotExist
	}
	snap, err := engine.Load(root, id)
	if err != nil {
		return agentBody{}, err
	}
	return review(hub, id, snap.Cursor), nil
}

func agentList(hub *live.Hub, root string) []agentBody {
	if hub == nil {
		return []agentBody{}
	}
	flights := hub.Flights()
	out := make([]agentBody, 0, len(flights))
	for id, flight := range flights {
		cur := engine.Cursor{ID: id, Stage: flight.Stage}
		if snap, err := engine.Load(root, id); err == nil {
			cur = snap.Cursor
		}
		body := review(hub, id, cur)
		body.ID = id
		out = append(out, body)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func review(hub *live.Hub, id string, cur engine.Cursor) agentBody {
	flight, flying := hub.Flight(id)
	proc, tracked := agent.Current(id)
	if tracked && !agent.Alive(proc.PID) {
		tracked = false
	}
	body := agentBody{Running: flying, Stage: cur.Stage}
	if flying {
		since := flight.Since
		if tracked {
			body.Name = proc.Name
			body.PID = proc.PID
			since = proc.Since
			body.Summary = strings.TrimSpace(proc.Name + " " + elapsed(since))
		} else {
			body.Summary = "working"
		}
		body.Elapsed = elapsed(since)
		return body
	}
	switch {
	case cur.Stop:
		body.Summary = "stopped"
	case cur.Done:
		body.Summary = "finished"
	case cur.Hold || engine.Waiting(cur):
		body.Summary = "paused"
	default:
		body.Summary = "not running"
		body.Relaunch = true
	}
	return body
}

func elapsed(since time.Time) string {
	if since.IsZero() {
		return ""
	}
	return time.Since(since).Truncate(time.Second).String()
}

func saveInstruction(root, id, stage string, raw []byte) error {
	dir := filepath.Join(root, id, "instructions")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, stage+".md"), raw, 0o644)
}

func instructionBody(r *http.Request) ([]byte, error) {
	if strings.HasPrefix(r.Header.Get("Content-Type"), "text/plain") {
		return io.ReadAll(io.LimitReader(r.Body, 1<<20))
	}
	if err := r.ParseForm(); err != nil {
		return nil, err
	}
	return []byte(r.PostForm.Get("body")), nil
}

func protect(token string, next http.Handler) http.Handler {
	if token == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "Bearer "+token || subprotocol(r, token) {
			next.ServeHTTP(w, r)
			return
		}
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})
}

func list(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, entry := range entries {
		if entry.IsDir() {
			ids = append(ids, entry.Name())
		}
	}
	sort.Strings(ids)
	return ids, nil
}

func storyRequest(r *http.Request) (setup.Request, error) {
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		if err := r.ParseMultipartForm(handoff.MaxBytes); err != nil {
			return setup.Request{}, err
		}
	} else if err := r.ParseForm(); err != nil {
		return setup.Request{}, err
	}
	var files []handoff.File
	if r.MultipartForm != nil {
		for _, header := range r.MultipartForm.File["file"] {
			opened, err := header.Open()
			if err != nil {
				return setup.Request{}, err
			}
			body, err := io.ReadAll(io.LimitReader(opened, handoff.MaxBytes+1))
			opened.Close()
			if err != nil {
				return setup.Request{}, err
			}
			files = append(files, handoff.File{Name: header.Filename, Body: body})
		}
	}
	return setup.Request{
		Workspace: r.Form.Get("workspace"),
		Story:     r.Form.Get("story"),
		Kind:      r.Form.Get("kind"),
		Issue:     r.Form.Get("issue"),
		Base:      r.Form.Get("base"),
		Repos:     r.Form["repo"],
		Files:     files,
	}, nil
}

func safe(name string) bool {
	return name != "" && name != "." && name != ".." && !strings.ContainsAny(name, `/\`)
}

func configureOnly(ctx context.Context, req setup.Request) (string, error) {
	targets, err := launch.Resolve(ctx, launch.Request{
		Path: req.Workspace, Names: req.Repos, Issue: req.Issue, Story: req.Story,
	}, nil)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, target := range targets {
		res, err := setup.Configure(setup.Request{
			Workspace: target.Path,
			Story:     target.Story,
			Kind:      req.Kind,
			Issue:     target.Issue,
			Base:      req.Base,
			Force:     req.Force,
		})
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "repo %s\npolicy %s\nbranch %s\nbase %s\n", target.Name, res.PolicyPath, res.Branch, res.Base)
	}
	return b.String(), nil
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeErr(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	code := "invalid"
	var coded interface{ SpanCode() string }
	if errors.As(err, &coded) {
		code = coded.SpanCode()
	}
	switch code {
	case "unauthorized":
		status = http.StatusUnauthorized
	case "not_found":
		status = http.StatusNotFound
	case "unreachable":
		status = http.StatusBadGateway
	}
	writeJSON(w, status, map[string]string{"error": code, "message": err.Error()})
}
