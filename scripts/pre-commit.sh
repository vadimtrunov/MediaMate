#!/bin/bash
# MediaMate Pre-commit Hook
# Prevents committing broken code to the repository

set -e
export PATH="$PATH:$HOME/go/bin"

echo "🔍 Running pre-commit checks..."

# Check if we're in a Go project
if [ ! -f "go.mod" ]; then
    echo "❌ Not a Go project (go.mod not found)"
    exit 1
fi

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Track if any checks fail
FAIL=0

# 1. Check Go formatting
echo -n "📝 Checking code formatting... "
UNFORMATTED=$(gofumpt -l . 2>&1)
if [ -z "$UNFORMATTED" ]; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${RED}✗${NC}"
    echo "  Unformatted files:"
    echo "$UNFORMATTED" | sed 's/^/    /'
    echo "  Run: make fmt"
    FAIL=1
fi

# 2. Run go vet
echo -n "🔎 Running go vet... "
VET_OUTPUT=$(go vet ./... 2>&1)
if [ -z "$VET_OUTPUT" ]; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${RED}✗${NC}"
    echo "$VET_OUTPUT"
    FAIL=1
fi

# 3. Run golangci-lint (if installed)
if command -v golangci-lint &> /dev/null; then
    echo -n "🚨 Running golangci-lint... "
    LINT_OUTPUT=$(golangci-lint run --timeout=5m ./... 2>&1)
    if echo "$LINT_OUTPUT" | grep -qE 'level=(error|warning)'; then
        echo -e "${RED}✗${NC}"
        echo "$LINT_OUTPUT"
        FAIL=1
    else
        echo -e "${GREEN}✓${NC}"
    fi
else
    echo -e "${YELLOW}⚠${NC} golangci-lint not installed (run: make install-tools)"
fi

# 4. Check for go.mod/go.sum consistency
echo -n "📦 Checking go.mod/go.sum... "
go mod tidy
if git diff --exit-code go.mod go.sum > /dev/null 2>&1; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${RED}✗${NC}"
    echo "  go.mod or go.sum is not tidy"
    echo "  Changes have been made - please review and add them to your commit"
    FAIL=1
fi

# 5. Run tests
echo -n "🧪 Running tests... "
TEST_OUTPUT=$(go test -short ./... 2>&1)
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${RED}✗${NC}"
    echo "$TEST_OUTPUT"
    FAIL=1
fi

# 6. Check for common issues
echo -n "🔒 Checking for secrets/sensitive data... "
SECRETS_FOUND=0

# Check for potential API keys or tokens in staged files
if git diff --cached --diff-filter=d | grep -iE '(api[_-]?key|secret[_-]?key|password|token|credentials).*[=:].*["\047][a-zA-Z0-9]{20,}["\047]'; then
    echo -e "${RED}✗${NC}"
    echo "  ⚠️  Potential secrets found in staged files!"
    echo "  Please review and use environment variables instead"
    SECRETS_FOUND=1
    FAIL=1
fi

if [ $SECRETS_FOUND -eq 0 ]; then
    echo -e "${GREEN}✓${NC}"
fi

# 7. Check for debug statements
echo -n "🐛 Checking for debug statements... "
DEBUG_FOUND=$(git diff --cached --diff-filter=d | grep -E '\+(.*)(fmt\.Print|log\.Print|println\(|panic\()' | grep -v '//' || true)
if [ -n "$DEBUG_FOUND" ]; then
    echo -e "${YELLOW}⚠${NC}"
    echo "  Warning: Debug/print statements found:"
    echo "$DEBUG_FOUND" | sed 's/^/    /'
    echo "  Consider removing them or using structured logging"
    # Don't fail, just warn
else
    echo -e "${GREEN}✓${NC}"
fi

# 8. Check file sizes
echo -n "📏 Checking file sizes... "
LARGE_FILES=""
for file in $(git diff --cached --name-only --diff-filter=d); do
    if [ -f "$file" ]; then
        # Cross-platform file size check
        if [ "$(uname)" = "Darwin" ]; then
            SIZE=$(stat -f%z "$file" 2>/dev/null || echo "0")
        else
            SIZE=$(stat -c%s "$file" 2>/dev/null || echo "0")
        fi

        if [ "$SIZE" -gt 1048576 ]; then # 1MB
            LARGE_FILES="${LARGE_FILES}${file} ($((SIZE / 1024))KB)\n"
        fi
    fi
done

if [ -n "$LARGE_FILES" ]; then
    echo -e "${YELLOW}⚠${NC}"
    echo "  Warning: Large files detected:"
    echo -e "$LARGE_FILES" | sed 's/^/    /'
    echo "  Consider using Git LFS for large files"
else
    echo -e "${GREEN}✓${NC}"
fi

# 9. Check for TODO/FIXME without issue reference
echo -n "📝 Checking TODOs... "
TODO_WITHOUT_ISSUE=$(git diff --cached --diff-filter=d | grep -E '^\+.*\b(TODO|FIXME)\b' | grep -vE '#[0-9]+' || true)
if [ -n "$TODO_WITHOUT_ISSUE" ]; then
    echo -e "${YELLOW}⚠${NC}"
    echo "  Warning: TODOs without issue reference:"
    echo "$TODO_WITHOUT_ISSUE" | sed 's/^/    /'
    echo "  Consider adding issue numbers (e.g., TODO: #123 fix this)"
else
    echo -e "${GREEN}✓${NC}"
fi

# Summary
echo ""
if [ $FAIL -eq 1 ]; then
    echo -e "${RED}❌ Pre-commit checks failed!${NC}"
    echo ""
    echo "Fix the issues above and try again."
    echo "Or skip this hook with: git commit --no-verify (not recommended)"
    exit 1
else
    echo -e "${GREEN}✅ All pre-commit checks passed!${NC}"
    exit 0
fi
