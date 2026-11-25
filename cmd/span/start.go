package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"span/internal/start"
)

const startUsage = `usage: ./span start precheck [--workspace .] [--story ...] [--issue ...] [--kind ...] [--base ...] [--repo NAME]... [--json]
       ./span start branches [--workspace .] [--json]
       ./span start what-will-happen [--project .] [--kind ...] [--base ...]
       ./span start sealed [--workspace .]
       ./span start run-key [--workspace .] [--shape ID]
       ./span start health [--url http://127.0.0.1:8787] [--json]
       (exit 0 ok, 1 error, 2 usage, 3 refused, 6 run exists, 8 Jira signed out, 14 not a git repository,
        15 base missing, 16 checkout busy, 17 no agent, 18 token refused; text and --json exit alike)`

func startCmd(args []string) int { return startMain(args, os.Stdout, os.Stderr) }

func startMain(args []string, out, errw io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(errw, startUsage)
		return 2
	}
	fs := flag.NewFlagSet("start "+args[0], flag.ContinueOnError)
	workspace := fs.String("workspace", ".", "project folder")
	project := fs.String("project", ".", "project folder")
	story := fs.String("story", "", "story text")
	issue := fs.String("issue", "", "Jira issue key")
	kind := fs.String("kind", "", "feat, fix, or hotfix")
	base := fs.String("base", "", "base branch")
	shapeID := fs.String("shape", "", "sealed shape session id")
	serveURL := fs.String("url", "http://127.0.0.1:8787", "span serve base URL")
	asJSON := fs.Bool("json", false, "print JSON")
	var repos repeated
	fs.Var(&repos, "repo", "repository name; repeat for several")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	emitOne := func(value any) int { return emit(out, errw, value) }
	switch args[0] {
	case "precheck":
		ref, err := start.Precheck(context.Background(), start.CheckInput{
			Workspace: *workspace, Story: *story, Kind: *kind,
			Issue: *issue, Base: *base, Repos: repos,
		})
		if err != nil {
			fmt.Fprintln(errw, err)
			return 1
		}
		if ref == nil {
			if *asJSON {
				return emitOne(map[string]bool{"ok": true})
			}
			fmt.Fprintln(out, "nothing blocks a start")
			return 0
		}
		// Text and JSON exit alike: a script chains on the refusal's number, and --json only
		// changes how the refusal is printed.
		if *asJSON {
			if c := emitOne(ref); c != 0 {
				return c
			}
		} else {
			fmt.Fprintln(errw, ref.Error())
		}
		return start.ExitCode(ref, false)
	case "branches":
		list, err := start.ListBranches(context.Background(), *workspace)
		if err != nil {
			fmt.Fprintln(errw, err)
			return 1
		}
		if *asJSON {
			return emitOne(list)
		}
		for _, name := range list.Branches {
			fmt.Fprintln(out, name)
		}
		return 0
	case "what-will-happen":
		return emitOne(start.WhatWillHappen(*project, *kind, *base))
	case "sealed":
		list, err := start.ListSealed(*workspace)
		if err != nil {
			fmt.Fprintln(errw, err)
			return 1
		}
		return emitOne(map[string]any{"stories": list})
	case "run-key":
		key, err := start.NextRunKey(*workspace, *shapeID, "")
		if err != nil {
			fmt.Fprintln(errw, err)
			return 1
		}
		return emitOne(key)
	case "health":
		body, refused, err := probeHealth(*serveURL, strings.TrimSpace(os.Getenv("SPAN_TOKEN")))
		if err != nil {
			fmt.Fprintln(errw, err)
			return 1
		}
		code := 0
		if refused {
			code = int(start.NumberTokenRefused)
		}
		if *asJSON {
			if c := emitOne(body); c != 0 {
				return c
			}
			return code
		}
		if refused {
			fmt.Fprintln(errw, "token refused")
			return code
		}
		fmt.Fprintf(out, "%s", body.Status)
		return 0
	default:
		fmt.Fprintln(errw, startUsage)
		return 2
	}
}

// probeHealth asks a running server for GET /health. refused is true when the server answered
// 401; a body that is not a health document is an error, never a connected server.
func probeHealth(baseURL, token string) (body start.Health, refused bool, err error) {
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(baseURL, "/")+"/health", nil)
	if err != nil {
		return body, false, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return body, false, err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()
	if resp.StatusCode == http.StatusUnauthorized {
		return start.RefusedHealth(), true, nil
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return start.Health{}, false, fmt.Errorf("health: the server did not answer with a health document: %w", err)
	}
	return body, false, nil
}
