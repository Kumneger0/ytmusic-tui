package cmd

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/kumneger0/ytmusic-tui/internal/command"
	"github.com/kumneger0/ytmusic-tui/internal/install"
	"github.com/spf13/cobra"
)

func installDeps() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "install",
		Short:        "install missing dependencies",
		SilenceUsage: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			var confirm bool
			fmt.Println("ytmusic-tui can install FFmpeg automatically.This installation may require administrator privileges")
			err := huh.NewConfirm().
				Title("Continue?").
				Affirmative("Yes").
				Negative("No").
				Value(&confirm).
				Run()

			if err != nil {
				return err
			}

			if !confirm {
				fmt.Println("Exiting Goodbye...")
				return nil
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			ffmpegInstallCmd, err := install.FFmpegInstallCommand()
			if err != nil {
				log.Fatal(err)
			}
			for _, step := range ffmpegInstallCmd {
				installCmd, err := command.ExecCommand(ctx, step.Command, step.Args...)
				if err != nil {
					log.Fatal(err)
				}
				installCmd.Stdout = os.Stdout
				installCmd.Stdin = os.Stdin
				installCmd.Stderr = os.Stderr
				if err := installCmd.Run(); err != nil {
					log.Fatal(err)
				}
			}

			return nil
		},
	}
	return cmd
}
