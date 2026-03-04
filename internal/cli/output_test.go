// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package cli

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/H0llyW00dzZ/nawala-checker/src/nawala"
)

// testWriter creates a Writer backed by a bytes.Buffer for testing.
func testWriter(jsonMode bool) (*Writer, *bytes.Buffer) {
	var buf bytes.Buffer
	return &Writer{w: bufio.NewWriter(&buf), json: jsonMode}, &buf
}

// flushAndRead flushes the Writer and returns the buffer contents.
func flushAndRead(w *Writer, buf *bytes.Buffer) string {
	w.w.Flush()
	return buf.String()
}

func TestNewWriter_Stdout(t *testing.T) {
	w, err := NewWriter("", false)
	if err != nil {
		t.Fatalf("NewWriter(\"\", false) error: %v", err)
	}
	defer w.Close()

	if w.closer != nil {
		t.Error("expected closer to be nil for stdout")
	}
}

func TestNewWriter_File(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.txt")
	w, err := NewWriter(path, false)
	if err != nil {
		t.Fatalf("NewWriter(%q, false) error: %v", path, err)
	}
	defer w.Close()

	if w.closer == nil {
		t.Error("expected closer to be non-nil for file output")
	}
}

func TestNewWriter_InvalidPath(t *testing.T) {
	_, err := NewWriter("/nonexistent/dir/out.txt", false)
	if err == nil {
		t.Fatal("expected error for invalid path, got nil")
	}
}

func TestWriter_WriteResult_Text(t *testing.T) {
	w, buf := testWriter(false)

	w.WriteResult(nawala.Result{
		Domain:  "google.com",
		Blocked: false,
		Server:  "8.8.8.8",
	})

	out := flushAndRead(w, buf)
	if !strings.Contains(out, "google.com") {
		t.Errorf("output missing domain: %q", out)
	}
	if !strings.Contains(out, "not_blocked") {
		t.Errorf("output missing status: %q", out)
	}
	if !strings.Contains(out, "8.8.8.8") {
		t.Errorf("output missing server: %q", out)
	}
}

func TestWriter_WriteResult_TextBlocked(t *testing.T) {
	w, buf := testWriter(false)

	w.WriteResult(nawala.Result{
		Domain:  "blocked.com",
		Blocked: true,
		Server:  "8.8.8.8",
	})

	out := flushAndRead(w, buf)
	if !strings.Contains(out, "BLOCKED") {
		t.Errorf("output missing BLOCKED status: %q", out)
	}
}

func TestWriter_WriteResult_TextError(t *testing.T) {
	w, buf := testWriter(false)

	w.WriteResult(nawala.Result{
		Domain: "fail.com",
		Error:  errors.New("timeout"),
		Server: "8.8.8.8",
	})

	out := flushAndRead(w, buf)
	if !strings.Contains(out, "error: timeout") {
		t.Errorf("output missing error status: %q", out)
	}
}

func TestWriter_WriteResult_JSON(t *testing.T) {
	w, buf := testWriter(true)

	w.WriteResult(nawala.Result{
		Domain:  "google.com",
		Blocked: false,
		Server:  "8.8.8.8",
	})

	out := flushAndRead(w, buf)
	var jr jsonResult
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &jr); err != nil {
		t.Fatalf("invalid JSON output: %v\nraw: %q", err, out)
	}
	if jr.Domain != "google.com" {
		t.Errorf("Domain = %q, want %q", jr.Domain, "google.com")
	}
	if jr.Blocked {
		t.Errorf("Blocked = %v, want false", jr.Blocked)
	}
	if jr.Error != "" {
		t.Errorf("Error = %q, want empty", jr.Error)
	}
}

func TestWriter_WriteResult_JSONWithError(t *testing.T) {
	w, buf := testWriter(true)

	w.WriteResult(nawala.Result{
		Domain: "fail.com",
		Error:  errors.New("dns timeout"),
		Server: "8.8.8.8",
	})

	out := flushAndRead(w, buf)
	var jr jsonResult
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &jr); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if jr.Error != "dns timeout" {
		t.Errorf("Error = %q, want %q", jr.Error, "dns timeout")
	}
}

func TestWriter_WriteStatus_Text_Online(t *testing.T) {
	w, buf := testWriter(false)

	w.WriteStatus(nawala.ServerStatus{
		Server:    "8.8.8.8",
		Online:    true,
		LatencyMs: 42,
	})

	out := flushAndRead(w, buf)
	if !strings.Contains(out, "8.8.8.8") {
		t.Errorf("output missing server: %q", out)
	}
	if !strings.Contains(out, "online") {
		t.Errorf("output missing online status: %q", out)
	}
	if !strings.Contains(out, "42ms") {
		t.Errorf("output missing latency: %q", out)
	}
}

func TestWriter_WriteStatus_Text_Offline(t *testing.T) {
	w, buf := testWriter(false)

	w.WriteStatus(nawala.ServerStatus{
		Server: "1.2.3.4",
		Online: false,
		Error:  errors.New("connection refused"),
	})

	out := flushAndRead(w, buf)
	if !strings.Contains(out, "OFFLINE") {
		t.Errorf("output missing OFFLINE status: %q", out)
	}
	if !strings.Contains(out, "connection refused") {
		t.Errorf("output missing error: %q", out)
	}
}

func TestWriter_WriteStatus_Text_Offline_NoError(t *testing.T) {
	w, buf := testWriter(false)

	w.WriteStatus(nawala.ServerStatus{
		Server: "1.2.3.4",
		Online: false,
	})

	out := flushAndRead(w, buf)
	if !strings.Contains(out, "OFFLINE") {
		t.Errorf("output missing OFFLINE status: %q", out)
	}
}

func TestWriter_WriteStatus_JSON_Online(t *testing.T) {
	w, buf := testWriter(true)

	w.WriteStatus(nawala.ServerStatus{
		Server:    "8.8.8.8",
		Online:    true,
		LatencyMs: 5,
	})

	out := flushAndRead(w, buf)
	var js jsonStatus
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &js); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if js.Server != "8.8.8.8" {
		t.Errorf("Server = %q, want %q", js.Server, "8.8.8.8")
	}
	if !js.Online {
		t.Error("Online = false, want true")
	}
	if js.LatencyMs != 5 {
		t.Errorf("LatencyMs = %d, want 5", js.LatencyMs)
	}
}

func TestWriter_WriteStatus_JSON_Offline(t *testing.T) {
	w, buf := testWriter(true)

	w.WriteStatus(nawala.ServerStatus{
		Server: "1.2.3.4",
		Online: false,
		Error:  errors.New("unreachable"),
	})

	out := flushAndRead(w, buf)
	var js jsonStatus
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &js); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if js.Error != "unreachable" {
		t.Errorf("Error = %q, want %q", js.Error, "unreachable")
	}
}

func TestWriter_Close_File(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.txt")
	w, err := NewWriter(path, false)
	if err != nil {
		t.Fatal(err)
	}

	w.WriteResult(nawala.Result{Domain: "test.com", Server: "8.8.8.8"})
	if err := w.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "test.com") {
		t.Errorf("file content missing domain: %q", string(data))
	}
}

func TestWriter_Close_Stdout(t *testing.T) {
	w, err := NewWriter("", false)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}
}

func TestWriter_Close_FlushError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "flush_err.txt")
	w, err := NewWriter(path, false)
	if err != nil {
		t.Fatal(err)
	}

	// Write something to fill the bufio buffer.
	w.WriteResult(nawala.Result{Domain: "test.com", Server: "8.8.8.8"})

	// Close the underlying file to force a flush error.
	w.closer.Close()
	w.closer = nil // prevent double-close

	// Write more data so bufio has unflushed content.
	w.WriteResult(nawala.Result{Domain: "test2.com", Server: "8.8.8.8"})

	err = w.Close()
	if err == nil {
		t.Fatal("expected error from Close() after underlying file was closed, got nil")
	}
}
