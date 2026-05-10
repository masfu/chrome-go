package cdp_test

import (
	"testing"

	"github.com/masfu/chrome-go/cdp"
)

func TestSession_ID(t *testing.T) {
	s := cdp.NewSession("session-xyz", nil)
	if s.ID() != "session-xyz" {
		t.Errorf("ID(): want session-xyz, got %q", s.ID())
	}
}

func TestSession_AddListener_Called(t *testing.T) {
	s := cdp.NewSession("s1", nil)

	called := 0
	s.AddListener(func(e *cdp.Event) { called++ })

	s.Dispatch(&cdp.Event{Method: "Page.loadEventFired"})

	if called != 1 {
		t.Errorf("listener called %d times, want 1", called)
	}
}

func TestSession_AddListener_MultipleListeners(t *testing.T) {
	s := cdp.NewSession("s1", nil)
	counts := [3]int{}
	for i := range counts {
		i := i
		s.AddListener(func(e *cdp.Event) { counts[i]++ })
	}

	s.Dispatch(&cdp.Event{Method: "Target.targetCreated"})

	for i, c := range counts {
		if c != 1 {
			t.Errorf("listener[%d] called %d times, want 1", i, c)
		}
	}
}

func TestSession_Dispatch_NoListeners(t *testing.T) {
	s := cdp.NewSession("s1", nil)
	// Must not panic.
	s.Dispatch(&cdp.Event{Method: "Runtime.consoleAPICalled"})
}
