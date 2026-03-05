// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	htmltemplate "html/template"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/H0llyW00dzZ/nawala-checker/src/nawala"
	"github.com/xuri/excelize/v2"
)

// Parsed HTML templates — validated once at startup via template.Must.
var (
	resultTmpl = htmltemplate.Must(htmltemplate.New("result").Parse(resultHTMLTemplate))
	statusTmpl = htmltemplate.Must(htmltemplate.New("status").Parse(statusHTMLTemplate))
)

// Output format constants.
const (
	FormatText = "text"
	FormatJSON = "json"
	FormatHTML = "html"
	FormatXLSX = "xlsx"
)

// Writer handles formatted output of check results to stdout or a file.
type Writer struct {
	w      *bufio.Writer
	tw     *tabwriter.Writer // tab-aligned text output (text mode only)
	closer io.Closer         // non-nil when writing to a file

	format      string // "text", "json", "html", "xlsx"
	jsonStarted bool   // tracks if we started the JSON array
	jsonKey     string // "result" or "status" — set by the first JSON write
	outputPath  string // original path (needed by XLSX SaveAs)

	// Buffered results/statuses for HTML and XLSX (rendered at Close time).
	results  []nawala.Result
	statuses []nawala.ServerStatus
}

// NewWriter creates a Writer that writes to the given path.
// If path is empty, it writes to stdout.
// Format must be one of: "text", "json", "html", "xlsx".
//
// For XLSX format with a non-empty path, no file is opened here;
// [excelize.File.SaveAs] creates the file directly at Close time.
func NewWriter(path string, format string) (*Writer, error) {
	w := &Writer{format: format, outputPath: path}

	// XLSX with a file path: excelize.SaveAs manages file I/O directly,
	// so we skip os.Create to avoid an unused file descriptor.
	if format == FormatXLSX && path != "" {
		// w.w and w.closer remain nil; closeXLSX writes via SaveAs.
		return w, nil
	}

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
	if format == FormatText {
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
	switch w.format {
	case FormatJSON:
		w.writeJSON(r)
	case FormatHTML, FormatXLSX:
		w.results = append(w.results, r)
	default:
		w.writeText(r)
	}
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
	_, _ = fmt.Fprintf(w.tw, "%s\t%s\t%s\n", r.Domain, status, r.Server)
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
		_, _ = w.w.WriteString(`{"nawala":{"result":[`)
		w.jsonStarted = true
		w.jsonKey = "result"
	} else {
		_, _ = w.w.WriteString(",")
	}
	_, _ = w.w.Write(data)
	_ = w.w.Flush()
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
	switch w.format {
	case FormatJSON:
		w.writeStatusJSON(s)
	case FormatHTML, FormatXLSX:
		w.statuses = append(w.statuses, s)
	default:
		w.writeStatusText(s)
	}
}

// writeStatusText writes a server health status as a tab-aligned text line.
func (w *Writer) writeStatusText(s nawala.ServerStatus) {
	status := "ONLINE"
	if !s.Online {
		status = "OFFLINE"
	}
	if s.Online {
		_, _ = fmt.Fprintf(w.tw, "%s\t%s\t%dms\n", s.Server, status, s.LatencyMs)
	} else {
		errMsg := ""
		if s.Error != nil {
			errMsg = strings.TrimPrefix(s.Error.Error(), "nawala: ")
		}
		_, _ = fmt.Fprintf(w.tw, "%s\t%s\t%s\n", s.Server, status, errMsg)
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
		_, _ = w.w.WriteString(`{"nawala":{"status":[`)
		w.jsonStarted = true
		w.jsonKey = "status"
	} else {
		_, _ = w.w.WriteString(",")
	}
	_, _ = w.w.Write(data)
	_ = w.w.Flush()
}

// htmlResultData is the data passed to the result HTML template.
type htmlResultData struct {
	Results []htmlResult
}

// htmlResult is a single row in the HTML result table.
type htmlResult struct {
	Domain  string
	Blocked bool
	Server  string
	Error   string
}

// htmlStatusData is the data passed to the status HTML template.
type htmlStatusData struct {
	Statuses []htmlStatus
}

// htmlStatus is a single row in the HTML status table.
type htmlStatus struct {
	Server    string
	Online    bool
	LatencyMs int64
	Error     string
}

// closeHTML renders the embedded HTML template with collected results or statuses.
func (w *Writer) closeHTML() error {
	if len(w.results) > 0 {
		data := htmlResultData{Results: make([]htmlResult, len(w.results))}
		for i, r := range w.results {
			data.Results[i] = htmlResult{
				Domain:  r.Domain,
				Blocked: r.Blocked,
				Server:  r.Server,
			}
			if r.Error != nil {
				data.Results[i].Error = r.Error.Error()
			}
		}
		return resultTmpl.Execute(w.w, data)
	}

	if len(w.statuses) > 0 {
		data := htmlStatusData{Statuses: make([]htmlStatus, len(w.statuses))}
		for i, s := range w.statuses {
			data.Statuses[i] = htmlStatus{
				Server:    s.Server,
				Online:    s.Online,
				LatencyMs: s.LatencyMs,
			}
			if s.Error != nil {
				data.Statuses[i].Error = s.Error.Error()
			}
		}
		return statusTmpl.Execute(w.w, data)
	}

	return nil
}

// closeXLSX builds an Excel workbook from collected results or statuses,
// applies green/red fill styles, and saves to outputPath.
func (w *Writer) closeXLSX() error {
	f := excelize.NewFile()
	defer func() {
		_ = f.Close()
	}()

	// Define cell styles.
	greenStyle, _ := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#C8E6C9"}, Pattern: 1},
		Font: &excelize.Font{Bold: true, Color: "#2E7D32"},
	})
	redStyle, _ := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#FFCDD2"}, Pattern: 1},
		Font: &excelize.Font{Bold: true, Color: "#C62828"},
	})
	orangeStyle, _ := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#FFE0B2"}, Pattern: 1},
		Font: &excelize.Font{Bold: true, Color: "#E65100"},
	})
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#37474F"}, Pattern: 1},
		Font: &excelize.Font{Bold: true, Color: "#ECEFF1"},
	})

	sheet := "Sheet1"

	if len(w.results) > 0 {
		// Header row.
		_ = f.SetCellValue(sheet, "A1", "Domain")
		_ = f.SetCellValue(sheet, "B1", "Status")
		_ = f.SetCellValue(sheet, "C1", "Server")
		_ = f.SetCellStyle(sheet, "A1", "C1", headerStyle)

		for i, r := range w.results {
			row := i + 2
			_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), r.Domain)
			_ = f.SetCellValue(sheet, fmt.Sprintf("C%d", row), r.Server)

			statusCell := fmt.Sprintf("B%d", row)
			if r.Error != nil {
				_ = f.SetCellValue(sheet, statusCell, fmt.Sprintf("error: %v", r.Error))
				_ = f.SetCellStyle(sheet, statusCell, statusCell, orangeStyle)
			} else if r.Blocked {
				_ = f.SetCellValue(sheet, statusCell, "BLOCKED")
				_ = f.SetCellStyle(sheet, statusCell, statusCell, redStyle)
			} else {
				_ = f.SetCellValue(sheet, statusCell, "NOT BLOCKED")
				_ = f.SetCellStyle(sheet, statusCell, statusCell, greenStyle)
			}
		}

		// Auto-fit column widths.
		_ = f.SetColWidth(sheet, "A", "A", 40)
		_ = f.SetColWidth(sheet, "B", "B", 18)
		_ = f.SetColWidth(sheet, "C", "C", 22)
	}

	if len(w.statuses) > 0 {
		// Header row.
		_ = f.SetCellValue(sheet, "A1", "Server")
		_ = f.SetCellValue(sheet, "B1", "Status")
		_ = f.SetCellValue(sheet, "C1", "Latency / Error")
		_ = f.SetCellStyle(sheet, "A1", "C1", headerStyle)

		for i, s := range w.statuses {
			row := i + 2
			_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), s.Server)

			statusCell := fmt.Sprintf("B%d", row)
			if s.Online {
				_ = f.SetCellValue(sheet, statusCell, "ONLINE")
				_ = f.SetCellStyle(sheet, statusCell, statusCell, greenStyle)
				_ = f.SetCellValue(sheet, fmt.Sprintf("C%d", row), fmt.Sprintf("%dms", s.LatencyMs))
			} else {
				_ = f.SetCellValue(sheet, statusCell, "OFFLINE")
				_ = f.SetCellStyle(sheet, statusCell, statusCell, redStyle)
				errMsg := ""
				if s.Error != nil {
					errMsg = s.Error.Error()
				}
				_ = f.SetCellValue(sheet, fmt.Sprintf("C%d", row), errMsg)
			}
		}

		_ = f.SetColWidth(sheet, "A", "A", 22)
		_ = f.SetColWidth(sheet, "B", "B", 12)
		_ = f.SetColWidth(sheet, "C", "C", 30)
	}

	// Save to the output path. If outputPath is empty, write to stdout.
	if w.outputPath != "" {
		return f.SaveAs(w.outputPath)
	}
	_, err := f.WriteTo(w.w)
	return err
}

// Close flushes any buffered data, writes JSON caps, and closes the file.
func (w *Writer) Close() error {
	switch w.format {
	case FormatJSON:
		if w.jsonStarted {
			_, _ = w.w.WriteString("]}}\n")
			w.jsonStarted = false
		} else if w.w != nil {
			// No results/statuses were written. Emit a valid empty envelope
			// so consumers always receive well-formed JSON.
			key := w.jsonKey
			if key == "" {
				key = "result" // default envelope for check output
			}
			_, _ = fmt.Fprintf(w.w, "{\"nawala\":{\"%s\":[]}}\n", key)
		}
	case FormatHTML:
		if err := w.closeHTML(); err != nil {
			return err
		}
	case FormatXLSX:
		if err := w.closeXLSX(); err != nil {
			return err
		}
		// XLSX writes directly via SaveAs/WriteTo; skip bufio/tabwriter flush.
		return nil
	}

	// Flush the tabwriter first (computes column widths and writes to w.w),
	// then flush the underlying bufio.Writer to the destination.
	if w.tw != nil {
		_ = w.tw.Flush()
	}

	if err := w.w.Flush(); err != nil {
		return err
	}
	if w.closer != nil {
		return w.closer.Close()
	}
	return nil
}
