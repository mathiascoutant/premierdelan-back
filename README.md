# Backend Premier de l'An - API Go avec MongoDB

Backend propre et bien structuré en Go avec MongoDB pour gérer l'authentification des utilisateurs.

## 🏗️ Architecture

Le projet suit une architecture propre et modulaire :

```
back/
├── config/           # Configuration de l'application
├── database/         # Connexion MongoDB et repositories
├── handlers/         # Handlers HTTP
├── middleware/       # Middlewares (CORS, Auth, Logging)
├── models/          # Modèles de données
├── utils/           # Utilitaires (JWT, validation, password)
├── main.go          # Point d'entrée de l'application
├── go.mod           # Dépendances Go
└── env.example      # Exemple de variables d'environnement
```

## 🚀 Installation

### Prérequis

- Go 1.21 ou supérieur
- MongoDB 4.4 ou supérieur (en local sur le port 27017 par défaut)

### Étapes d'installation

1. **Installer les dépendances**
   ```bash
   cd back
   go mod download
   ```

2. **Démarrer MongoDB localement**
   ```bash
   # Sur macOS avec Homebrew
   brew services start mongodb-community
   
   # Ou avec Docker
   docker run -d -p 27017:27017 --name mongodb mongo:latest
   ```

3. **Configurer les variables d'environnement**
   
   Créer un fichier `.env` :
   ```bash
   cp env.example .env
   ```
   
   Le fichier `.env` contient :
   ```env
   PORT=8090
   HOST=localhost
   
   MONGO_URI=mongodb://localhost:27017
   MONGO_DB=premier_an_db
   
   JWT_SECRET=votre_secret_jwt_tres_securise
   ENVIRONMENT=development
   
   CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:5173
   ```

4. **Lancer l'application**
   ```bash
   # Avec Make
   make run
   
   # Ou directement avec Go
   go run main.go
   ```

   Le serveur démarre sur **http://localhost:8090**

## 📡 API Endpoints

### Routes Publiques

#### 1. Health Check
```
GET /api/health
```

**Réponse :**
```json
{
  "message": "Le serveur fonctionne correctement",
  "data": {
    "status": "ok",
    "env": "development",
    "database": "MongoDB"
  }
}
```

#### 2. Inscription
```
POST /api/inscription
Content-Type: application/json
```

**Corps de la requête (depuis votre front) :**
```json
{
  "code_soiree": "CODE123",
  "prenom": "Jean",
  "nom": "Dupont",
  "email": "jean@email.com",
  "telephone": "0612345678",
  "password": "motdepasse123"
}
```

**Réponse succès (201) :**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "user": {
    "id": "507f1f77bcf86cd799439011",
    "code_soiree": "CODE123",
    "firstname": "Jean",
    "lastname": "Dupont",
    "email": "jean@email.com",
    "phone": "0612345678",
    "created_at": "2025-10-09T21:55:00Z"
  }
}
```

#### 3. Connexion
```
POST /api/connexion
Content-Type: application/json
```

**Corps de la requête (depuis votre front) :**
```json
{
  "email": "jean@email.com",
  "password": "motdepasse123"
}
```

**Réponse succès (200) :**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "user": {
    "id": "507f1f77bcf86cd799439011",
    "code_soiree": "CODE123",
    "firstname": "Jean",
    "lastname": "Dupont",
    "email": "jean@email.com",
    "phone": "0612345678",
    "created_at": "2025-10-09T21:55:00Z"
  }
}
```

**Réponse erreur (401) :**
```json
{
  "error": "Unauthorized",
  "message": "Email ou mot de passe incorrect"
}
```

### Routes Protégées

Les routes protégées nécessitent un token JWT dans l'en-tête :
```
Authorization: Bearer <votre_token>
```

#### Exemple : Profil
```
GET /api/protected/profile
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
```

**Réponse :**
```json
{
  "message": "Profil récupéré avec succès",
  "data": {
    "user_id": "507f1f77bcf86cd799439011",
    "email": "jean@email.com"
  }
}
```

## 🗄️ Structure MongoDB

### Collection : `users`

```javascript
{
  "_id": ObjectId("507f1f77bcf86cd799439011"),
  "code_soiree": "CODE123",        // optionnel
  "firstname": "Jean",
  "lastname": "Dupont",
  "email": "jean@email.com",       // unique
  "phone": "0612345678",
  "password": "$2a$10$...",         // bcrypt hash
  "created_at": ISODate("2025-10-09T21:55:00.000Z")
}
```

**Index créés automatiquement :**
- Index unique sur `email`

## 🔒 Sécurité

- **Mots de passe** : Hachés avec **bcrypt** (coût par défaut)
- **JWT** : Tokens avec expiration de 24 heures
- **CORS** : Configuré pour autoriser uniquement les origines spécifiées
- **Validation** : Toutes les entrées sont validées
- **Email unique** : Vérifié à l'inscription avec index MongoDB

## 🛠️ Commandes Make

Le projet inclut un `Makefile` pour faciliter le développement :

```bash
# Démarrer le serveur
make run

# Compiler le projet
make build

# Installer les dépendances
make install

# Nettoyer les fichiers compilés
make clean

# Formater le code
make fmt

# Vérifier le code
make vet

# Nettoyer les dépendances
make tidy

# Afficher l'aide
make help
```

## 🧪 Tests avec curl

```bash
# Health check
curl http://localhost:8090/api/health

# Inscription
curl -X POST http://localhost:8090/api/inscription \
  -H "Content-Type: application/json" \
  -d '{
    "code_soiree": "CODE123",
    "prenom": "Jean",
    "nom": "Dupont",
    "email": "jean@email.com",
    "telephone": "0612345678",
    "password": "motdepasse123"
  }'

# Connexion
curl -X POST http://localhost:8090/api/connexion \
  -H "Content-Type: application/json" \
  -d '{
    "email": "jean@email.com",
    "password": "motdepasse123"
  }'

# Route protégée (remplacer TOKEN par le token reçu)
curl http://localhost:8090/api/protected/profile \
  -H "Authorization: Bearer TOKEN"
```

## 🌐 Utilisation avec Ngrok

Si vous utilisez ngrok pour exposer votre API :

```bash
# Démarrer ngrok sur le port 8090
ngrok http 8090
```

Votre API sera accessible via l'URL ngrok fournie (ex: `https://xxx.ngrok-free.dev`)

## 📦 Dépendances

- **gorilla/mux** : Router HTTP performant
- **mongo-driver** : Driver officiel MongoDB pour Go
- **golang-jwt/jwt** : Gestion des tokens JWT
- **golang.org/x/crypto** : Bcrypt pour le hachage des mots de passe
- **joho/godotenv** : Chargement des variables d'environnement

## 🐛 Résolution de problèmes

### Erreur 502 Bad Gateway

Si vous obtenez une erreur 502 :
1. Vérifiez que le serveur Go est bien démarré
2. Vérifiez que MongoDB tourne localement
3. Vérifiez les logs du serveur

### MongoDB connection refused

```bash
# Vérifier si MongoDB tourne
mongosh

# Démarrer MongoDB
brew services start mongodb-community
# ou
docker start mongodb
```

### CORS errors

Ajoutez l'origine de votre frontend dans `.env` :
```env
CORS_ALLOWED_ORIGINS=http://localhost:3000,https://votre-frontend.com
```

## 📝 Logs

Le serveur log automatiquement :
- ✓ Connexion MongoDB établie
- ✓ Index MongoDB créés
- 🚀 Démarrage du serveur
- 📋 Routes disponibles
- Toutes les requêtes HTTP (méthode, URI, status, durée)
- ✓ Inscriptions et connexions réussies
- ❌ Erreurs détaillées

## 🤝 Contribution

Code propre et bien structuré suivant les bonnes pratiques Go :
- Architecture en couches (handlers, database, models, utils)
- Séparation des responsabilités
- Gestion appropriée des erreurs
- Documentation des fonctions
- Logs informatifs avec emojis

## 📄 Licence

Projet privé - Tous droits réservés
