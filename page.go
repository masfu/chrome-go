package chrome

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/masfu/chrome-go/cdp"
)

// WaitOption configures waiting behaviour for Page operations.
type WaitOption func(*waitConfig)

type waitConfig struct {
	timeout time.Duration
	events  []PageEvent
}

// WithTimeout sets the maximum time to wait.
func WithTimeout(d time.Duration) WaitOption {
	return func(c *waitConfig) { c.timeout = d }
}

// WithEvents sets the page lifecycle events that signal completion.
func WithEvents(events ...PageEvent) WaitOption {
	return func(c *waitConfig) { c.events = events }
}

// ScriptTagOptions configures a script tag injected into a page.
type ScriptTagOptions struct {
	// URL is the src attribute of the script tag (remote script).
	URL string
	// Content is the inline script content.
	Content string
}

// PreScriptOption configures a pre-script added to a page.
type PreScriptOption func(*preScriptConfig)

type preScriptConfig struct {
	worldName string
}

// WithWorldName sets the isolated world name for the pre-script.
func WithWorldName(name string) PreScriptOption {
	return func(c *preScriptConfig) { c.worldName = name }
}

// ScriptTag is returned by Page.AddScriptTag.
type ScriptTag struct {
	identifier string
}

// Identifier returns the internal script tag identifier.
func (s *ScriptTag) Identifier() string { return s.identifier }

// PageOperation is returned for page operations that may need to be awaited.
type PageOperation struct {
	page *Page
	err  error
}

// Wait blocks until the page operation completes.
func (po *PageOperation) Wait() error { return po.err }

// Page represents a single browser tab.
//
// Upstream equivalent: HeadlessChromium\PageLoader\Page
type Page struct {
	session  *cdp.Session
	browser  *Browser
	targetID string
}

func newPage(session *cdp.Session, browser *Browser, targetID string) *Page {
	p := &Page{
		session:  session,
		browser:  browser,
		targetID: targetID,
	}
	// Enable necessary CDP domains.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	p.enableDomains(ctx) //nolint:errcheck
	return p
}

func (p *Page) enableDomains(ctx context.Context) error {
	for _, method := range []string{"Page.enable", "Runtime.enable", "Network.enable", "DOM.enable"} {
		if _, err := p.session.SendMessageSync(ctx, cdp.Message{Method: method}); err != nil {
			return err
		}
	}
	return nil
}

// Navigate navigates the page to the given URL and returns a Navigation that
// can be used to wait for the page to finish loading.
//
// Upstream equivalent: $page->navigate($url).
func (p *Page) Navigate(url string) (*Navigation, error) {
	ctx, cancel := context.WithTimeout(context.Background(), p.browser.opts.SendSyncDefaultTimeout)
	defer cancel()

	reader, err := p.session.SendMessage(ctx, cdp.Message{
		Method: "Page.navigate",
		Params: map[string]any{"url": url},
	})
	if err != nil {
		return nil, fmt.Errorf("chrome-go: navigate %s: %w", url, err)
	}
	return &Navigation{page: p, reader: reader}, nil
}

// WaitForReload waits for the page to be reloaded.
//
// Upstream equivalent: $page->waitForReload().
func (p *Page) WaitForReload(opts ...WaitOption) error {
	cfg := &waitConfig{timeout: p.browser.opts.SendSyncDefaultTimeout, events: []PageEvent{EventLoad}}
	for _, o := range opts {
		o(cfg)
	}
	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout)
	defer cancel()
	_, err := p.session.WaitForEvent(ctx, "Page.loadEventFired")
	return err
}

// SetHTML sets the page's HTML content.
//
// Upstream equivalent: $page->setHtml($html).
func (p *Page) SetHTML(html string, opts ...WaitOption) error {
	cfg := &waitConfig{timeout: p.browser.opts.SendSyncDefaultTimeout}
	for _, o := range opts {
		o(cfg)
	}
	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout)
	defer cancel()

	// Encode the HTML as a data URL and navigate to it.
	encoded := "data:text/html;charset=utf-8," + urlEncodeHTML(html)
	reader, err := p.session.SendMessage(ctx, cdp.Message{
		Method: "Page.navigate",
		Params: map[string]any{"url": encoded},
	})
	if err != nil {
		return err
	}
	nav := &Navigation{page: p, reader: reader}
	return nav.WaitForNavigation()
}

// urlEncodeHTML percent-encodes characters that would break a data: URL.
func urlEncodeHTML(s string) string {
	var buf []byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		// Keep safe characters as-is.
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
			c == '-' || c == '_' || c == '.' || c == '~' ||
			c == '!' || c == '*' || c == '\'' || c == '(' || c == ')' ||
			c == ';' || c == ':' || c == '@' || c == '&' || c == '=' ||
			c == '+' || c == '$' || c == ',' || c == '/' || c == '?' ||
			c == '#' || c == '[' || c == ']' || c == '<' || c == '>' {
			buf = append(buf, c)
		} else {
			buf = append(buf, fmt.Sprintf("%%%02X", c)...)
		}
	}
	return string(buf)
}

// GetHTML returns the outer HTML of the page's document element.
//
// Upstream equivalent: $page->getHtml().
func (p *Page) GetHTML() (string, error) {
	eval, err := p.Evaluate("document.documentElement.outerHTML")
	if err != nil {
		return "", err
	}
	val, err := eval.ReturnValue()
	if err != nil {
		return "", err
	}
	s, ok := val.(string)
	if !ok {
		return fmt.Sprintf("%v", val), nil
	}
	return s, nil
}

// Evaluate evaluates a JavaScript expression in the page context.
//
// Upstream equivalent: $page->evaluate($js).
func (p *Page) Evaluate(script string) (*Evaluation, error) {
	ctx, cancel := context.WithTimeout(context.Background(), p.browser.opts.SendSyncDefaultTimeout)
	defer cancel()

	resp, err := p.session.SendMessageSync(ctx, cdp.Message{
		Method: "Runtime.evaluate",
		Params: map[string]any{
			"expression":    script,
			"returnByValue": true,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("chrome-go: evaluate: %w", err)
	}

	var result struct {
		Result struct {
			Type  string          `json:"type"`
			Value json.RawMessage `json:"value"`
		} `json:"result"`
		ExceptionDetails *struct {
			Text string `json:"text"`
		} `json:"exceptionDetails"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("chrome-go: evaluate parse: %w", err)
	}
	if result.ExceptionDetails != nil {
		return nil, fmt.Errorf("chrome-go: evaluate exception: %s", result.ExceptionDetails.Text)
	}

	return &Evaluation{raw: result.Result.Value, page: p}, nil
}

// CallFunction calls a JavaScript function with the given arguments.
//
// Upstream equivalent: $page->callFunction($fn, $args).
func (p *Page) CallFunction(fn string, args ...any) (*Evaluation, error) {
	argJSON, err := json.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("chrome-go: marshal args: %w", err)
	}
	script := fmt.Sprintf("(%s).apply(null, %s)", fn, argJSON)
	return p.Evaluate(script)
}

// AddScriptTag injects a <script> tag into the page.
//
// Upstream equivalent: $page->addScriptTag($opts).
func (p *Page) AddScriptTag(opts ScriptTagOptions) (*ScriptTag, error) {
	ctx, cancel := context.WithTimeout(context.Background(), p.browser.opts.SendSyncDefaultTimeout)
	defer cancel()

	params := map[string]any{}
	if opts.URL != "" {
		params["url"] = opts.URL
	}
	if opts.Content != "" {
		params["scriptSource"] = opts.Content
	}

	resp, err := p.session.SendMessageSync(ctx, cdp.Message{
		Method: "Page.addScriptToEvaluateOnNewDocument",
		Params: params,
	})
	if err != nil {
		return nil, fmt.Errorf("chrome-go: add script tag: %w", err)
	}

	var result struct {
		Identifier string `json:"identifier"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("chrome-go: add script tag parse: %w", err)
	}
	return &ScriptTag{identifier: result.Identifier}, nil
}

// AddPreScript registers a JavaScript snippet to evaluate before any page script runs.
//
// Upstream equivalent: $page->addPreScript($script).
func (p *Page) AddPreScript(script string, opts ...PreScriptOption) error {
	cfg := &preScriptConfig{}
	for _, o := range opts {
		o(cfg)
	}

	ctx, cancel := context.WithTimeout(context.Background(), p.browser.opts.SendSyncDefaultTimeout)
	defer cancel()

	params := map[string]any{"source": script}
	if cfg.worldName != "" {
		params["worldName"] = cfg.worldName
	}

	_, err := p.session.SendMessageSync(ctx, cdp.Message{
		Method: "Page.addScriptToEvaluateOnNewDocument",
		Params: params,
	})
	return err
}

// Screenshot captures a screenshot of the page.
//
// Upstream equivalent: $page->screenshot($opts).
func (p *Page) Screenshot(opts ScreenshotOptions) (*Screenshot, error) {
	return captureScreenshot(p, opts)
}

// PDF generates a PDF of the page.
//
// Upstream equivalent: $page->pdf($opts).
func (p *Page) PDF(opts PDFOptions) (*PDF, error) {
	return capturePDF(p, opts)
}

// FullPageClip returns the clip region for the full page.
//
// Upstream equivalent: $page->getFullPageClip().
func (p *Page) FullPageClip() (Clip, error) {
	eval, err := p.Evaluate(`({x:0,y:0,width:document.body.scrollWidth,height:document.body.scrollHeight,scale:1})`)
	if err != nil {
		return Clip{}, err
	}
	val, err := eval.ReturnValue()
	if err != nil {
		return Clip{}, err
	}
	m, ok := val.(map[string]any)
	if !ok {
		return Clip{}, fmt.Errorf("chrome-go: unexpected full page clip type")
	}
	toFloat := func(key string) float64 {
		if v, ok := m[key]; ok {
			switch n := v.(type) {
			case float64:
				return n
			case json.Number:
				f, _ := n.Float64()
				return f
			}
		}
		return 0
	}
	return Clip{
		X:      toFloat("x"),
		Y:      toFloat("y"),
		Width:  toFloat("width"),
		Height: toFloat("height"),
		Scale:  toFloat("scale"),
	}, nil
}

// SetViewport sets the page viewport dimensions.
//
// Upstream equivalent: $page->setViewport($width, $height).
func (p *Page) SetViewport(width, height int) *PageOperation {
	ctx, cancel := context.WithTimeout(context.Background(), p.browser.opts.SendSyncDefaultTimeout)
	defer cancel()

	_, err := p.session.SendMessageSync(ctx, cdp.Message{
		Method: "Emulation.setDeviceMetricsOverride",
		Params: map[string]any{
			"width":             width,
			"height":            height,
			"deviceScaleFactor": 1,
			"mobile":            false,
		},
	})
	return &PageOperation{page: p, err: err}
}

// Mouse returns the Mouse controller for this page.
//
// Upstream equivalent: $page->getMouse().
func (p *Page) Mouse() *Mouse {
	return &Mouse{page: p}
}

// Keyboard returns the Keyboard controller for this page.
//
// Upstream equivalent: $page->getKeyboard().
func (p *Page) Keyboard() *Keyboard {
	return &Keyboard{page: p}
}

// SetCookies sets cookies on the page.
//
// Upstream equivalent: $page->setCookies($cookies).
func (p *Page) SetCookies(cookies []Cookie) *PageOperation {
	ctx, cancel := context.WithTimeout(context.Background(), p.browser.opts.SendSyncDefaultTimeout)
	defer cancel()

	params := make([]map[string]any, len(cookies))
	for i, c := range cookies {
		params[i] = cookieToParams(c)
	}
	_, err := p.session.SendMessageSync(ctx, cdp.Message{
		Method: "Network.setCookies",
		Params: map[string]any{"cookies": params},
	})
	return &PageOperation{page: p, err: err}
}

// Cookies returns all cookies visible to the current page URL.
//
// Upstream equivalent: $page->getCookies().
func (p *Page) Cookies() (CookieList, error) {
	return p.fetchCookies("Network.getCookies")
}

// AllCookies returns all cookies in the browser.
//
// Upstream equivalent: $page->getAllCookies().
func (p *Page) AllCookies() (CookieList, error) {
	return p.fetchCookies("Network.getAllCookies")
}

func (p *Page) fetchCookies(method string) (CookieList, error) {
	ctx, cancel := context.WithTimeout(context.Background(), p.browser.opts.SendSyncDefaultTimeout)
	defer cancel()

	resp, err := p.session.SendMessageSync(ctx, cdp.Message{Method: method})
	if err != nil {
		return nil, err
	}

	var result struct {
		Cookies []cookieRaw `json:"cookies"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, err
	}

	list := make(CookieList, len(result.Cookies))
	for i, raw := range result.Cookies {
		list[i] = rawToCookie(raw)
	}
	return list, nil
}

// DOM returns the DOM controller for this page.
//
// Upstream equivalent: $page->getDom().
func (p *Page) DOM() *DOM {
	return &DOM{page: p}
}

// WaitUntilContainsElement blocks until the given selector is present in the DOM.
//
// Upstream equivalent: $page->waitUntilContainsElement($selector).
func (p *Page) WaitUntilContainsElement(selector Selector, opts ...WaitOption) (*Element, error) {
	cfg := &waitConfig{timeout: p.browser.opts.SendSyncDefaultTimeout}
	for _, o := range opts {
		o(cfg)
	}

	deadline := time.Now().Add(cfg.timeout)
	for time.Now().Before(deadline) {
		el, err := p.DOM().QuerySelector(selector.selectorString())
		if err == nil && el != nil {
			return el, nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return nil, fmt.Errorf("%w: %s", ErrElementNotFound, selector.selectorString())
}

// SetUserAgent overrides the user-agent for this page.
//
// Upstream equivalent: $page->setUserAgent($ua).
func (p *Page) SetUserAgent(ua string) error {
	ctx, cancel := context.WithTimeout(context.Background(), p.browser.opts.SendSyncDefaultTimeout)
	defer cancel()

	_, err := p.session.SendMessageSync(ctx, cdp.Message{
		Method: "Network.setUserAgentOverride",
		Params: map[string]any{"userAgent": ua},
	})
	return err
}

// SetDownloadPath sets the directory where downloads are saved.
//
// Upstream equivalent: $page->setDownloadPath($path).
func (p *Page) SetDownloadPath(path string) error {
	ctx, cancel := context.WithTimeout(context.Background(), p.browser.opts.SendSyncDefaultTimeout)
	defer cancel()

	_, err := p.session.SendMessageSync(ctx, cdp.Message{
		Method: "Browser.setDownloadBehavior",
		Params: map[string]any{
			"behavior":     "allow",
			"downloadPath": path,
		},
	})
	return err
}

// Close closes this page/tab.
func (p *Page) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := p.session.SendMessageSync(ctx, cdp.Message{
		Method: "Target.closeTarget",
		Params: map[string]any{"targetId": p.targetID},
	})
	p.browser.removePage(p)
	return err
}
