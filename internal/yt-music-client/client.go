package ytmusicclient

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"connectrpc.com/connect"
	"github.com/kumneger0/ytmusic-tui/gen/genconnect"
	"github.com/kumneger0/ytmusic-tui/internal/config"
)

func encodeAuthJSON(authJSON string) string {
	authJSON = strings.TrimSpace(authJSON)
	if authJSON == "" {
		return ""
	}
	return base64.StdEncoding.EncodeToString([]byte(authJSON))
}

func newAuthInterceptor() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if req.Header().Get("x-auth-json") == "" {
				authJSON := LoadAuthJSON()
				if authJSON != "" {
					req.Header().Set("x-auth-json", encodeAuthJSON(authJSON))
				}
			}
			return next(ctx, req)
		}
	}
}

func GetYtMusicClient(addr string) genconnect.MusicServiceClient {
	cleanAddr := strings.TrimSpace(addr)
	if !strings.HasPrefix(cleanAddr, "http://") && !strings.HasPrefix(cleanAddr, "https://") {
		if strings.HasSuffix(cleanAddr, ":443") {
			cleanAddr = "https://" + cleanAddr
		} else {
			cleanAddr = "http://" + cleanAddr
		}
	}

	httpClient := &http.Client{}
	if strings.HasPrefix(cleanAddr, "https://") {
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{},
		}
	}

	client := genconnect.NewMusicServiceClient(
		httpClient,
		cleanAddr,
		connect.WithInterceptors(newAuthInterceptor()),
	)
	return client
}

func LoadAuthJSON() string {
	configDir := config.GetConfigDir(runtime.GOOS)
	path := filepath.Join(configDir, "browser.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}
