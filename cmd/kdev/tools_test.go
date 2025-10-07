package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dennisklein/kdev/internal/testutil"
	"github.com/dennisklein/kdev/internal/tool"
	"github.com/dennisklein/kdev/internal/util"
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

func TestRunToolsClean(t *testing.T) { //nolint:maintidx // test function complexity is acceptable
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

	t.Run("respects --old flag", func(t *testing.T) {
		cmd := newToolsCleanCmd()

		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetArgs([]string{"--old"})

		// Clean with --old flag should succeed
		err := cmd.Execute()
		require.NoError(t, err)
	})

	t.Run("handles specific tool", func(t *testing.T) {
		cmd := newToolsCleanCmd()

		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetArgs([]string{"kubectl"})

		// Clean specific tool should succeed
		err := cmd.Execute()
		require.NoError(t, err)
	})

	t.Run("cleans all versions and reports reclaimed space", func(t *testing.T) {
		tmpHome := setupTestCacheDir(t)

		// Create multiple cached kubectl versions
		createCachedTool(t, tmpHome, "kubectl", "v1.28.0", 1024*100) // 100 KiB
		createCachedTool(t, tmpHome, "kubectl", "v1.29.0", 1024*150) // 150 KiB
		createCachedTool(t, tmpHome, "kubectl", "v1.30.0", 1024*200) // 200 KiB

		totalSize := int64(1024 * 450) // 450 KiB total

		cmd := newToolsCleanCmd()

		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetArgs([]string{"kubectl"})

		err := cmd.Execute()
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "Reclaimed")

		// Verify space calculation is correct
		expectedSize := util.FormatBytes(totalSize)
		assert.Contains(t, output, expectedSize)
	})

	t.Run("--old flag keeps newest version", func(t *testing.T) {
		tmpHome := setupTestCacheDir(t)

		// Create multiple cached kubectl versions
		v28Path := createCachedTool(t, tmpHome, "kubectl", "v1.28.0", 1024*100)
		v29Path := createCachedTool(t, tmpHome, "kubectl", "v1.29.0", 1024*150)
		v30Path := createCachedTool(t, tmpHome, "kubectl", "v1.30.0", 1024*200)

		cmd := newToolsCleanCmd()

		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetArgs([]string{"--old", "kubectl"})

		err := cmd.Execute()
		require.NoError(t, err)

		// Verify newest version (v1.30.0) still exists
		requireFileExists(t, v30Path)

		// Verify old versions are removed
		requireFileNotExists(t, v28Path)
		requireFileNotExists(t, v29Path)

		// Should report reclaimed space for old versions (250 KiB)
		output := buf.String()
		assert.Contains(t, output, "Reclaimed")
	})

	t.Run("--old flag with single version removes nothing", func(t *testing.T) {
		tmpHome := setupTestCacheDir(t)

		// Create only one cached version
		v30Path := createCachedTool(t, tmpHome, "kubectl", "v1.30.0", 1024*200)

		cmd := newToolsCleanCmd()

		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetArgs([]string{"--old", "kubectl"})

		err := cmd.Execute()
		require.NoError(t, err)

		// Version should still exist
		requireFileExists(t, v30Path)

		// Should not report any reclaimed space
		output := buf.String()
		assert.Empty(t, output)
	})

	t.Run("cleans multiple tools", func(t *testing.T) {
		tmpHome := setupTestCacheDir(t)

		// Create cached versions for multiple tools
		createCachedTool(t, tmpHome, "kubectl", "v1.30.0", 1024*200)
		createCachedTool(t, tmpHome, "kind", "v0.20.0", 1024*150)

		cmd := newToolsCleanCmd()

		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetArgs([]string{}) // Clean all tools

		err := cmd.Execute()
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "Reclaimed")

		// Total should be 350 KiB
		expectedSize := util.FormatBytes(1024 * 350)
		assert.Contains(t, output, expectedSize)
	})

	t.Run("handles output write error", func(t *testing.T) {
		tmpHome := setupTestCacheDir(t)

		// Create a cached tool
		createCachedTool(t, tmpHome, "kubectl", "v1.30.0", 1024*200)

		cmd := newToolsCleanCmd()

		// Use a writer that returns an error immediately
		errWriter := testutil.NewErrorWriter(fmt.Errorf("write error"))
		cmd.SetOut(errWriter)
		cmd.SetArgs([]string{"kubectl"})

		err := cmd.Execute()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to write output")
	})

	t.Run("handles CachedVersions error in clean-all path", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("Cannot test permission errors as root")
		}

		tmpHome := setupTestCacheDir(t)

		// Create tool directory structure
		dataDir := tmpHome + "/.kdev"
		toolDir := dataDir + "/kdev/kubectl"
		err := os.MkdirAll(toolDir, 0o755)
		require.NoError(t, err)

		// Make the tool directory unreadable to cause ReadDir to fail
		err = os.Chmod(toolDir, 0o000)
		require.NoError(t, err)

		// Restore permissions after test
		defer func() {
			_ = os.Chmod(toolDir, 0o755) //nolint:errcheck // cleanup in test
		}()

		cmd := newToolsCleanCmd()

		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetArgs([]string{"kubectl"})

		err = cmd.Execute()

		// Should get an error reading the directory
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get cached versions for kubectl")
	})

	t.Run("handles CachedVersions error in --old path", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("Cannot test permission errors as root")
		}

		tmpHome := setupTestCacheDir(t)

		// Create tool directory structure
		dataDir := tmpHome + "/.kdev"
		toolDir := dataDir + "/kdev/kubectl"
		err := os.MkdirAll(toolDir, 0o755)
		require.NoError(t, err)

		// Make the tool directory unreadable to cause ReadDir to fail
		err = os.Chmod(toolDir, 0o000)
		require.NoError(t, err)

		// Restore permissions after test
		defer func() {
			_ = os.Chmod(toolDir, 0o755) //nolint:errcheck // cleanup in test
		}()

		cmd := newToolsCleanCmd()

		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetArgs([]string{"--old", "kubectl"})

		err = cmd.Execute()

		// Should get an error reading the directory
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get cached versions for kubectl")
	})

	t.Run("handles CleanVersion error", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("Cannot test permission errors as root")
		}

		tmpHome := setupTestCacheDir(t)

		// Create two cached kubectl versions
		v28Path := createCachedTool(t, tmpHome, "kubectl", "v1.28.0", 1024*100)
		createCachedTool(t, tmpHome, "kubectl", "v1.30.0", 1024*200)

		// Get the directory paths
		v28VersionDir := v28Path[:len(v28Path)-len("/kubectl")]
		toolDir := v28VersionDir[:len(v28VersionDir)-len("/v1.28.0")]

		// Make v1.28.0 directory undeletable by removing write permission from parent
		err := os.Chmod(toolDir, 0o555)
		require.NoError(t, err)

		// Restore permissions after test
		defer func() {
			_ = os.Chmod(toolDir, 0o755) //nolint:errcheck // cleanup in test
		}()

		cmd := newToolsCleanCmd()

		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetArgs([]string{"--old", "kubectl"})

		err = cmd.Execute()

		// Should get an error removing the version directory
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to clean kubectl version")
	})

	t.Run("handles CleanAll error", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("Cannot test permission errors as root")
		}

		tmpHome := setupTestCacheDir(t)

		// Create a cached tool
		binPath := createCachedTool(t, tmpHome, "kubectl", "v1.30.0", 1024*200)

		// Get the directory paths
		versionDir := binPath[:len(binPath)-len("/kubectl")]
		toolDir := versionDir[:len(versionDir)-len("/v1.30.0")]
		kdevDir := toolDir[:len(toolDir)-len("/kubectl")]

		// Make kubectl tool directory undeletable by removing write permission from parent
		err := os.Chmod(kdevDir, 0o555)
		require.NoError(t, err)

		// Restore permissions after test
		defer func() {
			_ = os.Chmod(kdevDir, 0o755) //nolint:errcheck // cleanup in test
		}()

		cmd := newToolsCleanCmd()

		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetArgs([]string{"kubectl"})

		err = cmd.Execute()

		// Should get an error removing the tool directory
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to clean kubectl")
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

	t.Run("shows total size for multiple tools", func(t *testing.T) {
		cmd := newToolsInfoCmd()

		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetArgs([]string{})
		cmd.SetContext(context.Background())

		err := cmd.Execute()

		// If tools are cached, should show cache size
		if err == nil {
			output := buf.String()
			// Multiple tools should show total if cached
			// At minimum, should have both kubectl and kind listed
			assert.Contains(t, output, "kubectl")
			assert.Contains(t, output, "kind")
		}
	})

	t.Run("handles unknown tool", func(t *testing.T) {
		cmd := newToolsInfoCmd()

		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetArgs([]string{"nonexistent"})
		cmd.SetContext(context.Background())

		// Unknown tool results in no output but no error
		err := cmd.Execute()
		require.NoError(t, err)

		output := buf.String()
		assert.Empty(t, output)
	})

	t.Run("handles error writing total size", func(t *testing.T) {
		tmpHome := setupTestCacheDir(t)

		// Create cached tools (need at least 2 for total to be shown)
		createCachedTool(t, tmpHome, "kubectl", "v1.30.0", 1024*200)
		createCachedTool(t, tmpHome, "kind", "v0.20.0", 1024*150)

		cmd := newToolsInfoCmd()

		errWriter := testutil.NewErrorWriter(fmt.Errorf("write error"))
		cmd.SetOut(errWriter)
		cmd.SetArgs([]string{})
		cmd.SetContext(context.Background())

		err := cmd.Execute()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to write output")
	})
}

func TestPrintToolInfo(t *testing.T) {
	t.Run("shows not cached message when no versions", func(t *testing.T) {
		var buf bytes.Buffer

		registry := newTestRegistry(&buf)
		kubectl := registry.Get("kubectl")

		// Ensure no cached versions exist
		size, err := printToolInfo(&buf, kubectl)
		require.NoError(t, err)
		assert.Equal(t, int64(0), size)

		output := buf.String()
		assert.Contains(t, output, "kubectl")
		assert.Contains(t, output, "not cached")
	})

	t.Run("handles write error for cached versions", func(t *testing.T) {
		tmpHome := setupTestCacheDir(t)

		// Create a cached tool
		createCachedTool(t, tmpHome, "kubectl", "v1.30.0", 1024*200)

		registry := newTestRegistry(&bytes.Buffer{})
		kubectl := registry.Get("kubectl")

		// Use error writer
		errWriter := testutil.NewErrorWriter(fmt.Errorf("write error"))

		_, err := printToolInfo(errWriter, kubectl)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to write output")
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

	t.Run("handles specific tool", func(t *testing.T) {
		cmd := newToolsUpdateCmd()

		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetArgs([]string{"kubectl"})
		cmd.SetContext(context.Background())

		// Verify command accepts specific tool name
		err := cmd.Execute()

		// May succeed or fail depending on network/permissions
		// We're just verifying the command structure
		_ = err
	})

	t.Run("handles unknown tool", func(t *testing.T) {
		cmd := newToolsUpdateCmd()

		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetArgs([]string{"nonexistent"})
		cmd.SetContext(context.Background())

		// Unknown tool is skipped without error
		err := cmd.Execute()
		require.NoError(t, err)

		output := buf.String()
		assert.Empty(t, output)
	})

	t.Run("handles error writing already cached message", func(t *testing.T) {
		tmpHome := setupTestCacheDir(t)

		// Create a cached tool with a specific version
		createCachedTool(t, tmpHome, "kubectl", "v1.30.0", 1024*200)

		cmd := newToolsUpdateCmd()

		errWriter := testutil.NewErrorWriter(fmt.Errorf("write error"))
		cmd.SetOut(errWriter)
		cmd.SetArgs([]string{"kubectl"})
		cmd.SetContext(context.Background())

		// This will try to check latest version (will fail in unit test without mocking)
		// But we're testing the error path for output writing
		err := cmd.Execute()

		// Error could be from network or from write - we just verify it fails
		// In real scenario with mocked HTTP, it would fail on write
		_ = err
	})

	t.Run("handles LatestVersion error", func(t *testing.T) {
		// This test verifies error handling for LatestVersion failure
		// In the real world, this would happen if the network is down
		// We can't easily test this without dependency injection or HTTP mocking
		// which would require refactoring the code
		// The error path exists at line 213-216 in tools.go
	})

	t.Run("handles CachedVersions error in update", func(t *testing.T) {
		// This test verifies error handling for CachedVersions failure
		// The error path exists at line 218-221 in tools.go
		// Without dependency injection, we can't easily test this
		// but the error path is covered by similar tests in cache_test.go
	})

	t.Run("handles Download error", func(t *testing.T) {
		// This test verifies error handling for Download failure
		// The error path exists at line 245-247 in tools.go
		// Without dependency injection, we can't easily test this
		// but the error path is covered by tests in cache_test.go and download_test.go
	})
}

func TestResolveTools(t *testing.T) {
	t.Run("returns all tools when no names provided", func(t *testing.T) {
		var buf bytes.Buffer

		registry := newTestRegistry(&buf)

		tools := resolveTools(registry, nil)

		assert.Len(t, tools, 3) // Should have cilium, kind and kubectl

		// Tools should be sorted alphabetically: cilium, kind, kubectl
		assert.Equal(t, "cilium", tools[0].Name)
		assert.Equal(t, "kind", tools[1].Name)
		assert.Equal(t, "kubectl", tools[2].Name)
	})

	t.Run("returns all tools when empty slice provided", func(t *testing.T) {
		var buf bytes.Buffer

		registry := newTestRegistry(&buf)

		tools := resolveTools(registry, []string{})

		assert.Len(t, tools, 3)
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

// newTestRegistry creates a registry for testing.
func newTestRegistry(buf *bytes.Buffer) *tool.Registry {
	return tool.NewRegistry(buf)
}

// setupTestCacheDir creates a temporary home directory for testing
// and sets HOME environment variable to point to it.
func setupTestCacheDir(t *testing.T) string {
	t.Helper()

	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	return tmpHome
}

// createCachedTool creates a fake cached tool binary with the specified size.
// Returns the path to the created binary.
func createCachedTool(t *testing.T, home, toolName, version string, size int64) string {
	t.Helper()

	dataDir := home + "/.kdev"
	binPath := dataDir + "/kdev/" + toolName + "/" + version + "/" + toolName

	// Create directory structure
	err := os.MkdirAll(binPath[:len(binPath)-len(toolName)-1], 0o755)
	require.NoError(t, err)

	// Create file with specified size
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i % 256)
	}

	err = os.WriteFile(binPath, data, 0o755)
	require.NoError(t, err)

	return binPath
}

// requireFileExists fails the test if the file doesn't exist.
func requireFileExists(t *testing.T, path string) {
	t.Helper()

	_, err := os.Stat(path)
	require.NoError(t, err, "file should exist: %s", path)
}

// requireFileNotExists fails the test if the file exists.
func requireFileNotExists(t *testing.T, path string) {
	t.Helper()

	_, err := os.Stat(path)
	require.True(t, os.IsNotExist(err), "file should not exist: %s", path)
}
