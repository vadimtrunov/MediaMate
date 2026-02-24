package agent

import "github.com/vadimtrunov/MediaMate/internal/core"

// toolDefinitions returns the list of tool definitions available to the LLM.
func toolDefinitions() []core.Tool {
	return []core.Tool{
		toolSearchMovie(),
		toolDefGetMovieDetails(),
		toolDefDownloadMovie(),
		toolDefGetDownloadStatus(),
		toolDefRecommendSimilar(),
		toolDefListDownloads(),
		toolDefCheckAvailability(),
		toolDefGetWatchLink(),
	}
}

// tmdbIDParam returns a JSON Schema object requiring a single tmdb_id integer parameter.
func tmdbIDParam(desc string) map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"tmdb_id": map[string]any{
				"type":        "integer",
				"description": desc,
			},
		},
		"required": []string{"tmdb_id"},
	}
}

// toolSearchMovie returns the search_movie tool definition.
func toolSearchMovie() core.Tool {
	return core.Tool{
		Name:        "search_movie",
		Description: "Search for a movie by title. Returns a list of matching movies with their TMDb IDs, titles, years, and ratings.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": "The movie title to search for",
				},
				"year": map[string]any{
					"type":        "integer",
					"description": "Optional release year to filter results",
				},
			},
			"required": []string{"query"},
		},
	}
}

// toolDefGetMovieDetails returns the get_movie_details tool definition.
func toolDefGetMovieDetails() core.Tool {
	return core.Tool{
		Name: "get_movie_details",
		Description: "Get detailed information about a movie by its TMDb ID." +
			" Returns runtime, genres, tagline, full overview, and ratings.",
		Parameters: tmdbIDParam("The TMDb ID of the movie"),
	}
}

// toolDefDownloadMovie returns the download_movie tool definition.
func toolDefDownloadMovie() core.Tool {
	return core.Tool{
		Name: "download_movie",
		Description: "Add a movie to the download queue via Radarr." +
			" Searches for releases and starts downloading automatically.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"tmdb_id": map[string]any{
					"type":        "integer",
					"description": "The TMDb ID of the movie to download",
				},
				"title": map[string]any{
					"type":        "string",
					"description": "The movie title (for display purposes)",
				},
			},
			"required": []string{"tmdb_id", "title"},
		},
	}
}

// toolDefGetDownloadStatus returns the get_download_status tool definition.
func toolDefGetDownloadStatus() core.Tool {
	return core.Tool{
		Name: "get_download_status",
		Description: "Check the download status of a movie in Radarr by its Radarr ID." +
			" Returns whether it's wanted or downloaded.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"radarr_id": map[string]any{
					"type":        "integer",
					"description": "The Radarr ID of the movie",
				},
			},
			"required": []string{"radarr_id"},
		},
	}
}

// toolDefRecommendSimilar returns the recommend_similar tool definition.
func toolDefRecommendSimilar() core.Tool {
	return core.Tool{
		Name:        "recommend_similar",
		Description: "Get movie recommendations similar to a given movie. Returns a list of recommended movies from TMDb.",
		Parameters:  tmdbIDParam("The TMDb ID of the movie to get recommendations for"),
	}
}

// toolDefListDownloads returns the list_downloads tool definition.
func toolDefListDownloads() core.Tool {
	return core.Tool{
		Name:        "list_downloads",
		Description: "List all active torrent downloads with their progress, speed, and ETA.",
		Parameters: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	}
}

// titleParam returns a JSON Schema object requiring a single title string parameter.
func titleParam(desc string) map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"title": map[string]any{
				"type":        "string",
				"description": desc,
			},
		},
		"required": []string{"title"},
	}
}

// toolDefCheckAvailability returns the check_availability tool definition.
func toolDefCheckAvailability() core.Tool {
	return core.Tool{
		Name:        "check_availability",
		Description: "Check if a movie is available to watch on the media server (Jellyfin).",
		Parameters:  titleParam("The movie title to check availability for"),
	}
}

// toolDefGetWatchLink returns the get_watch_link tool definition.
func toolDefGetWatchLink() core.Tool {
	return core.Tool{
		Name:        "get_watch_link",
		Description: "Get a direct link to watch a movie on the media server (Jellyfin).",
		Parameters:  titleParam("The movie title to get the watch link for"),
	}
}
