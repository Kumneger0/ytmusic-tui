package cmd

import (
	"github.com/spf13/cobra"
)

func newLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:          "login",
		Short:        "login to YouTube Music",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// find different way
			return nil
		},
	}
}
