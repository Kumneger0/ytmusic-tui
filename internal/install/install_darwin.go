//go:build darwin

package install

import (
	"errors"
	"os/exec"

	"github.com/kumneger0/ytmusic-tui/internal/types"
)

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func FFmpegInstallCommand() ([]types.InstallStep, error) {
	switch {
	case commandExists("brew"):
		return []types.InstallStep{
			{
				Command: "brew",
				Args:    []string{"install", "ffmpeg"},
			},
		}, nil

	default:
		return nil, errors.New(
			"Homebrew is not installed. Please install Homebrew first: https://brew.sh",
		)
	}
}
