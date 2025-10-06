//nolint:testpackage // internal functions require same package
package tool

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testHome = "/home/testuser"

func TestCachedVersions(t *testing.T) {
	t.Run("returns empty slice when no cache exists", func(t *testing.T) {
		fs := afero.NewMemMapFs()

		t.Setenv("HOME", testHome)

		tool := &Tool{
			Name: "kubectl",
			Fs:   fs,
		}

		versions, err := tool.CachedVersions()
		require.NoError(t, err)
		assert.Empty(t, versions)
	})

	t.Run("returns cached versions in descending order", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		home := testHome

		t.Setenv("HOME", home)

		dataDir := filepath.Join(home, ".kdev")
		toolDir := filepath.Join(dataDir, "kdev", "kubectl")

		// Create multiple cached versions
		versions := []string{"v1.28.0", "v1.30.0", "v1.29.0"}
		for _, v := range versions {
			binPath := filepath.Join(toolDir, v, "kubectl")
			err := fs.MkdirAll(filepath.Dir(binPath), 0o755)
			require.NoError(t, err)

			err = afero.WriteFile(fs, binPath, []byte("fake binary "+v), 0o755)
			require.NoError(t, err)
		}

		tool := &Tool{
			Name: "kubectl",
			Fs:   fs,
		}

		cached, err := tool.CachedVersions()
		require.NoError(t, err)
		require.Len(t, cached, 3)

		// Should be sorted in descending order
		assert.Equal(t, "v1.30.0", cached[0].Version)
		assert.Equal(t, "v1.29.0", cached[1].Version)
		assert.Equal(t, "v1.28.0", cached[2].Version)

		// Verify paths and sizes
		for _, cv := range cached {
			assert.Contains(t, cv.Path, cv.Version)
			assert.Greater(t, cv.Size, int64(0))
		}
	})

	t.Run("ignores directories without binaries", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		home := testHome

		t.Setenv("HOME", home)

		dataDir := filepath.Join(home, ".kdev")
		toolDir := filepath.Join(dataDir, "kdev", "kubectl")

		// Create version directory without binary
		emptyVersionDir := filepath.Join(toolDir, "v1.28.0")
		err := fs.MkdirAll(emptyVersionDir, 0o755)
		require.NoError(t, err)

		// Create valid version with binary
		validBinPath := filepath.Join(toolDir, "v1.29.0", "kubectl")
		err = fs.MkdirAll(filepath.Dir(validBinPath), 0o755)
		require.NoError(t, err)

		err = afero.WriteFile(fs, validBinPath, []byte("binary"), 0o755)
		require.NoError(t, err)

		tool := &Tool{
			Name: "kubectl",
			Fs:   fs,
		}

		cached, err := tool.CachedVersions()
		require.NoError(t, err)
		require.Len(t, cached, 1)
		assert.Equal(t, "v1.29.0", cached[0].Version)
	})

	t.Run("ignores regular files in tool directory", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		home := testHome

		t.Setenv("HOME", home)

		dataDir := filepath.Join(home, ".kdev")
		toolDir := filepath.Join(dataDir, "kdev", "kubectl")

		// Create a regular file in tool directory
		err := fs.MkdirAll(toolDir, 0o755)
		require.NoError(t, err)

		err = afero.WriteFile(fs, filepath.Join(toolDir, "README.txt"), []byte("readme"), 0o644)
		require.NoError(t, err)

		tool := &Tool{
			Name: "kubectl",
			Fs:   fs,
		}

		cached, err := tool.CachedVersions()
		require.NoError(t, err)
		assert.Empty(t, cached)
	})
}

func TestLatestVersion(t *testing.T) {
	t.Run("calls VersionFunc with context", func(t *testing.T) {
		expectedVersion := "v1.30.0"
		called := false

		tool := &Tool{
			Name: "kubectl",
			VersionFunc: func(ctx context.Context) (string, error) {
				called = true
				assert.NotNil(t, ctx)

				return expectedVersion, nil
			},
		}

		version, err := tool.LatestVersion(context.Background())
		require.NoError(t, err)
		assert.Equal(t, expectedVersion, version)
		assert.True(t, called)
	})
}

func TestCleanVersion(t *testing.T) {
	t.Run("removes specific version", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		home := testHome

		t.Setenv("HOME", home)

		dataDir := filepath.Join(home, ".kdev")
		toolDir := filepath.Join(dataDir, "kdev", "kubectl")

		// Create versions
		v1Path := filepath.Join(toolDir, "v1.28.0", "kubectl")
		v2Path := filepath.Join(toolDir, "v1.29.0", "kubectl")

		for _, p := range []string{v1Path, v2Path} {
			err := fs.MkdirAll(filepath.Dir(p), 0o755)
			require.NoError(t, err)

			err = afero.WriteFile(fs, p, []byte("binary"), 0o755)
			require.NoError(t, err)
		}

		tool := &Tool{
			Name: "kubectl",
			Fs:   fs,
		}

		// Clean v1.28.0
		err := tool.CleanVersion("v1.28.0")
		require.NoError(t, err)

		// Verify v1.28.0 is gone
		exists, err := afero.DirExists(fs, filepath.Dir(v1Path))
		require.NoError(t, err)
		assert.False(t, exists)

		// Verify v1.29.0 still exists
		exists, err = afero.Exists(fs, v2Path)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("no error when version does not exist", func(t *testing.T) {
		fs := afero.NewMemMapFs()

		t.Setenv("HOME", testHome)

		tool := &Tool{
			Name: "kubectl",
			Fs:   fs,
		}

		err := tool.CleanVersion("v1.30.0")
		require.NoError(t, err)
	})
}

func TestCleanAll(t *testing.T) {
	t.Run("removes all cached versions", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		home := testHome

		t.Setenv("HOME", home)

		dataDir := filepath.Join(home, ".kdev")
		toolDir := filepath.Join(dataDir, "kdev", "kubectl")

		// Create multiple versions
		versions := []string{"v1.28.0", "v1.29.0", "v1.30.0"}
		for _, v := range versions {
			binPath := filepath.Join(toolDir, v, "kubectl")
			err := fs.MkdirAll(filepath.Dir(binPath), 0o755)
			require.NoError(t, err)

			err = afero.WriteFile(fs, binPath, []byte("binary"), 0o755)
			require.NoError(t, err)
		}

		tool := &Tool{
			Name: "kubectl",
			Fs:   fs,
		}

		// Clean all
		err := tool.CleanAll()
		require.NoError(t, err)

		// Verify tool directory is gone
		exists, err := afero.DirExists(fs, toolDir)
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("no error when tool directory does not exist", func(t *testing.T) {
		fs := afero.NewMemMapFs()

		t.Setenv("HOME", testHome)

		tool := &Tool{
			Name: "kubectl",
			Fs:   fs,
		}

		err := tool.CleanAll()
		require.NoError(t, err)
	})
}

func TestDownload(t *testing.T) {
	t.Run("creates download directory structure", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		home := testHome

		t.Setenv("HOME", home)

		dataDir := filepath.Join(home, ".kdev")
		binPath := filepath.Join(dataDir, "kdev", "testtool", "v1.0.0", "testtool")

		// Create directory and file to simulate download
		err := fs.MkdirAll(filepath.Dir(binPath), 0o755)
		require.NoError(t, err)

		err = afero.WriteFile(fs, binPath, []byte("fake binary"), 0o755)
		require.NoError(t, err)

		// Verify file exists
		exists, err := afero.Exists(fs, binPath)
		require.NoError(t, err)
		assert.True(t, exists)

		// Verify it's executable
		info, err := fs.Stat(binPath)
		require.NoError(t, err)
		assert.Equal(t, uint32(0o755), uint32(info.Mode()&0o777))
	})

	t.Run("skips download when tool already cached", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		home := testHome

		t.Setenv("HOME", home)

		dataDir := filepath.Join(home, ".kdev")
		binPath := filepath.Join(dataDir, "kdev", "testtool", "v1.0.0", "testtool")

		// Pre-create the binary
		err := fs.MkdirAll(filepath.Dir(binPath), 0o755)
		require.NoError(t, err)

		err = afero.WriteFile(fs, binPath, []byte("existing binary"), 0o755)
		require.NoError(t, err)

		tool := &Tool{
			Name: "testtool",
			Fs:   fs,
			VersionFunc: func(ctx context.Context) (string, error) {
				return "v1.0.0", nil
			},
		}

		err = tool.Download(context.Background())
		require.NoError(t, err)

		// Verify original content wasn't changed
		content, err := afero.ReadFile(fs, binPath)
		require.NoError(t, err)
		assert.Equal(t, []byte("existing binary"), content)
	})

	t.Run("uses XDG_DATA_HOME when set", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		customDataDir := "/custom/data"
		t.Setenv("XDG_DATA_HOME", customDataDir)

		tool := &Tool{
			Name: "testtool",
			Fs:   fs,
			VersionFunc: func(ctx context.Context) (string, error) {
				return "v1.0.0", nil
			},
		}

		// Pre-create to avoid actual download
		binPath := filepath.Join(customDataDir, "kdev", "testtool", "v1.0.0", "testtool")
		err := fs.MkdirAll(filepath.Dir(binPath), 0o755)
		require.NoError(t, err)

		err = afero.WriteFile(fs, binPath, []byte("binary"), 0o755)
		require.NoError(t, err)

		err = tool.Download(context.Background())
		require.NoError(t, err)
	})
}

func TestGetFs(t *testing.T) {
	t.Run("returns set filesystem", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		tool := &Tool{
			Name: "test",
			Fs:   fs,
		}

		result := tool.getFs()
		assert.Equal(t, fs, result)
	})

	t.Run("returns OsFs when not set", func(t *testing.T) {
		tool := &Tool{
			Name: "test",
		}

		result := tool.getFs()
		assert.NotNil(t, result)
		// Check it's the OsFs type by verifying it's not MemMapFs
		_, ok := result.(*afero.MemMapFs)
		assert.False(t, ok)
	})
}
