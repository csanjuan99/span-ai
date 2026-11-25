package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"span/internal/agent"
	"span/internal/fsutil"
	"span/internal/policy"
)

// The page is a different process from the runner, and the cursor belongs to the
// runner: a cursor written by the page is erased by the runner's next save. So both
// requests the page makes live in their own files beside the cursor, and the runner
// reads them where it needs them. Instructions and answers already work this way.
const (
	agentsFile = "agents.json"
	pauseFile  = "pause"
)

// AgentStages are the stages that call a model. The rest run commands.
var AgentStages = []string{frame, build, inspect}

// UsesAgent reports whether a stage can carry an agent choice.
func UsesAgent(stage string) bool {
	for _, name := range AgentStages {
		if name == stage {
			return true
		}
	}
	return false
}

// ReadStageAgents returns the run's per-stage choices. A file that does not parse is
// ignored: a policy fallback is better than refusing to run.
func ReadStageAgents(dir string) map[string]string {
	out := map[string]string{}
	raw, err := os.ReadFile(filepath.Join(dir, agentsFile))
	if err != nil {
		return out
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]string{}
	}
	clean := make(map[string]string, len(out))
	for stage, name := range out {
		if UsesAgent(stage) && name != "" && !agent.Auto(name) {
			clean[stage] = name
		}
	}
	return clean
}

// SetStageAgent records which CLI one stage of one run uses. An empty name or "any"
// clears the exception and falls back to the policy. The file is written atomically,
// so a reader never sees half of it.
func SetStageAgent(dir, stage, name string, home string) error {
	if !UsesAgent(stage) {
		return fmt.Errorf("stage %q does not call an agent; one of %v does", stage, AgentStages)
	}
	clean := ""
	if !agent.Auto(name) {
		if !agent.Known(name) {
			return fmt.Errorf("agent %q is not one of %s", name, agent.IDs())
		}
		if reason := agent.Explain(name, home); reason != "" {
			return errors.New(reason)
		}
		if spec, ok := agent.Find(name); ok {
			clean = spec.ID
		}
	}
	choices := ReadStageAgents(dir)
	if clean == "" {
		delete(choices, stage)
	} else {
		choices[stage] = clean
	}
	raw, err := json.MarshalIndent(choices, "", "  ")
	if err != nil {
		return err
	}
	return fsutil.WriteAtomic(filepath.Join(dir, agentsFile), append(raw, '\n'))
}

// ignoreStateRoot makes the state directory ignore itself, the way .git does. Span
// never commits it, and a workspace whose .gitignore does not name it should not show
// it as a change either.
func ignoreStateRoot(root string) error {
	if filepath.Base(root) != "runs" {
		return nil
	}
	parent := filepath.Dir(root)
	if filepath.Base(parent) != ".span" {
		return nil
	}
	writeIgnoreOnce(parent)
	return nil
}

// RequestPause asks the run to hold before the next stage starts.
func RequestPause(root, id string) error {
	if !validID(id) {
		return os.ErrNotExist
	}
	dir := filepath.Join(root, id)
	if _, err := os.Stat(filepath.Join(dir, "cursor.json")); err != nil {
		return err
	}
	return fsutil.WriteAtomic(filepath.Join(dir, pauseFile), []byte("paused by the page\n"))
}

// takePause reports whether a pause was requested and consumes it, so the next resume
// runs the stage instead of pausing again.
func (r *runner) takePause() bool {
	path := filepath.Join(r.spanDir(), pauseFile)
	if _, err := os.Stat(path); err != nil {
		return false
	}
	r.drop(path)
	return true
}

// drop removes a marker file the run has consumed. A file that is already gone is fine; any
// other failure is logged, because a marker that stays would act twice.
func (r *runner) drop(path string) {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		log.Printf("remove %s %s: %v", r.cur.ID, filepath.Base(path), err)
	}
}

// agentWanted is the CLI a stage asks for: the run's choice for it, else the one span.yaml
// names. Empty and "any" both leave the pick to the first ready CLI.
func (r *runner) agentWanted(stageChoice string) string {
	if stageChoice != "" {
		return stageChoice
	}
	return r.policy().Agent
}

// policy is span.yaml as it reads now when the run can reread it, else as it was at launch.
func (r *runner) policy() policy.File {
	if r.opt.Fresh != nil {
		return r.opt.Fresh()
	}
	return r.opt.Policy
}

// modeled is an Agents that can say which model a CLI would run.
type modeled interface{ Model(binary string) string }

// noteAgent records the CLI and model a stage is about to run. The page shows this, not
// the configuration, so a default nobody chose cannot pass unnoticed. A CLI that does not
// say its model leaves the field empty.
func (r *runner) noteAgent(stage, binary string) {
	detail := map[string]string{"stage": stage, "agent": binary, "model": ""}
	if m, ok := r.opt.Agents.(modeled); ok {
		detail["model"] = m.Model(binary)
	}
	r.journal("agent.run", detail)
}

// pick decides which binary runs a stage. The run's choice for that stage wins, then the
// stage's entry in the policy, then the policy's agent, then the first ready CLI. Inspect keeps using a different CLI from build
// when nothing was chosen, so a review is not the model that wrote the change.
func (r *runner) pick(stage string) (string, error) {
	want := ""
	if UsesAgent(stage) {
		want = r.wantedFor(stage, ReadStageAgents(r.spanDir())[stage])
	}
	sel, err := r.opt.Agents.Select(want)
	if err != nil {
		return "", err
	}
	binary := sel.Build
	if stage == inspect && want == "" {
		binary = sel.Inspect
	}
	r.noteAgent(stage, binary)
	return binary, nil
}
