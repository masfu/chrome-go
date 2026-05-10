package main

// Persistent browser example: connect to an already-running Chrome instance.

import (
	"context"
	"fmt"
	"log"

	chrome "github.com/masfu/chrome-go"
)

func main() {
	ctx := context.Background()

	// Connect to a running Chrome started with --remote-debugging-port=9222.
	// You can start one with:
	//   google-chrome --headless --remote-debugging-port=9222
	browser, err := chrome.ConnectToBrowser(ctx, "ws://localhost:9222/json/version")
	if err != nil {
		log.Fatalf("connect to browser: %v", err)
	}
	defer browser.Close()

	fmt.Printf("Connected to browser at %s\n", browser.SocketURI())

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

	html, err := page.GetHTML()
	if err != nil {
		log.Fatalf("get html: %v", err)
	}
	fmt.Printf("Page HTML length: %d bytes\n", len(html))
}
