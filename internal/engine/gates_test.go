package engine

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"span/internal/decision"
	"span/internal/overlay"
)

var ac400 = decision.Clash{
	Story: "AC 400: shows the rejection when the button is pressed",
	Rule:  "disable submit while the form is invalid",
	Ref:   "AGENTS.md:12",
}

func writeRules(t *testing.T, opt Options, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(opt.Workspace, "AGENTS.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestFrameReadsTheRepositoryRulesAndOnlyThen(t *testing.T) {
	opt, agents, _, _, _, _ := baseOpt(t)
	if _, err := Start(context.Background(), opt); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(agents.prompts[0], "Repository rules") {
		t.Fatalf("a repository without AGENTS.md got a rules block:\n%s", agents.prompts[0])
	}

	opt, agents, _, _, _, _ = baseOpt(t)
	writeRules(t, opt, "# Rules\nDisable submit while the form is invalid.\n")
	if _, err := Start(context.Background(), opt); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(agents.prompts[0], "Repository rules (AGENTS.md)") || !strings.Contains(agents.prompts[0], "Disable submit while the form is invalid.") {
		t.Fatalf("frame did not see the rules:\n%s", agents.prompts[0])
	}
}

func TestAStoryThatContradictsTheRulesAsksBeforeAnyBuild(t *testing.T) {
	opt, agents, _, _, _, _ := baseOpt(t)
	writeRules(t, opt, "Disable submit while the form is invalid.\n")
	agents.conflicts = []decision.Clash{ac400}

	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != "hold" || agents.builds != 0 {
		t.Fatalf("verdict %s, builds %d", got.Verdict, agents.builds)
	}
	d := readCursor(t, opt, got.ID).Decision
	if d == nil || d.Class != "spec-conflict" || d.Stage != "frame" || d.Validate() != nil || d.Recommendation != "follow-rule" {
		t.Fatalf("decision = %+v", d)
	}
	var quoted []string
	for _, e := range d.Evidence {
		quoted = append(quoted, e.Excerpt)
	}
	if joined := strings.Join(quoted, "|"); !strings.Contains(joined, ac400.Story) || !strings.Contains(joined, ac400.Rule) {
		t.Fatalf("the two sides are not both quoted: %v", quoted)
	}

	// The answer is a decision every later prompt carries. Frame still reports the same
	// conflict, and it is not asked a second time.
	if _, err := Decide(context.Background(), opt, got.ID, decisionAnswer("follow-rule", "")); err != nil {
		t.Fatal(err)
	}
	if agents.builds != 1 {
		t.Fatalf("builds = %d", agents.builds)
	}
	var build string
	for _, p := range agents.prompts {
		if strings.Contains(p, "ODD is mandatory") {
			build = p
		}
	}
	if !strings.Contains(build, "Follow the repository rule") {
		t.Fatalf("the build did not get the decision:\n%s", build)
	}
	entries, err := overlay.Read(filepath.Join(opt.Root, got.ID))
	if err != nil || len(overlay.Active(entries)) != 1 || overlay.Active(entries)[0].Class != "spec-conflict" {
		t.Fatalf("log = %+v (%v)", entries, err)
	}
}

func TestAnAnsweredConflictIsNotAskedAgainEvenIfFrameWordsItDifferently(t *testing.T) {
	opt, agents, _, _, _, _ := baseOpt(t)
	agents.conflicts = []decision.Clash{ac400}
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	agents.conflicts = []decision.Clash{{Story: "pressing submit explains why it failed", Rule: "the button stays disabled when invalid", Ref: "AGENTS.md:12"}}
	if _, err := Decide(context.Background(), opt, got.ID, decisionAnswer("follow-story", "")); err != nil {
		t.Fatal(err)
	}
	if d := readCursor(t, opt, got.ID).Decision; d != nil {
		t.Fatalf("asked again: %+v", d)
	}
}

func TestStoppingAConflictEndsTheRun(t *testing.T) {
	opt, agents, _, _, _, _ := baseOpt(t)
	agents.conflicts = []decision.Clash{ac400}
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	res, err := Decide(context.Background(), opt, got.ID, decisionAnswer("stop", ""))
	if err != nil || res.Verdict != "stop" {
		t.Fatalf("%+v %v", res, err)
	}
}

func TestAHeldPlanIsReviewedBeforeBuildAndApprovingDoesNotFrameAgain(t *testing.T) {
	opt, agents, _, _, _, _ := baseOpt(t)
	opt.Policy.Stages.Plan = "hold"
	agents.assumptions = []string{"amounts are in cents"}

	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	d := readCursor(t, opt, got.ID).Decision
	if got.Verdict != "hold" || agents.builds != 0 || d == nil || d.Class != "plan-review" || d.Validate() != nil {
		t.Fatalf("verdict %s builds %d decision %+v", got.Verdict, agents.builds, d)
	}
	var shown []string
	for _, e := range d.Evidence {
		shown = append(shown, e.Excerpt)
	}
	if joined := strings.Join(shown, "|"); !strings.Contains(joined, "charge the card") || !strings.Contains(joined, "amounts are in cents") {
		t.Fatalf("the plan is not on the card: %v", shown)
	}

	frames := agents.frames
	if _, err := Decide(context.Background(), opt, got.ID, decisionAnswer("approve", "")); err != nil {
		t.Fatal(err)
	}
	if agents.frames != frames {
		t.Fatalf("approving ran frame again: %d -> %d", frames, agents.frames)
	}
	if agents.builds != 1 || readCursor(t, opt, got.ID).Decision != nil {
		t.Fatalf("builds %d decision %+v", agents.builds, readCursor(t, opt, got.ID).Decision)
	}
}

func TestRevisingAPlanFramesAgainWithTheWordsAndAsksAgain(t *testing.T) {
	opt, agents, _, _, _, _ := baseOpt(t)
	opt.Policy.Stages.Plan = "hold"
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Decide(context.Background(), opt, got.ID, decisionAnswer("revise", "split it in two stories")); err != nil {
		t.Fatal(err)
	}
	if agents.frames != 2 || agents.builds != 0 {
		t.Fatalf("frames %d builds %d", agents.frames, agents.builds)
	}
	if !strings.Contains(agents.prompts[len(agents.prompts)-1], "split it in two stories") {
		t.Fatal("the second frame did not get the person's words")
	}
	if d := readCursor(t, opt, got.ID).Decision; d == nil || d.Class != "plan-review" {
		t.Fatalf("the revised plan was not put back for review: %+v", d)
	}
}

func TestAConditionalPlanGateOnlyHoldsAHeavyStory(t *testing.T) {
	opt, agents, _, _, _, _ := baseOpt(t)
	opt.Policy.Stages.Plan = "conditional"
	if _, err := Start(context.Background(), opt); err != nil {
		t.Fatal(err)
	}
	if agents.builds != 1 {
		t.Fatalf("a light story was held: builds %d", agents.builds)
	}

	opt, agents, _, _, _, _ = baseOpt(t)
	opt.Policy.Stages.Plan = "conditional"
	opt.Policy.Run.Weight = "deep"
	got, err := Start(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	if agents.builds != 0 || readCursor(t, opt, got.ID).Decision == nil {
		t.Fatalf("a deep story was not held: builds %d", agents.builds)
	}
}

func TestWithoutAPlanGateFrameAdvancesAsBefore(t *testing.T) {
	opt, agents, _, _, _, _ := baseOpt(t)
	if _, err := Start(context.Background(), opt); err != nil {
		t.Fatal(err)
	}
	if agents.builds != 1 || agents.frames != 1 {
		t.Fatalf("frames %d builds %d", agents.frames, agents.builds)
	}
}
