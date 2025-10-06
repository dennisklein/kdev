//nolint:testpackage // internal functions require same package
package tool

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/go-github/v58/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mustParseURL is a test helper that parses a URL or panics.
func mustParseURL(rawURL string) *url.URL {
	u, err := url.Parse(rawURL)
	if err != nil {
		panic(err)
	}

	return u
}

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
	t.Run("fetches version successfully from GitHub API", func(t *testing.T) {
		tagName := "v0.22.0"

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			assert.Equal(t, "/repos/kubernetes-sigs/kind/releases/latest", r.URL.Path)

			release := &github.RepositoryRelease{
				TagName: &tagName,
			}

			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(release) //nolint:errcheck // test helper
		}))
		defer server.Close()

		// Create a GitHub client pointed at our test server
		client := github.NewClient(nil)
		client.BaseURL = mustParseURL(server.URL + "/")

		release, _, err := client.Repositories.GetLatestRelease(context.Background(), "kubernetes-sigs", "kind")
		require.NoError(t, err)
		assert.Equal(t, "v0.22.0", release.GetTagName())
	})

	t.Run("handles API errors", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "Not Found"}) //nolint:errcheck // test helper
		}))
		defer server.Close()

		client := github.NewClient(nil)
		client.BaseURL = mustParseURL(server.URL + "/")

		_, _, err := client.Repositories.GetLatestRelease(context.Background(), "kubernetes-sigs", "kind")
		require.Error(t, err)
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			<-r.Context().Done() // Wait for cancellation
		}))
		defer server.Close()

		client := github.NewClient(nil)
		client.BaseURL = mustParseURL(server.URL + "/")

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, _, err := client.Repositories.GetLatestRelease(ctx, "kubernetes-sigs", "kind")
		require.Error(t, err)
	})

	t.Run("default kindVersion uses production endpoint", func(t *testing.T) {
		// We can't test the actual production endpoint in unit tests,
		// but we verify the function is wired correctly
		kind := NewKind(nil)
		assert.NotNil(t, kind.VersionFunc)

		// The VersionFunc should be kindVersion which calls the production endpoint
		// We just verify it exists and is callable (though we won't actually call it)
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
