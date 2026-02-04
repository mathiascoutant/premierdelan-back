# Environnements de Déploiement

Ce document décrit les différents environnements du projet Premier de l'An.

## Chaîne de déploiement

```
Développement (local)  →  Staging (dev.visionshow.fr)  →  Production (visionshow.fr)
     main.go run              branche dev                     branche main
```

---

## 1. Environnement de développement (local)

**Objectif** : Développement et tests locaux

**Configuration** :
- Fichier : `.env` ou `env.example`
- `ENVIRONMENT=development`
- `HOST=localhost`
- `PORT=8090`
- MongoDB local ou Atlas (dev)
- CORS : `http://localhost:3000`, `http://localhost:5173`

**Démarrage** :
```bash
make run
# ou avec rechargement automatique
make dev
```

**Tests** :
```bash
make test
make lint
make vet
make deps-vuln
```

---

## 2. Environnement de staging (test)

**Objectif** : Validation avant mise en production, tests utilisateurs

**URL** : https://dev.visionshow.fr  
**Branche Git** : `dev`  
**Base de données** : MongoDB (base dédiée ou copie de prod recommandée)

**Configuration** :
- `ENVIRONMENT=staging` ou `production`
- CORS : inclure `https://dev.visionshow.fr`
- Variables d'environnement distinctes de la production

**Fichier exemple** : `env.staging.example`

**Déploiement** :
```bash
git checkout dev
git pull
./deploy.sh
```

**Vérifications** :
- Tester les fonctionnalités critiques
- Vérifier la connexion mobile
- Tester les notifications
- Valider les performances

---

## 3. Environnement de production

**Objectif** : Application en production pour les utilisateurs finaux

**URL** : https://visionshow.fr  
**Branche Git** : `main`  
**Base de données** : MongoDB production

**Configuration** :
- `ENVIRONMENT=production`
- `HOST=0.0.0.0`
- CORS : `https://visionshow.fr`, `https://dev.visionshow.fr`
- Toutes les variables sensibles configurées
- Monitoring actif (Slack pour erreurs)

**Déploiement** :
```bash
git checkout main
git pull
./deploy.sh
```

---

## Tableau récapitulatif

| Critère       | Développement | Staging            | Production         |
| ------------- | ------------- | ------------------ | ------------------ |
| URL           | localhost     | dev.visionshow.fr  | visionshow.fr      |
| Branche       | -             | dev                | main               |
| ENVIRONMENT   | development   | staging            | production         |
| Base de données | locale      | dédiée/test        | production         |
| CORS          | localhost:*   | dev.visionshow.fr  | visionshow.fr      |
| Monitoring    | logs console  | logs + Slack       | logs + Slack       |

---

## Outils de suivi et qualité

### Qualité du code
- **golangci-lint** : `make lint`
- **go vet** : `make vet`
- **Tests unitaires** : `make test`

### Sécurité
- **govulncheck** : `make deps-vuln`
- **Vérification dépendances** : `make deps-check`

### Santé de l'application
- **Health check** : `GET /api/health`
  - Statut serveur
  - Connexion MongoDB
  - Uptime
  - Version Go

### Monitoring
- Logs structurés (journalctl, fichiers)
- Notifications Slack pour erreurs critiques (5xx, CORS)

---

## Checklist avant déploiement en production

- [ ] Tests passent (`make test`)
- [ ] Lint OK (`make lint`)
- [ ] Aucune vulnérabilité critique (`make deps-vuln`)
- [ ] Validation sur staging (dev.visionshow.fr)
- [ ] Backup base de données si nécessaire
- [ ] Communication aux utilisateurs si changement majeur

---

**Dernière mise à jour** : 2026-01-16
