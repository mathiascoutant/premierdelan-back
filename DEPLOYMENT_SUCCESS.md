# ✅ Déploiement Railway Réussi !

## 🌐 Informations de Déploiement

**URL Backend:** `https://believable-spontaneity-production.up.railway.app`

**Base de données:** MongoDB (Railway)

**Environnement:** Production

---

## 📋 Endpoints Disponibles

### Public (sans authentification)

- `GET /api/health` - Health check
- `GET /api/evenements/public` - Liste des événements
- `GET /api/evenements/{id}` - Détails d'un événement
- `POST /api/inscription` - Inscription utilisateur
- `POST /api/connexion` - Connexion utilisateur
- `POST /api/alerts/critical` - Alertes critiques admin

### Authentifié (token requis)

- `GET /api/protected/profile` - Profil utilisateur
- `POST /api/evenements/{id}/inscription` - S'inscrire à un événement
- `GET /api/evenements/{id}/inscription` - Voir son inscription
- `PUT /api/evenements/{id}/inscription` - Modifier son inscription
- `DELETE /api/evenements/{id}/desinscription` - Se désinscrire
- `GET /api/mes-evenements` - Mes événements inscrits

### Admin (admin=1 requis)

- `GET /api/admin/utilisateurs` - Liste utilisateurs
- `PUT /api/admin/utilisateurs/{id}` - Modifier utilisateur
- `DELETE /api/admin/utilisateurs/{id}` - Supprimer utilisateur
- `GET /api/admin/evenements` - Liste événements (admin)
- `POST /api/admin/evenements` - Créer événement
- `PUT /api/admin/evenements/{id}` - Modifier événement
- `DELETE /api/admin/evenements/{id}` - Supprimer événement
- `GET /api/admin/evenements/{id}/inscrits` - Liste des inscrits
- `GET /api/admin/stats` - Statistiques globales
- `POST /api/admin/notifications/send` - Envoyer notification
- `GET /api/admin/codes-soiree` - Liste des codes
- Plus...

---

## 🔑 Variables d'Environnement (Railway)

Configurées dans Railway → believable-spontaneity → Variables :

| Variable                      | Valeur                             | Description          |
| ----------------------------- | ---------------------------------- | -------------------- |
| `MONGO_URI`                   | `mongodb://...`                    | Connexion MongoDB    |
| `MONGO_DB`                    | `premierdelan`                     | Nom de la base       |
| `JWT_SECRET`                  | `***`                              | Secret pour JWT      |
| `ENVIRONMENT`                 | `production`                       | Environnement        |
| `PORT`                        | `8090`                             | Port du serveur      |
| `CORS_ALLOWED_ORIGINS`        | `https://mathiascoutant.github.io` | CORS                 |
| `FCM_VAPID_KEY`               | `***`                              | Clé VAPID Firebase   |
| `FIREBASE_CREDENTIALS_BASE64` | (optionnel)                        | Credentials Firebase |

---

## 📅 Événement de Test Créé

Un événement de test a été créé dans la base de données :

```json
{
  "titre": "Réveillon 2026",
  "date": "2026-01-01T06:00:00",
  "description": "Soirée du nouvel an - Célébrez avec nous !",
  "capacite": 100,
  "inscrits": 0,
  "statut": "ouvert",
  "lieu": "Bordeaux",
  "code_soiree": "PREMIER2026"
}
```

---

## 🧪 Tests

### Test API Publique

```bash
curl https://believable-spontaneity-production.up.railway.app/api/evenements/public
```

### Test Health Check

```bash
curl https://believable-spontaneity-production.up.railway.app/api/health
```

### Test Inscription (depuis le frontend)

```javascript
const response = await fetch(
  "https://believable-spontaneity-production.up.railway.app/api/inscription",
  {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "ngrok-skip-browser-warning": "true",
    },
    body: JSON.stringify({
      code_soiree: "PREMIER2026",
      firstname: "Test",
      lastname: "User",
      email: "test@example.com",
      phone: "0612345678",
      password: "password123",
    }),
  }
);
```

---

## ⚠️ Notifications Firebase (Optionnel)

Les notifications push sont actuellement **désactivées** sur Railway car `FIREBASE_CREDENTIALS_BASE64` n'est pas configuré correctement.

**Le backend fonctionne sans Firebase**, seules les notifications push sont indisponibles.

### Pour activer Firebase plus tard :

1. Télécharge le fichier `firebase-service-account.json` depuis Firebase Console
2. Encode-le en base64 :
   ```bash
   base64 -i firebase-service-account.json | tr -d '\n'
   ```
3. Ajoute la variable `FIREBASE_CREDENTIALS_BASE64` dans Railway avec la valeur encodée
4. Railway redémarrera automatiquement le service

---

## 🔄 Mises à Jour

Pour mettre à jour le backend :

1. **Localement** :

   ```bash
   git add .
   git commit -m "Description des modifications"
   git push origin main
   ```

2. **Railway détecte automatiquement le push GitHub** et redéploie

3. **Vérifier les logs** dans Railway → believable-spontaneity → Deploy Logs

---

## 📊 Surveillance

### Logs en temps réel

Railway → believable-spontaneity → Deploy Logs

### Métriques

Railway → believable-spontaneity → Metrics

### Base de données

Railway → MongoDB → Data (pour voir les collections)

---

## 🎯 Prochaines Étapes

1. ✅ Backend déployé et fonctionnel
2. ✅ MongoDB connecté
3. ✅ Événement de test créé
4. 🔲 Tester l'inscription depuis le frontend
5. 🔲 Tester la connexion et les inscriptions aux événements
6. 🔲 (Optionnel) Configurer Firebase pour les notifications

---

## 📞 Support

En cas de problème :

1. **Vérifier les logs Railway** (Deploy Logs)
2. **Tester l'API avec curl** (voir section Tests)
3. **Vérifier MongoDB** (Railway → MongoDB → Data)
4. **Vérifier les variables d'environnement** (Railway → Variables)

---

## 🎉 Félicitations !

Ton backend Go est maintenant en ligne sur Railway avec MongoDB et prêt à être utilisé par ton frontend !

**Date de déploiement:** 2025-10-12

**Status:** ✅ Opérationnel
