package cdp

import (
	"testing"
)

func TestSession_ID(t *testing.T) {
	s := NewSession("session-xyz", nil)
	if s.ID() != "session-xyz" {
		t.Errorf("ID(): want session-xyz, got %q", s.ID())
	}
}

func TestSession_AddListener_Called(t *testing.T) {
	s := NewSession("s1", nil)

	called := 0
	s.AddListener(func(e *Event) { called++ })

	evt := &Event{Method: "Page.loadEventFired"}
	s.Dispatch(evt)

	if called != 1 {
		t.Errorf("listener called %d times, want 1", called)
	}
}

func TestSession_AddListener_MultipleListeners(t *testing.T) {
	s := NewSession("s1", nil)
	counts := [3]int{}
	for i := range counts {
		i := i
		s.AddListener(func(e *Event) { counts[i]++ })
	}

	s.Dispatch(&Event{Method: "Target.targetCreated"})

	for i, c := range counts {
		if c != 1 {
			t.Errorf("listener[%d] called %d times, want 1", i, c)
		}
	}
}

func TestSession_Dispatch_NoListeners(t *testing.T) {
	s := NewSession("s1", nil)
	// Must not panic.
	s.Dispatch(&Event{Method: "Runtime.consoleAPICalled"})
}
