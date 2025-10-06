package tool

import (
	"context"
	"fmt"
	"io"

	"github.com/google/go-github/v58/github"
)

// NewKind creates a Tool configured for kind (Kubernetes in Docker).
func NewKind(progress io.Writer) *Tool {
	return NewToolFromConfig(kindConfig(), progress)
}

func kindVersion(ctx context.Context) (version string, err error) {
	client := github.NewClient(nil)

	release, _, err := client.Repositories.GetLatestRelease(ctx, "kubernetes-sigs", "kind")
	if err != nil {
		return "", fmt.Errorf("failed to get latest kind release: %w", err)
	}

	return release.GetTagName(), nil
}

func kindDownloadURL(version, goos, goarch string) string {
	return fmt.Sprintf("https://github.com/kubernetes-sigs/kind/releases/download/%s/kind-%s-%s",
		version, goos, goarch)
}

func kindChecksumURL(version, goos, goarch string) string {
	return kindDownloadURL(version, goos, goarch) + ".sha256sum"
}
