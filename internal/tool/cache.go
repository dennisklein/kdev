package tool

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"

	"github.com/Masterminds/semver/v3"
	"github.com/spf13/afero"
)

// CachedVersion represents a cached version of a tool.
type CachedVersion struct {
	Version string
	Path    string
	Size    int64
}

// CachedVersions returns all cached versions of this tool.
func (t *Tool) CachedVersions() ([]CachedVersion, error) {
	fs := t.getFs()
	helper := t.getFSHelper()

	dataDir, err := DataDir(fs)
	if err != nil {
		return nil, fmt.Errorf("failed to get data directory: %w", err)
	}

	toolDir := filepath.Join(dataDir, "kdev", t.Name)

	if !helper.IsDir(toolDir) {
		return nil, nil
	}

	entries, err := afero.ReadDir(fs, toolDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read tool directory: %w", err)
	}

	versions := make([]CachedVersion, 0, len(entries))

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		binPath := filepath.Join(toolDir, entry.Name(), t.Name)
		if !helper.Exists(binPath) {
			continue
		}

		info, err := fs.Stat(binPath)
		if err != nil {
			continue
		}

		versions = append(versions, CachedVersion{
			Version: entry.Name(),
			Path:    binPath,
			Size:    info.Size(),
		})
	}

	sort.Slice(versions, func(i, j int) bool {
		return compareVersions(versions[i].Version, versions[j].Version) > 0
	})

	return versions, nil
}

// compareVersions compares two version strings using semantic versioning.
// Returns: 1 if v1 > v2, -1 if v1 < v2, 0 if equal.
// Falls back to string comparison if versions aren't valid semver.
func compareVersions(v1, v2 string) int {
	ver1, err1 := semver.NewVersion(v1)
	ver2, err2 := semver.NewVersion(v2)

	// If either version is not valid semver, fall back to string comparison
	if err1 != nil || err2 != nil {
		if v1 > v2 {
			return 1
		} else if v1 < v2 {
			return -1
		}

		return 0
	}

	return ver1.Compare(ver2)
}

// LatestVersion returns the latest available version from the upstream source.
func (t *Tool) LatestVersion(ctx context.Context) (string, error) {
	return t.VersionFunc(ctx)
}

// CleanVersion removes a specific cached version.
func (t *Tool) CleanVersion(version string) error {
	fs := t.getFs()
	helper := t.getFSHelper()

	dataDir, err := DataDir(fs)
	if err != nil {
		return fmt.Errorf("failed to get data directory: %w", err)
	}

	versionDir := filepath.Join(dataDir, "kdev", t.Name, version)

	if !helper.IsDir(versionDir) {
		return nil
	}

	if err := fs.RemoveAll(versionDir); err != nil {
		return fmt.Errorf("failed to remove version directory: %w", err)
	}

	return nil
}

// CleanAll removes all cached versions of this tool.
func (t *Tool) CleanAll() error {
	fs := t.getFs()
	helper := t.getFSHelper()

	dataDir, err := DataDir(fs)
	if err != nil {
		return fmt.Errorf("failed to get data directory: %w", err)
	}

	toolDir := filepath.Join(dataDir, "kdev", t.Name)

	if !helper.IsDir(toolDir) {
		return nil
	}

	if err := fs.RemoveAll(toolDir); err != nil {
		return fmt.Errorf("failed to remove tool directory: %w", err)
	}

	return nil
}

// Download pre-downloads the tool without executing it.
func (t *Tool) Download(ctx context.Context) error {
	fs := t.getFs()
	helper := t.getFSHelper()

	dataDir, err := DataDir(fs)
	if err != nil {
		return fmt.Errorf("failed to determine data directory: %w", err)
	}

	version, err := t.VersionFunc(ctx)
	if err != nil {
		return fmt.Errorf("failed to get version: %w", err)
	}

	binPath := filepath.Join(dataDir, "kdev", t.Name, version, t.Name)

	if helper.Exists(binPath) {
		return nil
	}

	if err := t.writeProgress("Downloading %s %s...\n", t.Name, version); err != nil {
		return fmt.Errorf("failed to write progress: %w", err)
	}

	if err := t.download(ctx, binPath, version); err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}

	if err := fs.Chmod(binPath, 0o755); err != nil {
		return fmt.Errorf("failed to make executable: %w", err)
	}

	if err := t.writeProgress("%s %s downloaded successfully\n", t.Name, version); err != nil {
		return fmt.Errorf("failed to write progress: %w", err)
	}

	return nil
}
