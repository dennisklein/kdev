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

	client := getRetryableClient()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := client.StandardClient().Do(req)
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

	// Use progress reader if we have a progress writer and content length
	var reader io.Reader = resp.Body

	var progReader *ProgressReader

	if t.ProgressWriter != nil && resp.ContentLength > 0 {
		progReader = NewProgressReader(resp.Body, resp.ContentLength, t.ProgressWriter)
		reader = progReader
	}

	writer := io.MultiWriter(out, hasher)

	if _, err := io.Copy(writer, reader); err != nil {
		if closeErr := out.Close(); closeErr != nil {
			return closeErr
		}

		return err
	}

	// Finish progress display
	if progReader != nil {
		progReader.Finish()
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
	client := getRetryableClient()

	data, err := fetchHTTPContent(ctx, client.StandardClient(), url)
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
