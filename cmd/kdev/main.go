package main

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "kdev",
	Short: "Manage opinionated local kind-based Kubernetes dev clusters",
	Long:  `kdev is a tool for managing opinionated, local, kind-based Kubernetes development clusters.`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(newVersionCmd())
}

func main() {
	Execute()
}
