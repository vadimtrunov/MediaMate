# 🎬 MediaMate

**AI-powered media assistant for your home server.** Talk to it in natural language — it finds, downloads, and organizes movies, TV shows, and books. Stream to your TV. All self-hosted on a Raspberry Pi.

```
You: download something like Interstellar but darker
Bot: Based on your taste, here are some picks:
     1. Arrival (2016) — 7.9 ⭐
     2. Annihilation (2018) — 6.8 ⭐
     3. Ad Astra (2019) — 6.4 ⭐
     Want me to download any of these?
You: grab 1 and 2
Bot: ✅ Added Arrival and Annihilation. Searching for releases now.
```

---

## What It Does

MediaMate is a single Go binary that connects an LLM brain to your media stack. You chat via Telegram (or terminal) — it handles the rest.

- **Download** — "скачай Дюну 2" → searches Radarr → downloads via qBittorrent → ready to stream
- **Recommend** — "recommend thrillers like Se7en" → TMDb recommendations with ratings
- **Discover** — "best sci-fi of 2024" → curated by genre, year, rating
- **Monitor** — "what's downloading?" → live torrent progress
- **Stream** — downloads land in Jellyfin/Plex automatically → watch on any TV

Speaks any language. The LLM detects yours and responds accordingly.

## Architecture

```
You (Telegram / CLI)
    │
    ▼
MediaMate (Go binary)
    │
    ├── LLM ──── Claude / OpenAI / Ollama (pluggable)
    │
    ├── Media ── Radarr / Sonarr / Readarr (pluggable)
    │
    ├── Torrent ─ qBittorrent / Transmission / Deluge (pluggable)
    │
    └── Stream ── Jellyfin / Plex (pluggable)
```

Everything is an interface. Swap any component without touching the rest.

## Quick Start

```bash
# Install
curl -fsSL https://raw.githubusercontent.com/<owner>/mediamate/main/scripts/install.sh | bash

# Interactive setup — picks your services, generates Docker Compose
mediamate stack init

# Launch everything
mediamate stack up

# Start chatting
mediamate chat
```

Or with Docker directly:

```bash
docker pull ghcr.io/<owner>/mediamate:latest
```

## Stack

MediaMate manages a Docker Compose stack. You choose what to run:

| Component | Options | Default |
|-----------|---------|---------|
| Movies | Radarr | ✅ enabled |
| TV Shows | Sonarr | ✅ enabled |
| Books | Readarr | optional |
| Indexers | Prowlarr | ✅ enabled |
| Torrents | qBittorrent, Transmission, Deluge | qBittorrent |
| Streaming | Jellyfin, Plex | Jellyfin |
| LLM | Claude, OpenAI, Ollama | Claude |
| Chat | Telegram, CLI, (Discord, Matrix) | Telegram + CLI |
| VPN | Gluetun | optional |

Runs on **Raspberry Pi 5** (ARM64) and any Linux amd64 box.

## Configuration

Single YAML file. Secrets via environment variables.

```yaml
llm:
  provider: claude
  claude:
    api_key: ${ANTHROPIC_API_KEY}  # or use OAuth with your Claude subscription

frontends:
  telegram:
    enabled: true
    token: ${TELEGRAM_TOKEN}
  cli:
    enabled: true

backends:
  radarr:
    enabled: true
    url: http://radarr:7878
  sonarr:
    enabled: true
    url: http://sonarr:8989

media_server:
  type: jellyfin

torrent:
  client: qbittorrent
```

See [full config reference](configs/mediamate.example.yaml) and [spec](SPEC.md).

## Requirements

- Docker and Docker Compose
- One LLM API key (Anthropic, OpenAI, or local Ollama)
- Telegram bot token (from [@BotFather](https://t.me/BotFather))
- Free [TMDb API key](https://www.themoviedb.org/settings/api) for recommendations
- Storage for your media

## Roadmap

- [x] Spec & architecture
- [ ] **v0.1** — MVP: Claude + Radarr + qBittorrent + Jellyfin + Telegram + CLI
- [ ] **v0.2** — Sonarr, Readarr, Claude OAuth, history persistence
- [ ] **v0.3** — OpenAI, Ollama, Transmission, Deluge
- [ ] **v0.4** — Discord, Matrix, notifications
- [ ] **v1.0** — Polish, tests, docs site

## License

MIT
