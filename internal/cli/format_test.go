// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package cli

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().StringSlice("format", []string{"text"}, "")
	return cmd
}

func TestResolveFormat_DefaultText(t *testing.T) {
	cmd := newTestCmd()
	f, err := resolveFormat(cmd)
	require.NoError(t, err)
	assert.Equal(t, FormatText, f)
}

func TestResolveFormat_JSON(t *testing.T) {
	cmd := newTestCmd()
	_ = cmd.Flags().Set("format", "json")
	f, err := resolveFormat(cmd)
	require.NoError(t, err)
	assert.Equal(t, FormatJSON, f)
}

func TestResolveFormat_HTML(t *testing.T) {
	cmd := newTestCmd()
	_ = cmd.Flags().Set("format", "html")
	f, err := resolveFormat(cmd)
	require.NoError(t, err)
	assert.Equal(t, FormatHTML, f)
}

func TestResolveFormat_XLSX(t *testing.T) {
	cmd := newTestCmd()
	_ = cmd.Flags().Set("format", "xlsx")
	f, err := resolveFormat(cmd)
	require.NoError(t, err)
	assert.Equal(t, FormatXLSX, f)
}

func TestResolveFormat_Invalid(t *testing.T) {
	cmd := newTestCmd()
	_ = cmd.Flags().Set("format", "invalid")
	_, err := resolveFormat(cmd)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid output format")
}

func TestResolveFormat_MultipleFlags(t *testing.T) {
	cmd := newTestCmd()
	// Using StringSlice, setting multiple separate flags
	_ = cmd.Flags().Set("format", "json")
	_ = cmd.Flags().Set("format", "html")
	_, err := resolveFormat(cmd)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "only one output format flag may be used")
}

func TestResolveFormat_CommaSeparated(t *testing.T) {
	cmd := newTestCmd()
	// This simulates a string containing a comma that bypassed len > 1 check
	cmd.Flags().Set("format", "json,html")
	formats, _ := cmd.Flags().GetStringSlice("format")
	t.Logf("CommaSeparated formats count: %d, items: %v", len(formats), formats)
	_, err := resolveFormat(cmd)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "only one output format flag may be used")
}

func TestResolveFormat_Empty(t *testing.T) {
	cmd := newTestCmd()
	cmd.Flags().StringSlice("format2", []string{}, "")
	// Hack to force len(formats) == 0: Just pass a cmd with an empty slice default?
	// Wait, StringSlice flag default is ["text"]. If we just clear it?
	_ = cmd.Flags().Set("format", "") // This might result in [""]?
	formats, _ := cmd.Flags().GetStringSlice("format")
	t.Logf("Empty formats count: %d, items: %v", len(formats), formats)
	f, _ := resolveFormat(cmd)
	t.Logf("Empty format result: %v", f)
}

func TestResolveFormat_TypeMismatch(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("format", "text", "") // Defined as String, not StringSlice
	_, err := resolveFormat(cmd)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "trying to get stringSlice value of flag of type string")
}

func TestResolveFormat_EscapedComma(t *testing.T) {
	cmd := newTestCmd()
	// pflag parses StringSlice as CSV. To get a single item with a comma, we quote it.
	_ = cmd.Flags().Set("format", `"json,html"`)
	formats, _ := cmd.Flags().GetStringSlice("format")
	t.Logf("EscapedComma formats count: %d, items: %v", len(formats), formats)
	_, err := resolveFormat(cmd)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "only one output format flag may be used")
}
