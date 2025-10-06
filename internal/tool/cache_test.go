//nolint:testpackage // internal functions require same package
package tool

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testHome = "/home/testuser"

//nolint:maintidx // test function with multiple subtests
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

	t.Run("skips version when Stat fails on second call", func(t *testing.T) {
		baseFs := afero.NewMemMapFs()
		home := testHome

		t.Setenv("HOME", home)

		dataDir := filepath.Join(home, ".kdev")
		toolDir := filepath.Join(dataDir, "kdev", "kubectl")

		// Create multiple cached versions
		v1Path := filepath.Join(toolDir, "v1.28.0", "kubectl")
		v2Path := filepath.Join(toolDir, "v1.29.0", "kubectl")
		v3Path := filepath.Join(toolDir, "v1.30.0", "kubectl")

		for _, p := range []string{v1Path, v2Path, v3Path} {
			err := baseFs.MkdirAll(filepath.Dir(p), 0o755)
			require.NoError(t, err)

			err = afero.WriteFile(baseFs, p, []byte("binary"), 0o755)
			require.NoError(t, err)
		}

		// Wrap with errorFs that fails Stat on second call to v1.29.0
		// This simulates a race condition where the file exists when
		// exists() checks but fails when getting size info
		fs := &errorFs{
			Fs:               baseFs,
			statErrPath:      v2Path,
			statErr:          fmt.Errorf("permission denied"),
			statErrAfterCall: 1, // Fail on second call
		}

		tool := &Tool{
			Name: "kubectl",
			Fs:   fs,
		}

		cached, err := tool.CachedVersions()
		require.NoError(t, err)

		// Should only have 2 versions (v1.30.0 and v1.28.0)
		// v1.29.0 should be skipped due to Stat error on second call
		require.Len(t, cached, 2)
		assert.Equal(t, "v1.30.0", cached[0].Version)
		assert.Equal(t, "v1.28.0", cached[1].Version)
	})

	t.Run("sorts versions using semantic versioning", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		home := testHome

		t.Setenv("HOME", home)

		dataDir := filepath.Join(home, ".kdev")
		toolDir := filepath.Join(dataDir, "kdev", "kubectl")

		// Create versions that would be incorrectly sorted with string comparison
		// String sort: v1.2.2 > v1.2.10 (incorrect)
		// Semver sort: v1.2.10 > v1.2.2 (correct)
		versions := []string{"v1.2.2", "v1.2.10", "v1.2.3", "v1.2.1"}
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
		require.Len(t, cached, 4)

		// Should be sorted in descending semantic version order
		assert.Equal(t, "v1.2.10", cached[0].Version, "v1.2.10 should be first (newest)")
		assert.Equal(t, "v1.2.3", cached[1].Version)
		assert.Equal(t, "v1.2.2", cached[2].Version)
		assert.Equal(t, "v1.2.1", cached[3].Version, "v1.2.1 should be last (oldest)")
	})

	t.Run("handles non-semver versions with fallback", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		home := testHome

		t.Setenv("HOME", home)

		dataDir := filepath.Join(home, ".kdev")
		toolDir := filepath.Join(dataDir, "kdev", "kubectl")

		// Mix of semver and non-semver versions
		versions := []string{"v1.30.0", "latest", "v1.29.0", "dev-branch"}
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
		require.Len(t, cached, 4)

		// Semver versions should be properly sorted
		// Non-semver falls back to string comparison
		// v1.30.0 and v1.29.0 should be sorted correctly
		var semverVersions []string

		for _, cv := range cached {
			if cv.Version == "v1.30.0" || cv.Version == "v1.29.0" {
				semverVersions = append(semverVersions, cv.Version)
			}
		}

		require.Len(t, semverVersions, 2)
		assert.Equal(t, "v1.30.0", semverVersions[0], "v1.30.0 should come before v1.29.0")
		assert.Equal(t, "v1.29.0", semverVersions[1])
	})
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name     string
		v1       string
		v2       string
		expected int
	}{
		{
			name:     "v1 greater than v2",
			v1:       "v1.30.0",
			v2:       "v1.29.0",
			expected: 1,
		},
		{
			name:     "v1 less than v2",
			v1:       "v1.28.0",
			v2:       "v1.29.0",
			expected: -1,
		},
		{
			name:     "versions equal",
			v1:       "v1.29.0",
			v2:       "v1.29.0",
			expected: 0,
		},
		{
			name:     "semantic version edge case - double digit patch",
			v1:       "v1.2.10",
			v2:       "v1.2.2",
			expected: 1, // v1.2.10 > v1.2.2 (NOT string comparison)
		},
		{
			name:     "semantic version edge case - double digit minor",
			v1:       "v1.10.0",
			v2:       "v1.2.0",
			expected: 1, // v1.10.0 > v1.2.0
		},
		{
			name:     "non-semver fallback to string comparison v1 > v2",
			v1:       "latest",
			v2:       "dev",
			expected: 1, // "latest" > "dev" alphabetically
		},
		{
			name:     "non-semver fallback to string comparison v1 < v2",
			v1:       "alpha",
			v2:       "beta",
			expected: -1, // "alpha" < "beta" alphabetically
		},
		{
			name:     "non-semver equal",
			v1:       "latest",
			v2:       "latest",
			expected: 0,
		},
		{
			name:     "mixed: semver vs non-semver",
			v1:       "v1.30.0",
			v2:       "latest",
			expected: 1, // Falls back to string comparison: "v1.30.0" > "latest"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compareVersions(tt.v1, tt.v2)

			switch {
			case tt.expected > 0:
				assert.Greater(t, result, 0, "expected %s > %s", tt.v1, tt.v2)
			case tt.expected < 0:
				assert.Less(t, result, 0, "expected %s < %s", tt.v1, tt.v2)
			default:
				assert.Equal(t, 0, result, "expected %s == %s", tt.v1, tt.v2)
			}
		})
	}
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
				return "v1.0.0", nil //nolint:goconst // test version string
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
				return "v1.0.0", nil //nolint:goconst // test version string
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

func TestCachedVersionsErrors(t *testing.T) {
	t.Run("handles ReadDir error", func(t *testing.T) {
		fs := &errorFs{
			Fs:         afero.NewMemMapFs(),
			readDirErr: fmt.Errorf("permission denied"),
		}
		home := testHome
		t.Setenv("HOME", home)

		dataDir := filepath.Join(home, ".kdev")
		toolDir := filepath.Join(dataDir, "kdev", "kubectl")

		// Create tool directory
		err := fs.MkdirAll(toolDir, 0o755)
		require.NoError(t, err)

		tool := &Tool{
			Name: "kubectl",
			Fs:   fs,
		}

		_, err = tool.CachedVersions()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read tool directory")
	})
}

func TestCleanVersionErrors(t *testing.T) {
	t.Run("handles RemoveAll error", func(t *testing.T) {
		fs := &errorFs{
			Fs:           afero.NewMemMapFs(),
			removeAllErr: fmt.Errorf("permission denied"),
		}
		home := testHome
		t.Setenv("HOME", home)

		dataDir := filepath.Join(home, ".kdev")
		versionDir := filepath.Join(dataDir, "kdev", "kubectl", "v1.0.0")

		// Create version directory
		err := fs.MkdirAll(versionDir, 0o755)
		require.NoError(t, err)

		tool := &Tool{
			Name: "kubectl",
			Fs:   fs,
		}

		err = tool.CleanVersion("v1.0.0")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to remove version directory")
	})
}

func TestCleanAllErrors(t *testing.T) {
	t.Run("handles RemoveAll error", func(t *testing.T) {
		fs := &errorFs{
			Fs:           afero.NewMemMapFs(),
			removeAllErr: fmt.Errorf("permission denied"),
		}
		home := testHome
		t.Setenv("HOME", home)

		dataDir := filepath.Join(home, ".kdev")
		toolDir := filepath.Join(dataDir, "kdev", "kubectl")

		// Create tool directory
		err := fs.MkdirAll(toolDir, 0o755)
		require.NoError(t, err)

		tool := &Tool{
			Name: "kubectl",
			Fs:   fs,
		}

		err = tool.CleanAll()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to remove tool directory")
	})
}

func TestDownloadErrors(t *testing.T) {
	t.Run("handles Chmod error", func(t *testing.T) {
		fs := &errorFs{
			Fs:       afero.NewMemMapFs(),
			chmodErr: fmt.Errorf("permission denied"),
		}
		home := testHome
		t.Setenv("HOME", home)

		content := []byte("binary")
		checksum := fmt.Sprintf("%x", sha256.Sum256(content))

		checksumServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(checksum)) //nolint:errcheck // test helper
		}))
		defer checksumServer.Close()

		binaryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(content) //nolint:errcheck // test helper
		}))
		defer binaryServer.Close()

		tool := &Tool{
			Name: "testtool",
			Fs:   fs,
			VersionFunc: func(ctx context.Context) (string, error) {
				return "v1.0.0", nil //nolint:goconst // test version string
			},
			DownloadURL: func(version, goos, goarch string) string {
				return binaryServer.URL
			},
			ChecksumURL: func(version, goos, goarch string) string {
				return checksumServer.URL
			},
		}

		err := tool.Download(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to make executable")
	})

	t.Run("handles progress write error on download start", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		home := testHome
		t.Setenv("HOME", home)

		// Create a writer that always errors
		errWriter := &errorProgressWriter{err: fmt.Errorf("write error")}

		tool := &Tool{
			Name:           "testtool",
			Fs:             fs,
			ProgressWriter: errWriter,
			VersionFunc: func(ctx context.Context) (string, error) {
				return "v1.0.0", nil //nolint:goconst // test version string
			},
			DownloadURL: func(version, goos, goarch string) string {
				return "http://example.com/binary" //nolint:goconst // test URL
			},
			ChecksumURL: func(version, goos, goarch string) string {
				return "http://example.com/checksum" //nolint:goconst // test URL
			},
		}

		err := tool.Download(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to write progress")
	})

	t.Run("handles progress write error on download success", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		home := testHome
		t.Setenv("HOME", home)

		content := []byte("binary")
		checksum := fmt.Sprintf("%x", sha256.Sum256(content))

		checksumServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(checksum)) //nolint:errcheck // test helper
		}))
		defer checksumServer.Close()

		binaryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(content) //nolint:errcheck // test helper
		}))
		defer binaryServer.Close()

		// Writer that fails on second write
		errWriter := &errorProgressWriter{
			failAfter: 1,
			err:       fmt.Errorf("write error"),
		}

		tool := &Tool{
			Name:           "testtool",
			Fs:             fs,
			ProgressWriter: errWriter,
			VersionFunc: func(ctx context.Context) (string, error) {
				return "v1.0.0", nil //nolint:goconst // test version string
			},
			DownloadURL: func(version, goos, goarch string) string {
				return binaryServer.URL
			},
			ChecksumURL: func(version, goos, goarch string) string {
				return checksumServer.URL
			},
		}

		err := tool.Download(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to write progress")
	})
}

// errorFs is a test filesystem that can return errors for specific operations.
//
//nolint:govet // fieldalignment not critical for test helper
type errorFs struct {
	afero.Fs
	removeAllErr     error
	chmodErr         error
	readDirErr       error
	mkdirAllErr      error
	createErr        error
	renameErr        error
	statErrPath      string         // Path that should trigger statErr
	statErr          error          // Error to return for statErrPath
	statErrAfterCall int            // Only fail after this many calls to Stat
	statCallCount    map[string]int // Track call count per path
}

func (e *errorFs) RemoveAll(path string) error {
	if e.removeAllErr != nil {
		return e.removeAllErr
	}

	return e.Fs.RemoveAll(path)
}

func (e *errorFs) Chmod(name string, mode os.FileMode) error {
	if e.chmodErr != nil {
		return e.chmodErr
	}

	return e.Fs.Chmod(name, mode)
}

func (e *errorFs) Open(name string) (afero.File, error) {
	f, err := e.Fs.Open(name)
	if err != nil {
		return nil, err
	}

	return &errorFile{File: f, readDirErr: e.readDirErr}, nil
}

func (e *errorFs) MkdirAll(path string, perm os.FileMode) error {
	if e.mkdirAllErr != nil {
		return e.mkdirAllErr
	}

	return e.Fs.MkdirAll(path, perm)
}

func (e *errorFs) Create(name string) (afero.File, error) {
	if e.createErr != nil {
		return nil, e.createErr
	}

	return e.Fs.Create(name)
}

func (e *errorFs) Rename(oldname, newname string) error {
	if e.renameErr != nil {
		return e.renameErr
	}

	return e.Fs.Rename(oldname, newname)
}

func (e *errorFs) Stat(name string) (os.FileInfo, error) {
	if e.statErr != nil && e.statErrPath != "" && name == e.statErrPath {
		// Initialize call counter map if needed
		if e.statCallCount == nil {
			e.statCallCount = make(map[string]int)
		}

		// Increment call count for this path
		e.statCallCount[name]++

		// Only fail if we've exceeded the threshold
		if e.statErrAfterCall > 0 && e.statCallCount[name] > e.statErrAfterCall {
			return nil, e.statErr
		}
	}

	return e.Fs.Stat(name)
}

// errorFile wraps afero.File to return errors for ReadDir.
type errorFile struct {
	afero.File
	readDirErr error
}

func (e *errorFile) Readdir(count int) ([]os.FileInfo, error) {
	if e.readDirErr != nil {
		return nil, e.readDirErr
	}

	return e.File.Readdir(count)
}

// errorProgressWriter is a test writer that returns errors.
//
//nolint:govet // fieldalignment not critical for test helper
type errorProgressWriter struct {
	failAfter int // Number of writes before failing (0 = always fail)
	writes    int
	err       error
}

func (e *errorProgressWriter) Write(p []byte) (n int, err error) {
	e.writes++

	if e.failAfter == 0 || e.writes > e.failAfter {
		return 0, e.err
	}

	return len(p), nil
}
