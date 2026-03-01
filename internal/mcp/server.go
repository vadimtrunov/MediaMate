package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/vadimtrunov/MediaMate/internal/core"
	"github.com/vadimtrunov/MediaMate/internal/metadata/tmdb"
)

// Deps holds backend dependencies for MCP tool handlers.
type Deps struct {
	TMDb        *tmdb.Client
	Backend     core.MediaBackend
	Torrent     core.TorrentClient
	MediaServer core.MediaServer
}

// Server wraps an MCP SDK server with MediaMate tool handlers.
type Server struct {
	server *mcpsdk.Server
	deps   Deps
	logger *slog.Logger
}

// NewServer creates an MCP server with all MediaMate tools registered.
func NewServer(deps Deps, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}

	s := mcpsdk.NewServer(
		&mcpsdk.Implementation{
			Name:    "mediamate",
			Version: "0.2.0",
		},
		&mcpsdk.ServerOptions{Logger: logger},
	)

	srv := &Server{server: s, deps: deps, logger: logger}
	srv.registerTools()
	return srv
}

// ServeStdio runs the MCP server over stdin/stdout.
func (s *Server) ServeStdio(ctx context.Context) error {
	return s.server.Run(ctx, &mcpsdk.StdioTransport{})
}

// MCPServer returns the underlying MCP SDK server (for testing).
func (s *Server) MCPServer() *mcpsdk.Server {
	return s.server
}

// registerTools registers all 8 MediaMate tools on the MCP server.
func (s *Server) registerTools() {
	s.server.AddTool(searchMovieTool(), s.handleSearchMovie)
	s.server.AddTool(getMovieDetailsTool(), s.handleGetMovieDetails)
	s.server.AddTool(downloadMovieTool(), s.handleDownloadMovie)
	s.server.AddTool(getDownloadStatusTool(), s.handleGetDownloadStatus)
	s.server.AddTool(recommendSimilarTool(), s.handleRecommendSimilar)
	s.server.AddTool(listDownloadsTool(), s.handleListDownloads)
	s.server.AddTool(checkAvailabilityTool(), s.handleCheckAvailability)
	s.server.AddTool(getWatchLinkTool(), s.handleGetWatchLink)
}

// Tool definitions — same schemas as agent/tools.go.

func searchMovieTool() *mcpsdk.Tool {
	return &mcpsdk.Tool{
		Name:        "search_movie",
		Description: "Search for a movie by title. Returns a list of matching movies with their TMDb IDs, titles, years, and ratings.",
		InputSchema: map[string]any{
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
			"required": []any{"query"},
		},
	}
}

func getMovieDetailsTool() *mcpsdk.Tool {
	return &mcpsdk.Tool{
		Name:        "get_movie_details",
		Description: "Get detailed information about a movie by its TMDb ID. Returns runtime, genres, tagline, full overview, and ratings.",
		InputSchema: tmdbIDSchema("The TMDb ID of the movie"),
	}
}

func downloadMovieTool() *mcpsdk.Tool {
	return &mcpsdk.Tool{
		Name:        "download_movie",
		Description: "Add a movie to the download queue via Radarr. Searches for releases and starts downloading automatically.",
		InputSchema: map[string]any{
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
			"required": []any{"tmdb_id", "title"},
		},
	}
}

func getDownloadStatusTool() *mcpsdk.Tool {
	return &mcpsdk.Tool{
		Name:        "get_download_status",
		Description: "Check the download status of a movie in Radarr by its Radarr ID. Returns whether it's wanted or downloaded.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"radarr_id": map[string]any{
					"type":        "integer",
					"description": "The Radarr ID of the movie",
				},
			},
			"required": []any{"radarr_id"},
		},
	}
}

func recommendSimilarTool() *mcpsdk.Tool {
	return &mcpsdk.Tool{
		Name:        "recommend_similar",
		Description: "Get movie recommendations similar to a given movie. Returns a list of recommended movies from TMDb.",
		InputSchema: tmdbIDSchema("The TMDb ID of the movie to get recommendations for"),
	}
}

func listDownloadsTool() *mcpsdk.Tool {
	return &mcpsdk.Tool{
		Name:        "list_downloads",
		Description: "List all active torrent downloads with their progress, speed, and ETA.",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	}
}

func checkAvailabilityTool() *mcpsdk.Tool {
	return &mcpsdk.Tool{
		Name:        "check_availability",
		Description: "Check if a movie is available to watch on the media server (Jellyfin).",
		InputSchema: titleSchema("The movie title to check availability for"),
	}
}

func getWatchLinkTool() *mcpsdk.Tool {
	return &mcpsdk.Tool{
		Name:        "get_watch_link",
		Description: "Get a direct link to watch a movie on the media server (Jellyfin).",
		InputSchema: titleSchema("The movie title to get the watch link for"),
	}
}

func tmdbIDSchema(desc string) map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"tmdb_id": map[string]any{
				"type":        "integer",
				"description": desc,
			},
		},
		"required": []any{"tmdb_id"},
	}
}

func titleSchema(desc string) map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"title": map[string]any{
				"type":        "string",
				"description": desc,
			},
		},
		"required": []any{"title"},
	}
}

// Tool handlers — each parses arguments, calls backend, returns JSON text content.

func (s *Server) handleSearchMovie(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
	if s.deps.TMDb == nil {
		return toolError("TMDb client not configured"), nil
	}

	var args struct {
		Query string `json:"query"`
		Year  int    `json:"year"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		return toolError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	if args.Query == "" {
		return toolError("search_movie requires a 'query' string argument"), nil
	}

	movies, err := s.deps.TMDb.SearchMovies(ctx, args.Query, args.Year)
	if err != nil {
		return toolError(fmt.Sprintf("tmdb search failed: %v", err)), nil
	}
	return toolJSON(movies)
}

func (s *Server) handleGetMovieDetails(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
	if s.deps.TMDb == nil {
		return toolError("TMDb client not configured"), nil
	}

	tmdbID, err := extractIntFromArgs(req.Params.Arguments, "tmdb_id")
	if err != nil {
		return toolError(err.Error()), nil
	}

	details, err := s.deps.TMDb.GetMovie(ctx, tmdbID)
	if err != nil {
		return toolError(fmt.Sprintf("tmdb get movie failed: %v", err)), nil
	}
	return toolJSON(details)
}

func (s *Server) handleDownloadMovie(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
	if s.deps.Backend == nil {
		return toolError("no media backend configured for downloading"), nil
	}

	var args struct {
		TMDbID int    `json:"tmdb_id"`
		Title  string `json:"title"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		return toolError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	item := core.MediaItem{
		Title: args.Title,
		Type:  "movie",
		Metadata: map[string]string{
			"tmdbId": strconv.Itoa(args.TMDbID),
		},
	}

	if err := s.deps.Backend.Add(ctx, item); err != nil {
		return toolError(fmt.Sprintf("failed to add movie: %v", err)), nil
	}

	return toolJSON(map[string]any{
		"status":  "added",
		"title":   args.Title,
		"tmdb_id": args.TMDbID,
	})
}

func (s *Server) handleGetDownloadStatus(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
	if s.deps.Backend == nil {
		return toolError("no media backend configured"), nil
	}

	radarrID, err := extractIntFromArgs(req.Params.Arguments, "radarr_id")
	if err != nil {
		return toolError(err.Error()), nil
	}

	status, err := s.deps.Backend.GetStatus(ctx, strconv.Itoa(radarrID))
	if err != nil {
		return toolError(fmt.Sprintf("get status failed: %v", err)), nil
	}
	return toolJSON(status)
}

func (s *Server) handleRecommendSimilar(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
	if s.deps.TMDb == nil {
		return toolError("TMDb client not configured"), nil
	}

	tmdbID, err := extractIntFromArgs(req.Params.Arguments, "tmdb_id")
	if err != nil {
		return toolError(err.Error()), nil
	}

	movies, err := s.deps.TMDb.GetRecommendations(ctx, tmdbID)
	if err != nil {
		return toolError(fmt.Sprintf("tmdb recommendations failed: %v", err)), nil
	}
	return toolJSON(movies)
}

func (s *Server) handleListDownloads(ctx context.Context, _ *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
	if s.deps.Torrent == nil {
		return toolError("no torrent client configured"), nil
	}

	torrents, err := s.deps.Torrent.List(ctx)
	if err != nil {
		return toolError(fmt.Sprintf("list torrents failed: %v", err)), nil
	}
	return toolJSON(torrents)
}

func (s *Server) handleCheckAvailability(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
	if s.deps.MediaServer == nil {
		return toolError("no media server configured"), nil
	}

	title, err := extractStringFromArgs(req.Params.Arguments, "title")
	if err != nil {
		return toolError(err.Error()), nil
	}

	available, err := s.deps.MediaServer.IsAvailable(ctx, title)
	if err != nil {
		return toolError(fmt.Sprintf("check availability failed: %v", err)), nil
	}

	return toolJSON(map[string]any{
		"title":     title,
		"available": available,
	})
}

func (s *Server) handleGetWatchLink(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
	if s.deps.MediaServer == nil {
		return toolError("no media server configured"), nil
	}

	title, err := extractStringFromArgs(req.Params.Arguments, "title")
	if err != nil {
		return toolError(err.Error()), nil
	}

	link, err := s.deps.MediaServer.GetLink(ctx, title)
	if err != nil {
		return toolError(fmt.Sprintf("get watch link failed: %v", err)), nil
	}

	return toolJSON(map[string]any{
		"title": title,
		"link":  link,
	})
}

// Helper functions.

// toolJSON marshals v to JSON and returns it as text content.
func toolJSON(v any) (*mcpsdk.CallToolResult, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return toolError(fmt.Sprintf("marshal result: %v", err)), nil
	}
	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: string(data)}},
	}, nil
}

// toolError returns a tool result indicating an error.
func toolError(msg string) *mcpsdk.CallToolResult {
	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: msg}},
		IsError: true,
	}
}

// extractIntFromArgs extracts an integer argument from raw JSON arguments.
func extractIntFromArgs(raw json.RawMessage, key string) (int, error) {
	var args map[string]any
	if err := json.Unmarshal(raw, &args); err != nil {
		return 0, fmt.Errorf("invalid arguments: %w", err)
	}

	val, ok := args[key]
	if !ok {
		return 0, fmt.Errorf("%s is required", key)
	}

	switch v := val.(type) {
	case float64:
		return int(v), nil
	case string:
		n, err := strconv.Atoi(v)
		if err != nil {
			return 0, fmt.Errorf("%s must be a number: %w", key, err)
		}
		return n, nil
	default:
		return 0, fmt.Errorf("%s must be a number, got %T", key, val)
	}
}

// extractStringFromArgs extracts a string argument from raw JSON arguments.
func extractStringFromArgs(raw json.RawMessage, key string) (string, error) {
	var args map[string]any
	if err := json.Unmarshal(raw, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	val, ok := args[key]
	if !ok {
		return "", fmt.Errorf("%s is required", key)
	}

	s, ok := val.(string)
	if !ok || s == "" {
		return "", fmt.Errorf("%s must be a non-empty string", key)
	}
	return s, nil
}
