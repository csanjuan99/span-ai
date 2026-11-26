package serve

import (
	"context"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"span/internal/start"
)

const spanVersion = "dev"

// startRoutes wires N1 read APIs and POST /spans.
func startRoutes(mux *http.ServeMux, root string, launch Launch, starts *recent) {
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		h := start.Probe(spanVersion, time.Now())
		writeJSON(w, http.StatusOK, h)
	})
	mux.HandleFunc("GET /start/precheck", func(w http.ResponseWriter, r *http.Request) {
		ref, err := start.Precheck(r.Context(), checkInput(r, root))
		if err != nil {
			writeErr(w, err)
			return
		}
		if ref == nil {
			writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
			return
		}
		writeRefusal(w, ref)
	})
	mux.HandleFunc("GET /start/what-will-happen", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		writeJSON(w, http.StatusOK, start.WhatWillHappen(q.Get("path"), q.Get("kind"), q.Get("base")))
	})
	mux.HandleFunc("GET /start/sealed", func(w http.ResponseWriter, r *http.Request) {
		list, err := start.ListSealed(r.URL.Query().Get("path"))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"stories": list})
	})
	mux.HandleFunc("GET /start/run-key", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		key, err := start.NextRunKey(q.Get("path"), q.Get("shape"), spansRoot(q.Get("path"), root))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, key)
	})
	mux.HandleFunc("POST /spans", func(w http.ResponseWriter, r *http.Request) {
		handleStart(w, r, launch, starts)
	})
}

func handleStart(w http.ResponseWriter, r *http.Request, launch Launch, starts *recent) {
	out, ok := launchOutcome(w, r, launch, starts, writeLaunchErr)
	if !ok {
		return
	}
	body := mapBegin(out)
	if out.ID != "" {
		writeJSON(w, http.StatusAccepted, body)
		return
	}
	writeJSON(w, http.StatusOK, body)
}

// launchOutcome is the flow POST /quickstart and POST /spans share: read the story, only
// configure when nothing can launch, otherwise start the run once per key. It answers the
// request itself, through fail when the launch errors, and reports false when it did, so the caller only maps a successful outcome.
func launchOutcome(w http.ResponseWriter, r *http.Request, launch Launch, starts *recent, fail func(http.ResponseWriter, error)) (Outcome, bool) {
	req, err := storyRequest(r)
	if err != nil {
		writeErr(w, err)
		return Outcome{}, false
	}
	if launch == nil {
		text, err := configureOnly(r.Context(), req)
		if err != nil {
			writeErr(w, err)
			return Outcome{}, false
		}
		writeText(w, text)
		return Outcome{}, false
	}
	run := func() (Outcome, error) { return launch(context.Background(), req) }
	out, err := dedupe(r, req, starts, run)
	if err != nil {
		fail(w, err)
		return Outcome{}, false
	}
	return out, true
}

// writeLaunchErr answers a failed start with its numbered refusal when it has one.
func writeLaunchErr(w http.ResponseWriter, err error) {
	if ref := start.FromError(err); ref != nil && ref.Number != 0 {
		writeRefusal(w, ref)
		return
	}
	writeErr(w, err)
}

// launchBody is the part of a launch response the quickstart and start routes share.
func launchBody(out Outcome) map[string]any {
	body := map[string]any{}
	if out.ID != "" {
		body["id"] = out.ID
		body["stage"] = out.Stage
	}
	if len(out.Runs) > 0 {
		body["runs"] = out.Runs
	}
	if strings.TrimSpace(out.Text) != "" {
		body["notes"] = out.Text
	}
	if out.Partial {
		body["partial"] = true
	}
	if len(out.Failed) > 0 {
		body["failed"] = out.Failed
	}
	return body
}

func mapBegin(out Outcome) map[string]any {
	results := make([]start.RepoResult, 0, len(out.Runs)+len(out.Failed))
	for _, run := range out.Runs {
		results = append(results, start.RepoResult{
			Repo: run.Repo, OK: true, ID: run.ID, Stage: run.Stage,
		})
	}
	for _, failed := range out.Failed {
		results = append(results, start.RepoResult{
			Repo: failed.Repo, Refusal: &start.Refusal{Code: "not_started", Message: failed.Reason},
		})
	}
	body := launchBody(out)
	body["ok"] = len(out.Runs) > 0
	body["results"] = results
	return body
}

func checkInput(r *http.Request, defaultSpans string) start.CheckInput {
	q := r.URL.Query()
	repos := q["repo"]
	if len(repos) == 0 {
		repos = nil
	}
	path := q.Get("path")
	if path == "" {
		path = q.Get("workspace")
	}
	return start.CheckInput{
		Workspace: path,
		Story:     q.Get("story"),
		Kind:      q.Get("kind"),
		Issue:     q.Get("issue"),
		Base:      q.Get("base"),
		Repos:     repos,
		SpansRoot: spansRoot(path, defaultSpans),
	}
}

func spansRoot(path, serveRoot string) string {
	if strings.TrimSpace(serveRoot) != "" {
		return serveRoot
	}
	if strings.TrimSpace(path) == "" {
		path = "."
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return ""
	}
	return filepath.Join(abs, ".span", "runs")
}

func writeRefusal(w http.ResponseWriter, ref *start.Refusal) {
	status := http.StatusConflict
	switch ref.Number {
	case start.NumberTokenRefused:
		status = http.StatusUnauthorized
	case start.NumberNotGit:
		status = http.StatusBadRequest
	case start.NumberJiraSignedOut:
		status = http.StatusUnauthorized
	}
	writeJSON(w, status, ref)
}
