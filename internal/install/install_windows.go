//go:build windows

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
	case commandExists("winget"):
		return []types.InstallStep{
			{
				Command: "winget",
				Args: []string{
					"install",
					"--id",
					"Gyan.FFmpeg",
					"-e",
					"--accept-package-agreements",
					"--accept-source-agreements",
				},
			},
		}, nil

	case commandExists("choco"):
		return []types.InstallStep{
			{
				Command: "choco",
				Args: []string{
					"install",
					"-y",
					"ffmpeg",
				},
			},
		}, nil

	case commandExists("scoop"):
		return []types.InstallStep{
			{
				Command: "scoop",
				Args: []string{
					"install",
					"ffmpeg",
				},
			},
		}, nil

	default:
		return nil, errors.New(
			"no supported package manager found. Install FFmpeg manually or install winget, Chocolatey, or Scoop",
		)
	}
}
