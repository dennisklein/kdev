package tool

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/afero"
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

	// If the downloaded file is a tar.gz, extract it
	if strings.HasSuffix(url, ".tar.gz") {
		if err := extractTarGzFile(fs, tmpFile, destPath, t.Name); err != nil {
			return fmt.Errorf("failed to extract archive: %w", err)
		}

		return nil
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

// extractTarGzFile extracts a single binary from a tar.gz file.
// It looks for a file matching the tool name in the archive root.
func extractTarGzFile(fs afero.Fs, archivePath, destPath, toolName string) error {
	// Open the archive
	archiveFile, err := fs.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer archiveFile.Close() //nolint:errcheck // close on read-only file

	gzr, err := gzip.NewReader(archiveFile)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzr.Close() //nolint:errcheck // close on reader

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			return fmt.Errorf("failed to read tar: %w", err)
		}

		// Look for the binary matching the tool name
		if filepath.Base(header.Name) == toolName {
			out, err := fs.Create(destPath)
			if err != nil {
				return fmt.Errorf("failed to create output file: %w", err)
			}

			if _, err := io.Copy(out, tr); err != nil {
				_ = out.Close() //nolint:errcheck // close on error path

				return fmt.Errorf("failed to extract binary: %w", err)
			}

			if err := out.Close(); err != nil {
				return err
			}

			// Remove the archive file after successful extraction
			return fs.Remove(archivePath)
		}
	}

	return fmt.Errorf("binary %s not found in archive", toolName)
}
