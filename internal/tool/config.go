package tool

import (
	"context"
	"io"
)

// Config defines the configuration for creating a Tool.
//
//nolint:govet // fieldalignment: readability preferred over optimization
type Config struct {
	Name        string
	VersionFunc func(context.Context) (string, error)
	DownloadURL func(version, goos, goarch string) string
	ChecksumURL func(version, goos, goarch string) string
}

// NewToolFromConfig creates a Tool from a configuration.
func NewToolFromConfig(cfg Config, progress io.Writer) *Tool {
	return &Tool{
		Name:           cfg.Name,
		ProgressWriter: progress,
		VersionFunc:    cfg.VersionFunc,
		DownloadURL:    cfg.DownloadURL,
		ChecksumURL:    cfg.ChecksumURL,
	}
}

// kubectlConfig returns the configuration for kubectl.
func kubectlConfig() Config {
	return Config{
		Name:        "kubectl",
		VersionFunc: kubectlVersion,
		DownloadURL: kubectlDownloadURL,
		ChecksumURL: kubectlChecksumURL,
	}
}

// kindConfig returns the configuration for kind.
func kindConfig() Config {
	return Config{
		Name:        "kind",
		VersionFunc: kindVersion,
		DownloadURL: kindDownloadURL,
		ChecksumURL: kindChecksumURL,
	}
}
