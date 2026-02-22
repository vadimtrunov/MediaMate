# GitHub Repository Setup with AI/LLM Integration

## Цель: Максимальная автоматизация через AI

---

## 1. GitHub Actions + AI Code Review

### 1.1 AI Code Review Bot
**Что:** Автоматический AI-ревью каждого PR

**Опции:**
- **CodeRabbit** (https://coderabbit.ai) — ✅ РЕКОМЕНДУЮ
  - GPT-4 powered
  - Line-by-line код ревью
  - Находит баги, security issues, best practices
  - Поддержка Go
  - Free для open source
  - Комментирует прямо в PR

- **Qodo (бывший Codium)** (https://qodo.ai)
  - Генерация unit тестов автоматически
  - Улучшение coverage
  - Free для open source

- **Sourcery** (https://sourcery.ai)
  - Рефакторинг предложения
  - Но больше для Python

**Файл:** `.github/workflows/code-review.yml`
```yaml
name: AI Code Review
on:
  pull_request:
    types: [opened, synchronize]

jobs:
  coderabbit:
    runs-on: ubuntu-latest
    steps:
      - uses: coderabbitai/coderabbit-action@v1
        with:
          github_token: ${{ secrets.GITHUB_TOKEN }}
```

### 1.2 Automated Test Generation
**Qodo Cover** — генерирует unit тесты для Go

`.github/workflows/generate-tests.yml`
```yaml
name: Generate Tests
on:
  pull_request:
    types: [opened]

jobs:
  qodo:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: Codium-ai/pr-agent@main
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          OPENAI_KEY: ${{ secrets.OPENAI_API_KEY }}
        with:
          command: /test
```

---

## 2. Dependabot + AI Security

### 2.1 Dependabot Auto-merge
Автоматический merge безопасных обновлений зависимостей

`.github/dependabot.yml`
```yaml
version: 2
updates:
  - package-ecosystem: "gomod"
    directory: "/"
    schedule:
      interval: "weekly"
    open-pull-requests-limit: 10
    reviewers:
      - "vadimtrunov"
    labels:
      - "dependencies"
      - "automerge"

  - package-ecosystem: "docker"
    directory: "/"
    schedule:
      interval: "weekly"

  - package-ecosystem: "github-actions"
    directory: "/"
    schedule:
      interval: "weekly"
```

### 2.2 AI Security Scanning
**Snyk** + **CodeQL** + **Trivy**

`.github/workflows/security.yml`
```yaml
name: Security Scan
on:
  push:
    branches: [main]
  pull_request:
  schedule:
    - cron: '0 0 * * 0'  # Weekly

jobs:
  codeql:
    runs-on: ubuntu-latest
    permissions:
      security-events: write
    steps:
      - uses: actions/checkout@v4
      - uses: github/codeql-action/init@v3
        with:
          languages: go
      - uses: github/codeql-action/autobuild@v3
      - uses: github/codeql-action/analyze@v3

  snyk:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: snyk/actions/golang@master
        env:
          SNYK_TOKEN: ${{ secrets.SNYK_TOKEN }}
        with:
          args: --severity-threshold=high

  trivy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: aquasecurity/trivy-action@master
        with:
          scan-type: 'fs'
          scan-ref: '.'
          format: 'sarif'
          output: 'trivy-results.sarif'
      - uses: github/codeql-action/upload-sarif@v3
        with:
          sarif_file: 'trivy-results.sarif'
```

---

## 3. AI-Powered PR Assistant

### 3.1 PR Description Generator
**PR Agent** от Codium AI — автоматически генерирует описание PR

`.github/workflows/pr-agent.yml`
```yaml
name: PR Agent
on:
  pull_request:
    types: [opened, reopened, ready_for_review]
  issue_comment:
    types: [created]

jobs:
  pr_agent:
    runs-on: ubuntu-latest
    steps:
      - uses: Codium-ai/pr-agent@main
        env:
          OPENAI_KEY: ${{ secrets.OPENAI_API_KEY }}
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

**Команды в PR:**
- `/describe` — генерирует описание PR
- `/review` — AI код ревью
- `/improve` — предложения по улучшению
- `/test` — генерирует unit тесты
- `/ask "вопрос"` — задать вопрос о коде

### 3.2 Auto-labeling
AI определяет labels для PR/Issues

`.github/workflows/labeler.yml`
```yaml
name: Auto Label
on:
  pull_request:
    types: [opened, edited]
  issues:
    types: [opened, edited]

jobs:
  triage:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/labeler@v5
        with:
          repo-token: ${{ secrets.GITHUB_TOKEN }}
          configuration-path: .github/labeler.yml

      - uses: github/issue-labeler@v3
        with:
          repo-token: ${{ secrets.GITHUB_TOKEN }}
          configuration-path: .github/issue-labeler.yml
          enable-versioned-regex: 0
```

---

## 4. CI/CD Pipeline with AI

### 4.1 Build & Test
`.github/workflows/ci.yml`
```yaml
name: CI
on:
  push:
    branches: [main]
  pull_request:

jobs:
  test:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        go-version: ['1.22', '1.23']
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: ${{ matrix.go-version }}
          cache: true

      - name: Install dependencies
        run: go mod download

      - name: Run tests
        run: go test -v -race -coverprofile=coverage.out ./...

      - name: Upload coverage
        uses: codecov/codecov-action@v4
        with:
          token: ${{ secrets.CODECOV_TOKEN }}
          files: ./coverage.out

  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      - uses: golangci/golangci-lint-action@v4
        with:
          version: latest

  build:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        goos: [linux]
        goarch: [amd64, arm64]
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'

      - name: Build
        env:
          GOOS: ${{ matrix.goos }}
          GOARCH: ${{ matrix.goarch }}
        run: |
          make build

      - name: Upload artifact
        uses: actions/upload-artifact@v4
        with:
          name: mediamate-${{ matrix.goos }}-${{ matrix.goarch }}
          path: bin/mediamate
```

### 4.2 Docker Build (Multi-arch)
`.github/workflows/docker.yml`
```yaml
name: Docker Build
on:
  push:
    branches: [main]
    tags: ['v*']
  pull_request:

env:
  REGISTRY: ghcr.io
  IMAGE_NAME: ${{ github.repository }}

jobs:
  build:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      packages: write
    steps:
      - uses: actions/checkout@v4

      - uses: docker/setup-qemu-action@v3
      - uses: docker/setup-buildx-action@v3

      - uses: docker/login-action@v3
        with:
          registry: ${{ env.REGISTRY }}
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - uses: docker/metadata-action@v5
        id: meta
        with:
          images: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}
          tags: |
            type=ref,event=branch
            type=ref,event=pr
            type=semver,pattern={{version}}
            type=semver,pattern={{major}}.{{minor}}
            type=sha

      - uses: docker/build-push-action@v5
        with:
          context: .
          platforms: linux/amd64,linux/arm64
          push: ${{ github.event_name != 'pull_request' }}
          tags: ${{ steps.meta.outputs.tags }}
          labels: ${{ steps.meta.outputs.labels }}
          cache-from: type=gha
          cache-to: type=gha,mode=max
```

---

## 5. Release Automation

### 5.1 Semantic Release with AI Changelog
**Release Drafter** — AI генерирует changelog

`.github/workflows/release.yml`
```yaml
name: Release Drafter
on:
  push:
    branches: [main]
  pull_request:
    types: [opened, reopened, synchronize]

permissions:
  contents: write
  pull-requests: write

jobs:
  update_release_draft:
    runs-on: ubuntu-latest
    steps:
      - uses: release-drafter/release-drafter@v6
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

`.github/release-drafter.yml`
```yaml
name-template: 'v$RESOLVED_VERSION'
tag-template: 'v$RESOLVED_VERSION'
categories:
  - title: '🚀 Features'
    labels:
      - 'feature'
      - 'enhancement'
  - title: '🐛 Bug Fixes'
    labels:
      - 'fix'
      - 'bugfix'
      - 'bug'
  - title: '🧰 Maintenance'
    label: 'chore'
change-template: '- $TITLE @$AUTHOR (#$NUMBER)'
template: |
  ## Changes

  $CHANGES

  ## Contributors

  $CONTRIBUTORS
```

### 5.2 Automated Releases
`.github/workflows/release-on-tag.yml`
```yaml
name: Release
on:
  push:
    tags: ['v*']

jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'

      - uses: goreleaser/goreleaser-action@v5
        with:
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

---

## 6. AI Documentation

### 6.1 Auto-generate Docs
**Mintlify** или **ReadMe.com** — AI генерирует документацию из кода

`.github/workflows/docs.yml`
```yaml
name: Update Docs
on:
  push:
    branches: [main]
    paths:
      - 'internal/**/*.go'
      - 'pkg/**/*.go'

jobs:
  generate-docs:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Generate godoc
        run: |
          go install golang.org/x/tools/cmd/godoc@latest
          godoc -http=:6060 &
          sleep 2

      - name: Build docs site
        run: |
          # Mkdocs build or similar
          pip install mkdocs-material
          mkdocs build

      - name: Deploy to GitHub Pages
        uses: peaceiris/actions-gh-pages@v3
        with:
          github_token: ${{ secrets.GITHUB_TOKEN }}
          publish_dir: ./site
```

---

## 7. AI Issue Management

### 7.1 Issue Templates with AI
`.github/ISSUE_TEMPLATE/bug_report.yml`
```yaml
name: Bug Report
description: File a bug report
labels: ["bug", "triage"]
body:
  - type: markdown
    attributes:
      value: |
        Thanks for taking the time to fill out this bug report!

  - type: textarea
    id: what-happened
    attributes:
      label: What happened?
      description: Also tell us, what did you expect to happen?
      placeholder: Tell us what you see!
    validations:
      required: true

  - type: input
    id: version
    attributes:
      label: Version
      placeholder: "v0.1.0"
    validations:
      required: true
```

### 7.2 AI Triage Bot
**GitHub Copilot для Issues** или **Linear** интеграция

`.github/workflows/issue-triage.yml`
```yaml
name: Issue Triage
on:
  issues:
    types: [opened]

jobs:
  triage:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/github-script@v7
        with:
          script: |
            const issue = context.payload.issue;

            // AI categorization logic
            const labels = [];

            if (issue.body.toLowerCase().includes('crash')) {
              labels.push('bug', 'priority:high');
            }

            if (issue.body.toLowerCase().includes('feature')) {
              labels.push('enhancement');
            }

            if (labels.length > 0) {
              await github.rest.issues.addLabels({
                owner: context.repo.owner,
                repo: context.repo.repo,
                issue_number: issue.number,
                labels: labels
              });
            }
```

---

## 8. Testing Automation

### 8.1 AI Test Coverage Bot
**Codecov** с AI insights

`codecov.yml`
```yaml
coverage:
  status:
    project:
      default:
        target: 80%
        threshold: 1%
    patch:
      default:
        target: 90%

comment:
  behavior: default
  layout: "reach,diff,flags,tree,files"
  show_critical_paths: true

github_checks:
  annotations: true
```

### 8.2 Mutation Testing
**Go-mutesting** для проверки качества тестов

`.github/workflows/mutation-test.yml`
```yaml
name: Mutation Testing
on:
  pull_request:

jobs:
  mutate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'

      - name: Install go-mutesting
        run: go install github.com/zimmski/go-mutesting/cmd/go-mutesting@latest

      - name: Run mutation tests
        run: |
          go-mutesting ./...
```

---

## 9. Performance Monitoring

### 9.1 Benchmarks
`.github/workflows/benchmark.yml`
```yaml
name: Benchmark
on:
  pull_request:

jobs:
  benchmark:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'

      - name: Run benchmarks
        run: go test -bench=. -benchmem ./... | tee benchmark.txt

      - name: Compare with main
        uses: benchmark-action/github-action-benchmark@v1
        with:
          tool: 'go'
          output-file-path: benchmark.txt
          github-token: ${{ secrets.GITHUB_TOKEN }}
          auto-push: true
```

---

## 10. Secrets Management

### 10.1 GitHub Secrets
Нужно добавить:
```bash
# AI Services
OPENAI_API_KEY          # Для PR Agent, test generation
ANTHROPIC_API_KEY       # Для Claude integration в MediaMate

# Code Quality
CODECOV_TOKEN           # Для code coverage
SNYK_TOKEN              # Для security scanning

# Optional
CODERABBIT_TOKEN        # Если нужен paid план
SONAR_TOKEN             # SonarCloud (опционально)
```

Добавить через:
```bash
gh secret set OPENAI_API_KEY
gh secret set ANTHROPIC_API_KEY
gh secret set CODECOV_TOKEN
```

---

## 11. Recommended Setup Order

### Phase 1: Базовая автоматизация
1. ✅ CI/CD (build, test, lint)
2. ✅ Docker multi-arch build
3. ✅ Dependabot
4. ✅ CodeQL security scanning

### Phase 2: AI Code Review
5. ✅ CodeRabbit для PR review
6. ✅ PR Agent для описаний
7. ✅ Qodo для генерации тестов

### Phase 3: Release & Docs
8. ✅ Release Drafter
9. ✅ GoReleaser
10. ✅ Auto-docs генерация

### Phase 4: Advanced
11. ✅ Mutation testing
12. ✅ Performance benchmarks
13. ✅ AI issue triage

---

## Инструменты Summary

| Категория | Инструмент | Назначение | Free OSS? |
|-----------|-----------|-----------|-----------|
| Code Review | CodeRabbit | AI ревью кода | ✅ |
| Test Gen | Qodo | Генерация unit тестов | ✅ |
| PR Assistant | PR Agent | Описания, улучшения | ✅ |
| Security | CodeQL + Snyk + Trivy | Уязвимости | ✅ |
| Coverage | Codecov | Test coverage | ✅ |
| Release | Release Drafter + GoReleaser | Changelog, релизы | ✅ |
| Docs | MkDocs Material | Документация | ✅ |
| Dependencies | Dependabot | Обновления зависимостей | ✅ |

---

## Estimated Setup Time

- **Phase 1 (Basic CI/CD):** 2-3 часа
- **Phase 2 (AI Review):** 1-2 часа
- **Phase 3 (Release):** 1 час
- **Phase 4 (Advanced):** 2-3 часа

**Total:** ~1 рабочий день для полной настройки

---

## Next Steps

1. Создать все `.github/workflows/*.yml` файлы
2. Настроить secrets в GitHub
3. Включить CodeRabbit в репе
4. Создать первый PR и протестировать AI review
5. Настроить branch protection rules (require PR review, CI pass)
