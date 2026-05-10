package chrome

import (
	"context"
	"time"
)

// withDefaultTimeout returns a context with the browser's default send timeout.
func withDefaultTimeout(p *Page) (context.Context, context.CancelFunc) {
	d := p.browser.opts.SendSyncDefaultTimeout
	if d == 0 {
		d = 30 * time.Second
	}
	return context.WithTimeout(context.Background(), d)
}
