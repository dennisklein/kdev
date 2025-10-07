package tool

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/lipgloss"

	"github.com/dennisklein/kdev/internal/util"
)

// ProgressReader wraps an io.Reader and reports progress.
//
//nolint:govet // fieldalignment: readability preferred over minor memory optimization
type ProgressReader struct {
	reader   io.Reader
	total    int64
	current  int64
	progress progress.Model
	writer   io.Writer
	lastPct  int
}

// NewProgressReader creates a new progress reader.
func NewProgressReader(reader io.Reader, total int64, writer io.Writer) *ProgressReader {
	prog := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(40),
		progress.WithoutPercentage(),
	)

	return &ProgressReader{
		reader:   reader,
		total:    total,
		progress: prog,
		writer:   writer,
		lastPct:  -1,
	}
}

// Read implements io.Reader and updates progress.
func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	pr.current += int64(n)

	// Update progress display
	if pr.writer != nil && pr.total > 0 {
		percent := float64(pr.current) / float64(pr.total)
		currentPct := int(percent * 100)

		// Only update every 5% to avoid too many updates
		if currentPct != pr.lastPct && (currentPct%5 == 0 || currentPct == 100 || pr.lastPct == -1) {
			pr.lastPct = currentPct
			pr.render(percent)
		}
	}

	return n, err
}

// render displays the progress bar.
func (pr *ProgressReader) render(percent float64) {
	if pr.writer == nil {
		return
	}

	// Clear line and move cursor to start
	_, _ = fmt.Fprint(pr.writer, "\r\033[K") //nolint:errcheck // best effort progress display

	// Render progress bar
	bar := pr.progress.ViewAs(percent)

	// Add percentage and size info
	downloaded := util.FormatBytes(pr.current)
	total := util.FormatBytes(pr.total)

	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("green"))

	info := style.Render(fmt.Sprintf(" %3.0f%% (%s / %s)", percent*100, downloaded, total))

	_, _ = fmt.Fprint(pr.writer, bar+info) //nolint:errcheck // best effort progress display
}

// Finish completes the progress display.
func (pr *ProgressReader) Finish() {
	if pr.writer != nil {
		pr.render(1.0)
		_, _ = fmt.Fprintln(pr.writer) //nolint:errcheck // best effort progress display
	}
}

// ProgressWriter wraps progress messages for non-interactive output.
type ProgressWriter struct {
	writer io.Writer
}

// NewProgressWriter creates a simple progress writer.
func NewProgressWriter(writer io.Writer) *ProgressWriter {
	return &ProgressWriter{writer: writer}
}

// WriteMessage writes a progress message.
func (pw *ProgressWriter) WriteMessage(format string, args ...interface{}) error {
	if pw.writer == nil {
		return nil
	}

	_, err := fmt.Fprintf(pw.writer, format, args...)

	return err
}
