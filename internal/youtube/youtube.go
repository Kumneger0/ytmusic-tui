package youtube

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ebitengine/oto/v3"
	"github.com/kkdai/youtube/v2"
	"github.com/kumneger0/ytmusic-tui/internal/command"
	"github.com/kumneger0/ytmusic-tui/internal/config"
	"github.com/kumneger0/ytmusic-tui/internal/types"
	"github.com/smallnest/ringbuffer"
)

var otoContext *oto.Context
var once sync.Once

func getOtoContext() (*oto.Context, chan struct{}, error) {
	var readyChan chan struct{}
	var ctxErr error
	once.Do(func() {
		ctx, ready, err := oto.NewContext(&oto.NewContextOptions{
			SampleRate:   44100,
			ChannelCount: 2,
			BufferSize:   0,
			Format:       oto.FormatSignedInt16LE,
		})
		readyChan = ready
		ctxErr = err
		otoContext = ctx
	})
	return otoContext, readyChan, ctxErr
}

type CoreDepsPath struct {
	FFmpeg string
}

func SearchAndDownloadMusic(
	ctx context.Context,
	videoID string,
	coreDepsPath *CoreDepsPath,
	getStreamURL func() (*StreamAndDuration, error),
) tea.Cmd {
	return func() tea.Msg {
		if ctx.Err() != nil {
			return nil
		}

		if coreDepsPath == nil {
			return types.SearchAndDownloadMusicMsg{
				Player:  nil,
				VideoID: videoID,
				Err:     errors.New("failed to find necessary dependencies"),
			}
		}

		streamURL, err := getStreamURL()
		if err != nil || streamURL == nil {
			if ctx.Err() != nil {
				return nil
			}
			slog.Error(err.Error())
			return types.SearchAndDownloadMusicMsg{Player: nil, VideoID: videoID, Err: err}
		}

		if ctx.Err() != nil {
			return nil
		}

		appConfig := config.GetConfig()
		logPathName := appConfig.DebugDir
		ffStderr, err := os.Create(filepath.Join(*logPathName, "ffstderr.log"))
		if err != nil {
			slog.Error(err.Error())
			return types.SearchAndDownloadMusicMsg{
				Player:   nil,
				VideoID:  videoID,
				Duration: "",
				Err:      err,
			}
		}

		ff, err := command.ExecCommand(ctx,
			coreDepsPath.FFmpeg,
			"-reconnect", "1",
			"-reconnect_streamed", "1",
			"-reconnect_delay_max", "5",
			"-reconnect_on_network_error", "1",
			"-reconnect_on_http_error", "1",
			"-i", streamURL.URL,
			"-f", "s16le",
			"-ac", "2",
			"-ar", "44100",
			"pipe:1",
		)

		if err != nil {
			_ = ffStderr.Close()
			slog.Error(err.Error())
			return types.SearchAndDownloadMusicMsg{
				Player:   nil,
				VideoID:  videoID,
				Duration: streamURL.Duration,
				Err:      err,
			}
		}

		pr, pw := ringbuffer.New(1024 * 1024 * 5).Pipe()

		ff.Stderr = ffStderr
		ff.Stdout = pw

		if err := ff.Start(); err != nil {
			_ = pw.Close()
			_ = pr.Close()
			if ffStderr != nil {
				_ = ffStderr.Close()
			}
			if ctx.Err() != nil {
				return nil
			}
			return types.SearchAndDownloadMusicMsg{
				Player:   nil,
				VideoID:  videoID,
				Duration: streamURL.Duration,
				Err:      err,
			}
		}

		go func() {
			err := ff.Wait()
			if err != nil {
				slog.Info("ffmpeg exited", "err", err)
				_ = pw.CloseWithError(fmt.Errorf("ffmpeg: %w", err))
			} else {
				_ = pw.Close()
			}
		}()

		otoCtx, ready, err := getOtoContext()
		if err != nil {
			if ff.Process != nil {
				_ = command.KillProcess(ff.Process)
			}
			_ = pw.Close()
			_ = pr.Close()
			if ffStderr != nil {
				_ = ffStderr.Close()
			}
			if ctx.Err() != nil {
				return nil
			}
			return types.SearchAndDownloadMusicMsg{
				Player:   nil,
				Duration: streamURL.Duration,
				VideoID:  videoID,
				Err:      err,
			}
		}
		if ready != nil {
			<-ready
		}

		if ctx.Err() != nil {
			if ff.Process != nil {
				_ = command.KillProcess(ff.Process)
			}
			_ = pw.Close()
			_ = pr.Close()
			if ffStderr != nil {
				_ = ffStderr.Close()
			}
			return nil
		}

		counter := &types.ByteCounterReader{
			R: pr,
		}

		player := otoCtx.NewPlayer(counter)
		player.SetBufferSize(0)
		player.Play()

		var once sync.Once
		cleanup := func() error {
			var closeErr error
			once.Do(func() {
				if ff.Process != nil {
					_ = command.KillProcess(ff.Process)
				}
				_ = pw.CloseWithError(fmt.Errorf("player closed"))
				_ = pr.Close()
				if player != nil {
					player.Pause()
					player.Close()
				}
				if ffStderr != nil {
					_ = ffStderr.Close()
				}
			})
			return closeErr
		}

		return types.SearchAndDownloadMusicMsg{
			Player: &types.Player{
				OtoPlayer:         player,
				ByteCounterReader: counter,
				Close:             cleanup,
			},
			VideoID:  videoID,
			Duration: streamURL.Duration,
			Err:      nil,
		}
	}
}

type StreamAndDuration struct {
	URL      string
	Duration string
}

func GetStreamURLAndDuration(videoID string) (*StreamAndDuration, error) {
	client := youtube.Client{}
	video, err := client.GetVideo(videoID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch video metadata: %w", err)
	}
	formats := video.Formats.WithAudioChannels()
	if len(formats) == 0 {
		return nil, fmt.Errorf("no audio format found for video: %s", videoID)
	}
	formats.Sort()
	streamURL, err := client.GetStreamURL(video, &formats[0])
	if err != nil {
		return nil, fmt.Errorf("failed to get stream url: %w", err)
	}
	durationInSeconds := int64(video.Duration.Seconds())
	return &StreamAndDuration{
		URL:      streamURL,
		Duration: strconv.FormatInt(durationInSeconds, 10),
	}, nil
}
