# 🔐 Gestion des Dépendances et Surveillance des Vulnérabilités

## 📋 Processus de Gestion des Dépendances

Ce document décrit le processus de gestion des mises à jour des dépendances et de surveillance des vulnérabilités pour le projet.

## 🔄 Processus Régulier

### 1. Surveillance Hebdomadaire (Automatique)

- **Dependabot** : Configuration automatique via `.github/dependabot.yml`
  - Vérification hebdomadaire tous les lundis à 9h
  - Création automatique de Pull Requests pour les mises à jour
  - Groupement des mises à jour mineures et patches

### 2. Vérification Manuelle

```bash
# Vérifier l'état des dépendances
make deps-check

# Vérifier les vulnérabilités
make deps-vuln

# Ou utiliser le script directement
./scripts/check-dependencies.sh
```

## 🔒 Vérification des Vulnérabilités

### Outils Utilisés

1. **govulncheck** (Recommandé)

   ```bash
   # Installation
   ./scripts/install-govulncheck.sh
   # Ou manuellement:
   go install golang.org/x/vuln/cmd/govulncheck@latest

   # Utilisation
   make deps-vuln
   # Ou directement:
   govulncheck ./...
   ```

   **Note** : Si `govulncheck` n'est pas trouvé après installation, vérifiez que `$GOPATH/bin` est dans votre PATH.

2. **GitHub Dependabot**

   - Surveille automatiquement les vulnérabilités connues
   - Envoie des alertes via GitHub

3. **go list -u**
   - Permet de voir les mises à jour disponibles
   - `go list -u -m all`

## 📦 Types de Mises à Jour

### Mises à Jour Majeures

- Changements breaking possibles
- Nécessitent une analyse approfondie
- Processus manuel requis

### Mises à Jour Mineures

- Nouvelles fonctionnalités compatibles
- Peuvent être automatisées via Dependabot

### Mises à Jour de Patch

- Corrections de bugs et de sécurité
- Haute priorité pour les corrections de sécurité
- Automatisées via Dependabot

## 🔄 Processus de Mise à Jour

### Étape 1 : Identifier les Mises à Jour

```bash
# Vérifier les mises à jour disponibles
go list -u -m all

# Ou utiliser le script
make deps-check
```

### Étape 2 : Évaluer l'Impact

Pour chaque mise à jour :

1. **Consulter les release notes** du package
2. **Vérifier les breaking changes**
3. **Tester localement** avant déploiement
4. **Analyser les vulnérabilités** corrigées (si patch de sécurité)

### Étape 3 : Mettre à Jour

```bash
# Mise à jour d'une dépendance spécifique
go get -u package@version

# Mise à jour de toutes les dépendances (attention!)
make deps-update

# Mise à jour des patches/mineures uniquement (plus sûr)
make deps-update-minor

# Nettoyer les dépendances inutilisées
go mod tidy
```

### Étape 4 : Tests et Validation

```bash
# Exécuter les tests
make test

# Vérifier la compilation
make build

# Tester manuellement les fonctionnalités critiques
```

### Étape 5 : Déploiement

1. Commiter les changements (`go.mod` et `go.sum`)
2. Créer une Pull Request avec description des changements
3. Revoir et valider
4. Déployer en staging (si disponible)
5. Déployer en production après validation

## 🚨 Gestion des Vulnérabilités Critiques

### Processus d'Urgence

1. **Détection** : Alertes Dependabot ou vérification manuelle
2. **Évaluation** : Analyser la criticité et l'impact
3. **Correction Immédiate** : Mettre à jour vers la version corrigée
4. **Tests Rapides** : Vérifier que l'application fonctionne
5. **Déploiement** : Déployer la correction rapidement

```bash
# Exemple pour une vulnérabilité critique
go get -u vulnérable-package@version-corrigée
go mod tidy
make test
make build
git commit -m "security: fix vulnerability in package"
git push
```

## 📊 Reporting et Documentation

### Journal des Mises à Jour

Toutes les mises à jour sont documentées dans :

- Les commits Git (avec préfixe `chore:`, `fix:`, `security:`)
- Les Pull Requests Dependabot
- Ce document (pour les mises à jour importantes)

### Format de Commit Recommandé

```
chore(deps): update package-name from v1.0.0 to v1.1.0
fix(deps): fix security vulnerability in package-name
security(deps): patch critical vulnerability CVE-YYYY-XXXXX
```

## 🔍 Dépendances à Surveiller Particulièrement

### Dépendances Critiques

1. **go.mongodb.org/mongo-driver** - Base de données
2. **github.com/golang-jwt/jwt/v5** - Authentification
3. **github.com/gorilla/mux** - Routing HTTP
4. **github.com/joho/godotenv** - Configuration
5. **golang.org/x/crypto** - Cryptographie

### Fréquence de Vérification

- **Hebdomadaire** : Tous les lundis (automatique via Dependabot)
- **Mensuel** : Vérification manuelle approfondie
- **Immédiat** : En cas d'alerte de sécurité

## 📝 Checklist de Mise à Jour

- [ ] Vérifier les mises à jour disponibles
- [ ] Consulter les release notes et changelog
- [ ] Vérifier les breaking changes
- [ ] Tester localement
- [ ] Exécuter les tests unitaires
- [ ] Vérifier la compilation
- [ ] Analyser l'impact sur la performance
- [ ] Documenter les changements
- [ ] Créer une Pull Request
- [ ] Déployer en staging (si disponible)
- [ ] Valider en production

## 🔗 Ressources

- [Go Vulnerability Database](https://pkg.go.dev/vuln)
- [Dependabot Documentation](https://docs.github.com/en/code-security/dependabot)
- [Go Modules Documentation](https://go.dev/ref/mod)

## 📅 Historique des Mises à Jour Importantes

| Date       | Package | Version | Raison                                 |
| ---------- | ------- | ------- | -------------------------------------- |
| 2026-01-16 | Tous    | -       | Initialisation du processus de gestion |

---

**Dernière mise à jour** : 2026-01-16  
**Prochaine vérification prévue** : Lundi prochain (automatique via Dependabot)
