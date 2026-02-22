# First-Time Setup Guide

## GitHub Repository Setup Checklist

После пуша этих файлов в GitHub:

### ✅ Автоматически заработают:

1. **GitHub Actions** (CI/CD)
   - ✅ Build & Test на каждый PR
   - ✅ Docker multi-arch builds
   - ✅ CodeQL security scanning
   - ✅ Release automation

2. **Dependabot**
   - ✅ Автоматические обновления зависимостей
   - ✅ Weekly проверка Go modules, Docker, GitHub Actions

3. **Issue Templates**
   - ✅ Структурированные формы для bug reports
   - ✅ Feature request templates

### 🔧 Нужно настроить вручную:

#### 1. CodeRabbit (AI Code Review) - 2 минуты

См. подробную инструкцию: [CODERABBIT_SETUP.md](CODERABBIT_SETUP.md)

**Кратко:**
1. Открой https://coderabbit.ai
2. Sign in with GitHub
3. Install CodeRabbit на репозиторий `vadimtrunov/MediaMate`
4. Готово! ✅

#### 2. Codecov (Test Coverage) - опционально

Если хочешь отслеживать test coverage:

1. Открой https://codecov.io
2. Sign in with GitHub
3. Add repository: `vadimtrunov/MediaMate`
4. Скопируй `CODECOV_TOKEN`
5. Добавь в GitHub Secrets:
   ```bash
   gh secret set CODECOV_TOKEN
   # Вставь токен из Codecov
   ```

#### 3. Branch Protection Rules - рекомендуется

Защита main ветки:

```bash
# Через GitHub CLI
gh api repos/vadimtrunov/MediaMate/branches/main/protection -X PUT -f required_status_checks='{"strict":true,"contexts":["test","lint","build"]}' -f enforce_admins=false -f required_pull_request_reviews='{"required_approving_review_count":0}' -f restrictions=null
```

Или через UI:
1. Settings → Branches → Add rule
2. Branch name pattern: `main`
3. ✅ Require a pull request before merging
4. ✅ Require status checks to pass before merging
   - Select: `test`, `lint`, `build`
5. Save changes

---

## Первый коммит

```bash
# Проверь что все файлы добавлены
git status

# Добавь все новые файлы
git add .

# Коммит
git commit -m "chore: setup GitHub workflows and automation

- Add CI/CD workflows (build, test, lint, docker)
- Add security scanning (CodeQL, Trivy, Gosec)
- Add release automation (GoReleaser, Release Drafter)
- Configure Dependabot for automated updates
- Add CodeRabbit configuration for AI code review
- Add issue and PR templates
- Add golangci-lint and goreleaser configs"

# Пуш
git push origin main
```

---

## Проверка работоспособности

После пуша проверь GitHub Actions:

1. Открой https://github.com/vadimtrunov/MediaMate/actions
2. Должны запуститься workflows:
   - ✅ **CI** (build, test, lint)
   - ✅ **Security Scan** (CodeQL, Trivy)
   - ✅ **Release Drafter** (создаст draft release)

Если что-то упадёт — это нормально на начальном этапе (пока нет Go кода).

---

## Что дальше?

### Phase 0: Project Structure

Следующий шаг из [ROADMAP.md](ROADMAP.md):

1. Создать Go module структуру
2. Определить core интерфейсы
3. Настроить конфигурацию
4. Написать первые тесты

### Создай первый PR

Проверь что CodeRabbit работает:

```bash
git checkout -b feat/project-structure
# ... создай Go файлы ...
git add .
git commit -m "feat: add initial project structure"
git push origin feat/project-structure
gh pr create --title "feat: Add initial project structure" --body "Phase 0 from roadmap"
```

CodeRabbit автоматически прокомментирует PR! 🎉

---

## Полезные команды

```bash
# Локальный build
make build

# Запустить тесты
make test

# Линтинг
make lint

# Проверить что workflows валидны
gh workflow list

# Посмотреть статус последнего workflow
gh run list --limit 5

# Посмотреть логи workflow
gh run view
```

---

## Troubleshooting

### GitHub Actions не запускаются?

Проверь что workflows enabled:
1. Settings → Actions → General
2. ✅ Allow all actions and reusable workflows

### CodeQL падает?

Это нормально пока нет Go кода. После создания `cmd/mediamate/main.go` заработает.

### Docker build падает?

Нужно создать `Dockerfile` (будет в Phase 0).

---

## Summary

**Что уже работает (бесплатно):**
- ✅ Автоматический build и тесты на каждый PR
- ✅ Security scanning (CodeQL, Trivy, Gosec)
- ✅ Dependabot обновления
- ✅ Release automation
- ✅ Issue/PR templates

**Что нужно добавить:**
- 🔧 CodeRabbit (2 минуты через UI)
- 🔧 Branch protection (опционально, но рекомендуется)
- 📊 Codecov (опционально для test coverage)

**Стоимость:** $0 — всё бесплатно для open source! 🎉
