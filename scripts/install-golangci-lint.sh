#!/bin/bash

# Script d'installation de golangci-lint
# Usage: ./scripts/install-golangci-lint.sh

set -e

echo "📦 Installation de golangci-lint..."

if ! command -v go &> /dev/null; then
    echo "❌ Go n'est pas installé"
    exit 1
fi

go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

if command -v golangci-lint &> /dev/null; then
    echo "✅ golangci-lint installé"
    golangci-lint --version
else
    GOPATH=$(go env GOPATH)
    echo "✅ golangci-lint installé dans $GOPATH/bin"
    echo "💡 Ajoutez au PATH: export PATH=\$PATH:$GOPATH/bin"
fi
