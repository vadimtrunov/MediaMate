package notification

// EventGrab is the Radarr webhook event type for grabbed (started) downloads.
const EventGrab = "Grab"

// EventDownload is the Radarr webhook event type for completed downloads.
const EventDownload = "Download"

// RadarrWebhookPayload represents the JSON body sent by Radarr webhooks.
type RadarrWebhookPayload struct {
	EventType      string        `json:"eventType"`
	InstanceName   string        `json:"instanceName,omitempty"`
	ApplicationURL string        `json:"applicationUrl,omitempty"`
	Movie          RadarrMovie   `json:"movie"`
	RemoteMovie    RadarrMovie   `json:"remoteMovie"`
	MovieFile      RadarrFile    `json:"movieFile,omitempty"`
	Release        RadarrRelease `json:"release,omitempty"`
	DownloadClient string        `json:"downloadClient,omitempty"`
	DownloadID     string        `json:"downloadId,omitempty"`
	IsUpgrade      bool          `json:"isUpgrade"`
}

// RadarrMovie holds movie metadata from a Radarr webhook.
type RadarrMovie struct {
	ID          int    `json:"id,omitempty"`
	TmdbID      int    `json:"tmdbId,omitempty"`
	ImdbID      string `json:"imdbId,omitempty"`
	Title       string `json:"title"`
	Year        int    `json:"year"`
	FolderPath  string `json:"folderPath,omitempty"`
	ReleaseDate string `json:"releaseDate,omitempty"`
	Overview    string `json:"overview,omitempty"`
}

// RadarrQuality represents the nested quality object from Radarr webhooks.
type RadarrQuality struct {
	ID   int    `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// RadarrFile holds information about the downloaded movie file.
type RadarrFile struct {
	ID             int           `json:"id,omitempty"`
	RelativePath   string        `json:"relativePath,omitempty"`
	Path           string        `json:"path,omitempty"`
	Quality        RadarrQuality `json:"quality,omitempty"`
	QualityVersion int           `json:"qualityVersion,omitempty"`
	ReleaseGroup   string        `json:"releaseGroup,omitempty"`
	SceneName      string        `json:"sceneName,omitempty"`
	Size           int64         `json:"size,omitempty"`
}

// RadarrRelease holds release information from a Radarr webhook.
type RadarrRelease struct {
	Quality        RadarrQuality `json:"quality,omitempty"`
	QualityVersion int           `json:"qualityVersion,omitempty"`
	ReleaseGroup   string        `json:"releaseGroup,omitempty"`
	ReleaseTitle   string        `json:"releaseTitle,omitempty"`
	Indexer        string        `json:"indexer,omitempty"`
	Size           int64         `json:"size,omitempty"`
}

// MovieTitle returns the best available title from the payload.
// It prefers movie.Title, falls back to remoteMovie.Title.
func (p *RadarrWebhookPayload) MovieTitle() string {
	if p.Movie.Title != "" {
		return p.Movie.Title
	}
	return p.RemoteMovie.Title
}

// MovieYear returns the best available year from the payload.
func (p *RadarrWebhookPayload) MovieYear() int {
	if p.Movie.Year != 0 {
		return p.Movie.Year
	}
	return p.RemoteMovie.Year
}
