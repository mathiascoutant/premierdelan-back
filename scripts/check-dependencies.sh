#!/bin/bash

# Script de vérification des dépendances et vulnérabilités
# Usage: ./scripts/check-dependencies.sh

set -e

echo "🔍 Vérification des dépendances Go..."

# Aller dans le répertoire du projet
cd "$(dirname "$0")/.."

# Vérifier que Go est installé
if ! command -v go &> /dev/null; then
    echo "❌ Go n'est pas installé"
    exit 1
fi

# Afficher la version de Go
echo "📦 Version de Go: $(go version)"

# Vérifier les modules obsolètes
echo ""
echo "🔍 Vérification des modules obsolètes..."
go list -u -m all 2>/dev/null | grep -E "\[" || echo "✅ Tous les modules sont à jour"

# Vérifier les vulnérabilités avec govulncheck si disponible
if command -v govulncheck &> /dev/null; then
    echo ""
    echo "🔒 Vérification des vulnérabilités avec govulncheck..."
    if govulncheck ./... 2>/dev/null; then
        echo "✅ Aucune vulnérabilité connue détectée"
    else
        echo "⚠️  Des vulnérabilités ont été détectées. Vérifiez les résultats ci-dessus."
    fi
else
    echo ""
    echo "⚠️  govulncheck n'est pas installé."
    echo "💡 Pour installer: go install golang.org/x/vuln/cmd/govulncheck@latest"
    echo "   Puis relancer: make deps-vuln"
fi

# Afficher les versions actuelles des dépendances principales
echo ""
echo "📋 Versions actuelles des dépendances principales:"
go list -m -f '{{.Path}} {{.Version}}' all | grep -E "(mongo|gorilla|jwt|godotenv)" || true

# Vérifier les mises à jour disponibles
echo ""
echo "🔄 Mises à jour disponibles (top 10):"
go list -u -m -json all 2>/dev/null | grep -A 2 '"Update"' | head -20 || echo "✅ Aucune mise à jour disponible"

# Résumé
echo ""
echo "✅ Vérification terminée"
echo ""
echo "💡 Commandes utiles:"
echo "   - Mettre à jour toutes les dépendances: go get -u ./..."
echo "   - Mettre à jour une dépendance spécifique: go get -u package@version"
echo "   - Voir les changements: go mod tidy && git diff go.mod go.sum"
