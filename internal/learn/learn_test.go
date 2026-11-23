package learn

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestASealedMistakeIsShownOnTheNextRead(t *testing.T) {
	book := &Book{}
	if text, _ := book.Select("frame"); text != "" {
		t.Fatalf("empty book produced %q", text)
	}
	if !book.Observe("frame", "hold", "Agent reply has no JSON object", nil) {
		t.Fatal("first failure was not stored")
	}
	text, sigs := book.Select("frame")
	if text != "- Agent reply has no JSON object" || len(sigs) != 1 || sigs[0] != "agent reply has no json object" {
		t.Fatalf("text %q sigs %v", text, sigs)
	}
	if _, other := book.Select("build"); other != nil {
		t.Fatalf("build saw a frame lesson: %v", other)
	}
}

func TestOpenQuestionsAndLongReasonsStayOut(t *testing.T) {
	book := &Book{}
	if book.Observe("frame", "hold", "open questions: Which permission atom?", nil) {
		t.Fatal("a story question became a lesson")
	}
	long := strings.Repeat("x", maxRule+1)
	if book.Observe("build", "stop", long, nil) {
		t.Fatal("a long reason became a lesson")
	}
	if book.Observe("prove", "hold", "npm test failed", nil) {
		t.Fatal("prove wrote a lesson")
	}
	if len(book.Lessons) != 0 {
		t.Fatalf("%+v", book.Lessons)
	}
}

func TestTheSameReasonUpdatesOneLesson(t *testing.T) {
	book := &Book{}
	book.Observe("frame", "hold", "agent reply has no JSON object", nil)
	_, shown := book.Select("frame")
	book.Observe("frame", "hold", "agent reply has no JSON object", shown)
	if len(book.Lessons) != 1 || book.Lessons[0].Seen != 2 || book.Lessons[0].Harmful != 1 {
		t.Fatalf("%+v", book.Lessons)
	}
}

func TestAPassingStageCreditsTheShownLessons(t *testing.T) {
	book := &Book{}
	book.Observe("inspect", "hold", "missing spec", nil)
	book.Observe("inspect", "advance", "matches the spec", []string{"missing spec"})
	if book.Lessons[0].Helpful != 1 || book.Lessons[0].Seen != 1 {
		t.Fatalf("%+v", book.Lessons[0])
	}
}

func TestALessonThatFailsTwiceLeavesThePrompt(t *testing.T) {
	book := &Book{}
	reason := "agent reply has no JSON object"
	book.Observe("frame", "hold", reason, nil)
	_, shown := book.Select("frame")
	book.Observe("frame", "hold", reason, shown)
	_, shown = book.Select("frame")
	if !book.Observe("frame", "hold", reason, shown) {
		t.Fatal("third failure did not change the book")
	}
	if len(book.Lessons) != 0 {
		t.Fatalf("retired lesson remained: %+v", book.Lessons)
	}
	text, _ := book.Select("frame")
	if text != "" {
		t.Fatalf("retired lesson was shown: %q", text)
	}
}

func TestSelectKeepsFiveAndRanksAUsefulLessonFirst(t *testing.T) {
	book := &Book{}
	for i, reason := range []string{"one", "two", "three", "four", "five", "six"} {
		book.Lessons = append(book.Lessons, Lesson{Stage: "build", Signature: reason, Rule: reason, Seen: i + 1})
	}
	book.Lessons[1].Helpful = 3
	text, sigs := book.Select("build")
	if len(sigs) != maxShown || sigs[0] != "two" {
		t.Fatalf("%v\n%s", sigs, text)
	}
	for _, sig := range sigs {
		if sig == "one" {
			t.Fatal("the lowest lesson took a slot")
		}
	}
}

func TestAFullBookKeepsARepeatedLesson(t *testing.T) {
	book := &Book{}
	for i := 0; i < maxLessons; i++ {
		book.Lessons = append(book.Lessons, Lesson{
			Stage: "frame", Signature: string(rune('a' + i)), Rule: string(rune('a' + i)), Seen: 2, Helpful: 1,
		})
	}
	if book.Observe("frame", "hold", "brand new mistake", nil) {
		t.Fatal("a one-off evicted a repeated lesson")
	}
	if len(book.Lessons) != maxLessons {
		t.Fatalf("len %d", len(book.Lessons))
	}
}

func TestLoadAndSaveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".span", "lessons.json")
	book := &Book{Lessons: []Lesson{{Stage: "build", Signature: "missing feature document", Rule: "missing feature document", Seen: 2}}}
	if err := Save(path, book); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil || len(loaded.Lessons) != 1 || loaded.Lessons[0].Seen != 2 {
		t.Fatalf("%v %+v", err, loaded)
	}
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Fatal("temp file was left behind")
	}
}

func TestAMissingFileIsEmptyAndACorruptFileIsAnError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lessons.json")
	book, err := Load(path)
	if err != nil || len(book.Lessons) != 0 {
		t.Fatalf("%v %+v", err, book)
	}
	if err := os.WriteFile(path, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("corrupt ledger was accepted")
	}
}
