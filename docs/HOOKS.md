# Git Hooks Documentation

MediaMate uses Git hooks to maintain code quality and prevent broken code from being committed.

## Installation

Install the hooks automatically:

```bash
make install-hooks
```

Or manually:

```bash
bash scripts/install-hooks.sh
```

## Available Hooks

### Pre-commit Hook

Runs before each commit to ensure code quality. Performs the following checks:

#### 1. Code Formatting
- **Tool:** `gofumpt`
- **What it checks:** All Go files are properly formatted
- **Fix:** Run `make fmt`

#### 2. Go Vet
- **Tool:** `go vet`
- **What it checks:** Common Go programming errors
- **Fix:** Address the issues reported by `go vet`

#### 3. Linting
- **Tool:** `golangci-lint`
- **What it checks:** Code quality, potential bugs, style issues
- **Fix:** Run `make lint-fix` or manually fix reported issues

#### 4. Module Consistency
- **Tool:** `go mod tidy`
- **What it checks:** `go.mod` and `go.sum` are up-to-date
- **Fix:** Run `go mod tidy` and add the changes to your commit

#### 5. Tests
- **Tool:** `go test`
- **What it checks:** All tests pass
- **Fix:** Fix failing tests

#### 6. Secrets Detection
- **What it checks:** No API keys, passwords, or tokens in code
- **Fix:** Use environment variables or config files (add to `.gitignore`)

#### 7. Debug Statements
- **What it checks:** No `fmt.Print`, `println`, or `panic` calls
- **Note:** This is a warning, not a blocker
- **Fix:** Use structured logging (`slog`) instead

#### 8. File Sizes
- **What it checks:** No files larger than 1MB
- **Note:** This is a warning, not a blocker
- **Fix:** Consider using Git LFS for large files

#### 9. TODO Comments
- **What it checks:** TODOs have issue references (e.g., `TODO: #123`)
- **Note:** This is a warning, not a blocker
- **Fix:** Link TODOs to GitHub issues

### Commit-msg Hook

Validates commit messages follow [Conventional Commits](https://www.conventionalcommits.org/) format.

#### Format

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

#### Types

- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, semicolons, etc.)
- `refactor`: Code refactoring
- `perf`: Performance improvements
- `test`: Adding or updating tests
- `chore`: Maintenance tasks
- `build`: Build system changes
- `ci`: CI/CD changes
- `revert`: Revert a previous commit

#### Examples

**Good:**
```
feat: add user authentication
fix(api): handle null pointer exception
docs: update README with installation steps
refactor(config): simplify configuration loading
```

**Bad:**
```
Added feature          ❌ No type
Fix bug               ❌ No type
feat: Add Feature     ❌ Description starts with uppercase
```

#### Rules

1. **Type is required** (feat, fix, docs, etc.)
2. **Description is required** (after colon)
3. **Description should be lowercase**
4. **Subject line max 72 characters**
5. **Use imperative mood** ("add" not "added" or "adds")

## Skipping Hooks

**⚠️ Not recommended!**

To skip hooks in emergency situations:

```bash
git commit --no-verify
```

**Note:** This bypasses all quality checks. Use only when absolutely necessary.

## Troubleshooting

### Hook is not running

Check if the hook is executable:
```bash
ls -la .git/hooks/pre-commit
chmod +x .git/hooks/pre-commit
```

### Hook is too slow

For faster commits during development, you can temporarily disable specific checks by editing `.git/hooks/pre-commit`.

### golangci-lint not found

Install development tools:
```bash
make install-tools
```

### Tests are failing

Run tests manually to see detailed output:
```bash
make test
```

## Best Practices

1. **Run checks before committing:**
   ```bash
   make pre-commit
   ```

2. **Fix issues incrementally:**
   - Run `make fmt` after writing code
   - Run `make lint` periodically
   - Run `make test` after changes

3. **Keep commits small:**
   - Smaller commits = faster hook execution
   - Easier to debug when hooks fail

4. **Use conventional commits:**
   - Makes git history readable
   - Enables automatic changelog generation
   - Helps with semantic versioning

## CI/CD Integration

The same checks run in GitHub Actions CI/CD pipeline. Pre-commit hooks ensure you catch issues locally before pushing.

See `.github/workflows/ci.yml` for CI configuration.

## Related

- [Git Workflow](GIT_WORKFLOW.md) - Branch strategy and PR process
- [Linters](LINTERS.md) - Detailed linter configuration
- [Makefile](../Makefile) - Available make commands
