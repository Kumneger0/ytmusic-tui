package ytmusicclient

import (
	musicpb "github.com/kumneger0/ytmusic-tui/gen"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func GetYtMusicClient(addr string) (musicpb.MusicServiceClient, *grpc.ClientConn, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	client := musicpb.NewMusicServiceClient(conn)
	return client, conn, nil
}
