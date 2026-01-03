package music

type SpotifyToken struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

type Playlist struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	ExternalUrls ExternalUrls `json:"external_urls"`
	Owner        Owner        `json:"owner"`
	Tracks       TracksObject `json:"tracks"`
}

type TracksObject struct {
	Items []PlaylistItem `json:"items"`
}

type PlaylistItem struct {
	Track Track `json:"track"`
}

type ExternalUrls struct {
	Spotify string `json:"spotify"`
}

type Owner struct {
	DisplayName    string       `json:"display_name"`
	ID             string       `json:"id"`
	ExExternalUrls ExternalUrls `json:"external_urls"`
}

type Artist struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	ExternalUrls ExternalUrls `json:"external_urls"`
}

type Album struct {
	ExternalUrls ExternalUrls `json:"external_urls"`
	Artists      []Artist     `json:"artists"`
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	ReleaseDate  string       `json:"release_date"`
}

type Track struct {
	Name         string       `json:"name"`
	DurationMs   int          `json:"duration_ms"`
	ExternalUrls ExternalUrls `json:"external_urls"`
	Artists      []Artist     `json:"artists"`
	Album        Album        `json:"album"`
}

type RecentlyPlayedItem struct {
	Track    Track       `json:"track"`
	PlayedAt string      `json:"played_at"`
	Context  PlayContext `json:"context"`
}

type PlayContext struct {
	Type         string       `json:"type"`
	URI          string       `json:"uri"`
	Href         string       `json:"href"`
	ExternalUrls ExternalUrls `json:"external_urls"`
}

type RecentlyPlayedResponse struct {
	Items []RecentlyPlayedItem `json:"items"`
	Next  string               `json:"next"`
	Limit int                  `json:"limit"`
	Href  string               `json:"href"`
}
