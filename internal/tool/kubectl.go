package tool

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// NewKubectl creates a Tool configured for kubectl.
func NewKubectl(progress io.Writer) *Tool {
	return &Tool{
		Name:           "kubectl",
		ProgressWriter: progress,
		VersionFunc:    kubectlVersion,
		DownloadURL:    kubectlDownloadURL,
		ChecksumURL:    kubectlChecksumURL,
	}
}

func kubectlVersion(ctx context.Context) (version string, err error) {
	return kubectlVersionWithClient(ctx, http.DefaultClient, "https://dl.k8s.io/release/stable.txt")
}

// kubectlVersionWithClient fetches kubectl version from the specified URL using the given client.
// This function is exported for testing purposes.
func kubectlVersionWithClient(ctx context.Context, client *http.Client, url string) (version string, err error) {
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

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(data)), nil
}

func kubectlDownloadURL(version, goos, goarch string) string {
	return fmt.Sprintf("https://dl.k8s.io/release/%s/bin/%s/%s/kubectl",
		version, goos, goarch)
}

func kubectlChecksumURL(version, goos, goarch string) string {
	return kubectlDownloadURL(version, goos, goarch) + ".sha256"
}
