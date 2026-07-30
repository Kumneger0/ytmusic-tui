//go:build windows

package mpris

import (
	"github.com/kumneger0/ytmusic-tui/internal/types"
	"github.com/kumneger0/ytmusic-tui/internal/ui"
)

func GetDbusInstance() (*ui.Instance, *chan types.DBusMessage, error) {
	// TODO: implement mpris for windows
	return nil, nil, nil
}
