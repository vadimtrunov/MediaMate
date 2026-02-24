package jellyfin

// jellyfinItemsResponse represents the paginated response from Jellyfin's /Items endpoint.
type jellyfinItemsResponse struct {
	Items            []jellyfinItem `json:"Items"`
	TotalRecordCount int            `json:"TotalRecordCount"`
}

// jellyfinItem represents a single media item returned by the Jellyfin API.
type jellyfinItem struct {
	ID              string            `json:"Id"`
	Name            string            `json:"Name"`
	ProductionYear  int               `json:"ProductionYear"`
	Overview        string            `json:"Overview"`
	Type            string            `json:"Type"`
	ImageTags       map[string]string `json:"ImageTags"`
	CommunityRating float64           `json:"CommunityRating"`
	RunTimeTicks    int64             `json:"RunTimeTicks"`
}
