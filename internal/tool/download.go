package tool

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"
)

func (t *Tool) download(ctx context.Context, destPath, version string) error {
	fs := t.getFs()

	url := t.DownloadURL(version, runtime.GOOS, runtime.GOARCH)
	checksumURL := t.ChecksumURL(version, runtime.GOOS, runtime.GOARCH)

	if err := fs.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	expectedChecksum, err := fetchChecksum(ctx, checksumURL)
	if err != nil {
		return fmt.Errorf("failed to fetch checksum: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			err = closeErr
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	tmpFile := destPath + ".tmp"

	out, err := fs.Create(tmpFile)
	if err != nil {
		return err
	}

	defer func() {
		if removeErr := fs.Remove(tmpFile); removeErr != nil && err == nil {
			err = removeErr
		}
	}()

	hasher := sha256.New()
	writer := io.MultiWriter(out, hasher)

	if _, err := io.Copy(writer, resp.Body); err != nil {
		if closeErr := out.Close(); closeErr != nil {
			return closeErr
		}

		return err
	}

	if err := out.Close(); err != nil {
		return err
	}

	actualChecksum := fmt.Sprintf("%x", hasher.Sum(nil))
	if actualChecksum != expectedChecksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum)
	}

	return fs.Rename(tmpFile, destPath)
}

func fetchChecksum(ctx context.Context, url string) (string, error) {
	data, err := fetchHTTPContent(ctx, http.DefaultClient, url)
	if err != nil {
		return "", err
	}

	checksumStr := strings.TrimSpace(string(data))

	// Handle checksums in the format "checksum  filename" (like sha256sum output)
	// Extract just the checksum part (first field)
	if parts := strings.Fields(checksumStr); len(parts) > 0 {
		return parts[0], nil
	}

	return checksumStr, nil
}
