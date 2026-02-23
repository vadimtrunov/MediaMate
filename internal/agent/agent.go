package agent

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/vadimtrunov/MediaMate/internal/core"
	"github.com/vadimtrunov/MediaMate/internal/metadata/tmdb"
)

const (
	maxToolIterations = 10
	systemPrompt      = `You are MediaMate, a helpful AI assistant for managing a personal media server.
You help users search for movies, get recommendations, check download status, and manage their media library.

When a user asks about a movie, use the search_movie tool to find it.
When they want to download something, use download_movie with the TMDb ID.
When they want recommendations, use recommend_similar.
When they ask about active downloads, use list_downloads.

Be concise but friendly. Format movie information clearly with title, year, and rating.
When presenting search results, number them for easy reference.`
)

// Agent orchestrates conversations between the user, LLM, and backend services.
type Agent struct {
	llm     core.LLMProvider
	tmdb    *tmdb.Client
	backend core.MediaBackend
	torrent core.TorrentClient
	history []core.Message
	tools   []core.Tool
	logger  *slog.Logger
}

// New creates a new Agent.
func New(llm core.LLMProvider, tmdbClient *tmdb.Client, backend core.MediaBackend, torrent core.TorrentClient, logger *slog.Logger) *Agent {
	if logger == nil {
		logger = slog.Default()
	}
	return &Agent{
		llm:     llm,
		tmdb:    tmdbClient,
		backend: backend,
		torrent: torrent,
		history: []core.Message{
			{Role: "system", Content: systemPrompt},
		},
		tools:  toolDefinitions(),
		logger: logger,
	}
}

// HandleMessage processes a user message and returns the assistant's response.
func (a *Agent) HandleMessage(ctx context.Context, userMessage string) (string, error) {
	a.history = append(a.history, core.Message{
		Role:    "user",
		Content: userMessage,
	})

	for range maxToolIterations {
		resp, err := a.llm.Chat(ctx, a.history, a.tools)
		if err != nil {
			return "", fmt.Errorf("llm chat: %w", err)
		}

		if len(resp.ToolCalls) > 0 {
			a.history = append(a.history, core.Message{
				Role:      "assistant",
				Content:   resp.Content,
				ToolCalls: resp.ToolCalls,
			})

			for _, call := range resp.ToolCalls {
				result, toolErr := a.executeTool(ctx, call)
				isError := toolErr != nil
				content := result
				if isError {
					content = toolErr.Error()
				}
				a.history = append(a.history, core.Message{
					Role:         "user",
					Content:      content,
					ToolResultID: call.ID,
					IsError:      isError,
				})
			}
			continue
		}

		a.history = append(a.history, core.Message{
			Role:    "assistant",
			Content: resp.Content,
		})
		return resp.Content, nil
	}

	return "", fmt.Errorf("agent exceeded maximum tool iterations (%d)", maxToolIterations)
}

func (a *Agent) executeTool(ctx context.Context, call core.ToolCall) (string, error) {
	a.logger.Debug("executing tool", slog.String("tool", call.Name), slog.Any("args", call.Arguments))

	switch call.Name {
	case "search_movie":
		return a.toolSearchMovie(ctx, call.Arguments)
	case "get_movie_details":
		return a.toolGetMovieDetails(ctx, call.Arguments)
	case "download_movie":
		return a.toolDownloadMovie(ctx, call.Arguments)
	case "get_download_status":
		return a.toolGetDownloadStatus(ctx, call.Arguments)
	case "recommend_similar":
		return a.toolRecommendSimilar(ctx, call.Arguments)
	case "list_downloads":
		return a.toolListDownloads(ctx, call.Arguments)
	default:
		return "", fmt.Errorf("unknown tool: %s", call.Name)
	}
}

// Reset clears the conversation history.
func (a *Agent) Reset() {
	a.history = []core.Message{
		{Role: "system", Content: systemPrompt},
	}
}
