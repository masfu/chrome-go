package cdp_test

import (
	"encoding/json"
	"testing"

	"github.com/masfu/chrome-go/cdp"
)

// ---- ResponseError ----

func TestResponseError_WithData(t *testing.T) {
	e := &cdp.ResponseError{Code: -32000, Message: "session not found", Data: "sessionId=abc"}
	got := e.Error()
	want := "cdp error -32000: session not found (sessionId=abc)"
	if got != want {
		t.Errorf("ResponseError.Error() with data:\n got  %q\n want %q", got, want)
	}
}

func TestResponseError_WithoutData(t *testing.T) {
	e := &cdp.ResponseError{Code: -32601, Message: "method not found"}
	got := e.Error()
	want := "cdp error -32601: method not found"
	if got != want {
		t.Errorf("ResponseError.Error() without data:\n got  %q\n want %q", got, want)
	}
}

func TestResponseError_ImplementsError(t *testing.T) {
	var _ error = &cdp.ResponseError{}
}

// ---- ResponseReader ----

func TestResponseReader_Read_Success(t *testing.T) {
	ch := make(chan *cdp.Response, 1)
	raw := json.RawMessage(`{"value":42}`)
	ch <- &cdp.Response{ID: 1, Result: raw}

	rr := cdp.NewResponseReader(ch)
	resp, err := rr.Read()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ID != 1 {
		t.Errorf("ID: want 1, got %d", resp.ID)
	}
}

func TestResponseReader_Read_CDPError(t *testing.T) {
	ch := make(chan *cdp.Response, 1)
	ch <- &cdp.Response{ID: 2, Error: &cdp.ResponseError{Code: -32000, Message: "boom"}}

	rr := cdp.NewResponseReader(ch)
	_, err := rr.Read()
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	re, ok := err.(*cdp.ResponseError)
	if !ok {
		t.Fatalf("want *cdp.ResponseError, got %T", err)
	}
	if re.Code != -32000 {
		t.Errorf("Code: want -32000, got %d", re.Code)
	}
}

func TestResponseReader_Read_ClosedChannel(t *testing.T) {
	ch := make(chan *cdp.Response)
	close(ch)

	rr := cdp.NewResponseReader(ch)
	_, err := rr.Read()
	if err == nil {
		t.Fatal("expected an error from closed channel, got nil")
	}
}

// ---- Event JSON parsing ----

func TestEvent_JSONParsing(t *testing.T) {
	raw := `{"method":"Page.loadEventFired","params":{"timestamp":1234.5},"sessionId":"s1"}`
	var evt cdp.Event
	if err := json.Unmarshal([]byte(raw), &evt); err != nil {
		t.Fatalf("unmarshal event: %v", err)
	}
	if evt.Method != "Page.loadEventFired" {
		t.Errorf("Method: want Page.loadEventFired, got %q", evt.Method)
	}
	if evt.SessionID != "s1" {
		t.Errorf("SessionID: want s1, got %q", evt.SessionID)
	}
	if len(evt.Params) == 0 {
		t.Error("Params should not be empty")
	}
}

// ---- NewConnection ----

func TestNewConnection_NotNil(t *testing.T) {
	c := cdp.NewConnection("ws://localhost:9222/devtools/browser/abc")
	if c == nil {
		t.Fatal("NewConnection returned nil")
	}
}

func TestNewConnection_SetConnectionDelay(t *testing.T) {
	c := cdp.NewConnection("ws://x")
	// SetConnectionDelay must not panic.
	c.SetConnectionDelay(50)
}
