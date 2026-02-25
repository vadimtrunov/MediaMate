# Feature: Docker Compose Stack Init (Phase 4.2 Part 1)

## Task ID
docker-compose-stack-init

## Описание
Реализовать команду `mediamate stack init` — интерактивный Bubble Tea wizard для генерации
docker-compose.yml, .env и mediamate.yaml. MediaMate включается в compose как сервис.

## Acceptance Criteria
- [x] `mediamate stack init` запускает Bubble Tea TUI wizard
- [x] Wizard позволяет выбрать/отключить компоненты с дефолтами
- [x] Wizard спрашивает пути для media (movies, tv, books) и downloads
- [x] Генерируется валидный `docker-compose.yml` в текущей директории (или --output)
- [x] Генерируется `.env` с плейсхолдерами для API ключей и сгенерированными паролями
- [x] Генерируется `mediamate.yaml` конфиг, совместимый с существующей config системой
- [x] Если файлы уже существуют — generator возвращает ошибку (--overwrite для перезаписи)
- [x] `--non-interactive` флаг для CI/скриптов (использует дефолты)
- [x] Тесты для генерации шаблонов

## План
- [x] Шаг 1: Types & config — `internal/stack/stack.go` — StackConfig, ComponentSelection, дефолты
- [x] Шаг 2: Password generation — `internal/stack/secrets.go` — crypto/rand пароли
- [x] Шаг 3: Docker Compose template — `internal/stack/templates/docker-compose.yml.tmpl` + embed
- [x] Шаг 4: Env template — `internal/stack/templates/env.tmpl` + генерация .env
- [x] Шаг 5: MediaMate config generation — `internal/stack/configgen.go` — mediamate.yaml
- [x] Шаг 6: Generator orchestrator — `internal/stack/generator.go` — Generate() + file overwrite check
- [x] Шаг 7: Bubble Tea wizard — `internal/stack/wizard.go` — multi-step TUI
- [x] Шаг 8: Cobra command — `cmd/mediamate/stack_cmd.go` — stack init + flags + register in root
- [x] Шаг 9: Tests — `internal/stack/generator_test.go`
- [x] Шаг 10: Build verification — go build, go vet, tests pass

## Constraints
- Follow existing patterns: New() constructors, *slog.Logger, nil checks
- NO docker SDK or compose libraries — plain YAML template generation
- NO stack up/down/health checks — Part 2
- Generate files ONLY, don't run Docker
- Passwords via crypto/rand
- ARM64 compatible Docker images for all services
- Go templates + //go:embed for compose/env generation

## Context
- CLI pattern: cmd/mediamate/ — Cobra with newXxxCmd() + helpers.go
- Config: internal/config/config.go — Config struct
- Bubble Tea used in cmd/mediamate/chat.go
- Services in compose must be on a single Docker network
- Components: Radarr, Sonarr, Readarr, Prowlarr, qBittorrent/Transmission/Deluge, Jellyfin/Plex, Gluetun, MediaMate

## Feedback Log

### Шаг 1-6: Core types and generation (2026-02-24)
- Результат: OK
- Заметки: All types, templates, and generation logic implemented in one batch since they're tightly coupled

### Шаг 7: Bubble Tea wizard (2026-02-24)
- Результат: OK
- Заметки: 3-step wizard: component selection (radio/checkbox), path config (textinput), confirmation

### Шаг 8: Cobra command (2026-02-24)
- Результат: OK
- Заметки: stack init with --output, --overwrite, --non-interactive flags

### Шаг 9-10: Tests and verification (2026-02-24)
- Результат: OK
- Заметки: 11 tests / 32 subtests, all pass. Full project builds clean.

---
*Создано: 2026-02-24*
*Статус: complete*
