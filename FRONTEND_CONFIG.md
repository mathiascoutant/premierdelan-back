# 🚀 Configuration Frontend - Backend Railway

## 📡 URL de l'API

**URL Permanente Railway** :

```
https://believable-spontaneity-production.up.railway.app
```

**Remplace toutes les occurrences de ngrok** par cette URL dans ton code frontend.

---

## 🔧 Configuration Frontend

### Dans ton fichier de configuration (config.js ou .env) :

```javascript
// URL de base de l'API
const API_BASE_URL = "https://believable-spontaneity-production.up.railway.app/api";

// Ou en .env
VITE_API_URL=https://believable-spontaneity-production.up.railway.app/api
REACT_APP_API_URL=https://believable-spontaneity-production.up.railway.app/api
```

---

## 📋 Tous les Endpoints Disponibles

### Public (sans authentification)

```javascript
// Health check
GET ${API_BASE_URL}/health

// Événements
GET ${API_BASE_URL}/evenements/public
GET ${API_BASE_URL}/evenements/{event_id}

// Médias (galerie)
GET ${API_BASE_URL}/evenements/{event_id}/medias

// Inscription / Connexion
POST ${API_BASE_URL}/inscription
POST ${API_BASE_URL}/connexion

// Alertes critiques
POST ${API_BASE_URL}/alerts/critical
```

### Authentifié (header Authorization requis)

```javascript
// Mes événements
GET ${API_BASE_URL}/mes-evenements

// Inscriptions
POST ${API_BASE_URL}/evenements/{event_id}/inscription
GET ${API_BASE_URL}/evenements/{event_id}/inscription
PUT ${API_BASE_URL}/evenements/{event_id}/inscription
DELETE ${API_BASE_URL}/evenements/{event_id}/desinscription

// Médias
POST ${API_BASE_URL}/evenements/{event_id}/medias
DELETE ${API_BASE_URL}/evenements/{event_id}/medias/{media_id}

// FCM (notifications)
POST ${API_BASE_URL}/fcm/subscribe
GET ${API_BASE_URL}/fcm/vapid-key
```

### Admin (admin=1 requis)

```javascript
// Utilisateurs
GET ${API_BASE_URL}/admin/utilisateurs
PUT ${API_BASE_URL}/admin/utilisateurs/{id}
DELETE ${API_BASE_URL}/admin/utilisateurs/{id}

// Événements
GET ${API_BASE_URL}/admin/evenements
POST ${API_BASE_URL}/admin/evenements
GET ${API_BASE_URL}/admin/evenements/{id}
PUT ${API_BASE_URL}/admin/evenements/{id}
DELETE ${API_BASE_URL}/admin/evenements/{id}

// Inscrits
GET ${API_BASE_URL}/admin/evenements/{id}/inscrits
DELETE ${API_BASE_URL}/admin/evenements/{id}/inscrits/{inscription_id}
DELETE ${API_BASE_URL}/admin/evenements/{id}/inscrits/{inscription_id}/accompagnant/{index}

// Stats et notifications
GET ${API_BASE_URL}/admin/stats
POST ${API_BASE_URL}/admin/notifications/send

// Codes soirée
GET ${API_BASE_URL}/admin/codes-soiree
POST ${API_BASE_URL}/admin/code-soiree/generate
GET ${API_BASE_URL}/admin/code-soiree/current
```

---

## 🔑 Headers Requis

### Pour toutes les requêtes

```javascript
headers: {
  'Content-Type': 'application/json',
  'ngrok-skip-browser-warning': 'true' // Optionnel pour Railway mais garde-le
}
```

### Pour routes authentifiées

```javascript
headers: {
  'Content-Type': 'application/json',
  'Authorization': `Bearer ${token}` // Token JWT reçu à la connexion
}
```

---

## 📊 Format des Données

### Inscription

```javascript
POST /api/inscription

Body:
{
  "code_soiree": "PREMIER2026",
  "firstname": "Jean",
  "lastname": "Dupont",
  "email": "jean@email.com",
  "phone": "0612345678",
  "password": "password123"
}

Response:
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "user": {
    "id": "...",
    "email": "jean@email.com",
    "firstname": "Jean",
    "lastname": "Dupont",
    "phone": "0612345678",
    "admin": 0
  }
}
```

### Connexion

```javascript
POST /api/connexion

Body:
{
  "email": "jean@email.com",
  "password": "password123"
}

Response:
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "user": {
    "id": "...",
    "email": "jean@email.com",
    "firstname": "Jean",
    "lastname": "Dupont",
    "admin": 0
  }
}
```

### Créer Événement (Admin)

```javascript
POST /api/admin/evenements
Authorization: Bearer <token>

Body:
{
  "titre": "Soirée 2026",
  "date": "2026-01-20T20:00", // Format: YYYY-MM-DDTHH:MM (heure française)
  "description": "Description",
  "capacite": 100,
  "lieu": "Paris",
  "code_soiree": "CODE2026",
  "statut": "ouvert",
  "date_ouverture_inscription": "2025-12-01T10:00", // Optionnel
  "date_fermeture_inscription": "2026-01-19T23:59"  // Optionnel
}
```

---

## ⚠️ Important : Timezone

**Toutes les dates sont en HEURE FRANÇAISE** :

- Envoie : `"2025-12-31T20:00"` (SANS le Z)
- Le backend stocke et retourne en heure française
- Pas de conversion UTC côté frontend

---

## 🔐 Compte Admin Initial

```
Email : mathiascoutant@icloud.com
Password : test1234
Code soirée : PREMIER2026
```

**Après inscription, passe-toi en admin** :

- Dans Railway MongoDB → Collection `users`
- Trouve ton user et mets `admin: 1`

---

## 📱 Notifications (Désactivées Temporairement)

⚠️ **Firebase est désactivé sur Railway** (pour l'instant)

- Les notifications push ne fonctionneront pas
- Toutes les autres fonctionnalités marchent
- On activera Firebase plus tard si besoin

---

## 🎯 Résumé pour le Développeur Frontend

**Change uniquement l'URL** :

```javascript
const API_URL = "https://believable-spontaneity-production.up.railway.app/api";
```

**Tout le reste reste identique !**

- Mêmes endpoints
- Mêmes headers
- Même format de données
- Même système d'authentification JWT

---

## ✅ Backend : Tout est Prêt !

Rien à changer côté backend ! Tout fonctionne :

- ✅ API REST complète
- ✅ MongoDB connecté
- ✅ JWT authentification
- ✅ CORS configuré
- ✅ Tous les endpoints opérationnels
- ⚠️ Notifications FCM désactivées (temporaire)

**Ton backend est maintenant en production 24/7 ! 🎉**
