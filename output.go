package chrome

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"

	"github.com/masfu/chrome-go/cdp"
)

// ScreenshotOptions configures a page screenshot.
type ScreenshotOptions struct {
	// Format is "png" (default) or "jpeg".
	Format string
	// Quality is JPEG quality (0-100), only used when Format is "jpeg".
	Quality int
	// Clip restricts the screenshot to a specific region.
	Clip *Clip
	// FullPage captures the full scrollable page.
	FullPage bool
}

// Clip defines a rectangular region for a screenshot.
type Clip struct {
	X      float64
	Y      float64
	Width  float64
	Height float64
	Scale  float64
}

// Screenshot holds the bytes of a captured screenshot.
//
// Upstream equivalent: HeadlessChromium\PageLoader\Action\CaptureScreenshot
type Screenshot struct {
	data []byte
}

func captureScreenshot(p *Page, opts ScreenshotOptions) (*Screenshot, error) {
	if opts.Format == "" {
		opts.Format = "png"
	}

	params := map[string]any{
		"format": opts.Format,
	}
	if opts.Format == "jpeg" && opts.Quality > 0 {
		params["quality"] = opts.Quality
	}

	if opts.FullPage {
		clip, err := p.FullPageClip()
		if err != nil {
			return nil, err
		}
		opts.Clip = &clip
	}

	if opts.Clip != nil {
		params["clip"] = map[string]any{
			"x":      opts.Clip.X,
			"y":      opts.Clip.Y,
			"width":  opts.Clip.Width,
			"height": opts.Clip.Height,
			"scale":  opts.Clip.Scale,
		}
	}

	ctx := p.browser.opts.SendSyncDefaultTimeout
	bgCtx, cancel := withDefaultTimeout(p)
	defer cancel()

	_ = ctx
	resp, err := p.session.SendMessageSync(bgCtx, cdp.Message{
		Method: "Page.captureScreenshot",
		Params: params,
	})
	if err != nil {
		return nil, fmt.Errorf("chrome-go: capture screenshot: %w", err)
	}

	var result struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("chrome-go: screenshot parse: %w", err)
	}

	data, err := base64.StdEncoding.DecodeString(result.Data)
	if err != nil {
		return nil, fmt.Errorf("chrome-go: screenshot decode: %w", err)
	}

	return &Screenshot{data: data}, nil
}

// NewScreenshotFromBytes wraps raw image bytes in a Screenshot.
// Intended for testing; production code receives Screenshot from Page.Screenshot.
func NewScreenshotFromBytes(data []byte) *Screenshot { return &Screenshot{data: data} }

// Bytes returns the raw screenshot bytes.
func (s *Screenshot) Bytes() []byte { return s.data }

// Base64 returns the screenshot encoded as a base64 string.
func (s *Screenshot) Base64() string { return base64.StdEncoding.EncodeToString(s.data) }

// SaveToFile writes the screenshot to the given file path.
func (s *Screenshot) SaveToFile(path string) error {
	return os.WriteFile(path, s.data, 0o644)
}

// PDFOptions configures PDF generation.
//
// Field names follow Go conventions; JSON tags use camelCase to match Chrome's CDP.
type PDFOptions struct {
	// Landscape renders the page in landscape orientation.
	Landscape bool `json:"landscape,omitempty"`
	// DisplayHeaderFooter prints header and footer.
	DisplayHeaderFooter bool `json:"displayHeaderFooter,omitempty"`
	// PrintBackground prints background graphics.
	PrintBackground bool `json:"printBackground,omitempty"`
	// Scale is the page scale factor (default 1).
	Scale float64 `json:"scale,omitempty"`
	// PaperWidth in inches (default 8.5).
	PaperWidth float64 `json:"paperWidth,omitempty"`
	// PaperHeight in inches (default 11).
	PaperHeight float64 `json:"paperHeight,omitempty"`
	// MarginTop in inches.
	MarginTop float64 `json:"marginTop,omitempty"`
	// MarginBottom in inches.
	MarginBottom float64 `json:"marginBottom,omitempty"`
	// MarginLeft in inches.
	MarginLeft float64 `json:"marginLeft,omitempty"`
	// MarginRight in inches.
	MarginRight float64 `json:"marginRight,omitempty"`
	// PageRanges is the range of pages to print, e.g. "1-5, 8, 11-13".
	PageRanges string `json:"pageRanges,omitempty"`
	// HeaderTemplate is the HTML template for the print header.
	HeaderTemplate string `json:"headerTemplate,omitempty"`
	// FooterTemplate is the HTML template for the print footer.
	FooterTemplate string `json:"footerTemplate,omitempty"`
	// PreferCSSPageSize uses the page size defined by CSS.
	PreferCSSPageSize bool `json:"preferCSSPageSize,omitempty"`
}

// PDF holds the bytes of a generated PDF.
//
// Upstream equivalent: HeadlessChromium\PageLoader\Action\PrintToPdf
type PDF struct {
	data []byte
}

func capturePDF(p *Page, opts PDFOptions) (*PDF, error) {
	params := map[string]any{}

	// Marshal options to map, then strip zero values.
	raw, err := json.Marshal(opts)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &params); err != nil {
		return nil, err
	}

	bgCtx, cancel := withDefaultTimeout(p)
	defer cancel()

	resp, err := p.session.SendMessageSync(bgCtx, cdp.Message{
		Method: "Page.printToPDF",
		Params: params,
	})
	if err != nil {
		return nil, fmt.Errorf("chrome-go: print to pdf: %w", err)
	}

	var result struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("chrome-go: pdf parse: %w", err)
	}

	data, err := base64.StdEncoding.DecodeString(result.Data)
	if err != nil {
		return nil, fmt.Errorf("chrome-go: pdf decode: %w", err)
	}

	return &PDF{data: data}, nil
}

// NewPDFFromBytes wraps raw PDF bytes in a PDF.
// Intended for testing; production code receives PDF from Page.PDF.
func NewPDFFromBytes(data []byte) *PDF { return &PDF{data: data} }

// Bytes returns the raw PDF bytes.
func (p *PDF) Bytes() []byte { return p.data }

// Base64 returns the PDF encoded as a base64 string.
func (p *PDF) Base64() string { return base64.StdEncoding.EncodeToString(p.data) }

// SaveToFile writes the PDF to the given file path.
func (p *PDF) SaveToFile(path string) error {
	return os.WriteFile(path, p.data, 0o644)
}

// Evaluation holds the result of a JavaScript evaluation.
//
// Upstream equivalent: HeadlessChromium\PageLoader\Action\EvaluateScript
type Evaluation struct {
	raw  json.RawMessage
	page *Page
}

// NewEvaluationFromJSON wraps a raw JSON value in an Evaluation.
// Intended for testing; production code receives Evaluation from Page.Evaluate.
func NewEvaluationFromJSON(raw json.RawMessage) *Evaluation { return &Evaluation{raw: raw} }

// ReturnValue unmarshals and returns the JavaScript return value.
func (e *Evaluation) ReturnValue() (any, error) {
	if len(e.raw) == 0 {
		return nil, nil
	}
	var v any
	if err := json.Unmarshal(e.raw, &v); err != nil {
		return nil, fmt.Errorf("chrome-go: unmarshal return value: %w", err)
	}
	return v, nil
}

// WaitForPageReload blocks until the page has reloaded after this evaluation.
func (e *Evaluation) WaitForPageReload() error {
	return e.page.WaitForReload()
}
