package agent

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

const cursorOut = "Available models\n\nauto - Auto (default)\ngpt-5.3-codex-low - Codex 5.3 Low\ngpt-5.3-codex - Codex 5.3\n--evil - Not a model\n$(id) - Nope\ngpt-5.3-codex - Duplicate\n"

const piOut = "provider  model                      context  max-out  thinking  images\n" +
	"ollama    deepseek-v4.1-flash:cloud  1.0M     16.4K    yes       yes   \n" +
	"xai       grok-4.7                   500K     500K     yes       yes   \n"

func TestCursorModelsAreReadFromItsListAndUnsafeNamesAreDropped(t *testing.T) {
	var got []string
	for _, m := range ParseModels("cursor", cursorOut) {
		got = append(got, m.ID+"="+m.Label)
	}
	want := "auto=Auto (default),gpt-5.3-codex-low=Codex 5.3 Low,gpt-5.3-codex=Codex 5.3"
	if strings.Join(got, ",") != want {
		t.Fatalf("got %q", got)
	}
}

func TestPiModelsAreProviderSlashModel(t *testing.T) {
	var got []string
	for _, m := range ParseModels("pi", piOut) {
		got = append(got, m.ID)
	}
	if strings.Join(got, ",") != "ollama/deepseek-v4.1-flash:cloud,xai/grok-4.7" {
		t.Fatalf("got %q", got)
	}
}

func TestClaudeOffersAliasesAndCodexOnlyAFreeName(t *testing.T) {
	claude, err := Models(context.Background(), "claude")
	if err != nil || claude.Source != "fixed" || !claude.Free || len(claude.Models) != 3 || claude.Models[0].ID != "opus" {
		t.Fatalf("%+v %v", claude, err)
	}
	codex, err := Models(context.Background(), "codex")
	if err != nil || codex.Source != "none" || !codex.Free || len(codex.Models) != 0 {
		t.Fatalf("%+v %v", codex, err)
	}
}

func TestAListThatFailsStillLetsThePersonTypeAName(t *testing.T) {
	orig := runList
	t.Cleanup(func() { runList = orig; listCache = map[string]cachedList{} })
	listCache = map[string]cachedList{}
	runList = func(context.Context, string, ...string) (string, error) { return "", errors.New("not logged in") }
	list, err := Models(context.Background(), "cursor")
	if err == nil || !list.Free || len(list.Models) != 0 {
		t.Fatalf("%+v %v", list, err)
	}
}

func TestTheListIsAskedOnceAMinute(t *testing.T) {
	orig := runList
	t.Cleanup(func() { runList = orig; listCache = map[string]cachedList{} })
	listCache = map[string]cachedList{}
	calls := 0
	runList = func(context.Context, string, ...string) (string, error) { calls++; return cursorOut, nil }
	for range 3 {
		if _, err := Models(context.Background(), "cursor"); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 1 {
		t.Fatalf("the CLI was asked %d times", calls)
	}
}

func TestAFailedListIsRememberedSoABrokenCLIIsNotSpawnedOnEveryRequest(t *testing.T) {
	orig := runList
	t.Cleanup(func() { runList = orig; listCache = map[string]cachedList{} })
	listCache = map[string]cachedList{}
	calls := 0
	runList = func(context.Context, string, ...string) (string, error) {
		calls++
		return "", errors.New("not logged in")
	}
	for range 3 {
		if _, err := Models(context.Background(), "pi"); err == nil || !strings.Contains(err.Error(), "not logged in") {
			t.Fatalf("the failure was lost: %v", err)
		}
	}
	if calls != 1 {
		t.Fatalf("the broken CLI was spawned %d times", calls)
	}
}

func TestConcurrentRequestsForOneCLIShareOneProcess(t *testing.T) {
	orig := runList
	t.Cleanup(func() { runList = orig; listCache = map[string]cachedList{} })
	listCache = map[string]cachedList{}
	var mu sync.Mutex
	calls := 0
	runList = func(context.Context, string, ...string) (string, error) {
		mu.Lock()
		calls++
		mu.Unlock()
		time.Sleep(50 * time.Millisecond)
		return cursorOut, nil
	}
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := Models(context.Background(), "cursor"); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if calls != 1 {
		t.Fatalf("%d processes for one list", calls)
	}
}
