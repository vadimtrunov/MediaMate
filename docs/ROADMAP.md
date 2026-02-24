# MediaMate Development Roadmap

## Implementation Philosophy

We build incrementally — each phase is a fully working product. Start with the minimum, gradually add functionality.

---

## Phase 0: Foundation (Week 1-2)

### 0.1 Project Structure
- [ ] Set up Go module structure
- [ ] Define packages: `cmd/`, `internal/`, `pkg/`
- [ ] Set up CI/CD (GitHub Actions)
- [ ] Makefile for builds
- [ ] Docker multi-stage build for ARM64 and AMD64

### 0.2 Core Interfaces
```go
// internal/core/interfaces.go
type LLMProvider interface {}
type MediaBackend interface {}
type TorrentClient interface {}
type MediaServer interface {}
type Frontend interface {}
```

### 0.3 Configuration System
- [ ] YAML configuration (`internal/config/`)
- [ ] Environment variables override
- [ ] Config validation
- [ ] Example configs: `configs/mediamate.example.yaml`

### 0.4 Logging & Observability
- [ ] Structured logging (zerolog or slog)
- [ ] Log levels
- [ ] Context-aware logging

**Deliverable:** Empty project skeleton with interfaces and config

---

## Phase 1: MVP Backend (Week 3-5)

### 1.1 LLM Integration (Claude)
- [ ] `internal/llm/claude/` — Claude API client
- [ ] Tool calling support (Claude function calling)
- [ ] Retry logic and error handling
- [ ] Rate limiting
- [ ] Tests with mock LLM

**Tools for Claude:**
```go
// Define tools that the LLM can invoke
- SearchMovie(query string) []Movie
- DownloadMovie(movieID int) Status
- GetDownloadStatus() []Download
- RecommendSimilar(movieID int) []Movie
```

### 1.2 TMDb Integration
- [ ] `internal/metadata/tmdb/` — TMDb API client
- [ ] Movie search
- [ ] Recommendations
- [ ] Movie details (rating, description, posters)
- [ ] Caching for popular queries

### 1.3 Radarr Integration
- [ ] `internal/backend/radarr/` — Radarr API client
- [ ] Release search
- [ ] Adding movies to library
- [ ] Status monitoring
- [ ] Quality profiles

### 1.4 qBittorrent Integration
- [ ] `internal/torrent/qbittorrent/` — qBittorrent Web API client
- [ ] Getting torrent list
- [ ] Download progress
- [ ] Management (pause/resume)

### 1.5 Core Orchestration
- [ ] `internal/agent/` — AI agent orchestrator
- [ ] Intent parsing from LLM
- [ ] Mapping tool calls to actual API calls
- [ ] Conversation state management
- [ ] Multi-step flows (search → confirm → download)

**Deliverable:** Backend that understands requests and manages media via Radarr + qBittorrent

---

## Phase 2: CLI Frontend (Week 6)

### 2.1 Interactive CLI
- [ ] `cmd/mediamate/chat.go` — interactive chat in terminal
- [ ] Message history (arrows up/down)
- [ ] Beautiful output (bubbles/lipgloss)
- [ ] Streaming responses from LLM
- [ ] Colored output (movies, statuses)

### 2.2 CLI Commands
```bash
mediamate chat              # Interactive chat
mediamate query "download Dune"  # One-off query
mediamate status            # Download status
mediamate config validate   # Config validation
```

**Deliverable:** Full chat with the bot via terminal

---

## Phase 3: Telegram Frontend (Week 7-8)

### 3.1 Telegram Bot
- [ ] `internal/frontend/telegram/` — Telegram Bot API
- [ ] Text message handling
- [ ] Inline keyboard for confirmations
- [ ] Streaming responses (typing indicator)
- [ ] Multi-user support (user isolation)
- [ ] Admin whitelist (only allowed user IDs)

### 3.2 Rich Media
- [ ] Sending movie posters
- [ ] Recommendation formatting (Markdown)
- [ ] Progress bars for downloads
- [ ] Callback buttons (download 1/2/3)

**Deliverable:** v0.1 — Full MVP with Telegram + CLI

---

## Phase 4: Jellyfin & Stack Management (Week 9-10)

### 4.1 Jellyfin Integration
- [ ] `internal/mediaserver/jellyfin/` — Jellyfin API
- [ ] Movie availability check
- [ ] Generating watch links
- [ ] Webhook for new content notifications

### 4.2 Docker Compose Stack
- [ ] `mediamate stack init` — interactive wizard
  - Component selection (Radarr, Sonarr, Jellyfin, etc)
  - Generate `docker-compose.yml`
  - Generate `.env` with secrets
- [ ] `mediamate stack up` — start the stack
- [ ] `mediamate stack down` — stop the stack
- [ ] Health checks for all services

### 4.3 Setup Wizard
- [ ] Automatic Radarr setup (quality profiles, root folders)
- [ ] Automatic Prowlarr + indexers setup
- [ ] Linking Radarr <-> Prowlarr <-> qBittorrent
- [ ] Connection tests

**Deliverable:** `mediamate stack init` → fully working stack in 5 minutes

---

## Phase 5: Sonarr & Readarr (Week 11-12)

### 5.1 Sonarr Support
- [ ] `internal/backend/sonarr/` — Sonarr API client
- [ ] TV show search
- [ ] Season/episode monitoring
- [ ] Update LLM tools for TV shows

### 5.2 Readarr Support
- [ ] `internal/backend/readarr/` — Readarr API client
- [ ] Book search
- [ ] E-book formats (epub, mobi, pdf)

### 5.3 Unified Search
- [ ] LLM determines content type (movie/show/book)
- [ ] Unified search interface
- [ ] Result prioritization

**Deliverable:** v0.2 — Support for movies, TV shows, books

---

## Phase 6: Alternative Providers (Week 13-14)

### 6.1 OpenAI Support
- [ ] `internal/llm/openai/` — OpenAI client
- [ ] GPT-4 Turbo function calling
- [ ] Switching via config

### 6.2 Ollama Support
- [ ] `internal/llm/ollama/` — Ollama client
- [ ] Local models (Llama 3, Mistral)
- [ ] Auto-detection of available models

### 6.3 Alternative Torrent Clients
- [ ] `internal/torrent/transmission/`
- [ ] `internal/torrent/deluge/`
- [ ] Unified `TorrentClient` interface

**Deliverable:** v0.3 — Flexible component selection

---

## Phase 7: Advanced Features (Week 15-16)

### 7.1 Conversation History
- [ ] SQLite for history storage
- [ ] Previous request context
- [ ] "Remember, I asked you to download that movie?" → LLM searches history

### 7.2 Claude OAuth
- [ ] Authorization via Claude.ai subscription
- [ ] Reduced costs for users with a subscription

### 7.3 Notifications
- [ ] Webhook from Radarr/Sonarr when download completes
- [ ] Telegram notification: "Interstellar is ready to watch!"
- [ ] Direct link to Jellyfin

### 7.4 Smart Recommendations
- [ ] Personal recommendations based on history
- [ ] "What to watch tonight?" → analysis of likes/dislikes
- [ ] Integration with Jellyfin viewing history

**Deliverable:** v0.4 — Smart assistant with memory

---

## Phase 8: Polish & Release (Week 17-20)

### 8.1 Testing
- [ ] Unit tests (>80% coverage)
- [ ] Integration tests with mock API
- [ ] End-to-end tests in Docker
- [ ] Tests on ARM64 (Raspberry Pi 5)

### 8.2 Documentation
- [ ] Docs site (Hugo/MkDocs)
- [ ] Step-by-step guides
- [ ] API reference
- [ ] Troubleshooting
- [ ] Video demos

### 8.3 Install Script
- [ ] One-liner install: `curl ... | bash`
- [ ] Auto-detection of ARM64/AMD64
- [ ] Systemd service
- [ ] Auto-updates

### 8.4 Performance
- [ ] Benchmarks for LLM response time
- [ ] Memory optimization (important for RPi)
- [ ] Parallel API requests
- [ ] Graceful shutdown

**Deliverable:** v1.0 — Production-ready release

---

## Future Ideas (Post v1.0)

### Discord & Matrix Frontends
- Similar to Telegram, but for other platforms

### Web UI
- React/Vue dashboard
- Visual library browser
- Download manager
- Settings UI

### Advanced AI Features
- Voice input (Whisper)
- Playlist generation ("create a movie collection for the weekend")
- Automatic content ("every Friday download the new episode of The Mandalorian")

### Multi-instance Support
- Multiple Radarr instances (4K + 1080p)
- Quality profiles based on playback device

### Subtitle Management
- Bazarr integration
- Automatic subtitle download in the desired language

---

## Success Metrics

### v0.1 (MVP)
- [ ] Can ask to download a movie via Telegram
- [ ] Movie appears in Jellyfin within ~10-30 minutes
- [ ] Works on Raspberry Pi 5

### v1.0 (Production)
- [ ] <5 seconds for LLM response
- [ ] <100MB RAM on ARM64
- [ ] Install in 1 command
- [ ] Documentation on par with Homelab projects

---

## Tech Stack Summary

| Layer | Technology |
|-------|-----------|
| Language | Go 1.22+ |
| LLM | Claude API / OpenAI / Ollama |
| Metadata | TMDb API |
| Media Management | Radarr / Sonarr / Readarr |
| Torrents | qBittorrent / Transmission / Deluge |
| Streaming | Jellyfin / Plex |
| Chat | Telegram Bot API |
| CLI | Cobra + Bubble Tea |
| Storage | SQLite (conversation history) |
| Config | YAML + env vars |
| Deployment | Docker + Docker Compose |
| CI/CD | GitHub Actions |
| Docs | MkDocs Material |

---

## Development Principles

1. **Interface-first** — Every component behind an interface, easy to swap
2. **Test-friendly** — All API clients are mockable
3. **ARM64 first** — Raspberry Pi 5 as primary target
4. **Config-driven** — No hardcoded values
5. **Fail-safe** — Graceful degradation if a service is unavailable
6. **User-centric** — Ease of setup is more important than flexibility

---

## Timeline

- **Weeks 1-8:** MVP (v0.1) — Claude + Radarr + Telegram
- **Weeks 9-12:** Core features (v0.2) — Sonarr, Readarr, Jellyfin
- **Weeks 13-16:** Alternative providers (v0.3-v0.4)
- **Weeks 17-20:** Polish & release (v1.0)

**Total:** ~5 months to v1.0

---

## Getting Started (For Developers)

```bash
# Clone
git clone https://github.com/<owner>/mediamate
cd mediamate

# Install dependencies
go mod download

# Run locally
cp configs/mediamate.example.yaml configs/mediamate.yaml
# Edit configs/mediamate.yaml with your API keys
go run cmd/mediamate/main.go chat

# Build
make build

# Run tests
make test

# Build Docker image
make docker-build
```
