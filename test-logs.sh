#!/bin/bash
# Script de test pour vérifier que les logs fonctionnent

echo "Test des logs - envoi d'une requête POST vers /api/connexion"
echo ""

# Test avec curl
curl -X POST http://localhost:8080/api/connexion \
  -H "Content-Type: application/json" \
  -H "Origin: http://localhost:3000" \
  -d '{"email":"test@test.com","password":"test123"}' \
  -v

echo ""
echo "Vérifiez les logs avec: journalctl -u backend -f"
