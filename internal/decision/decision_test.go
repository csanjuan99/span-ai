package decision

import (
	"strings"
	"testing"
	"time"

	"span/internal/blocker"
)

func option(id, then string) Option {
	return Option{ID: id, Label: "L " + id, Consequence: "C " + id, Then: then}
}

func TestValidate(t *testing.T) {
	base := func() Request {
		return Request{Title: "t", Happened: "h", Options: []Option{option("a", Build), option("b", Stop)}}
	}
	tests := []struct {
		name    string
		mutate  func(*Request)
		wantErr string
	}{
		{name: "valid", mutate: func(*Request) {}},
		{name: "no title", mutate: func(r *Request) { r.Title = " " }, wantErr: "title"},
		{name: "one option", mutate: func(r *Request) { r.Options = r.Options[:1] }, wantErr: "2 to 4"},
		{name: "five options", mutate: func(r *Request) {
			r.Options = []Option{option("a", Build), option("b", Stop), option("c", Retry), option("d", Retry), option("e", Retry)}
		}, wantErr: "2 to 4"},
		{name: "duplicate id", mutate: func(r *Request) { r.Options[1].ID = "a" }, wantErr: "twice"},
		{name: "no consequence", mutate: func(r *Request) { r.Options[0].Consequence = "" }, wantErr: "consequence"},
		{name: "an action Span does not offer", mutate: func(r *Request) { r.Options[0].Then = "skip-hook" }, wantErr: "does something Span does not offer"},
		{name: "recommendation outside the options", mutate: func(r *Request) { r.Recommendation = "z" }, wantErr: "not an option"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := base()
			tt.mutate(&r)
			err := r.Validate()
			if tt.wantErr == "" && err != nil {
				t.Fatalf("err = %v", err)
			}
			if tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)) {
				t.Fatalf("err = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestParseKeepsTheWordsAndTakesTheIdentityFromSpan(t *testing.T) {
	raw := `{"class":"commit-rejected","id":"forged","dedupe":"forged","title":"T","happened":"H","evidence":[{"ref":"a:1","excerpt":"x"}],
"options":[{"id":"a","label":"A","consequence":"c","then":"build"},{"id":"b","label":"B","consequence":"c","then":"stop"}]}`
	got, err := Parse([]byte(raw), "build", time.Unix(1, 0))
	if err != nil {
		t.Fatal(err)
	}
	if got.Class != Question || got.ID == "forged" || got.Dedupe == "forged" || got.Stage != "build" {
		t.Fatalf("the agent chose its own identity: %+v", got)
	}
	again, _ := Parse([]byte(raw), "build", time.Unix(2, 0))
	if again.Dedupe != got.Dedupe {
		t.Fatal("the same problem got two dedupe keys")
	}
	if _, err := Parse([]byte("not json"), "build", time.Now()); err == nil {
		t.Fatal("garbage was accepted")
	}
}

func TestChoose(t *testing.T) {
	r := Request{Options: []Option{option("a", Build)}}
	if got, err := r.Choose(Answer{Option: "a"}); err != nil || got.ID != "a" {
		t.Fatalf("got %+v %v", got, err)
	}
	if _, err := r.Choose(Answer{Option: "x"}); err == nil {
		t.Fatal("an unknown option was chosen")
	}
}

func TestACommitDecisionOffersNoWayAroundTheHook(t *testing.T) {
	b := &blocker.Blocker{Class: blocker.CommitRejected, Fingerprint: strings.Repeat("ab", 32), Count: 2, Evidence: "rule x"}
	req := FromCommit(b, "build", time.Unix(1, 0))
	if err := req.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, o := range req.Options {
		if !strings.Contains("build retry stop", o.Then) {
			t.Fatalf("option %+v", o)
		}
	}
}
