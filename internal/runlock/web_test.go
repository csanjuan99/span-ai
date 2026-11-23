package runlock

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The page recognises a busy-checkout hold by the first words of the server's message.
// The words live twice, here and in web/src/lib/failure.ts, so this test fails the moment
// they drift apart instead of the page quietly showing a generic retry.
func TestThePageRecognisesTheMessageTheServerSends(t *testing.T) {
	path := filepath.Join("..", "..", "web", "src", "lib", "failure.ts")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("the page source is part of this repository and must be readable: %v", err)
	}
	found := regexp.MustCompile(`checkoutBusyPrefix\s*=\s*"([^"]+)"`).FindSubmatch(raw)
	if found == nil {
		t.Fatal("web/src/lib/failure.ts no longer declares checkoutBusyPrefix")
	}
	sent := (&BusyError{Scope: Workspace, Holder: "pid 1 run span-1"}).Error()
	if !strings.HasPrefix(sent, string(found[1])) {
		t.Fatalf("the server sends %q but the page looks for %q", sent, found[1])
	}
	if !IsWorkspaceBusy(string(found[1])) {
		t.Fatalf("the page's prefix %q is not recognised by the server's own check", found[1])
	}
}
