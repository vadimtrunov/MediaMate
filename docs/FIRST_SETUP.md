# First-Time Setup Guide

## GitHub Repository Setup Checklist

After pushing these files to GitHub:

### Automatically enabled:

1. **GitHub Actions** (CI/CD)
   - Build & Test on every PR
   - Docker multi-arch builds
   - CodeQL security scanning
   - Release automation

2. **Dependabot**
   - Automatic dependency updates
   - Weekly checks for Go modules, Docker, GitHub Actions

3. **Issue Templates**
   - Structured forms for bug reports
   - Feature request templates

### Manual setup required:

#### 1. CodeRabbit (AI Code Review) - 2 minutes

See detailed instructions: [CODERABBIT_SETUP.md](CODERABBIT_SETUP.md)

**In short:**
1. Open https://coderabbit.ai
2. Sign in with GitHub
3. Install CodeRabbit on the `vadimtrunov/MediaMate` repository
4. Done!

#### 2. Codecov (Test Coverage) - optional

If you want to track test coverage:

1. Open https://codecov.io
2. Sign in with GitHub
3. Add repository: `vadimtrunov/MediaMate`
4. Copy `CODECOV_TOKEN`
5. Add to GitHub Secrets:
   ```bash
   gh secret set CODECOV_TOKEN
   # Paste the token from Codecov
   ```

#### 3. Branch Protection Rules - recommended

Protect the main branch:

```bash
# Via GitHub CLI
gh api repos/vadimtrunov/MediaMate/branches/main/protection -X PUT -f required_status_checks='{"strict":true,"contexts":["test","lint","build"]}' -f enforce_admins=false -f required_pull_request_reviews='{"required_approving_review_count":0}' -f restrictions=null
```

Or via the UI:
1. Settings → Branches → Add rule
2. Branch name pattern: `main`
3. Require a pull request before merging
4. Require status checks to pass before merging
   - Select: `test`, `lint`, `build`
5. Save changes

---

## First Commit

```bash
# Check that all files are added
git status

# Add all new files
git add .

# Commit
git commit -m "chore: setup GitHub workflows and automation

- Add CI/CD workflows (build, test, lint, docker)
- Add security scanning (CodeQL, Trivy, Gosec)
- Add release automation (GoReleaser, Release Drafter)
- Configure Dependabot for automated updates
- Add CodeRabbit configuration for AI code review
- Add issue and PR templates
- Add golangci-lint and goreleaser configs"

# Push
git push origin main
```

---

## Verifying Everything Works

After pushing, check GitHub Actions:

1. Open https://github.com/vadimtrunov/MediaMate/actions
2. The following workflows should start:
   - **CI** (build, test, lint)
   - **Security Scan** (CodeQL, Trivy)
   - **Release Drafter** (creates a draft release)

If something fails — that's normal at the initial stage (before there's any Go code).

---

## What's Next?

### Phase 0: Project Structure

Next step from [ROADMAP.md](ROADMAP.md):

1. Create Go module structure
2. Define core interfaces
3. Set up configuration
4. Write initial tests

### Create Your First PR

Verify that CodeRabbit works:

```bash
git checkout -b feat/project-structure
# ... create Go files ...
git add .
git commit -m "feat: add initial project structure"
git push origin feat/project-structure
gh pr create --title "feat: Add initial project structure" --body "Phase 0 from roadmap"
```

CodeRabbit will automatically comment on the PR!

---

## Useful Commands

```bash
# Local build
make build

# Run tests
make test

# Linting
make lint

# Check that workflows are valid
gh workflow list

# View status of the latest workflow
gh run list --limit 5

# View workflow logs
gh run view
```

---

## Troubleshooting

### GitHub Actions not running?

Check that workflows are enabled:
1. Settings → Actions → General
2. Allow all actions and reusable workflows

### CodeQL failing?

This is normal until there's Go code. It will work after creating `cmd/mediamate/main.go`.

### Docker build failing?

A `Dockerfile` needs to be created (will be done in Phase 0).

---

## Summary

**What already works (for free):**
- Automatic build and tests on every PR
- Security scanning (CodeQL, Trivy, Gosec)
- Dependabot updates
- Release automation
- Issue/PR templates

**What needs to be added:**
- CodeRabbit (2 minutes via UI)
- Branch protection (optional but recommended)
- Codecov (optional, for test coverage)

**Cost:** $0 — everything is free for open source!
