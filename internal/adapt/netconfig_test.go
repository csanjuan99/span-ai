package adapt

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// The conflict phase turns off the system git config so nothing there can run code. A
// fetch from a private origin needs that very config on macOS, where the keychain credential
// helper is set only at system level. The fetch must therefore keep it, while the rest of the
// phase stays closed.
func TestTheBaseFetchStillSeesTheSystemGitConfig(t *testing.T) {
	bare := filepath.Join(t.TempDir(), "origin.git")
	for _, args := range [][]string{{"init", "-q", "--bare", "-b", "dev", bare}} {
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	seed := repo(t)
	commitAll(t, seed, "a.txt", "x\n")
	for _, args := range [][]string{
		{"-C", seed, "branch", "-M", "dev"},
		{"-C", seed, "push", "-q", bare, "dev"},
	} {
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	work := repo(t)
	commitAll(t, work, "b.txt", "y\n")
	// origin is an address only the system config knows how to reach, the way a credential
	// helper is only known there.
	const remote = "https://git.invalid/o/r.git"
	if out, err := exec.Command("git", "-C", work, "remote", "add", "origin", remote).CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	system := filepath.Join(t.TempDir(), "gitconfig")
	rule := "[url \"" + bare + "\"]\n\tinsteadOf = " + remote + "\n"
	if err := os.WriteFile(system, []byte(rule), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_SYSTEM", system)

	ref, _, note := (Git{Dir: work}).baseRef(context.Background(), "dev")
	if ref != "origin/dev" || note != "" {
		t.Fatalf("the base was not fetched: ref=%q note=%q", ref, note)
	}
}
