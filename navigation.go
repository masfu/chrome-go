package chrome

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/masfu/chrome-go/cdp"
)

// Navigation is returned by Page.Navigate and allows waiting for the page to load.
//
// Upstream equivalent: HeadlessChromium\PageLoader\PageLoader
type Navigation struct {
	page   *Page
	reader *cdp.ResponseReader
}

// WaitForNavigation blocks until the page has finished loading for the given events.
// If no events are provided, EventLoad with a 30-second timeout is used.
//
// Upstream equivalent: $navigation->waitForNavigation($events...).
func (n *Navigation) WaitForNavigation(events ...PageEvent) error {
	if len(events) == 0 {
		events = []PageEvent{EventLoad}
	}

	timeout := n.page.browser.opts.SendSyncDefaultTimeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	// First, make sure the navigate command itself succeeded.
	resp, err := n.reader.Read()
	if err != nil {
		return fmt.Errorf("%w: navigate command: %v", ErrNavigationExpired, err)
	}

	var navResult struct {
		ErrorText string `json:"errorText"`
	}
	if err := json.Unmarshal(resp.Result, &navResult); err == nil && navResult.ErrorText != "" {
		return fmt.Errorf("%w: %s", ErrNavigationExpired, navResult.ErrorText)
	}

	// Then wait for the requested lifecycle events.
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for _, event := range events {
		if err := waitForPageEvent(ctx, n.page.session, event); err != nil {
			return fmt.Errorf("%w: %v", ErrOperationTimedOut, err)
		}
	}
	return nil
}

// waitForPageEvent blocks until the given page lifecycle event fires.
func waitForPageEvent(ctx context.Context, session *cdp.Session, event PageEvent) error {
	switch event {
	case EventLoad:
		_, err := session.WaitForEvent(ctx, "Page.loadEventFired")
		return err
	case EventDOMContentLoaded:
		_, err := session.WaitForEvent(ctx, "Page.domContentEventFired")
		return err
	case EventNetworkIdle:
		// Wait for Page.lifecycleEvent with name "networkIdle".
		for {
			evt, err := session.WaitForEvent(ctx, "Page.lifecycleEvent")
			if err != nil {
				return err
			}
			var lce struct {
				Name string `json:"name"`
			}
			if err := json.Unmarshal(evt.Params, &lce); err == nil && lce.Name == "networkIdle" {
				return nil
			}
		}
	default:
		// For other lifecycle events, listen for Page.lifecycleEvent and match by name.
		for {
			evt, err := session.WaitForEvent(ctx, "Page.lifecycleEvent")
			if err != nil {
				return err
			}
			var lce struct {
				Name string `json:"name"`
			}
			if err := json.Unmarshal(evt.Params, &lce); err == nil && lce.Name == string(event) {
				return nil
			}
		}
	}
}
