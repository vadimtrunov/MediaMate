#!/bin/bash
# Install git hooks for MediaMate

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
HOOKS_DIR="$PROJECT_ROOT/.git/hooks"

echo "🔧 Installing git hooks..."

# Check if we're in a git repository
if [ ! -d "$PROJECT_ROOT/.git" ]; then
    echo "❌ Error: Not a git repository"
    exit 1
fi

# Install pre-commit hook
echo "  📝 Installing pre-commit hook..."
cp -f "$SCRIPT_DIR/pre-commit.sh" "$HOOKS_DIR/pre-commit"
chmod +x "$HOOKS_DIR/pre-commit"

# Create commit-msg hook for conventional commits
echo "  💬 Installing commit-msg hook..."
cat > "$HOOKS_DIR/commit-msg" << 'EOF'
#!/bin/bash
# Conventional Commits validator

commit_msg_file=$1
SUBJECT_LINE=$(head -n 1 "$commit_msg_file")

# Allow merge commits
if echo "$SUBJECT_LINE" | grep -qE "^Merge "; then
    exit 0
fi

# Allow revert commits
if echo "$SUBJECT_LINE" | grep -qE "^Revert "; then
    exit 0
fi

# Check conventional commit format
if ! echo "$SUBJECT_LINE" | grep -qE "^(feat|fix|docs|style|refactor|perf|test|chore|build|ci|revert)(\(.+\))?: .{1,}"; then
    echo "❌ Invalid commit message format!"
    echo ""
    echo "Commit message must follow Conventional Commits format:"
    echo "  <type>[optional scope]: <description>"
    echo ""
    echo "Types: feat, fix, docs, style, refactor, perf, test, chore, build, ci, revert"
    echo ""
    echo "Examples:"
    echo "  feat: add user authentication"
    echo "  fix(api): handle null pointer exception"
    echo "  docs: update README"
    echo ""
    exit 1
fi

# Check for proper capitalization
if echo "$SUBJECT_LINE" | grep -qE "^[a-z]+(\(.+\))?: [A-Z]"; then
    echo "⚠️  Warning: Commit description should start with lowercase"
fi

# Check length
if [ ${#SUBJECT_LINE} -gt 72 ]; then
    echo "⚠️  Warning: Subject line is too long (${#SUBJECT_LINE} chars, max 72)"
fi

exit 0
EOF

chmod +x "$HOOKS_DIR/commit-msg"

echo ""
echo "✅ Git hooks installed successfully!"
echo ""
echo "Installed hooks:"
echo "  • pre-commit  - Runs code quality checks"
echo "  • commit-msg  - Validates conventional commit format"
echo ""
echo "To skip hooks (not recommended): git commit --no-verify"
