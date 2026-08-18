package youtube

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ebitengine/oto/v3"
	"github.com/kumneger0/ytmusic-tui/internal/command"
	"github.com/kumneger0/ytmusic-tui/internal/config"
	"github.com/kumneger0/ytmusic-tui/internal/cookie"
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
	YtDlp  string
}

func SearchAndDownloadMusic(
	ctx context.Context,
	videoID string,
	coreDepsPath *CoreDepsPath,
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

		streamURL, err := GetStreamURLAndDuration(ctx, videoID, coreDepsPath.YtDlp)
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

		var headersBuf strings.Builder
		if streamURL.HTTPHeaders != nil {
			for k, v := range streamURL.HTTPHeaders {
				fmt.Fprintf(&headersBuf, "%s: %s\r\n", k, v)
			}
		}

		ffArgs := []string{
			"-reconnect", "1",
			"-reconnect_streamed", "1",
			"-reconnect_delay_max", "5",
			"-reconnect_on_network_error", "1",
			"-reconnect_on_http_error", "1",
		}
		if h := headersBuf.String(); h != "" {
			ffArgs = append(ffArgs, "-headers", h)
		}
		ffArgs = append(ffArgs,
			"-i", streamURL.URL,
			"-f", "s16le",
			"-ac", "2",
			"-ar", "44100",
			"pipe:1",
		)

		ff, err := command.ExecCommand(ctx, coreDepsPath.FFmpeg, ffArgs...)

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
	URL         string
	Duration    string
	HTTPHeaders map[string]string
}

func GetStreamURLAndDuration(ctx context.Context, videoID string, ytdlpPath string) (*StreamAndDuration, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	appConfig := config.GetConfig()
	logPathName := appConfig.DebugDir
	ytDlpError, err := os.Create(filepath.Join(*logPathName, "yt-dlp-error.log"))

	cookiePath := cookie.EnsureCookieFile()
	targetURL := "https://www.youtube.com/watch?v=" + videoID

	args := []string{"-j", "--no-warnings"}
	if cookiePath != "" {
		args = append(args, "--cookies", cookiePath)
	}
	args = append(args, "-f", "ba[acodec=opus][tbr<=300]/ba[tbr<=300]/ba/b", targetURL)

	cmd, err := command.ExecCommand(timeoutCtx, ytdlpPath, args...)
	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}
	cmd.Stderr = ytDlpError
	outBytes, err := cmd.Output()
	if err != nil {
		slog.Error(err.Error())
		fallbackArgs := []string{"-j", "--no-warnings", "-f", "ba[acodec=opus][tbr<=300]/ba[tbr<=300]/ba/b"}
		if cookiePath != "" {
			fallbackArgs = append(fallbackArgs, "--cookies", cookiePath)
		}
		fallbackArgs = append(fallbackArgs, targetURL)
		fallbackCtx, fallbackCancel := context.WithTimeout(ctx, 45*time.Second)
		defer fallbackCancel()
		cmd, err = command.ExecCommand(fallbackCtx, ytdlpPath, fallbackArgs...)
		if err != nil {
			slog.Error(err.Error())
			return nil, err
		}
		cmd.Stderr = ytDlpError
		outBytes, err = cmd.Output()
		if err != nil {
			slog.Error(err.Error())
			return nil, fmt.Errorf("failed to fetch metadata using yt-dlp: %w", err)
		}
	}

	var data struct {
		URL         string            `json:"url"`
		Duration    float64           `json:"duration"`
		HTTPHeaders map[string]string `json:"http_headers"`
	}
	if err := json.Unmarshal(outBytes, &data); err != nil {
		slog.Error(err.Error())
		return nil, fmt.Errorf("failed to parse yt-dlp metadata: %w", err)
	}

	if data.URL == "" {
		err := fmt.Errorf("empty stream url returned by yt-dlp for video: %s", videoID)
		slog.Error(err.Error())
		return nil, err
	}

	if data.HTTPHeaders == nil {
		data.HTTPHeaders = make(map[string]string)
	}
	cookieHeader := ""
	for key, value := range data.HTTPHeaders {
		if strings.EqualFold(key, "Cookie") {
			delete(data.HTTPHeaders, key)
			if cookieHeader == "" && strings.TrimSpace(value) != "" {
				cookieHeader = value
			}
		}
	}
	if cookieHeader == "" {
		cookieHeader = cookie.GetCookieHeader()
	}
	if cookieHeader != "" {
		data.HTTPHeaders["Cookie"] = cookieHeader
	}

	durationInSeconds := int64(data.Duration)
	return &StreamAndDuration{
		URL:         data.URL,
		Duration:    strconv.FormatInt(durationInSeconds, 10),
		HTTPHeaders: data.HTTPHeaders,
	}, nil
}
