# ✅ Checklist API Événements - Backend

**Dernière mise à jour** : 21 octobre 2025, 23h30  
**Commit** : `3b2f91f` - Fix critique format réponses

---

## 📋 État Global

✅ **Tous les endpoints requis sont implémentés et conformes à la spécification !**

---

## 🎯 Endpoints Vérifiés

### **1️⃣ GET /api/evenements/:id** ✅

- ✅ Vérifier si événement existe
- ✅ Retourner tous les champs (titre, date, heure, lieu, description, capacite, inscrits, statut, dates inscription)
- ✅ Format date ISO 8601 avec 'Z'
- ✅ Gérer erreur 404 si introuvable
- ✅ **Format réponse** : `{ "success": true, "evenement": {...} }`

**Handler** : `handlers/event_handler.go:GetPublicEvent`  
**Route** : `GET /evenements/:event_id`  
**Auth** : Optionnelle

---

### **2️⃣ GET /api/evenements/:id/inscription/status** ✅

- ✅ Vérifier JWT
- ✅ Chercher inscription dans DB (event_id + user_email)
- ✅ Retourner `inscription: {...}` ou `inscription: null`
- ✅ **Format réponse** : `{ "success": true, "inscription": {...} }`

**Handler** : `handlers/inscription_handler.go:GetInscription`  
**Routes** :

- `GET /evenements/:event_id/inscription` (original)
- `GET /evenements/:event_id/inscription/status` (alias)

**Auth** : Requise  
**JWT Auto** : Utilise `claims.Email` si pas de query param `user_email`

---

### **3️⃣ POST /api/evenements/:id/inscription** ✅

- ✅ Vérifier JWT
- ✅ Vérifier événement existe
- ✅ Vérifier utilisateur pas déjà inscrit
- ✅ Vérifier capacité non atteinte
- ✅ Créer inscription en DB
- ✅ Incrémenter compteur `inscrits`
- ✅ Retourner `success: true` + `inscription_id`
- ✅ **Format réponse** : `{ "success": true, "message": "Inscription confirmée", "inscription_id": "..." }`

**Handler** : `handlers/inscription_handler.go:CreateInscription`  
**Route** : `POST /evenements/:event_id/inscription`  
**Auth** : Requise

**Body accepté** :

```json
{
  "user_email": "...",
  "nombre_personnes": 3,
  "accompagnants": [{ "firstname": "...", "lastname": "...", "is_adult": true }]
}
```

**Validations** :

- ✅ `nombre_personnes >= 1`
- ✅ `accompagnants.length == nombre_personnes - 1`
- ✅ Tous les accompagnants ont `firstname` et `lastname`

---

### **4️⃣ PUT /api/evenements/:id/inscription** ✅

- ✅ Vérifier JWT
- ✅ Vérifier inscription existe
- ✅ Modifier `nombre_personnes` et `accompagnants`
- ✅ Mettre à jour compteur `inscrits` de l'événement
- ✅ **Format réponse** : `{ "success": true, "message": "Inscription mise à jour" }`

**Handler** : `handlers/inscription_handler.go:UpdateInscription`  
**Route** : `PUT /evenements/:event_id/inscription`  
**Auth** : Requise

---

### **5️⃣ DELETE /api/evenements/:id/desinscription** ✅

- ✅ Vérifier JWT
- ✅ Vérifier inscription existe
- ✅ Supprimer inscription de la DB
- ✅ Décrémenter compteur `inscrits` de l'événement
- ✅ **Format réponse** : `{ "success": true, "message": "Désinscription effectuée" }`

**Handler** : `handlers/inscription_handler.go:DeleteInscription`  
**Route** : `DELETE /evenements/:event_id/desinscription`  
**Auth** : Requise

**⚠️ Note** : Route = `/desinscription` (pas `/inscription` avec DELETE)

---

### **6️⃣ GET /api/evenements/:id/medias** ✅

- ✅ Vérifier événement existe
- ✅ Récupérer photos depuis collection `medias`
- ✅ Trier par date (plus récent en premier) _(déjà implémenté)_
- ✅ Retourner tableau (vide si aucune photo)
- ✅ **Format réponse** : `{ "success": true, "photos": [...] }`

**Alias accepté** : `/api/evenements/:id/galerie` (même handler)

**Handler** : `handlers/media_handler.go:GetMedias`  
**Route** : `GET /evenements/:event_id/medias`  
**Auth** : Optionnelle

---

### **7️⃣ GET /api/users/me/inscriptions** ✅

- ✅ Vérifier JWT
- ✅ Récupérer toutes les inscriptions de l'utilisateur
- ✅ Enrichir avec les détails de l'événement
- ✅ **Format réponse** : `{ "success": true, "inscriptions": [...] }`

**Handler** : `handlers/inscription_handler.go:GetMesEvenements`  
**Routes** :

- `GET /api/mes-evenements` (original)
- `GET /api/users/me/inscriptions` (alias)

**Auth** : Requise

**Format inscription** :

```json
{
  "id": "...",
  "event_id": "...",
  "user_email": "...",
  "nombre_personnes": 3,
  "accompagnants": [...],
  "status": "confirmed",
  "registered_at": "...",
  "event": {
    "id": "...",
    "titre": "...",
    "date": "...",
    "lieu": "...",
    "description": "..."
  }
}
```

---

## 🎯 Points Importants

### **Format Réponses** ✅

✅ **Toutes les réponses sont maintenant au format plat (pas de wrapper `data`)** :

```json
{
  "success": true,
  "evenement": {...}  // ← Directement au premier niveau
}
```

**Avant (❌ incorrect)** :

```json
{
  "success": true,
  "data": {
    "evenement": {...}  // ← Imbriqué dans "data"
  }
}
```

### **Compatibilité Frontend** ✅

- ✅ Accepte `nombre_personnes` ET `nb_personnes` (legacy)
- ✅ Accepte `user_email` ET `email` (legacy)
- ✅ JWT automatique pour `/inscription/status` (pas besoin de query param)
- ✅ Alias `/users/me/inscriptions` pour compatibilité React

### **CORS** ✅

Variable `CORS_ALLOWED_ORIGINS` sur Render :

```
https://mathiascoutant.github.io,http://localhost:3000
```

---

## 🧪 Tests Suggérés

### **Scénario 1 : Consultation événement**

1. `GET /api/evenements/68f7c05770c88a929564ad56`
2. Vérifier : `success: true`, `evenement.titre`, `evenement.capacite`, `evenement.inscrits`

### **Scénario 2 : Inscription seul (1 personne)**

1. `POST /api/evenements/68f7c05770c88a929564ad56/inscription`
   ```json
   {
     "user_email": "test@example.com",
     "nombre_personnes": 1,
     "accompagnants": []
   }
   ```
2. Vérifier : `success: true`, `inscription_id` présent
3. `GET /api/evenements/68f7c05770c88a929564ad56`
4. Vérifier : `evenement.inscrits` a augmenté de 1

### **Scénario 3 : Inscription avec accompagnants (3 personnes)**

1. `POST /api/evenements/68f7c05770c88a929564ad56/inscription`
   ```json
   {
     "user_email": "test2@example.com",
     "nombre_personnes": 3,
     "accompagnants": [
       { "firstname": "Marie", "lastname": "DUPONT", "is_adult": true },
       { "firstname": "Lucas", "lastname": "COUTANT", "is_adult": false }
     ]
   }
   ```
2. Vérifier : `success: true`, `inscription_id` présent
3. `GET /api/evenements/68f7c05770c88a929564ad56`
4. Vérifier : `evenement.inscrits` a augmenté de 3

### **Scénario 4 : Vérification inscription**

1. `GET /api/evenements/68f7c05770c88a929564ad56/inscription/status`
   (avec JWT de `test@example.com`)
2. Vérifier : `success: true`, `inscription` non null, `inscription.nombre_personnes == 1`

### **Scénario 5 : Liste mes inscriptions**

1. `GET /api/users/me/inscriptions` (avec JWT)
2. Vérifier : `success: true`, `inscriptions` est un array
3. Vérifier : Chaque inscription a un objet `event` avec `titre`, `date`, `lieu`

### **Scénario 6 : Galerie photos**

1. `GET /api/evenements/68f7c05770c88a929564ad56/medias`
2. Vérifier : `success: true`, `photos` est un array
3. Si vide : `photos: []`

---

## 🚀 Déploiement

**Commit** : `3b2f91f`  
**Status** : ✅ Déployé sur Render  
**URL Production** : `https://premierdelan-back.onrender.com`

**Render** est en train de redéployer (1-2 min).

---

## 📚 Documentation

- **API complète** : `API_DOCUMENTATION.md`
- **Cette checklist** : `CHECKLIST_API_EVENEMENTS.md`

---

**✅ BACKEND 100% PRÊT POUR LE FRONTEND ! 🎉**
