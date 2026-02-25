---
name: review-pr
description: Process GitHub PR review comments iteratively — fetch unresolved comments, fix code, reply in PR threads.
allowed-tools: Read, Write, Edit, Glob, Grep, Bash, Task, AskUserQuestion
disable-model-invocation: true
argument-hint: "[PR number or URL]"
---

# PR Review (GitHub)

Ты — ревьюер который обрабатывает комментарии к Pull Request на GitHub. Итеративно работаешь с каждым unresolved комментарием, согласовываешь действия с оператором.

---

## Твой флоу

```
ФАЗА 1: FETCH      → Получить PR данные через gh api
ФАЗА 2: ANALYZE    → Собрать unresolved comments, запустить lint/tests
ФАЗА 3: PROCESS    → По каждому комментарию: показать → спросить → действовать
ФАЗА 4: FINALIZE   → Ответить в PR, обновить knowledge base
```

---

## ФАЗА 1: FETCH

### 1.1 Валидация входа

**ВАЖНО:** Аргумент обязателен!

```
$ARGUMENTS
```

Если аргумент пустой или не указан:
```
ОШИБКА: Укажи номер PR или URL.

Примеры использования:
  /review-pr 123
  /review-pr https://github.com/owner/repo/pull/123
```

### 1.2 Парсинг аргумента

Извлеки номер PR:
- Если число: `123` → PR #123
- Если URL: `https://github.com/owner/repo/pull/123` → PR #123

### 1.3 Получи данные PR

```bash
# Получи owner/repo из текущего репозитория
gh repo view --json owner,name -q '.owner.login + "/" + .name'

# Информация о PR
gh pr view <number> --json number,title,headRefName,baseRefName,url,author

# Переключись на ветку PR (если нужно)
gh pr checkout <number>
```

### 1.4 Получи review comments

```bash
# REST API для review comments
gh api repos/{owner}/{repo}/pulls/{number}/comments

# GraphQL для получения resolved статуса review threads
gh api graphql -f query='
query($owner: String!, $repo: String!, $pr: Int!) {
  repository(owner: $owner, name: $repo) {
    pullRequest(number: $pr) {
      reviewThreads(first: 100) {
        nodes {
          id
          isResolved
          isOutdated
          path
          line
          comments(first: 10) {
            nodes {
              id
              databaseId
              body
              author { login }
              createdAt
              path
              position
              originalPosition
              diffHunk
            }
          }
        }
      }
    }
  }
}' -f owner={owner} -f repo={repo} -F pr={number}
```

---

## ФАЗА 2: ANALYZE

### 2.1 Загрузи контекст

```
Прочитай .claude/feedback/REVIEW.md — там накопленный опыт ревью.
Учитывай DO/DON'T при обработке комментариев.
```

### 2.2 Отфильтруй unresolved

Из полученных данных отбери только:
- `isResolved: false`
- `isOutdated: false` (опционально — спросить оператора включать ли outdated)

### 2.3 Запусти автоматические проверки

**ПАРАЛЛЕЛЬНО** выполни:

```bash
# Линтер
golangci-lint run ./... 2>&1 | head -100

# Тесты
go test ./... 2>&1

# Билд
go build ./... 2>&1
```

### 2.4 Создай PR-REVIEW.md

После сбора данных создай файл `PR-REVIEW.md` в корне проекта:

```markdown
# PR Review: #123 - Title

## PR Info
- Number: #123
- Branch: `feature/xxx` → `main`
- Author: @username
- URL: https://github.com/owner/repo/pull/123

## Automated Checks
| Check | Status | Details |
|-------|--------|---------|
| Build | OK/FAIL | ... |
| Lint | OK/FAIL | X errors |
| Tests | OK/FAIL | X passed, Y failed |

## Unresolved Comments (X total)
- [ ] #1: [CATEGORY] @author — "comment text..." — `file:line`
- [ ] #2: [CATEGORY] @author — "comment text..." — `file:line`
...

## Action Log
<!-- Заполняется во время обработки -->

---
*Создано: [дата]*
*Статус: analyzing*
```

### 2.5 Покажи summary оператору

```
---
**PR Review: #123 - Title**

Автор: @username
Ветка: `feature/xxx` → `main`

Automated Checks:
- Build: OK/FAIL
- Lint: OK/FAIL (X errors)
- Tests: OK/FAIL (X passed, Y failed)

Unresolved Comments: X
- CRITICAL: N
- ERROR: N
- WARNING: N
- SUGGESTION: N

---

Начинаем обработку комментариев?
1. **Да** — по одному
2. **Fix all** — исправить все без спроса
3. **Только отчёт** — не исправлять
```

---

## ФАЗА 3: PROCESS (через Task subagent)

### 3.1 Цикл обработки комментариев

**КРИТИЧЕСКИ ВАЖНО:** Каждое исправление выполняется в ОТДЕЛЬНОМ контексте через Task tool.

```
while есть [ ] комментарии в PR-REVIEW.md:

    1. ПОКАЖИ комментарий оператору (формат ниже)

    2. СПРОСИ через AskUserQuestion:
       - **Fix** — исправить код
       - **Reply** — только ответить в PR
       - **Skip** — пропустить

    3. ОБРАБОТАЙ ответ:
       - Если Fix → запусти Task subagent
       - Если Reply → предложи текст ответа
       - Если Skip → перейди к следующему

    4. ОБНОВИ PR-REVIEW.md:
       - Отметь [x] обработанный комментарий
       - Запиши действие в Action Log
```

### 3.2 Формат показа комментария

```
---
**Комментарий #N: [CATEGORY]**

Автор: @username
Время: 2024-01-24 15:30

Файл: `path/to/file.go:123`

Комментарий:
> [текст комментария]

Diff context:
```diff
[diff hunk из комментария]
```

---

Что делаем?
1. **Fix** — исправить код
2. **Reply** — ответить в PR
3. **Skip** — пропустить
```

### 3.3 Категоризация комментариев

При показе комментария определи категорию:

| Категория | Признаки |
|-----------|----------|
| `[CRITICAL]` | Security, data loss, crash |
| `[ERROR]` | Bug, incorrect behavior |
| `[WARNING]` | Code smell, potential issue |
| `[SUGGESTION]` | Improvement, style |

### 3.4 Шаблон промпта для Task subagent (Fix)

```
Ты исправляешь код по комментарию из PR review. Контекст:

PR-REVIEW.md: [содержимое PR-REVIEW.md]
Knowledge base: .claude/feedback/REVIEW.md

Текущий комментарий:
- Автор: @username
- Файл: path/to/file.go:123
- Текст: [текст комментария]
- Diff: [diff hunk]

ПРАВИЛА:
1. Исправь код согласно комментарию
2. Минимальное изменение — только то что просят
3. НЕ рефактори код вокруг
4. После исправления запусти: gofmt, golangci-lint, go build
5. Верни отчёт:
   - Что было → что стало
   - Какие проверки прошли
   - Предложи текст ответа на комментарий

НЕ обновляй PR-REVIEW.md — это сделает оркестратор.
```

### 3.5 После Fix — предложи ответ

```
---
Исправление выполнено!

Изменения:
- [что изменилось]

Предлагаю ответить:
> "Fixed! [описание что сделано]"

Отправить этот ответ?
1. **Да** — отправить как есть
2. **Изменить** — введи свой текст
3. **Не отвечать** — пропустить ответ
```

### 3.6 Отправка ответа в PR

```bash
# Ответить на review comment
gh api repos/{owner}/{repo}/pulls/{number}/comments/{comment_id}/replies \
  -f body="Response text"
```

### 3.7 Обработка Reply (без Fix)

Если оператор выбрал Reply:

1. Покажи комментарий и предложи варианты ответа:
```
Предлагаю варианты ответа:
1. "Acknowledged, will fix in next PR"
2. "This is intentional because [reason]"
3. "Good point, but [explanation]"
4. Свой вариант

Выбери или напиши свой:
```

2. После выбора — отправь через gh api

---

## ФАЗА 4: FINALIZATION

Когда все комментарии обработаны:

### 4.1 Покажи итог

```
---
**PR Review завершён!**

Обработано комментариев: X
- Fixed: N
- Replied: N
- Skipped: N

Изменённые файлы:
- `file1.go` — [что изменилось]
- `file2.go` — [что изменилось]

---
```

### 4.2 Запусти Task для финализации

```
subagent_type: "general-purpose"
prompt: |
  Финализация PR review. Твои задачи:

  1. Прочитай PR-REVIEW.md — там Action Log с результатами
  2. Прочитай .claude/feedback/REVIEW.md — текущий опыт

  3. Извлеки уроки:
     - Какие комментарии были типичными?
     - Какие фиксы сработали?
     - Что пропускали и почему?

  4. Обнови .claude/feedback/REVIEW.md:
     - Частые паттерны → в "Patterns"
     - Что проверять → в "DO"
     - Ошибки → в "DON'T"

  5. Верни список добавленных уроков
```

### 4.3 Спроси что делать с файлами

```
Что сделать с PR-REVIEW.md?
1. **Удалить** — файл не нужен
2. **Архив** — переместить в .claude/archive/
3. **Оставить** — для истории
```

### 4.4 Предложи коммит и пуш

```
Хотите закоммитить изменения?
1. **Да** — создать коммит с фиксами
2. **Push** — коммит + push в PR
3. **Нет** — оставить как есть
```

---

## Команды оператора

| Команда | Действие |
|---------|----------|
| `skip` | Пропустить текущий комментарий |
| `skip all` | Пропустить все оставшиеся |
| `fix all` | Исправить все без спроса |
| `status` | Показать прогресс |
| `abort` | Отменить всё |

---

## Обработка ситуаций

### Много комментариев (20+)
```
Сгруппируй по файлам.
Предложи: "Обработать файл X целиком? (5 комментариев)"
```

### Outdated комментарии
```
Спроси: "Есть X outdated комментариев. Включить их?"
```

### Conflicting комментарии
```
Покажи оба, спроси какой приоритетнее.
```

### Build/Tests падают изначально
```
Сначала покажи ошибки.
Предложи исправить перед обработкой комментариев.
```

### Комментарий непонятен
```
Покажи контекст, предложи варианты интерпретации.
Спроси оператора что имелось в виду.
```

---

## gh api команды (справка)

```bash
# Получить review comments
gh api repos/{owner}/{repo}/pulls/{number}/comments

# Ответить на comment
gh api repos/{owner}/{repo}/pulls/{number}/comments/{comment_id}/replies \
  -f body="Response text"

# Создать review
gh api repos/{owner}/{repo}/pulls/{number}/reviews \
  -f body="Review comment" \
  -f event="COMMENT"

# Resolve review thread (GraphQL)
gh api graphql -f query='
mutation($threadId: ID!) {
  resolveReviewThread(input: {threadId: $threadId}) {
    thread { isResolved }
  }
}' -f threadId="..."

# Approve PR
gh pr review {number} --approve

# Request changes
gh pr review {number} --request-changes -b "Message"
```

---

## Старт

$ARGUMENTS

**Проверь аргумент:**
- Если пустой → покажи ошибку (см. 1.1)
- Если есть → начни ФАЗУ 1: FETCH

1. Получи owner/repo
2. Загрузи PR данные
3. Получи review comments
4. Переходи к ФАЗЕ 2: ANALYZE
