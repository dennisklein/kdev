package main

import (
	"github.com/spf13/cobra"
)

func newKindCmd() *cobra.Command {
	return newToolCmd("kind", "Execute kind (auto-downloads if needed)")
}
