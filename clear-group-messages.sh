#!/bin/bash

# Script pour supprimer toutes les données de la table chat_group_messages
# ATTENTION: Cette action est IRRÉVERSIBLE !

echo "🗑️  Suppression de toutes les données de chat_group_messages"
echo "=========================================================="
echo ""

# Configuration MongoDB
MONGO_URI="mongodb://localhost:27017"
DATABASE_NAME="premierdelan"
COLLECTION_NAME="chat_group_messages"

echo "📊 Base de données: $DATABASE_NAME"
echo "📋 Collection: $COLLECTION_NAME"
echo ""

# Vérifier la connexion MongoDB
echo "🔍 Vérification de la connexion MongoDB..."
if ! mongosh --eval "db.runCommand('ping')" > /dev/null 2>&1; then
    echo "❌ Erreur: Impossible de se connecter à MongoDB"
    echo "💡 Assurez-vous que MongoDB est démarré"
    exit 1
fi

echo "✅ Connexion MongoDB OK"
echo ""

# Compter les documents avant suppression
echo "📊 Comptage des documents avant suppression..."
COUNT_BEFORE=$(mongosh --quiet --eval "db.$COLLECTION_NAME.countDocuments({})" $DATABASE_NAME)
echo "📋 Nombre de documents: $COUNT_BEFORE"
echo ""

if [ "$COUNT_BEFORE" -eq 0 ]; then
    echo "ℹ️  La collection est déjà vide"
    exit 0
fi

# Demander confirmation
echo "⚠️  ATTENTION: Cette action va supprimer $COUNT_BEFORE documents de façon IRRÉVERSIBLE !"
echo ""
read -p "Êtes-vous sûr de vouloir continuer ? (tapez 'SUPPRIMER' pour confirmer): " confirmation

if [ "$confirmation" != "SUPPRIMER" ]; then
    echo "❌ Opération annulée"
    exit 0
fi

echo ""
echo "🗑️  Suppression en cours..."

# Supprimer tous les documents
RESULT=$(mongosh --quiet --eval "db.$COLLECTION_NAME.deleteMany({})" $DATABASE_NAME)

if [ $? -eq 0 ]; then
    echo "✅ Suppression réussie"
    
    # Vérifier le résultat
    COUNT_AFTER=$(mongosh --quiet --eval "db.$COLLECTION_NAME.countDocuments({})" $DATABASE_NAME)
    echo "📊 Documents restants: $COUNT_AFTER"
    
    if [ "$COUNT_AFTER" -eq 0 ]; then
        echo "🎉 Tous les messages de groupe ont été supprimés avec succès"
    else
        echo "⚠️  Il reste $COUNT_AFTER documents"
    fi
else
    echo "❌ Erreur lors de la suppression"
    exit 1
fi

echo ""
echo "✅ Script terminé"
