package tool

import (
	"context"
	"fmt"
	"io"

	"github.com/google/go-github/v58/github"
)

// NewCilium creates a Tool configured for cilium CLI.
func NewCilium(progress io.Writer) *Tool {
	return NewToolFromConfig(ciliumConfig(), progress)
}

func ciliumVersion(ctx context.Context) (version string, err error) {
	client := github.NewClient(nil)

	release, _, err := client.Repositories.GetLatestRelease(ctx, "cilium", "cilium-cli")
	if err != nil {
		return "", fmt.Errorf("failed to get latest cilium-cli release: %w", err)
	}

	return release.GetTagName(), nil
}

func ciliumDownloadURL(version, goos, goarch string) string {
	return fmt.Sprintf("https://github.com/cilium/cilium-cli/releases/download/%s/cilium-%s-%s.tar.gz",
		version, goos, goarch)
}

func ciliumChecksumURL(version, goos, goarch string) string {
	return ciliumDownloadURL(version, goos, goarch) + ".sha256sum"
}
