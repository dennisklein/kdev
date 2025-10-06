package main

import (
	"fmt"
	"io"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/dennisklein/kdev/internal/tool"
)

const (
	toolNameWidth = 15
	versionWidth  = 10
	sizeWidth     = 10
)

var (
	toolNameStyle   = lipgloss.NewStyle().Bold(true).Width(toolNameWidth).Align(lipgloss.Left)
	versionStyle    = lipgloss.NewStyle().Width(versionWidth).Align(lipgloss.Left)
	latestStyle     = versionStyle.Foreground(lipgloss.Color("10"))
	oldVersionStyle = versionStyle.Foreground(lipgloss.Color("8"))
	notCachedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	sizeStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Width(sizeWidth).Align(lipgloss.Right)
	totalSizeStyle  = sizeStyle.Bold(true)
	successStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	infoStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
)

func newToolsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tools",
		Short: "Manage cached tools",
		Long:  `Manage cached CLI tools (clean, info, update).`,
	}

	cmd.AddCommand(newToolsCleanCmd())
	cmd.AddCommand(newToolsInfoCmd())
	cmd.AddCommand(newToolsUpdateCmd())

	return cmd
}

func newToolsCleanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clean [tool...]",
		Short: "Remove cached tools",
		Long:  `Remove cached tool binaries. If no tool names are specified, cleans all tools.`,
		RunE:  runToolsClean,
	}

	cmd.Flags().Bool("old", false, "Only remove obsolete versions (keep most recent)")

	return cmd
}

func newToolsInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info [tool...]",
		Short: "Show cached tool information",
		Long:  `Show version, path, and size information for cached tools. If no tool names are specified, shows all tools.`,
		RunE:  runToolsInfo,
	}
}

func newToolsUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update [tool...]",
		Short: "Update tools to latest version",
		Long:  `Check for and download the latest version of tools. If no tool names are specified, updates all tools.`,
		RunE:  runToolsUpdate,
	}
}

func runToolsClean(cmd *cobra.Command, args []string) error {
	out := cmd.OutOrStdout()
	registry := tool.NewRegistry(out)
	tools := resolveTools(registry, args)

	cleanOld, err := cmd.Flags().GetBool("old")
	if err != nil {
		return fmt.Errorf("failed to get --old flag: %w", err)
	}

	var totalReclaimed int64

	for _, t := range tools {
		if cleanOld {
			// Clean only old versions (keep most recent)
			versions, err := t.CachedVersions()
			if err != nil {
				return fmt.Errorf("failed to get cached versions for %s: %w", t.Name, err)
			}

			// Skip the first version (newest), clean the rest
			for i := 1; i < len(versions); i++ {
				totalReclaimed += versions[i].Size

				if err := t.CleanVersion(versions[i].Version); err != nil {
					return fmt.Errorf("failed to clean %s version %s: %w", t.Name, versions[i].Version, err)
				}
			}
		} else {
			// Clean all versions
			versions, err := t.CachedVersions()
			if err != nil {
				return fmt.Errorf("failed to get cached versions for %s: %w", t.Name, err)
			}

			for _, v := range versions {
				totalReclaimed += v.Size
			}

			if err := t.CleanAll(); err != nil {
				return fmt.Errorf("failed to clean %s: %w", t.Name, err)
			}
		}
	}

	if totalReclaimed > 0 {
		reclaimedStr := successStyle.Bold(true).Render(formatBytes(totalReclaimed))
		message := "Reclaimed"

		if _, err := fmt.Fprintf(out, "%s %s\n", message, reclaimedStr); err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}
	}

	return nil
}

func runToolsInfo(cmd *cobra.Command, args []string) error {
	out := cmd.OutOrStdout()
	registry := tool.NewRegistry(nil)
	tools := resolveTools(registry, args)

	var totalSize int64

	for _, t := range tools {
		size, err := printToolInfo(out, t)
		if err != nil {
			return err
		}

		totalSize += size
	}

	// Print total if more than one tool
	if len(tools) > 1 && totalSize > 0 {
		totalName := toolNameStyle.Render("cache size")
		emptyVersion := versionStyle.Render("")
		totalSizeStr := totalSizeStyle.Render(formatBytes(totalSize))

		if _, err := fmt.Fprintf(out, "\n%s  %s  %s\n", totalName, emptyVersion, totalSizeStr); err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}
	}

	return nil
}

func printToolInfo(out io.Writer, t *tool.Tool) (int64, error) {
	versions, err := t.CachedVersions()
	if err != nil {
		return 0, fmt.Errorf("failed to get cached versions for %s: %w", t.Name, err)
	}

	if len(versions) == 0 {
		toolName := toolNameStyle.Render(t.Name)
		notCached := notCachedStyle.Render("(not cached)")

		if _, writeErr := fmt.Fprintf(out, "%s  %s\n", toolName, notCached); writeErr != nil {
			return 0, fmt.Errorf("failed to write output: %w", writeErr)
		}

		return 0, nil
	}

	// Print each cached version on one line: toolname version size path
	// Highlight the newest cached version (first in list) in green
	var totalSize int64

	for i, v := range versions {
		totalSize += v.Size

		toolName := toolNameStyle.Render(t.Name)

		style := oldVersionStyle
		if i == 0 {
			// First version is the newest (versions are sorted descending)
			style = latestStyle
		}

		styledVersion := style.Render(v.Version)
		styledSize := sizeStyle.Render(formatBytes(v.Size))

		if _, err := fmt.Fprintf(out, "%s  %s  %s  %s\n", toolName, styledVersion, styledSize, v.Path); err != nil {
			return 0, fmt.Errorf("failed to write output: %w", err)
		}
	}

	return totalSize, nil
}

func runToolsUpdate(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	out := cmd.OutOrStdout()
	registry := tool.NewRegistry(out)
	tools := resolveTools(registry, args)

	for _, t := range tools {
		latest, err := t.LatestVersion(ctx)
		if err != nil {
			return fmt.Errorf("failed to get latest version for %s: %w", t.Name, err)
		}

		versions, err := t.CachedVersions()
		if err != nil {
			return fmt.Errorf("failed to get cached versions for %s: %w", t.Name, err)
		}

		alreadyCached := false

		for _, v := range versions {
			if v.Version == latest {
				alreadyCached = true

				break
			}
		}

		toolName := toolNameStyle.Render(t.Name)
		version := latestStyle.Render(latest)

		if alreadyCached {
			message := infoStyle.Render("already cached")
			if _, err := fmt.Fprintf(out, "%s %s %s\n", toolName, version, message); err != nil {
				return fmt.Errorf("failed to write output: %w", err)
			}

			continue
		}

		if err := t.Download(ctx); err != nil {
			return fmt.Errorf("failed to download %s: %w", t.Name, err)
		}
	}

	return nil
}

func resolveTools(registry *tool.Registry, names []string) []*tool.Tool {
	if len(names) == 0 {
		return registry.AllTools()
	}

	tools := make([]*tool.Tool, 0, len(names))

	for _, name := range names {
		if t := registry.Get(name); t != nil {
			tools = append(tools, t)
		}
	}

	return tools
}

func formatBytes(bytes int64) string {
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
