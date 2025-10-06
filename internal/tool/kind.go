package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// NewKind creates a Tool configured for kind (Kubernetes in Docker).
func NewKind(progress io.Writer) *Tool {
	return &Tool{
		Name:           "kind",
		ProgressWriter: progress,
		VersionFunc:    kindVersion,
		DownloadURL:    kindDownloadURL,
		ChecksumURL:    kindChecksumURL,
	}
}

func kindVersion(ctx context.Context) (version string, err error) {
	return kindVersionWithClient(ctx, http.DefaultClient, "https://api.github.com/repos/kubernetes-sigs/kind/releases/latest")
}

// kindVersionWithClient fetches kind version from the specified URL using the given client.
// This function is exported for testing purposes.
func kindVersionWithClient(ctx context.Context, client *http.Client, url string) (version string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	return release.TagName, nil
}

func kindDownloadURL(version, goos, goarch string) string {
	return fmt.Sprintf("https://github.com/kubernetes-sigs/kind/releases/download/%s/kind-%s-%s",
		version, goos, goarch)
}

func kindChecksumURL(version, goos, goarch string) string {
	return kindDownloadURL(version, goos, goarch) + ".sha256sum"
}
