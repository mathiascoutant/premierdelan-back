# Documentation Technique d'Exploitation

Documentation technique pour le suivi des équipes et les évolutions du logiciel.

---

## Architecture

### Stack technique
- **Langage** : Go 1.21+
- **Base de données** : MongoDB
- **API** : REST (Gorilla Mux)
- **WebSocket** : Gorilla WebSocket (chat)
- **Authentification** : JWT
- **Notifications** : Firebase Cloud Messaging

### Structure du projet
```
├── cmd/           # Commandes utilitaires
├── config/        # Configuration
├── database/      # Repositories MongoDB
├── handlers/      # Handlers HTTP
├── middleware/    # Middlewares (auth, CORS, logging)
├── models/        # Modèles de données
├── services/      # Services (FCM, Slack, cron)
├── utils/         # Utilitaires (JWT, validation)
├── websocket/     # Hub WebSocket
├── main.go        # Point d'entrée
└── deploy.sh      # Script de déploiement
```

---

## Démarrage

### Prérequis
- Go 1.21+
- MongoDB
- Variables d'environnement (voir `env.example`)

### Développement
```bash
make install   # Dépendances
make run       # Démarrer le serveur
# ou
make dev       # Avec rechargement automatique (air)
```

### Production
```bash
./deploy.sh    # Arrêt, pull, build, redémarrage
```

---

## Endpoints principaux

| Méthode | Route | Description |
|---------|-------|-------------|
| GET | /api/health | Santé du serveur (uptime, DB, Go version) |
| POST | /api/connexion | Connexion utilisateur |
| POST | /api/inscription | Inscription utilisateur |
| GET | /api/theme | Thème global (public) |
| GET | /api/evenements/public | Liste événements publics |
| GET | /ws/chat | WebSocket chat |

---

## Variables d'environnement

| Variable | Obligatoire | Description |
|----------|-------------|-------------|
| JWT_SECRET | Oui | Secret pour signature JWT |
| MONGO_URI | Oui | URI de connexion MongoDB |
| MONGO_DB | Oui | Nom de la base |
| PORT | Non | Port serveur (défaut: 8090) |
| HOST | Non | Interface (défaut: 0.0.0.0) |
| ENVIRONMENT | Non | development, staging, production |
| CORS_ALLOWED_ORIGINS | Oui | Origines autorisées (séparées par virgule) |

---

## Maintenance

### Logs
```bash
journalctl -u backend -f
```

### Vérifications
```bash
make quality      # Vet + Lint + Tests
make deps-vuln    # Vulnérabilités
make deps-check   # Dépendances obsolètes
```

### Redémarrage
```bash
sudo systemctl restart backend
```

---

## Dépannage

### Erreur 403 CORS
Ajouter l'origine dans `CORS_ALLOWED_ORIGINS` (fichier `.env`).

### Connexion MongoDB
Vérifier `MONGO_URI`, pare-feu, et exécuter `make run` pour tester.

### Health check
`GET /api/health` retourne le statut DB. Si `db_status: "error"`, vérifier MongoDB.

---

## Documents associés
- [ENVIRONNEMENTS.md](../ENVIRONNEMENTS.md) - Environnements dev/staging/prod
- [GESTION_DEPENDANCES.md](../GESTION_DEPENDANCES.md) - Gestion des dépendances
- [API_DOCUMENTATION.md](../API_DOCUMENTATION.md) - Documentation API complète

---

**Dernière mise à jour** : 2026-01-16
