package chrome

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"sync"

	"github.com/masfu/chrome-go/cdp"
)

// Browser represents a running Chrome/Chromium instance.
//
// Upstream equivalent: HeadlessChromium\Browser\Browser
type Browser struct {
	conn    *cdp.Connection
	wsURI   string
	opts    Options
	cmd     *exec.Cmd // nil when using ConnectToBrowser
	closeCh chan struct{}

	preScript string
	pagesMu   sync.RWMutex
	pages     []*Page
}

// CreatePage opens a new tab and returns a Page ready for use.
//
// Upstream equivalent: $browser->createPage().
func (b *Browser) CreatePage(ctx context.Context) (*Page, error) {
	select {
	case <-b.closeCh:
		return nil, ErrBrowserClosed
	default:
	}

	// Create a new target (tab).
	reader, err := b.conn.SendMessage(ctx, cdp.Message{
		Method: "Target.createTarget",
		Params: map[string]any{"url": "about:blank"},
	})
	if err != nil {
		return nil, fmt.Errorf("chrome-go: create target: %w", err)
	}
	resp, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("chrome-go: create target response: %w", err)
	}

	var result struct {
		TargetID string `json:"targetId"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("chrome-go: parse target id: %w", err)
	}

	session, err := b.conn.CreateSession(result.TargetID)
	if err != nil {
		return nil, fmt.Errorf("chrome-go: create session: %w", err)
	}

	page := newPage(session, b, result.TargetID)

	// Apply page pre-script if set.
	if b.preScript != "" {
		if err := page.AddPreScript(b.preScript); err != nil {
			return nil, fmt.Errorf("chrome-go: apply page pre-script: %w", err)
		}
	}

	b.pagesMu.Lock()
	b.pages = append(b.pages, page)
	b.pagesMu.Unlock()

	return page, nil
}

// Pages returns the list of currently open pages.
//
// Upstream equivalent: $browser->getPages().
func (b *Browser) Pages() []*Page {
	b.pagesMu.RLock()
	defer b.pagesMu.RUnlock()
	out := make([]*Page, len(b.pages))
	copy(out, b.pages)
	return out
}

// Close closes the browser. If the browser was launched by this library the
// underlying process is also killed.
//
// Upstream equivalent: $browser->close().
func (b *Browser) Close() error {
	select {
	case <-b.closeCh:
		return nil
	default:
		close(b.closeCh)
	}

	connErr := b.conn.Close()
	if b.cmd != nil && b.cmd.Process != nil {
		if err := b.cmd.Process.Kill(); err != nil && connErr == nil {
			return err
		}
		b.cmd.Wait() //nolint:errcheck
	}
	return connErr
}

// SetPagePreScript sets a JavaScript snippet that will be evaluated in every
// new page before any other script runs.
//
// Upstream equivalent: $browser->setPagePreScript($js).
func (b *Browser) SetPagePreScript(js string) error {
	b.preScript = js
	return nil
}

// SocketURI returns the WebSocket URI of the browser's DevTools endpoint.
//
// Upstream equivalent: $browser->getSocketUri().
func (b *Browser) SocketURI() string {
	return b.wsURI
}

// removePage is called by Page.Close to unregister itself.
func (b *Browser) removePage(p *Page) {
	b.pagesMu.Lock()
	defer b.pagesMu.Unlock()
	for i, pg := range b.pages {
		if pg == p {
			b.pages = append(b.pages[:i], b.pages[i+1:]...)
			return
		}
	}
}
