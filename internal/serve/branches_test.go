package serve

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func gitIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v %s", args, err, out)
	}
}

// branchRepo is a clone with dev and feat/SAL-3 locally and release/2.1 only on origin.
func branchRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	seed := t.TempDir()
	gitIn(t, seed, "init", "-q", "-b", "dev")
	gitIn(t, seed, "config", "user.name", "Span Tests")
	gitIn(t, seed, "config", "user.email", "tests@example.test")
	if err := os.WriteFile(filepath.Join(seed, "README.md"), []byte("span\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, seed, "add", "README.md")
	gitIn(t, seed, "commit", "-q", "-m", "init")
	gitIn(t, seed, "branch", "release/2.1")
	origin := filepath.Join(t.TempDir(), "origin.git")
	gitIn(t, seed, "clone", "-q", "--bare", seed, origin)
	clone := filepath.Join(t.TempDir(), "clone")
	gitIn(t, t.TempDir(), "clone", "-q", "-b", "dev", origin, clone)
	gitIn(t, clone, "branch", "feat/SAL-3")
	return clone
}

type branchesBody struct {
	Branches []string `json:"branches"`
	Default  string   `json:"default"`
}

func getBranches(t *testing.T, path string) (int, branchesBody) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/branches?path="+url.QueryEscape(path), nil)
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()
	Handler(t.TempDir(), "").ServeHTTP(rec, req)
	var body branchesBody
	if rec.Code == http.StatusOK {
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("%v: %s", err, rec.Body.String())
		}
	}
	return rec.Code, body
}

func TestBranchesListsLocalAndOriginBranchesOnce(t *testing.T) {
	repo := branchRepo(t)
	code, body := getBranches(t, repo)
	if code != http.StatusOK {
		t.Fatalf("status %d", code)
	}
	want := []string{"dev", "feat/SAL-3", "release/2.1"}
	if !reflect.DeepEqual(body.Branches, want) || body.Default != "dev" {
		t.Fatalf("%+v, want %v default dev", body, want)
	}
}

func TestBranchesDefaultIsThePolicyBase(t *testing.T) {
	repo := branchRepo(t)
	if err := os.WriteFile(filepath.Join(repo, "span.yaml"), []byte("feature: app\nbase: feat/SAL-3\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, body := getBranches(t, repo); body.Default != "feat/SAL-3" {
		t.Fatalf("default %q", body.Default)
	}
}

func TestBranchesRefusesAPathThatIsNotAFolder(t *testing.T) {
	for _, path := range []string{"", filepath.Join(t.TempDir(), "missing")} {
		if code, _ := getBranches(t, path); code != http.StatusBadRequest {
			t.Fatalf("path %q: status %d", path, code)
		}
	}
}

func TestBranchesOfAFolderThatIsNotARepoIsEmpty(t *testing.T) {
	code, body := getBranches(t, t.TempDir())
	if code != http.StatusOK || len(body.Branches) != 0 || body.Default != "dev" {
		t.Fatalf("status %d %+v", code, body)
	}
}

// Past the cap, the default and the most recently committed branches come first, so the
// branch a person just pushed is never the one cut off.
func TestBranchesPinTheDefaultAndTheRecentOnes(t *testing.T) {
	repo := branchRepo(t)
	git := func(env []string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		cmd.Env = append(os.Environ(), env...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	for i := 0; i < 210; i++ {
		git(nil, "branch", fmt.Sprintf("a-old-%03d", i), "dev")
	}
	git(nil, "checkout", "-q", "-b", "zz-recent")
	date := []string{"GIT_COMMITTER_DATE=2099-01-01T00:00:00Z", "GIT_AUTHOR_DATE=2099-01-01T00:00:00Z"}
	git(date, "-c", "user.name=T", "-c", "user.email=t@example.test", "commit", "-q", "--allow-empty", "-m", "recent")
	git(nil, "checkout", "-q", "dev")
	_, body := getBranches(t, repo)
	if len(body.Branches) != 200 {
		t.Fatalf("%d branches", len(body.Branches))
	}
	if body.Branches[0] != "dev" || body.Branches[1] != "zz-recent" {
		t.Fatalf("first %v", body.Branches[:3])
	}
	seen := map[string]bool{}
	for _, name := range body.Branches {
		if seen[name] {
			t.Fatalf("%s listed twice", name)
		}
		seen[name] = true
	}
	if rest := body.Branches[51:]; !sort.StringsAreSorted(rest) {
		t.Fatalf("the rest is not alphabetical: %v", rest[:5])
	}
}
