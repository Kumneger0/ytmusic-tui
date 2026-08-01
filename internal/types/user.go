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
