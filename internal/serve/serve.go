// Package serve exposes span statements and accepts stage instructions.
// It is the same binary as the engine, bound to localhost or to a token-protected address.
package serve

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"span/internal/agent"
	"span/internal/branch"
	"span/internal/decision"
	"span/internal/engine"
	"span/internal/handoff"
	"span/internal/jira"
	"span/internal/launch"
	"span/internal/live"
	"span/internal/overlay"
	"span/internal/policy"
	"span/internal/runlock"
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
	// Runs lists every run started, one per repo. ID and Stage name the first so a client
	// that follows one run keeps working.
	Runs []RunRef
	// Partial means some repos started and some did not; Text says which.
	Partial bool
}

// RunRef is one run a submission started.
type RunRef struct {
	ID    string `json:"id"`
	Stage string `json:"stage"`
	Repo  string `json:"repo,omitempty"`
}

// Launch configures a project and, when the request includes a story, starts a run.
type Launch func(ctx context.Context, req setup.Request) (Outcome, error)

// Signal is how a person continues a run from the page.
type Signal struct {
	Answers []engine.Answer
	// Decision answers the run's open decision request.
	Decision *decision.Answer
	Reload   bool
	Resume   bool
}

// Wake continues a waiting or paused run. A nil wake leaves those routes absent.
type Wake func(ctx context.Context, id string, signal Signal) error

// Handler serves the runs under root. A non-empty token is required on every request.
func Handler(root, token string) http.Handler {
	return New(root, token, nil, nil, nil, nil)
}

// New is Handler plus an optional quickstart launcher and the static page.
func New(root, token string, launch Launch, wake Wake, hub *live.Hub, pages fs.FS, opts ...Option) http.Handler {
	starts := newRecent()
	mux := http.NewServeMux()
	shapeRoutes(mux, newShapeConfig(opts))
	mux.HandleFunc("GET /spans", func(w http.ResponseWriter, r *http.Request) {
		if !wantsJSON(r) {
			servePage(pages, w, r)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"spans": engine.Heads(root)})
	})
	mux.HandleFunc("GET /metrics", func(w http.ResponseWriter, r *http.Request) {
		if !wantsJSON(r) {
			servePage(pages, w, r)
			return
		}
		writeJSON(w, http.StatusOK, engine.CollectMetrics(root))
	})
	mux.HandleFunc("POST /quickstart", func(w http.ResponseWriter, r *http.Request) {
		req, err := storyRequest(r)
		if err != nil {
			writeErr(w, err)
			return
		}
		if launch != nil {
			start := func() (Outcome, error) { return launch(context.Background(), req) }
			out, err := dedupe(r, req, starts, start)
			if err != nil {
				writeErr(w, err)
				return
			}
			if out.ID != "" {
				body := map[string]any{"id": out.ID, "stage": out.Stage}
				if len(out.Runs) > 0 {
					body["runs"] = out.Runs
				}
				if strings.TrimSpace(out.Text) != "" {
					body["notes"] = out.Text
				}
				if out.Partial {
					body["partial"] = true
				}
				writeJSON(w, http.StatusAccepted, body)
				return
			}
			writeText(w, out.Text)
			return
		}
		text, err := configureOnly(r.Context(), req)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeText(w, text)
	})
	mux.HandleFunc("GET /repos", func(w http.ResponseWriter, r *http.Request) {
		set, err := scope.Discover(r.URL.Query().Get("path"))
		if err != nil {
			// A folder with no checkout is an empty list, not a failed request. The shaping
			// page asks for "." and for folders that are not repositories yet.
			var noRepo *scope.Error
			if errors.As(err, &noRepo) && noRepo.SpanCode() == "no_repo" && set.Root != "" {
				if set.Repos == nil {
					set.Repos = []scope.Repo{}
				}
				writeJSON(w, http.StatusOK, set)
				return
			}
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, set)
	})
	mux.HandleFunc("GET /branches", handleBranches)
	mux.HandleFunc("GET /onboarding/folder", handleFolder)
	mux.HandleFunc("GET /onboarding/folders", handleFolders)
	mux.HandleFunc("POST /jira/check", handleJiraCheck)
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
		account, err := jira.Load(project)
		if err != nil {
			log.Printf("jira account %s: %v", project, err)
		}
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
			writeErr(w, &requestError{"not_found", "run " + id + " is not on this desk"})
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
	registerImplementation(mux, root)
	mux.HandleFunc("GET /spans/{id}/feature", func(w http.ResponseWriter, r *http.Request) {
		body, err := readFeature(root, r.PathValue("id"))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, body)
	})
	mux.HandleFunc("GET /spans/{id}/log", func(w http.ResponseWriter, r *http.Request) {
		serveRunLog(root, w, r)
	})
	mux.HandleFunc("GET /spans/{id}/events", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if !safe(id) {
			http.NotFound(w, r)
			return
		}
		view, err := readEvents(root, id)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, view)
	})
	mux.HandleFunc("GET /spans/{id}/acceptance", func(w http.ResponseWriter, r *http.Request) {
		body, err := readAcceptance(root, r.PathValue("id"))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, body)
	})
	qaRoutes(mux, root)
	transcriptRoutes(mux, root)
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
	mux.HandleFunc("POST /spans/{id}/decide", func(w http.ResponseWriter, r *http.Request) {
		decideRun(wake, root, r.PathValue("id"), w, r)
	})
	mux.HandleFunc("POST /spans/{id}/story", func(w http.ResponseWriter, r *http.Request) {
		continueRun(wake, root, r.PathValue("id"), w, r, true, false)
	})
	mux.HandleFunc("POST /spans/{id}/agent", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if err := r.ParseForm(); err != nil {
			writeErr(w, err)
			return
		}
		if !safe(id) {
			writeErr(w, errors.New("invalid span id"))
			return
		}
		if err := engine.SetStageAgent(filepath.Join(root, id), r.PostForm.Get("stage"), r.PostForm.Get("agent"), agent.Home()); err != nil {
			writeErr(w, err)
			return
		}
		snap, err := engine.Load(root, id)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, snap)
	})
	mux.HandleFunc("POST /spans/{id}/stop", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if !safe(id) {
			writeErr(w, errors.New("invalid span id"))
			return
		}
		// A click here is always valid: the answer says whether there was anything to stop.
		// A 4xx would read as "the button is broken" when the run simply is not in flight.
		if hub == nil || !hub.Stop(id) {
			writeJSON(w, http.StatusOK, map[string]any{"stopped": false, "reason": "nothing is running this span"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"stopped": true, "reason": "the stage in flight was cancelled; the run holds with the error"})
	})
	mux.HandleFunc("POST /spans/{id}/delete", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if !safe(id) {
			writeErr(w, errors.New("invalid span id"))
			return
		}
		// A run flying in this process holds no file lock between stages, so the hub is asked too.
		if _, flying := hub.Flight(id); flying {
			writeErr(w, &runlock.BusyError{Scope: runlock.Run, Holder: "this server"})
			return
		}
		if err := engine.Delete(root, id); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"deleted": true, "id": id})
	})
	mux.HandleFunc("POST /spans/{id}/rewind", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if err := r.ParseForm(); err != nil {
			writeErr(w, err)
			return
		}
		stage := r.PostForm.Get("stage")
		if !safe(id) || !safe(stage) {
			writeErr(w, errors.New("invalid span or stage"))
			return
		}
		if hub != nil {
			if _, running := hub.Flight(id); running {
				writeErr(w, errors.New("this span is running; stop it before restarting from a stage"))
				return
			}
		}
		if note := strings.TrimSpace(r.PostForm.Get("note")); note != "" {
			if err := saveInstruction(root, id, stage, []byte(note)); err != nil {
				writeErr(w, err)
				return
			}
		}
		if err := engine.Rewind(root, id, stage); err != nil {
			writeErr(w, err)
			return
		}
		snap, err := engine.Load(root, id)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, snap)
	})
	mux.HandleFunc("POST /spans/{id}/pause", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if hub != nil {
			if _, running := hub.Flight(id); !running {
				// A pause is consumed at a stage boundary. With nothing in flight there is
				// no boundary coming, and the file would wait until a resume discarded it.
				writeJSON(w, http.StatusOK, map[string]any{
					"paused": false,
					"reason": "this span is not running; a pause only applies to a stage in flight",
				})
				return
			}
		}
		if err := engine.RequestPause(root, id); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"paused": true, "reason": "the run holds before the next stage"})
	})
	mux.HandleFunc("POST /spans/{id}/resume", func(w http.ResponseWriter, r *http.Request) {
		continueRun(wake, root, r.PathValue("id"), w, r, false, true)
	})
	mux.HandleFunc("POST /spans/{id}/approve", func(w http.ResponseWriter, r *http.Request) {
		approveStage(wake, root, r.PathValue("id"), w, r)
	})
	conflictRoutes(mux, root, wake)
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
	mux.HandleFunc("GET /config", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, configView(r.URL.Query().Get("path")))
	})
	mux.HandleFunc("POST /config/preview", previewConfig)
	mux.HandleFunc("POST /config", saveConfig)
	mux.HandleFunc("GET /agent", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, agentView(r.URL.Query().Get("path")))
	})
	mux.HandleFunc("POST /agent", func(w http.ResponseWriter, r *http.Request) {
		project, path, ok := agentPolicyPath(w, r)
		if !ok {
			return
		}
		if err := policy.SetAgent(path, r.PostForm.Get("agent")); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, agentView(project))
	})
	mux.HandleFunc("GET /agent/models", func(w http.ResponseWriter, r *http.Request) {
		view, err := modelsOf(r.Context(), r.URL.Query().Get("cli"))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, view)
	})
	mux.HandleFunc("POST /agent/model", func(w http.ResponseWriter, r *http.Request) {
		project, path, ok := agentPolicyPath(w, r)
		if !ok {
			return
		}
		cli, model := r.PostForm.Get("cli"), strings.TrimSpace(r.PostForm.Get("model"))
		// Clearing a model is always allowed: a CLI that stopped being ready must not keep a
		// saved model the person can no longer remove.
		if model != "" {
			if err := canChooseModel(cli); err != nil {
				writeErr(w, err)
				return
			}
		}
		if err := policy.SetModel(path, cli, model); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, agentView(project))
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
	// Every route that changes state is a POST, and a browser tab on any site can send one
	// to 127.0.0.1. Refuse the ones the browser marks as cross-site; clients that send no
	// Origin or Sec-Fetch-Site (curl, the CLI) pass.
	return protect(token, http.NewCrossOriginProtection().Handler(journalled(root, mux)))
}

// Listen checks the address policy and serves.
func Listen(addr, root, token string, launch Launch, wake Wake, hub *live.Hub, pages fs.FS, opts ...Option) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}
	ip := net.ParseIP(host)
	if (ip == nil || !ip.IsLoopback()) && token == "" {
		return ErrToken
	}
	handler := New(root, token, launch, wake, hub, pages, opts...)
	if token == "" {
		// Without a token the loopback bind is the only guard, and a page that rebinds its
		// own DNS name to 127.0.0.1 looks same-origin to the check in New.
		handler = loopbackOnly(handler)
	}
	return http.ListenAndServe(addr, handler)
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
		if snap.Cursor.Decision != nil {
			writeConflict(w, "this run is waiting for a decision")
			return
		}
		if engine.Waiting(snap.Cursor) {
			writeConflict(w, "this run is waiting for an answer")
			return
		}
		if snap.Cursor.Stop || snap.Cursor.Done || !snap.Cursor.Hold {
			writeConflict(w, "this run is not paused")
			return
		}
	} else if !engine.Waiting(snap.Cursor) {
		writeConflict(w, "this run is not waiting")
		return
	}
	if err := wake(r.Context(), id, signal); err != nil {
		writeBusy(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"id": id, "stage": snap.Cursor.Stage})
}

// writeConflict answers a request the run's state cannot take, with the reason.
func writeConflict(w http.ResponseWriter, message string) {
	writeJSON(w, http.StatusConflict, map[string]string{"error": "invalid", "message": message})
}

// decideRun answers the decision a run is waiting on. An answer that names no option of
// that decision is refused before anything is woken.
func decideRun(wake Wake, root, id string, w http.ResponseWriter, r *http.Request) {
	if wake == nil || !safe(id) {
		http.NotFound(w, r)
		return
	}
	snap, err := engine.Load(root, id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	req := snap.Cursor.Decision
	if req == nil {
		writeConflict(w, "this run has no open decision")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	if err := r.ParseForm(); err != nil {
		writeErr(w, err)
		return
	}
	answer := decision.Answer{ID: r.PostForm.Get("decision"), Option: r.PostForm.Get("option"), Text: r.PostForm.Get("text")}
	if answer.ID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid", "message": "an answer names the decision it is for"})
		return
	}
	if _, err := req.Choose(answer); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid", "message": err.Error()})
		return
	}
	if err := wake(r.Context(), id, Signal{Decision: &answer}); err != nil {
		writeBusy(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"id": id, "stage": snap.Cursor.Stage})
}

// approveStage releases a stage whose mode is hold and resumes the run. The approval is
// recorded before the wake, so the stage that runs finds it.
func approveStage(wake Wake, root, id string, w http.ResponseWriter, r *http.Request) {
	if wake == nil || !safe(id) {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		writeErr(w, err)
		return
	}
	if _, err := engine.Load(root, id); err != nil {
		http.NotFound(w, r)
		return
	}
	if err := engine.Approve(root, id, r.PostForm.Get("stage"), r.PostForm.Get("environment")); err != nil {
		writeConflict(w, err.Error())
		return
	}
	if err := wake(r.Context(), id, Signal{Resume: true}); err != nil {
		writeBusy(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"id": id, "stage": r.PostForm.Get("stage")})
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
		writeConflict(w, "this run will not continue")
		return
	}
	if engine.Waiting(snap.Cursor) {
		writeConflict(w, "this run is waiting for an answer")
		return
	}
	// An agent that cannot run is a hold, and the board's Relaunch starts that stage again.
	// Every other hold stays paused until Resume, so a waiting answer is not skipped.
	if snap.Cursor.Hold && snap.Cursor.AgentBlock == nil {
		writeConflict(w, "this run is paused")
		return
	}
	if _, ok := hub.Flight(id); ok {
		writeConflict(w, "an agent is already running")
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
		writeConflict(w, err.Error())
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
		// The agent-blocked board offers Relaunch beside Resume. Other holds do not.
		if cur.AgentBlock != nil {
			body.Relaunch = true
		}
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
	if err := os.WriteFile(filepath.Join(dir, stage+".md"), raw, 0o644); err != nil {
		return err
	}
	return logNote(filepath.Join(root, id), stage, string(raw))
}

// logNote keeps every version of a note in the run's decision log. The stage's file holds
// only the latest, so without this a second note would erase the first for good. A version
// equal to the last one saved for the stage adds nothing.
func logNote(runDir, stage, text string) error {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	entries, err := overlay.Read(runDir)
	if err != nil {
		return err
	}
	for i := len(entries) - 1; i >= 0; i-- {
		if entries[i].Kind == overlay.Note && entries[i].Stage == stage && entries[i].Option == "" {
			if entries[i].Text == text {
				return nil
			}
			break
		}
	}
	_, err = overlay.Append(runDir, overlay.Entry{Kind: overlay.Note, Stage: stage, Title: "note for " + stage, Text: text})
	return err
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

// loopbackOnly refuses a request whose Host names anything but this machine. A browser
// sends the name the page was loaded from, so a DNS-rebinding page arrives as its own
// domain and is stopped here.
func loopbackOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
		host = strings.ToLower(strings.TrimSuffix(strings.Trim(host, "[]"), "."))
		if ip := net.ParseIP(host); host == "localhost" || (ip != nil && ip.IsLoopback()) {
			next.ServeHTTP(w, r)
			return
		}
		http.Error(w, "forbidden host", http.StatusForbidden)
	})
}

func protect(token string, next http.Handler) http.Handler {
	if token == "" {
		return next
	}
	want := []byte("Bearer " + token)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := []byte(r.Header.Get("Authorization"))
		if subtle.ConstantTimeCompare(got, want) == 1 || subprotocol(r, token) {
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
			if err := errors.Join(err, opened.Close()); err != nil {
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
		project, err := policy.ProjectBase(target.Path)
		if err != nil {
			return "", err
		}
		kind, base, err := branch.Choose(target.Story, req.Kind, req.Base, project)
		if err != nil {
			return "", err
		}
		res, err := setup.Configure(setup.Request{
			Workspace: target.Path,
			Story:     target.Story,
			Kind:      kind,
			Issue:     target.Issue,
			Base:      base,
			Force:     req.Force,
		})
		if err != nil {
			return "", err
		}
		b.WriteString(fmt.Sprintf("repo %s\npolicy %s\nbranch %s\nbase %s\n", target.Name, res.PolicyPath, res.Branch, res.Base))
	}
	return b.String(), nil
}

// agentPolicyPath reads the form of a request that writes span.yaml and returns the project
// and the policy path. span.yaml is optional, so a write may create it, but only inside a
// project folder that exists: an empty path would write next to the server.
func agentPolicyPath(w http.ResponseWriter, r *http.Request) (string, string, bool) {
	if err := r.ParseForm(); err != nil {
		writeErr(w, err)
		return "", "", false
	}
	project := r.PostForm.Get("project")
	if info, err := os.Stat(project); project == "" || err != nil || !info.IsDir() {
		writeErr(w, fmt.Errorf("project %q is not a folder", project))
		return "", "", false
	}
	path := filepath.Join(project, policy.Name)
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) && !spanProject(project) {
		writeErr(w, fmt.Errorf("%s is not a git repository or a Span project, so span.yaml is not created there", project))
		return "", "", false
	}
	return project, path, true
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("write response: %v", err)
	}
}

// writeText sends a plain-text reply. The client is the only one who could be told that
// the write failed, so the failure goes to the server log.
func writeText(w http.ResponseWriter, text string) {
	if _, err := io.WriteString(w, text); err != nil {
		log.Printf("write response: %v", err)
	}
}

// spanProject reports whether dir is a git repository (.git, a folder or a worktree's file)
// or already holds Span state (.span/). Only such a folder may get a new span.yaml, so the
// agent picker cannot leave one in a home directory.
func spanProject(dir string) bool {
	if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
		return true
	}
	info, err := os.Stat(filepath.Join(dir, ".span"))
	return err == nil && info.IsDir()
}

func writeErr(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	code := "invalid"
	var coded interface{ SpanCode() string }
	if errors.As(err, &coded) {
		code = coded.SpanCode()
	}
	var busy *runlock.BusyError
	if errors.As(err, &busy) {
		code = "busy"
	}
	switch code {
	case "busy":
		status = http.StatusConflict
	case "idempotency_mismatch":
		status = http.StatusUnprocessableEntity
	case "unauthorized":
		status = http.StatusUnauthorized
	case "not_found", "shape_not_found", "shape_source_not_found", "shape_revision_not_found", "shape_step_not_found", "shape_share_not_found":
		status = http.StatusNotFound
	case "shape_share_expired", "shape_share_revoked":
		status = http.StatusGone
	case "shape_not_ready", "shape_not_sealed", "shape_sealed", "shape_repo_changed", "shape_context_written", "shape_no_split",
		"jira_backlog_disabled", "jira_board_ambiguous", "jira_not_in_backlog", "shape_conflicts_open",
		"shape_not_undoable", "shape_has_dependents", "shape_context_changed":
		status = http.StatusConflict
	case "unreachable":
		status = http.StatusBadGateway
	}
	writeJSON(w, status, map[string]string{"error": code, "message": err.Error()})
}

func dedupe(r *http.Request, req setup.Request, starts *recent, start func() (Outcome, error)) (Outcome, error) {
	key, window, fingerprint := launchKey(r, req)
	if key == "" {
		return start()
	}
	return starts.do(r.Context(), key, fingerprint, window, start)
}
