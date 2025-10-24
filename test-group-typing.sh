#!/bin/bash

# Script de test pour les indicateurs de frappe des groupes
# Ce script simule l'envoi d'événements WebSocket typing pour les groupes

echo "🧪 Test des indicateurs de frappe pour les groupes"
echo "=================================================="

# Configuration
WS_URL="ws://localhost:8080/ws"
JWT_TOKEN="your-jwt-token-here"  # Remplacez par un vrai token
GROUP_ID="507f1f77bcf86cd799439011"  # Remplacez par un vrai group_id

echo "📡 Connexion WebSocket à: $WS_URL"
echo "👥 Group ID: $GROUP_ID"
echo ""

# Fonction pour envoyer un événement typing
send_typing_event() {
    local is_typing=$1
    local event_type="typing"
    
    echo "📤 Envoi événement typing: is_typing=$is_typing"
    
    # Message d'authentification
    echo '{"type":"authenticate","token":"'$JWT_TOKEN'"}'
    
    # Attendre un peu
    sleep 1
    
    # Message de typing
    echo '{"type":"typing","group_id":"'$GROUP_ID'","is_typing":'$is_typing'}'
}

echo "🔧 Instructions pour tester manuellement:"
echo "1. Connectez-vous à votre application frontend"
echo "2. Ouvrez la console du navigateur"
echo "3. Rejoignez un groupe de chat"
echo "4. Tapez dans le champ de message du groupe"
echo "5. Vérifiez les logs du backend pour voir les événements"
echo ""
echo "📋 Logs à surveiller dans le backend:"
echo "- 📤 Typing groupe: user=..., group=..., typing=true"
echo "- 📡 Broadcast groupe: GroupID=..., Exclude=..."
echo "- ✅ Message groupe envoyé à ..."
echo "- 📊 Broadcast groupe terminé: X messages envoyés"
echo ""
echo "🎯 Événements WebSocket à envoyer:"
echo ""
echo "1. Authentification:"
echo '{"type":"authenticate","token":"VOTRE_JWT_TOKEN"}'
echo ""
echo "2. Rejoindre le groupe:"
echo '{"type":"join_group","group_id":"'$GROUP_ID'"}'
echo ""
echo "3. Indicateur de frappe (commencer à taper):"
echo '{"type":"typing","group_id":"'$GROUP_ID'","is_typing":true}'
echo ""
echo "4. Indicateur de frappe (arrêter de taper):"
echo '{"type":"typing","group_id":"'$GROUP_ID'","is_typing":false}'
echo ""
echo "📥 Événements attendus du backend:"
echo '{"type":"user_typing","group_id":"'$GROUP_ID'","user_id":"user@example.com","username":"Nom Utilisateur","is_typing":true}'
echo ""
echo "✅ Test terminé - Vérifiez les logs du backend et du frontend"
