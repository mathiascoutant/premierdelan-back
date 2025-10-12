#!/bin/bash

echo "🧪 Script de test pour les notifications PWA"
echo "=============================================="
echo ""

# 1. Se connecter pour obtenir un token
echo "1️⃣  Connexion..."
TOKEN=$(curl -s -X POST http://localhost:8090/api/connexion \
  -H "Content-Type: application/json" \
  -d '{"email":"test@email.com","password":"password123"}' \
  | python3 -c "import sys, json; print(json.load(sys.stdin)['token'])" 2>/dev/null)

if [ -z "$TOKEN" ]; then
  echo "❌ Échec de la connexion"
  exit 1
fi

echo "✅ Token obtenu: ${TOKEN:0:30}..."
echo ""

# 2. Envoyer une notification de test
echo "2️⃣  Envoi d'une notification à tous les abonnés..."
echo ""

RESPONSE=$(curl -s -X POST http://localhost:8090/api/notification/test \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "user_id": "test@email.com",
    "title": "🎉 Notification de test",
    "message": "Ceci est une notification de test envoyée depuis l'\''API!",
    "data": {
      "action": "test",
      "timestamp": "2025-01-01"
    }
  }')

echo "$RESPONSE" | python3 -m json.tool

echo ""
echo "✅ Test terminé!"
echo ""
echo "📝 Pour voir les logs du serveur:"
echo "   tail -f /tmp/server_notif.log"

