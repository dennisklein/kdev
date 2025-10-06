//nolint:testpackage // internal functions require same package
package tool

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDataDir(t *testing.T) {
	t.Run("uses XDG_DATA_HOME when set", func(t *testing.T) {
		fs := afero.NewMemMapFs()

		t.Setenv("XDG_DATA_HOME", "/custom/data")

		dir, err := DataDir(fs)
		require.NoError(t, err)
		assert.Equal(t, "/custom/data", dir)
	})

	t.Run("uses ~/.local/share when it exists", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		home := "/home/testuser"

		// Clear XDG_DATA_HOME
		_ = os.Unsetenv("XDG_DATA_HOME") //nolint:errcheck // best effort

		// Mock home directory
		t.Setenv("HOME", home)

		// Create .local/share directory
		xdgDefault := filepath.Join(home, ".local", "share")
		err := fs.MkdirAll(xdgDefault, 0o755)
		require.NoError(t, err)

		dir, err := DataDir(fs)
		require.NoError(t, err)
		assert.Equal(t, xdgDefault, dir)
	})

	t.Run("falls back to ~/.kdev when .local/share does not exist", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		home := "/home/testuser"

		// Clear XDG_DATA_HOME
		_ = os.Unsetenv("XDG_DATA_HOME") //nolint:errcheck // best effort

		// Mock home directory
		t.Setenv("HOME", home)

		dir, err := DataDir(fs)
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(home, ".kdev"), dir)
	})
}

func TestExists(t *testing.T) {
	t.Run("returns true for existing file", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		path := "/test/file.txt"

		err := fs.MkdirAll("/test", 0o755)
		require.NoError(t, err)

		err = afero.WriteFile(fs, path, []byte("content"), 0o644)
		require.NoError(t, err)

		assert.True(t, exists(fs, path))
	})

	t.Run("returns false for non-existent file", func(t *testing.T) {
		fs := afero.NewMemMapFs()

		assert.False(t, exists(fs, "/nonexistent"))
	})

	t.Run("returns false for directory", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		path := "/test/dir"

		err := fs.MkdirAll(path, 0o755)
		require.NoError(t, err)

		assert.False(t, exists(fs, path))
	})
}

func TestIsDir(t *testing.T) {
	t.Run("returns true for existing directory", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		path := "/test/dir"

		err := fs.MkdirAll(path, 0o755)
		require.NoError(t, err)

		assert.True(t, isDir(fs, path))
	})

	t.Run("returns false for non-existent directory", func(t *testing.T) {
		fs := afero.NewMemMapFs()

		assert.False(t, isDir(fs, "/nonexistent"))
	})

	t.Run("returns false for file", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		path := "/test/file.txt"

		err := fs.MkdirAll("/test", 0o755)
		require.NoError(t, err)

		err = afero.WriteFile(fs, path, []byte("content"), 0o644)
		require.NoError(t, err)

		assert.False(t, isDir(fs, path))
	})
}
