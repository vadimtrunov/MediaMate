# MediaMate Development Roadmap

## Implementation Philosophy

We build incrementally — each phase is a fully working product. Start with the minimum, gradually add functionality.

---

## Phase 0: Foundation (Week 1-2)

### 0.1 Project Structure ✅
- [x] Set up Go module structure
- [x] Define packages: `cmd/`, `internal/`, `pkg/`
- [x] Set up CI/CD (GitHub Actions)
- [x] Makefile for builds
- [ ] Docker multi-stage build for ARM64 and AMD64 (→ Phase 7.3)

### 0.2 Core Interfaces ✅
```go
// internal/core/interfaces.go
type LLMProvider interface {}
type MediaBackend interface {}
type TorrentClient interface {}
type MediaServer interface {}
type Frontend interface {}
```

### 0.3 Configuration System ✅
- [x] YAML configuration (`internal/config/`)
- [x] Environment variables override
- [x] Config validation
- [x] Example configs: `configs/mediamate.example.yaml`

### 0.4 Logging & Observability ✅
- [x] Structured logging (zerolog or slog)
- [x] Log levels
- [x] Context-aware logging

**Deliverable:** Empty project skeleton with interfaces and config

---

## Phase 1: MVP Backend (Week 3-5)

### 1.1 LLM Integration (Claude) ✅
- [x] `internal/llm/claude/` — Claude API client
- [x] Tool calling support (Claude function calling)
- [x] Retry logic and error handling
- [ ] Rate limiting (→ Phase 7.4)
- [x] Tests with mock LLM

**Tools for Claude:**
```go
// Define tools that the LLM can invoke
- SearchMovie(query string) []Movie
- DownloadMovie(movieID int) Status
- GetDownloadStatus() []Download
- RecommendSimilar(movieID int) []Movie
```

### 1.2 TMDb Integration ✅
- [x] `internal/metadata/tmdb/` — TMDb API client
- [x] Movie search
- [x] Recommendations
- [x] Movie details (rating, description, posters)
- [x] Caching for popular queries

### 1.3 Radarr Integration ✅
- [x] `internal/backend/radarr/` — Radarr API client
- [x] Release search
- [x] Adding movies to library
- [x] Status monitoring
- [x] Quality profiles

### 1.4 qBittorrent Integration ✅
- [x] `internal/torrent/qbittorrent/` — qBittorrent Web API client
- [x] Getting torrent list
- [x] Download progress
- [x] Management (pause/resume)

### 1.5 Core Orchestration ✅
- [x] `internal/agent/` — AI agent orchestrator
- [x] Intent parsing from LLM
- [x] Mapping tool calls to actual API calls
- [x] Conversation state management
- [x] Multi-step flows (search → confirm → download)

**Deliverable:** Backend that understands requests and manages media via Radarr + qBittorrent

---

## Phase 2: CLI Frontend (Week 6)

### 2.1 Interactive CLI ✅
- [x] `cmd/mediamate/chat.go` — interactive chat in terminal
- [x] Message history (arrows up/down)
- [x] Beautiful output (bubbles/lipgloss)
- [ ] Streaming responses from LLM (→ Phase 7.4)
- [x] Colored output (movies, statuses)

### 2.2 CLI Commands ✅
```bash
mediamate chat              # Interactive chat
mediamate query "download Dune"  # One-off query
mediamate status            # Download status
mediamate config validate   # Config validation
```

**Deliverable:** Full chat with the bot via terminal

---

## Phase 3: Telegram Frontend (Week 7-8)

### 3.1 Telegram Bot ✅
- [x] `internal/frontend/telegram/` — Telegram Bot API
- [x] Text message handling
- [x] Inline keyboard for confirmations
- [x] Streaming responses (typing indicator)
- [x] Multi-user support (user isolation)
- [x] Admin whitelist (only allowed user IDs)

### 3.2 Rich Media ✅
- [x] Sending movie posters
- [x] Recommendation formatting (Markdown)
- [ ] Progress bars for downloads (→ Phase 5.2)
- [x] Callback buttons (download 1/2/3)

**Deliverable:** v0.1 — Full MVP with Telegram + CLI

---

## Phase 4: Jellyfin & Stack Management (Week 9-10)

### 4.1 Jellyfin Integration ✅
- [x] `internal/mediaserver/jellyfin/` — Jellyfin API
- [x] Movie availability check
- [x] Generating watch links
- [ ] Webhook for new content notifications (deferred to Phase 5.1)

### 4.2 Docker Compose Stack ✅
- [x] `mediamate stack init` — interactive wizard
  - Component selection (Radarr, Sonarr, Jellyfin, etc.)
  - Generate `docker-compose.yml`
  - Generate `.env` with secrets
- [x] `mediamate stack up` — start the stack
- [x] `mediamate stack down` — stop the stack
- [x] Health checks for all services

### 4.3 Setup Wizard ✅
- [x] Automatic Radarr setup (root folders, download client)
- [x] Automatic Prowlarr setup (application, download client, indexer proxy)
- [x] Linking Radarr <-> Prowlarr <-> qBittorrent
- [x] Health check polling before configuration
- [x] API key extraction from config.xml files
- [x] Auto-update .env and mediamate.yaml with real API keys
- [x] qBittorrent save path configuration
- [x] Idempotent operations (skip if already configured)

**Deliverable:** `mediamate stack init` → fully working stack in 5 minutes

---

## Phase 5: Notifications

### 5.1 Download Notifications ✅
- [x] Webhook endpoint for Radarr events
- [x] Telegram notification: "Interstellar is ready to watch!"
- [x] Direct link to Jellyfin
- [x] Webhook secret validation
- [x] Auto-register webhook in Radarr via setup wizard
- [ ] Sonarr webhook support (→ Phase 6)

### 5.2 Download Progress ✅
- [x] Progress bars for active downloads in Telegram
- [x] Periodic status updates

**Deliverable:** End-to-end flow — request movie → get notified when ready

---

## Phase 6: Sonarr & Readarr

### 6.1 Sonarr Support
- [ ] `internal/backend/sonarr/` — Sonarr API client
- [ ] TV show search
- [ ] Season/episode monitoring
- [ ] Update LLM tools for TV shows

### 6.2 Readarr Support
- [ ] `internal/backend/readarr/` — Readarr API client
- [ ] Book search
- [ ] E-book formats (epub, mobi, pdf)

### 6.3 Unified Search
- [ ] LLM determines content type (movie/show/book)
- [ ] Unified search interface
- [ ] Result prioritization

**Deliverable:** Support for movies, TV shows, books

---

## Phase 7: Polish & Release

### 7.1 Testing
- [ ] Unit tests (>80% coverage)
- [ ] Integration tests with mock API
- [ ] End-to-end tests in Docker
- [ ] Tests on ARM64 (Raspberry Pi 5)

### 7.2 Documentation
- [ ] Docs site (Hugo/MkDocs)
- [ ] Step-by-step guides
- [ ] API reference
- [ ] Troubleshooting

### 7.3 Install Script
- [ ] One-liner install: `curl ... | bash`
- [ ] Auto-detection of ARM64/AMD64
- [ ] Systemd service
- [ ] Docker multi-stage build for ARM64 and AMD64

### 7.4 Performance
- [ ] Memory optimization (important for RPi)
- [ ] Parallel API requests
- [ ] Graceful shutdown

**Deliverable:** v1.0 — Production-ready release

---

## Phase 8: Advanced Features

### 8.1 Conversation History
- [ ] SQLite for history storage
- [ ] Previous request context
- [ ] "Remember, I asked you to download that movie?" → LLM searches history

### 8.2 Smart Recommendations
- [ ] Personal recommendations based on history
- [ ] "What to watch tonight?" → analysis of likes/dislikes
- [ ] Integration with Jellyfin viewing history

**Deliverable:** Smart assistant with memory

---

## Phase 9: Alternative Providers

### 9.1 OpenAI Support
- [ ] `internal/llm/openai/` — OpenAI client
- [ ] GPT-4 function calling
- [ ] Switching via config

### 9.2 Ollama Support
- [ ] `internal/llm/ollama/` — Ollama client
- [ ] Local models (Llama 3, Mistral)
- [ ] Auto-detection of available models

### 9.3 Alternative Torrent Clients
- [ ] `internal/torrent/transmission/`
- [ ] `internal/torrent/deluge/`
- [ ] Unified `TorrentClient` interface

**Deliverable:** Flexible component selection

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

### Claude OAuth
- Authorization via Claude.ai subscription
- Reduced costs for users with a subscription

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

- **Phases 0-4:** MVP (v0.1) — Claude + Radarr + Telegram + Jellyfin + Stack ✅
- **Phase 5.1:** Download Notifications ✅ — webhook + Telegram alert
- **Phase 5.2:** Download Progress ✅ — live progress bars in Telegram
- **Phase 6:** Sonarr & Readarr — TV shows and books
- **Phase 7:** Polish & release (v1.0) — tests, docs, install script
- **Phase 8-9:** Advanced features & alternative providers

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
