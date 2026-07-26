package cmd

import (
	"fmt"
	"os"
	"os/exec"

	backend "github.com/kumneger0/clispot/backend"
	"github.com/spf13/cobra"
)

func newLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:          "login",
		Short:        "login to YouTube Music",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			binaryPath, err := backend.GetExecutablePath(backend.PythonBacked)
			if err != nil {
				return fmt.Errorf("failed to prepare backend executable: %w", err)
			}

			childCmd := exec.Command(binaryPath, "--login")
			childCmd.Stdin = os.Stdin
			childCmd.Stdout = os.Stdout
			childCmd.Stderr = os.Stderr

			if err := childCmd.Run(); err != nil {
				return fmt.Errorf("login failed: %w", err)
			}

			return nil
		},
	}
}
