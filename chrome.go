// Package chrome provides a Go client for controlling headless Chrome/Chromium
// via the Chrome DevTools Protocol (CDP).
//
// It mirrors the public API surface of chrome-php/chrome so that teams porting
// PHP services to Go can move with minimal cognitive overhead.
//
// Basic usage:
//
//	factory := chrome.NewBrowserFactory()
//	browser, err := factory.CreateBrowser(ctx)
//	if err != nil { /* handle */ }
//	defer browser.Close()
//
//	page, err := browser.CreatePage(ctx)
//	if err != nil { /* handle */ }
//
//	nav, err := page.Navigate("https://example.com")
//	if err != nil { /* handle */ }
//	if err := nav.WaitForNavigation(); err != nil { /* handle */ }
//
//	screenshot, err := page.Screenshot(chrome.ScreenshotOptions{})
//	if err != nil { /* handle */ }
//	screenshot.SaveToFile("example.png")
package chrome

import (
	"errors"
)

// Sentinel errors returned by chrome-go. Check with errors.Is.
var (
	// ErrOperationTimedOut is returned when an operation exceeds its timeout.
	ErrOperationTimedOut = errors.New("chrome-go: operation timed out")
	// ErrNavigationExpired is returned when a navigation response is no longer valid.
	ErrNavigationExpired = errors.New("chrome-go: navigation expired")
	// ErrBrowserConnection is returned when the browser connection cannot be established.
	ErrBrowserConnection = errors.New("chrome-go: browser connection failed")
	// ErrElementNotFound is returned when a DOM element cannot be found.
	ErrElementNotFound = errors.New("chrome-go: element not found")
	// ErrBrowserClosed is returned when operating on a closed browser.
	ErrBrowserClosed = errors.New("chrome-go: browser is closed")
)
