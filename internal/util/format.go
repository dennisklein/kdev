package util

import "fmt"

// FormatBytes formats a byte count into a human-readable string with appropriate units.
func FormatBytes(bytes int64) string {
	const unit = 1024

	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	units := []string{"KiB", "MiB", "GiB", "TiB"}

	return fmt.Sprintf("%.1f %s", float64(bytes)/float64(div), units[exp])
}
