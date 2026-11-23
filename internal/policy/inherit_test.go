package policy

import "testing"

func TestInheritFillsAnUnsetAgentAndMissingModelsFromTheFolder(t *testing.T) {
	repo := File{Agent: "any", Models: map[string]string{"pi": "own-model"}}
	folder := File{Agent: "pi", Models: map[string]string{"pi": "folder-model", "claude": "opus"}}
	got := repo.Inherit(folder)
	if got.Agent != "pi" {
		t.Fatalf("agent %q, want the folder's pi", got.Agent)
	}
	if got.Models["pi"] != "own-model" || got.Models["claude"] != "opus" {
		t.Fatalf("models %v: the repo's own model wins and the folder fills the rest", got.Models)
	}
}

func TestInheritKeepsAnAgentTheRepoChose(t *testing.T) {
	got := File{Agent: "claude"}.Inherit(File{Agent: "pi", Models: map[string]string{"pi": "m"}})
	if got.Agent != "claude" {
		t.Fatalf("agent %q, the repo's explicit choice must stand", got.Agent)
	}
}

func TestInheritDoesNotChangeEitherInput(t *testing.T) {
	repo := File{Agent: "any"}
	folder := File{Agent: "pi", Models: map[string]string{"pi": "m"}}
	_ = repo.Inherit(folder)
	if repo.Models != nil || folder.Models["pi"] != "m" || len(folder.Models) != 1 {
		t.Fatalf("inputs were changed: %v %v", repo, folder)
	}
}

func TestInheritFromAnEmptyFolderIsTheRepoPolicy(t *testing.T) {
	repo := File{Agent: "any"}
	if got := repo.Inherit(File{}); got.Agent != "any" {
		t.Fatalf("agent %q", got.Agent)
	}
}
