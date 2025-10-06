package tool

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"github.com/spf13/afero"
)

// Tool represents a managed CLI tool that can be downloaded and executed.
//
//nolint:govet // fieldalignment: readability preferred over 8-byte optimization
type Tool struct {
	Name           string
	ProgressWriter io.Writer
	VersionFunc    func(context.Context) (string, error)
	DownloadURL    func(version, goos, goarch string) string
	ChecksumURL    func(version, goos, goarch string) string
	Fs             afero.Fs // Filesystem abstraction for testing (defaults to OsFs)
}

// Exec downloads the tool if not cached and executes it with the given arguments.
// It uses syscall.Exec to replace the current process with the tool.
func (t *Tool) Exec(ctx context.Context, args []string) error {
	fs := t.getFs()

	dataDir, err := DataDir(fs)
	if err != nil {
		return fmt.Errorf("failed to determine data directory: %w", err)
	}

	version, err := t.VersionFunc(ctx)
	if err != nil {
		return fmt.Errorf("failed to get version: %w", err)
	}

	binPath := filepath.Join(dataDir, "kdev", t.Name, version, t.Name)

	if !exists(fs, binPath) {
		if t.ProgressWriter != nil {
			if _, err := fmt.Fprintf(t.ProgressWriter, "Downloading %s %s...\n", t.Name, version); err != nil {
				return fmt.Errorf("failed to write progress: %w", err)
			}
		}

		if err := t.download(ctx, binPath, version); err != nil {
			return fmt.Errorf("failed to download: %w", err)
		}

		if t.ProgressWriter != nil {
			if _, err := fmt.Fprintf(t.ProgressWriter, "%s %s downloaded successfully\n", t.Name, version); err != nil {
				return fmt.Errorf("failed to write progress: %w", err)
			}
		}
	}

	if err := fs.Chmod(binPath, 0o755); err != nil {
		return fmt.Errorf("failed to make executable: %w", err)
	}

	execArgs := append([]string{t.Name}, args...)

	return syscall.Exec(binPath, execArgs, os.Environ())
}

// getFs returns the filesystem to use, defaulting to OsFs if not set.
func (t *Tool) getFs() afero.Fs {
	if t.Fs == nil {
		return afero.NewOsFs()
	}

	return t.Fs
}
