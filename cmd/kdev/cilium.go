package main

import (
	"github.com/spf13/cobra"
)

func newCiliumCmd() *cobra.Command {
	return newToolCmd("cilium", "Execute cilium CLI (auto-downloads if needed)")
}
