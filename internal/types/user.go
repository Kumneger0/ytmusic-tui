package types // nolint:revive

import (
	musicpb "github.com/kumneger0/ytmusic-tui/gen"
)

type UserSavedTracksListItem struct {
	Name string
}

func (u UserSavedTracksListItem) FilterValue() string {
	return u.Name
}

func (u UserSavedTracksListItem) Title() string {
	return u.Name
}

type SidebarItem struct {
	Name string
	Icon string
}

func (h SidebarItem) FilterValue() string {
	return h.Name
}

func (h SidebarItem) Title() string {
	return h.Name
}

type HomePageSectionItem struct {
	SectionTitle    string
	Index           int
	Artists         []*musicpb.Artist
	DurationSeconds int32
}

func (h HomePageSectionItem) FilterValue() string {
	return h.SectionTitle
}

func (h HomePageSectionItem) Title() string {
	return h.SectionTitle
}

type HomePageContentItem struct {
	ItemTitle       string
	PlaylistID      string
	VideoID         string
	BrowseID        string
	ContentType     string
	Description     string
	Artists         []*musicpb.Artist
	DurationSeconds int32
}

func (h HomePageContentItem) FilterValue() string {
	return h.ItemTitle
}

func (h HomePageContentItem) Title() string {
	return h.ItemTitle
}

func (h HomePageContentItem) Subtitle() string {
	return h.Description
}

type Breadcrumb struct{ Name, Icon string }
