package chrome_test

import (
	"log/slog"
	"testing"
	"time"

	chrome "github.com/masfu/chrome-go"
)

// ---- DefaultOptions ----

func TestDefaultOptions_HeadlessTrue(t *testing.T) {
	o := chrome.DefaultOptions()
	if o.Headless == nil {
		t.Fatal("Headless should not be nil")
	}
	if !*o.Headless {
		t.Error("Headless should default to true")
	}
}

func TestDefaultOptions_StartupTimeout(t *testing.T) {
	o := chrome.DefaultOptions()
	if o.StartupTimeout != 10*time.Second {
		t.Errorf("StartupTimeout want 10s, got %v", o.StartupTimeout)
	}
}

func TestDefaultOptions_SendSyncDefaultTimeout(t *testing.T) {
	o := chrome.DefaultOptions()
	if o.SendSyncDefaultTimeout != 30*time.Second {
		t.Errorf("SendSyncDefaultTimeout want 30s, got %v", o.SendSyncDefaultTimeout)
	}
}

func TestDefaultOptions_DebugLoggerSet(t *testing.T) {
	o := chrome.DefaultOptions()
	if o.DebugLogger == nil {
		t.Error("DebugLogger should not be nil")
	}
}

// ---- MergeOptions ----

func TestMergeOptions_SrcOverridesBase(t *testing.T) {
	base := chrome.Options{ProxyServer: "base:8080"}
	src := chrome.Options{ProxyServer: "src:9090"}
	out := chrome.MergeOptions(base, src)
	if out.ProxyServer != "src:9090" {
		t.Errorf("ProxyServer want src:9090, got %q", out.ProxyServer)
	}
}

func TestMergeOptions_BaseRetainedWhenSrcZero(t *testing.T) {
	base := chrome.Options{ProxyServer: "base:8080"}
	out := chrome.MergeOptions(base, chrome.Options{})
	if out.ProxyServer != "base:8080" {
		t.Errorf("ProxyServer should be retained from base, got %q", out.ProxyServer)
	}
}

func TestMergeOptions_ConnectionDelay(t *testing.T) {
	base := chrome.Options{ConnectionDelay: 100 * time.Millisecond}
	src := chrome.Options{ConnectionDelay: 200 * time.Millisecond}
	out := chrome.MergeOptions(base, src)
	if out.ConnectionDelay != 200*time.Millisecond {
		t.Errorf("ConnectionDelay want 200ms, got %v", out.ConnectionDelay)
	}
}

func TestMergeOptions_ConnectionDelayZeroDoesNotOverride(t *testing.T) {
	base := chrome.Options{ConnectionDelay: 100 * time.Millisecond}
	out := chrome.MergeOptions(base, chrome.Options{})
	if out.ConnectionDelay != 100*time.Millisecond {
		t.Errorf("ConnectionDelay should stay 100ms, got %v", out.ConnectionDelay)
	}
}

func TestMergeOptions_BoolFlags(t *testing.T) {
	out := chrome.MergeOptions(chrome.Options{}, chrome.Options{
		DisableNotifications:    true,
		IgnoreCertificateErrors: true,
		KeepAlive:               true,
		NoSandbox:               true,
		NoProxyServer:           true,
	})
	if !out.DisableNotifications {
		t.Error("DisableNotifications should be true")
	}
	if !out.IgnoreCertificateErrors {
		t.Error("IgnoreCertificateErrors should be true")
	}
	if !out.KeepAlive {
		t.Error("KeepAlive should be true")
	}
	if !out.NoSandbox {
		t.Error("NoSandbox should be true")
	}
	if !out.NoProxyServer {
		t.Error("NoProxyServer should be true")
	}
}

func TestMergeOptions_CustomFlags(t *testing.T) {
	base := chrome.Options{CustomFlags: []string{"--a"}}
	src := chrome.Options{CustomFlags: []string{"--b", "--c"}}
	out := chrome.MergeOptions(base, src)
	if len(out.CustomFlags) != 2 || out.CustomFlags[0] != "--b" {
		t.Errorf("CustomFlags not overridden correctly, got %v", out.CustomFlags)
	}
}

func TestMergeOptions_EnableImages(t *testing.T) {
	f := false
	out := chrome.MergeOptions(chrome.Options{}, chrome.Options{EnableImages: &f})
	if out.EnableImages == nil || *out.EnableImages {
		t.Error("EnableImages should be set to false")
	}
}

func TestMergeOptions_WindowSize(t *testing.T) {
	out := chrome.MergeOptions(chrome.Options{}, chrome.Options{WindowSize: [2]int{1920, 1080}})
	if out.WindowSize != [2]int{1920, 1080} {
		t.Errorf("WindowSize not set correctly: %v", out.WindowSize)
	}
}

func TestMergeOptions_ExcludedSwitches(t *testing.T) {
	out := chrome.MergeOptions(chrome.Options{}, chrome.Options{ExcludedSwitches: []string{"disable-gpu"}})
	if len(out.ExcludedSwitches) != 1 || out.ExcludedSwitches[0] != "disable-gpu" {
		t.Errorf("ExcludedSwitches not set correctly: %v", out.ExcludedSwitches)
	}
}

func TestMergeOptions_DebugLogger(t *testing.T) {
	custom := slog.Default()
	out := chrome.MergeOptions(chrome.Options{}, chrome.Options{DebugLogger: custom})
	if out.DebugLogger != custom {
		t.Error("DebugLogger should be overridden by src")
	}
}

// ---- BuildArgs ----

func containsArg(args []string, target string) bool {
	for _, a := range args {
		if a == target {
			return true
		}
	}
	return false
}

func TestBuildArgs_HeadlessNew(t *testing.T) {
	h := true
	args := chrome.BuildArgs(chrome.Options{Headless: &h})
	if !containsArg(args, "--headless=new") {
		t.Errorf("expected --headless=new in args, got %v", args)
	}
}

func TestBuildArgs_NoHeadlessWhenFalse(t *testing.T) {
	h := false
	args := chrome.BuildArgs(chrome.Options{Headless: &h})
	if containsArg(args, "--headless=new") {
		t.Errorf("--headless=new should not appear when Headless=false, got %v", args)
	}
}

func TestBuildArgs_NoSandbox(t *testing.T) {
	args := chrome.BuildArgs(chrome.Options{NoSandbox: true})
	if !containsArg(args, "--no-sandbox") {
		t.Errorf("expected --no-sandbox in args, got %v", args)
	}
}

func TestBuildArgs_ProxyServer(t *testing.T) {
	args := chrome.BuildArgs(chrome.Options{ProxyServer: "socks5://proxy:1080"})
	want := "--proxy-server=socks5://proxy:1080"
	if !containsArg(args, want) {
		t.Errorf("expected %q in args, got %v", want, args)
	}
}

func TestBuildArgs_UserAgent(t *testing.T) {
	args := chrome.BuildArgs(chrome.Options{UserAgent: "MyBot/1.0"})
	want := "--user-agent=MyBot/1.0"
	if !containsArg(args, want) {
		t.Errorf("expected %q in args, got %v", want, args)
	}
}

func TestBuildArgs_WindowSize(t *testing.T) {
	args := chrome.BuildArgs(chrome.Options{WindowSize: [2]int{1280, 720}})
	want := "--window-size=1280,720"
	if !containsArg(args, want) {
		t.Errorf("expected %q in args, got %v", want, args)
	}
}

func TestBuildArgs_DisableImages(t *testing.T) {
	f := false
	args := chrome.BuildArgs(chrome.Options{EnableImages: &f})
	want := "--blink-settings=imagesEnabled=false"
	if !containsArg(args, want) {
		t.Errorf("expected %q in args, got %v", want, args)
	}
}

func TestBuildArgs_ExcludedSwitchesRemovesFlag(t *testing.T) {
	h := true
	args := chrome.BuildArgs(chrome.Options{
		Headless:         &h,
		ExcludedSwitches: []string{"headless"},
	})
	if containsArg(args, "--headless=new") {
		t.Errorf("--headless=new should be excluded, got %v", args)
	}
}

func TestBuildArgs_CustomFlagsAppended(t *testing.T) {
	args := chrome.BuildArgs(chrome.Options{CustomFlags: []string{"--my-flag", "--another"}})
	if !containsArg(args, "--my-flag") || !containsArg(args, "--another") {
		t.Errorf("custom flags not present in args: %v", args)
	}
}

func TestBuildArgs_DisableNotifications(t *testing.T) {
	args := chrome.BuildArgs(chrome.Options{DisableNotifications: true})
	if !containsArg(args, "--disable-notifications") {
		t.Errorf("expected --disable-notifications in args, got %v", args)
	}
}

func TestBuildArgs_IgnoreCertificateErrors(t *testing.T) {
	args := chrome.BuildArgs(chrome.Options{IgnoreCertificateErrors: true})
	if !containsArg(args, "--ignore-certificate-errors") {
		t.Errorf("expected --ignore-certificate-errors in args, got %v", args)
	}
}

func TestBuildArgs_ProxyBypassList(t *testing.T) {
	args := chrome.BuildArgs(chrome.Options{ProxyBypassList: []string{"localhost", "127.0.0.1"}})
	want := "--proxy-bypass-list=localhost,127.0.0.1"
	if !containsArg(args, want) {
		t.Errorf("expected %q in args, got %v", want, args)
	}
}
