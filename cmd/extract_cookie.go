package cmd

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	musicpb "github.com/kumneger0/ytmusic-tui/gen"
	"github.com/kumneger0/ytmusic-tui/internal/config"
	ytMusicClient "github.com/kumneger0/ytmusic-tui/internal/yt-music-client"

	"connectrpc.com/connect"
	"github.com/browserutils/kooky"
	_ "github.com/browserutils/kooky/browser/all" // register cookie store finders!
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

const youtubeOrigin = "https://music.youtube.com"

func buildAuthJSON(cookies []*kooky.Cookie) (string, error) {
	var cookieParts []string
	seen := make(map[string]bool)
	for _, c := range cookies {
		if c.Name == "" || seen[c.Name] {
			continue
		}
		seen[c.Name] = true
		cookieParts = append(cookieParts, fmt.Sprintf("%s=%s", c.Name, c.Value))
	}
	cookieHeader := strings.Join(cookieParts, "; ")

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

	timestamp := time.Now().Unix()
	hashInput := fmt.Sprintf("%d %s %s", timestamp, sapisid, youtubeOrigin)
	sha1Hash := sha1.Sum([]byte(hashInput))
	authorization := fmt.Sprintf("SAPISIDHASH %d_%x", timestamp, sha1Hash)

	headers := map[string]string{
		"Accept":          "*/*",
		"Authorization":   authorization,
		"Content-Type":    "application/json",
		"X-Goog-AuthUser": "0",
		"x-origin":        youtubeOrigin,
		"Cookie":          cookieHeader,
	}

	data, err := json.MarshalIndent(headers, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal auth JSON: %w", err)
	}
	return string(data), nil
}

func saveAuthJSON(authJSON string) (string, error) {
	configDir := config.GetConfigDir(runtime.GOOS)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}
	path := filepath.Join(configDir, "browser.json")
	if err := os.WriteFile(path, []byte(authJSON), 0600); err != nil {
		return "", fmt.Errorf("failed to write browser.json: %w", err)
	}
	return path, nil
}

func extractCookiesForBrowser(ctx context.Context, browserName string) ([]*kooky.Cookie, error) {
	targetBrowser := strings.ReplaceAll(strings.ToLower(browserName), "-", "_")
	stores := kooky.FindAllCookieStores(ctx)
	var cookies []*kooky.Cookie

	for _, store := range stores {
		storeBrowser := strings.ReplaceAll(strings.ToLower(store.Browser()), "-", "_")
		if strings.Contains(storeBrowser, targetBrowser) || strings.Contains(targetBrowser, storeBrowser) {
			seq := store.TraverseCookies(kooky.Valid, kooky.DomainHasSuffix(`.youtube.com`))
			for cookie, err := range seq {
				if err == nil && cookie != nil {
					cookies = append(cookies, cookie)
				}
			}
			_ = store.Close()
		}
	}

	if len(cookies) == 0 {
		cookiesSeq := kooky.TraverseCookies(
			ctx,
			kooky.Valid,
			kooky.DomainHasSuffix(`.youtube.com`),
		).OnlyCookies()

		for cookie := range cookiesSeq {
			cookies = append(cookies, cookie)
		}
	}

	return cookies, nil
}

func newExtractCookieCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "extract-cookie",
		Short: "Extract YouTube Music cookies from a browser and verify with the API",
		Long: `Extract YouTube Music authentication cookies directly from a browser's cookie store.
Uses kooky to read cookies, constructs auth headers, verifies them against
the YouTube Music API via the Python backend, and saves credentials locally.

Supported browsers: chrome, firefox, brave, edge, chromium, opera, opera-gx, vivaldi, librewolf, safari

Example:
  clispot extract-cookie --chrome
  clispot extract-cookie --firefox
  clispot extract-cookie --brave`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			var selectedBrowser string
			for _, browser := range supportedBrowsers {
				val, err := cmd.Flags().GetBool(browser)
				if err != nil {
					continue
				}
				if val {
					if selectedBrowser != "" {
						return fmt.Errorf("only one browser can be selected at a time, got both --%s and --%s", selectedBrowser, browser)
					}
					selectedBrowser = browser
				}
			}

			if selectedBrowser == "" {
				return fmt.Errorf("no browser specified. Use one of the browser flags, e.g. --chrome, --firefox, --brave")
			}

			fmt.Printf("  Extracting cookies from %s...\n", selectedBrowser)

			ctx := context.TODO()
			cookies, err := extractCookiesForBrowser(ctx, selectedBrowser)
			if err != nil {
				return fmt.Errorf("failed to extract cookies: %w", err)
			}

			if len(cookies) == 0 {
				return fmt.Errorf("no YouTube cookies found in %s. Make sure you are logged into YouTube Music in this browser", selectedBrowser)
			}
			fmt.Printf("Found %d YouTube cookie(s)\n", len(cookies))

			authJSON, err := buildAuthJSON(cookies)
			if err != nil {
				return fmt.Errorf("failed to build auth headers: %w", err)
			}

			fmt.Println(" Verifying credentials with YouTube Music API...")

			client := ytMusicClient.GetYtMusicClient("http://localhost:8080")

			rpcCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			resp, err := client.Login(rpcCtx, connect.NewRequest(&musicpb.LoginRequest{
				AuthJson: authJSON,
			}))
			if err != nil {
				return fmt.Errorf("verification RPC failed: %w", err)
			}

			if !resp.Msg.Authenticated {
				return fmt.Errorf("authentication failed: %s", resp.Msg.Error)
			}

			savedPath, err := saveAuthJSON(authJSON)
			if err != nil {
				return fmt.Errorf("credentials verified but failed to save: %w", err)
			}

			userName := resp.Msg.UserName
			if userName != "" {
				fmt.Printf("  ✓ Authenticated as: %s\n", userName)
			} else {
				fmt.Println("  ✓ Authentication successful!")
			}
			fmt.Printf("  ✓ Saved credentials to: %s\n", savedPath)
			fmt.Println("  You can now run clispot normally.")

			healthReq := connect.NewRequest(&musicpb.HealthCheckRequest{})
			healthReq.Header().Set("x-auth-json", authJSON)
			healthResp, err := client.HealthCheck(rpcCtx, healthReq)
			if err != nil || !healthResp.Msg.Ok {
				fmt.Println("  ⚠ Warning: health check with saved credentials failed, but credentials were saved.")
			}

			return nil
		},
	}

	for _, browser := range supportedBrowsers {
		cmd.Flags().Bool(browser, false, fmt.Sprintf("Extract cookies from %s", browser))
	}

	return cmd
}
