# Review Knowledge Base

## DO
- Always add `t.Parallel()` on parent test functions that don't use `t.Setenv`
- Always add `t.Parallel()` on subtests (unless parent uses `t.Setenv`)
- Split test functions approaching 60-line limit proactively (revive rule)
- Use interfaces for external service dependencies (not concrete types) — enables mocking and testability
- Add `Close()` method when a struct creates temp files or acquires resources — callers should `defer x.Close()`. If the resource is behind an interface, the interface must include `Close() error` too
- Make `Close()` idempotent: track whether already closed, return early if so, otherwise clean up and mark as closed. For file cleanup, ignore `os.ErrNotExist` on temp file removal
- Verify test assertions match actual YAML/config values (not just defaults)
- Keep `NewForTest()` constructors exported when needed cross-package — standard Go pattern
- Verify event lifecycle completeness: if there's a Start/Grab handler, there must be a matching Complete/Finish handler that cleans up state
- Add nil/empty guards on all webhook payload fields before using them — external services send garbage
- Escape user-provided strings (movie titles, etc.) before embedding in Markdown — prevents Telegram parse errors
- Add `ReadTimeout` and `WriteTimeout` to all `http.Server` instances (15s minimum)
- Add nil guards (panic or early return) for required dependencies in constructors (`NewService`, `NewTracker`, etc.)
- Wire all new config fields end-to-end: struct → YAML → env override → usage site — missing any link = silent bug
- Use ephemeral port `:0` in tests + `listener.Addr()` to get the actual port — avoids port conflicts in CI

## DON'T
- Don't reimplement stdlib functions (`strings.Contains`, `slices.Contains`, etc.) — use the standard library
- Don't use concrete types for dependencies in structs that need testing — define a minimal interface
- Don't put test constructors in `export_test.go` if they need cross-package access — `export_test.go` is only visible within the same package
- Don't add error checks on `json.NewEncoder(w).Encode()` in httptest handlers — lint doesn't flag it, risk is minimal
- Don't add `t.Parallel()` to tests using `t.Setenv` — Go panics
- Don't call a method that acquires a mutex from within a function that already holds the same mutex — causes deadlock or redundant locking. Return needed values from the locked section instead
- Don't iterate Go maps when output order matters (logs, UI text, tests) — use `slices.Sort` on keys first
- Don't silently ignore startup errors (e.g. webhook server bind failure) — propagate via error channel or fail fast
- Don't silently swallow invalid env var values — use a sentinel (e.g. `-1`) or return an explicit error
- Don't use inline comments for struct field documentation — use godoc-style comments above the field

## Patterns
- Test helper for shared setup: extract common test setup into `t.Helper()` function to share between split tests (e.g. `chatWithHistory`)
- Table-driven validate tests: extract `validateCase` struct + `runValidateTests` helper to keep functions under 60 lines
- Split by domain: `TestSetDefaults_Webhook` + `TestSetDefaults_AppConfig` instead of one monolithic function
- Panic tests: `defer func() { r := recover(); ... }()` with `strings.Contains` on stable token
- Mutex + return value: when a caller needs data computed under a lock, have the locked helper return the value instead of the caller re-acquiring the lock (e.g. `buildProgressText` returns `(string, int)`)
- Error channel for goroutine startup: `errCh := make(chan error, 1)` → goroutine sends bind error → caller selects on errCh with timeout

## Lessons Learned
- Adding `t.Parallel()` to all subtests can push a test function over the 60-line limit — account for +1 line per subtest when planning splits
- `export_test.go` with `package foo` is for exporting unexported symbols to external `_test` packages, but the exported function is only visible to tests of the same module — NOT to arbitrary external packages. For cross-package test helpers, use a regular exported function.
- Most common review issue category: **missing lifecycle/cleanup calls** (3 of 4 CRITICAL/MAJOR issues). After adding any "start" or "track" call, immediately add the corresponding "stop" or "complete" call before moving on.
- Second most common: **nil/empty input not guarded** (4 occurrences across both reviews). Webhook payloads, constructor dependencies, env vars — always validate inputs at trust boundaries.
- Third most common: **config not wired end-to-end** (2 occurrences). After adding a config field, trace the value from YAML → struct → flag/env → usage and verify each link.
- Map iteration order is a recurring source of flaky tests and inconsistent UI — default to sorting keys whenever map contents are rendered or compared.
- Fourth most common: **concrete types instead of interfaces** for dependencies. Using `*tmdb.Client` instead of an interface left 3/8 MCP tool handlers untestable. Always define a minimal interface at the consumer side.
- Temp file leaks: when a constructor creates temp files, always pair it with a `Close()` method. Flagged in claudecode client (3rd review).
