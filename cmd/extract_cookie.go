package cmd

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	musicpb "github.com/kumneger0/ytmusic-tui/gen"
	"github.com/kumneger0/ytmusic-tui/internal/config"
	"github.com/kumneger0/ytmusic-tui/internal/cookie"
	ytMusicClient "github.com/kumneger0/ytmusic-tui/internal/yt-music-client"

	"connectrpc.com/connect"
	"github.com/browserutils/kooky"
	_ "github.com/browserutils/kooky/browser/all"
	"github.com/spf13/cobra"
)

var supportedBrowsers = []string{"chrome", "firefox", "safari"}

const youtubeOrigin = "https://music.youtube.com"

func extractFromBrowserStore(ctx context.Context, targetBrowser string) ([]*kooky.Cookie, error) {
	stores := kooky.FindAllCookieStores(ctx)
	var cookies []*kooky.Cookie
	for _, store := range stores {
		if strings.Contains(strings.ToLower(store.Browser()), strings.ToLower(targetBrowser)) {
			seq := store.TraverseCookies(kooky.Valid, kooky.DomainHasSuffix(`.youtube.com`))
			for c, err := range seq {
				if err == nil && c != nil {
					cookies = append(cookies, c)
				}
			}
		}
		_ = store.Close()
	}
	return cookies, nil
}

func buildAuthJSON(cookies []*kooky.Cookie) (string, error) {
	var parts []string
	seen := make(map[string]bool)
	for _, c := range cookies {
		if c.Name == "" || seen[c.Name] {
			continue
		}
		seen[c.Name] = true
		parts = append(parts, fmt.Sprintf("%s=%s", c.Name, c.Value))
	}
	header := strings.Join(parts, "; ")
	var sapisid string
	for _, c := range cookies {
		if c.Name == "SAPISID" {
			sapisid = c.Value
			break
		}
		if c.Name == "__Secure-3PAPISID" && sapisid == "" {
			sapisid = c.Value
		}
	}
	if sapisid == "" {
		return "", fmt.Errorf("could not find SAPISID or __Secure-3PAPISID cookie; make sure you are logged into YouTube Music")
	}
	ts := time.Now().Unix()
	hash := fmt.Sprintf("%d %s %s", ts, sapisid, youtubeOrigin)
	sha := sha1.Sum([]byte(hash))
	auth := fmt.Sprintf("SAPISIDHASH %d_%x", ts, sha)
	headers := map[string]string{"Accept": "*/*", "Authorization": auth, "Content-Type": "application/json", "X-Goog-AuthUser": "0", "x-origin": youtubeOrigin, "Cookie": header}
	data, err := json.MarshalIndent(headers, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal auth JSON: %w", err)
	}
	return string(data), nil
}

func saveAuthJSON(authJSON string) (string, error) {
	dir := config.GetConfigDir(runtime.GOOS)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}
	path := filepath.Join(dir, "browser.json")
	if err := os.WriteFile(path, []byte(authJSON), 0600); err != nil {
		return "", fmt.Errorf("failed to write browser.json: %w", err)
	}
	if _, err := cookie.SaveCookieFile(authJSON); err != nil {
		slog.Error("failed to write cookie.txt", "err", err)
	}
	return path, nil
}

func newExtractCookieCmd(serverURL string) *cobra.Command {
	cmd := &cobra.Command{Use: "extract-cookies",
		Short:        "Extract YouTube Music cookies and verify",
		Long:         `Extract YouTube Music authentication cookies from supported browsers, verify via API, and store locally.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			var authJSON string
			for _, b := range supportedBrowsers {
				cookies, err := extractFromBrowserStore(ctx, b)
				if err != nil || len(cookies) == 0 {
					continue
				}
				jsonStr, err := buildAuthJSON(cookies)
				if err != nil {
					continue
				}

				client := ytMusicClient.GetYtMusicClient(serverURL)
				rpcCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				resp, err := client.Login(rpcCtx, connect.NewRequest(&musicpb.LoginRequest{AuthJson: jsonStr}))
				if err != nil || !resp.Msg.Authenticated {
					continue
				}
				path, err := saveAuthJSON(jsonStr)
				if err != nil {
					return fmt.Errorf("failed to save credentials: %w", err)
				}
				fmt.Printf("Authenticated as: %s (%s)\n", resp.Msg.UserName, b)
				fmt.Printf("Saved credentials to: %s\n", path)
				authJSON = jsonStr
				break
			}
			if authJSON == "" {
				return fmt.Errorf("could not find valid YouTube Music cookies in any supported browser")
			}

			fmt.Println("Verification completed successfully.")
			return nil
		},
	}
	return cmd
}
