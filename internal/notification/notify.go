package notification

import (
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/gen2brain/beeep"
	"github.com/kumneger0/ytmusic-tui/assets"
)

var (
	once sync.Once
)

func getAppLogo() *[]byte {
	var appLogo *[]byte
	logo, err := assets.Assets.ReadFile("logo.png")
	if err != nil {
		slog.Error(err.Error())
		return nil
	}
	appLogo = &logo
	return appLogo
}

func getAppIconPath() string {
	path := filepath.Join(os.TempDir(), "ytmusic-tui-icon.png")
	once.Do(func() {
		logoPNG := getAppLogo()
		if logoPNG == nil {
			slog.Error("logo.png not found")
			return
		}

		if err := os.WriteFile(path, *logoPNG, 0o644); err != nil {
			slog.Error(err.Error())
			return
		}
	})
	return path
}

func Notify(title string, message string) {
	beeep.AppName = "YTMusic-TUI"
	logo := getAppIconPath()
	shortenMessage := trimWithEllipsis(message, 300)
	err := beeep.Notify(title, shortenMessage, logo)
	if err != nil {
		slog.Error(err.Error())
	}
}

func trimWithEllipsis(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
