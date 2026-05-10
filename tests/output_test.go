package chrome_test

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	chrome "github.com/masfu/chrome-go"
)

// ---- Screenshot ----

func TestScreenshot_Bytes(t *testing.T) {
	data := []byte("fake-png-bytes")
	s := chrome.NewScreenshotFromBytes(data)
	if string(s.Bytes()) != "fake-png-bytes" {
		t.Errorf("Bytes(): got %q", s.Bytes())
	}
}

func TestScreenshot_Base64(t *testing.T) {
	data := []byte("fake-png-bytes")
	s := chrome.NewScreenshotFromBytes(data)
	want := base64.StdEncoding.EncodeToString(data)
	if s.Base64() != want {
		t.Errorf("Base64(): got %q, want %q", s.Base64(), want)
	}
}

func TestScreenshot_SaveToFile(t *testing.T) {
	data := []byte{0x89, 0x50, 0x4E, 0x47} // PNG magic bytes
	s := chrome.NewScreenshotFromBytes(data)

	dir := t.TempDir()
	path := filepath.Join(dir, "test.png")

	if err := s.SaveToFile(path); err != nil {
		t.Fatalf("SaveToFile: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != string(data) {
		t.Errorf("file contents mismatch")
	}
}

func TestScreenshot_SaveToFile_PermissionError(t *testing.T) {
	s := chrome.NewScreenshotFromBytes([]byte("x"))
	err := s.SaveToFile("/nonexistent/path/test.png")
	if err == nil {
		t.Error("expected error when saving to non-existent directory, got nil")
	}
}

// ---- PDF ----

func TestPDF_Bytes(t *testing.T) {
	data := []byte("%PDF-1.4")
	p := chrome.NewPDFFromBytes(data)
	if string(p.Bytes()) != "%PDF-1.4" {
		t.Errorf("Bytes(): got %q", p.Bytes())
	}
}

func TestPDF_Base64(t *testing.T) {
	data := []byte("%PDF-1.4")
	p := chrome.NewPDFFromBytes(data)
	want := base64.StdEncoding.EncodeToString(data)
	if p.Base64() != want {
		t.Errorf("Base64(): got %q, want %q", p.Base64(), want)
	}
}

func TestPDF_SaveToFile(t *testing.T) {
	data := []byte("%PDF-1.4 fake content")
	p := chrome.NewPDFFromBytes(data)

	dir := t.TempDir()
	path := filepath.Join(dir, "test.pdf")

	if err := p.SaveToFile(path); err != nil {
		t.Fatalf("SaveToFile: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != string(data) {
		t.Errorf("file contents mismatch")
	}
}

// ---- Evaluation ----

func TestEvaluation_ReturnValue_String(t *testing.T) {
	e := chrome.NewEvaluationFromJSON(json.RawMessage(`"hello"`))
	val, err := e.ReturnValue()
	if err != nil {
		t.Fatalf("ReturnValue: %v", err)
	}
	s, ok := val.(string)
	if !ok || s != "hello" {
		t.Errorf("ReturnValue: want string \"hello\", got %T %v", val, val)
	}
}

func TestEvaluation_ReturnValue_Number(t *testing.T) {
	e := chrome.NewEvaluationFromJSON(json.RawMessage(`42`))
	val, err := e.ReturnValue()
	if err != nil {
		t.Fatalf("ReturnValue: %v", err)
	}
	n, ok := val.(float64)
	if !ok || n != 42 {
		t.Errorf("ReturnValue: want float64(42), got %T %v", val, val)
	}
}

func TestEvaluation_ReturnValue_Bool(t *testing.T) {
	e := chrome.NewEvaluationFromJSON(json.RawMessage(`true`))
	val, err := e.ReturnValue()
	if err != nil {
		t.Fatalf("ReturnValue: %v", err)
	}
	b, ok := val.(bool)
	if !ok || !b {
		t.Errorf("ReturnValue: want bool(true), got %T %v", val, val)
	}
}

func TestEvaluation_ReturnValue_Null(t *testing.T) {
	e := chrome.NewEvaluationFromJSON(json.RawMessage(`null`))
	val, err := e.ReturnValue()
	if err != nil {
		t.Fatalf("ReturnValue: %v", err)
	}
	if val != nil {
		t.Errorf("ReturnValue: want nil for JSON null, got %v", val)
	}
}

func TestEvaluation_ReturnValue_Empty(t *testing.T) {
	e := chrome.NewEvaluationFromJSON(nil)
	val, err := e.ReturnValue()
	if err != nil {
		t.Fatalf("ReturnValue: %v", err)
	}
	if val != nil {
		t.Errorf("ReturnValue: want nil for empty raw, got %v", val)
	}
}

func TestEvaluation_ReturnValue_Map(t *testing.T) {
	e := chrome.NewEvaluationFromJSON(json.RawMessage(`{"key":"value"}`))
	val, err := e.ReturnValue()
	if err != nil {
		t.Fatalf("ReturnValue: %v", err)
	}
	m, ok := val.(map[string]any)
	if !ok {
		t.Fatalf("ReturnValue: want map, got %T", val)
	}
	if m["key"] != "value" {
		t.Errorf("ReturnValue map[key]: want value, got %v", m["key"])
	}
}
