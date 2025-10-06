package tool

import "github.com/spf13/afero"

// FSHelper provides filesystem helper methods.
type FSHelper struct {
	fs afero.Fs
}

// NewFSHelper creates a new filesystem helper.
func NewFSHelper(fs afero.Fs) *FSHelper {
	if fs == nil {
		fs = afero.NewOsFs()
	}
	return &FSHelper{fs: fs}
}

// Exists checks if a file exists and is not a directory.
func (h *FSHelper) Exists(path string) bool {
	info, err := h.fs.Stat(path)
	return err == nil && !info.IsDir()
}

// IsDir checks if a path exists and is a directory.
func (h *FSHelper) IsDir(path string) bool {
	info, err := h.fs.Stat(path)
	return err == nil && info.IsDir()
}

// Fs returns the underlying filesystem.
func (h *FSHelper) Fs() afero.Fs {
	return h.fs
}