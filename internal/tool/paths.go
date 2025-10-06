package tool

import (
	"os"
	"path/filepath"

	"github.com/spf13/afero"
)

// DataDir returns the appropriate data directory following XDG Base Directory spec.
// Priority: XDG_DATA_HOME > ~/.local/share (if exists) > ~/.kdev (fallback).
func DataDir(fs afero.Fs) (string, error) {
	if xdgData := os.Getenv("XDG_DATA_HOME"); xdgData != "" {
		return xdgData, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	xdgDefault := filepath.Join(homeDir, ".local", "share")
	if isDir(fs, xdgDefault) {
		return xdgDefault, nil
	}

	return filepath.Join(homeDir, ".kdev"), nil
}

func exists(fs afero.Fs, path string) bool {
	info, err := fs.Stat(path)

	return err == nil && !info.IsDir()
}

func isDir(fs afero.Fs, path string) bool {
	info, err := fs.Stat(path)

	return err == nil && info.IsDir()
}
