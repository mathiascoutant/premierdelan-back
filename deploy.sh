#!/bin/bash

# Script de déploiement pour le backend
# Usage: ./deploy.sh

set -e  # Arrêter en cas d'erreur

echo "🔄 Déploiement du backend..."

# Arrêter le service
echo "⏹️  Arrêt du service backend..."
sudo systemctl stop backend || echo "⚠️  Service déjà arrêté"

# Aller dans le répertoire du projet
cd "$(dirname "$0")"

# Récupérer les dernières modifications
echo "📥 Récupération des modifications depuis Git..."
git pull origin main

# Compiler le projet
echo "🔨 Compilation du projet..."
go build -o backend .

# Vérifier que la compilation a réussi
if [ ! -f "./backend" ]; then
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
