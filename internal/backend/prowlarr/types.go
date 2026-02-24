package prowlarr

// Application represents a Prowlarr application (Radarr/Sonarr) configuration.
type Application struct {
	ID             int     `json:"id,omitempty"`
	Name           string  `json:"name"`
	Implementation string  `json:"implementation"`
	ConfigContract string  `json:"configContract"`
	SyncLevel      string  `json:"syncLevel"`
	Fields         []Field `json:"fields"`
}

// DownloadClient represents a download client in Prowlarr.
type DownloadClient struct {
	ID             int     `json:"id,omitempty"`
	Name           string  `json:"name"`
	Implementation string  `json:"implementation"`
	ConfigContract string  `json:"configContract"`
	Enable         bool    `json:"enable"`
	Protocol       string  `json:"protocol"`
	Priority       int     `json:"priority"`
	Fields         []Field `json:"fields"`
}

// IndexerProxy represents a Prowlarr indexer proxy (e.g., FlareSolverr).
type IndexerProxy struct {
	ID             int     `json:"id,omitempty"`
	Name           string  `json:"name"`
	Implementation string  `json:"implementation"`
	ConfigContract string  `json:"configContract"`
	Fields         []Field `json:"fields"`
}

// Field represents a configuration field for Prowlarr API objects.
type Field struct {
	Name  string `json:"name"`
	Value any    `json:"value"`
}
