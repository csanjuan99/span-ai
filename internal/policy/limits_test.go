package policy

import (
	"testing"
	"time"
)

func TestStageTimeoutIsReadFromTheYAMLAndAbsentMeansNone(t *testing.T) {
	got, err := Parse([]byte("limits:\n  stageTimeout: 45m\n"))
	if err != nil {
		t.Fatal(err)
	}
	if got.StageTimeout() != 45*time.Minute {
		t.Fatalf("%v", got.StageTimeout())
	}
	none, err := Parse([]byte("agent: any\n"))
	if err != nil {
		t.Fatal(err)
	}
	if none.StageTimeout() != 0 {
		t.Fatalf("an absent limit is %v, want none", none.StageTimeout())
	}
}

func TestStageTimeoutRefusesWhatIsNotASaneDuration(t *testing.T) {
	for _, bad := range []string{"soon", "-5m", "0s", "500ms", "49h"} {
		if _, err := Parse([]byte("limits:\n  stageTimeout: " + bad + "\n")); err == nil {
			t.Fatalf("%q was accepted", bad)
		}
	}
}
