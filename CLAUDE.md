# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

MediaMate — chat-based media assistant powered by LLM. Pluggable architecture: swap LLM providers (Claude/OpenAI/Ollama), media backends (Radarr/Sonarr/Readarr), torrent clients (qBittorrent/Transmission/Deluge), streaming servers (Jellyfin/Plex), and frontends (Telegram/CLI). Self-hosted, Docker-first, ARM64-first (Raspberry Pi 5 target).

**Status:** Early-stage — infrastructure (CI/CD, linting, release automation) is set up, but no Go source code exists yet. Phase 0 (foundation) is next.

## Build & Development Commands

```bash
make build              # Build binary to bin/mediamate
make run                # Run application
make run-dev            # Run with hot reload (air)
make test               # Unit tests with -race and coverage
make test-coverage      # Generate coverage.html
make test-integration   # Integration tests (build tag)
make bench              # Benchmarks with memory stats
make lint               # golangci-lint (45+ linters)
make lint-fix           # Auto-fix lint issues
make fmt                # Format with gofumpt
make imports            # Organize imports with goimports
make prepare            # Pre-commit prep: fmt → imports → lint-fix → tidy
make check              # Full check: fmt → imports → lint → vet → test
make ci                 # CI pipeline: deps-verify → lint → test
make install-tools      # Install all dev tools
make install-hooks      # Install git pre-commit hook
```

Run a single test: `go test -v -run TestName ./path/to/package/...`

## Architecture

```
cmd/mediamate/main.go          # Entry point (planned)
internal/
  core/interfaces.go           # Core interfaces: LLMProvider, MediaBackend,
                                #   TorrentClient, MediaServer, Frontend
  config/                      # YAML + env var configuration
  llm/                         # LLM provider implementations
  media/                       # Radarr, Sonarr, Readarr clients
  torrent/                     # Torrent client implementations
  stream/                      # Jellyfin, Plex clients
  frontend/                    # Telegram bot, CLI (Cobra + Bubble Tea)
```

Interface-first design — every external service sits behind an interface for testability and swappability. All API clients should be mockable.

## Code Style & Linting

- **Import order:** stdlib → third-party → `github.com/vadimtrunov/MediaMate`
- **Line length:** 140 chars max
- **Function length:** 120 lines / 60 statements max
- **Cyclomatic complexity:** max 20, cognitive complexity: max 30
- **Nesting depth:** max 5
- **Duplicate threshold:** 150 tokens
- **Formatter:** gofumpt (not gofmt)
- **Magic numbers:** flagged by `mnd` linter (use named constants)
- Tests have relaxed rules for: mnd, funlen, gocognit, gocyclo, dupl, lll, goconst, errcheck

## Key Design Principles

- Config-driven (YAML + env vars), no hardcoded values
- CGO_ENABLED=0 for static binaries
- Fail-safe: graceful degradation when services are unavailable
- Multi-language: LLM detects and responds in user's language
- Target: <5s response time, <100MB RAM

## CI/CD

- **ci.yml:** Tests (Go 1.22 + 1.23 matrix), lint, build (linux amd64/arm64)
- **release.yml:** GoReleaser on `v*` tags — multi-platform binaries + Docker images
- **docker.yml:** Multi-arch Docker builds pushed to `ghcr.io/vadimtrunov/mediamate`
- **security.yml:** CodeQL + Trivy + gosec (weekly + on PRs)
- **CodeRabbit:** AI code review on PRs (profile: "chill")
