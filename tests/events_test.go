package chrome_test

import (
	"testing"

	chrome "github.com/masfu/chrome-go"
)

func TestPageEvent_Values(t *testing.T) {
	cases := []struct {
		event chrome.PageEvent
		want  string
	}{
		{chrome.EventDOMContentLoaded, "DOMContentLoaded"},
		{chrome.EventFirstContentfulPaint, "firstContentfulPaint"},
		{chrome.EventFirstImagePaint, "firstImagePaint"},
		{chrome.EventFirstMeaningfulPaint, "firstMeaningfulPaint"},
		{chrome.EventFirstPaint, "firstPaint"},
		{chrome.EventInit, "init"},
		{chrome.EventInteractiveTime, "interactiveTime"},
		{chrome.EventLoad, "load"},
		{chrome.EventNetworkIdle, "networkIdle"},
	}
	for _, tt := range cases {
		if string(tt.event) != tt.want {
			t.Errorf("PageEvent %q: want %q", tt.event, tt.want)
		}
	}
}
