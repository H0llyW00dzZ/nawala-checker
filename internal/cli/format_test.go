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
	cmd.Flags().Bool("json", false, "")
	cmd.Flags().Bool("html", false, "")
	cmd.Flags().Bool("xlsx", false, "")
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
	cmd.Flags().Set("json", "true")
	f, err := resolveFormat(cmd)
	require.NoError(t, err)
	assert.Equal(t, FormatJSON, f)
}

func TestResolveFormat_HTML(t *testing.T) {
	cmd := newTestCmd()
	cmd.Flags().Set("html", "true")
	f, err := resolveFormat(cmd)
	require.NoError(t, err)
	assert.Equal(t, FormatHTML, f)
}

func TestResolveFormat_XLSX(t *testing.T) {
	cmd := newTestCmd()
	cmd.Flags().Set("xlsx", "true")
	f, err := resolveFormat(cmd)
	require.NoError(t, err)
	assert.Equal(t, FormatXLSX, f)
}

func TestResolveFormat_MultipleFlags_Error(t *testing.T) {
	cmd := newTestCmd()
	cmd.Flags().Set("json", "true")
	cmd.Flags().Set("html", "true")
	_, err := resolveFormat(cmd)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "only one output format flag")
}

func TestResolveFormat_AllThreeFlags_Error(t *testing.T) {
	cmd := newTestCmd()
	cmd.Flags().Set("json", "true")
	cmd.Flags().Set("html", "true")
	cmd.Flags().Set("xlsx", "true")
	_, err := resolveFormat(cmd)
	assert.Error(t, err)
}
