package chrome

import (
	"log/slog"
	"time"
)

// Options holds the configuration for launching or connecting to a Chrome browser.
// All fields are optional; zero values are treated as "unset" and defaults are applied.
type Options struct {
	// ConnectionDelay adds an artificial delay between CDP messages (useful for debugging).
	ConnectionDelay time.Duration

	// CustomFlags passes arbitrary command-line flags to the Chrome binary.
	CustomFlags []string

	// DebugLogger receives internal debug messages. Defaults to slog.Default().
	DebugLogger *slog.Logger

	// DisableNotifications suppresses browser notification prompts.
	DisableNotifications bool

	// EnableImages controls whether images are loaded. nil means "unset" (browser default).
	EnableImages *bool

	// EnvVariables are additional environment variables set for the browser process.
	EnvVariables map[string]string

	// Headers sets default HTTP headers sent on every request.
	Headers map[string]string

	// Headless runs the browser without a visible window. nil means "unset" (defaults to true).
	Headless *bool

	// IgnoreCertificateErrors disables TLS certificate verification.
	IgnoreCertificateErrors bool

	// KeepAlive prevents the browser process from being killed when the last page is closed.
	KeepAlive bool

	// NoSandbox disables the Chrome sandbox (required in some container environments).
	NoSandbox bool

	// NoProxyServer disables proxy auto-detection.
	NoProxyServer bool

	// ProxyBypassList is a comma-separated list of hosts to bypass the proxy.
	ProxyBypassList []string

	// ProxyServer sets the proxy server address (host:port).
	ProxyServer string

	// SendSyncDefaultTimeout is the default timeout for synchronous CDP operations.
	// Defaults to 30s.
	SendSyncDefaultTimeout time.Duration

	// StartupTimeout is how long to wait for the browser to start. Defaults to 10s.
	StartupTimeout time.Duration

	// UserAgent overrides the default browser user-agent string.
	UserAgent string

	// UserDataDir sets the profile directory Chrome uses.
	UserDataDir string

	// UserCrashDumpsDir sets the directory where crash dumps are written.
	UserCrashDumpsDir string

	// WindowSize sets the initial browser window size as [width, height].
	// Zero values mean "unset".
	WindowSize [2]int

	// ExcludedSwitches lists Chrome command-line switches to remove even if
	// they would normally be set.
	ExcludedSwitches []string
}

// DefaultOptions returns an Options populated with all library defaults.
// Useful for inspecting or overriding defaults before passing to BrowserFactory.
func DefaultOptions() Options {
	var o Options
	applyDefaults(&o)
	return o
}

// MergeOptions overlays src onto base: non-zero src fields replace base fields.
// Values already set in base are kept when the corresponding src field is zero.
func MergeOptions(base, src Options) Options { return mergeOptions(base, src) }

// applyDefaults fills in sensible defaults for unset fields.
func applyDefaults(o *Options) {
	if o.Headless == nil {
		t := true
		o.Headless = &t
	}
	if o.StartupTimeout == 0 {
		o.StartupTimeout = 10 * time.Second
	}
	if o.SendSyncDefaultTimeout == 0 {
		o.SendSyncDefaultTimeout = 30 * time.Second
	}
	if o.DebugLogger == nil {
		o.DebugLogger = slog.Default()
	}
}

// merge returns a new Options that overlays src onto base.
// Non-zero / non-nil values in src overwrite those in base.
func mergeOptions(base, src Options) Options {
	out := base

	if src.ConnectionDelay != 0 {
		out.ConnectionDelay = src.ConnectionDelay
	}
	if len(src.CustomFlags) > 0 {
		out.CustomFlags = src.CustomFlags
	}
	if src.DebugLogger != nil {
		out.DebugLogger = src.DebugLogger
	}
	if src.DisableNotifications {
		out.DisableNotifications = true
	}
	if src.EnableImages != nil {
		out.EnableImages = src.EnableImages
	}
	if len(src.EnvVariables) > 0 {
		out.EnvVariables = src.EnvVariables
	}
	if len(src.Headers) > 0 {
		out.Headers = src.Headers
	}
	if src.Headless != nil {
		out.Headless = src.Headless
	}
	if src.IgnoreCertificateErrors {
		out.IgnoreCertificateErrors = true
	}
	if src.KeepAlive {
		out.KeepAlive = true
	}
	if src.NoSandbox {
		out.NoSandbox = true
	}
	if src.NoProxyServer {
		out.NoProxyServer = true
	}
	if len(src.ProxyBypassList) > 0 {
		out.ProxyBypassList = src.ProxyBypassList
	}
	if src.ProxyServer != "" {
		out.ProxyServer = src.ProxyServer
	}
	if src.SendSyncDefaultTimeout != 0 {
		out.SendSyncDefaultTimeout = src.SendSyncDefaultTimeout
	}
	if src.StartupTimeout != 0 {
		out.StartupTimeout = src.StartupTimeout
	}
	if src.UserAgent != "" {
		out.UserAgent = src.UserAgent
	}
	if src.UserDataDir != "" {
		out.UserDataDir = src.UserDataDir
	}
	if src.UserCrashDumpsDir != "" {
		out.UserCrashDumpsDir = src.UserCrashDumpsDir
	}
	if src.WindowSize != [2]int{} {
		out.WindowSize = src.WindowSize
	}
	if len(src.ExcludedSwitches) > 0 {
		out.ExcludedSwitches = src.ExcludedSwitches
	}
	return out
}
