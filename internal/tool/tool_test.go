//nolint:testpackage // internal functions require same package
package tool

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	kubectlTestVersion = "v1.30.0"
	testUser           = "/home/testuser"
)

func TestToolGetFs(t *testing.T) {
	t.Run("returns configured filesystem", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		tool := &Tool{
			Name: "kubectl",
			Fs:   fs,
		}

		result := tool.getFs()
		assert.Equal(t, fs, result)
	})

	t.Run("returns OsFs when nil", func(t *testing.T) {
		tool := &Tool{
			Name: "kubectl",
		}

		result := tool.getFs()
		assert.NotNil(t, result)
	})
}

// TestToolExecPreparation tests everything up to the syscall.Exec call.
// We cannot test syscall.Exec itself as it replaces the current process.
func TestToolExecPreparation(t *testing.T) {
	t.Run("downloads tool when not cached", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		home := testUser
		t.Setenv("HOME", home)

		content := []byte("fake kubectl binary")
		checksum := fmt.Sprintf("%x", sha256.Sum256(content))

		// Mock servers
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

		var progressBuf bytes.Buffer

		tool := &Tool{
			Name:           "kubectl",
			Fs:             fs,
			ProgressWriter: &progressBuf,
			VersionFunc: func(ctx context.Context) (string, error) {
				return kubectlTestVersion, nil
			},
			DownloadURL: func(version, goos, goarch string) string {
				return binaryServer.URL
			},
			ChecksumURL: func(version, goos, goarch string) string {
				return checksumServer.URL
			},
		}

		dataDir := filepath.Join(home, ".kdev")
		binPath := filepath.Join(dataDir, "kdev", "kubectl", "v1.30.0", "kubectl")

		// Download the tool (this tests everything except syscall.Exec)
		err := tool.Download(context.Background())
		require.NoError(t, err)

		// Verify binary was downloaded
		exists, err := afero.Exists(fs, binPath)
		require.NoError(t, err)
		assert.True(t, exists)

		// Verify content
		data, err := afero.ReadFile(fs, binPath)
		require.NoError(t, err)
		assert.Equal(t, content, data)

		// Verify progress messages
		progress := progressBuf.String()
		assert.Contains(t, progress, "Downloading kubectl v1.30.0")
		assert.Contains(t, progress, "kubectl v1.30.0 downloaded successfully")

		// Verify file is executable
		info, err := fs.Stat(binPath)
		require.NoError(t, err)
		assert.Equal(t, uint32(0o755), uint32(info.Mode()&0o777))
	})

	t.Run("uses cached tool when available", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		home := testUser
		t.Setenv("HOME", home)

		dataDir := filepath.Join(home, ".kdev")
		binPath := filepath.Join(dataDir, "kdev", "kubectl", "v1.30.0", "kubectl")

		// Pre-create cached binary
		err := fs.MkdirAll(filepath.Dir(binPath), 0o755)
		require.NoError(t, err)

		err = afero.WriteFile(fs, binPath, []byte("cached binary"), 0o755)
		require.NoError(t, err)

		var progressBuf bytes.Buffer

		tool := &Tool{
			Name:           "kubectl",
			Fs:             fs,
			ProgressWriter: &progressBuf,
			VersionFunc: func(ctx context.Context) (string, error) {
				return kubectlTestVersion, nil
			},
			DownloadURL: func(version, goos, goarch string) string {
				return "http://should-not-be-called.example.com"
			},
			ChecksumURL: func(version, goos, goarch string) string {
				return "http://should-not-be-called.example.com"
			},
		}

		// Verify tool exists (simulating what Exec does before syscall.Exec)
		err = tool.Download(context.Background())
		require.NoError(t, err)

		// Verify no download messages (already cached)
		progress := progressBuf.String()
		assert.Empty(t, progress)

		// Verify cached binary is still there
		data, err := afero.ReadFile(fs, binPath)
		require.NoError(t, err)
		assert.Equal(t, []byte("cached binary"), data)
	})

	t.Run("handles version fetch error", func(t *testing.T) {
		fs := afero.NewMemMapFs()

		t.Setenv("HOME", testUser)

		tool := &Tool{
			Name: "kubectl",
			Fs:   fs,
			VersionFunc: func(ctx context.Context) (string, error) {
				return "", fmt.Errorf("network error")
			},
		}

		err := tool.Download(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get version")
	})

	t.Run("writes progress messages", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		home := testUser
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

		var progressBuf bytes.Buffer

		tool := &Tool{
			Name:           "testtool",
			Fs:             fs,
			ProgressWriter: &progressBuf,
			VersionFunc: func(ctx context.Context) (string, error) {
				return "v2.0.0", nil
			},
			DownloadURL: func(version, goos, goarch string) string {
				return binaryServer.URL
			},
			ChecksumURL: func(version, goos, goarch string) string {
				return checksumServer.URL
			},
		}

		err := tool.Download(context.Background())
		require.NoError(t, err)

		progress := progressBuf.String()
		assert.Contains(t, progress, "Downloading testtool v2.0.0")
		assert.Contains(t, progress, "testtool v2.0.0 downloaded successfully")
	})

	t.Run("no progress messages when writer is nil", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		home := testUser
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
			Name:           "testtool",
			Fs:             fs,
			ProgressWriter: nil, // No writer
			VersionFunc: func(ctx context.Context) (string, error) {
				return "v2.0.0", nil
			},
			DownloadURL: func(version, goos, goarch string) string {
				return binaryServer.URL
			},
			ChecksumURL: func(version, goos, goarch string) string {
				return checksumServer.URL
			},
		}

		// Should not panic even without progress writer
		err := tool.Download(context.Background())
		require.NoError(t, err)
	})
}

// Note: We cannot test syscall.Exec directly as it replaces the current process.
// The actual Exec method is tested indirectly through integration tests and
// manual testing with the CLI.
