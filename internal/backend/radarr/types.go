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

type radarrRatings struct {
	Tmdb radarrRating `json:"tmdb"`
	Imdb radarrRating `json:"imdb"`
}

type radarrRating struct {
	Value float64 `json:"value"`
}

type radarrAddOpts struct {
	SearchForMovie bool `json:"searchForMovie"`
}

type radarrQualityProfile struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type radarrRootFolder struct {
	ID   int    `json:"id"`
	Path string `json:"path"`
}
