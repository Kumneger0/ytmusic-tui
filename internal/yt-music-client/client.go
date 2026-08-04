package ytmusicclient

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	musicpb "github.com/kumneger0/ytmusic-tui/gen"
	"github.com/kumneger0/ytmusic-tui/internal/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func encodeAuthJSON(authJSON string) string {
	authJSON = strings.TrimSpace(authJSON)
	if authJSON == "" {
		return ""
	}
	return base64.StdEncoding.EncodeToString([]byte(authJSON))
}

func authInterceptor(
	ctx context.Context,
	method string,
	req, reply any,
	cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker,
	opts ...grpc.CallOption,
) error {
	if md, ok := metadata.FromOutgoingContext(ctx); !ok || len(md.Get("x-auth-json")) == 0 {
		authJSON := LoadAuthJSON()
		if authJSON != "" {
			ctx = metadata.AppendToOutgoingContext(ctx, "x-auth-json", encodeAuthJSON(authJSON))
		}
	}
	return invoker(ctx, method, req, reply, cc, opts...)
}

func GetYtMusicClient(addr string) (musicpb.MusicServiceClient, *grpc.ClientConn, error) {
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(authInterceptor),
	)
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
	md := metadata.Pairs("x-auth-json", encodeAuthJSON(authJSON))
	return metadata.NewOutgoingContext(ctx, md)
}
