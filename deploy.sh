#!/bin/bash

# Script de déploiement pour le backend
# Usage: ./deploy.sh [branche]
#   branche: main (défaut) ou dev

set -e  # Arrêter en cas d'erreur

BRANCH="${1:-main}"

echo "🔄 Déploiement du backend (branche: $BRANCH)..."

# Arrêter le service
echo "⏹️  Arrêt du service backend..."
sudo systemctl stop backend || echo "⚠️  Service déjà arrêté"

# Aller dans le répertoire du projet
cd "$(dirname "$0")"

# Récupérer les dernières modifications
echo "📥 Récupération des modifications depuis Git (origin $BRANCH)..."
git fetch origin
git checkout "$BRANCH"
git pull origin "$BRANCH"

# Compiler le projet (GOPROXY=direct évite 403 sur certains VPS OVH)
echo "🔨 Compilation du projet..."
export GOPROXY=direct
go build -o backend .

# Vérifier que la compilation a réussi
if [[ ! -f "./backend" ]]; then
    echo "❌ Erreur: La compilation a échoué"
    exit 1
fi

echo "✅ Compilation réussie"

# Redémarrer le service
echo "▶️  Démarrage du service backend..."
sudo systemctl start backend

# Attendre un peu pour que le service démarre
sleep 2

# Vérifier le statut
echo "📊 Statut du service:"
sudo systemctl status backend --no-pager -l

echo ""
echo "✅ Déploiement terminé!"
echo "📋 Pour voir les logs en temps réel: journalctl -u backend -f"
