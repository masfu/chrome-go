package chrome

// Selector is a type that can be used to locate DOM elements.
//
// Upstream equivalent: HeadlessChromium\Dom\Selector
type Selector interface {
	selectorString() string
}

// CSSSelector selects elements using a CSS selector expression.
//
// Upstream equivalent: HeadlessChromium\Dom\Selector\CssSelector
type CSSSelector struct {
	// Query is the CSS selector expression, e.g. "#main .button".
	Query string
}

func (s CSSSelector) selectorString() string { return s.Query }

// XPathSelector selects elements using an XPath expression.
//
// Upstream equivalent: HeadlessChromium\Dom\Selector\XPathSelector
type XPathSelector struct {
	// Query is the XPath expression, e.g. "//div[@class='button']".
	Query string
}

func (s XPathSelector) selectorString() string { return s.Query }
