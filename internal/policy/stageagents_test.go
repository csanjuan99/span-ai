package policy

import (
	"strings"
	"testing"
)

func TestAPolicyWithoutAgentsKeepsEveryStageOnTheProjectAgent(t *testing.T) {
	f, err := Parse([]byte("agent: codex\n"))
	if err != nil {
		t.Fatal(err)
	}
	for _, stage := range AgentStageNames {
		if got := f.StageAgent(stage); got != "" {
			t.Fatalf("%s = %q, want none", stage, got)
		}
	}
	if f.Agent != "codex" || f.DoneStatus() != "" {
		t.Fatalf("%+v", f)
	}
}

func TestAStageAgentIsReadAndAutoMeansNone(t *testing.T) {
	f, err := Parse([]byte("agents:\n  build: codex\n  inspect: any\n"))
	if err != nil {
		t.Fatal(err)
	}
	if f.StageAgent("build") != "codex" || f.StageAgent("inspect") != "" || f.StageAgent("frame") != "" {
		t.Fatalf("%+v", f.Agents)
	}
}

func TestAnUnknownStageOrAgentIsRefused(t *testing.T) {
	for _, body := range []string{"agents:\n  prove: codex\n", "agents:\n  build: nobody\n"} {
		if _, err := Parse([]byte(body)); err == nil {
			t.Fatalf("%q loaded", body)
		}
	}
}

func TestAFolderFillsTheStagesARepoLeavesUnset(t *testing.T) {
	repo := File{Agents: map[string]string{"build": "codex"}}
	folder := File{Agents: map[string]string{"build": "claude", "inspect": "cursor"}}
	got := repo.Inherit(folder)
	if got.StageAgent("build") != "codex" || got.StageAgent("inspect") != "cursor" {
		t.Fatalf("%+v", got.Agents)
	}
	if repo.Agents["inspect"] != "" {
		t.Fatal("the input changed")
	}
}

func TestEditWritesAStageAgentAndClearsIt(t *testing.T) {
	path := write(t, sample)
	preview, err := Edit(path, Changes{Agents: map[string]string{"build": "codex"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Lines) != 1 || preview.Lines[0].Key != "agents.build" || preview.Lines[0].After != "codex" {
		t.Fatalf("%+v", preview)
	}
	if got := read(t, path); !strings.Contains(got, "agents:\n  build: codex\n") || !strings.Contains(got, "# Hand-written policy.") {
		t.Fatalf("%q", got)
	}
	if _, err := Edit(path, Changes{Agents: map[string]string{"build": ""}}); err != nil {
		t.Fatal(err)
	}
	if got := read(t, path); got != sample {
		t.Fatalf("clearing did not restore the file:\n%s", got)
	}
}

func TestEditRefusesAnAgentThatDoesNotExistAndWritesNothing(t *testing.T) {
	path := write(t, sample)
	if _, err := Edit(path, Changes{Agents: map[string]string{"build": "nobody"}}); err == nil {
		t.Fatal("accepted")
	}
	if _, err := Edit(path, Changes{Agents: map[string]string{"prove": "codex"}}); err == nil {
		t.Fatal("accepted a stage with no agent")
	}
	if read(t, path) != sample {
		t.Fatal("the file changed")
	}
}

func TestEditWritesTheJiraDoneStatus(t *testing.T) {
	path := write(t, sample)
	if _, err := Edit(path, Changes{DoneStatus: str("Done")}); err != nil {
		t.Fatal(err)
	}
	f, err := Load(path)
	if err != nil || f.DoneStatus() != "Done" {
		t.Fatalf("%+v %v", f.Jira, err)
	}
	if _, err := Edit(path, Changes{DoneStatus: str("")}); err != nil {
		t.Fatal(err)
	}
	if read(t, path) != sample {
		t.Fatal("clearing did not restore the file")
	}
}
