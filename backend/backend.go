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

var userCacheDir = os.UserCacheDir

// GetExecutablePath extracts the embedded Python backend executable to the user cache directory
// and verifies its SHA256 integrity hash.
func GetExecutablePath(fs embed.FS) (string, error) {
	data, err := fs.ReadFile(embedFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read embedded backend (%s): %w", embedFilePath, err)
	}

	hash := sha256.Sum256(data)
	actualHash := fmt.Sprintf("%x", hash)

	cacheDir, err := userCacheDir()
	if err != nil {
		return "", err
	}

	appDir := filepath.Join(cacheDir, "ytmusic-tui")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return "", err
	}

	execName := fmt.Sprintf("backend-%s", actualHash[:12])
	if runtime.GOOS == "windows" {
		execName = fmt.Sprintf("backend-%s.exe", actualHash[:12])
	}
	binaryPath := filepath.Join(appDir, execName)

	file, err := os.ReadFile(binaryPath)
	if err == nil {
		expectedHash := fmt.Sprintf("%x", sha256.Sum256(file))
		if actualHash == expectedHash {
			return binaryPath, nil
		}
	}

	if err := writeBinaryAtomically(data, appDir, binaryPath, actualHash); err != nil {
		return "", err
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

func writeBinaryAtomically(data []byte, appDir, binaryPath, actualHash string) error {
	tmpFile, err := os.CreateTemp(appDir, "backend-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file for backend binary: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write backend binary to temp file: %w", err)
	}
	if err := tmpFile.Chmod(0755); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to set backend binary permissions: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close backend binary temp file: %w", err)
	}

	if err := os.Rename(tmpPath, binaryPath); err != nil {
		if file, readErr := os.ReadFile(binaryPath); readErr == nil {
			if fmt.Sprintf("%x", sha256.Sum256(file)) == actualHash {
				return nil
			}
		}
		return fmt.Errorf("failed to rename temp file to binary path: %w", err)
	}

	return nil
}
