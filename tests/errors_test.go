package chrome_test

import (
	"errors"
	"testing"

	chrome "github.com/masfu/chrome-go"
)

func TestSentinelErrors_Distinct(t *testing.T) {
	errs := []error{
		chrome.ErrOperationTimedOut,
		chrome.ErrNavigationExpired,
		chrome.ErrBrowserConnection,
		chrome.ErrElementNotFound,
		chrome.ErrBrowserClosed,
	}

	for i, a := range errs {
		for j, b := range errs {
			if i == j {
				continue
			}
			if errors.Is(a, b) {
				t.Errorf("errors.Is(%v, %v) should be false but is true", a, b)
			}
		}
	}
}

func TestSentinelErrors_ErrorsIs_Join(t *testing.T) {
	wrapped := errors.Join(chrome.ErrOperationTimedOut, errors.New("context"))
	if !errors.Is(wrapped, chrome.ErrOperationTimedOut) {
		t.Error("errors.Is should match ErrOperationTimedOut in joined error")
	}
}

func TestSentinelErrors_Messages(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{chrome.ErrOperationTimedOut, "chrome-go: operation timed out"},
		{chrome.ErrNavigationExpired, "chrome-go: navigation expired"},
		{chrome.ErrBrowserConnection, "chrome-go: browser connection failed"},
		{chrome.ErrElementNotFound, "chrome-go: element not found"},
		{chrome.ErrBrowserClosed, "chrome-go: browser is closed"},
	}
	for _, tt := range cases {
		if tt.err.Error() != tt.want {
			t.Errorf("error message: got %q, want %q", tt.err.Error(), tt.want)
		}
	}
}
