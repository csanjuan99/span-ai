package prove

import (
	"context"
	"errors"
	"testing"

	"span/internal/statement"
)

type script struct {
	calls []string
	fail  map[string]int
	err   map[string]error
}

func (s *script) Run(_ context.Context, command, _ string) (int, error) {
	s.calls = append(s.calls, command)
	if err := s.err[command]; err != nil {
		return 0, err
	}
	return s.fail[command], nil
}

func TestFastFailureSkipsAcceptance(t *testing.T) {
	s := &script{fail: map[string]int{"go test ./...": 1}}
	got, err := Bands(context.Background(), s, t.TempDir(), []string{"go test ./..."}, []string{"go test ./acceptance"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Outcome != "iterate" {
		t.Fatalf("outcome %s", got.Outcome)
	}
	if len(s.calls) != 1 {
		t.Fatalf("calls %#v", s.calls)
	}
}

func TestProseHoldsWithoutRunning(t *testing.T) {
	s := &script{}
	got, err := Bands(context.Background(), s, t.TempDir(), []string{"go test ./... | wc"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Outcome != statement.Hold || len(s.calls) != 0 {
		t.Fatalf("outcome %s calls %#v", got.Outcome, s.calls)
	}
}

func TestStartErrorHolds(t *testing.T) {
	s := &script{err: map[string]error{"missing": errors.New("executable not found")}}
	got, err := Bands(context.Background(), s, t.TempDir(), []string{"missing"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Outcome != statement.Hold {
		t.Fatalf("outcome %s", got.Outcome)
	}
}

func TestBothBandsPass(t *testing.T) {
	s := &script{}
	got, err := Bands(context.Background(), s, t.TempDir(), []string{"go test ./..."}, []string{"go test ./acceptance"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Outcome != statement.Advance || len(got.Commands) != 2 {
		t.Fatalf("%+v", got)
	}
}
