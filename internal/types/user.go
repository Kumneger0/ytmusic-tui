package types // nolint:revive

type UserTokenInfo struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
	ExpiresAt    int64  `json:"expires_at,omitempty"`
}

type InstallStep struct {
	Command string
	Args    []string
}

// UserSavedTracksListItem is a simple sidebar item representing the user's liked songs.
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
	SectionTitle string
	Index        int
}

func (h HomePageSectionItem) FilterValue() string {
	return h.SectionTitle
}

func (h HomePageSectionItem) Title() string {
	return h.SectionTitle
}

type HomePageContentItem struct {
	ItemTitle   string
	PlaylistID  string
	VideoID     string
	BrowseID    string
	ContentType string
	Description string
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

type SearchResultType int

const (
	SearchResultTrack SearchResultType = iota
	SearchResultArtist
	SearchResultPlaylist
	SearchResultAlbum
	SearchResultPodcast
	SearchResultEpisode
)

type SearchResultItem interface {
	Title() string
	FilterValue() string
	Kind() SearchResultType
}

func (t Track) Title() string             { return t.Name }
func (t Track) FilterValue() string       { return t.Name }
func (t Track) Kind() SearchResultType    { return SearchResultTrack }
func (a Artist) Kind() SearchResultType   { return SearchResultArtist }
func (p Playlist) Kind() SearchResultType { return SearchResultPlaylist }
func (a Album) Title() string             { return a.Name }
func (a Album) FilterValue() string       { return a.Name }
func (a Album) Kind() SearchResultType    { return SearchResultAlbum }
