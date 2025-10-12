#!/bin/bash

echo "🧪 Test d'envoi de notification avec logs détaillés"
echo "===================================================="
echo ""

# Se connecter
echo "1️⃣  Connexion..."
RESPONSE=$(curl -s -X POST http://localhost:8090/api/connexion \
  -H "Content-Type: application/json" \
  -d '{"email":"mathias@gmail.com","password":"123"}')

TOKEN=$(echo "$RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin)['token'])" 2>/dev/null)

if [ -z "$TOKEN" ]; then
  echo "❌ Échec de la connexion"
  echo "$RESPONSE"
  exit 1
fi

echo "✅ Token obtenu"
echo ""

# Envoyer notification
echo "2️⃣  Envoi de la notification..."
curl -s -X POST http://localhost:8090/api/notification/test \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "user_id": "mathias@gmail.com",
    "title": "🎉 Test iPhone",
    "message": "Notification de test depuis le backend Go!"
  }' | python3 -m json.tool

echo ""
echo "✅ Requête envoyée! Vérifiez les logs du serveur pour plus de détails."

