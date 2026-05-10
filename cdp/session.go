package cdp

import (
	"context"
	"fmt"
	"sync"
)

// Session represents a CDP session attached to a specific target (tab/page).
type Session struct {
	id   string
	conn *Connection

	listeners   []func(*Event)
	listenersMu sync.RWMutex
}

func NewSession(id string, conn *Connection) *Session {
	return &Session{id: id, conn: conn}
}

// ID returns the session identifier.
func (s *Session) ID() string { return s.id }

// SendMessageSync sends a CDP message and blocks until a response is received.
func (s *Session) SendMessageSync(ctx context.Context, m Message) (*Response, error) {
	reader, err := s.conn.sendMessage(ctx, s.id, m)
	if err != nil {
		return nil, err
	}
	resp, err := reader.Read()
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// SendMessage sends a CDP message and returns a ResponseReader to await the reply.
func (s *Session) SendMessage(ctx context.Context, m Message) (*ResponseReader, error) {
	return s.conn.sendMessage(ctx, s.id, m)
}

// AddListener registers a function that is called on every event in this session.
func (s *Session) AddListener(fn func(*Event)) {
	s.listenersMu.Lock()
	s.listeners = append(s.listeners, fn)
	s.listenersMu.Unlock()
}

// Dispatch delivers an event to all registered listeners for this session.
func (s *Session) Dispatch(evt *Event) {
	s.listenersMu.RLock()
	fns := make([]func(*Event), len(s.listeners))
	copy(fns, s.listeners)
	s.listenersMu.RUnlock()
	for _, fn := range fns {
		fn(evt)
	}
}

// WaitForEvent blocks until an event matching the given method arrives or the context is cancelled.
func (s *Session) WaitForEvent(ctx context.Context, method string) (*Event, error) {
	ch := make(chan *Event, 1)
	s.AddListener(func(evt *Event) {
		if evt.Method == method {
			select {
			case ch <- evt:
			default:
			}
		}
	})
	select {
	case evt := <-ch:
		return evt, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("cdp: wait for event %q: %w", method, ctx.Err())
	}
}
