package main

import (
	"errors"
	"runtime/debug"

	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version info",
		Long:  ``,
		RunE:  version,
	}

	return cmd
}

func version(cmd *cobra.Command, args []string) error {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return errors.New("could not read embedded build info ('go build -buildvcs=true')")
	}

	cmd.Println(info.Main.Version)

	return nil
}
