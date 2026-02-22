# Git Workflow Guide

## Стандартный Git Flow

Используем упрощённую версию Git Flow с двумя основными ветками.

---

## Основные ветки

### `main` — Production
- Всегда стабильная и готова к релизу
- Прямые коммиты **запрещены**
- Только через PR из `develop`
- Каждый merge в main = релиз

### `develop` — Development
- Основная ветка разработки
- Сюда мержатся feature ветки
- Всегда рабочая, но может содержать новые фичи
- От неё создаём feature branches

---

## Feature Branches

### Создание feature ветки

```bash
# Убедись что на develop
git checkout develop
git pull origin develop

# Создай feature ветку
git checkout -b feat/llm-integration
# или
git checkout -b fix/config-validation
```

### Naming Convention

**Префиксы:**
- `feat/` — новая функциональность
- `fix/` — баг фикс
- `refactor/` — рефакторинг
- `docs/` — документация
- `test/` — тесты
- `chore/` — инфраструктурные изменения

**Примеры:**
```
feat/claude-client
feat/radarr-integration
fix/config-loading
refactor/llm-interface
docs/api-reference
test/integration-tests
chore/update-deps
```

### Работа над фичей

```bash
# Делай изменения
vim internal/llm/claude/client.go

# Коммиты
git add .
git commit -m "feat(llm): add Claude API client

- Implement basic Claude client
- Add retry logic with exponential backoff
- Add context timeout support"

# Пуш в remote
git push origin feat/claude-client
```

### Создание Pull Request

```bash
# Создай PR в develop (не в main!)
gh pr create --base develop --title "feat: Add Claude LLM integration" --body "
## What

Adds Claude API client for LLM integration.

## Changes

- Claude API client with retry logic
- Configuration for API keys
- Unit tests with mocking

## Testing

- [x] Unit tests pass
- [x] Integration test with real API
- [x] Linter passes

## Related

Part of Phase 1 from roadmap.
"
```

### После PR Review

```bash
# Если нужны изменения - просто коммить в ту же ветку
git add .
git commit -m "fix: address review comments"
git push origin feat/claude-client

# PR автоматически обновится
```

### После Merge

```bash
# Переключись на develop
git checkout develop
git pull origin develop

# Удали локальную ветку
git branch -d feat/claude-client

# Удали remote ветку (автоматически через GitHub обычно)
git push origin --delete feat/claude-client
```

---

## Релизы (main branch)

### Когда делать релиз

Когда `develop` достиг milestone:
- v0.1 — MVP готов
- v0.2 — Новые фичи стабильны
- v1.0 — Production ready

### Процесс релиза

```bash
# Убедись что develop стабильна
git checkout develop
git pull origin develop

# Обнови версию в коде (если есть)
# vim internal/version/version.go

# Создай PR из develop в main
gh pr create --base main --head develop --title "Release v0.1.0" --body "
## Release v0.1.0

### Features
- Claude LLM integration
- Radarr backend
- Telegram frontend
- CLI interface

### Tested
- [x] All tests pass
- [x] Integration tests
- [x] Manual testing on RPi5
"

# После merge в main - создай тэг
git checkout main
git pull origin main
git tag -a v0.1.0 -m "Release v0.1.0

- Claude LLM integration
- Radarr backend support
- Telegram + CLI frontends
- Docker multi-arch build"

# Пуш тэга
git push origin v0.1.0

# GitHub Actions автоматически:
# - Запустит GoReleaser
# - Создаст GitHub Release
# - Соберёт бинарники
# - Опубликует Docker образы
```

---

## Hotfix

Если критический баг в production (main):

```bash
# Создай hotfix ветку от main
git checkout main
git pull origin main
git checkout -b hotfix/critical-security-issue

# Фикс
vim internal/security/fix.go
git add .
git commit -m "fix(security): patch critical vulnerability

CVE-2024-XXXXX - SQL injection in search endpoint"

# Push
git push origin hotfix/critical-security-issue

# PR в main (не в develop!)
gh pr create --base main --title "hotfix: Critical security patch" --body "..."

# После merge в main:
# 1. Создай тэг (v0.1.1)
# 2. Merge main обратно в develop
git checkout develop
git merge main
git push origin develop
```

---

## Синхронизация с upstream

Если отстал develop от main:

```bash
git checkout develop
git pull origin develop
git merge main
# Разреши конфликты если есть
git push origin develop
```

---

## Правила

### ✅ DO:
- Всегда создавай feature ветку от `develop`
- Пиши понятные commit messages
- PR только через GitHub (не direct push)
- Squash коммиты если их слишком много
- Удаляй merged ветки
- Регулярно синкай с develop

### ❌ DON'T:
- Прямые коммиты в `main` или `develop`
- Огромные PR (>500 строк лучше разбить)
- PR без тестов
- Merge своих PR без review (если проект не личный)
- Force push в `main` или `develop`

---

## Useful Commands

```bash
# Переключиться на develop
git checkout develop

# Обновить develop
git pull origin develop

# Создать feature ветку
git checkout -b feat/my-feature

# Статус
git status

# Посмотреть изменения
git diff

# История
git log --oneline --graph --all

# Отменить незакоммиченные изменения
git checkout -- file.go

# Мягкий reset последнего коммита
git reset --soft HEAD~1

# Sync с remote
git fetch origin
git status

# Посмотреть все ветки
git branch -a

# Удалить локальную ветку
git branch -d feat/old-feature

# Удалить remote ветку
git push origin --delete feat/old-feature
```

---

## GitHub CLI Shortcuts

```bash
# Создать PR
gh pr create

# Посмотреть статус PR
gh pr status

# Список PR
gh pr list

# Checkout PR локально
gh pr checkout 123

# Merge PR
gh pr merge 123

# Посмотреть Actions
gh run list

# Посмотреть логи workflow
gh run view
```

---

## Commit Message Format

Используем Conventional Commits:

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat` — новая функциональность
- `fix` — баг фикс
- `refactor` — рефакторинг
- `docs` — документация
- `test` — тесты
- `chore` — инфраструктура
- `perf` — производительность
- `style` — форматирование

**Scope (опционально):**
- `llm`, `radarr`, `telegram`, `config`, etc.

**Примеры:**

```bash
feat(llm): add Claude API integration

Implements basic Claude client with retry logic and
timeout handling.

Closes #42
```

```bash
fix(config): validate API keys on startup

Previously invalid API keys would cause runtime panic.
Now we validate during config loading.

Fixes #15
```

```bash
refactor(llm): extract retry logic to separate package

- Move retry logic to pkg/retry
- Make it reusable across different clients
- Add exponential backoff configuration
```

---

## Current Workflow Summary

```
main (production)
  ↑
  └── PR when ready for release
       │
develop (integration)
  ↑
  └── feat/claude-client
  └── feat/radarr-backend
  └── fix/config-bug
```

**Вся разработка идёт в feature ветках → PR в develop → релиз через PR в main**

---

## Next Steps

1. Настрой branch protection для `main`:
   ```bash
   # Требуем PR review
   # Требуем прохождения CI
   # Запрещаем force push
   ```

2. Опционально: настрой auto-merge для dependabot PR

3. Начинай работу над Phase 0 из roadmap!

```bash
git checkout develop
git checkout -b feat/project-structure
# ... код ...
git commit -m "feat: add initial project structure"
gh pr create --base develop
```
