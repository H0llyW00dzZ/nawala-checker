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
	"testing"
	"text/tabwriter"

	"github.com/H0llyW00dzZ/nawala-checker/src/nawala"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testWriter creates a Writer backed by a bytes.Buffer for testing.
func testWriter(format string) (*Writer, *bytes.Buffer) {
	var buf bytes.Buffer
	bw := bufio.NewWriter(&buf)
	w := &Writer{w: bw, format: format}
	if format == FormatText {
		w.tw = tabwriter.NewWriter(bw, 0, 0, 4, ' ', 0)
	}
	return w, &buf
}

// flushAndRead flushes the Writer, closes it (triggering array caps), and returns the buffer contents.
func flushAndRead(w *Writer, buf *bytes.Buffer) string {
	_ = w.Close()
	return buf.String()
}

func TestNewWriter_Stdout(t *testing.T) {
	w, err := NewWriter("", FormatText)
	require.NoError(t, err)
	defer func() {
		_ = w.Close()
	}()

	assert.Nil(t, w.closer, "expected closer to be nil for stdout")
}

func TestNewWriter_File(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.txt")
	w, err := NewWriter(path, FormatText)
	require.NoError(t, err)
	defer func() {
		_ = w.Close()
	}()

	assert.NotNil(t, w.closer, "expected closer to be non-nil for file output")
}

func TestNewWriter_InvalidPath(t *testing.T) {
	_, err := NewWriter("/nonexistent/dir/out.txt", FormatText)
	assert.Error(t, err, "expected error for invalid path")
}

func TestWriter_WriteResult_Text(t *testing.T) {
	w, buf := testWriter(FormatText)

	w.WriteResult(nawala.Result{
		Domain:  "google.com",
		Blocked: false,
		Server:  "8.8.8.8",
	})

	out := flushAndRead(w, buf)
	assert.Contains(t, out, "google.com")
	assert.Contains(t, out, "NOT BLOCKED")
	assert.Contains(t, out, "8.8.8.8")
}

func TestWriter_WriteResult_TextBlocked(t *testing.T) {
	w, buf := testWriter(FormatText)

	w.WriteResult(nawala.Result{
		Domain:  "blocked.com",
		Blocked: true,
		Server:  "8.8.8.8",
	})

	out := flushAndRead(w, buf)
	assert.Contains(t, out, "BLOCKED")
}

func TestWriter_WriteResult_TextError(t *testing.T) {
	w, buf := testWriter(FormatText)

	w.WriteResult(nawala.Result{
		Domain: "fail.com",
		Error:  errors.New("timeout"),
		Server: "8.8.8.8",
	})

	out := flushAndRead(w, buf)
	assert.Contains(t, out, "error: timeout")
}

func TestWriter_WriteResult_JSON(t *testing.T) {
	w, buf := testWriter(FormatJSON)

	w.WriteResult(nawala.Result{
		Domain:  "google.com",
		Blocked: false,
		Server:  "8.8.8.8",
	})

	out := flushAndRead(w, buf)
	var wrapper struct {
		Nawala struct {
			Result []jsonResult `json:"result"`
		} `json:"nawala"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &wrapper))
	require.Len(t, wrapper.Nawala.Result, 1)
	jr := wrapper.Nawala.Result[0]

	assert.Equal(t, "google.com", jr.Domain)
	assert.False(t, jr.Blocked)
	assert.Empty(t, jr.Error)
}

func TestWriter_WriteResult_JSONWithError(t *testing.T) {
	w, buf := testWriter(FormatJSON)

	w.WriteResult(nawala.Result{
		Domain: "fail.com",
		Error:  errors.New("dns timeout"),
		Server: "8.8.8.8",
	})

	out := flushAndRead(w, buf)
	var wrapper struct {
		Nawala struct {
			Result []jsonResult `json:"result"`
		} `json:"nawala"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &wrapper))
	require.Len(t, wrapper.Nawala.Result, 1)
	jr := wrapper.Nawala.Result[0]

	assert.Equal(t, "dns timeout", jr.Error)
}

func TestWriter_WriteResult_JSON_MultipleResults(t *testing.T) {
	w, buf := testWriter(FormatJSON)

	w.WriteResult(nawala.Result{Domain: "google.com", Blocked: false, Server: "8.8.8.8"})
	w.WriteResult(nawala.Result{Domain: "reddit.com", Blocked: true, Server: "8.8.8.8"})

	out := flushAndRead(w, buf)
	var wrapper struct {
		Nawala struct {
			Result []jsonResult `json:"result"`
		} `json:"nawala"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &wrapper))
	require.Len(t, wrapper.Nawala.Result, 2)
	assert.Equal(t, "google.com", wrapper.Nawala.Result[0].Domain)
	assert.Equal(t, "reddit.com", wrapper.Nawala.Result[1].Domain)
	assert.True(t, wrapper.Nawala.Result[1].Blocked)
}

func TestWriter_WriteStatus_Text_Online(t *testing.T) {
	w, buf := testWriter(FormatText)

	w.WriteStatus(nawala.ServerStatus{
		Server:    "8.8.8.8",
		Online:    true,
		LatencyMs: 42,
	})

	out := flushAndRead(w, buf)
	assert.Contains(t, out, "8.8.8.8")
	assert.Contains(t, out, "ONLINE")
	assert.Contains(t, out, "42ms")
}

func TestWriter_WriteStatus_Text_Offline(t *testing.T) {
	w, buf := testWriter(FormatText)

	w.WriteStatus(nawala.ServerStatus{
		Server: "1.2.3.4",
		Online: false,
		Error:  errors.New("connection refused"),
	})

	out := flushAndRead(w, buf)
	assert.Contains(t, out, "OFFLINE")
	assert.Contains(t, out, "connection refused")
}

func TestWriter_WriteStatus_Text_Offline_NoError(t *testing.T) {
	w, buf := testWriter(FormatText)

	w.WriteStatus(nawala.ServerStatus{
		Server: "1.2.3.4",
		Online: false,
	})

	out := flushAndRead(w, buf)
	assert.Contains(t, out, "OFFLINE")
}

func TestWriter_WriteStatus_JSON_Online(t *testing.T) {
	w, buf := testWriter(FormatJSON)

	w.WriteStatus(nawala.ServerStatus{
		Server:    "8.8.8.8",
		Online:    true,
		LatencyMs: 5,
	})

	out := flushAndRead(w, buf)
	var wrapper struct {
		Nawala struct {
			Status []jsonStatus `json:"status"`
		} `json:"nawala"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &wrapper))
	require.Len(t, wrapper.Nawala.Status, 1)
	js := wrapper.Nawala.Status[0]

	assert.Equal(t, "8.8.8.8", js.Server)
	assert.True(t, js.Online)
	assert.Equal(t, int64(5), js.LatencyMs)
}

func TestWriter_WriteStatus_JSON_Offline(t *testing.T) {
	w, buf := testWriter(FormatJSON)

	w.WriteStatus(nawala.ServerStatus{
		Server: "1.2.3.4",
		Online: false,
		Error:  errors.New("unreachable"),
	})

	out := flushAndRead(w, buf)
	var wrapper struct {
		Nawala struct {
			Status []jsonStatus `json:"status"`
		} `json:"nawala"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &wrapper))
	require.Len(t, wrapper.Nawala.Status, 1)
	js := wrapper.Nawala.Status[0]

	assert.Equal(t, "unreachable", js.Error)
}

func TestWriter_WriteStatus_JSON_MultipleStatuses(t *testing.T) {
	w, buf := testWriter(FormatJSON)

	w.WriteStatus(nawala.ServerStatus{Server: "8.8.8.8", Online: true, LatencyMs: 10})
	w.WriteStatus(nawala.ServerStatus{Server: "1.1.1.1", Online: true, LatencyMs: 20})

	out := flushAndRead(w, buf)
	var wrapper struct {
		Nawala struct {
			Status []jsonStatus `json:"status"`
		} `json:"nawala"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &wrapper))
	require.Len(t, wrapper.Nawala.Status, 2)
	assert.Equal(t, "8.8.8.8", wrapper.Nawala.Status[0].Server)
	assert.Equal(t, "1.1.1.1", wrapper.Nawala.Status[1].Server)
}

func TestWriter_Close_File(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.txt")
	w, err := NewWriter(path, FormatText)
	require.NoError(t, err)

	w.WriteResult(nawala.Result{Domain: "test.com", Server: "8.8.8.8"})
	require.NoError(t, w.Close())

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), "test.com")
}

func TestWriter_Close_Stdout(t *testing.T) {
	w, err := NewWriter("", FormatText)
	require.NoError(t, err)
	assert.NoError(t, w.Close())
}

func TestWriter_Close_FlushError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "flush_err.txt")
	w, err := NewWriter(path, FormatText)
	require.NoError(t, err)

	// Write something to fill the bufio buffer.
	w.WriteResult(nawala.Result{Domain: "test.com", Server: "8.8.8.8"})

	// Close the underlying file to force a flush error.
	_ = w.closer.Close()
	w.closer = nil // prevent double-close

	// Write more data so bufio has unflushed content.
	w.WriteResult(nawala.Result{Domain: "test2.com", Server: "8.8.8.8"})

	err = w.Close()
	assert.Error(t, err, "expected error from Close() after underlying file was closed")
}

// --- HTML output tests ---

func TestWriter_WriteResult_HTML(t *testing.T) {
	w, buf := testWriter(FormatHTML)

	w.WriteResult(nawala.Result{Domain: "google.com", Blocked: false, Server: "8.8.8.8"})
	w.WriteResult(nawala.Result{Domain: "blocked.com", Blocked: true, Server: "8.8.8.8"})
	w.WriteResult(nawala.Result{Domain: "fail.com", Error: errors.New("timeout"), Server: "8.8.8.8"})

	out := flushAndRead(w, buf)
	assert.Contains(t, out, "<table>")
	assert.Contains(t, out, "google.com")
	assert.Contains(t, out, "NOT BLOCKED")
	assert.Contains(t, out, "not-blocked")
	assert.Contains(t, out, "blocked.com")
	assert.Contains(t, out, "BLOCKED")
	assert.Contains(t, out, `class="blocked"`)
	assert.Contains(t, out, "fail.com")
	assert.Contains(t, out, "error: timeout")
	assert.Contains(t, out, `class="error"`)
}

func TestWriter_WriteStatus_HTML(t *testing.T) {
	w, buf := testWriter(FormatHTML)

	w.WriteStatus(nawala.ServerStatus{Server: "8.8.8.8", Online: true, LatencyMs: 5})
	w.WriteStatus(nawala.ServerStatus{Server: "1.2.3.4", Online: false, Error: errors.New("refused")})

	out := flushAndRead(w, buf)
	assert.Contains(t, out, "<table>")
	assert.Contains(t, out, "8.8.8.8")
	assert.Contains(t, out, "ONLINE")
	assert.Contains(t, out, `class="online"`)
	assert.Contains(t, out, "1.2.3.4")
	assert.Contains(t, out, "OFFLINE")
	assert.Contains(t, out, `class="offline"`)
	assert.Contains(t, out, "refused")
}

// --- XLSX output tests ---

func TestWriter_WriteResult_XLSX(t *testing.T) {
	path := filepath.Join(t.TempDir(), "results.xlsx")
	w, err := NewWriter(path, FormatXLSX)
	require.NoError(t, err)

	w.WriteResult(nawala.Result{Domain: "google.com", Blocked: false, Server: "8.8.8.8"})
	w.WriteResult(nawala.Result{Domain: "blocked.com", Blocked: true, Server: "8.8.8.8"})
	w.WriteResult(nawala.Result{Domain: "fail.com", Error: errors.New("timeout"), Server: "8.8.8.8"})
	require.NoError(t, w.Close())

	// Verify the file was created and is non-empty.
	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0), "XLSX file should not be empty")
}

func TestWriter_WriteStatus_XLSX(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.xlsx")
	w, err := NewWriter(path, FormatXLSX)
	require.NoError(t, err)

	w.WriteStatus(nawala.ServerStatus{Server: "8.8.8.8", Online: true, LatencyMs: 5})
	w.WriteStatus(nawala.ServerStatus{Server: "1.2.3.4", Online: false, Error: errors.New("refused")})
	require.NoError(t, w.Close())

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0), "XLSX file should not be empty")
}

func TestWriter_XLSX_StdoutWrite(t *testing.T) {
	// Test the XLSX stdout/WriteTo path (outputPath == "").
	w, buf := testWriter(FormatXLSX)

	w.WriteResult(nawala.Result{Domain: "example.com", Blocked: false, Server: "8.8.8.8"})
	err := w.Close()
	require.NoError(t, err)
	assert.Greater(t, buf.Len(), 0, "XLSX stdout output should not be empty")
}

func TestWriter_Close_XLSX_WithCloser(t *testing.T) {
	// Test the Close() XLSX branch where closer != nil (file output).
	path := filepath.Join(t.TempDir(), "close_test.xlsx")
	w, err := NewWriter(path, FormatXLSX)
	require.NoError(t, err)

	w.WriteResult(nawala.Result{Domain: "test.com", Blocked: false, Server: "8.8.8.8"})
	require.NoError(t, w.Close())

	// Verify file exists and is valid.
	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0))
}

func TestWriter_Close_HTML_EmptyResults(t *testing.T) {
	// Test closeHTML with no results and no statuses — returns nil.
	w, buf := testWriter(FormatHTML)
	err := w.Close()
	require.NoError(t, err)
	assert.Empty(t, buf.String(), "empty HTML output should produce no content")
}

func TestWriter_WriteResult_HTML_ErrorOnly(t *testing.T) {
	// Specifically target the error branch in HTML result rendering.
	w, buf := testWriter(FormatHTML)

	w.WriteResult(nawala.Result{Domain: "err.com", Error: errors.New("dns fail"), Server: "8.8.8.8"})

	out := flushAndRead(w, buf)
	assert.Contains(t, out, "err.com")
	assert.Contains(t, out, "error: dns fail")
	assert.Contains(t, out, `class="error"`)
}

func TestWriter_WriteStatus_HTML_Online_Only(t *testing.T) {
	// Specifically target the online-only path in HTML status rendering.
	w, buf := testWriter(FormatHTML)

	w.WriteStatus(nawala.ServerStatus{Server: "8.8.8.8", Online: true, LatencyMs: 10})

	out := flushAndRead(w, buf)
	assert.Contains(t, out, `class="online"`)
	assert.Contains(t, out, "10ms")
}

func TestWriter_WriteStatus_XLSX_OfflineNoError(t *testing.T) {
	// Test XLSX status with offline server and no error message.
	path := filepath.Join(t.TempDir(), "status_no_err.xlsx")
	w, err := NewWriter(path, FormatXLSX)
	require.NoError(t, err)

	w.WriteStatus(nawala.ServerStatus{Server: "1.2.3.4", Online: false})
	require.NoError(t, w.Close())

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0))
}

type errWriter struct{}

func (errWriter) Write(p []byte) (n int, err error) {
	return 0, errors.New("forced write error")
}

func TestWriter_Close_HTML_Error(t *testing.T) {
	// Use an extremely small bufio.Writer over an errWriter to force
	// template.Execute to fail when it flushes.
	w := &Writer{format: FormatHTML}
	w.w = bufio.NewWriterSize(errWriter{}, 16)
	w.WriteResult(nawala.Result{Domain: "test.com", Server: "8.8.8.8"})

	err := w.Close()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "forced write error")
}

func TestWriter_Close_XLSX_Error(t *testing.T) {
	// Force f.WriteTo(w.w) to fail by using errWriter.
	w := &Writer{format: FormatXLSX}
	w.w = bufio.NewWriterSize(errWriter{}, 16)
	w.WriteResult(nawala.Result{Domain: "test.com", Server: "8.8.8.8"})

	err := w.Close()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "forced write error")
}

// --- Tests for pitfall fixes ---

func TestNewWriter_XLSX_SkipsFileCreate(t *testing.T) {
	// Pitfall #6: XLSX with a file path should NOT open an os.Create file
	// descriptor. excelize.SaveAs manages file I/O directly.
	path := filepath.Join(t.TempDir(), "skip_create.xlsx")
	w, err := NewWriter(path, FormatXLSX)
	require.NoError(t, err)

	assert.Nil(t, w.w, "expected w.w to be nil for XLSX file path (no bufio.Writer)")
	assert.Nil(t, w.closer, "expected closer to be nil for XLSX file path (no os.Create)")
	assert.Equal(t, path, w.outputPath, "expected outputPath to be set")

	// Write and close; the XLSX should still be created via SaveAs.
	w.WriteResult(nawala.Result{Domain: "test.com", Blocked: false, Server: "8.8.8.8"})
	require.NoError(t, w.Close())

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0), "XLSX file should be non-empty")
}

func TestWriter_Close_JSON_EmptyResults(t *testing.T) {
	// Pitfall #8: Closing a JSON writer with zero results should emit
	// a valid empty JSON envelope instead of an empty file.
	w, buf := testWriter(FormatJSON)

	err := w.Close()
	require.NoError(t, err)

	out := buf.String()
	var wrapper struct {
		Nawala struct {
			Result []jsonResult `json:"result"`
		} `json:"nawala"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &wrapper),
		"expected valid JSON, got: %s", out)
	assert.Empty(t, wrapper.Nawala.Result, "expected empty result array")
}

func TestWriter_Close_JSON_EmptyStatuses(t *testing.T) {
	// Pitfall #8: When a JSON writer with "status" key has zero writes,
	// Close should emit a valid empty envelope. We simulate this by
	// setting jsonKey manually (since no WriteStatus was called).
	w, buf := testWriter(FormatJSON)
	w.jsonKey = "status" // simulate the intent to write statuses

	err := w.Close()
	require.NoError(t, err)

	out := buf.String()
	var wrapper struct {
		Nawala struct {
			Status []jsonStatus `json:"status"`
		} `json:"nawala"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &wrapper),
		"expected valid JSON, got: %s", out)
	assert.Empty(t, wrapper.Nawala.Status, "expected empty status array")
}
