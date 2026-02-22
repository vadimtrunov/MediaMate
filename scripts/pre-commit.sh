#!/bin/bash
# Pre-commit hook для MediaMate

set -e

echo "🔍 Running pre-commit checks..."

# Форматирование
echo "📝 Formatting code..."
gofumpt -l -w .

# Импорты
echo "📦 Organizing imports..."
goimports -w -local github.com/vadimtrunov/MediaMate .

# Линтинг
echo "🔎 Running linter..."
golangci-lint run ./...

# Тесты
echo "🧪 Running tests..."
go test -race ./...

echo "✅ Pre-commit checks passed!"
