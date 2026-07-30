package config

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetConfigDir_LinuxFallback(t *testing.T) {
	original := userConfigDir
	t.Cleanup(func() { userConfigDir = original })

	userConfigDir = func() (string, error) {
		return "/home/kune/.config", nil
	}
	dir := GetConfigDir("linux")
	want := "/home/kune/.config/ytmusic-tui"
	assert.Equal(t, want, dir)
}

func TestGetConfigDir_WindowsFallback(t *testing.T) {
	original := userConfigDir
	t.Cleanup(func() { userConfigDir = original })

	userConfigDir = func() (string, error) {
		return "", errors.New("boom")
	}

	want := filepath.Join(
		`C:\Users\kune\AppData\Roaming`,
		"ytmusic-tui",
	)

	t.Setenv("APPDATA", `C:\Users\kune\AppData\Roaming`)
	dir := GetConfigDir("windows")
	assert.Equal(t, want, dir)
}

func TestGetConfigDir_DarwinFallback(t *testing.T) {
	original := userConfigDir
	t.Cleanup(func() { userConfigDir = original })

	userConfigDir = func() (string, error) {
		return "/Users/kune/Library/Application Support", nil
	}

	dir := GetConfigDir("darwin")
	want := filepath.Join(
		`/Users/kune/Library/Application Support`,
		"ytmusic-tui",
	)
	assert.Equal(t, want, dir)
}

func TestGetConfigDir_WithCustomConfigDir_Linux(t *testing.T) {
	original := userConfigDir
	t.Cleanup(func() { userConfigDir = original })

	userConfigDir = func() (string, error) {
		return "", errors.New("boom")
	}
	t.Setenv("XDG_CONFIG_HOME", "/home/kune/.custom-config")
	t.Setenv("HOME", "/home/kune")
	dir := GetConfigDir("linux")
	want := "/home/kune/.custom-config/ytmusic-tui"
	assert.Equal(t, want, dir)
}

func TestGetConfigDir_WithCustomConfigDir_Darwin(t *testing.T) {
	original := userConfigDir
	t.Cleanup(func() { userConfigDir = original })

	userConfigDir = func() (string, error) {
		return "", errors.New("boom")
	}
	t.Setenv("XDG_CONFIG_HOME", "/home/kune/.custom-config")
	t.Setenv("HOME", "/home/kune")
	dir := GetConfigDir("darwin")
	want := "/home/kune/.custom-config/ytmusic-tui"
	assert.Equal(t, want, dir)
}

func TestGetConfigDir_WithCustomConfigDir_Windows(t *testing.T) {
	original := userConfigDir
	t.Cleanup(func() { userConfigDir = original })

	userConfigDir = func() (string, error) {
		return "", errors.New("boom")
	}
	t.Setenv("APPDATA", `C:\Users\kune\AppData\Roaming`)
	dir := GetConfigDir("windows")
	want := filepath.Join(
		`C:\Users\kune\AppData\Roaming`,
		"ytmusic-tui",
	)
	assert.Equal(t, want, dir)
}

func TestGetStateDir_LinuxFallback(t *testing.T) {
	original := userHomeDir
	t.Cleanup(func() { userHomeDir = original })

	userHomeDir = func() (string, error) {
		return "/home/kune", nil
	}
	t.Setenv("HOME", "/home/kune")
	dir := GetStateDir("linux")
	want := "/home/kune/.local/state/ytmusic-tui"
	assert.Equal(t, want, dir)
}

func TestGetStateDir_WindowsFallback(t *testing.T) {
	original := userConfigDir
	t.Cleanup(func() { userConfigDir = original })

	userConfigDir = func() (string, error) {
		return "", errors.New("boom")
	}
	t.Setenv("APPDATA", `C:\Users\kune\AppData\Local`)
	dir := GetStateDir("windows")
	want := filepath.Join(
		`C:\Users\kune\AppData\Local`,
		"ytmusic-tui",
	)
	assert.Equal(t, want, dir)
}

func TestGetStateDir_DarwinFallback(t *testing.T) {
	original := userHomeDir
	t.Cleanup(func() { userHomeDir = original })

	userHomeDir = func() (string, error) {
		return "/Users/kune", nil
	}
	t.Setenv("HOME", "/Users/kune")
	dir := GetStateDir("darwin")
	want := filepath.Join(
		`/Users/kune/Library/State`,
		"ytmusic-tui",
	)
	assert.Equal(t, want, dir)
}

func TestGetStateDir_WithCustomStateDir_Linux(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "/home/kune/.custom-state")
	dir := GetStateDir("linux")
	want := filepath.Join(
		`/home/kune/.custom-state`,
		"ytmusic-tui",
	)
	assert.Equal(t, want, dir)
}

func TestGetStateDir_WithCustomStateDir_Darwin(t *testing.T) {
	t.Setenv("HOME", "/Users/kune")
	t.Setenv("XDG_STATE_HOME", "/Users/kune/.custom-state")
	dir := GetStateDir("darwin")
	want := filepath.Join(
		`/Users/kune/Library/State`,
		"ytmusic-tui",
	)
	assert.Equal(t, want, dir)
}

func TestGetStateDir_WithCustomStateDir_Windows(t *testing.T) {
	original := userConfigDir
	t.Cleanup(func() { userConfigDir = original })

	userConfigDir = func() (string, error) {
		return "", errors.New("boom")
	}
	t.Setenv("APPDATA", `C:\Users\kune\AppData\Local`)
	dir := GetStateDir("windows")
	want := filepath.Join(
		`C:\Users\kune\AppData\Local`,
		"ytmusic-tui",
	)
	assert.Equal(t, want, dir)
}

func TestGetCacheDir_LinuxFallback(t *testing.T) {
	original := userCacheDir
	t.Cleanup(func() { userCacheDir = original })

	userCacheDir = func() (string, error) {
		return "", errors.New("boom")
	}

	originalHome := userHomeDir
	t.Cleanup(func() { userHomeDir = originalHome })
	userHomeDir = func() (string, error) {
		return "/home/kune", nil
	}
	t.Setenv("HOME", "/home/kune")
	dir := GetCacheDir("linux")
	want := filepath.Join(
		`/home/kune/.cache`,
		"ytmusic-tui",
	)
	assert.Equal(t, want, dir)
}

func TestGetCacheDir_WindowsFallback(t *testing.T) {
	original := userCacheDir
	t.Cleanup(func() { userCacheDir = original })

	userCacheDir = func() (string, error) {
		return "", errors.New("boom")
	}
	t.Setenv("LOCALAPPDATA", `C:\Users\kune\AppData\Local`)
	dir := GetCacheDir("windows")
	want := filepath.Join(
		`C:\Users\kune\AppData\Local`,
		"ytmusic-tui",
	)
	assert.Equal(t, want, dir)
}

func TestGetCacheDir_DarwinFallback(t *testing.T) {
	original := userCacheDir
	t.Cleanup(func() { userCacheDir = original })

	userCacheDir = func() (string, error) {
		return "", errors.New("boom")
	}

	originalHome := userHomeDir
	t.Cleanup(func() { userHomeDir = originalHome })
	userHomeDir = func() (string, error) {
		return "/Users/kune", nil
	}
	t.Setenv("HOME", "/Users/kune")
	dir := GetCacheDir("darwin")
	want := filepath.Join(
		`/Users/kune/Library/Caches`,
		"ytmusic-tui",
	)
	assert.Equal(t, want, dir)
}
