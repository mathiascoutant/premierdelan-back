#!/bin/bash

# Script de génération d'un rapport de surveillance des dépendances
# Usage: ./scripts/generate-deps-report.sh

set -e

echo "📊 Génération du rapport de surveillance des dépendances..."
echo "Date: $(date '+%Y-%m-%d %H:%M:%S')"
echo ""

cd "$(dirname "$0")/.."

# Créer le dossier reports s'il n'existe pas
mkdir -p reports

REPORT_FILE="reports/deps-report-$(date '+%Y%m%d').md"

cat > "$REPORT_FILE" << EOF
# 📊 Rapport de Surveillance des Dépendances

**Date** : $(date '+%d/%m/%Y à %H:%M:%S')  
**Environnement** : Production  
**Généré par** : Script automatisé

---

## 📋 Résumé Exécutif

Ce rapport présente l'état des dépendances du projet, les mises à jour disponibles, et les vulnérabilités détectées.

---

## 🔍 État des Dépendances

### Version de Go
\`\`\`
$(go version)
\`\`\`

### Dépendances Principales

| Package | Version Actuelle | Dernière Version | Statut |
|---------|------------------|------------------|--------|
EOF

# Extraire les dépendances principales
echo "Extraction des informations sur les dépendances principales..."

for pkg in "go.mongodb.org/mongo-driver" "github.com/golang-jwt/jwt/v5" "github.com/gorilla/mux" "github.com/gorilla/websocket" "github.com/joho/godotenv" "golang.org/x/crypto"; do
    current=$(go list -m -f '{{.Version}}' "$pkg" 2>/dev/null || echo "N/A")
    latest=$(go list -m -u -f '{{if .Update}}{{.Update.Version}}{{else}}{{.Version}}{{end}}' "$pkg" 2>/dev/null || echo "N/A")
    
    if [[ "$current" != "$latest" && "$latest" != "N/A" ]]; then
        status="⚠️ Mise à jour disponible"
    else
        status="✅ À jour"
    fi
    
    echo "| \`$pkg\` | \`$current\` | \`$latest\` | $status |" >> "$REPORT_FILE"
done

cat >> "$REPORT_FILE" << EOF

---

## 🔒 Analyse des Vulnérabilités

EOF

# Vérifier les vulnérabilités si govulncheck est installé
GOVULNCHECK_CMD=$(command -v govulncheck 2>/dev/null || echo "$(go env GOPATH)/bin/govulncheck")

if [[ -f "$GOVULNCHECK_CMD" ]] || command -v govulncheck &> /dev/null; then
    echo "Analyse des vulnérabilités en cours..."
    cat >> "$REPORT_FILE" << EOF
### Résultats govulncheck

\`\`\`
EOF
    $GOVULNCHECK_CMD ./... >> "$REPORT_FILE" 2>&1 || echo "⚠️ Des vulnérabilités ont été détectées (voir détails ci-dessus)" >> "$REPORT_FILE"
    echo "\`\`\`" >> "$REPORT_FILE"
else
    cat >> "$REPORT_FILE" << EOF
⚠️ **govulncheck n'est pas installé**

Pour installer : \`go install golang.org/x/vuln/cmd/govulncheck@latest\`

Ou utiliser : \`./scripts/install-govulncheck.sh\`
EOF
fi

cat >> "$REPORT_FILE" << EOF

---

## 🔄 Mises à Jour Disponibles

### Dépendances avec Mises à Jour Disponibles

EOF

# Lister les mises à jour disponibles (top 20)
echo "Extraction des mises à jour disponibles..."
go list -u -m all 2>/dev/null | grep -E "\[" | head -20 | while read line; do
    echo "- \`$line\`" >> "$REPORT_FILE"
done || echo "Aucune mise à jour disponible détectée" >> "$REPORT_FILE"

cat >> "$REPORT_FILE" << EOF

---

## 📈 Recommandations

### Actions Immédiates

EOF

# Analyser les vulnérabilités critiques
GOVULNCHECK_CMD=$(command -v govulncheck 2>/dev/null || echo "$(go env GOPATH)/bin/govulncheck")

if [[ -f "$GOVULNCHECK_CMD" ]] || command -v govulncheck &> /dev/null; then
    vuln_count=$($GOVULNCHECK_CMD ./... 2>&1 | grep -c "Found" || echo "0")
    if [[ "$vuln_count" -gt 0 ]]; then
        cat >> "$REPORT_FILE" << EOF
🚨 **Vulnérabilités détectées** : Des vulnérabilités ont été identifiées. Action immédiate requise :
   - Consulter les détails dans la section "Analyse des Vulnérabilités"
   - Prioriser les correctifs de sécurité
   - Appliquer les mises à jour critiques

EOF
    else
        echo "✅ Aucune vulnérabilité critique détectée" >> "$REPORT_FILE"
    fi
fi

cat >> "$REPORT_FILE" << EOF
### Actions Planifiées

- ✅ Surveillance automatique hebdomadaire via Dependabot (tous les lundis)
- 📋 Révision mensuelle des dépendances majeures
- 🔄 Application des patches de sécurité dans les 48h

---

## 📝 Historique des Actions

| Date | Action | Dépendance | Version | Raison |
|------|--------|------------|---------|--------|
| $(date '+%Y-%m-%d') | Rapport généré | - | - | Surveillance régulière |

---

## 🔗 Ressources

- [Documentation complète](../GESTION_DEPENDANCES.md)
- [Base de données de vulnérabilités Go](https://pkg.go.dev/vuln)
- [Dependabot Alerts](https://github.com/mathiascoutant/premierdelan-back/security/dependabot)

---

**Prochain rapport** : $(date -d '+7 days' '+%d/%m/%Y') (surveillance hebdomadaire)

EOF

echo "✅ Rapport généré : $REPORT_FILE"
echo ""
echo "📄 Aperçu du rapport :"
head -30 "$REPORT_FILE"
