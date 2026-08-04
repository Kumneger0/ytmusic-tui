package ytmusicclient

import (
	"context"
	"os"
	"path/filepath"
	"runtime"

	musicpb "github.com/kumneger0/ytmusic-tui/gen"
	"github.com/kumneger0/ytmusic-tui/internal/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func GetYtMusicClient(addr string) (musicpb.MusicServiceClient, *grpc.ClientConn, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	client := musicpb.NewMusicServiceClient(conn)
	return client, conn, nil
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

func WithAuthContext(ctx context.Context, authJSON string) context.Context {
	if authJSON == "" {
		authJSON = LoadAuthJSON()
	}
	if authJSON == "" {
		return ctx
	}
	md := metadata.Pairs("x-auth-json", authJSON)
	return metadata.NewOutgoingContext(ctx, md)
}
