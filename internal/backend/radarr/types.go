package radarr

// radarrMovie represents a movie in Radarr's API v3 format.
type radarrMovie struct {
	ID               int            `json:"id,omitempty"`
	Title            string         `json:"title"`
	Year             int            `json:"year"`
	TmdbID           int            `json:"tmdbId"`
	Overview         string         `json:"overview"`
	RemotePoster     string         `json:"remotePoster,omitempty"`
	Ratings          radarrRatings  `json:"ratings,omitempty"`
	HasFile          bool           `json:"hasFile"`
	Monitored        bool           `json:"monitored"`
	QualityProfileID int            `json:"qualityProfileId,omitempty"`
	RootFolderPath   string         `json:"rootFolderPath,omitempty"`
	AddOptions       *radarrAddOpts `json:"addOptions,omitempty"`
}

// radarrRatings holds TMDb and IMDb ratings for a movie.
type radarrRatings struct {
	Tmdb radarrRating `json:"tmdb"`
	Imdb radarrRating `json:"imdb"`
}

// radarrRating holds a single rating value.
type radarrRating struct {
	Value float64 `json:"value"`
}

// radarrAddOpts holds options for adding a movie to Radarr.
type radarrAddOpts struct {
	SearchForMovie bool `json:"searchForMovie"`
}

// radarrQualityProfile represents a Radarr quality profile.
type radarrQualityProfile struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// radarrRootFolder represents a Radarr root folder path.
type radarrRootFolder struct {
	ID   int    `json:"id"`
	Path string `json:"path"`
}

// QualityProfile represents a Radarr quality profile (exported for setup).
type QualityProfile struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// RootFolder represents a Radarr root folder (exported for setup).
type RootFolder struct {
	ID   int    `json:"id"`
	Path string `json:"path"`
}

// DownloadClientConfig represents a download client configuration in Radarr.
type DownloadClientConfig struct {
	Name           string                `json:"name"`
	Implementation string                `json:"implementation"`
	ConfigContract string                `json:"configContract"`
	Enable         bool                  `json:"enable"`
	Protocol       string                `json:"protocol"`
	Priority       int                   `json:"priority"`
	Fields         []DownloadClientField `json:"fields"`
}

// DownloadClientField represents a field in a download client config.
type DownloadClientField struct {
	Name  string `json:"name"`
	Value any    `json:"value"`
}
