# Feature: Stack Management (up, down, status)

## Task ID
phase-4.2-part2-stack-management

## Description
Implement `mediamate stack up`, `mediamate stack down`, and `mediamate stack status` commands.
Add Docker healthcheck directives to docker-compose.yml template.

## Acceptance Criteria
- [x] `mediamate stack up` starts the stack via `docker compose up -d`
- [x] `mediamate stack down` stops the stack via `docker compose down`
- [x] `mediamate stack status` shows container states + HTTP health probes
- [x] healthcheck: directives added to docker-compose.yml.tmpl for all services
- [x] Beautiful output using lipgloss styles from helpers.go
- [x] Clear error if Docker/docker compose not found
- [x] Tests for health check logic via httptest
- [x] `stack up` and `stack down` support --file/-f flag
- [x] go build, go test, go vet — clean

## Plan
- [x] Step 1: Docker compose runner — internal/stack/compose.go
- [x] Step 2: Health checker — internal/stack/health.go
- [x] Step 3: Health checker tests — internal/stack/health_test.go
- [x] Step 4: Healthcheck directives in docker-compose.yml.tmpl
- [x] Step 5: CLI commands — cmd/mediamate/stack_cmd.go (stack up, down, status)
- [x] Step 6: Build verification — go build, go test, go vet

## Constraints
- NO Docker SDK — os/exec only for `docker compose`
- NO new go.mod dependencies
- Follow patterns: New() constructors, *slog.Logger, nil checks
- Wrap errors with fmt.Errorf and context
- Tests via httptest.NewServer for health checks
- ARM64 compatible (RPi 5)
- Branch: feat/docker-compose-stack

## Context
- Existing code: internal/stack/ (stack.go, generator.go, wizard.go, configgen.go, secrets.go)
- CLI: cmd/mediamate/stack_cmd.go — newStackCmd() + newStackInitCmd()
- Docker Compose template: internal/stack/templates/docker-compose.yml.tmpl
- Cobra pattern: newXxxCmd() functions, cmd.AddCommand()
- Styles: helpers.go — styleError, styleSuccess, styleInfo, styleDim
- Service ports: Radarr 7878, Sonarr 8989, Readarr 8787, Prowlarr 9696, qBittorrent 8080, Transmission 9091, Deluge 8112, Jellyfin 8096, Plex 32400

## Feedback Log

### Step 1: Docker compose runner (2026-02-24)
- Result: OK
- Created compose.go with Compose struct, Up(), Down(), PS() methods
- Uses os/exec, checks for docker compose availability via LookPath

### Step 2: Health checker (2026-02-24)
- Result: OK
- Created health.go with service port map, HTTP probe with configurable timeout
- CheckAll() returns []ServiceHealth with name, status, endpoint info

### Step 3: Health checker tests (2026-02-24)
- Result: OK
- Tests for healthy/unhealthy/unreachable services via httptest

### Step 4: Healthcheck directives (2026-02-24)
- Result: OK
- Added healthcheck blocks to all services in docker-compose.yml.tmpl
- Updated depends_on for mediamate with condition: service_healthy

### Step 5: CLI commands (2026-02-24)
- Result: OK
- Added stack up, stack down, stack status commands with --file flag
- Status shows table with container name, state, health, and HTTP probe result

### Step 6: Build verification (2026-02-24)
- Result: OK
- go build ./... — clean
- go test ./... — all pass (70 tests including 4 new health tests)
- go vet ./... — clean

---
*Created: 2026-02-24*
*Status: complete*
