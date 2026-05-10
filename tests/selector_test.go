package chrome_test

import (
	"testing"

	chrome "github.com/masfu/chrome-go"
)

func TestCSSSelector_SelectorString(t *testing.T) {
	s := chrome.CSSSelector{Query: "#main .btn"}
	if s.Query != "#main .btn" {
		t.Errorf("CSSSelector.Query: got %q", s.Query)
	}
}

func TestXPathSelector_SelectorString(t *testing.T) {
	s := chrome.XPathSelector{Query: "//div[@id='main']"}
	if s.Query != "//div[@id='main']" {
		t.Errorf("XPathSelector.Query: got %q", s.Query)
	}
}

func TestCSSSelector_ImplementsSelector(t *testing.T) {
	var _ chrome.Selector = chrome.CSSSelector{Query: "div"}
}

func TestXPathSelector_ImplementsSelector(t *testing.T) {
	var _ chrome.Selector = chrome.XPathSelector{Query: "//div"}
}
