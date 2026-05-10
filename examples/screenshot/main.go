package main

// Screenshot example: capture a screenshot of a web page and save it to disk.

import (
	"context"
	"fmt"
	"log"

	chrome "github.com/masfu/chrome-go"
)

func main() {
	ctx := context.Background()

	factory := chrome.NewBrowserFactory()
	factory.SetOptions(chrome.Options{
		NoSandbox: true, // required in many CI/container environments
	})

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

	screenshot, err := page.Screenshot(chrome.ScreenshotOptions{
		FullPage: true,
	})
	if err != nil {
		log.Fatalf("screenshot: %v", err)
	}

	if err := screenshot.SaveToFile("example.png"); err != nil {
		log.Fatalf("save screenshot: %v", err)
	}

	fmt.Println("Screenshot saved to example.png")
}
