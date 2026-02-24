# Git Workflow Guide

## Standard Git Flow

We use a simplified version of Git Flow with two main branches.

---

## Main Branches

### `main` — Production
- Always stable and ready for release
- Direct commits are **forbidden**
- Only via PR from `develop`
- Each merge to main = release

### `develop` — Development
- Main development branch
- Feature branches are merged here
- Always functional, but may contain new features
- Feature branches are created from here

---

## Feature Branches

### Creating a feature branch

```bash
# Make sure you're on develop
git checkout develop
git pull origin develop

# Create a feature branch
git checkout -b feat/llm-integration
# or
git checkout -b fix/config-validation
```

### Naming Convention

**Prefixes:**
- `feat/` — new functionality
- `fix/` — bug fix
- `refactor/` — refactoring
- `docs/` — documentation
- `test/` — tests
- `chore/` — infrastructure changes

**Examples:**
```
feat/claude-client
feat/radarr-integration
fix/config-loading
refactor/llm-interface
docs/api-reference
test/integration-tests
chore/update-deps
```

### Working on a feature

```bash
# Make changes
vim internal/llm/claude/client.go

# Commits
git add .
git commit -m "feat(llm): add Claude API client

- Implement basic Claude client
- Add retry logic with exponential backoff
- Add context timeout support"

# Push to remote
git push origin feat/claude-client
```

### Creating a Pull Request

```bash
# Create PR to develop (not to main!)
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

### After PR Review

```bash
# If changes are needed — just commit to the same branch
git add .
git commit -m "fix: address review comments"
git push origin feat/claude-client

# PR will be updated automatically
```

### After Merge

```bash
# Switch to develop
git checkout develop
git pull origin develop

# Delete local branch
git branch -d feat/claude-client

# Delete remote branch (usually automatic via GitHub)
git push origin --delete feat/claude-client
```

---

## Releases (main branch)

### When to release

When `develop` reaches a milestone:
- v0.1 — MVP is ready
- v0.2 — New features are stable
- v1.0 — Production ready

### Release process

```bash
# Make sure develop is stable
git checkout develop
git pull origin develop

# Update version in code (if applicable)
# vim internal/version/version.go

# Create PR from develop to main
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

# After merge to main — create a tag
git checkout main
git pull origin main
git tag -a v0.1.0 -m "Release v0.1.0

- Claude LLM integration
- Radarr backend support
- Telegram + CLI frontends
- Docker multi-arch build"

# Push the tag
git push origin v0.1.0

# GitHub Actions will automatically:
# - Run GoReleaser
# - Create a GitHub Release
# - Build binaries
# - Publish Docker images
```

---

## Hotfix

If there is a critical bug in production (main):

```bash
# Create a hotfix branch from main
git checkout main
git pull origin main
git checkout -b hotfix/critical-security-issue

# Fix
vim internal/security/fix.go
git add .
git commit -m "fix(security): patch critical vulnerability

CVE-2024-XXXXX - SQL injection in search endpoint"

# Push
git push origin hotfix/critical-security-issue

# PR to main (not to develop!)
gh pr create --base main --title "hotfix: Critical security patch" --body "..."

# After merge to main:
# 1. Create a tag (v0.1.1)
# 2. Merge main back into develop
git checkout develop
git merge main
git push origin develop
```

---

## Syncing with upstream

If develop is behind main:

```bash
git checkout develop
git pull origin develop
git merge main
# Resolve conflicts if any
git push origin develop
```

---

## Rules

### DO:
- Always create feature branches from `develop`
- Write clear commit messages
- PR only through GitHub (no direct push)
- Squash commits if there are too many
- Delete merged branches
- Regularly sync with develop

### DON'T:
- Direct commits to `main` or `develop`
- Huge PRs (>500 lines — better to split)
- PRs without tests
- Merge your own PRs without review (unless it's a personal project)
- Force push to `main` or `develop`

---

## Useful Commands

```bash
# Switch to develop
git checkout develop

# Update develop
git pull origin develop

# Create a feature branch
git checkout -b feat/my-feature

# Status
git status

# View changes
git diff

# History
git log --oneline --graph --all

# Discard uncommitted changes
git checkout -- file.go

# Soft reset of the last commit
git reset --soft HEAD~1

# Sync with remote
git fetch origin
git status

# View all branches
git branch -a

# Delete a local branch
git branch -d feat/old-feature

# Delete a remote branch
git push origin --delete feat/old-feature
```

---

## GitHub CLI Shortcuts

```bash
# Create PR
gh pr create

# View PR status
gh pr status

# List PRs
gh pr list

# Checkout PR locally
gh pr checkout 123

# Merge PR
gh pr merge 123

# View Actions
gh run list

# View workflow logs
gh run view
```

---

## Commit Message Format

We use Conventional Commits:

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat` — new functionality
- `fix` — bug fix
- `refactor` — refactoring
- `docs` — documentation
- `test` — tests
- `chore` — infrastructure
- `perf` — performance
- `style` — formatting

**Scope (optional):**
- `llm`, `radarr`, `telegram`, `config`, etc.

**Examples:**

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

**All development happens in feature branches → PR to develop → release via PR to main**

---

## Next Steps

1. Set up branch protection for `main`:
   ```bash
   # Require PR review
   # Require CI to pass
   # Forbid force push
   ```

2. Optionally: set up auto-merge for dependabot PRs

3. Start working on Phase 0 from the roadmap!

```bash
git checkout develop
git checkout -b feat/project-structure
# ... code ...
git commit -m "feat: add initial project structure"
gh pr create --base develop
```
