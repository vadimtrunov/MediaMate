# MediaMate

> **Chat-based media assistant powered by LLM**

Download movies, TV shows, and books through natural conversation. Pluggable architecture — swap LLM providers, messengers, media backends, and torrent clients. Self-hosted, Docker-first.

[![Go Version](https://img.shields.io/github/go-mod/go-version/vadimtrunov/MediaMate)](https://golang.org/dl/)
[![License](https://img.shields.io/github/license/vadimtrunov/MediaMate)](LICENSE)
[![Build Status](https://github.com/vadimtrunov/MediaMate/workflows/CI/badge.svg)](https://github.com/vadimtrunov/MediaMate/actions)

---

## 🎯 What is MediaMate?

MediaMate is an AI-powered media assistant that lets you manage your home media library through natural language. Just tell it what you want to watch, and it handles everything:

- 🔍 **Search** for movies/shows/books using natural language
- ⬇️ **Download** content automatically via Radarr/Sonarr/Readarr
- 📊 **Monitor** download progress
- 🎬 **Stream** directly from Jellyfin/Plex
- 💬 **Chat** via Telegram, CLI, or Discord

**Example:**
```text
You: "Download Dune Part 2 in 4K"
MediaMate: "Found Dune: Part Two (2024). Adding to download queue with 4K quality profile..."
```

---

## ✨ Features

- **🤖 Multiple LLM Providers:** Claude, OpenAI GPT-4, Ollama (local models)
- **📱 Multiple Frontends:** Telegram Bot, CLI, Discord (planned)
- **🎬 Media Management:** Radarr (movies), Sonarr (TV), Readarr (books)
- **📦 Torrent Clients:** qBittorrent, Transmission, Deluge
- **🎥 Media Servers:** Jellyfin, Plex
- **🔧 Fully Configurable:** YAML config + environment variables
- **🐳 Docker Ready:** Multi-arch support (ARM64 + AMD64)
- **🏠 Raspberry Pi 5 Optimized**

---

## 🚀 Quick Start

### Prerequisites

- Go 1.23+ (for building from source)
- Docker & Docker Compose (recommended)
- API keys for:
  - Claude/OpenAI (or local Ollama)
  - TMDb (free at https://www.themoviedb.org/settings/api)

### Installation

#### Option 1: Build from source

```bash
# Clone the repository
git clone https://github.com/vadimtrunov/MediaMate.git
cd MediaMate

# Build
make build

# Copy example config
cp configs/mediamate.example.yaml configs/mediamate.yaml

# Edit config with your API keys
nano configs/mediamate.yaml

# Run
./bin/mediamate
```

#### Option 2: Docker (coming soon)

```bash
docker run -d \
  -v ./config:/config \
  -v ./data:/data \
  -e MEDIAMATE_LLM_API_KEY=your-key \
  ghcr.io/vadimtrunov/mediamate:latest
```

---

## 📋 Configuration

Create `configs/mediamate.yaml` from the example:

```yaml
llm:
  provider: "claude"
  api_key: "your-api-key"
  model: "claude-3-5-sonnet-20241022"

tmdb:
  api_key: "your-tmdb-key"

radarr:
  url: "http://localhost:7878"
  api_key: "your-radarr-key"

qbittorrent:
  url: "http://localhost:8080"
  username: "your-qbittorrent-username"
  password: "your-qbittorrent-password"

telegram:
  bot_token: "your-bot-token"
```

See `configs/mediamate.example.yaml` for all options.

### Environment Variables

Supported config values can be overridden with environment variables:

```bash
export MEDIAMATE_LLM_API_KEY="sk-..."
export MEDIAMATE_TELEGRAM_BOT_TOKEN="123456:ABC..."
```

See `configs/mediamate.example.yaml` for the complete list of supported environment variables.

---

## 🛠️ Development

### Project Structure

```text
MediaMate/
├── cmd/mediamate/          # Main application entry point
├── internal/
│   ├── core/               # Core interfaces
│   ├── config/             # Configuration & logging
│   ├── llm/                # LLM provider implementations
│   ├── metadata/           # TMDb, OMDb, etc.
│   ├── backend/            # Radarr, Sonarr, Readarr
│   ├── torrent/            # qBittorrent, Transmission
│   ├── mediaserver/        # Jellyfin, Plex
│   ├── frontend/           # Telegram, CLI, Discord
│   └── agent/              # AI orchestration logic
├── pkg/                    # Public libraries
├── configs/                # Configuration files
└── docs/                   # Documentation

```

### Available Make Targets

```bash
make help           # Show all available commands
make build          # Build binary
make test           # Run tests
make lint           # Run linters
make fmt            # Format code
make pre-commit     # Run all checks before commit
```

### Running Tests

```bash
make test           # Unit tests
make test-coverage  # With coverage report
make test-integration  # Integration tests
```

---

## 📖 Documentation

- [Roadmap](docs/ROADMAP.md) - Development roadmap
- [Specification](docs/SPEC.md) - Technical specification
- [Git Workflow](docs/GIT_WORKFLOW.md) - Contributing guidelines
- [Linters Setup](docs/LINTERS.md) - Code quality tools

---

## 🗺️ Roadmap

**Current Status:** Phase 0 (Foundation) ✅

- [x] Phase 0: Foundation (Project structure, config, logging)
- [ ] Phase 1: MVP Backend (Claude + Radarr + qBittorrent)
- [ ] Phase 2: CLI Frontend
- [ ] Phase 3: Telegram Frontend
- [ ] Phase 4: Jellyfin Integration
- [ ] Phase 5: Sonarr & Readarr Support
- [ ] Phase 6: Alternative Providers
- [ ] Phase 7: Advanced Features
- [ ] Phase 8: v1.0 Release

See [ROADMAP.md](docs/ROADMAP.md) for details.

---

## 🤝 Contributing

Contributions are welcome! Please read our [Git Workflow Guide](docs/GIT_WORKFLOW.md) first.

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Run `make pre-commit` to ensure quality
5. Commit with conventional commits (`git commit -m 'feat: add amazing feature'`)
6. Push to the branch
7. Open a Pull Request

---

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## 🙏 Acknowledgments

- Built with [Claude API](https://www.anthropic.com/claude)
- Powered by [Radarr](https://radarr.video/), [Sonarr](https://sonarr.tv/), and [Jellyfin](https://jellyfin.org/)
- Metadata from [TMDb](https://www.themoviedb.org/)

---

## 📬 Support

- **Issues:** [GitHub Issues](https://github.com/vadimtrunov/MediaMate/issues)
- **Discussions:** [GitHub Discussions](https://github.com/vadimtrunov/MediaMate/discussions)

---

**Made with ❤️ for the self-hosted community**
