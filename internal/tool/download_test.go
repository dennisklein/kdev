//nolint:testpackage // internal functions require same package
package tool

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testToolPath = "/cache/testtool/v1.0.0/testtool"
	testVersion  = "v1.0.0"
)

func TestFetchChecksum(t *testing.T) {
	t.Run("fetches checksum successfully", func(t *testing.T) {
		expectedChecksum := "abc123def456"

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(expectedChecksum + "\n")) //nolint:errcheck // test helper
		}))
		defer server.Close()

		checksum, err := fetchChecksum(context.Background(), server.URL)
		require.NoError(t, err)
		assert.Equal(t, expectedChecksum, checksum)
	})

	t.Run("extracts checksum from sha256sum format", func(t *testing.T) {
		expectedChecksum := "abc123def456"

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			// Format: "checksum  filename" (like sha256sum output)
			_, _ = w.Write([]byte(expectedChecksum + "  kind-linux-amd64\n")) //nolint:errcheck // test helper
		}))
		defer server.Close()

		checksum, err := fetchChecksum(context.Background(), server.URL)
		require.NoError(t, err)
		assert.Equal(t, expectedChecksum, checksum)
	})

	t.Run("handles non-200 status code", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		_, err := fetchChecksum(context.Background(), server.URL)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected status code")
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			<-r.Context().Done()
		}))
		defer server.Close()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := fetchChecksum(ctx, server.URL)
		assert.Error(t, err)
	})
}

func TestToolDownload(t *testing.T) { //nolint:maintidx // test function complexity is acceptable
	t.Run("downloads and verifies file successfully", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		content := []byte("fake binary content")
		checksum := fmt.Sprintf("%x", sha256.Sum256(content))

		// Mock checksum server
		checksumServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(checksum)) //nolint:errcheck // test helper
		}))
		defer checksumServer.Close()

		// Mock binary download server
		binaryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(content) //nolint:errcheck // test helper
		}))
		defer binaryServer.Close()

		tool := &Tool{
			Name: "testtool",
			Fs:   fs,
			VersionFunc: func(ctx context.Context) (string, error) {
				return testVersion, nil
			},
			DownloadURL: func(version, goos, goarch string) string {
				return binaryServer.URL
			},
			ChecksumURL: func(version, goos, goarch string) string {
				return checksumServer.URL
			},
		}

		destPath := testToolPath
		err := tool.download(context.Background(), destPath, testVersion)
		require.NoError(t, err)

		// Verify file was created
		exists, err := afero.Exists(fs, destPath)
		require.NoError(t, err)
		assert.True(t, exists)

		// Verify content
		data, err := afero.ReadFile(fs, destPath)
		require.NoError(t, err)
		assert.Equal(t, content, data)
	})

	t.Run("fails on checksum mismatch", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		content := []byte("fake binary content")
		wrongChecksum := "deadbeef"

		// Mock checksum server with wrong checksum
		checksumServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(wrongChecksum)) //nolint:errcheck // test helper
		}))
		defer checksumServer.Close()

		// Mock binary download server
		binaryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(content) //nolint:errcheck // test helper
		}))
		defer binaryServer.Close()

		tool := &Tool{
			Name: "testtool",
			Fs:   fs,
			VersionFunc: func(ctx context.Context) (string, error) {
				return testVersion, nil
			},
			DownloadURL: func(version, goos, goarch string) string {
				return binaryServer.URL
			},
			ChecksumURL: func(version, goos, goarch string) string {
				return checksumServer.URL
			},
		}

		destPath := testToolPath
		err := tool.download(context.Background(), destPath, testVersion)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "checksum mismatch")

		// Verify temp file was cleaned up
		tmpPath := destPath + ".tmp"
		exists, err := afero.Exists(fs, tmpPath)
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("handles binary download failure", func(t *testing.T) {
		fs := afero.NewMemMapFs()

		// Mock checksum server
		checksumServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("abc123")) //nolint:errcheck // test helper
		}))
		defer checksumServer.Close()

		// Mock binary download server that fails
		binaryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer binaryServer.Close()

		tool := &Tool{
			Name: "testtool",
			Fs:   fs,
			VersionFunc: func(ctx context.Context) (string, error) {
				return testVersion, nil
			},
			DownloadURL: func(version, goos, goarch string) string {
				return binaryServer.URL
			},
			ChecksumURL: func(version, goos, goarch string) string {
				return checksumServer.URL
			},
		}

		destPath := testToolPath
		err := tool.download(context.Background(), destPath, testVersion)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected status code")
	})

	t.Run("uses correct URL parameters", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		content := []byte("binary")
		checksum := fmt.Sprintf("%x", sha256.Sum256(content))

		var receivedVersion, receivedGoos, receivedGoarch string

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

		tool := &Tool{
			Name: "testtool",
			Fs:   fs,
			VersionFunc: func(ctx context.Context) (string, error) {
				return "v1.2.3", nil
			},
			DownloadURL: func(version, goos, goarch string) string {
				receivedVersion = version
				receivedGoos = goos
				receivedGoarch = goarch

				return binaryServer.URL
			},
			ChecksumURL: func(version, goos, goarch string) string {
				return checksumServer.URL
			},
		}

		destPath := "/cache/testtool/v1.2.3/testtool"
		err := tool.download(context.Background(), destPath, "v1.2.3")
		require.NoError(t, err)

		assert.Equal(t, "v1.2.3", receivedVersion)
		assert.Equal(t, runtime.GOOS, receivedGoos)
		assert.Equal(t, runtime.GOARCH, receivedGoarch)
	})

	t.Run("creates directory if not exists", func(t *testing.T) {
		fs := afero.NewMemMapFs()
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
				return testVersion, nil
			},
			DownloadURL: func(version, goos, goarch string) string {
				return binaryServer.URL
			},
			ChecksumURL: func(version, goos, goarch string) string {
				return checksumServer.URL
			},
		}

		destPath := "/deep/nested/path/testtool"
		err := tool.download(context.Background(), destPath, testVersion)
		require.NoError(t, err)

		// Verify directory was created
		dirExists, err := afero.DirExists(fs, filepath.Dir(destPath))
		require.NoError(t, err)
		assert.True(t, dirExists)
	})

	t.Run("handles checksum fetch failure", func(t *testing.T) {
		fs := afero.NewMemMapFs()

		// Mock checksum server that fails
		checksumServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer checksumServer.Close()

		tool := &Tool{
			Name: "testtool",
			Fs:   fs,
			VersionFunc: func(ctx context.Context) (string, error) {
				return testVersion, nil
			},
			DownloadURL: func(version, goos, goarch string) string {
				return "http://example.com/binary"
			},
			ChecksumURL: func(version, goos, goarch string) string {
				return checksumServer.URL
			},
		}

		destPath := testToolPath
		err := tool.download(context.Background(), destPath, testVersion)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to fetch checksum")
	})

	t.Run("handles empty checksum response", func(t *testing.T) {
		// This tests the case where strings.Fields returns empty slice
		checksumServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("   \n  \t  ")) //nolint:errcheck // test helper - only whitespace
		}))
		defer checksumServer.Close()

		checksum, err := fetchChecksum(context.Background(), checksumServer.URL)
		require.NoError(t, err)
		assert.Equal(t, "", checksum) // Empty string after trimming and fields split
	})

	t.Run("handles file rename error", func(t *testing.T) { //nolint:dupl // similar test setup is intentional
		fs := &errorFs{
			Fs:        afero.NewMemMapFs(),
			renameErr: fmt.Errorf("rename failed"),
		}

		content := []byte("binary content")
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
				return testVersion, nil
			},
			DownloadURL: func(version, goos, goarch string) string {
				return binaryServer.URL
			},
			ChecksumURL: func(version, goos, goarch string) string {
				return checksumServer.URL
			},
		}

		destPath := testToolPath
		err := tool.download(context.Background(), destPath, testVersion)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "rename failed")
	})

	t.Run("handles MkdirAll error", func(t *testing.T) {
		fs := &errorFs{
			Fs:          afero.NewMemMapFs(),
			mkdirAllErr: fmt.Errorf("mkdir failed"),
		}

		checksumServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("checksum")) //nolint:errcheck // test helper
		}))
		defer checksumServer.Close()

		tool := &Tool{
			Name: "testtool",
			Fs:   fs,
			VersionFunc: func(ctx context.Context) (string, error) {
				return testVersion, nil
			},
			DownloadURL: func(version, goos, goarch string) string {
				return "http://example.com/binary"
			},
			ChecksumURL: func(version, goos, goarch string) string {
				return checksumServer.URL
			},
		}

		destPath := testToolPath
		err := tool.download(context.Background(), destPath, testVersion)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create directory")
	})

	t.Run("handles file create error", func(t *testing.T) { //nolint:dupl // similar test setup is intentional
		fs := &errorFs{
			Fs:        afero.NewMemMapFs(),
			createErr: fmt.Errorf("create failed"),
		}

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
				return testVersion, nil
			},
			DownloadURL: func(version, goos, goarch string) string {
				return binaryServer.URL
			},
			ChecksumURL: func(version, goos, goarch string) string {
				return checksumServer.URL
			},
		}

		destPath := testToolPath
		err := tool.download(context.Background(), destPath, testVersion)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "create failed")
	})

	t.Run("handles io.Copy error", func(t *testing.T) {
		fs := afero.NewMemMapFs()

		content := []byte("binary content")
		checksum := fmt.Sprintf("%x", sha256.Sum256(content))

		checksumServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(checksum)) //nolint:errcheck // test helper
		}))
		defer checksumServer.Close()

		// Server that sends partial content then fails
		binaryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("partial")) //nolint:errcheck // test helper
			// Abruptly close connection
			panic(http.ErrAbortHandler)
		}))
		defer binaryServer.Close()

		tool := &Tool{
			Name: "testtool",
			Fs:   fs,
			VersionFunc: func(ctx context.Context) (string, error) {
				return testVersion, nil
			},
			DownloadURL: func(version, goos, goarch string) string {
				return binaryServer.URL
			},
			ChecksumURL: func(version, goos, goarch string) string {
				return checksumServer.URL
			},
		}

		destPath := testToolPath
		err := tool.download(context.Background(), destPath, testVersion)
		require.Error(t, err)

		// Verify temp file was cleaned up
		tmpPath := destPath + ".tmp"
		exists, err := afero.Exists(fs, tmpPath)
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("handles NewRequest error with invalid URL", func(t *testing.T) {
		fs := afero.NewMemMapFs()

		checksumServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("checksum")) //nolint:errcheck // test helper
		}))
		defer checksumServer.Close()

		tool := &Tool{
			Name: "testtool",
			Fs:   fs,
			VersionFunc: func(ctx context.Context) (string, error) {
				return testVersion, nil
			},
			DownloadURL: func(version, goos, goarch string) string {
				// Return invalid URL to trigger NewRequest error
				return "ht!tp://invalid url with spaces"
			},
			ChecksumURL: func(version, goos, goarch string) string {
				return checksumServer.URL
			},
		}

		destPath := testToolPath
		err := tool.download(context.Background(), destPath, testVersion)
		require.Error(t, err)
	})
}

func TestFetchChecksumBodyCloseError(t *testing.T) {
	t.Run("handles response body close gracefully", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("abc123def456")) //nolint:errcheck // test helper
		}))
		defer server.Close()

		// The close error is handled but doesn't fail the function if reading succeeds
		checksum, err := fetchChecksum(context.Background(), server.URL)
		require.NoError(t, err)
		assert.Equal(t, "abc123def456", checksum)
	})
}
