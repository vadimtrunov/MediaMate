# CodeRabbit Setup Instructions

## What is CodeRabbit?

CodeRabbit is an AI-powered code review tool that automatically reviews every Pull Request in your repo.

**Free for open source projects!**

---

## Installation (2 minutes)

### Step 1: Install GitHub App

1. Open https://coderabbit.ai
2. Click **"Sign in with GitHub"**
3. Select **"Install CodeRabbit"**
4. Select repository: **vadimtrunov/MediaMate**
5. Click **"Install & Authorize"**

### Step 2: Done!

CodeRabbit will automatically detect the `.github/.coderabbit.yaml` config and start working.

---

## How It Works

### Automatic PR Review

When you create a Pull Request:

1. CodeRabbit automatically analyzes the code
2. Comments on potential issues directly in the PR
3. Gives improvement recommendations
4. Generates a brief summary of changes

### Commands in PR

You can use commands in PR comments:

```
@coderabbitai help                  # Show all available commands
@coderabbitai review                # Re-run review
@coderabbitai explain               # Explain changes
@coderabbitai fix                   # Suggest a fix
@coderabbitai generate tests        # Generate tests
```

### Example Output

```markdown
## CodeRabbit Summary

### Changes
- Added LLM interface in `internal/llm/interface.go`
- Implemented Claude client with retry logic
- Added configuration for API keys

### Potential Issues
Warning: `internal/llm/claude/client.go:42`
Consider adding context timeout to API calls

### Suggestions
`internal/config/config.go:15`
Use environment variables for sensitive data

### Security
No security issues detected
```

---

## Configuration (already done)

Config is located at `.github/.coderabbit.yaml`:

```yaml
language: "en"
enable_free_tier: true

reviews:
  profile: "chill"              # Relaxed mode (not too picky)
  auto_review:
    enabled: true               # Automatic review on every PR
    drafts: false               # Don't review draft PRs

  path_filters:
    - "!**/*.md"                # Don't review Markdown files
    - "!**/*.json"              # Don't review JSON
    - "!**/*.yaml"              # Don't review YAML

  path_instructions:
    - path: "internal/**/*.go"
      instructions: |
        - Focus on Go best practices
        - Check error handling
        - Look for race conditions
        - Verify context usage
```

### Customization options:

- **`profile`** — strictness level:
  - `"assertive"` — strict (many comments)
  - `"chill"` — relaxed (only important stuff) <- **current**
  - `"default"` — balanced

- **`path_filters`** — which files to ignore

- **`path_instructions`** — specific instructions for different parts of the code

---

## Verifying It Works

### Create a test PR:

```bash
# Create a new branch
git checkout -b test/coderabbit-test

# Add a test file
cat > test.go <<EOF
package main

func add(a, b int) int {
    return a + b  // Simple function
}
EOF

git add test.go
git commit -m "test: add simple function"
git push origin test/coderabbit-test

# Create PR
gh pr create --title "Test: CodeRabbit integration" --body "Testing AI code review"
```

Within 10-30 seconds CodeRabbit should:
- Comment on the PR
- Give a summary of changes
- Suggest improvements (if any)

---

## Troubleshooting

### CodeRabbit not commenting on PR?

1. Check that the GitHub App is installed:
   - https://github.com/settings/installations
   - **CodeRabbit** should have access to MediaMate

2. Check that the PR is not a draft:
   - Draft PRs are not reviewed by default

3. Check CodeRabbit logs:
   - Open PR → "Checks" tab → CodeRabbit

### Too many comments?

Change `profile` in `.github/.coderabbit.yaml`:

```yaml
reviews:
  profile: "chill"  # Switch to chill if there are too many
```

### Need to ignore specific files?

Add to `path_filters`:

```yaml
reviews:
  path_filters:
    - "!**/*.pb.go"          # Ignore protobuf
    - "!**/generated/**"     # Ignore generated code
    - "!vendor/**"           # Ignore vendor
```

---

## Pricing

- **Open source projects:** Free
- **Private repos (personal use):** Free up to 5000 lines/month
- **Private repos (team):** Paid plans

Your project **MediaMate** is public, so it's **100% free** with no limitations!

---

## Useful Links

- Documentation: https://docs.coderabbit.ai
- Dashboard: https://app.coderabbit.ai
- Examples: https://github.com/coderabbitai/coderabbit-examples

---

## Bonus: Integration with Other Tools

CodeRabbit works great with the already configured workflows:

- **CodeQL** — finds security issues
- **golangci-lint** — code linting
- **Tests** — checks that tests pass
- **CodeRabbit** — AI review of logic and architecture

Together they create a powerful code quality system!
