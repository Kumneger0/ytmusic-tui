//go:build linux

package mpris

import (
	"errors"
	"log/slog"
	"runtime"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/prop"
	"github.com/kumneger0/ytmusic-tui/internal/types"
	"github.com/kumneger0/ytmusic-tui/internal/ui"
)

func newProp(value any, cb func(*prop.Change) *dbus.Error) *prop.Prop {
	return &prop.Prop{
		Value:    value,
		Writable: true,
		Emit:     prop.EmitTrue,
		Callback: cb,
	}
}

func getPlayer() map[string]*prop.Prop {
	return map[string]*prop.Prop{
		"PlaybackStatus": newProp("paused", nil),
		"Rate":           newProp(1.0, nil),
		"Metadata":       newProp(map[string]interface{}{}, nil),
		"Volume":         newProp(float64(100), nil),
		"Position":       newProp(int64(0), nil),
		"MinimumRate":    newProp(1.0, nil),
		"MaximumRate":    newProp(1.0, nil),
		"CanGoNext":      newProp(true, nil),
		"CanGoPrevious":  newProp(true, nil),
		"CanPlay":        newProp(true, nil),
		"CanPause":       newProp(true, nil),
		"CanSeek":        newProp(false, nil),
		"CanControl":     newProp(true, nil),
	}
}

var mediaPlayer2 = map[string]*prop.Prop{
	"CanQuit":             newProp(false, nil),
	"CanRaise":            newProp(false, nil),
	"HasTrackList":        newProp(false, nil),
	"Identity":            newProp("ytmusic-tui", nil),
	"SupportedUriSchemes": newProp([]string{}, nil),
	"SupportedMimeTypes":  newProp([]string{}, nil),
}

type MediaPlayer2 struct {
	Props       *prop.Properties
	messageChan *chan types.DBusMessage
}

func (m *MediaPlayer2) Pause() *dbus.Error {
	return nil
}
func (m *MediaPlayer2) Next() *dbus.Error {
	*m.messageChan <- types.DBusMessage{
		MessageType: types.NextTrack,
	}
	return nil
}

func (m *MediaPlayer2) Previous() *dbus.Error {
	*m.messageChan <- types.DBusMessage{
		MessageType: types.PreviousTrack,
	}
	return nil
}

func (m *MediaPlayer2) Play() *dbus.Error {
	return nil
}

func (m *MediaPlayer2) PlayPause() *dbus.Error {
	*m.messageChan <- types.DBusMessage{
		MessageType: types.PlayPause,
	}
	return nil
}

func GetDbusInstance() (*ui.Instance, *chan types.DBusMessage, error) {
	if runtime.GOOS != "linux" {
		return nil, nil, nil
	}
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		slog.Error(err.Error())
		return nil, nil, err
	}

	reply, err := conn.RequestName("org.mpris.MediaPlayer2.ytmusic-tui", dbus.NameFlagReplaceExisting)
	if err != nil {
		slog.Error(err.Error())
		return nil, nil, err
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		return nil, nil, errors.New("Name already taken")
	}

	ins := &ui.Instance{
		Conn: conn,
	}

	ins.Props, err = prop.Export(
		conn,
		"/org/mpris/MediaPlayer2",
		map[string]map[string]*prop.Prop{
			"org.mpris.MediaPlayer2":        mediaPlayer2,
			"org.mpris.MediaPlayer2.Player": getPlayer(),
		},
	)
	if err != nil {
		slog.Error(err.Error())
		return nil, nil, err
	}

	messageChan := make(chan types.DBusMessage, 10)

	mp2 := &MediaPlayer2{
		Props:       ins.Props,
		messageChan: &messageChan,
	}

	if err := conn.Export(mp2, "/org/mpris/MediaPlayer2", "org.mpris.MediaPlayer2.Player"); err != nil {
		slog.Error(err.Error())
		return nil, nil, err
	}

	return ins, &messageChan, nil
}
