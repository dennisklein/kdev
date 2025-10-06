package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/dennisklein/kdev/internal/tool"
)

func newKindCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                "kind",
		Short:              "Execute kind (auto-downloads if needed)",
		Long:               `Lazily downloads and executes kind, passing through all arguments.`,
		DisableFlagParsing: true,
		RunE:               runKind,
	}

	return cmd
}

func runKind(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	kind := tool.NewKind(os.Stdout)

	return kind.Exec(ctx, args)
}
