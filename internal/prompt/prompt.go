// Package prompt builds the text handed to whichever agent binary is selected.
// Skill bodies are not copied in. The harness loads those by name.
package prompt

import (
	"fmt"
	"strings"
)

// Input is the context one stage is allowed to see.
type Input struct {
	Stage        string
	FeaturePath  string
	Brief        string
	Instructions string
	Answers      string
	Handoffs     []string
	Failure      string
	Worktree     bool
	Manifest     string
	Standards    string
	Acceptance   []string
	Browser      bool
	Lessons      string
}

// Frame asks for verifiable claims and forbids invented requirements.
func Frame(in Input) string {
	return preamble(in) + `
Read the story in the language it is written. Carry what it already decides into this JSON.
{"claims":[],"assumptions":[],"questions":[],"acceptance":[],"outOfScope":false}
acceptance lists each criterion the story already states, one entry, in the story's words.
A question is allowed only when the story cannot be implemented without inventing a requirement.
Do not invent a criterion the story does not state.
Do not edit files. Do not push, merge, or deploy.
`
}

// Build requires Gentle AI ODD before any product code.
func Build(in Input) string {
	var extra strings.Builder
	if len(in.Acceptance) > 0 {
		fmt.Fprintf(&extra, "Add one test per acceptance id in the project's test runner: %s.\n", strings.Join(in.Acceptance, ", "))
		extra.WriteString("Each id must appear on a test declaration line, as AC-1 or AC1. A comment is not enough. Use func Test, test(, it(, or def test_.\n")
		extra.WriteString("A criterion the gate reports as unverified has no declaration yet. Add the test, do not change the brief.\n")
	}
	if in.Browser {
		extra.WriteString("This repo is checked in a browser. Test user-visible behavior. Prefer getByRole. Use web-first expect assertions. Keep each test isolated. Record a trace on the first retry, not on every test.\n")
		extra.WriteString("Authoring aid, not the gate: @playwright/mcp --caps=testing. The gate is the qa command.\n")
	}
	return preamble(in) + fmt.Sprintf(`
ODD is mandatory. You are running Gentle AI Organic Driven Development.
Before writing product code, create or update %s with a heading, "## Specs", "## Tasks", and "## Log".
Quote the brief in the log. Implement only after that document exists.
Write so two people produce one author's code. Read a harness skill only when its trigger matches this change. Do not paste unused skills into the feature.
%sDo not push, merge, or deploy.
`, in.FeaturePath, extra.String())
}

// Inspect reads the diff against the feature document. It does not edit.
func Inspect(in Input) string {
	return preamble(in) + `
You did not write this change. Read the diff against the feature document and the project rules.
Reply with JSON only: {"blocking":false,"reason":""}
blocking is true when the change misses a spec or breaks a project rule.
Do not edit files. Do not push, merge, or deploy.
`
}

func preamble(in Input) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Stage: %s\nFeature document: %s\n", in.Stage, in.FeaturePath)
	if in.Manifest != "" {
		fmt.Fprintf(&b, "Manifest names (load these through your harness, bodies are not included):\n%s\n", in.Manifest)
	}
	if in.Standards != "" {
		fmt.Fprintf(&b, "Standards:\n%s\n", in.Standards)
	}
	fmt.Fprintf(&b, "Brief:\n%s\n", in.Brief)
	if in.Instructions != "" {
		fmt.Fprintf(&b, "Stage instructions:\n%s\n", in.Instructions)
	}
	if in.Answers != "" {
		fmt.Fprintf(&b, "Answers:\n%s\n", in.Answers)
	}
	if len(in.Handoffs) > 0 {
		fmt.Fprintf(&b, "Handoffs (open these workspace files; they are part of the story, including images):\n%s\n", strings.Join(in.Handoffs, "\n"))
	}
	if in.Worktree {
		b.WriteString("This workspace is a git worktree on its own branch. The branch is already checked out.\n")
		b.WriteString("Do not run git checkout, git switch, git worktree add, or git stash: they move or hide a tree another run may be using.\n")
		b.WriteString("Read `git worktree list` if you need to see the other trees. Do not commit; Span commits.\n")
	}
	if in.Failure != "" {
		fmt.Fprintf(&b, "Previous check failed:\n%s\n", in.Failure)
	}
	if in.Lessons != "" {
		fmt.Fprintf(&b, "Lessons from earlier runs of this stage. Do not repeat them.\n%s\n", in.Lessons)
	}
	return b.String()
}
