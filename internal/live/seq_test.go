package live

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"span/internal/engine"
)

func readSeq(t *testing.T, raw []byte) uint64 {
	t.Helper()
	var event engine.Event
	if err := json.Unmarshal(raw, &event); err != nil {
		t.Fatal(err)
	}
	return event.Seq
}

// drain counts the frames left in ch and fails when the hub never closes it.
func drain(t *testing.T, ch <-chan []byte) int {
	t.Helper()
	count := 0
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				return count
			}
			count++
		case <-time.After(time.Second):
			t.Fatalf("the channel was never closed after %d frames", count)
		}
	}
}

func TestPublishStampsAStrictlyIncreasingSeqPerRun(t *testing.T) {
	hub := New()
	one, detachOne := hub.Subscribe("s1")
	defer detachOne()
	two, detachTwo := hub.Subscribe("s2")
	defer detachTwo()
	event := engine.Event{Type: "stage", Stage: "frame"}
	hub.Publish("s1", event)
	hub.Publish("s1", event)
	hub.Publish("s2", event)
	hub.Publish("s1", event)
	if event.Seq != 0 {
		t.Fatalf("the caller's event was mutated: %d", event.Seq)
	}
	for want := uint64(1); want <= 3; want++ {
		if got := readSeq(t, <-one); got != want {
			t.Fatalf("s1 seq %d, want %d", got, want)
		}
	}
	if got := readSeq(t, <-two); got != 1 {
		t.Fatalf("s2 seq %d, want 1", got)
	}
	if hub.Seq("s1") != 3 || hub.Seq("s2") != 1 || hub.Seq("absent") != 0 {
		t.Fatalf("seq s1=%d s2=%d absent=%d", hub.Seq("s1"), hub.Seq("s2"), hub.Seq("absent"))
	}
}

func TestAFullSubscriberIsCutInsteadOfSkipped(t *testing.T) {
	hub := New()
	slow, detachSlow := hub.Subscribe("s1")
	defer detachSlow()
	fast, detachFast := hub.Subscribe("s1")
	defer detachFast()
	for want := uint64(1); want <= 40; want++ {
		hub.Publish("s1", engine.Event{Type: "lines"})
		if got := readSeq(t, <-fast); got != want {
			t.Fatalf("fast subscriber seq %d, want %d", got, want)
		}
	}
	count := drain(t, slow)
	if count != 32 {
		t.Fatalf("the slow subscriber got %d frames before the cut, want 32", count)
	}
}

func TestDetachAfterTheHubCutTheSubscriberIsSafe(t *testing.T) {
	hub := New()
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		_, detach := hub.Subscribe("s1")
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				hub.Publish("s1", engine.Event{Type: "lines"})
			}
			detach()
			detach()
		}()
	}
	wg.Wait()
	if hub.Seq("s1") != 20*50 {
		t.Fatalf("seq %d", hub.Seq("s1"))
	}
}
