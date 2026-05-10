package main

// Evaluate example: run JavaScript in a page and retrieve the result.

import (
	"context"
	"fmt"
	"log"

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

	eval, err := page.Evaluate("document.title")
	if err != nil {
		log.Fatalf("evaluate: %v", err)
	}

	title, err := eval.ReturnValue()
	if err != nil {
		log.Fatalf("return value: %v", err)
	}
	fmt.Printf("Page title: %v\n", title)

	// Call a function with arguments.
	eval2, err := page.CallFunction("(a, b) => a + b", 3, 4)
	if err != nil {
		log.Fatalf("call function: %v", err)
	}
	sum, _ := eval2.ReturnValue()
	fmt.Printf("3 + 4 = %v\n", sum)
}
