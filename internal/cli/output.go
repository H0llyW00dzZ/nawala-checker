// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/H0llyW00dzZ/nawala-checker/src/nawala"
)

// Writer handles formatted output of check results to stdout or a file.
type Writer struct {
	w           *bufio.Writer
	tw          *tabwriter.Writer // tab-aligned text output (nil in JSON mode)
	closer      io.Closer        // non-nil when writing to a file
	json        bool             // output as JSON
	jsonStarted bool             // tracks if we started the JSON array
}

// NewWriter creates a Writer that writes to the given path.
// If path is empty, it writes to stdout.
func NewWriter(path string, jsonMode bool) (*Writer, error) {
	w := &Writer{json: jsonMode}

	if path == "" {
		w.w = bufio.NewWriter(os.Stdout)
	} else {
		f, err := os.Create(path)
		if err != nil {
			return nil, fmt.Errorf("creating output file: %w", err)
		}
		w.w = bufio.NewWriter(f)
		w.closer = f
	}

	// For text-mode output, wrap the buffered writer in a tabwriter
	// so columns align dynamically regardless of domain length.
	if !jsonMode {
		w.tw = tabwriter.NewWriter(w.w, 0, 0, 4, ' ', 0)
	}

	return w, nil
}

// jsonResult is the JSON representation of a check result.
type jsonResult struct {
	Domain  string `json:"domain"`
	Blocked bool   `json:"blocked"`
	Server  string `json:"server"`
	Error   string `json:"error,omitempty"`
}

// WriteResult formats and writes a single check result.
func (w *Writer) WriteResult(r nawala.Result) {
	if w.json {
		w.writeJSON(r)
		return
	}
	w.writeText(r)
}

// writeText writes a check result as a tab-aligned text line.
// The tabwriter computes column widths at flush time so that all
// rows share the same alignment, regardless of domain length.
func (w *Writer) writeText(r nawala.Result) {
	status := "NOT BLOCKED"
	if r.Blocked {
		status = "BLOCKED"
	}
	if r.Error != nil {
		status = fmt.Sprintf("error: %v", r.Error)
	}
	fmt.Fprintf(w.tw, "%s\t%s\t%s\n", r.Domain, status, r.Server)
}

// writeJSON writes a check result as a JSON array element.
func (w *Writer) writeJSON(r nawala.Result) {
	jr := jsonResult{
		Domain:  r.Domain,
		Blocked: r.Blocked,
		Server:  r.Server,
	}
	if r.Error != nil {
		jr.Error = r.Error.Error()
	}

	data, _ := json.Marshal(jr)
	if !w.jsonStarted {
		w.w.WriteString(`{"nawala":{"result":[`)
		w.jsonStarted = true
	} else {
		w.w.WriteString(",")
	}
	w.w.Write(data)
	w.w.Flush()
}

// jsonStatus is the JSON representation of a server health status.
type jsonStatus struct {
	Server    string `json:"server"`
	Online    bool   `json:"online"`
	LatencyMs int64  `json:"latency_ms,omitempty"`
	Error     string `json:"error,omitempty"`
}

// WriteStatus formats and writes a single server status.
func (w *Writer) WriteStatus(s nawala.ServerStatus) {
	if w.json {
		w.writeStatusJSON(s)
		return
	}
	w.writeStatusText(s)
}

// writeStatusText writes a server health status as a tab-aligned text line.
func (w *Writer) writeStatusText(s nawala.ServerStatus) {
	status := "ONLINE"
	if !s.Online {
		status = "OFFLINE"
	}
	if s.Online {
		fmt.Fprintf(w.tw, "%s\t%s\t%dms\n", s.Server, status, s.LatencyMs)
	} else {
		errMsg := ""
		if s.Error != nil {
			errMsg = s.Error.Error()
		}
		fmt.Fprintf(w.tw, "%s\t%s\t%s\n", s.Server, status, errMsg)
	}
}

// writeStatusJSON writes a server health status as a JSON array element.
func (w *Writer) writeStatusJSON(s nawala.ServerStatus) {
	js := jsonStatus{
		Server:    s.Server,
		Online:    s.Online,
		LatencyMs: s.LatencyMs,
	}
	if s.Error != nil {
		js.Error = s.Error.Error()
	}
	data, _ := json.Marshal(js)

	if !w.jsonStarted {
		w.w.WriteString(`{"nawala":{"status":[`)
		w.jsonStarted = true
	} else {
		w.w.WriteString(",")
	}
	w.w.Write(data)
	w.w.Flush()
}

// Close flushes any buffered data, writes JSON caps, and closes the file.
func (w *Writer) Close() error {
	if w.json && w.jsonStarted {
		w.w.WriteString("]}}\n")
		w.jsonStarted = false
	}

	// Flush the tabwriter first (computes column widths and writes to w.w),
	// then flush the underlying bufio.Writer to the destination.
	if w.tw != nil {
		w.tw.Flush()
	}

	if err := w.w.Flush(); err != nil {
		return err
	}
	if w.closer != nil {
		return w.closer.Close()
	}
	return nil
}
