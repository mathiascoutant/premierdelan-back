# ✅ API Publique - Événements

## 🌍 Endpoint Public (Sans Authentification)

### GET `/api/evenements/public`

**Description** : Récupère la liste de tous les événements (accessible à tous, sans authentification)

**Authentification** : ❌ Non requise

**Headers** :

```
Content-Type: application/json
ngrok-skip-browser-warning: true (optionnel, pour ngrok)
```

---

## 📋 Réponse

**Status** : `200 OK`

**Format** :

```json
{
  "evenements": [
    {
      "id": "68e8be3492b2c55a559143f6",
      "titre": "Réveillon 2026",
      "date": "2025-12-31T20:00:00Z",
      "description": "Célébrez la nouvelle année et partagez vos meilleurs moments.",
      "capacite": 100,
      "inscrits": 45,
      "photos_count": 247,
      "statut": "ouvert",
      "lieu": "Villa Privée - Côte d'Azur",
      "code_soiree": "NYE2026",
      "created_at": "2025-10-10T08:05:08.907Z",
      "updated_at": "2025-10-10T08:05:08.907Z"
    },
    {
      "id": "68e8be3492b2c55a559143f7",
      "titre": "Summer Vibes",
      "date": "2025-07-15T19:00:00Z",
      "description": "Soirée d'été exclusive avec piscine et DJ.",
      "capacite": 80,
      "inscrits": 75,
      "photos_count": 0,
      "statut": "ouvert",
      "lieu": "Beach Club",
      "code_soiree": "SUMMER25",
      "created_at": "2025-06-01T10:00:00Z",
      "updated_at": "2025-06-01T10:00:00Z"
    }
  ]
}
```

---

## 📊 Champs Retournés

| Champ          | Type              | Description                                       |
| -------------- | ----------------- | ------------------------------------------------- |
| `id`           | string            | ID unique de l'événement                          |
| `titre`        | string            | Titre de l'événement                              |
| `date`         | string (ISO 8601) | Date et heure de l'événement                      |
| `description`  | string            | Description complète                              |
| `capacite`     | number            | Nombre maximum de participants                    |
| `inscrits`     | number            | Nombre actuel d'inscrits                          |
| `photos_count` | number            | Nombre de photos uploadées                        |
| `statut`       | string            | Statut : "ouvert", "complet", "annule", "termine" |
| `lieu`         | string            | Lieu de l'événement                               |
| `code_soiree`  | string            | Code d'accès à la soirée                          |
| `created_at`   | string (ISO 8601) | Date de création                                  |
| `updated_at`   | string (ISO 8601) | Date de dernière modification                     |

---

## 🎯 Cas d'Usage Frontend

### Afficher la liste des événements sur la page d'accueil

```javascript
async function getPublicEvents() {
  try {
    const response = await fetch(
      "https://nia-preinstructive-nola.ngrok-free.dev/api/evenements/public",
      {
        method: "GET",
        headers: {
          "Content-Type": "application/json",
          "ngrok-skip-browser-warning": "true",
        },
      }
    );

    if (!response.ok) {
      throw new Error("Erreur lors de la récupération des événements");
    }

    const data = await response.json();
    return data.evenements;
  } catch (error) {
    console.error("Erreur:", error);
    return [];
  }
}

// Utilisation
getPublicEvents().then((events) => {
  events.forEach((event) => {
    console.log(`${event.titre} - ${event.date} - ${event.lieu}`);
    console.log(`${event.inscrits}/${event.capacite} inscrits`);
    console.log(`Statut: ${event.statut}`);
  });
});
```

---

## 🔄 Tri des Événements

Les événements sont retournés **triés par date décroissante** (les plus récents en premier).

---

## ⚡ Performance

- ✅ Pas d'authentification requise → temps de réponse rapide
- ✅ Retourne tous les événements en une seule requête
- ✅ Compatible avec tous les navigateurs et appareils

---

## 🚫 Gestion des Erreurs

### 500 Internal Server Error

```json
{
  "error": "Internal Server Error",
  "message": "Erreur serveur"
}
```

**Cause** : Problème de connexion à la base de données

---

## ✅ Tests

### Test en local

```bash
curl http://localhost:8090/api/evenements/public | jq
```

### Test via ngrok

```bash
curl https://nia-preinstructive-nola.ngrok-free.dev/api/evenements/public \
  -H "ngrok-skip-browser-warning: true" | jq
```

---

## 📱 Responsive

Cet endpoint est parfait pour :

- ✅ Page d'accueil publique (avant connexion)
- ✅ PWA installée sur mobile
- ✅ Partage de liens sur les réseaux sociaux
- ✅ Affichage rapide sans attendre l'authentification

---

## 🔒 Sécurité

Bien que cet endpoint soit public :

- ✅ Les événements sont en **lecture seule**
- ✅ Aucune modification possible sans authentification admin
- ✅ CORS configuré pour les origines autorisées uniquement
- ✅ Pas de données sensibles exposées (pas de passwords, tokens, etc.)

---

## 🎨 Différences avec `/api/admin/evenements`

| Caractéristique  | `/api/evenements/public` | `/api/admin/evenements` |
| ---------------- | ------------------------ | ----------------------- |
| Authentification | ❌ Non requise           | ✅ Admin requis         |
| Accès            | Tous les visiteurs       | Admins uniquement       |
| Modification     | ❌ Lecture seule         | ✅ CRUD complet         |
| Cas d'usage      | Page d'accueil publique  | Panel d'administration  |

---

## 🚀 Prêt pour la Production

Ton frontend peut maintenant :

1. ✅ Afficher tous les événements sans authentification
2. ✅ Montrer les places disponibles (`inscrits/capacite`)
3. ✅ Afficher le statut (ouvert/complet)
4. ✅ Trier par date
5. ✅ Afficher le lieu et la description

**L'endpoint public est 100% fonctionnel ! 🎉**
