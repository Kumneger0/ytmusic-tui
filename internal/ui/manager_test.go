package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kumneger0/ytmusic-tui/internal/types"
)

type dummyModel struct {
	receivedMsg tea.Msg
}

func (d *dummyModel) Init() tea.Cmd { return nil }
func (d *dummyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	d.receivedMsg = msg
	return d, nil
}
func (d *dummyModel) View() string { return "dummy" }

func TestManager_MessageRoutingInForegroundState(t *testing.T) {
	fg := &dummyModel{}
	bg := &dummyModel{}

	mgr := Manager{
		State:      Foreground,
		Foreground: fg,
		Background: bg,
	}

	addMsg := types.AddToPlaylistMsg{PlaylistID: "PL1", TrackID: "T1"}
	updatedMgr, _ := mgr.Update(addMsg)
	mgr = updatedMgr.(Manager)

	if bg.receivedMsg != addMsg {
		t.Fatalf("expected Background model to receive AddToPlaylistMsg, got %v", bg.receivedMsg)
	}

	respMsg := types.AddToPlaylistResponseMsg{Success: true}
	updatedMgr, _ = mgr.Update(respMsg)
	mgr = updatedMgr.(Manager)

	if fg.receivedMsg != respMsg {
		t.Fatalf("expected Foreground model to receive AddToPlaylistResponseMsg, got %v", fg.receivedMsg)
	}
	if bg.receivedMsg != respMsg {
		t.Fatalf("expected Background model to receive AddToPlaylistResponseMsg, got %v", bg.receivedMsg)
	}
}

func TestManager_KeyMsgIsolationInForegroundState(t *testing.T) {
	fg := &dummyModel{}
	bg := &dummyModel{}

	mgr := Manager{
		State:      Foreground,
		Foreground: fg,
		Background: bg,
	}

	keyMsg := tea.KeyMsg{Type: tea.KeyDown}
	updatedMgr, _ := mgr.Update(keyMsg)
	mgr = updatedMgr.(Manager)

	k, ok := fg.receivedMsg.(tea.KeyMsg)
	if !ok || k.Type != keyMsg.Type {
		t.Fatalf("expected Foreground model to receive KeyMsg, got %v", fg.receivedMsg)
	}
	if bg.receivedMsg != nil {
		t.Fatalf("expected Background model NOT to receive KeyMsg while in Foreground state, got %v", bg.receivedMsg)
	}
}
