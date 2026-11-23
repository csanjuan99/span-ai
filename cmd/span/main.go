package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"span/internal/adapt"
	"span/internal/branch"
	"span/internal/engine"
	"span/internal/handoff"
	"span/internal/jira"
	"span/internal/launch"
	"span/internal/live"
	"span/internal/policy"
	"span/internal/proc"
	"span/internal/runlock"
	"span/internal/scope"
	"span/internal/serve"
	"span/internal/setup"
	"span/internal/worktree"
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

// runFlags are the flags of span run.
type runFlags struct {
	workspace, brief, spans, policy, branch, base string
	isolate                                       bool
}

func run(args []string) int {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	var f runFlags
	fs.StringVar(&f.workspace, "workspace", ".", "project to change")
	fs.StringVar(&f.brief, "brief", "", "path to the brief")
	fs.StringVar(&f.spans, "spans", "", "state directory (default <workspace>/.span/runs)")
	fs.StringVar(&f.policy, "policy", "", "policy file (default <workspace>/span.yaml)")
	fs.StringVar(&f.branch, "branch", "", "feat/, fix/ or hotfix/ branch to work on, cut from the base when new; with --worktree it gets its own tree")
	fs.StringVar(&f.base, "base", "", "branch to start from and open the change into (default: base in span.yaml for feat/fix, else dev; master for hotfix/)")
	fs.BoolVar(&f.isolate, "worktree", false, "isolate the run in its own git worktree")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	f, note, err := isolateWhenBusy(f)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if note != "" {
		fmt.Fprintln(os.Stderr, note)
	}
	opt, err := runOptions(f)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	ctx, stop := interruptContext()
	defer stop()
	result, err := engine.Start(ctx, opt)
	return printResult(result, err)
}

// runOptions turns span run's flags into the run. --branch is the branch the run works on,
// with or without --worktree: the engine checks it out, cut from the base when it is new.
// Without --branch the run works on the checked-out branch, and that branch's prefix
// decides the default base, so a hotfix/ checkout opens into master. The change always goes
// to the base, never to whatever the host calls its default branch.
func runOptions(f runFlags) (engine.Options, error) {
	code, err := prepare(f.workspace, f.spans, f.policy, f.branch, f.base, f.isolate)
	if err != nil {
		return engine.Options{}, err
	}
	opt, err := options(code.Workspace, code.Spans, code.Policy, code.Project, code.Explicit)
	if err != nil {
		return engine.Options{}, err
	}
	raw, err := os.ReadFile(f.brief)
	if err != nil {
		return engine.Options{}, err
	}
	opt.Brief = string(raw)
	name := f.branch
	if name == "" {
		name, err = adapt.Git{Dir: code.Workspace}.Branch(context.Background())
		if err != nil {
			return engine.Options{}, fmt.Errorf("the checked-out branch could not be read: %w", err)
		}
	}
	opt.Base, err = runBase(f.base, code.Project, name)
	if err != nil {
		return engine.Options{}, err
	}
	opt.Branch = f.branch
	return opt, nil
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
	project := *workspace
	tree, policyFile := *workspace, *policyPath
	// Without --spans the runs are in the workspace, as options() reads them. Looking in ""
	// found nothing, so an isolated run resumed in the project instead of its own worktree.
	root := *spans
	if root == "" {
		root = filepath.Join(*workspace, ".span", "runs")
	}
	if snap, err := engine.Load(root, fs.Arg(0)); err == nil && snap.Cursor.Project != "" {
		project = snap.Cursor.Project
		// An isolated run is resumed in its own worktree, not in the project.
		if snap.Cursor.Workspace != "" {
			tree = snap.Cursor.Workspace
		}
		if policyFile == "" {
			policyFile = projectPolicy(project)
		}
		// The run's state stays where it was found, not under its worktree.
		*spans = root
	}
	if policyFile == "" {
		policyFile = resolvePolicy(*spans, tree, "")
	}
	opt, err := options(tree, *spans, policyFile, project, *policyPath != "")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	ctx, stop := interruptContext()
	defer stop()
	result, err := engine.Resume(ctx, opt, fs.Arg(0))
	return printResult(result, err)
}

func serveCmd(args []string) int {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	addr := fs.String("addr", "127.0.0.1:8787", "listen address")
	spans := fs.String("spans", ".span/runs", "state directory, relative to where serve starts (not to a workspace)")
	maxParallel := fs.Int("max-parallel", 0, "runs driven at once; more are refused until one ends (0 is no limit)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *maxParallel < 0 {
		fmt.Fprintln(os.Stderr, "--max-parallel must be 0 or more")
		return 2
	}
	token := os.Getenv("SPAN_TOKEN")
	hub := live.New()
	hub.SetLimit(*maxParallel)
	pages, err := web.Pages()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	// Agents run in their own process group (proc.Tame), so Ctrl-C no longer reaches them
	// through the terminal. Cancel every flight first so each one ends its own tree.
	ctx, stop := interruptContext()
	defer stop()
	go func() {
		<-ctx.Done()
		hub.StopAll()
		hub.Drain(proc.Grace + 5*time.Second)
		os.Exit(130)
	}()
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
	return func(ctx context.Context, id string, wakeup serve.Signal) error {
		snap, err := engine.Load(spans, id)
		if err != nil {
			return err
		}
		opt, err := options(snap.Cursor.Workspace, spans, resolvePolicy(spans, snap.Cursor.Workspace, snap.Cursor.Project), snap.Cursor.Project, false)
		if err != nil {
			return err
		}
		opt.Watch = hub
		opt.Issue = snap.Cursor.Issue
		var brief string
		if wakeup.Reload {
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
		if err := hub.Admit(id, snap.Cursor.Stage); err != nil {
			return err
		}
		// The claim only covers this process. Another one (`span resume`) may hold the run;
		// the engine would refuse inside the flight, where the answers are already lost.
		if holder, busy := engine.RunBusy(spans, id); busy {
			hub.EndFlight(id)
			return &runlock.BusyError{Scope: runlock.Run, Holder: holder}
		}
		hub.Fly(id, func(flight context.Context) {
			if wakeup.Resume {
				_, _ = engine.Resume(flight, opt, id)
				return
			}
			if wakeup.Reload {
				_, _ = engine.Reload(flight, opt, id, brief)
				return
			}
			_, _ = engine.Continue(flight, opt, id, wakeup.Answers)
		})
		return nil
	}
}

// interruptContext ends on the first SIGINT, SIGTERM, or SIGHUP (closing the terminal) and then restores the default
// handling, so a second Ctrl-C still kills the process if the graceful stop hangs.
// forceExit is what the second interrupt does: end every command Span started, then go.
// A variable so a test can observe it without ending the test process.
var forceExit = func() {
	proc.KillAll()
	os.Exit(130)
}

// interruptContext cancels on the first SIGINT, SIGTERM or SIGHUP, which is a graceful
// stop: commands get their grace period to end. A second signal does not wait: it kills
// whatever Span started and exits, so an agent cannot outlive a person who pressed Ctrl-C
// twice. Finishing normally cancels the context without arming the exit.
func interruptContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	sigs := make(chan os.Signal, 2)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		defer signal.Stop(sigs)
		select {
		case <-sigs:
			cancel()
		case <-ctx.Done():
			return
		}
		<-sigs
		forceExit()
	}()
	return ctx, cancel
}

func quickstartCmd(args []string) int {
	fs := flag.NewFlagSet("quickstart", flag.ContinueOnError)
	workspace := fs.String("workspace", ".", "project to configure")
	story := fs.String("story", "", "user story")
	storyFile := fs.String("story-file", "", "file that holds the user story")
	kind := fs.String("kind", "", "feat, fix, or hotfix")
	issue := fs.String("issue", "", "linked story key, such as AET-1234")
	base := fs.String("base", "", "branch to start from and open the change into (default: base in span.yaml, else dev for feat/fix, master for hotfix)")
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
		one, err := withBase(one)
		if err != nil {
			fmt.Fprintf(&b, "repo %s\n%s\n", target.Name, err)
			code = 1
			continue
		}
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
	var runs []serve.RunRef
	var failures []error
	for _, target := range targets {
		one := req
		one.Workspace = target.Path
		one.Story = target.Story
		one.Issue = target.Issue
		one, err := withBase(one)
		if err != nil {
			return serve.Outcome{}, fmt.Errorf("repo %s: %w", target.Name, err)
		}
		if strings.TrimSpace(one.Story) == "" {
			res, claim, err := claimAndConfigure(one, "configuring")
			if err != nil {
				return serve.Outcome{}, err
			}
			claim.Release()
			fmt.Fprintf(&text, "repo %s\npolicy %s\nbranch %s\nbase %s\nconfiguration ready\n", target.Name, res.PolicyPath, res.Branch, res.Base)
			continue
		}
		gotID, gotStage, err := beginOne(one, spans, hub)
		if err != nil {
			// Keep going: one busy checkout must not stop the other repos, and the runs that
			// did start must be reported, not dropped.
			failures = append(failures, fmt.Errorf("repo %s: %w", target.Name, err))
			fmt.Fprintf(&text, "repo %s did not start: %v\n", target.Name, err)
			continue
		}
		runs = append(runs, serve.RunRef{ID: gotID, Stage: gotStage, Repo: target.Name})
	}
	if len(runs) == 0 && len(failures) > 0 {
		return serve.Outcome{}, errors.Join(failures...)
	}
	out := serve.Outcome{Text: text.String(), Runs: runs, Partial: len(failures) > 0}
	if len(runs) > 0 {
		out.ID, out.Stage = runs[0].ID, runs[0].Stage
	}
	return out, nil
}

// claimAndConfigure takes the checkout and then writes the project's configuration into
// it. The order matters: no run can be editing the tree while span.yaml is rewritten. The
// caller releases the claim, or hands it to a run in engine.Options.HeldWorkspace.
func claimAndConfigure(req setup.Request, label string) (setup.Result, *runlock.Lock, error) {
	claim, err := engine.ClaimWorkspace(req.Workspace, label)
	if err != nil {
		var busy *runlock.BusyError
		if errors.As(err, &busy) {
			return setup.Result{}, nil, fmt.Errorf("%w; wait for it or stop it first", busy)
		}
		return setup.Result{}, nil, err
	}
	res, err := setup.Configure(req)
	if err != nil {
		claim.Release()
		return setup.Result{}, nil, err
	}
	return res, claim, nil
}

// resumeRun drives a launched run. A variable so a test can launch without an agent CLI.
var resumeRun = engine.Resume

func beginOne(req setup.Request, spans string, hub *live.Hub) (string, string, error) {
	// Take the checkout before anything is written into it. Checking first and writing
	// after leaves a window in which two launches both pass the check and the second
	// rewrites span.yaml and the brief under the run that is starting. A second story in a
	// checkout another run is editing would also swap its branch and commit its files, so
	// that run gets its own worktree instead and the two go side by side.
	// A process at its --max-parallel limit refuses first, before anything is claimed,
	// written or created. Admit below is the atomic check; this one only keeps a refused
	// launch from leaving a run behind.
	if err := hub.Room(); err != nil {
		return "", "", err
	}
	st, err := claimSite(req, spans, "starting a run")
	if err != nil {
		return "", "", err
	}
	res, claim := st.Result, st.Claim
	handedOver, opened := false, false
	defer func() {
		if !handedOver {
			claim.Release()
		}
		if !opened {
			// No run exists, so nobody owns an isolated tree made for this launch.
			st.discard()
		}
	}()
	opt, err := options(st.Code.Workspace, st.Code.Spans, st.Code.Policy, st.Code.Project, st.Code.Explicit)
	if err != nil {
		return "", "", err
	}
	opt.Brief = req.Story
	opt.Issue = req.Issue
	opt.Branch = res.Branch
	opt.Base = res.Base
	opt.Files = req.Files
	opt.Watch = hub
	opt.HeldWorkspace = claim
	id, err := engine.Open(opt)
	if err != nil {
		return "", "", err
	}
	opened = true
	claim.Relabel("run " + id)
	if err := hub.Admit(id, "intake"); err != nil {
		return "", "", err
	}
	handedOver = true
	// Fly wires the stop handle, so Stop works from the first frame and not only after
	// the run has held once and been woken.
	hub.Fly(id, func(flight context.Context) {
		defer claim.Release()
		_, _ = resumeRun(flight, opt, id)
	})
	return id, "intake", nil
}

func runOne(ctx context.Context, req setup.Request) (string, int) {
	res, claim, err := claimAndConfigure(req, "span run")
	if err != nil {
		return err.Error() + "\n", 1
	}
	defer claim.Release()
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
	// The run works on its branch in its own tree; the checkout the person pointed at is
	// never switched, so a second story can run beside this one.
	code, err := prepare(req.Workspace, "", "", res.Branch, res.Base, res.Branch != "")
	if err != nil {
		return b.String() + err.Error() + "\n", 1
	}
	isolated := code.Workspace != code.Project
	if isolated {
		fmt.Fprintf(&b, "worktree %s\n", code.Workspace)
		// The claim only protected the configuration just written. The run edits its own
		// tree, so the checkout is free for the next story at once.
		claim.Release()
	}
	opt, err := options(code.Workspace, code.Spans, code.Policy, code.Project, code.Explicit)
	if err != nil {
		return b.String() + err.Error() + "\n", 1
	}
	opt.Brief = req.Story
	opt.Issue = req.Issue
	opt.Branch = res.Branch
	opt.Base = res.Base
	opt.Files = req.Files
	if !isolated {
		opt.HeldWorkspace = claim
	}
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

// options builds a run's engine options. explicit says the person named policyPath
// (--policy): that file must exist. The implicit span.yaml is optional, and without it the
// run gets the policy quickstart would write.
func options(workspace, spans, policyPath, project string, explicit bool) (engine.Options, error) {
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
	load := setup.LoadOrDefault
	if explicit {
		load = func(path, _ string) (policy.File, error) { return policy.Load(path) }
	}
	file, err := load(policyPath, workspace)
	if err != nil {
		return engine.Options{}, err
	}
	opt := engine.Options{
		Workspace: workspace,
		Root:      spans,
		Policy:    file,
		Agents:    &adapt.CLIAgents{Models: file.Models},
		Check:     adapt.Exec{},
		Source:    adapt.Git{Dir: workspace, Allow: file.Commit.Allow},
		Review:    adapt.Host{Workspace: workspace, State: spans},
		Release:   adapt.CommandRelease{},
		Worktree:  adapt.IsWorktree(workspace),
		Project:   project,
	}
	if project == "" {
		project = workspace
	}
	if account, err := jira.Load(project); err == nil {
		opt.Issues = jira.Client{Account: account}
		if account.Token != "" {
			opt.Secrets = append(opt.Secrets, account.Token)
		}
	}
	if serveToken := os.Getenv("SPAN_TOKEN"); serveToken != "" {
		opt.Secrets = append(opt.Secrets, serveToken)
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
./span quickstart [--workspace .] [--repo name] [--file path] [--story "..."] [--story-file story.md] [--kind feat|fix|hotfix] [--issue AET-1234] [--base branch]
./span run --brief brief.md [--workspace .] [--branch name] [--base branch] [--worktree]
./span resume <id> [--workspace .]
./span serve [--addr 127.0.0.1:8787] [--spans .span/runs]`)
}

// prepared is where a run writes: the checkout the person pointed at, and the code either
// there or in a worktree of it.
type prepared struct {
	Workspace string
	Project   string
	Spans     string
	Policy    string
	// Explicit is true when the person named the policy file (--policy). Only the implicit
	// <project>/span.yaml may be missing.
	Explicit bool
}

// prepare isolates a run in its own git worktree when asked. The state directory and the
// policy stay in the checkout the person pointed at, so the desk keeps one list of runs
// and the policy is not copied into every tree.
func prepare(workspace, spans, policyPath, name, base string, isolate bool) (prepared, error) {
	project, err := filepath.Abs(workspace)
	if err != nil {
		return prepared{}, err
	}
	out := prepared{Workspace: project, Project: project}
	if spans == "" {
		out.Spans = filepath.Join(project, ".span", "runs")
	} else {
		out.Spans = spans
	}
	if policyPath == "" {
		out.Policy = filepath.Join(project, "span.yaml")
	} else {
		out.Policy, out.Explicit = policyPath, true
	}
	if !isolate {
		return out, nil
	}
	if strings.TrimSpace(name) == "" {
		return prepared{}, errors.New("--worktree needs --branch: the tree is the branch")
	}
	dir := worktree.Path(project, name)
	start, err := runBase(base, project, name)
	if err != nil {
		return prepared{}, err
	}
	// Add cuts a new branch from the local base, or from origin/<base> after a fetch, and
	// refuses a base that is nowhere.
	if err := worktree.Add(context.Background(), adapt.Git{Dir: project}, project, dir, name, start); err != nil {
		return prepared{}, err
	}
	out.Workspace = dir
	return out, nil
}

// runBase is the base a run starts from and opens its change into: the one it names, else
// the base: line of span.yaml in project for feat and fix, else the kind's default (master
// for a hotfix/ branch, dev otherwise). The kind is the branch name's prefix.
func runBase(explicit, project, name string) (string, error) {
	kind, _, _ := strings.Cut(name, "/")
	if kind != branch.Hotfix {
		kind = branch.Feat
	}
	chosen, err := projectBase(project)
	if err != nil {
		return "", err
	}
	_, base, err := branch.Choose("", kind, explicit, chosen)
	if err != nil || base != "" {
		return base, err
	}
	return branch.DefaultBase(kind), nil
}

// withBase settles a launch's kind and base before its branch is derived (branch.Choose):
// the base the run names wins; otherwise span.yaml's base applies to feat and fix only,
// and the kind is fixed so that base cannot make the story a hotfix. Still empty, the
// kind's default applies when the branch is derived.
func withBase(req setup.Request) (setup.Request, error) {
	chosen, err := projectBase(req.Workspace)
	if err != nil {
		return req, err
	}
	kind, base, err := branch.Choose(req.Story, req.Kind, req.Base, chosen)
	if err != nil {
		return req, err
	}
	req.Kind, req.Base = kind, base
	return req, nil
}

// projectBase is the base: line of span.yaml in project, or "" when there is none.
func projectBase(project string) (string, error) {
	if project == "" {
		return "", nil
	}
	return policy.ProjectBase(project)
}
