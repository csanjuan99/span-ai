package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"span/internal/shape"
)

// shapeTransferUsage lists the verbs that bring stories in and out. They call the same shape
// functions as POST /shapes/import, GET /shapes/{id}/export and the share routes.
const shapeTransferUsage = `  import   (--jira KEY | --file story.md) [--repo r] [--agent a] [--model m] [--dry-run] [--no-run]
  export   <id|SES-ref> [--format md|json] [--sources] [--assumptions] [--digest] [--out file]
  share    <id|SES-ref> [--days n] | --list | --revoke TOKEN | --open TOKEN`

// transferVerbs are the verbs of shapeTransferUsage.
var transferVerbs = map[string]bool{"import": true, "export": true, "share": true}

// transferOpts are the flags of the transfer verbs.
type transferOpts struct {
	jira, file, format, out string
	sources, assumptions    bool
	digest, dry, noRun      bool
	days                    int
	list                    bool
	revoke, open            string
}

func (o *shapeOpts) transferFlags(fs *flag.FlagSet, verb string) {
	switch verb {
	case "import":
		fs.StringVar(&o.xfer.jira, "jira", "", "Jira story key to start from; Span only reads it")
		fs.StringVar(&o.xfer.file, "file", "", "markdown file to start from")
		fs.StringVar(&o.repo, "repo", "", "repo inside the workspace")
		fs.StringVar(&o.agent, "agent", "", "interview agent; empty uses the project default")
		fs.StringVar(&o.model, "model", "", "model of the agent")
		fs.BoolVar(&o.xfer.dry, "dry-run", false, "say what was read and start nothing")
		fs.BoolVar(&o.xfer.noRun, "no-run", false, "create the session without running the first interview turn")
	case "export":
		fs.StringVar(&o.xfer.format, "format", "md", "md or json")
		fs.BoolVar(&o.xfer.sources, "sources", false, "include where each rule and example comes from")
		fs.BoolVar(&o.xfer.assumptions, "assumptions", false, "include the assumptions the story was sealed with")
		fs.BoolVar(&o.xfer.digest, "digest", false, "include the story's digest")
		fs.StringVar(&o.xfer.out, "out", "", "write the file here instead of the terminal")
	case "share":
		fs.IntVar(&o.xfer.days, "days", 0, "how long the link lasts; 7 when not given")
		fs.BoolVar(&o.xfer.list, "list", false, "list the links of the session")
		fs.StringVar(&o.xfer.revoke, "revoke", "", "revoke the link with this token")
		fs.StringVar(&o.xfer.open, "open", "", "show what the link with this token shows")
	}
}

func (o *shapeOpts) transferVerb(ctx context.Context, verb string) int {
	if verb == "import" {
		return o.importStory(ctx)
	}
	if len(o.args) != 1 {
		fmt.Fprintln(o.err, shapeUsage)
		return shapeExitUsage
	}
	ids, err := o.resolveSessions(o.args)
	if err != nil {
		return o.fail(err, false)
	}
	if verb == "export" {
		return o.export(ids[0])
	}
	return o.share(ids[0])
}

func (o *shapeOpts) importStory(ctx context.Context) int {
	x := o.xfer
	if (x.jira == "") == (x.file == "") {
		fmt.Fprintln(o.err, "import needs exactly one of --jira or --file")
		return shapeExitUsage
	}
	if x.file != "" {
		return o.importFile()
	}
	return o.importJira(ctx)
}

func (o *shapeOpts) importFile() int {
	raw, err := os.ReadFile(o.xfer.file)
	if err != nil {
		fmt.Fprintln(o.err, err)
		return shapeExitFailed
	}
	if o.xfer.dry {
		preview, err := shape.ParseMarkdown(filepath.Base(o.xfer.file), string(raw))
		if err != nil {
			return o.fail(err, false)
		}
		if o.json {
			return o.encode(preview)
		}
		fmt.Fprintf(o.out, "%s, %d bytes\n", preview.Name, preview.Bytes)
		for _, line := range preview.Reads {
			fmt.Fprintf(o.out, "  %s\n", line)
		}
		return 0
	}
	s, preview, err := shape.ImportMarkdown(o.ws, shape.ImportInput{
		Repo: o.repo, Agent: o.agent, Model: o.model, Name: filepath.Base(o.xfer.file), Markdown: string(raw),
	})
	if err != nil {
		return o.fail(err, false)
	}
	if !o.xfer.noRun {
		eng := &shape.Engine{Run: shapeAgents(o.ws, s.Agent, s.Model)}
		next, resumeErr := eng.Resume(context.Background(), o.ws, s.ID, "", "")
		if resumeErr != nil {
			return o.turn(next, resumeErr)
		}
		s = next
	}
	// POST /shapes/import answers the session and what the file was read as.
	if o.json {
		return o.encode(map[string]any{"session": s.View(), "read": preview})
	}
	return o.show(s, nil)
}

func (o *shapeOpts) importJira(ctx context.Context) int {
	reader, err := shapeIssues(o.ws)
	if err != nil {
		return o.fail(err, false)
	}
	if o.xfer.dry {
		read, err := shape.ReadStory(ctx, reader, o.xfer.jira)
		if err != nil {
			return o.fail(err, false)
		}
		if o.json {
			return o.encode(read)
		}
		fmt.Fprintf(o.out, "summary:     %s\nstatus:      %s\ndescription: %d paragraphs\ncriteria:    %d bullets\ncomments:    %s\n",
			read.Summary, read.Status, read.Paragraphs, read.Criteria, read.CommentRead)
		return 0
	}
	idea, issue, err := shape.IdeaFromIssue(ctx, reader, o.xfer.jira, "")
	if err != nil {
		return o.fail(err, false)
	}
	eng := &shape.Engine{Run: shapeAgents(o.ws, o.agent, o.model)}
	s, err := eng.Start(ctx, shape.StartInput{Workspace: o.ws, Repo: o.repo, Idea: idea, Agent: o.agent, Model: o.model, SourceIssue: issue})
	return o.turn(s, err)
}

func (o *shapeOpts) export(id string) int {
	file, err := shape.Export(o.ws, id, shape.ExportOptions{
		Format: o.xfer.format, Sources: o.xfer.sources, Assumptions: o.xfer.assumptions, Digest: o.xfer.digest,
	})
	if err != nil {
		return o.fail(err, false)
	}
	if o.xfer.out == "" {
		fmt.Fprint(o.out, string(file.Body))
		return 0
	}
	if err := os.WriteFile(o.xfer.out, file.Body, 0o644); err != nil {
		fmt.Fprintln(o.err, err)
		return shapeExitFailed
	}
	if o.json {
		return o.encode(map[string]any{"name": file.Name, "out": o.xfer.out, "bytes": len(file.Body)})
	}
	fmt.Fprintf(o.out, "wrote %s (%d bytes)\n", o.xfer.out, len(file.Body))
	return 0
}

func (o *shapeOpts) share(id string) int {
	x := o.xfer
	switch {
	case x.list:
		list, err := shape.Shares(o.ws, id)
		if err != nil {
			return o.fail(err, false)
		}
		if o.json {
			return o.encode(list)
		}
		for _, l := range list {
			state := "live until " + l.ExpiresAt.Local().Format("2 Jan 15:04")
			if l.Revoked {
				state = "revoked"
			}
			fmt.Fprintf(o.out, "%s  %s  opened %d times\n", l.Token, state, l.Opens)
		}
		return 0
	case x.revoke != "":
		link, err := shape.RevokeShare(o.ws, id, x.revoke)
		if err != nil {
			return o.fail(err, false)
		}
		if o.json {
			return o.encode(link)
		}
		fmt.Fprintf(o.out, "revoked %s\n", link.Token)
		return 0
	case x.open != "":
		story, err := shape.OpenShare(o.ws, id, x.open, time.Now())
		if err != nil {
			return o.fail(err, false)
		}
		if o.json {
			return o.encode(story)
		}
		fmt.Fprint(o.out, story.Story)
		return 0
	}
	link, err := shape.CreateShare(o.ws, id, x.days, time.Now())
	if err != nil {
		return o.fail(err, false)
	}
	path := "/shaping/" + id + "/shared/" + link.Token
	if o.json {
		return o.encode(map[string]any{"link": link, "path": path})
	}
	fmt.Fprintf(o.out, "%s\nexpires %s · read-only · revoke with: span shape share %s --revoke %s\n",
		path, link.ExpiresAt.Local().Format("2 Jan 15:04"), id, link.Token)
	return 0
}
