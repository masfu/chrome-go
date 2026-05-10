package main

// PDF example: generate a PDF of a web page and save it to disk.

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
		NoSandbox: true,
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

	nav, err := page.Navigate("https://google.com")
	if err != nil {
		log.Fatalf("navigate: %v", err)
	}
	if err := nav.WaitForNavigation(chrome.EventLoad); err != nil {
		log.Fatalf("wait for navigation: %v", err)
	}

	pdf, err := page.PDF(chrome.PDFOptions{
		PrintBackground: true,
		Landscape:       false,
		PaperWidth:      8.5,
		PaperHeight:     11,
	})
	if err != nil {
		log.Fatalf("pdf: %v", err)
	}

	if err := pdf.SaveToFile("example.pdf"); err != nil {
		log.Fatalf("save pdf: %v", err)
	}

	fmt.Println("PDF saved to example.pdf")
}
