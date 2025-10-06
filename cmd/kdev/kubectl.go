package main

import (
	"github.com/spf13/cobra"
)

func newKubectlCmd() *cobra.Command {
	return newToolCmd("kubectl", "Execute kubectl (auto-downloads if needed)")
}
