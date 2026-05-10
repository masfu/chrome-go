package main

// Cookies example: set and retrieve cookies on a page.

import (
	"context"
	"fmt"
	"log"
	"time"

	chrome "github.com/masfu/chrome-go"
)

func main() {
	ctx := context.Background()

	factory := chrome.NewBrowserFactory()
	browser, err := factory.CreateBrowser(ctx)
	if err != nil {
		log.Fatalf("create browser: %v", err)
	}
	defer browser.Close()

	page, err := browser.CreatePage(ctx)
	if err != nil {
		log.Fatalf("create page: %v", err)
	}
	defer page.Close()

	nav, err := page.Navigate("https://example.com")
	if err != nil {
		log.Fatalf("navigate: %v", err)
	}
	if err := nav.WaitForNavigation(chrome.EventLoad); err != nil {
		log.Fatalf("wait for navigation: %v", err)
	}

	// Set a cookie.
	cookie := chrome.NewCookie("session", "abc123", chrome.CookieOptions{
		Domain:   "example.com",
		Path:     "/",
		Expires:  time.Now().Add(24 * time.Hour),
		HTTPOnly: true,
		Secure:   true,
	})
	if err := page.SetCookies([]chrome.Cookie{cookie}).Wait(); err != nil {
		log.Fatalf("set cookies: %v", err)
	}

	// Read cookies back.
	cookies, err := page.Cookies()
	if err != nil {
		log.Fatalf("get cookies: %v", err)
	}

	for _, c := range cookies {
		fmt.Printf("Cookie: %s=%s (domain=%s)\n", c.Name, c.Value, c.Domain)
	}

	// Filter by name.
	if c, ok := cookies.FindOneBy("name", "session"); ok {
		fmt.Printf("Found session cookie: %s\n", c.Value)
	}
}
