#!/bin/bash

# Script d'installation de govulncheck
# Usage: ./scripts/install-govulncheck.sh

set -e

echo "📦 Installation de govulncheck..."

# Vérifier que Go est installé
if ! command -v go &> /dev/null; then
    echo "❌ Go n'est pas installé"
    exit 1
fi

# Installer govulncheck
echo "🔧 Installation en cours..."
go install golang.org/x/vuln/cmd/govulncheck@latest

# Vérifier l'installation
if command -v govulncheck &> /dev/null; then
    echo "✅ govulncheck installé avec succès"
    echo "📍 Emplacement: $(which govulncheck)"
    echo ""
    echo "💡 Vous pouvez maintenant utiliser:"
    echo "   make deps-vuln"
    echo "   ou directement: govulncheck ./..."
else
    # Vérifier si Go bin est dans le PATH
    GOPATH=$(go env GOPATH)
    if [ -f "$GOPATH/bin/govulncheck" ]; then
        echo "✅ govulncheck installé dans $GOPATH/bin/govulncheck"
        echo ""
        echo "⚠️  Le répertoire Go bin n'est pas dans votre PATH"
        echo "💡 Ajoutez cette ligne à votre ~/.bashrc ou ~/.zshrc:"
        echo "   export PATH=\$PATH:$GOPATH/bin"
        echo ""
        echo "   Ou utilisez directement: $GOPATH/bin/govulncheck ./..."
    else
        echo "❌ Échec de l'installation"
        exit 1
    fi
fi
