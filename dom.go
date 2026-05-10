package chrome

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/masfu/chrome-go/cdp"
)

// DOM provides access to the page's Document Object Model.
//
// Upstream equivalent: HeadlessChromium\Dom\Dom
type DOM struct {
	page *Page
}

// QuerySelector returns the first element matching the CSS selector.
//
// Upstream equivalent: $dom->querySelector($css).
func (d *DOM) QuerySelector(css string) (*Element, error) {
	eval, err := d.page.Evaluate(fmt.Sprintf(
		`document.querySelector(%s)`, jsonString(css),
	))
	if err != nil {
		return nil, err
	}
	return d.evalToElement(eval, css)
}

// QuerySelectorAll returns all elements matching the CSS selector.
//
// Upstream equivalent: $dom->querySelectorAll($css).
func (d *DOM) QuerySelectorAll(css string) ([]*Element, error) {
	nodeIDs, err := d.resolveAll(css)
	if err != nil {
		return nil, err
	}
	elements := make([]*Element, len(nodeIDs))
	for i, id := range nodeIDs {
		elements[i] = &Element{page: d.page, nodeID: id}
	}
	return elements, nil
}

// Search returns all elements matching the XPath expression.
//
// Upstream equivalent: $dom->search($xpath).
func (d *DOM) Search(xpath string) ([]*Element, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d.page.browser.opts.SendSyncDefaultTimeout)
	defer cancel()

	// Get the document root.
	docResp, err := d.page.session.SendMessageSync(ctx, cdp.Message{Method: "DOM.getDocument"})
	if err != nil {
		return nil, fmt.Errorf("chrome-go: get document: %w", err)
	}
	var doc struct {
		Root struct {
			NodeID int `json:"nodeId"`
		} `json:"root"`
	}
	if err := json.Unmarshal(docResp.Result, &doc); err != nil {
		return nil, err
	}

	resp, err := d.page.session.SendMessageSync(ctx, cdp.Message{
		Method: "DOM.performSearch",
		Params: map[string]any{"query": xpath},
	})
	if err != nil {
		return nil, fmt.Errorf("chrome-go: xpath search: %w", err)
	}

	var searchResult struct {
		SearchID    string `json:"searchId"`
		ResultCount int    `json:"resultCount"`
	}
	if err := json.Unmarshal(resp.Result, &searchResult); err != nil {
		return nil, err
	}
	if searchResult.ResultCount == 0 {
		return nil, nil
	}

	resp2, err := d.page.session.SendMessageSync(ctx, cdp.Message{
		Method: "DOM.getSearchResults",
		Params: map[string]any{
			"searchId":  searchResult.SearchID,
			"fromIndex": 0,
			"toIndex":   searchResult.ResultCount,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("chrome-go: get search results: %w", err)
	}

	var ids struct {
		NodeIDs []int `json:"nodeIds"`
	}
	if err := json.Unmarshal(resp2.Result, &ids); err != nil {
		return nil, err
	}

	elements := make([]*Element, len(ids.NodeIDs))
	for i, id := range ids.NodeIDs {
		elements[i] = &Element{page: d.page, nodeID: id}
	}
	return elements, nil
}

// evalToElement converts a JS object reference to an Element.
func (d *DOM) evalToElement(eval *Evaluation, selector string) (*Element, error) {
	val, err := eval.ReturnValue()
	if err != nil {
		return nil, err
	}
	if val == nil {
		return nil, fmt.Errorf("%w: %s", ErrElementNotFound, selector)
	}
	// We need to resolve the JS object to a DOM node id via Runtime.
	nodeIDs, err := d.resolveAll(selector)
	if err != nil {
		return nil, err
	}
	if len(nodeIDs) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrElementNotFound, selector)
	}
	return &Element{page: d.page, nodeID: nodeIDs[0]}, nil
}

// resolveAll uses DOM.querySelectorAll to resolve CSS selectors to node IDs.
func (d *DOM) resolveAll(css string) ([]int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d.page.browser.opts.SendSyncDefaultTimeout)
	defer cancel()

	docResp, err := d.page.session.SendMessageSync(ctx, cdp.Message{Method: "DOM.getDocument"})
	if err != nil {
		return nil, fmt.Errorf("chrome-go: get document: %w", err)
	}
	var doc struct {
		Root struct {
			NodeID int `json:"nodeId"`
		} `json:"root"`
	}
	if err := json.Unmarshal(docResp.Result, &doc); err != nil {
		return nil, err
	}

	resp, err := d.page.session.SendMessageSync(ctx, cdp.Message{
		Method: "DOM.querySelectorAll",
		Params: map[string]any{
			"nodeId":   doc.Root.NodeID,
			"selector": css,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("chrome-go: querySelectorAll: %w", err)
	}

	var result struct {
		NodeIDs []int `json:"nodeIds"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, err
	}
	return result.NodeIDs, nil
}

// jsonString returns a JSON-encoded string literal.
func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// boundingBox is used by Mouse.Find.
type boundingBox struct {
	x, y, width, height float64
}

// Element represents a single DOM element.
//
// Upstream equivalent: HeadlessChromium\Dom\Element
type Element struct {
	page   *Page
	nodeID int
}

// Click clicks the element.
//
// Upstream equivalent: $element->click().
func (e *Element) Click() error {
	box, err := e.boundingBox()
	if err != nil {
		return err
	}
	m := e.page.Mouse()
	m.Move(int(box.x+box.width/2), int(box.y+box.height/2))
	m.Click()
	return nil
}

// SendKeys types text into the element (focuses it first).
//
// Upstream equivalent: $element->sendKeys($text).
func (e *Element) SendKeys(text string) error {
	if err := e.focus(); err != nil {
		return err
	}
	e.page.Keyboard().TypeText(text)
	return nil
}

// SendFile sets the value of a file input element.
//
// Upstream equivalent: $element->sendFile($path).
func (e *Element) SendFile(path string) error {
	ctx, cancel := context.WithTimeout(context.Background(), e.page.browser.opts.SendSyncDefaultTimeout)
	defer cancel()

	_, err := e.page.session.SendMessageSync(ctx, cdp.Message{
		Method: "DOM.setFileInputFiles",
		Params: map[string]any{
			"files":  []string{path},
			"nodeId": e.nodeID,
		},
	})
	return err
}

// Text returns the element's innerText.
//
// Upstream equivalent: $element->getText().
func (e *Element) Text() (string, error) {
	nodeID, err := e.resolveObjectID()
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), e.page.browser.opts.SendSyncDefaultTimeout)
	defer cancel()

	resp, err := e.page.session.SendMessageSync(ctx, cdp.Message{
		Method: "Runtime.callFunctionOn",
		Params: map[string]any{
			"objectId":            nodeID,
			"functionDeclaration": "function() { return this.innerText; }",
			"returnByValue":       true,
		},
	})
	if err != nil {
		return "", err
	}

	var result struct {
		Result struct {
			Value string `json:"value"`
		} `json:"result"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return "", err
	}
	return result.Result.Value, nil
}

// Attribute returns the value of the named attribute.
//
// Upstream equivalent: $element->getAttribute($name).
func (e *Element) Attribute(name string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), e.page.browser.opts.SendSyncDefaultTimeout)
	defer cancel()

	resp, err := e.page.session.SendMessageSync(ctx, cdp.Message{
		Method: "DOM.getAttributes",
		Params: map[string]any{"nodeId": e.nodeID},
	})
	if err != nil {
		return "", err
	}

	var result struct {
		Attributes []string `json:"attributes"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return "", err
	}

	// Attributes are returned as [name, value, name, value, ...] pairs.
	for i := 0; i+1 < len(result.Attributes); i += 2 {
		if result.Attributes[i] == name {
			return result.Attributes[i+1], nil
		}
	}
	return "", fmt.Errorf("%w: attribute %q not found", ErrElementNotFound, name)
}

// QuerySelector finds a child element matching the CSS selector.
//
// Upstream equivalent: $element->querySelector($css).
func (e *Element) QuerySelector(css string) (*Element, error) {
	ctx, cancel := context.WithTimeout(context.Background(), e.page.browser.opts.SendSyncDefaultTimeout)
	defer cancel()

	resp, err := e.page.session.SendMessageSync(ctx, cdp.Message{
		Method: "DOM.querySelector",
		Params: map[string]any{
			"nodeId":   e.nodeID,
			"selector": css,
		},
	})
	if err != nil {
		return nil, err
	}

	var result struct {
		NodeID int `json:"nodeId"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, err
	}
	if result.NodeID == 0 {
		return nil, fmt.Errorf("%w: %s", ErrElementNotFound, css)
	}
	return &Element{page: e.page, nodeID: result.NodeID}, nil
}

// QuerySelectorAll finds all child elements matching the CSS selector.
//
// Upstream equivalent: $element->querySelectorAll($css).
func (e *Element) QuerySelectorAll(css string) ([]*Element, error) {
	ctx, cancel := context.WithTimeout(context.Background(), e.page.browser.opts.SendSyncDefaultTimeout)
	defer cancel()

	resp, err := e.page.session.SendMessageSync(ctx, cdp.Message{
		Method: "DOM.querySelectorAll",
		Params: map[string]any{
			"nodeId":   e.nodeID,
			"selector": css,
		},
	})
	if err != nil {
		return nil, err
	}

	var result struct {
		NodeIDs []int `json:"nodeIds"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, err
	}

	elements := make([]*Element, len(result.NodeIDs))
	for i, id := range result.NodeIDs {
		elements[i] = &Element{page: e.page, nodeID: id}
	}
	return elements, nil
}

// focus focuses the element via JavaScript.
func (e *Element) focus() error {
	ctx, cancel := context.WithTimeout(context.Background(), e.page.browser.opts.SendSyncDefaultTimeout)
	defer cancel()
	_, err := e.page.session.SendMessageSync(ctx, cdp.Message{
		Method: "DOM.focus",
		Params: map[string]any{"nodeId": e.nodeID},
	})
	return err
}

// boundingBox returns the element's position and dimensions.
func (e *Element) boundingBox() (*boundingBox, error) {
	ctx, cancel := context.WithTimeout(context.Background(), e.page.browser.opts.SendSyncDefaultTimeout)
	defer cancel()

	resp, err := e.page.session.SendMessageSync(ctx, cdp.Message{
		Method: "DOM.getBoxModel",
		Params: map[string]any{"nodeId": e.nodeID},
	})
	if err != nil {
		return nil, err
	}

	var result struct {
		Model struct {
			Content []float64 `json:"content"`
			Width   float64   `json:"width"`
			Height  float64   `json:"height"`
		} `json:"model"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, err
	}

	if len(result.Model.Content) < 2 {
		return nil, fmt.Errorf("chrome-go: empty box model for node %d", e.nodeID)
	}

	return &boundingBox{
		x:      result.Model.Content[0],
		y:      result.Model.Content[1],
		width:  result.Model.Width,
		height: result.Model.Height,
	}, nil
}

// resolveObjectID resolves the element to a Runtime object ID for function calls.
func (e *Element) resolveObjectID() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), e.page.browser.opts.SendSyncDefaultTimeout)
	defer cancel()

	resp, err := e.page.session.SendMessageSync(ctx, cdp.Message{
		Method: "DOM.resolveNode",
		Params: map[string]any{"nodeId": e.nodeID},
	})
	if err != nil {
		return "", err
	}

	var result struct {
		Object struct {
			ObjectID string `json:"objectId"`
		} `json:"object"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return "", err
	}
	return result.Object.ObjectID, nil
}

// waitForElement is an unexported helper for WaitUntilContainsElement.
func waitForElement(p *Page, selector Selector, timeout time.Duration) (*Element, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		el, err := p.DOM().QuerySelector(selector.selectorString())
		if err == nil && el != nil {
			return el, nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return nil, fmt.Errorf("%w: %s", ErrElementNotFound, selector.selectorString())
}
