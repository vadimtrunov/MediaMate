# MediaMate Development Roadmap

## Философия реализации

Строим поэтапно, каждая фаза — полностью рабочий продукт. Начинаем с минимума, постепенно добавляем функциональность.

---

## Phase 0: Foundation (Week 1-2)

### 0.1 Project Structure
- [ ] Настроить Go module структуру
- [ ] Определить пакеты: `cmd/`, `internal/`, `pkg/`
- [ ] Настроить CI/CD (GitHub Actions)
- [ ] Makefile для сборки
- [ ] Docker multi-stage build для ARM64 и AMD64

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
- [ ] YAML конфигурация (`internal/config/`)
- [ ] Environment variables override
- [ ] Валидация конфига
- [ ] Примеры конфигов: `configs/mediamate.example.yaml`

### 0.4 Logging & Observability
- [ ] Structured logging (zerolog или slog)
- [ ] Уровни логирования
- [ ] Context-aware logging

**Deliverable:** Пустой скелет проекта с интерфейсами и конфигом

---

## Phase 1: MVP Backend (Week 3-5)

### 1.1 LLM Integration (Claude)
- [ ] `internal/llm/claude/` — Claude API клиент
- [ ] Tool calling поддержка (Claude function calling)
- [ ] Retry logic и error handling
- [ ] Rate limiting
- [ ] Тесты с mock LLM

**Tools для Claude:**
```go
// Определяем инструменты, которые LLM может вызывать
- SearchMovie(query string) []Movie
- DownloadMovie(movieID int) Status
- GetDownloadStatus() []Download
- RecommendSimilar(movieID int) []Movie
```

### 1.2 TMDb Integration
- [ ] `internal/metadata/tmdb/` — TMDb API клиент
- [ ] Поиск фильмов
- [ ] Рекомендации
- [ ] Детали фильма (рейтинг, описание, постеры)
- [ ] Кеширование популярных запросов

### 1.3 Radarr Integration
- [ ] `internal/backend/radarr/` — Radarr API клиент
- [ ] Поиск релизов
- [ ] Добавление фильмов в библиотеку
- [ ] Мониторинг статуса
- [ ] Quality profiles

### 1.4 qBittorrent Integration
- [ ] `internal/torrent/qbittorrent/` — qBittorrent Web API клиент
- [ ] Получение списка торрентов
- [ ] Прогресс загрузки
- [ ] Управление (пауза/возобновление)

### 1.5 Core Orchestration
- [ ] `internal/agent/` — AI агент-оркестратор
- [ ] Парсинг интента из LLM
- [ ] Маппинг tool calls в реальные вызовы API
- [ ] Conversation state management
- [ ] Multi-step flows (search → confirm → download)

**Deliverable:** Backend, который понимает запросы и управляет медиа через Radarr + qBittorrent

---

## Phase 2: CLI Frontend (Week 6)

### 2.1 Interactive CLI
- [ ] `cmd/mediamate/chat.go` — интерактивный чат в терминале
- [ ] История сообщений (arrows up/down)
- [ ] Красивый вывод (bubbles/lipgloss)
- [ ] Streaming responses от LLM
- [ ] Цветной вывод (фильмы, статусы)

### 2.2 CLI Commands
```bash
mediamate chat              # Интерактивный чат
mediamate query "скачай Дюну"  # Разовый запрос
mediamate status            # Статус загрузок
mediamate config validate   # Проверка конфига
```

**Deliverable:** Можно полноценно общаться с ботом через терминал

---

## Phase 3: Telegram Frontend (Week 7-8)

### 3.1 Telegram Bot
- [ ] `internal/frontend/telegram/` — Telegram Bot API
- [ ] Обработка текстовых сообщений
- [ ] Inline keyboard для подтверждений
- [ ] Streaming ответов (typing indicator)
- [ ] Multi-user support (user isolation)
- [ ] Admin whitelist (только разрешенные user IDs)

### 3.2 Rich Media
- [ ] Отправка постеров фильмов
- [ ] Форматирование рекомендаций (Markdown)
- [ ] Progress bars для загрузок
- [ ] Callback buttons (скачать 1/2/3)

**Deliverable:** v0.1 — Полноценный MVP с Telegram + CLI

---

## Phase 4: Jellyfin & Stack Management (Week 9-10)

### 4.1 Jellyfin Integration
- [ ] `internal/mediaserver/jellyfin/` — Jellyfin API
- [ ] Проверка доступности фильма
- [ ] Генерация ссылок на просмотр
- [ ] Webhook для уведомлений о новом контенте

### 4.2 Docker Compose Stack
- [ ] `mediamate stack init` — интерактивный wizard
  - Выбор компонентов (Radarr, Sonarr, Jellyfin, etc)
  - Генерация `docker-compose.yml`
  - Генерация `.env` с секретами
- [ ] `mediamate stack up` — запуск стека
- [ ] `mediamate stack down` — остановка
- [ ] Health checks всех сервисов

### 4.3 Setup Wizard
- [ ] Автоматическая настройка Radarr (quality profiles, root folders)
- [ ] Автоматическая настройка Prowlarr + indexers
- [ ] Связывание Radarr ↔ Prowlarr ↔ qBittorrent
- [ ] Тесты подключений

**Deliverable:** `mediamate stack init` → полностью рабочий стек за 5 минут

---

## Phase 5: Sonarr & Readarr (Week 11-12)

### 5.1 Sonarr Support
- [ ] `internal/backend/sonarr/` — Sonarr API клиент
- [ ] Поиск сериалов
- [ ] Мониторинг сезонов/эпизодов
- [ ] Обновление LLM tools для TV shows

### 5.2 Readarr Support
- [ ] `internal/backend/readarr/` — Readarr API клиент
- [ ] Поиск книг
- [ ] E-book форматы (epub, mobi, pdf)

### 5.3 Unified Search
- [ ] LLM определяет тип контента (movie/show/book)
- [ ] Единый интерфейс поиска
- [ ] Приоритизация результатов

**Deliverable:** v0.2 — Поддержка фильмов, сериалов, книг

---

## Phase 6: Alternative Providers (Week 13-14)

### 6.1 OpenAI Support
- [ ] `internal/llm/openai/` — OpenAI клиент
- [ ] GPT-4 Turbo function calling
- [ ] Переключение в конфиге

### 6.2 Ollama Support
- [ ] `internal/llm/ollama/` — Ollama клиент
- [ ] Локальные модели (Llama 3, Mistral)
- [ ] Автоопределение доступных моделей

### 6.3 Alternative Torrent Clients
- [ ] `internal/torrent/transmission/`
- [ ] `internal/torrent/deluge/`
- [ ] Единый интерфейс `TorrentClient`

**Deliverable:** v0.3 — Гибкость выбора компонентов

---

## Phase 7: Advanced Features (Week 15-16)

### 7.1 Conversation History
- [ ] SQLite для хранения истории
- [ ] Контекст предыдущих запросов
- [ ] "Помнишь, я просил скачать тот фильм?" → LLM ищет в истории

### 7.2 Claude OAuth
- [ ] Авторизация через Claude.ai subscription
- [ ] Снижение затрат для пользователей с подпиской

### 7.3 Notifications
- [ ] Webhook от Radarr/Sonarr при завершении загрузки
- [ ] Уведомление в Telegram: "Interstellar готов к просмотру!"
- [ ] Прямая ссылка на Jellyfin

### 7.4 Smart Recommendations
- [ ] Персональные рекомендации на основе истории
- [ ] "Что посмотреть сегодня вечером?" → анализ лайков/дизлайков
- [ ] Интеграция с Jellyfin viewing history

**Deliverable:** v0.4 — Умный ассистент с памятью

---

## Phase 8: Polish & Release (Week 17-20)

### 8.1 Testing
- [ ] Unit тесты (>80% coverage)
- [ ] Integration тесты с mock API
- [ ] End-to-end тесты в Docker
- [ ] Тесты на ARM64 (Raspberry Pi 5)

### 8.2 Documentation
- [ ] Docs site (Hugo/MkDocs)
- [ ] Пошаговые гайды
- [ ] API reference
- [ ] Troubleshooting
- [ ] Video demos

### 8.3 Install Script
- [ ] One-liner install: `curl ... | bash`
- [ ] Автоопределение ARM64/AMD64
- [ ] Systemd service
- [ ] Auto-updates

### 8.4 Performance
- [ ] Benchmarks LLM response time
- [ ] Оптимизация памяти (важно для RPi)
- [ ] Параллельные запросы к API
- [ ] Graceful shutdown

**Deliverable:** v1.0 — Production-ready release

---

## Future Ideas (Post v1.0)

### Discord & Matrix Frontends
- Аналогично Telegram, но для других платформ

### Web UI
- React/Vue dashboard
- Visual library browser
- Download manager
- Settings UI

### Advanced AI Features
- Голосовой ввод (Whisper)
- Генерация плейлистов ("создай подборку фильмов на выходные")
- Автоматический контент ("каждую пятницу скачивай новый эпизод The Mandalorian")

### Multi-instance Support
- Несколько Radarr (4K + 1080p)
- Профили качества на основе устройства воспроизведения

### Subtitle Management
- Bazarr интеграция
- Автоскачивание субтитров на нужном языке

---

## Success Metrics

### v0.1 (MVP)
- [ ] Можно попросить скачать фильм через Telegram
- [ ] Фильм появляется в Jellyfin через ~10-30 минут
- [ ] Работает на Raspberry Pi 5

### v1.0 (Production)
- [ ] <5 секунд на ответ LLM
- [ ] <100MB RAM на ARM64
- [ ] Install в 1 команду
- [ ] Документация на уровне Homelab-проектов

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

1. **Interface-first** — Каждый компонент за интерфейсом, легко swap
2. **Test-friendly** — Все API клиенты mockable
3. **ARM64 first** — Raspberry Pi 5 как primary target
4. **Config-driven** — Никаких hardcoded значений
5. **Fail-safe** — Graceful degradation если какой-то сервис недоступен
6. **User-centric** — Простота setup важнее гибкости

---

## Timeline

- **Weeks 1-8:** MVP (v0.1) — Claude + Radarr + Telegram
- **Weeks 9-12:** Core features (v0.2) — Sonarr, Readarr, Jellyfin
- **Weeks 13-16:** Alternative providers (v0.3-v0.4)
- **Weeks 17-20:** Polish & release (v1.0)

**Total:** ~5 месяцев до v1.0

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
