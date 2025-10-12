# ✅ API Événements - PRÊT POUR LE FRONTEND

## 🎯 Résumé

Tous les endpoints événements sont **100% fonctionnels** et conformes à la documentation frontend.

---

## 📋 Endpoints Implémentés

### 1. ✅ GET `/api/admin/evenements`

**Description** : Liste tous les événements (triés par date décroissante)

**Authentification** : Admin requis (`admin: 1`)

**Headers requis** :

```
Authorization: Bearer <token>
ngrok-skip-browser-warning: true
```

**Réponse** :

```json
{
  "evenements": [
    {
      "id": "68e8bcf911798c8509c232c2",
      "titre": "Réveillon 2026",
      "date": "2025-12-31T20:00:00Z",
      "description": "Célébrez la nouvelle année",
      "capacite": 100,
      "inscrits": 0,
      "photos_count": 0,
      "statut": "ouvert",
      "lieu": "Villa Privée",
      "code_soiree": "NYE2026",
      "created_at": "2025-10-10T07:59:53.407Z",
      "updated_at": "2025-10-10T07:59:53.407Z"
    }
  ]
}
```

---

### 2. ✅ POST `/api/admin/evenements`

**Description** : Créer un nouvel événement

**Authentification** : Admin requis

**Body accepté (format frontend)** :

```json
{
  "titre": "Summer Vibes 2025",
  "date": "2025-07-15T19:00:00",
  "description": "Soirée d'été exclusive",
  "capacite": 80,
  "lieu": "Villa Privée",
  "code_soiree": "SUMMER25",
  "statut": "ouvert"
}
```

**⚠️ Important** : La date peut être au format :

- ✅ `"2025-07-15T19:00:00"` (datetime-local HTML5)
- ✅ `"2025-07-15T19:00:00Z"` (ISO 8601)

**Champs obligatoires** :

- `titre` (string)
- `code_soiree` (string)

**Champs optionnels** :

- `date` (datetime)
- `description` (string)
- `capacite` (int)
- `lieu` (string)
- `statut` (string: "ouvert", "complet", "annule", "termine")

**Valeurs par défaut** :

- `inscrits: 0`
- `photos_count: 0`
- `statut: "ouvert"` (si non fourni)

**Réponse (201 Created)** :

```json
{
  "success": true,
  "message": "Événement créé avec succès",
  "evenement": { ... }
}
```

---

### 3. ✅ PUT `/api/admin/evenements/{id}`

**Description** : Modifier un événement existant

**Authentification** : Admin requis

**Body** : Tous les champs sont optionnels (seuls les champs fournis seront mis à jour)

```json
{
  "titre": "Nouveau titre",
  "date": "2025-12-31T20:00:00",
  "description": "Nouvelle description",
  "capacite": 120,
  "lieu": "Nouveau lieu",
  "code_soiree": "NEWCODE",
  "statut": "complet"
}
```

**Réponse (200 OK)** :

```json
{
  "success": true,
  "message": "Événement modifié",
  "evenement": { ... }
}
```

**Erreurs possibles** :

- `400 Bad Request` : ID invalide ou aucune donnée à mettre à jour
- `404 Not Found` : Événement non trouvé
- `500 Internal Server Error` : Erreur serveur

---

### 4. ✅ DELETE `/api/admin/evenements/{id}`

**Description** : Supprimer un événement

**Authentification** : Admin requis

**Réponse (200 OK)** :

```json
{
  "success": true,
  "message": "Événement supprimé"
}
```

**Erreurs possibles** :

- `400 Bad Request` : ID invalide
- `500 Internal Server Error` : Erreur serveur

---

## 🎨 Statuts Disponibles

Les statuts suivants sont pris en charge :

- `"ouvert"` - Inscriptions ouvertes
- `"complet"` - Capacité maximale atteinte
- `"annule"` - Événement annulé
- `"termine"` - Événement terminé

---

## 📅 Format des Dates

### Frontend → Backend

Le backend accepte **automatiquement** les deux formats :

- ✅ `"2025-12-31T20:00:00"` (format HTML datetime-local)
- ✅ `"2025-12-31T20:00:00Z"` (format ISO 8601)
- ✅ `"2025-12-31T20:00:00+02:00"` (format ISO 8601 avec timezone)

### Backend → Frontend

Le backend retourne **toujours** au format **ISO 8601** :

```
"2025-12-31T20:00:00Z"
```

---

## 🔒 Sécurité

- ✅ Tous les endpoints nécessitent un **token JWT valide**
- ✅ Tous les endpoints nécessitent un **utilisateur admin** (`admin: 1`)
- ✅ Middleware `RequireAdmin` vérifie les droits à chaque requête

---

## 🧪 Tests

### Créer un événement

```bash
curl -X POST http://localhost:8090/api/admin/evenements \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{
    "titre": "Réveillon 2026",
    "date": "2025-12-31T20:00:00",
    "description": "Célébrez la nouvelle année",
    "capacite": 100,
    "lieu": "Villa Privée",
    "code_soiree": "NYE2026",
    "statut": "ouvert"
  }'
```

### Lister les événements

```bash
curl http://localhost:8090/api/admin/evenements \
  -H "Authorization: Bearer <token>" \
  -H "ngrok-skip-browser-warning: true"
```

### Modifier un événement

```bash
curl -X PUT http://localhost:8090/api/admin/evenements/{id} \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{
    "titre": "Nouveau titre",
    "statut": "complet"
  }'
```

### Supprimer un événement

```bash
curl -X DELETE http://localhost:8090/api/admin/evenements/{id} \
  -H "Authorization: Bearer <token>"
```

---

## ✅ Checklist Frontend

Le backend est prêt pour :

- ✅ Afficher la liste des événements
- ✅ Créer un nouvel événement via modale
- ✅ Modifier un événement existant
- ✅ Supprimer un événement avec confirmation
- ✅ Gérer tous les statuts (ouvert, complet, annulé, terminé)
- ✅ Accepter le format de date HTML5 datetime-local
- ✅ Retourner les dates au format ISO 8601

---

## 📊 Base de Données

**Collection** : `events`

**Champs** :

- `_id` : ObjectID MongoDB
- `titre` : string
- `date` : Date
- `description` : string
- `capacite` : int
- `inscrits` : int (géré automatiquement)
- `photos_count` : int (géré automatiquement)
- `statut` : string
- `lieu` : string
- `code_soiree` : string
- `created_at` : Date
- `updated_at` : Date

---

## 🚀 Prêt pour la Production

Ton frontend peut maintenant :

1. Se connecter en tant qu'admin
2. Afficher tous les événements
3. Créer, modifier et supprimer des événements
4. Gérer les statuts
5. Utiliser le format de date natif des inputs HTML5

**Tout est fonctionnel ! 🎉**
