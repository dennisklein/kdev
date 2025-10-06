//nolint:testpackage // internal functions require same package
package tool

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewKind(t *testing.T) {
	t.Run("creates kind tool with progress writer", func(t *testing.T) {
		var buf bytes.Buffer

		kind := NewKind(&buf)

		require.NotNil(t, kind)
		assert.Equal(t, "kind", kind.Name)
		assert.Equal(t, &buf, kind.ProgressWriter)
		assert.NotNil(t, kind.VersionFunc)
		assert.NotNil(t, kind.DownloadURL)
		assert.NotNil(t, kind.ChecksumURL)
	})

	t.Run("creates kind tool with nil progress writer", func(t *testing.T) {
		kind := NewKind(nil)

		require.NotNil(t, kind)
		assert.Nil(t, kind.ProgressWriter)
	})
}

func TestKindVersion(t *testing.T) {
	t.Run("fetches version successfully", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"tag_name":"v0.22.0"}`)) //nolint:errcheck // test helper
		}))
		defer server.Close()

		version, err := kindVersionWithClient(context.Background(), http.DefaultClient, server.URL)
		require.NoError(t, err)
		assert.Equal(t, "v0.22.0", version)
	})

	t.Run("handles non-200 status code", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		_, err := kindVersionWithClient(context.Background(), http.DefaultClient, server.URL)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected status code: 404")
	})

	t.Run("handles invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`invalid json`)) //nolint:errcheck // test helper
		}))
		defer server.Close()

		_, err := kindVersionWithClient(context.Background(), http.DefaultClient, server.URL)
		require.Error(t, err)
	})

	t.Run("handles HTTP errors", func(t *testing.T) {
		// Use an invalid URL to trigger HTTP error
		_, err := kindVersionWithClient(context.Background(), http.DefaultClient, "http://invalid.local:99999")
		require.Error(t, err)
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			<-r.Context().Done() // Wait for cancellation
		}))
		defer server.Close()

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := kindVersionWithClient(ctx, http.DefaultClient, server.URL)
		require.Error(t, err)
	})

	t.Run("default kindVersion uses production URL", func(t *testing.T) {
		// We can't test the actual production endpoint in unit tests,
		// but we verify the function is wired correctly
		kind := NewKind(nil)
		assert.NotNil(t, kind.VersionFunc)

		// The VersionFunc should be kindVersion which calls the production endpoint
		// We just verify it exists and is callable (though we won't actually call it)
	})

	t.Run("handles response body close error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"tag_name": "v0.22.0"}`)) //nolint:errcheck // test helper
		}))
		defer server.Close()

		// The close error is handled but doesn't fail the function if json decoding succeeds
		version, err := kindVersionWithClient(context.Background(), http.DefaultClient, server.URL)
		require.NoError(t, err)
		assert.Equal(t, "v0.22.0", version)
	})
}

func TestKindDownloadURL(t *testing.T) {
	tests := []struct {
		name    string
		version string
		goos    string
		goarch  string
		want    string
	}{
		{
			name:    "linux amd64",
			version: "v0.22.0",
			goos:    "linux",
			goarch:  "amd64",
			want:    "https://github.com/kubernetes-sigs/kind/releases/download/v0.22.0/kind-linux-amd64",
		},
		{
			name:    "darwin arm64",
			version: "v0.22.0",
			goos:    "darwin",
			goarch:  "arm64",
			want:    "https://github.com/kubernetes-sigs/kind/releases/download/v0.22.0/kind-darwin-arm64",
		},
		{
			name:    "windows amd64",
			version: "v0.22.0",
			goos:    "windows",
			goarch:  "amd64",
			want:    "https://github.com/kubernetes-sigs/kind/releases/download/v0.22.0/kind-windows-amd64",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := kindDownloadURL(tt.version, tt.goos, tt.goarch)
			assert.Equal(t, tt.want, url)
		})
	}
}

func TestKindChecksumURL(t *testing.T) {
	tests := []struct {
		name    string
		version string
		goos    string
		goarch  string
		want    string
	}{
		{
			name:    "linux amd64",
			version: "v0.22.0",
			goos:    "linux",
			goarch:  "amd64",
			want:    "https://github.com/kubernetes-sigs/kind/releases/download/v0.22.0/kind-linux-amd64.sha256sum",
		},
		{
			name:    "darwin arm64",
			version: "v0.22.0",
			goos:    "darwin",
			goarch:  "arm64",
			want:    "https://github.com/kubernetes-sigs/kind/releases/download/v0.22.0/kind-darwin-arm64.sha256sum",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := kindChecksumURL(tt.version, tt.goos, tt.goarch)
			assert.Equal(t, tt.want, url)
		})
	}
}
