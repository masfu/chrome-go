# chrome-go

[![Go Reference](https://pkg.go.dev/badge/github.com/masfu/chrome-go.svg)](https://pkg.go.dev/github.com/masfu/chrome-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/masfu/chrome-go)](https://goreportcard.com/report/github.com/masfu/chrome-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Instrument headless Chrome/Chromium instances from Go.

`chrome-go` is a Go port of the popular [chrome-php/chrome](https://github.com/chrome-php/chrome) library. The public API mirrors the PHP one closely so that teams porting code between the two languages can do so with minimal friction. If you want a pipeline/action-style API instead, take a look at [`chromedp`](https://github.com/chromedp/chromedp); both libraries are good, they just have different shapes.

## Features

- Launch Chrome or Chromium from Go (headless or headful)
- Create pages and navigate to URLs
- Take screenshots (PNG, JPEG, WebP — full-page or clipped)
- Generate PDFs with full Chrome printing options
- Evaluate JavaScript and call functions in the page context
- Mouse and keyboard emulation
- Cookie management
- DOM querying via CSS or XPath selectors
- File uploads and download path configuration
- Connect to and reuse a long-running Chrome instance
- Direct access to the Chrome DevTools Protocol when you need to go lower-level

## Requirements

- Go 1.22 or later
- A Chrome or Chromium 65+ binary on the host (or use the Docker image below)

## Installation

```bash
go get github.com/masfu/chrome-go
```

## Quick start

```go
package main

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
        log.Fatal(err)
    }
    defer browser.Close()

    page, err := browser.CreatePage(ctx)
    if err != nil {
        log.Fatal(err)
    }

    nav, err := page.Navigate("https://example.com")
    if err != nil {
        log.Fatal(err)
    }
    if err := nav.WaitForNavigation(); err != nil {
        log.Fatal(err)
    }

    // Read the page title
    eval, _ := page.Evaluate("document.title")
    title, _ := eval.ReturnValue()
    fmt.Println("title:", title)

    // Save a screenshot
    shot, _ := page.Screenshot(chrome.ScreenshotOptions{Format: "png"})
    _ = shot.SaveToFile("example.png")

    // Save a PDF
    pdf, _ := page.PDF(chrome.PDFOptions{PrintBackground: true})
    _ = pdf.SaveToFile("example.pdf")
}
```

## Common patterns

### Use a specific Chrome binary

The factory honors the `CHROME_PATH` environment variable. You can also pass the binary name or path explicitly:

```go
factory := chrome.NewBrowserFactory("chromium-browser")
```

### Disable headless mode for debugging

```go
headless := false
browser, err := factory.CreateBrowser(ctx, chrome.Options{
    Headless: &headless,
})
```

### Run inside a container

When running in Docker, you almost always need `--no-sandbox`:

```go
browser, err := factory.CreateBrowser(ctx, chrome.Options{
    NoSandbox: true,
})
```

### Full-page screenshot

```go
clip, _ := page.FullPageClip()
shot, _ := page.Screenshot(chrome.ScreenshotOptions{
    CaptureBeyondViewport: true,
    Clip:                  &clip,
    Format:                "jpeg",
})
shot.SaveToFile("full.jpg")
```

### Custom PDF

```go
pdf, _ := page.PDF(chrome.PDFOptions{
    Landscape:           true,
    PrintBackground:     true,
    DisplayHeaderFooter: true,
    MarginTop:           0.4,
    MarginBottom:        0.4,
    PaperWidth:          8.5,
    PaperHeight:         11.0,
    HeaderTemplate:      `<div style="font-size:8px"><span class="title"></span></div>`,
    FooterTemplate:      `<div style="font-size:8px"><span class="pageNumber"></span> / <span class="totalPages"></span></div>`,
})
pdf.SaveToFile("report.pdf")
```

### Reuse a browser across runs

```go
const socketFile = "/tmp/chrome-go-socket"

uri, err := os.ReadFile(socketFile)
var browser *chrome.Browser
if err == nil {
    browser, err = chrome.ConnectToBrowser(ctx, string(uri))
}
if err != nil {
    factory := chrome.NewBrowserFactory()
    browser, err = factory.CreateBrowser(ctx, chrome.Options{KeepAlive: true})
    if err != nil {
        log.Fatal(err)
    }
    _ = os.WriteFile(socketFile, []byte(browser.SocketURI()), 0o600)
}
```

For the full API, see the [PRD](./PRD.md) and the godoc on [pkg.go.dev](https://pkg.go.dev/github.com/masfu/chrome-go).

---

## Docker

Running `chrome-go` in Docker is the most common deployment shape. The tricky part is getting Chromium and its runtime dependencies installed correctly — the example below shows a multi-stage build that produces a small final image with everything Chromium needs to actually start.

### Dockerfile

```dockerfile
# syntax=docker/dockerfile:1.6

# ---- build stage ----
FROM golang:1.22-bookworm AS build

WORKDIR /src

# Cache modules first
COPY go.mod go.sum ./
RUN go mod download

# Build the binary
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/app ./cmd/app

# ---- runtime stage ----
FROM debian:bookworm-slim

# Install Chromium and the libraries it needs at runtime.
# fonts-liberation + fonts-noto-color-emoji give you sensible defaults so
# screenshots and PDFs don't render with tofu boxes.
# tini is the init so zombie Chromium processes get reaped properly.
RUN apt-get update && apt-get install -y --no-install-recommends \
        chromium \
        ca-certificates \
        fonts-liberation \
        fonts-noto-color-emoji \
        fonts-noto-cjk \
        libnss3 \
        libatk1.0-0 \
        libatk-bridge2.0-0 \
        libcups2 \
        libdrm2 \
        libxkbcommon0 \
        libxcomposite1 \
        libxdamage1 \
        libxfixes3 \
        libxrandr2 \
        libgbm1 \
        libpango-1.0-0 \
        libcairo2 \
        libasound2 \
        tini \
    && rm -rf /var/lib/apt/lists/*

# Run as a non-root user. Chromium's sandbox cannot be used from PID 1 in many
# container runtimes, so we combine non-root + --no-sandbox in code.
RUN groupadd --system --gid 1000 app \
    && useradd --system --uid 1000 --gid app --create-home app

# Tell chrome-go where to find the binary
ENV CHROME_PATH=/usr/bin/chromium

WORKDIR /home/app
COPY --from=build /out/app /usr/local/bin/app
USER app

ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/app"]
```

### Build and run

```bash
docker build -t chrome-go-app .
docker run --rm chrome-go-app
```

If you see Chromium crash immediately on startup, you have two quick fixes:

```bash
# Option 1: bigger /dev/shm — Chromium uses shared memory heavily
docker run --rm --shm-size=1g chrome-go-app

# Option 2: more permissive seccomp profile (only if --shm-size isn't enough)
docker run --rm --security-opt seccomp=unconfined chrome-go-app
```

In code, when running in any container, set `NoSandbox: true`:

```go
browser, err := factory.CreateBrowser(ctx, chrome.Options{
    NoSandbox: true,
})
```

### docker-compose example

```yaml
services:
  app:
    build: .
    image: chrome-go-app
    shm_size: "1gb"           # avoids the most common Chromium crash in containers
    environment:
      CHROME_PATH: /usr/bin/chromium
    volumes:
      - ./output:/home/app/output
```

### Alpine variant (smaller image, more caveats)

If image size matters more than ease of setup, you can use Alpine. Note that Alpine ships `chromium` from the community repository and you must install the matching font packages, otherwise rendered text will be invisible:

```dockerfile
FROM golang:1.22-alpine AS build
WORKDIR /src
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/app ./cmd/app

FROM alpine:3.19
RUN apk add --no-cache \
        chromium \
        nss \
        freetype \
        harfbuzz \
        ca-certificates \
        ttf-freefont \
        font-noto-emoji \
        tini
ENV CHROME_PATH=/usr/bin/chromium
RUN addgroup -S app && adduser -S app -G app
COPY --from=build /out/app /usr/local/bin/app
USER app
ENTRYPOINT ["/sbin/tini", "--", "/usr/local/bin/app"]
```

The Debian-based image is recommended unless you have a specific reason to prefer Alpine — Chromium has historically been less buggy on glibc than musl.

### Image size reference

Approximate sizes for the same trivial program:

| Base | Final image size |
|---|---|
| `debian:bookworm-slim` + chromium | ~450 MB |
| `alpine:3.19` + chromium | ~280 MB |

If you need it smaller, consider using `chromium-headless-shell` (no UI), which trims another ~100 MB but loses headful debugging.

## Migrating from chrome-php/chrome

Most call sites translate almost line-for-line. The two notable shifts are:

1. **`context.Context` is required on entry points.** Pass `context.Background()` if you don't need cancellation, or a derived context with a deadline if you do.
2. **Errors are returned, not thrown.** Replace upstream's `try/catch (OperationTimedOut $e)` with `if errors.Is(err, chrome.ErrOperationTimedOut)`.

| chrome-php | chrome-go |
|---|---|
| `new BrowserFactory()` | `chrome.NewBrowserFactory()` |
| `$factory->createBrowser([...])` | `factory.CreateBrowser(ctx, chrome.Options{...})` |
| `$browser->createPage()` | `browser.CreatePage(ctx)` |
| `$page->navigate($url)->waitForNavigation()` | `nav, _ := page.Navigate(url); nav.WaitForNavigation()` |
| `$page->evaluate($js)->getReturnValue()` | `eval, _ := page.Evaluate(js); v, _ := eval.ReturnValue()` |
| `$page->screenshot($opts)->saveToFile($p)` | `s, _ := page.Screenshot(opts); s.SaveToFile(p)` |
| `$page->pdf($opts)->saveToFile($p)` | `p, _ := page.PDF(opts); p.SaveToFile(p)` |
| `$browser->close()` | `browser.Close()` (or `defer browser.Close()`) |

## Contributing

PRs welcome. See [CONTRIBUTING.md](./CONTRIBUTING.md) for the development setup, including how to run the integration tests against the Docker image above.

## License

MIT. See [LICENSE](./LICENSE).

`chrome-go` is a port of [chrome-php/chrome](https://github.com/chrome-php/chrome), which is also MIT-licensed. Thanks to the chrome-php maintainers and contributors for shaping the API this library follows.