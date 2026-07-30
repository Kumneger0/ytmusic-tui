//go:build darwin

package mpris

import (
	"github.com/kumneger0/ytmusic-tui/internal/types"
	"github.com/kumneger0/ytmusic-tui/internal/ui"
)

func GetDbusInstance() (*ui.Instance, *chan types.DBusMessage, error) {
	// TODO: implement mpris for darwin
	return nil, nil, nil
}
