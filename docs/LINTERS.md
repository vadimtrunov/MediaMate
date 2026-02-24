# Linters Configuration Guide

## Strict but Reasonable Linters!

This project uses **45+ linters** for code quality control.

**Philosophy:** Catch real bugs and issues, don't annoy over trivial things.

---

## What's Enabled

### Security
- **gosec** — security vulnerability detection
  - Hardcoded passwords in code
  - Insecure HTTP connections
  - Weak encryption algorithms (MD5, SHA1, DES)
  - SQL injection
  - Path traversal
  - Command injection

### Bug Detection
- **errcheck** — all errors are checked
- **bodyclose** — HTTP response body is closed
- **rowserrcheck** — sql.Rows.Err is checked
- **sqlclosecheck** — sql.Rows/Stmt are closed
- **nilerr** — returning nil instead of error
- **nilnil** — disallow returning (nil, nil)
- **makezero** — correct usage of make
- **reassign** — reassignment checks

### Code Complexity
- **cyclop** — cyclomatic complexity <= 15
- **gocognit** — cognitive complexity <= 20
- **gocyclo** — duplicate complexity check
- **funlen** — functions <= 80 lines / 40 statements
- **nestif** — if nesting <= 4

### Code Style
- **gofmt** / **gofumpt** — code formatting
- **goimports** / **gci** — import sorting
- **stylecheck** — Go code style
- **revive** — golint replacement with extended checks
- **godot** — comments end with a period
- **whitespace** — extra whitespace
- **lll** — line length <= 140 characters

### Best Practices
- **govet** — official vet checker
- **staticcheck** — advanced static analysis
- **gocritic** — ~100 code quality checks
- **unconvert** — unnecessary type conversions
- **unparam** — unused parameters
- **wastedassign** — useless assignments
- **ineffassign** — ineffective assignments

### Code Smells
- **dupl** — code duplication (threshold: 100 tokens)
- **goconst** — repeated strings (>= 3 times) → constants
- **gomnd** — magic numbers → named constants
- **nakedret** — naked returns in long functions

### Error Handling
- **errorlint** — proper error wrap/unwrap
- **wrapcheck** — errors must be wrapped
- **errname** — error names end with Error

### Testing
- **tenv** — os.Setenv only in tests
- **thelper** — test helpers use t.Helper()
- **tparallel** — parallel tests done correctly
- **testpackage** — tests in separate package (_test)
- **testableexamples** — examples are testable

### Context & Concurrency
- **contextcheck** — context is properly passed
- **noctx** — HTTP request with context
- **exportloopref** — no loop variable leaks

### Naming & Conventions
- **goprintffuncname** — Printf functions named correctly
- **interfacebloat** — interface <= 10 methods
- **predeclared** — no redefinition of predeclared identifiers
- **usestdlibvars** — use stdlib constants
- **tagliatelle** — consistent tag style (json: snake_case)

### Unicode & Encoding
- **asciicheck** — ASCII only in code
- **bidichk** — dangerous bidirectional unicode characters
- **misspell** — typos in code/comments

### Dependencies
- **gomoddirectives** — go.mod directives check
- **gomodguard** — block forbidden dependencies

### Performance
- **gocritic** (performance tag) — performance issues

---

## What's Disabled in Tests

Tests have less strict rules:

```yaml
- gomnd        # Magic numbers allowed
- funlen       # Long test functions are OK
- gocognit     # Complexity doesn't matter in tests
- dupl         # Duplicates are acceptable
- lll          # Long lines are OK (e.g., URLs)
- goconst      # Repetitions are acceptable
- wrapcheck    # Error wrapping not required
```

---

## Usage

### Check code

```bash
make lint
```

### Auto-fix where possible

```bash
make lint-fix
```

### Strict mode (all linters)

```bash
make lint-strict
```

### Formatting

```bash
make fmt         # Format code
make imports     # Sort imports
```

### Pre-commit check

```bash
make pre-commit  # fmt + imports + lint + test
```

### Install git hook

```bash
make install-hooks
```

Now checks will run automatically before each commit.

---

## Examples of Detected Issues

### Magic Numbers

```go
// BAD
func calculate(x int) int {
    return x * 100
}

// GOOD
const multiplier = 100

func calculate(x int) int {
    return x * multiplier
}
```

### Unchecked Errors

```go
// BAD
file, _ := os.Open("config.yaml")
defer file.Close()

// GOOD
file, err := os.Open("config.yaml")
if err != nil {
    return fmt.Errorf("open config: %w", err)
}
defer func() {
    if cerr := file.Close(); cerr != nil {
        log.Warn("failed to close file", "error", cerr)
    }
}()
```

### Code Duplication

```go
// BAD
func createUser() {
    db.Exec("INSERT INTO users ...")
    log.Info("User created")
    metrics.Inc("users_created")
}

func createAdmin() {
    db.Exec("INSERT INTO users ...")
    log.Info("User created")
    metrics.Inc("users_created")
}

// GOOD
func createUser(isAdmin bool) {
    insertUser(isAdmin)
}

func insertUser(isAdmin bool) {
    db.Exec("INSERT INTO users ...")
    log.Info("User created")
    metrics.Inc("users_created")
}
```

### High Complexity

```go
// BAD - cyclomatic complexity 18
func processRequest(r *Request) error {
    if r.Method == "GET" {
        if r.Auth != nil {
            if r.Auth.Valid() {
                // ...
                if x > 10 {
                    if y < 5 {
                        // Too much nesting!
                    }
                }
            }
        }
    }
}

// GOOD - split into functions
func processRequest(r *Request) error {
    if err := validateRequest(r); err != nil {
        return err
    }
    return handleRequest(r)
}

func validateRequest(r *Request) error { /* ... */ }
func handleRequest(r *Request) error { /* ... */ }
```

### Unwrapped Errors

```go
// BAD
func loadConfig() error {
    _, err := os.ReadFile("config.yaml")
    if err != nil {
        return err  // Lost context!
    }
}

// GOOD
func loadConfig() error {
    _, err := os.ReadFile("config.yaml")
    if err != nil {
        return fmt.Errorf("load config: %w", err)
    }
}
```

### Missing Context

```go
// BAD
func fetchData() ([]byte, error) {
    resp, err := http.Get("https://api.example.com")
    // ...
}

// GOOD
func fetchData(ctx context.Context) ([]byte, error) {
    req, err := http.NewRequestWithContext(ctx, "GET", "https://api.example.com", nil)
    // ...
}
```

---

## Disabling Linters (if absolutely necessary)

### For a specific line

```go
//nolint:gosec // G104: We intentionally ignore this error
defer file.Close()
```

An explanation is required (require-explanation: true)

### For a function

```go
//nolint:funlen,gocognit // This is a complex initialization function
func setupApplication() {
    // ...
}
```

### For a file

```go
//nolint:all
package generated

// Auto-generated code
```

---

## CI/CD Integration

Linters run automatically in the GitHub Actions CI/CD pipeline:

```yaml
- name: Run golangci-lint
  uses: golangci/golangci-lint-action@v4
  with:
    version: latest
    args: --timeout=5m
```

PRs will not pass if there are linter errors!

---

## Customization

If linters are too strict, you can relax them in `.golangci.yml`:

```yaml
linters-settings:
  funlen:
    lines: 120        # Was 80
    statements: 60    # Was 40

  cyclop:
    max-complexity: 20  # Was 15
```

But **it's better to write quality code** instead of relaxing linters!

---

## Linter Statistics

Current configuration:

- **45+ active linters**
- **80+ checks** from gocritic + revive
- **Security checks** from gosec (medium severity)
- **5 complexity checks** (cyclop <= 20, gocognit <= 30, funlen <= 120)
- **15+ error checks**
- **10+ style checks**

**Balanced limits:**
- Functions: <= 120 lines (instead of 80)
- Complexity: <= 20/30 (instead of 15/20)
- Lines: <= 140 characters
- Duplicates: >= 150 tokens

**This makes code:**
- Safer (security)
- More readable (style)
- More maintainable (complexity)
- Bug-free (bug detection)
- More performant (performance)

## What's DISABLED (and why)

These linters are too strict or annoying:

- **varnamelen** — would complain about `i`, `j`, `k` in loops
- **testpackage** — tests in separate packages are not always convenient
- **godot** — period at the end of every comment is annoying
- **wrapcheck** — requires wrapping all errors (too aggressive)
- **tagliatelle** — dictates tag style (may not fit)
- **nilnil** — (nil, nil) is sometimes valid
- **gochecknoinits** — init() is sometimes needed
- **exhaustruct** — all struct fields (too strict)
- **goerr113** — requires sentinel errors (overkill)
- **paralleltest** — t.Parallel is not always needed
- **wsl** — whitespace linter (too picky)

**We can enable these later if needed!**

---

## Useful Commands

```bash
# Show all available linters
golangci-lint linters

# Show what each linter checks
golangci-lint help linters

# Check configuration
golangci-lint config

# Run only a specific linter
golangci-lint run --disable-all --enable=gosec

# Ignore specific files
golangci-lint run --skip-files=".*_test.go"
```

---

## Recommendations

1. **Install git hook**: `make install-hooks`
2. **Run before committing**: `make pre-commit`
3. **Fix warnings immediately**, don't accumulate debt
4. **Don't disable linters without a reason**
5. **If nolint is needed** — write an explanation

---

## Resources

- golangci-lint docs: https://golangci-lint.run
- All linters list: https://golangci-lint.run/usage/linters/
- Effective Go: https://go.dev/doc/effective_go
- Code Review Comments: https://go.dev/wiki/CodeReviewComments

---

## Problems?

### Linter crashes with timeout?

Increase timeout:

```bash
golangci-lint run --timeout=10m
```

### False positives?

Add to exclude-rules in `.golangci.yml`

### Too strict?

Not at all! This is good practice. But if you really need to:

```yaml
linters:
  disable:
    - gocritic  # For example
```

---

**Remember:** Linters are your friends — they find bugs before your users do!
