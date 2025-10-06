package util_test

import (
	"testing"

	"github.com/dennisklein/kdev/internal/util"
)

func TestFormatBytes(t *testing.T) {
	//nolint:govet // fieldalignment: test readability over optimization
	tests := []struct {
		name  string
		bytes int64
		want  string
	}{
		{
			name:  "bytes",
			bytes: 512,
			want:  "512 B",
		},
		{
			name:  "kilobytes",
			bytes: 1024,
			want:  "1.0 KiB",
		},
		{
			name:  "megabytes",
			bytes: 1024 * 1024,
			want:  "1.0 MiB",
		},
		{
			name:  "gigabytes",
			bytes: 1024 * 1024 * 1024,
			want:  "1.0 GiB",
		},
		{
			name:  "partial_kilobytes",
			bytes: 1536,
			want:  "1.5 KiB",
		},
		{
			name:  "large_value",
			bytes: 1024*1024*1024 + 536870912,
			want:  "1.5 GiB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := util.FormatBytes(tt.bytes)
			if got != tt.want {
				t.Errorf("FormatBytes(%d) = %v, want %v", tt.bytes, got, tt.want)
			}
		})
	}
}
