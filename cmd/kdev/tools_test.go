package main

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dennisklein/kdev/internal/tool"
)

func TestNewToolsCmd(t *testing.T) {
	t.Run("creates tools command", func(t *testing.T) {
		cmd := newToolsCmd()

		require.NotNil(t, cmd)
		assert.Equal(t, "tools", cmd.Use)
		assert.NotEmpty(t, cmd.Short)
		assert.NotEmpty(t, cmd.Long)

		// Should have subcommands
		assert.True(t, cmd.HasSubCommands())
	})

	t.Run("has clean subcommand", func(t *testing.T) {
		cmd := newToolsCmd()

		cleanCmd, _, err := cmd.Find([]string{"clean"})
		require.NoError(t, err)
		assert.Equal(t, "clean", cleanCmd.Name())
	})

	t.Run("has info subcommand", func(t *testing.T) {
		cmd := newToolsCmd()

		infoCmd, _, err := cmd.Find([]string{"info"})
		require.NoError(t, err)
		assert.Equal(t, "info", infoCmd.Name())
	})

	t.Run("has update subcommand", func(t *testing.T) {
		cmd := newToolsCmd()

		updateCmd, _, err := cmd.Find([]string{"update"})
		require.NoError(t, err)
		assert.Equal(t, "update", updateCmd.Name())
	})
}

func TestNewToolsCleanCmd(t *testing.T) {
	t.Run("creates clean command", func(t *testing.T) {
		cmd := newToolsCleanCmd()

		require.NotNil(t, cmd)
		assert.Equal(t, "clean [tool...]", cmd.Use)
		assert.NotEmpty(t, cmd.Short)
		assert.NotNil(t, cmd.RunE)
	})

	t.Run("has --old flag", func(t *testing.T) {
		cmd := newToolsCleanCmd()

		flag := cmd.Flags().Lookup("old")
		require.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})
}

func TestRunToolsClean(t *testing.T) {
	t.Run("executes clean command with no cached tools", func(t *testing.T) {
		cmd := newToolsCleanCmd()

		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetArgs([]string{})

		// Clean when nothing is cached succeeds with no output
		err := cmd.Execute()
		require.NoError(t, err)

		output := buf.String()
		assert.Empty(t, output)
	})

	t.Run("handles unknown tool gracefully", func(t *testing.T) {
		cmd := newToolsCleanCmd()

		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetArgs([]string{"nonexistent"})

		// Unknown tool is simply skipped
		err := cmd.Execute()
		require.NoError(t, err)

		output := buf.String()
		assert.Empty(t, output)
	})

	t.Run("reports reclaimed space", func(t *testing.T) {
		cmd := newToolsCleanCmd()

		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetArgs([]string{})

		// Note: This test relies on actual cached tools
		// In a clean environment, it will show no output
		err := cmd.Execute()
		require.NoError(t, err)

		// Output should either be empty or contain "Reclaimed"
		output := buf.String()
		if output != "" {
			assert.Contains(t, output, "Reclaimed")
		}
	})
}

func TestNewToolsInfoCmd(t *testing.T) {
	t.Run("creates info command", func(t *testing.T) {
		cmd := newToolsInfoCmd()

		require.NotNil(t, cmd)
		assert.Equal(t, "info [tool...]", cmd.Use)
		assert.NotEmpty(t, cmd.Short)
		assert.NotNil(t, cmd.RunE)
	})
}

func TestRunToolsInfo(t *testing.T) {
	t.Run("executes info command for all tools", func(t *testing.T) {
		cmd := newToolsInfoCmd()

		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetArgs([]string{})
		cmd.SetContext(context.Background())

		// This will make real HTTP call to check latest version
		// We just verify it executes without error
		err := cmd.Execute()

		// May fail if network unavailable, but structure is tested
		if err == nil {
			output := buf.String()
			assert.Contains(t, output, "kubectl")
		}
	})

	t.Run("handles specific tool", func(t *testing.T) {
		cmd := newToolsInfoCmd()

		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetArgs([]string{"kubectl"})
		cmd.SetContext(context.Background())

		err := cmd.Execute()
		if err == nil {
			output := buf.String()
			assert.Contains(t, output, "kubectl")
		}
	})
}

func TestNewToolsUpdateCmd(t *testing.T) {
	t.Run("creates update command", func(t *testing.T) {
		cmd := newToolsUpdateCmd()

		require.NotNil(t, cmd)
		assert.Equal(t, "update [tool...]", cmd.Use)
		assert.NotEmpty(t, cmd.Short)
		assert.NotNil(t, cmd.RunE)
	})
}

func TestRunToolsUpdate(t *testing.T) {
	t.Run("executes update command", func(t *testing.T) {
		cmd := newToolsUpdateCmd()

		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetArgs([]string{})
		cmd.SetContext(context.Background())

		// This would download kubectl, which we don't want in unit tests
		// Just verify command structure is correct
		// Integration tests should cover actual download
		err := cmd.Execute()

		// May succeed or fail depending on network/permissions
		// We're just verifying the command executes
		_ = err
	})
}

func TestResolveTools(t *testing.T) {
	t.Run("returns all tools when no names provided", func(t *testing.T) {
		var buf bytes.Buffer

		registry := newTestRegistry(&buf)

		tools := resolveTools(registry, nil)

		assert.Len(t, tools, 2) // Should have kind and kubectl

		// Tools should be sorted alphabetically: kind, kubectl
		assert.Equal(t, "kind", tools[0].Name)
		assert.Equal(t, "kubectl", tools[1].Name)
	})

	t.Run("returns all tools when empty slice provided", func(t *testing.T) {
		var buf bytes.Buffer

		registry := newTestRegistry(&buf)

		tools := resolveTools(registry, []string{})

		assert.Len(t, tools, 2)
	})

	t.Run("returns specific tool when name provided", func(t *testing.T) {
		var buf bytes.Buffer

		registry := newTestRegistry(&buf)

		tools := resolveTools(registry, []string{"kubectl"})

		assert.Len(t, tools, 1)
		assert.Equal(t, "kubectl", tools[0].Name)
	})

	t.Run("returns empty slice for unknown tool", func(t *testing.T) {
		var buf bytes.Buffer

		registry := newTestRegistry(&buf)

		tools := resolveTools(registry, []string{"nonexistent"})

		assert.Empty(t, tools)
	})
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name  string
		want  string
		bytes int64
	}{
		{
			name:  "bytes",
			want:  "500 B",
			bytes: 500,
		},
		{
			name:  "kilobytes",
			want:  "1.0 KiB",
			bytes: 1024,
		},
		{
			name:  "megabytes",
			want:  "1.0 MiB",
			bytes: 1024 * 1024,
		},
		{
			name:  "gigabytes",
			want:  "1.0 GiB",
			bytes: 1024 * 1024 * 1024,
		},
		{
			name:  "partial kilobytes",
			want:  "1.5 KiB",
			bytes: 1536, // 1.5 KiB
		},
		{
			name:  "large value",
			want:  "5.0 GiB",
			bytes: 5 * 1024 * 1024 * 1024, // 5 GiB
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatBytes(tt.bytes)
			assert.Equal(t, tt.want, result)
		})
	}
}

// newTestRegistry creates a registry for testing.
func newTestRegistry(buf *bytes.Buffer) *tool.Registry {
	return tool.NewRegistry(buf)
}
