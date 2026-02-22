package tmdb

// Movie represents a movie from TMDb search results.
type Movie struct {
	ID          int     `json:"id"`
	Title       string  `json:"title"`
	Overview    string  `json:"overview"`
	ReleaseDate string  `json:"release_date"`
	PosterPath  string  `json:"poster_path"`
	VoteAverage float64 `json:"vote_average"`
	GenreIDs    []int   `json:"genre_ids"`
}

// MovieDetails represents detailed movie information.
type MovieDetails struct {
	ID          int     `json:"id"`
	Title       string  `json:"title"`
	Overview    string  `json:"overview"`
	ReleaseDate string  `json:"release_date"`
	PosterPath  string  `json:"poster_path"`
	VoteAverage float64 `json:"vote_average"`
	Runtime     int     `json:"runtime"`
	Status      string  `json:"status"`
	Tagline     string  `json:"tagline"`
	IMDbID      string  `json:"imdb_id"`
	Genres      []Genre `json:"genres"`
}

// Genre represents a movie genre.
type Genre struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// searchResponse is the TMDb paginated search response.
type searchResponse struct {
	Page         int     `json:"page"`
	Results      []Movie `json:"results"`
	TotalPages   int     `json:"total_pages"`
	TotalResults int     `json:"total_results"`
}

// recommendationsResponse wraps the recommendations/similar endpoint response.
type recommendationsResponse struct {
	Page    int     `json:"page"`
	Results []Movie `json:"results"`
}
