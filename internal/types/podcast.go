package types // nolint:revive

type Podcast struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Author      string  `json:"author"`
	Description string  `json:"description"`
	Images      []Image `json:"images"`
}

func (p Podcast) Title() string       { return p.Name }
func (p Podcast) FilterValue() string { return p.Name + " (podcast)" }
func (p Podcast) Kind() SearchResultType {
	return SearchResultPodcast
}

type Episode struct {
	ID          string  `json:"id"`
	TitleName   string  `json:"title"`
	PodcastName string  `json:"podcast_name"`
	PodcastID   string  `json:"podcast_id"`
	Date        string  `json:"date"`
	Images      []Image `json:"images"`
}

func (e Episode) Title() string       { return e.TitleName }
func (e Episode) FilterValue() string { return e.TitleName + " (episode)" }
func (e Episode) Kind() SearchResultType {
	return SearchResultEpisode
}
