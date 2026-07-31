package cmd

import (
	"fmt"
	"os"
	"os/exec"

	backend "github.com/kumneger0/ytmusic-tui/backend"
	"github.com/spf13/cobra"
)

var supportedBrowsers = []string{
	"chrome",
	"firefox",
	"brave",
	"edge",
	"chromium",
	"opera",
	"opera-gx",
	"vivaldi",
	"librewolf",
	"safari",
}

func newExtractCookieCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "extract-cookie",
		Short: "Extract YouTube Music cookies from a browser using browser-cookie3",
		Long: `Extract YouTube Music authentication cookies directly from a browser's cookie store.
This uses browser-cookie3 to read cookies and saves them as credentials for clispot.

Supported browsers: chrome, firefox, brave, edge, chromium, opera, opera-gx, vivaldi, librewolf, safari

Example:
  clispot extract-cookie --chrome
  clispot extract-cookie --firefox
  clispot extract-cookie --brave`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			var selectedBrowser string
			for _, browser := range supportedBrowsers {
				flagName := browser
				val, err := cmd.Flags().GetBool(flagName)
				if err != nil {
					continue
				}
				if val {
					if selectedBrowser != "" {
						return fmt.Errorf("only one browser can be selected at a time, got both --%s and --%s", selectedBrowser, browser)
					}
					pythonName := browser
					if browser == "opera-gx" {
						pythonName = "opera_gx"
					}
					selectedBrowser = pythonName
				}
			}

			if selectedBrowser == "" {
				return fmt.Errorf("no browser specified. Use one of the browser flags, e.g. --chrome, --firefox, --brave")
			}

			binaryPath, err := backend.GetExecutablePath(backend.PythonBackend)
			if err != nil {
				return fmt.Errorf("failed to prepare backend executable: %w", err)
			}

			childCmd := exec.Command(binaryPath, "--extract-cookie", selectedBrowser)
			childCmd.Stdin = os.Stdin
			childCmd.Stdout = os.Stdout
			childCmd.Stderr = os.Stderr

			if err := childCmd.Run(); err != nil {
				return fmt.Errorf("cookie extraction failed: %w", err)
			}

			return nil
		},
	}

	for _, browser := range supportedBrowsers {
		cmd.Flags().Bool(browser, false, fmt.Sprintf("Extract cookies from %s", browser))
	}

	return cmd
}
