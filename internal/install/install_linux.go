//go:build linux

package install

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/kumneger0/ytmusic-tui/internal/types"
)

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func privilegePrefix() (string, error) {
	if os.Getuid() == 0 {
		return "", nil
	}
	for _, wrapper := range []string{"sudo", "doas"} {
		if commandExists(wrapper) {
			return wrapper, nil
		}
	}
	return "", fmt.Errorf("root privileges required but neither sudo nor doas was found in PATH")
}

func privilegedStep(prefix, cmd string, args ...string) types.InstallStep {
	if prefix == "" {
		return types.InstallStep{Command: cmd, Args: args}
	}
	return types.InstallStep{Command: prefix, Args: append([]string{cmd}, args...)}
}

func FFmpegInstallCommand() ([]types.InstallStep, error) {
	prefix, err := privilegePrefix()
	if err != nil {
		return nil, err
	}

	switch {
	case commandExists("apt"):
		return []types.InstallStep{
			privilegedStep(prefix, "apt", "update"),
			privilegedStep(prefix, "apt", "install", "-y", "ffmpeg"),
		}, nil

	case commandExists("dnf"):
		return []types.InstallStep{
			privilegedStep(prefix, "dnf", "install", "-y", "ffmpeg"),
		}, nil

	case commandExists("yum"):
		return []types.InstallStep{
			privilegedStep(prefix, "yum", "install", "-y", "ffmpeg"),
		}, nil

	case commandExists("pacman"):
		return []types.InstallStep{
			privilegedStep(prefix, "pacman", "-S", "--noconfirm", "ffmpeg"),
		}, nil

	case commandExists("zypper"):
		return []types.InstallStep{
			privilegedStep(prefix, "zypper", "--non-interactive", "install", "ffmpeg"),
		}, nil

	case commandExists("apk"):
		return []types.InstallStep{
			privilegedStep(prefix, "apk", "add", "ffmpeg"),
		}, nil

	case commandExists("xbps-install"):
		return []types.InstallStep{
			privilegedStep(prefix, "xbps-install", "-Sy", "ffmpeg"),
		}, nil

	case commandExists("emerge"):
		return []types.InstallStep{
			privilegedStep(prefix, "emerge", "media-video/ffmpeg"),
		}, nil

	case commandExists("nix"):
		return []types.InstallStep{
			{
				Command: "nix",
				Args:    []string{"profile", "install", "nixpkgs#ffmpeg"},
			},
		}, nil

	default:
		return nil, errors.New("unsupported Linux distribution")
	}
}
