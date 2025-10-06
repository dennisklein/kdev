package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/dennisklein/kdev/internal/tool"
)

func newKubectlCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                "kubectl",
		Short:              "Execute kubectl (auto-downloads if needed)",
		Long:               `Lazily downloads and executes kubectl, passing through all arguments.`,
		DisableFlagParsing: true,
		RunE:               runKubectl,
	}

	return cmd
}

func runKubectl(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	kubectl := tool.NewKubectl(os.Stdout)

	return kubectl.Exec(ctx, args)
}
