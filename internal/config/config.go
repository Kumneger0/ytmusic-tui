package config

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
)

type YtDlpArgs struct {
	CookiesFromBrowser *string `json:"cookies-from-browser"`
	Cookies            *string `json:"cookies"`
}

type Config struct {
	DebugDir      *string    `json:"debug-dir"`
	CacheDisabled bool       `json:"disable-cache"`
	CacheDir      *string    `json:"cache-dir"`
	YtDlpArgs     *YtDlpArgs `json:"yt-dlp-args"`
	HeadlessMode  bool       `json:"headless-mode"`
	SkipOnNoMatch bool       `json:"skip-on-no-match"`
}

var userConfigDir = os.UserConfigDir
var userCacheDir = os.UserCacheDir
var userHomeDir = os.UserHomeDir

func GetConfigDir(goos string) string {
	configDir, err := userConfigDir()
	if err != nil {
		if goos == "windows" {
			return filepath.Join(os.Getenv("APPDATA"), "ytmusic-tui")
		}
		xdgConfig := os.Getenv("XDG_CONFIG_HOME")
		if xdgConfig != "" {
			return filepath.Join(xdgConfig, "ytmusic-tui")
		}
		if goos == "darwin" {
			return filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "ytmusic-tui")
		}
		return filepath.Join(os.Getenv("HOME"), ".config", "ytmusic-tui")
	}
	return filepath.Join(configDir, "ytmusic-tui")
}

func GetStateDir(goos string) string {
	if goos == "windows" {
		return filepath.Join(os.Getenv("APPDATA"), "ytmusic-tui")
	}
	if goos == "darwin" {
		return filepath.Join(os.Getenv("HOME"), "Library", "State", "ytmusic-tui")
	}
	stateDir := os.Getenv("XDG_STATE_HOME")
	if stateDir == "" {
		homeDir, _ := userHomeDir()
		stateDir = filepath.Join(homeDir, ".local", "state")
	}
	return filepath.Join(stateDir, "ytmusic-tui")
}

func GetCacheDir(goos string) string {
	cacheDir, err := userCacheDir()
	if err != nil {
		homeDir, _ := userHomeDir()
		if goos == "windows" {
			return filepath.Join(os.Getenv("LOCALAPPDATA"), "ytmusic-tui")
		}
		if goos == "darwin" {
			return filepath.Join(os.Getenv("HOME"), "Library", "Caches", "ytmusic-tui")
		}
		return filepath.Join(homeDir, ".cache", "ytmusic-tui")
	}
	return filepath.Join(cacheDir, "ytmusic-tui")
}

func GetDefaultConfig(goos string) *Config {
	defaultDebugDir := filepath.Join(GetStateDir(goos), "logs")
	defaultCacheDir := GetCacheDir(goos)
	return &Config{
		DebugDir:      &defaultDebugDir,
		CacheDisabled: true,
		CacheDir:      &defaultCacheDir,
		YtDlpArgs:     &YtDlpArgs{},
		HeadlessMode:  false,
		SkipOnNoMatch: true,
	}
}

func GetUserConfig(goos string) *Config {
	configPath := filepath.Join(GetConfigDir(goos), "config.json")
	fileStat, err := os.Stat(configPath)
	if err != nil {
		return GetDefaultConfig(goos)
	}
	if fileStat.IsDir() {
		slog.Error("User config is a directory", "path", configPath)
		return GetDefaultConfig(goos)
	}
	configFile, err := os.ReadFile(configPath)
	if err != nil {
		slog.Error("Failed to read user config", "err", err)
		return GetDefaultConfig(goos)
	}

	config := GetDefaultConfig(goos)
	err = json.Unmarshal(configFile, config)
	if err != nil {
		slog.Error("Failed to unmarshal user config", "err", err)
		return GetDefaultConfig(goos)
	}
	return config
}

var AppConfig Config

func SetConfig(config *Config) {
	AppConfig = *config
}

func GetConfig() *Config {
	return &AppConfig
}
