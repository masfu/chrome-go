// Package cdp provides a low-level Chrome DevTools Protocol transport.
package cdp

import (
	"encoding/json"
	"fmt"
)

// Message represents a CDP command to be sent to the browser.
type Message struct {
	// Method is the CDP method name, e.g. "Page.navigate".
	Method string
	// Params holds the parameters for the method.
	Params map[string]any
}

// request is the wire format sent over WebSocket.
type request struct {
	ID        int            `json:"id"`
	SessionID string         `json:"sessionId,omitempty"`
	Method    string         `json:"method"`
	Params    map[string]any `json:"params,omitempty"`
}

// Response is a CDP response received from the browser.
type Response struct {
	ID     int             `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *ResponseError  `json:"error,omitempty"`
}

// ResponseError represents a CDP error response.
type ResponseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

func (e *ResponseError) Error() string {
	if e.Data != "" {
		return fmt.Sprintf("cdp error %d: %s (%s)", e.Code, e.Message, e.Data)
	}
	return fmt.Sprintf("cdp error %d: %s", e.Code, e.Message)
}

// Event is a CDP event pushed from the browser.
type Event struct {
	Method    string          `json:"method"`
	Params    json.RawMessage `json:"params,omitempty"`
	SessionID string          `json:"sessionId,omitempty"`
}

// ResponseReader allows callers to await a CDP response.
type ResponseReader struct {
	ch <-chan *Response
}

// NewResponseReader wraps ch in a ResponseReader. The caller retains ownership
// of the write end of the channel and is responsible for sending exactly one
// *Response and/or closing the channel.
func NewResponseReader(ch <-chan *Response) *ResponseReader {
	return &ResponseReader{ch: ch}
}

// Read blocks until the CDP response arrives and returns it.
func (r *ResponseReader) Read() (*Response, error) {
	resp, ok := <-r.ch
	if !ok {
		return nil, fmt.Errorf("cdp: response channel closed unexpectedly")
	}
	if resp.Error != nil {
		return nil, resp.Error
	}
	return resp, nil
}
