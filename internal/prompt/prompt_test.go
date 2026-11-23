package prompt

import (
	"strings"
	"testing"
)

func TestFrameReadsTheStoryInItsOwnLanguage(t *testing.T) {
	got := Frame(Input{Stage: "frame", FeaturePath: "odd/tasks/payments.md", Brief: "Quien tiene el permiso ve la entrada."})
	for _, want := range []string{"language it is written", `"acceptance":[]`, "story's words", "Quien tiene el permiso ve la entrada."} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q\n%s", want, got)
		}
	}
}

func TestBuildRequiresODDAndRefusesDelivery(t *testing.T) {
	got := Build(Input{
		Stage: "build", FeaturePath: "odd/tasks/payments.md",
		Brief:        "Charge the card.",
		Instructions: "Use the existing ledger.",
		Manifest:     "skills: review\n",
	})
	for _, want := range []string{
		"ODD is mandatory",
		"odd/tasks/payments.md",
		"Use the existing ledger.",
		"Do not push, merge, or deploy.",
		"skills: review",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q\n%s", want, got)
		}
	}
}

func TestBuildNamesAcceptanceIDsAndThePlaywrightAid(t *testing.T) {
	got := Build(Input{
		Stage: "build", FeaturePath: "odd/tasks/payments.md",
		Acceptance: []string{"AC-1"}, Browser: true,
		Manifest: "mcp: @playwright/mcp --caps=testing",
	})
	for _, want := range []string{"AC-1", "getByRole", "@playwright/mcp --caps=testing", "not the gate"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q\n%s", want, got)
		}
	}
}

func TestLessonsSitAfterTheBrief(t *testing.T) {
	got := Frame(Input{
		Stage: "frame", FeaturePath: "odd/tasks/payments.md", Brief: "Charge the card.",
		Lessons: "- agent reply has no JSON object",
	})
	brief := strings.Index(got, "Brief:\nCharge the card.")
	lesson := strings.Index(got, "Lessons from earlier runs of this stage. Do not repeat them.\n- agent reply has no JSON object")
	if brief < 0 || lesson < brief {
		t.Fatalf("lesson %d brief %d\n%s", lesson, brief, got)
	}
}

func TestBuildDoesNotInlineASkillBody(t *testing.T) {
	got := Build(Input{Stage: "build", Manifest: "skills: review"})
	if strings.Contains(got, "function review") {
		t.Fatal("skill body leaked into the prompt")
	}
}

func TestAWorktreePromptForbidsSwitchingBranches(t *testing.T) {
	text := Build(Input{Stage: "build", FeaturePath: "odd/tasks/x.md", Worktree: true})
	for _, want := range []string{"git worktree on its own branch", "git checkout", "git worktree add", "Span commits"} {
		if !strings.Contains(text, want) {
			t.Fatalf("the worktree rules are missing %q", want)
		}
	}
	if bare := Build(Input{Stage: "build", FeaturePath: "odd/tasks/x.md"}); strings.Contains(bare, "git worktree on its own branch") {
		t.Fatal("a shared checkout was told it is a worktree")
	}
}

func TestFrameForbidsEditingFiles(t *testing.T) {
	// The baseline is recorded after frame, on the tree frame left; frame must not change it.
	if got := Frame(Input{Stage: "frame", Brief: "Charge the card."}); !strings.Contains(got, "Do not edit files.") {
		t.Fatalf("frame prompt:\n%s", got)
	}
}

func TestBuildAsksForDecisionAndRiskLinesInTheLog(t *testing.T) {
	got := Build(Input{Stage: "build", FeaturePath: "odd/tasks/payments.md"})
	for _, want := range []string{`"Decision: <what> because <why>"`, `"Risk: <what could go wrong>`, `"Risk: none found" only when you looked`} {
		if !strings.Contains(got, want) {
			t.Fatalf("the build prompt does not ask for %s:\n%s", want, got)
		}
	}
}
