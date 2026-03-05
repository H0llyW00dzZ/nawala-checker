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
func testWriter(jsonMode bool) (*Writer, *bytes.Buffer) {
	var buf bytes.Buffer
	bw := bufio.NewWriter(&buf)
	w := &Writer{w: bw, json: jsonMode}
	if !jsonMode {
		w.tw = tabwriter.NewWriter(bw, 0, 0, 4, ' ', 0)
	}
	return w, &buf
}

// flushAndRead flushes the Writer, closes it (triggering array caps), and returns the buffer contents.
func flushAndRead(w *Writer, buf *bytes.Buffer) string {
	w.Close()
	return buf.String()
}

func TestNewWriter_Stdout(t *testing.T) {
	w, err := NewWriter("", false)
	require.NoError(t, err)
	defer w.Close()

	assert.Nil(t, w.closer, "expected closer to be nil for stdout")
}

func TestNewWriter_File(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.txt")
	w, err := NewWriter(path, false)
	require.NoError(t, err)
	defer w.Close()

	assert.NotNil(t, w.closer, "expected closer to be non-nil for file output")
}

func TestNewWriter_InvalidPath(t *testing.T) {
	_, err := NewWriter("/nonexistent/dir/out.txt", false)
	assert.Error(t, err, "expected error for invalid path")
}

func TestWriter_WriteResult_Text(t *testing.T) {
	w, buf := testWriter(false)

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
	w, buf := testWriter(false)

	w.WriteResult(nawala.Result{
		Domain:  "blocked.com",
		Blocked: true,
		Server:  "8.8.8.8",
	})

	out := flushAndRead(w, buf)
	assert.Contains(t, out, "BLOCKED")
}

func TestWriter_WriteResult_TextError(t *testing.T) {
	w, buf := testWriter(false)

	w.WriteResult(nawala.Result{
		Domain: "fail.com",
		Error:  errors.New("timeout"),
		Server: "8.8.8.8",
	})

	out := flushAndRead(w, buf)
	assert.Contains(t, out, "error: timeout")
}

func TestWriter_WriteResult_JSON(t *testing.T) {
	w, buf := testWriter(true)

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
	w, buf := testWriter(true)

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
	w, buf := testWriter(true)

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
	w, buf := testWriter(false)

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
	w, buf := testWriter(false)

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
	w, buf := testWriter(false)

	w.WriteStatus(nawala.ServerStatus{
		Server: "1.2.3.4",
		Online: false,
	})

	out := flushAndRead(w, buf)
	assert.Contains(t, out, "OFFLINE")
}

func TestWriter_WriteStatus_JSON_Online(t *testing.T) {
	w, buf := testWriter(true)

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
	w, buf := testWriter(true)

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
	w, buf := testWriter(true)

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
	w, err := NewWriter(path, false)
	require.NoError(t, err)

	w.WriteResult(nawala.Result{Domain: "test.com", Server: "8.8.8.8"})
	require.NoError(t, w.Close())

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), "test.com")
}

func TestWriter_Close_Stdout(t *testing.T) {
	w, err := NewWriter("", false)
	require.NoError(t, err)
	assert.NoError(t, w.Close())
}

func TestWriter_Close_FlushError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "flush_err.txt")
	w, err := NewWriter(path, false)
	require.NoError(t, err)

	// Write something to fill the bufio buffer.
	w.WriteResult(nawala.Result{Domain: "test.com", Server: "8.8.8.8"})

	// Close the underlying file to force a flush error.
	w.closer.Close()
	w.closer = nil // prevent double-close

	// Write more data so bufio has unflushed content.
	w.WriteResult(nawala.Result{Domain: "test2.com", Server: "8.8.8.8"})

	err = w.Close()
	assert.Error(t, err, "expected error from Close() after underlying file was closed")
}
