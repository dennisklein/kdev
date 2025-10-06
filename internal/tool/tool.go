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
	fsHelper       *FSHelper
}

// Exec downloads the tool if not cached and executes it with the given arguments.
// It uses syscall.Exec to replace the current process with the tool.
func (t *Tool) Exec(ctx context.Context, args []string) error {
	binPath, execArgs, err := t.prepareExec(ctx, args)
	if err != nil {
		return err
	}

	return syscall.Exec(binPath, execArgs, os.Environ())
}

// prepareExec prepares the binary for execution by ensuring it's downloaded,
// cached, and executable. Returns the binary path and arguments to execute.
func (t *Tool) prepareExec(ctx context.Context, args []string) (string, []string, error) {
	fs := t.getFs()
	helper := t.getFSHelper()

	dataDir, err := DataDir(fs)
	if err != nil {
		return "", nil, fmt.Errorf("failed to determine data directory: %w", err)
	}

	version, err := t.VersionFunc(ctx)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get version: %w", err)
	}

	binPath := filepath.Join(dataDir, "kdev", t.Name, version, t.Name)

	if !helper.Exists(binPath) {
		if err := t.writeProgress("Downloading %s %s...\n", t.Name, version); err != nil {
			return "", nil, fmt.Errorf("failed to write progress: %w", err)
		}

		if err := t.download(ctx, binPath, version); err != nil {
			return "", nil, fmt.Errorf("failed to download: %w", err)
		}

		if err := t.writeProgress("%s %s downloaded successfully\n", t.Name, version); err != nil {
			return "", nil, fmt.Errorf("failed to write progress: %w", err)
		}
	}

	if err := fs.Chmod(binPath, 0o755); err != nil {
		return "", nil, fmt.Errorf("failed to make executable: %w", err)
	}

	execArgs := append([]string{t.Name}, args...)

	return binPath, execArgs, nil
}

// getFs returns the filesystem to use, defaulting to OsFs if not set.
func (t *Tool) getFs() afero.Fs {
	if t.Fs == nil {
		return afero.NewOsFs()
	}

	return t.Fs
}

// getFSHelper returns the filesystem helper, creating one if needed.
func (t *Tool) getFSHelper() *FSHelper {
	if t.fsHelper == nil {
		t.fsHelper = NewFSHelper(t.getFs())
	}
	return t.fsHelper
}

// writeProgress writes a progress message if a ProgressWriter is configured.
func (t *Tool) writeProgress(format string, args ...interface{}) error {
	if t.ProgressWriter != nil {
		_, err := fmt.Fprintf(t.ProgressWriter, format, args...)
		return err
	}

	return nil
}
