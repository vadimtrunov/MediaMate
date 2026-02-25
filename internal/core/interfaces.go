package core

import "context"

// LLMProvider defines the interface for Large Language Model providers (Claude, OpenAI, Ollama)
type LLMProvider interface {
	// Chat sends a message and receives a response with tool calls
	Chat(ctx context.Context, messages []Message, tools []Tool) (*Response, error)

	// Name returns the provider name (e.g., "claude", "openai", "ollama")
	Name() string
}

// MediaBackend defines the interface for media management backends (Radarr, Sonarr, Readarr)
type MediaBackend interface {
	// Search searches for media items (movies, TV shows, books)
	Search(ctx context.Context, query string) ([]MediaItem, error)

	// Add adds a media item to the library
	Add(ctx context.Context, item MediaItem) error

	// GetStatus gets the status of a media item
	GetStatus(ctx context.Context, itemID string) (*MediaStatus, error)

	// ListItems lists all items in the library
	ListItems(ctx context.Context) ([]MediaItem, error)

	// Type returns the backend type (e.g., "radarr", "sonarr", "readarr")
	Type() string
}

// TorrentClient defines the interface for torrent clients (qBittorrent, Transmission, Deluge)
type TorrentClient interface {
	// List returns all active torrents
	List(ctx context.Context) ([]Torrent, error)

	// GetProgress gets download progress for a specific torrent
	GetProgress(ctx context.Context, hash string) (*TorrentProgress, error)

	// Pause pauses a torrent
	Pause(ctx context.Context, hash string) error

	// Resume resumes a torrent
	Resume(ctx context.Context, hash string) error

	// Remove removes a torrent
	Remove(ctx context.Context, hash string, deleteFiles bool) error

	// Name returns the client name (e.g., "qbittorrent", "transmission")
	Name() string
}

// MediaServer defines the interface for media servers (Jellyfin, Plex)
type MediaServer interface {
	// IsAvailable checks if a media item is available on the server
	IsAvailable(ctx context.Context, itemName string) (bool, error)

	// GetLink generates a direct link to watch the media
	GetLink(ctx context.Context, itemName string) (string, error)

	// GetLibraryItems gets all items in the library
	GetLibraryItems(ctx context.Context) ([]MediaItem, error)

	// Name returns the server name (e.g., "jellyfin", "plex")
	Name() string
}

// Frontend defines the interface for user-facing frontends (CLI, Telegram, Discord)
type Frontend interface {
	// Start starts the frontend
	Start(ctx context.Context) error

	// Stop stops the frontend
	Stop(ctx context.Context) error

	// SendMessage sends a message to the user
	SendMessage(ctx context.Context, userID string, message string) error

	// Name returns the frontend name (e.g., "cli", "telegram", "discord")
	Name() string
}

// Message represents a chat message
type Message struct {
	Role         string     // "user", "assistant", "system"
	Content      string     // The message content
	ToolCalls    []ToolCall // Tool calls made by the assistant
	ToolResultID string     // If non-empty, this message is a tool result for this call ID
	IsError      bool       // Whether the tool result indicates an error
}

// Tool represents a tool that can be called by the LLM
type Tool struct {
	Name        string         // Tool name
	Description string         // What the tool does
	Parameters  map[string]any // JSON schema for parameters
}

// ToolCall represents a tool invocation
type ToolCall struct {
	ID        string         // Unique call ID
	Name      string         // Tool name
	Arguments map[string]any // Tool arguments
}

// Response represents an LLM response
type Response struct {
	Content   string     // Text response
	ToolCalls []ToolCall // Any tool calls requested
	Done      bool       // Whether the conversation is complete
}

// MediaItem represents a generic media item (movie, show, book)
type MediaItem struct {
	ID          string            // Backend-specific ID
	Title       string            // Item title
	Year        int               // Release year
	Type        string            // "movie", "tv", "book"
	Description string            // Plot/summary
	PosterURL   string            // URL to poster image
	Rating      float64           // Rating (0-10)
	Metadata    map[string]string // Additional metadata
}

// MediaStatus represents the status of a media item in the backend
type MediaStatus struct {
	ItemID         string  // Media item ID
	Status         string  // "wanted", "downloading", "downloaded", "available"
	Progress       float64 // Download progress (0-100)
	ETA            int64   // Estimated time remaining in seconds
	QualityProfile string  // Quality profile name
}

// Torrent represents a torrent download
type Torrent struct {
	Hash          string  // Torrent hash
	Name          string  // Torrent name
	Size          int64   // Total size in bytes
	Progress      float64 // Progress percentage (0-100)
	Status        string  // "downloading", "seeding", "paused", "error"
	DownloadSpeed int64   // Download speed in bytes/sec
	UploadSpeed   int64   // Upload speed in bytes/sec
	ETA           int64   // Estimated time remaining in seconds
}

// TorrentProgress represents detailed progress information
type TorrentProgress struct {
	Hash          string  // Torrent hash
	Progress      float64 // Progress percentage (0-100)
	Downloaded    int64   // Bytes downloaded
	Total         int64   // Total bytes
	DownloadSpeed int64   // Current download speed
	ETA           int64   // Estimated seconds remaining
}
