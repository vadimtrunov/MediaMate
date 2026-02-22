package agent

import "github.com/vadimtrunov/MediaMate/internal/core"

func toolDefinitions() []core.Tool {
	return []core.Tool{
		toolSearchMovie(),
		toolDefGetMovieDetails(),
		toolDefDownloadMovie(),
		toolDefGetDownloadStatus(),
		toolDefRecommendSimilar(),
		toolDefListDownloads(),
	}
}

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

func toolDefGetMovieDetails() core.Tool {
	return core.Tool{
		Name: "get_movie_details",
		Description: "Get detailed information about a movie by its TMDb ID." +
			" Returns runtime, genres, tagline, full overview, and ratings.",
		Parameters: tmdbIDParam("The TMDb ID of the movie"),
	}
}

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

func toolDefGetDownloadStatus() core.Tool {
	return core.Tool{
		Name: "get_download_status",
		Description: "Check the download status of a movie in Radarr by its Radarr ID." +
			" Returns whether it's wanted or downloaded.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"radarr_id": map[string]any{
					"type":        "string",
					"description": "The Radarr ID of the movie",
				},
			},
			"required": []string{"radarr_id"},
		},
	}
}

func toolDefRecommendSimilar() core.Tool {
	return core.Tool{
		Name:        "recommend_similar",
		Description: "Get movie recommendations similar to a given movie. Returns a list of recommended movies from TMDb.",
		Parameters:  tmdbIDParam("The TMDb ID of the movie to get recommendations for"),
	}
}

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
