# CodeRabbit Setup Instructions

## Что такое CodeRabbit?

CodeRabbit — это AI-powered code review инструмент, который автоматически ревьюит каждый Pull Request в твоей репе.

**Бесплатно для open source проектов!** ✅

---

## Установка (2 минуты)

### Шаг 1: Установить GitHub App

1. Открой https://coderabbit.ai
2. Нажми **"Sign in with GitHub"**
3. Выбери **"Install CodeRabbit"**
4. Выбери репозиторий: **vadimtrunov/MediaMate**
5. Нажми **"Install & Authorize"**

### Шаг 2: Всё! 🎉

CodeRabbit автоматически обнаружит конфиг `.github/.coderabbit.yaml` и начнёт работать.

---

## Как работает?

### Автоматический ревью PR

Когда ты создаёшь Pull Request:

1. CodeRabbit автоматически анализирует код
2. Комментирует потенциальные проблемы прямо в PR
3. Даёт рекомендации по улучшению
4. Генерирует краткое резюме изменений

### Команды в PR

Можешь использовать команды в комментариях PR:

```
@coderabbitai help                  # Показать все доступные команды
@coderabbitai review                # Повторить ревью
@coderabbitai explain               # Объяснить изменения
@coderabbitai fix                   # Предложить фикс
@coderabbitai generate tests        # Сгенерировать тесты
```

### Пример работы

```markdown
## CodeRabbit Summary

### Changes
- Added LLM interface in `internal/llm/interface.go`
- Implemented Claude client with retry logic
- Added configuration for API keys

### Potential Issues
⚠️ `internal/llm/claude/client.go:42`
Consider adding context timeout to API calls

### Suggestions
💡 `internal/config/config.go:15`
Use environment variables for sensitive data

### Security
🔒 No security issues detected
```

---

## Настройка (уже сделано)

Конфиг находится в `.github/.coderabbit.yaml`:

```yaml
language: "en"
enable_free_tier: true

reviews:
  profile: "chill"              # Мягкий режим (не слишком придирчивый)
  auto_review:
    enabled: true               # Автоматический ревью при каждом PR
    drafts: false               # Не ревьюить draft PR

  path_filters:
    - "!**/*.md"                # Не ревьюить Markdown файлы
    - "!**/*.json"              # Не ревьюить JSON
    - "!**/*.yaml"              # Не ревьюить YAML

  path_instructions:
    - path: "internal/**/*.go"
      instructions: |
        - Focus on Go best practices
        - Check error handling
        - Look for race conditions
        - Verify context usage
```

### Можно настроить:

- **`profile`** — уровень придирчивости:
  - `"assertive"` — строгий (много комментариев)
  - `"chill"` — мягкий (только важное) ← **текущий**
  - `"default"` — баланс

- **`path_filters`** — какие файлы игнорировать

- **`path_instructions`** — специфичные инструкции для разных частей кода

---

## Проверить что работает

### Создай тестовый PR:

```bash
# Создай новую ветку
git checkout -b test/coderabbit-test

# Добавь тестовый файл
cat > test.go <<EOF
package main

func add(a, b int) int {
    return a + b  // Simple function
}
EOF

git add test.go
git commit -m "test: add simple function"
git push origin test/coderabbit-test

# Создай PR
gh pr create --title "Test: CodeRabbit integration" --body "Testing AI code review"
```

Через 10-30 секунд CodeRabbit должен:
- Прокомментировать PR
- Дать резюме изменений
- Предложить улучшения (если есть)

---

## Troubleshooting

### CodeRabbit не комментирует PR?

1. Проверь что GitHub App установлен:
   - https://github.com/settings/installations
   - Должен быть **CodeRabbit** с доступом к MediaMate

2. Проверь что PR не draft:
   - Draft PR не ревьюятся по умолчанию

3. Проверь логи CodeRabbit:
   - Открой PR → вкладка "Checks" → CodeRabbit

### Слишком много комментариев?

Измени `profile` в `.github/.coderabbit.yaml`:

```yaml
reviews:
  profile: "chill"  # Поменяй на chill если слишком много
```

### Нужно игнорировать определённые файлы?

Добавь в `path_filters`:

```yaml
reviews:
  path_filters:
    - "!**/*.pb.go"          # Игнорировать protobuf
    - "!**/generated/**"     # Игнорировать generated код
    - "!vendor/**"           # Игнорировать vendor
```

---

## Стоимость

- **Open Source проекты:** Бесплатно ✅
- **Private repos (личное использование):** Бесплатно до 5000 строк/месяц
- **Private repos (команда):** Платные планы

Твой проект **MediaMate** — public, так что **100% бесплатно** без ограничений!

---

## Полезные ссылки

- Документация: https://docs.coderabbit.ai
- Dashboard: https://app.coderabbit.ai
- Примеры: https://github.com/coderabbitai/coderabbit-examples

---

## Bonus: Интеграция с другими инструментами

CodeRabbit работает отлично с уже настроенными workflows:

- ✅ **CodeQL** — находит security issues
- ✅ **golangci-lint** — линтинг кода
- ✅ **Tests** — проверяет что тесты проходят
- ✅ **CodeRabbit** — AI ревью логики и архитектуры

Вместе они создают мощную систему проверки качества кода!
