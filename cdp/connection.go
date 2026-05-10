package cdp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

// Connection manages a WebSocket connection to a Chrome DevTools Protocol endpoint.
type Connection struct {
	wsURI string
	delay time.Duration
	log   *slog.Logger

	conn *websocket.Conn
	mu   sync.RWMutex

	nextID    atomic.Int64
	pending   map[int]chan *Response
	pendingMu sync.Mutex

	sessions   map[string]*Session
	sessionsMu sync.RWMutex

	listeners   []func(*Event)
	listenersMu sync.RWMutex

	closeCh chan struct{}
}

// NewConnection creates a new CDP connection to the given WebSocket URI.
func NewConnection(wsURI string) *Connection {
	return &Connection{
		wsURI:    wsURI,
		pending:  make(map[int]chan *Response),
		sessions: make(map[string]*Session),
		closeCh:  make(chan struct{}),
		log:      slog.Default(),
	}
}

// SetConnectionDelay sets an artificial delay between messages (for debugging).
func (c *Connection) SetConnectionDelay(d time.Duration) {
	c.delay = d
}

// SetLogger sets the logger for this connection.
func (c *Connection) SetLogger(l *slog.Logger) {
	c.log = l
}

// Connect establishes the WebSocket connection and starts reading messages.
func (c *Connection) Connect(ctx context.Context) error {
	conn, _, err := websocket.Dial(ctx, c.wsURI, nil)
	if err != nil {
		return fmt.Errorf("cdp: dial %s: %w", c.wsURI, err)
	}
	conn.SetReadLimit(32 << 20) // 32 MiB
	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	go c.readLoop(context.Background())
	return nil
}

// Close closes the WebSocket connection.
func (c *Connection) Close() error {
	select {
	case <-c.closeCh:
	default:
		close(c.closeCh)
	}
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()
	if conn == nil {
		return nil
	}
	return conn.Close(websocket.StatusNormalClosure, "closing")
}

// SendMessage sends a CDP message and returns a ResponseReader to await the reply.
func (c *Connection) SendMessage(ctx context.Context, m Message) (*ResponseReader, error) {
	return c.sendMessage(ctx, "", m)
}

func (c *Connection) sendMessage(ctx context.Context, sessionID string, m Message) (*ResponseReader, error) {
	id := int(c.nextID.Add(1))
	req := request{
		ID:        id,
		SessionID: sessionID,
		Method:    m.Method,
		Params:    m.Params,
	}

	ch := make(chan *Response, 1)
	c.pendingMu.Lock()
	c.pending[id] = ch
	c.pendingMu.Unlock()

	if c.delay > 0 {
		time.Sleep(c.delay)
	}

	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()
	if conn == nil {
		c.pendingMu.Lock()
		delete(c.pending, id)
		c.pendingMu.Unlock()
		return nil, fmt.Errorf("cdp: not connected")
	}

	if err := wsjson.Write(ctx, conn, req); err != nil {
		c.pendingMu.Lock()
		delete(c.pending, id)
		c.pendingMu.Unlock()
		return nil, fmt.Errorf("cdp: write message: %w", err)
	}

	return &ResponseReader{ch: ch}, nil
}

// AddListener registers a function that is called on every incoming CDP event.
func (c *Connection) AddListener(fn func(*Event)) {
	c.listenersMu.Lock()
	c.listeners = append(c.listeners, fn)
	c.listenersMu.Unlock()
}

// CreateSession creates a new CDP session for the given target ID.
func (c *Connection) CreateSession(targetID string) (*Session, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	reader, err := c.SendMessage(ctx, Message{
		Method: "Target.attachToTarget",
		Params: map[string]any{
			"targetId": targetID,
			"flatten":  true,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("cdp: attach to target: %w", err)
	}

	resp, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("cdp: attach to target response: %w", err)
	}

	var result struct {
		SessionID string `json:"sessionId"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("cdp: parse session id: %w", err)
	}

	session := NewSession(result.SessionID, c)
	c.sessionsMu.Lock()
	c.sessions[result.SessionID] = session
	c.sessionsMu.Unlock()

	return session, nil
}

// GetSession returns the session with the given ID.
func (c *Connection) GetSession(sessionID string) (*Session, bool) {
	c.sessionsMu.RLock()
	s, ok := c.sessions[sessionID]
	c.sessionsMu.RUnlock()
	return s, ok
}

func (c *Connection) readLoop(ctx context.Context) {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	for {
		var raw json.RawMessage
		if err := wsjson.Read(ctx, conn, &raw); err != nil {
			select {
			case <-c.closeCh:
				return
			default:
			}
			c.log.Error("cdp: read error", "err", err)
			// drain pending with error
			c.pendingMu.Lock()
			for id, ch := range c.pending {
				ch <- &Response{
					ID:    id,
					Error: &ResponseError{Code: -1, Message: err.Error()},
				}
				delete(c.pending, id)
			}
			c.pendingMu.Unlock()
			return
		}

		// Peek at the raw message to determine whether it is a response or an event.
		var peek struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
		}
		if err := json.Unmarshal(raw, &peek); err != nil {
			c.log.Warn("cdp: unmarshal peek failed", "err", err)
			continue
		}

		if peek.ID != 0 {
			// It's a response.
			var resp Response
			if err := json.Unmarshal(raw, &resp); err != nil {
				c.log.Warn("cdp: unmarshal response failed", "err", err)
				continue
			}
			c.pendingMu.Lock()
			ch, ok := c.pending[resp.ID]
			if ok {
				delete(c.pending, resp.ID)
			}
			c.pendingMu.Unlock()
			if ok {
				ch <- &resp
			}
			continue
		}

		// It's an event.
		var evt Event
		if err := json.Unmarshal(raw, &evt); err != nil {
			c.log.Warn("cdp: unmarshal event failed", "err", err)
			continue
		}

		// Route to session if it has a sessionId.
		if evt.SessionID != "" {
			c.sessionsMu.RLock()
			s, ok := c.sessions[evt.SessionID]
			c.sessionsMu.RUnlock()
			if ok {
				s.Dispatch(&evt)
				continue
			}
		}

		// Broadcast to connection-level listeners.
		c.listenersMu.RLock()
		for _, fn := range c.listeners {
			fn(&evt)
		}
		c.listenersMu.RUnlock()
	}
}
