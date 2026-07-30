package backend

import (
	"crypto/sha256"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// GetExecutablePath extracts the embedded Python backend executable to the user cache directory
// and verifies its SHA256 integrity hash.
func GetExecutablePath(fs embed.FS) (string, error) {
	data, err := fs.ReadFile(embedFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read embedded backend (%s): %w", embedFilePath, err)
	}

	hash := sha256.Sum256(data)
	actualHash := fmt.Sprintf("%x", hash)

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}

	appDir := filepath.Join(cacheDir, "yt-music-tui")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return "", err
	}

	execName := "backend"
	if runtime.GOOS == "windows" {
		execName = "backend.exe"
	}
	binaryPath := filepath.Join(appDir, execName)

	file, err := os.ReadFile(binaryPath)
	if err != nil {
		if os.IsNotExist(err) {
			if err := writeBinaryToCacheFolder(data, binaryPath); err != nil {
				return "", err
			}
			return binaryPath, nil
		}
		return "", err
	}

	expectedHash := fmt.Sprintf("%x", sha256.Sum256(file))
	if actualHash != expectedHash {
		if err := os.Remove(binaryPath); err != nil {
			return "", fmt.Errorf("backend binary integrity check failed: %w", err)
		}
		if err := writeBinaryToCacheFolder(data, binaryPath); err != nil {
			return "", err
		}
	}

	return binaryPath, nil
}

func StartBackend(fs embed.FS) (*exec.Cmd, error) {
	binaryPath, err := GetExecutablePath(fs)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(binaryPath)
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return cmd, nil
}

func writeBinaryToCacheFolder(data []byte, binaryPath string) error {
	if err := os.WriteFile(binaryPath, data, 0755); err != nil {
		return err
	}

	return os.Chmod(binaryPath, 0755)
}
