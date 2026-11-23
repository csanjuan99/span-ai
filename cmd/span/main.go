package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"span/internal/adapt"
	"span/internal/engine"
	"span/internal/handoff"
	"span/internal/jira"
	"span/internal/launch"
	"span/internal/live"
	"span/internal/policy"
	"span/internal/scope"
	"span/internal/serve"
	"span/internal/setup"
	"span/web"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "run":
		os.Exit(run(os.Args[2:]))
	case "resume":
		os.Exit(resume(os.Args[2:]))
	case "serve":
		os.Exit(serveCmd(os.Args[2:]))
	case "quickstart":
		os.Exit(quickstartCmd(os.Args[2:]))
	case "repos":
		os.Exit(reposCmd(os.Args[2:]))
	case "jira":
		os.Exit(jiraCmd(os.Args[2:]))
	default:
		usage()
		os.Exit(2)
	}
}

func run(args []string) int {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	workspace := fs.String("workspace", ".", "project to change")
	brief := fs.String("brief", "", "path to the brief")
	spans := fs.String("spans", "", "state directory (default <workspace>/.span/runs)")
	policyPath := fs.String("policy", "", "policy file (default <workspace>/span.yaml)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	opt, err := options(*workspace, *spans, *policyPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	raw, err := os.ReadFile(*brief)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	opt.Brief = string(raw)
	result, err := engine.Start(context.Background(), opt)
	return printResult(result, err)
}

func resume(args []string) int {
	fs := flag.NewFlagSet("resume", flag.ContinueOnError)
	workspace := fs.String("workspace", ".", "project to change")
	spans := fs.String("spans", "", "state directory")
	policyPath := fs.String("policy", "", "policy file")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: span resume <id>")
		return 2
	}
	opt, err := options(*workspace, *spans, *policyPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	result, err := engine.Resume(context.Background(), opt, fs.Arg(0))
	return printResult(result, err)
}

func serveCmd(args []string) int {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	addr := fs.String("addr", "127.0.0.1:8787", "listen address")
	spans := fs.String("spans", ".span/runs", "state directory")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	token := os.Getenv("SPAN_TOKEN")
	hub := live.New()
	pages, err := web.Pages()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	err = serve.Listen(*addr, *spans, token, func(ctx context.Context, req setup.Request) (serve.Outcome, error) {
		return beginRuns(ctx, req, *spans, hub)
	}, wakeRuns(*spans, hub), hub, pages)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func wakeRuns(spans string, hub *live.Hub) serve.Wake {
	return func(ctx context.Context, id string, signal serve.Signal) error {
		snap, err := engine.Load(spans, id)
		if err != nil {
			return err
		}
		opt, err := options(snap.Cursor.Workspace, spans, "")
		if err != nil {
			return err
		}
		opt.Watch = hub
		opt.Issue = snap.Cursor.Issue
		var brief string
		if signal.Reload {
			key := snap.Cursor.Issue
			if key == "" {
				raw, readErr := os.ReadFile(filepath.Join(spans, id, "brief.md"))
				if readErr != nil {
					return readErr
				}
				key = engine.IssueKey(string(raw))
			}
			account, err := jira.Load(snap.Cursor.Workspace)
			if err != nil {
				return err
			}
			story, err := (jira.Client{Account: account}).Issue(ctx, key)
			if err != nil {
				return err
			}
			brief = story.Brief
		}
		if !hub.Claim(id, snap.Cursor.Stage) {
			return live.ErrBusy
		}
		go func() {
			defer hub.EndFlight(id)
			if signal.Resume {
				_, _ = engine.Resume(context.Background(), opt, id)
				return
			}
			if signal.Reload {
				_, _ = engine.Reload(context.Background(), opt, id, brief)
				return
			}
			_, _ = engine.Continue(context.Background(), opt, id, signal.Answers)
		}()
		return nil
	}
}

func quickstartCmd(args []string) int {
	fs := flag.NewFlagSet("quickstart", flag.ContinueOnError)
	workspace := fs.String("workspace", ".", "project to configure")
	story := fs.String("story", "", "user story")
	storyFile := fs.String("story-file", "", "file that holds the user story")
	kind := fs.String("kind", "", "feat, fix, or hotfix")
	issue := fs.String("issue", "", "linked story key, such as AET-1234")
	base := fs.String("base", "", "dev or master")
	force := fs.Bool("force", false, "rewrite span.yaml")
	var repos repeated
	fs.Var(&repos, "repo", "checkout to include; repeat to include more than one")
	var files repeated
	fs.Var(&files, "file", "handoff, image, or other file to attach; repeat to add more")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	text := *story
	if *storyFile != "" {
		raw, err := os.ReadFile(*storyFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		text = string(raw)
	}
	attached, err := readAttached(files)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	out, code := runQuickstart(context.Background(), setup.Request{
		Workspace: *workspace,
		Story:     text,
		Kind:      *kind,
		Issue:     *issue,
		Base:      *base,
		Repos:     repos,
		Files:     attached,
		Force:     *force,
	})
	if code == 0 {
		fmt.Print(out)
	} else {
		fmt.Fprint(os.Stderr, out)
	}
	return code
}

func runQuickstart(ctx context.Context, req setup.Request) (string, int) {
	targets, err := launch.Resolve(ctx, launch.Request{
		Path: req.Workspace, Names: req.Repos, Issue: req.Issue, Story: req.Story,
	}, nil)
	if err != nil {
		return err.Error() + "\n", 1
	}
	if len(req.Files) > 0 {
		for _, target := range targets {
			if strings.TrimSpace(target.Story) == "" {
				return "invalid: files need a story\n", 1
			}
		}
	}
	var b strings.Builder
	code := 0
	for _, target := range targets {
		one := req
		one.Workspace = target.Path
		one.Story = target.Story
		one.Issue = target.Issue
		text, item := runOne(ctx, one)
		fmt.Fprintf(&b, "repo %s\n%s", target.Name, text)
		if item != 0 {
			code = item
		}
	}
	return b.String(), code
}

func beginRuns(ctx context.Context, req setup.Request, spans string, hub *live.Hub) (serve.Outcome, error) {
	targets, err := launch.Resolve(ctx, launch.Request{
		Path: req.Workspace, Names: req.Repos, Issue: req.Issue, Story: req.Story,
	}, nil)
	if err != nil {
		return serve.Outcome{}, err
	}
	if len(req.Files) > 0 {
		for _, target := range targets {
			if strings.TrimSpace(target.Story) == "" {
				return serve.Outcome{}, fmt.Errorf("invalid: files need a story")
			}
		}
	}
	var text strings.Builder
	var id, stage string
	for _, target := range targets {
		one := req
		one.Workspace = target.Path
		one.Story = target.Story
		one.Issue = target.Issue
		if strings.TrimSpace(one.Story) == "" {
			res, err := setup.Configure(one)
			if err != nil {
				return serve.Outcome{}, err
			}
			fmt.Fprintf(&text, "repo %s\npolicy %s\nbranch %s\nbase %s\nconfiguration ready\n", target.Name, res.PolicyPath, res.Branch, res.Base)
			continue
		}
		gotID, gotStage, err := beginOne(one, spans, hub)
		if err != nil {
			return serve.Outcome{}, err
		}
		if id == "" {
			id, stage = gotID, gotStage
		}
	}
	return serve.Outcome{ID: id, Stage: stage, Text: text.String()}, nil
}

func beginOne(req setup.Request, spans string, hub *live.Hub) (string, string, error) {
	res, err := setup.Configure(req)
	if err != nil {
		return "", "", err
	}
	opt, err := options(req.Workspace, spans, "")
	if err != nil {
		return "", "", err
	}
	opt.Brief = req.Story
	opt.Issue = req.Issue
	opt.Branch = res.Branch
	opt.Base = res.Base
	opt.Files = req.Files
	opt.Watch = hub
	id, err := engine.Open(opt)
	if err != nil {
		return "", "", err
	}
	if !hub.Claim(id, "intake") {
		return "", "", live.ErrBusy
	}
	go func() {
		defer hub.EndFlight(id)
		_, _ = engine.Resume(context.Background(), opt, id)
	}()
	return id, "intake", nil
}

func runOne(ctx context.Context, req setup.Request) (string, int) {
	res, err := setup.Configure(req)
	if err != nil {
		return err.Error() + "\n", 1
	}
	var b strings.Builder
	fmt.Fprintf(&b, "policy %s\n", res.PolicyPath)
	if res.Wrote {
		fmt.Fprintf(&b, "checks %s\n", strings.Join(res.Checks, ", "))
	}
	if strings.TrimSpace(req.Story) == "" {
		b.WriteString("configuration ready\n")
		return b.String(), 0
	}
	fmt.Fprintf(&b, "branch %s from %s\n", res.Branch, res.Base)
	opt, err := options(req.Workspace, "", "")
	if err != nil {
		return b.String() + err.Error() + "\n", 1
	}
	opt.Brief = req.Story
	opt.Issue = req.Issue
	opt.Branch = res.Branch
	opt.Base = res.Base
	opt.Files = req.Files
	result, err := engine.Start(ctx, opt)
	if err != nil {
		return b.String() + err.Error() + "\n", 1
	}
	fmt.Fprintf(&b, "%s %s %s\n%s\n", result.ID, result.Stage, result.Verdict, result.Reason)
	if result.Verdict == "advance" {
		return b.String(), 0
	}
	return b.String(), 1
}

type repeated []string

func (r *repeated) String() string { return strings.Join(*r, ",") }

func (r *repeated) Set(value string) error {
	*r = append(*r, value)
	return nil
}

func reposCmd(args []string) int {
	fs := flag.NewFlagSet("repos", flag.ContinueOnError)
	path := fs.String("path", ".", "folder to read")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	set, err := scope.Discover(*path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Printf("%s %s\n", set.Mode, set.Root)
	for _, repo := range set.Repos {
		fmt.Printf("%s %s\n", repo.Name, repo.Path)
	}
	if len(set.Repos) > 1 {
		fmt.Println("select with --repo")
	}
	return 0
}

func jiraCmd(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: ./span jira configure|status|logout|issue|search [--project .]")
		return 2
	}
	switch args[0] {
	case "status":
		_, project, code := jiraFlags("jira status", args[1:], nil)
		if code != 0 {
			return code
		}
		account, err := jira.Load(project)
		view := jira.View(account, err == nil)
		fmt.Printf("configured %t\nproject %s\nsite %s\nemail %s\nauth %s\n", view.Configured, project, view.BaseURL, view.Email, view.Auth)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	case "logout":
		_, project, code := jiraFlags("jira logout", args[1:], nil)
		if code != 0 {
			return code
		}
		if _, err := jira.Path(project); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		view := jira.Logout(project)
		if view.Configured {
			fmt.Printf("still logged in as %s at %s\n", view.Email, view.BaseURL)
			return 1
		}
		fmt.Printf("signed out of %s\n", project)
		return 0
	case "configure":
		var site, email, token string
		_, project, code := jiraFlags("jira configure", args[1:], func(fs *flag.FlagSet) {
			fs.StringVar(&site, "url", "", "jira site, such as https://example.atlassian.net")
			fs.StringVar(&email, "email", "", "atlassian account email; empty uses a bearer token")
			fs.StringVar(&token, "token", "", "api token")
		})
		if code != 0 {
			return code
		}
		if err := jira.Save(project, jira.Account{BaseURL: site, Email: email, Token: token}); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Printf("saved %s\n", project)
		return 0
	case "issue":
		fs, project, code := jiraFlags("jira issue", args[1:], nil)
		if code != 0 {
			return code
		}
		if fs.NArg() != 1 {
			fmt.Fprintln(os.Stderr, "usage: ./span jira issue [--project .] AET-1234")
			return 2
		}
		account, err := jira.Load(project)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		story, err := (jira.Client{Account: account}).Issue(context.Background(), fs.Arg(0))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Print(story.Brief)
		return 0
	case "search":
		var jql string
		_, project, code := jiraFlags("jira search", args[1:], func(fs *flag.FlagSet) {
			fs.StringVar(&jql, "jql", "", "query; empty lists your recent issues")
		})
		if code != 0 {
			return code
		}
		account, err := jira.Load(project)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		stories, err := (jira.Client{Account: account}).Search(context.Background(), jql)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		for _, story := range stories {
			fmt.Printf("%s %s\n", story.Key, story.Summary)
		}
		return 0
	default:
		fmt.Fprintln(os.Stderr, "usage: ./span jira configure|status|logout|issue|search [--project .]")
		return 2
	}
}

func jiraFlags(name string, args []string, extra func(*flag.FlagSet)) (*flag.FlagSet, string, int) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	project := fs.String("project", ".", "folder that owns this Jira login")
	if extra != nil {
		extra(fs)
	}
	if err := fs.Parse(args); err != nil {
		return nil, "", 2
	}
	return fs, *project, 0
}

func options(workspace, spans, policyPath string) (engine.Options, error) {
	workspace, err := filepath.Abs(workspace)
	if err != nil {
		return engine.Options{}, err
	}
	if spans == "" {
		spans = filepath.Join(workspace, ".span", "runs")
	}
	if policyPath == "" {
		policyPath = filepath.Join(workspace, "span.yaml")
	}
	file, err := policy.Load(policyPath)
	if err != nil {
		return engine.Options{}, err
	}
	opt := engine.Options{
		Workspace: workspace,
		Root:      spans,
		Policy:    file,
		Agents:    &adapt.CLIAgents{},
		Check:     adapt.Exec{},
		Source:    adapt.Git{Dir: workspace},
		Review:    adapt.Host{Workspace: workspace, State: spans},
		Release:   adapt.CommandRelease{},
	}
	if account, err := jira.Load(workspace); err == nil {
		opt.Issues = jira.Client{Account: account}
		if account.Token != "" {
			opt.Secrets = []string{account.Token}
		}
	}
	return opt, nil
}

func readAttached(paths []string) ([]handoff.File, error) {
	out := make([]handoff.File, 0, len(paths))
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		if info.Size() > handoff.MaxBytes {
			return nil, fmt.Errorf("invalid: %s is larger than 32MiB", path)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		out = append(out, handoff.File{Name: filepath.Base(path), Body: raw})
	}
	return out, nil
}

func printResult(result engine.Result, err error) int {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Printf("%s %s %s\n%s\n", result.ID, result.Stage, result.Verdict, result.Reason)
	if result.Verdict == "advance" {
		return 0
	}
	return 1
}

func usage() {
	fmt.Fprintln(os.Stderr, `./span repos [--path .]
./span jira configure --project . --url https://example.atlassian.net --email you@example.com --token <token>
./span jira status [--project .]
./span jira logout [--project .]
./span jira issue [--project .] AET-1234
./span jira search [--project .] [--jql "project = AET"]
./span quickstart [--workspace .] [--repo name] [--file path] [--story "..."] [--story-file story.md] [--kind feat|fix|hotfix] [--issue AET-1234] [--base dev|master]
./span run --brief brief.md [--workspace .]
./span resume <id> [--workspace .]
./span serve [--addr 127.0.0.1:8787] [--spans .span/runs]`)
}
