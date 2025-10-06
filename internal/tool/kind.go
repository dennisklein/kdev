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
	return NewToolFromConfig(kindConfig(), progress)
}

func kindVersion(ctx context.Context) (version string, err error) {
	return kindVersionWithClient(ctx, http.DefaultClient, "https://api.github.com/repos/kubernetes-sigs/kind/releases/latest")
}

// kindVersionWithClient fetches kind version from the specified URL using the given client.
// This function is exported for testing purposes.
func kindVersionWithClient(ctx context.Context, client *http.Client, url string) (string, error) {
	data, err := fetchHTTPContent(ctx, client, url)
	if err != nil {
		return "", err
	}

	var release struct {
		TagName string `json:"tag_name"`
	}

	if err := json.Unmarshal(data, &release); err != nil {
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
