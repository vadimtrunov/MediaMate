# Feature: Setup Wizard — Auto-configuration of Services

## Task ID
setup-wizard

## Description
Extend `mediamate stack init` to automatically configure running services after stack generation:
- Read API keys from service XML configs
- Configure Radarr (root folders, quality profiles, download client)
- Configure Prowlarr (link to Radarr/Sonarr, add download client, FlareSolverr)
- Configure qBittorrent (download path)
- Run connection tests to verify everything works

## Acceptance Criteria
- [ ] `mediamate stack init` generates files AND can auto-configure running services
- [ ] API keys extracted from config.xml and written to .env/mediamate.yaml
- [ ] Radarr gets root folder + download client (qBittorrent)
- [ ] Prowlarr linked to Radarr/Sonarr + qBittorrent
- [ ] FlareSolverr added to docker-compose.yml when Prowlarr selected
- [ ] Connection tests verify all service links work
- [ ] All new methods covered by unit tests
- [ ] Graceful degradation: manual setup instructions on failure

## Plan
- [x] Step 1: Radarr client — add public setup methods
- [x] Step 2: Radarr setup methods — tests
- [x] Step 3: Prowlarr client — create package
- [x] Step 4: Prowlarr client — setup methods
- [x] Step 5: Prowlarr client — tests
- [x] Step 6: qBittorrent — add preferences methods + test
- [x] Step 7: FlareSolverr — add to stack infrastructure
- [x] Step 8: API Key Bootstrap — XML config parser
- [x] Step 9: API Key Bootstrap — tests
- [x] Step 10: Setup orchestrator
- [x] Step 11: Integrate into stack init flow
- [x] Step 12: Connection tests

## Constraints
- Follow existing HTTP client patterns (baseURL + apiKey + httpclient.Client)
- Do NOT add Prowlarr indexers automatically (need user credentials)
- Read API keys from XML files on disk (not from API)
- Don't break existing stack init flow — setup is optional post-generation step
- FlareSolverr only when Prowlarr is selected

## Context
- Radarr client pattern: `internal/backend/radarr/client.go`
- qBittorrent pattern: `internal/torrent/qbittorrent/client.go`
- Stack wizard: `internal/stack/wizard.go`
- Stack generator: `internal/stack/generator.go`
- Compose template: `internal/stack/templates/docker-compose.yml.tmpl`
- Config: `internal/config/config.go`

## Feedback Log

### Step 10: Setup orchestrator (2026-02-24)
- Result: OK
- Created internal/stack/setup.go with SetupRunner
- 6-step flow: health wait → read keys → update configs → Radarr → Prowlarr → qBittorrent
- All actions idempotent (list-then-check before create)
- Graceful degradation: errors logged, don't halt other services

### Step 11: Integrate into stack init flow (2026-02-24)
- Result: OK
- Added `mediamate stack setup` command to cmd/mediamate/stack_cmd.go
- Updated "Next steps" hint in stack init output
- printSetupResults displays formatted table of outcomes

### Step 12: Connection tests (2026-02-24)
- Result: OK
- 13 test functions, 33 total tests in stack package
- Full integration test with mock Radarr/Prowlarr/qBittorrent servers
- Idempotency tests verify skip behavior
- Error path tests for API failures

---
*Created: 2026-02-24*
*Status: complete*
