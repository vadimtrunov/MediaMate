# Feature: Download Progress in Telegram

## Task ID
download-progress

## Описание
Отображение прогресса активных загрузок в Telegram с periodic polling и live-update через EditMessage. Одно сводное сообщение на юзера, обновляемое через EditMessage.

## Acceptance Criteria
- [x] При Radarr "Grab" event — юзер получает сообщение о начале загрузки
- [x] Сообщение обновляется каждые ~15 сек с progress bar, скоростью, ETA
- [x] Одно сводное сообщение на юзера для всех активных загрузок
- [x] При завершении загрузки — сообщение обновляется на "Завершено"
- [x] При перезапуске бота — подхватывает уже идущие загрузки через polling
- [x] Graceful shutdown — корректно останавливает ticker
- [x] Throttle — обновляет по 2% threshold + configurable interval
- [x] Тесты для tracker service

## План
- [x] Шаг 1: Добавить ProgressNotifier интерфейс в core + progress config
- [x] Шаг 2: Реализовать SendProgressMessage / EditProgressMessage в Telegram bot
- [x] Шаг 3: Добавить Grab event в webhook handler + progress tracker types
- [x] Шаг 4: Реализовать ProgressTracker service (polling loop + update logic)
- [x] Шаг 5: Интегрировать tracker в cmd/mediamate/bot.go (wiring)
- [x] Шаг 6: Обновить example config + progress message formatting
- [x] Шаг 7: Тесты для progress tracker
- [x] Шаг 8: Lint + build проверка

## Constraints
- revive function-length max 60 lines, line-length-limit 140
- gosec G306: WriteFile permissions 0600 or less
- gci sections: standard, default, prefix(github.com/vadimtrunov/MediaMate)
- Отдельный интерфейс ProgressNotifier, НЕ менять core.Frontend
- Telegram rate limits: ~30 req/sec per bot

## Context
- `internal/core/interfaces.go` — ProgressNotifier interface
- `internal/notification/progress.go` — Tracker service
- `internal/notification/service.go` — ProgressTracker interface, NotifyGrab
- `internal/notification/webhook.go` — Grab event handling
- `internal/frontend/telegram/bot.go` — SendProgressMessage, EditProgressMessage
- `cmd/mediamate/bot.go` — wiring
- `internal/config/config.go` — ProgressConfig

## Feedback Log

### Review (2026-02-25)
- Исправлен дублированный case в formatETA (seconds <= 0 и seconds < 60)
- Убрана зависимость notification -> telegram; ProgressBar вынесен как локальная функция
- Русский текст в уведомлениях оставлен (consistent с existing service.go)

---
*Создано: 2026-02-25*
*Статус: completed*
