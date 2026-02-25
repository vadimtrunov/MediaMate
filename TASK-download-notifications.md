# Feature: Download Notifications (Phase 5.1)

## Task ID
download-notifications

## Описание
Webhook-сервер для получения событий от Radarr + отправка Telegram уведомлений с Jellyfin ссылкой.

## Архитектурные решения
- HTTP-сервер встроен в `mediamate bot` — один процесс (бот + webhooks)
- Только `Download` event (фильм скачан и импортирован)
- Получатели: все `allowed_user_ids` из Telegram конфига
- Setup wizard автоматически регистрирует webhook в Radarr
- `net/http` only — без внешних фреймворков

## Acceptance Criteria
- [x] Webhook endpoint принимает POST от Radarr и парсит Download event
- [x] Telegram уведомление отправляется всем allowed_user_ids
- [x] Уведомление содержит название фильма, год и Jellyfin ссылку
- [x] Если Jellyfin недоступен — уведомление без ссылки (graceful degradation)
- [x] Webhook secret для безопасности
- [x] Setup wizard регистрирует webhook в Radarr (idempotent)
- [x] `go build ./...` и `golangci-lint run` проходят
- [x] Тесты покрывают handler, service, server, radarr webhook methods

## План
- [x] Шаг 1: Config — добавить WebhookConfig в config.go (port, secret, env vars, validation)
- [x] Шаг 2: Webhook payload types — internal/notification/types.go (RadarrWebhookPayload, RadarrMovie)
- [x] Шаг 3: Notification service — internal/notification/service.go (NotifyDownloadComplete)
- [x] Шаг 4: Webhook HTTP handler — internal/notification/webhook.go (POST /webhooks/radarr)
- [x] Шаг 5: HTTP server lifecycle — internal/notification/server.go (Start/Stop, routing)
- [x] Шаг 6: Integration in bot command — cmd/mediamate/bot.go (parallel startup, graceful shutdown)
- [x] Шаг 7: Radarr webhook methods + setup wizard (AddNotification, ListNotifications, setup step)
- [x] Шаг 8: Tests (handler, service, server, radarr methods)
- [x] Шаг 9: Update ROADMAP.md — mark Phase 5.1 ✅

## Constraints
- Revive: function-length max 60 lines, line-length-limit 140
- gosec G306: WriteFile permissions 0600 or less
- gci sections: standard, default, prefix(github.com/vadimtrunov/MediaMate)
- `net/http` only — НЕ тянуть фреймворк
- Sonarr events НЕ делаем (Phase 6)

## Context
- Radarr webhook API: POST `/api/v3/notification` для создания
- Radarr webhook payload: `eventType`, `movie` (title, year, tmdbId), `movieFile`
- Telegram: `Frontend.SendMessage(ctx, userID, message)` — userID как string
- Jellyfin: `MediaServer.GetLink(ctx, itemName)` — ищет по имени фильма
- Config: YAML + env vars override, pointer для optional секций
- Logger: `*slog.Logger`, JSON output

## Feedback Log
<!-- Заполняется во время имплементации -->

---
*Создано: 2026-02-24*
*Статус: done*
