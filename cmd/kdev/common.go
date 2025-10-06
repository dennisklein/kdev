package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/dennisklein/kdev/internal/tool"
)

// newToolCmd creates a generic command for tools that can be auto-downloaded and executed.
func newToolCmd(toolName, shortDesc string) *cobra.Command {
	return &cobra.Command{
		Use:                toolName,
		Short:              shortDesc,
		Long:               fmt.Sprintf("Lazily downloads and executes %s, passing through all arguments.", toolName),
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			registry := tool.NewRegistry(os.Stdout)
			t := registry.Get(toolName)
			if t == nil {
				return fmt.Errorf("unknown tool: %s", toolName)
			}

			return t.Exec(ctx, args)
		},
	}
}