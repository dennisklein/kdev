package tool

import (
	"context"
	"fmt"
	"io"
	"strings"
)

// NewKubectl creates a Tool configured for kubectl.
func NewKubectl(progress io.Writer) *Tool {
	return NewToolFromConfig(kubectlConfig(), progress)
}

func kubectlVersion(ctx context.Context) (version string, err error) {
	client := getRetryableClient()

	return kubectlVersionWithClient(ctx, client.StandardClient(), "https://dl.k8s.io/release/stable.txt")
}

// kubectlVersionWithClient fetches kubectl version from the specified URL using the given client.
// This function is exported for testing purposes.
func kubectlVersionWithClient(ctx context.Context, client HTTPClient, url string) (string, error) {
	data, err := fetchHTTPContent(ctx, client, url)
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
