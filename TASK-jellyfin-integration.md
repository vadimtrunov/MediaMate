# Feature: Jellyfin Integration

## Task ID
phase-4.1-jellyfin

## Description
Implement Jellyfin API client in internal/mediaserver/jellyfin/ and integrate with the agent so LLM can check movie availability and generate watch links.

## Acceptance Criteria
- [x] Jellyfin client implements core.MediaServer (IsAvailable, GetLink, GetLibraryItems, Name)
- [x] Client uses httpclient for HTTP requests with retry
- [x] IsAvailable searches for a movie by name in Jellyfin library
- [x] GetLink generates a watch URL
- [x] GetLibraryItems returns all movies from the library
- [x] Agent has check_availability and get_watch_link tools
- [x] Tests with httptest mocks
- [x] Config already supports jellyfin.url and api_key — use existing

## Plan
- [x] Step 1: Jellyfin types — internal/mediaserver/jellyfin/types.go
- [x] Step 2: Jellyfin client — internal/mediaserver/jellyfin/client.go (implements core.MediaServer)
- [x] Step 3: Jellyfin client tests — internal/mediaserver/jellyfin/client_test.go
- [x] Step 4: Agent integration — add mediaServer field, tool definitions, handlers
- [x] Step 5: Wiring — initMediaServer() in helpers.go, pass to agent.New()
- [x] Step 6: Build & test — go build/test/vet

## Constraints
- Follow patterns from radarr/tmdb/qbittorrent clients
- Use httpclient with retry
- Wrap errors with fmt.Errorf and context
- No new go.mod dependencies
- Tests via httptest.NewServer

## Context
- Interface: internal/core/interfaces.go — MediaServer
- Client pattern: internal/backend/radarr/client.go
- Agent: internal/agent/agent.go, tools.go, handlers.go
- Config: internal/config/config.go — JellyfinConfig already exists
- Wiring: cmd/mediamate/helpers.go — initServices()

## Feedback Log

### Step 1: Jellyfin types (2026-02-24)
- Result: OK
- Created types.go with jellyfinItemsResponse and jellyfinItem

### Step 2: Jellyfin client (2026-02-24)
- Result: OK
- Implemented all MediaServer methods, X-Emby-Token auth, get() helper

### Step 3: Tests (2026-02-24)
- Result: OK
- 7 tests, all passing

### Step 4: Agent integration (2026-02-24)
- Result: OK
- Added mediaServer field, check_availability + get_watch_link tools and handlers
- Updated system prompt

### Step 5: Wiring (2026-02-24)
- Result: OK with fixes
- Added initMediaServer() and jellyfin import
- Had to fix 2 additional agent.New() call sites in chat_test.go and session_test.go

### Step 6: Build & test (2026-02-24)
- Result: OK
- go build ./... — clean
- go test ./... — all pass
- go vet ./... — clean

---
*Created: 2026-02-24*
*Status: complete*
