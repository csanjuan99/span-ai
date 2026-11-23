package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"span/internal/engine"
	"span/internal/runlock"
	"span/internal/setup"
)

func goProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestConfigureWaitsForTheCheckoutInsteadOfRewritingItUnderARun(t *testing.T) {
	dir := goProject(t)
	owner, err := engine.ClaimWorkspace(dir, "run span-1")
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Release()

	_, _, err = claimAndConfigure(setup.Request{Workspace: dir, Story: "AET-1 Charge", Kind: "fix"}, "starting a run")
	var busy *runlock.BusyError
	if !errors.As(err, &busy) {
		t.Fatalf("err = %v, want a BusyError", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "span.yaml")); statErr == nil {
		t.Fatal("span.yaml was written into a checkout another run owns")
	}
}

func TestClaimAndConfigureHoldsTheCheckoutUntilTheCallerLetsGo(t *testing.T) {
	dir := goProject(t)
	res, claim, err := claimAndConfigure(setup.Request{Workspace: dir, Story: "AET-1 Charge", Kind: "fix"}, "starting a run")
	if err != nil {
		t.Fatal(err)
	}
	if !res.Wrote {
		t.Fatalf("%+v", res)
	}
	if _, busy := engine.WorkspaceBusy(dir); !busy {
		t.Fatal("the checkout was free while the caller still held the claim")
	}
	claim.Release()
	if holder, busy := engine.WorkspaceBusy(dir); busy {
		t.Fatalf("still held by %q after release", holder)
	}
}

func TestAFailedConfigureReleasesTheClaim(t *testing.T) {
	dir := goProject(t)
	// A kind that is not feat, fix, or hotfix makes Configure fail after the claim.
	_, claim, err := claimAndConfigure(setup.Request{Workspace: dir, Story: "AET-1 Charge", Kind: "bogus"}, "starting a run")
	if err == nil {
		claim.Release()
		t.Fatal("expected Configure to refuse the kind")
	}
	if holder, busy := engine.WorkspaceBusy(dir); busy {
		t.Fatalf("a failed configure left the checkout held by %q", holder)
	}
}
