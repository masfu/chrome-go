package chrome

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/masfu/chrome-go/cdp"
)

// BrowserFactory creates and configures Browser instances.
//
// Upstream equivalent: HeadlessChromium\Browser\BrowserFactory
type BrowserFactory struct {
	executable string
	opts       Options
}

// NewBrowserFactory returns a BrowserFactory. An optional executable path can
// be provided; otherwise CHROME_PATH is consulted, then common OS locations.
func NewBrowserFactory(executable ...string) *BrowserFactory {
	f := &BrowserFactory{}
	if len(executable) > 0 && executable[0] != "" {
		f.executable = executable[0]
	}
	return f
}

// SetOptions replaces the factory's options entirely.
func (f *BrowserFactory) SetOptions(opts Options) {
	f.opts = opts
}

// AddOptions merges opts on top of the factory's existing options.
func (f *BrowserFactory) AddOptions(opts Options) {
	f.opts = mergeOptions(f.opts, opts)
}

// CreateBrowser launches a new Chrome/Chromium process and returns a connected Browser.
// Per-call opts, if provided, are merged on top of the factory-level options.
func (f *BrowserFactory) CreateBrowser(ctx context.Context, opts ...Options) (*Browser, error) {
	o := f.opts
	for _, extra := range opts {
		o = mergeOptions(o, extra)
	}
	applyDefaults(&o)

	exe, err := resolveExecutable(f.executable)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrBrowserConnection, err.Error())
	}

	args := buildArgs(o)
	// Ask Chrome to pick a random debugging port and tell us which one it chose.
	args = append(args, "--remote-debugging-port=0")

	cmd := exec.CommandContext(ctx, exe, args...)

	// Pass env variables.
	cmd.Env = os.Environ()
	for k, v := range o.EnvVariables {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	// We read stderr to find the DevTools URL.
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("chrome-go: pipe stderr: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrBrowserConnection, err.Error())
	}

	// Read lines from stderr to find "DevTools listening on ws://..."
	wsURI, err := readDevToolsURI(stderr, o.StartupTimeout)
	if err != nil {
		cmd.Process.Kill() //nolint:errcheck
		return nil, fmt.Errorf("%w: %s", ErrBrowserConnection, err.Error())
	}

	b, err := connectBrowser(ctx, wsURI, o, cmd)
	if err != nil {
		cmd.Process.Kill() //nolint:errcheck
		return nil, err
	}
	return b, nil
}

// ConnectToBrowser connects to an already-running Chrome instance at the given
// WebSocket debugging URI (e.g. "ws://localhost:9222/devtools/browser/<id>").
//
// Upstream equivalent: BrowserFactory::connectToBrowser($uri).
func ConnectToBrowser(ctx context.Context, wsURI string) (*Browser, error) {
	var o Options
	applyDefaults(&o)
	return connectBrowser(ctx, wsURI, o, nil)
}

// connectBrowser dials the CDP WebSocket and returns a live Browser.
func connectBrowser(ctx context.Context, wsURI string, o Options, cmd *exec.Cmd) (*Browser, error) {
	conn := cdp.NewConnection(wsURI)
	conn.SetConnectionDelay(o.ConnectionDelay)
	if o.DebugLogger != nil {
		conn.SetLogger(o.DebugLogger)
	}

	if err := conn.Connect(ctx); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrBrowserConnection, err.Error())
	}

	b := &Browser{
		conn:    conn,
		wsURI:   wsURI,
		opts:    o,
		cmd:     cmd,
		closeCh: make(chan struct{}),
	}
	return b, nil
}

// resolveExecutable finds the Chrome/Chromium binary.
func resolveExecutable(hint string) (string, error) {
	if hint != "" {
		return exec.LookPath(hint)
	}
	if env := os.Getenv("CHROME_PATH"); env != "" {
		return exec.LookPath(env)
	}
	candidates := executableCandidates()
	for _, c := range candidates {
		if path, err := exec.LookPath(c); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("chrome or chromium executable not found; set CHROME_PATH or pass a path to NewBrowserFactory")
}

func executableCandidates() []string {
	switch runtime.GOOS {
	case "darwin":
		return []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
			"google-chrome",
			"chromium",
		}
	case "windows":
		return []string{
			`C:\Program Files\Google\Chrome\Application\chrome.exe`,
			`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
			"chrome.exe",
			"chromium.exe",
		}
	default: // linux and others
		return []string{
			"google-chrome",
			"google-chrome-stable",
			"chromium",
			"chromium-browser",
			"chromium-headless-shell",
		}
	}
}

// BuildArgs returns the Chrome command-line flags that would be generated for
// the given Options. Useful for inspecting flag construction in tests.
func BuildArgs(o Options) []string { return buildArgs(o) }

func buildArgs(o Options) []string {
	args := []string{
		"--disable-gpu",
	}

	if o.Headless != nil && *o.Headless {
		args = append(args, "--headless=new")
	}
	if o.NoSandbox {
		args = append(args, "--no-sandbox")
	}
	if o.DisableNotifications {
		args = append(args, "--disable-notifications")
	}
	if o.IgnoreCertificateErrors {
		args = append(args, "--ignore-certificate-errors")
	}
	if o.NoProxyServer {
		args = append(args, "--no-proxy-server")
	}
	if o.ProxyServer != "" {
		args = append(args, "--proxy-server="+o.ProxyServer)
	}
	if len(o.ProxyBypassList) > 0 {
		args = append(args, "--proxy-bypass-list="+strings.Join(o.ProxyBypassList, ","))
	}
	if o.UserAgent != "" {
		args = append(args, "--user-agent="+o.UserAgent)
	}
	if o.UserDataDir != "" {
		args = append(args, "--user-data-dir="+o.UserDataDir)
	}
	if o.UserCrashDumpsDir != "" {
		args = append(args, "--crash-dumps-dir="+o.UserCrashDumpsDir)
	}
	if o.WindowSize != [2]int{} {
		args = append(args, fmt.Sprintf("--window-size=%d,%d", o.WindowSize[0], o.WindowSize[1]))
	}
	if o.EnableImages != nil && !*o.EnableImages {
		args = append(args, "--blink-settings=imagesEnabled=false")
	}

	args = append(args, o.CustomFlags...)

	// Remove excluded switches.
	excluded := make(map[string]bool, len(o.ExcludedSwitches))
	for _, s := range o.ExcludedSwitches {
		excluded[s] = true
	}
	filtered := args[:0]
	for _, a := range args {
		key := strings.TrimLeft(strings.SplitN(a, "=", 2)[0], "-")
		if !excluded[key] {
			filtered = append(filtered, a)
		}
	}
	return filtered
}

// readDevToolsURI reads stderr from a Chrome process until it finds the
// "DevTools listening on ws://..." line, then returns the ws:// URI.
func readDevToolsURI(r interface{ Read([]byte) (int, error) }, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	buf := make([]byte, 0, 4096)
	tmp := make([]byte, 256)
	for time.Now().Before(deadline) {
		n, err := r.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
			if idx := strings.Index(string(buf), "DevTools listening on "); idx >= 0 {
				rest := string(buf[idx+len("DevTools listening on "):])
				line := strings.SplitN(rest, "\n", 2)[0]
				return strings.TrimSpace(line), nil
			}
		}
		if err != nil {
			break
		}
	}
	return "", fmt.Errorf("timed out waiting for DevTools URI from Chrome stderr")
}

// getDevToolsWSURIFromHTTP fetches the WebSocket URI by querying the /json/version endpoint.
// This is a fallback for browsers started with a fixed port.
func getDevToolsWSURIFromHTTP(host string, timeout time.Duration) (string, error) {
	client := &http.Client{Timeout: timeout}
	url := "http://" + host + "/json/version"
	resp, err := client.Get(url) //nolint:noctx
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.WebSocketDebuggerURL == "" {
		return "", fmt.Errorf("empty webSocketDebuggerUrl from %s", url)
	}
	return result.WebSocketDebuggerURL, nil
}
