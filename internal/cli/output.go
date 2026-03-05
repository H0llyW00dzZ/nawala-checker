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
	closer io.Closer // non-nil when writing to a file

	format      string // "text", "json", "html", "xlsx"
	jsonStarted bool   // tracks if we started the JSON array
	jsonKey     string // "result" or "status" — set by the first JSON write
	outputPath  string // original path (needed by XLSX SaveAs)

	// Buffered results/statuses for text, HTML and XLSX (rendered at Close time).
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

	return w, nil
}

// jsonResult is the JSON representation of a check result.
// Blocked is omitted when Error is set — a failed check has no meaningful block
// state, and emitting false would be misleading to callers.
type jsonResult struct {
	Domain  string `json:"domain"`
	Blocked *bool  `json:"blocked,omitempty"`
	Server  string `json:"server"`
	Error   string `json:"error,omitempty"`
}

// WriteResult formats and writes a single check result.
// For JSON output the result is streamed immediately; all other formats
// buffer results in w.results and render them together at Close time.
func (w *Writer) WriteResult(r nawala.Result) {
	if w.format == FormatJSON {
		w.writeJSON(r)
		return
	}
	w.results = append(w.results, r)
}

// writeJSON writes a check result as a JSON array element.
func (w *Writer) writeJSON(r nawala.Result) {
	jr := jsonResult{
		Domain: r.Domain,
		Server: r.Server,
	}
	if r.Error != nil {
		jr.Error = r.Error.Error()
	} else {
		// Only emit blocked when the check succeeded — a failed check has no
		// meaningful block state.
		jr.Blocked = &r.Blocked
	}

	data, _ := json.Marshal(jr)
	if !w.jsonStarted {
		_, _ = fmt.Fprintf(w.w, `{"nawala":{"version":%q,"result":[`, nawala.Version)
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
// For JSON output the status is streamed immediately; all other formats
// buffer statuses in w.statuses and render them together at Close time.
func (w *Writer) WriteStatus(s nawala.ServerStatus) {
	if w.format == FormatJSON {
		w.writeStatusJSON(s)
		return
	}
	w.statuses = append(w.statuses, s)
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
		_, _ = fmt.Fprintf(w.w, `{"nawala":{"version":%q,"status":[`, nawala.Version)
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
	Version string
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
	Version  string
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
		data := htmlResultData{
			Version: nawala.Version,
			Results: make([]htmlResult, len(w.results)),
		}
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
		data := htmlStatusData{
			Version:  nawala.Version,
			Statuses: make([]htmlStatus, len(w.statuses)),
		}
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

// xlsxColWidth converts a character count to an excelize [excelize.File.SetColWidth]
// value. It adds a small padding so text never hugs the cell border, and
// enforces a minimum so narrow columns (e.g. "Status") still look readable.
func xlsxColWidth(charCount int) float64 {
	const padding = 2
	const minWidth = 8
	return max(float64(charCount+padding), minWidth)
}

// xlsxStyles groups the pre-made excelize style IDs used when writing data rows.
type xlsxStyles struct {
	green, red, orange, cell, header, title int
}

// xlsxBorder returns the shared thin blue-grey border slice.
func xlsxBorder() []excelize.Border {
	return []excelize.Border{
		{Type: "left", Color: "#B0BEC5", Style: 1},
		{Type: "right", Color: "#B0BEC5", Style: 1},
		{Type: "top", Color: "#B0BEC5", Style: 1},
		{Type: "bottom", Color: "#B0BEC5", Style: 1},
	}
}

// xlsxNewStyles creates all named cell styles and returns them as xlsxStyles.
func xlsxNewStyles(f *excelize.File, border []excelize.Border) xlsxStyles {
	new := func(s *excelize.Style) int { id, _ := f.NewStyle(s); return id }
	return xlsxStyles{
		green: new(&excelize.Style{
			Fill:      excelize.Fill{Type: "pattern", Color: []string{"#C8E6C9"}, Pattern: 1},
			Font:      &excelize.Font{Bold: true, Color: "#2E7D32"},
			Alignment: &excelize.Alignment{Horizontal: "center"},
			Border:    border,
		}),
		red: new(&excelize.Style{
			Fill:      excelize.Fill{Type: "pattern", Color: []string{"#FFCDD2"}, Pattern: 1},
			Font:      &excelize.Font{Bold: true, Color: "#C62828"},
			Alignment: &excelize.Alignment{Horizontal: "center"},
			Border:    border,
		}),
		orange: new(&excelize.Style{
			Fill:      excelize.Fill{Type: "pattern", Color: []string{"#FFE0B2"}, Pattern: 1},
			Font:      &excelize.Font{Bold: true, Color: "#E65100"},
			Alignment: &excelize.Alignment{Horizontal: "center"},
			Border:    border,
		}),
		cell: new(&excelize.Style{Border: border}),
		header: new(&excelize.Style{
			Fill:      excelize.Fill{Type: "pattern", Color: []string{"#37474F"}, Pattern: 1},
			Font:      &excelize.Font{Bold: true, Color: "#ECEFF1", Size: 11},
			Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
			Border:    border,
		}),
		title: new(&excelize.Style{
			Fill:      excelize.Fill{Type: "pattern", Color: []string{"#00ACD7"}, Pattern: 1},
			Font:      &excelize.Font{Bold: true, Color: "#FFFFFF", Size: 12},
			Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		}),
	}
}

// xlsxTitleAndHeader writes the merged version title row (row 1) and the
// column-header row (row 2) with the supplied labels into sheet.
func xlsxTitleAndHeader(f *excelize.File, sheet string, st xlsxStyles, colA, colB, colC string) {
	title := "Nawala Checker v" + nawala.Version
	_ = f.SetCellValue(sheet, "A1", title)
	_ = f.MergeCell(sheet, "A1", "C1")
	_ = f.SetCellStyle(sheet, "A1", "C1", st.title)
	_ = f.SetRowHeight(sheet, 1, 22)
	_ = f.SetCellValue(sheet, "A2", colA)
	_ = f.SetCellValue(sheet, "B2", colB)
	_ = f.SetCellValue(sheet, "C2", colC)
	_ = f.SetCellStyle(sheet, "A2", "C2", st.header)
	_ = f.SetRowHeight(sheet, 2, 20)
}

// xlsxSetResultCell writes the status value and style for one result row into
// the B column of sheet. It returns the status string for width tracking.
func xlsxSetResultCell(f *excelize.File, sheet, cell string, r nawala.Result, st xlsxStyles) string {
	var text string
	var style int
	switch {
	case r.Error != nil:
		text, style = fmt.Sprintf("error: %v", r.Error), st.orange
	case r.Blocked:
		text, style = "BLOCKED", st.red
	default:
		text, style = "NOT BLOCKED", st.green
	}
	_ = f.SetCellValue(sheet, cell, text)
	_ = f.SetCellStyle(sheet, cell, cell, style)
	return text
}

// xlsxResultRows fills data rows (starting at row 3) for the check-results
// sheet and returns the max column widths for auto-fitting.
func xlsxResultRows(f *excelize.File, sheet string, results []nawala.Result, st xlsxStyles) (maxA, maxB, maxC int) {
	maxA, maxB, maxC = len("Domain"), len("Status"), len("Server")
	for i, r := range results {
		row := i + 3
		aCell, cCell := fmt.Sprintf("A%d", row), fmt.Sprintf("C%d", row)
		_ = f.SetCellValue(sheet, aCell, r.Domain)
		_ = f.SetCellStyle(sheet, aCell, aCell, st.cell)
		_ = f.SetCellValue(sheet, cCell, r.Server)
		_ = f.SetCellStyle(sheet, cCell, cCell, st.cell)
		if n := len(r.Domain); n > maxA {
			maxA = n
		}
		if n := len(r.Server); n > maxC {
			maxC = n
		}
		text := xlsxSetResultCell(f, sheet, fmt.Sprintf("B%d", row), r, st)
		if n := len(text); n > maxB {
			maxB = n
		}
	}
	return
}

// xlsxStatusRows fills data rows (starting at row 3) for the server-status
// sheet and returns the max column widths for auto-fitting.
func xlsxStatusRows(f *excelize.File, sheet string, statuses []nawala.ServerStatus, st xlsxStyles) (maxA, maxB, maxC int) {
	maxA, maxB, maxC = len("Server"), len("Status"), len("Latency / Error")
	for i, s := range statuses {
		row := i + 3
		aCell, bCell, cCell := fmt.Sprintf("A%d", row), fmt.Sprintf("B%d", row), fmt.Sprintf("C%d", row)
		_ = f.SetCellValue(sheet, aCell, s.Server)
		_ = f.SetCellStyle(sheet, aCell, aCell, st.cell)
		if n := len(s.Server); n > maxA {
			maxA = n
		}
		var label, colC string
		if s.Online {
			label = "ONLINE"
			_ = f.SetCellValue(sheet, bCell, label)
			_ = f.SetCellStyle(sheet, bCell, bCell, st.green)
			colC = fmt.Sprintf("%dms", s.LatencyMs)
		} else {
			label = "OFFLINE"
			_ = f.SetCellValue(sheet, bCell, label)
			_ = f.SetCellStyle(sheet, bCell, bCell, st.red)
			if s.Error != nil {
				colC = s.Error.Error()
			}
		}
		_ = f.SetCellValue(sheet, cCell, colC)
		_ = f.SetCellStyle(sheet, cCell, cCell, st.cell)
		if n := len(label); n > maxB {
			maxB = n
		}
		if n := len(colC); n > maxC {
			maxC = n
		}
	}
	return
}

// closeXLSX builds an Excel workbook from collected results or statuses,
// applies styled fills, and saves to outputPath.
func (w *Writer) closeXLSX() error {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	st := xlsxNewStyles(f, xlsxBorder())
	sheet := "Sheet1"

	if len(w.results) > 0 {
		xlsxTitleAndHeader(f, sheet, st, "Domain", "Status", "Server")
		maxA, maxB, maxC := xlsxResultRows(f, sheet, w.results, st)
		_ = f.SetColWidth(sheet, "A", "A", xlsxColWidth(maxA))
		_ = f.SetColWidth(sheet, "B", "B", xlsxColWidth(maxB))
		_ = f.SetColWidth(sheet, "C", "C", xlsxColWidth(maxC))
	}
	if len(w.statuses) > 0 {
		xlsxTitleAndHeader(f, sheet, st, "Server", "Status", "Latency / Error")
		maxA, maxB, maxC := xlsxStatusRows(f, sheet, w.statuses, st)
		_ = f.SetColWidth(sheet, "A", "A", xlsxColWidth(maxA))
		_ = f.SetColWidth(sheet, "B", "B", xlsxColWidth(maxB))
		_ = f.SetColWidth(sheet, "C", "C", xlsxColWidth(maxC))
	}

	// Save to the output path. If outputPath is empty, write to stdout.
	if w.outputPath != "" {
		return f.SaveAs(w.outputPath)
	}
	_, err := f.WriteTo(w.w)
	return err
}

// textBanner returns a centered ─-ruler banner of the given total width.
func textBanner(totalW int) string {
	title := " Nawala Checker v" + nawala.Version + " "
	fill := max(0, totalW-len(title))
	return strings.Repeat("─", fill/2) + title + strings.Repeat("─", fill-fill/2)
}

// textCenterPad returns s padded with spaces to center it in a field of
// width w.
func textCenterPad(s string, w int) string {
	sp := w - len(s)
	return strings.Repeat(" ", sp/2) + s + strings.Repeat(" ", sp-sp/2)
}

// writeTextResults renders w.results with a centered status column.
func (w *Writer) writeTextResults() {
	const pad = "    "
	type row struct{ domain, status, server string }
	rows := make([]row, len(w.results))
	maxD, maxS := 0, 0
	for i, r := range w.results {
		var s string
		switch {
		case r.Error != nil:
			s = fmt.Sprintf("error: %v", r.Error)
		case r.Blocked:
			s = "BLOCKED"
		default:
			s = "NOT BLOCKED"
		}
		rows[i] = row{r.Domain, s, r.Server}
		if l := len(r.Domain); l > maxD {
			maxD = l
		}
		if l := len(s); l > maxS {
			maxS = l
		}
	}
	totalW := maxD + len(pad) + maxS + len(pad) + len(rows[0].server)
	_, _ = fmt.Fprintf(w.w, "%s\n\n", textBanner(totalW))
	for _, r := range rows {
		_, _ = fmt.Fprintf(w.w, "%-*s%s%s%s%s\n",
			maxD, r.domain, pad, textCenterPad(r.status, maxS), pad, r.server)
	}
}

// writeTextStatuses renders w.statuses with a centered status column.
func (w *Writer) writeTextStatuses() {
	const pad = "    "
	type row struct{ server, status, info string }
	rows := make([]row, len(w.statuses))
	maxSv, maxSt, maxI := 0, 0, 0
	for i, s := range w.statuses {
		st := "ONLINE"
		if !s.Online {
			st = "OFFLINE"
		}
		var info string
		if s.Online {
			info = fmt.Sprintf("%dms", s.LatencyMs)
		} else if s.Error != nil {
			info = strings.TrimPrefix(s.Error.Error(), "nawala: ")
		}
		rows[i] = row{s.Server, st, info}
		if l := len(s.Server); l > maxSv {
			maxSv = l
		}
		if l := len(st); l > maxSt {
			maxSt = l
		}
		if l := len(info); l > maxI {
			maxI = l
		}
	}
	totalW := maxSv + len(pad) + maxSt + len(pad) + maxI
	_, _ = fmt.Fprintf(w.w, "%s\n\n", textBanner(totalW))
	for _, r := range rows {
		_, _ = fmt.Fprintf(w.w, "%-*s%s%s%s%s\n",
			maxSv, r.server, pad, textCenterPad(r.status, maxSt), pad, r.info)
	}
}

// Close flushes any buffered data, writes JSON caps, and closes the file.
//
// Note: the output is more creative now for excel, html, and text
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
			_, _ = fmt.Fprintf(w.w, `{"nawala":{"version":%q,"%s":[]}}`, nawala.Version, key)
			_, _ = w.w.WriteString("\n")
		}
	case FormatText:
		if len(w.results) > 0 {
			w.writeTextResults()
		}
		if len(w.statuses) > 0 {
			w.writeTextStatuses()
		}
	case FormatHTML:
		if err := w.closeHTML(); err != nil {
			return err
		}
	case FormatXLSX:
		if err := w.closeXLSX(); err != nil {
			return err
		}
		// XLSX writes directly via SaveAs/WriteTo; skip bufio flush.
		return nil
	}

	if err := w.w.Flush(); err != nil {
		return err
	}
	if w.closer != nil {
		return w.closer.Close()
	}
	return nil
}
