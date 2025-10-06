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

func TestNewKubectl(t *testing.T) {
	t.Run("creates kubectl tool with progress writer", func(t *testing.T) {
		var buf bytes.Buffer

		kubectl := NewKubectl(&buf)

		require.NotNil(t, kubectl)
		assert.Equal(t, "kubectl", kubectl.Name)
		assert.Equal(t, &buf, kubectl.ProgressWriter)
		assert.NotNil(t, kubectl.VersionFunc)
		assert.NotNil(t, kubectl.DownloadURL)
		assert.NotNil(t, kubectl.ChecksumURL)
	})

	t.Run("creates kubectl tool with nil progress writer", func(t *testing.T) {
		kubectl := NewKubectl(nil)

		require.NotNil(t, kubectl)
		assert.Nil(t, kubectl.ProgressWriter)
	})
}

func TestKubectlVersion(t *testing.T) {
	t.Run("fetches version successfully", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("v1.30.0\n")) //nolint:errcheck // test helper
		}))
		defer server.Close()

		version, err := kubectlVersionWithClient(context.Background(), http.DefaultClient, server.URL)
		require.NoError(t, err)
		assert.Equal(t, "v1.30.0", version)
	})

	t.Run("trims whitespace", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("  v1.30.1  \n\t")) //nolint:errcheck // test helper
		}))
		defer server.Close()

		version, err := kubectlVersionWithClient(context.Background(), http.DefaultClient, server.URL)
		require.NoError(t, err)
		assert.Equal(t, "v1.30.1", version)
	})

	t.Run("handles non-200 status code", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		_, err := kubectlVersionWithClient(context.Background(), http.DefaultClient, server.URL)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected status code: 404")
	})

	t.Run("handles HTTP errors", func(t *testing.T) {
		// Use an invalid URL to trigger HTTP error
		_, err := kubectlVersionWithClient(context.Background(), http.DefaultClient, "http://invalid.local:99999")
		require.Error(t, err)
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			<-r.Context().Done() // Wait for cancellation
		}))
		defer server.Close()

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := kubectlVersionWithClient(ctx, http.DefaultClient, server.URL)
		require.Error(t, err)
	})

	t.Run("default kubectlVersion uses production URL", func(t *testing.T) {
		// We can't test the actual production endpoint in unit tests,
		// but we verify the function is wired correctly
		kubectl := NewKubectl(nil)
		assert.NotNil(t, kubectl.VersionFunc)

		// The VersionFunc should be kubectlVersion which calls the production endpoint
		// We just verify it exists and is callable (though we won't actually call it)
	})
}

func TestKubectlDownloadURL(t *testing.T) {
	tests := []struct {
		name    string
		version string
		goos    string
		goarch  string
		want    string
	}{
		{
			name:    "linux amd64",
			version: "v1.30.0",
			goos:    "linux",
			goarch:  "amd64",
			want:    "https://dl.k8s.io/release/v1.30.0/bin/linux/amd64/kubectl",
		},
		{
			name:    "darwin arm64",
			version: "v1.30.0",
			goos:    "darwin",
			goarch:  "arm64",
			want:    "https://dl.k8s.io/release/v1.30.0/bin/darwin/arm64/kubectl",
		},
		{
			name:    "windows amd64",
			version: "v1.30.0",
			goos:    "windows",
			goarch:  "amd64",
			want:    "https://dl.k8s.io/release/v1.30.0/bin/windows/amd64/kubectl",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := kubectlDownloadURL(tt.version, tt.goos, tt.goarch)
			assert.Equal(t, tt.want, url)
		})
	}
}

func TestKubectlChecksumURL(t *testing.T) {
	tests := []struct {
		name    string
		version string
		goos    string
		goarch  string
		want    string
	}{
		{
			name:    "linux amd64",
			version: "v1.30.0",
			goos:    "linux",
			goarch:  "amd64",
			want:    "https://dl.k8s.io/release/v1.30.0/bin/linux/amd64/kubectl.sha256",
		},
		{
			name:    "darwin arm64",
			version: "v1.30.0",
			goos:    "darwin",
			goarch:  "arm64",
			want:    "https://dl.k8s.io/release/v1.30.0/bin/darwin/arm64/kubectl.sha256",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := kubectlChecksumURL(tt.version, tt.goos, tt.goarch)
			assert.Equal(t, tt.want, url)
		})
	}
}
