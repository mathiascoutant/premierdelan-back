# 🔔 Fonctionnement des Notifications Push - Persistance des Tokens

## ❓ Question

**Scénario** : Un utilisateur se connecte, active les notifications, se déconnecte puis se reconnecte. Est-ce que les notifications continueront de fonctionner ?

## ✅ Réponse : OUI, les notifications fonctionneront !

### 🔍 Explication Technique

#### 1. **Le Token FCM est Persistant**

Quand un utilisateur active les notifications :

```javascript
// Frontend : Demande de permission et récupération du token
const messaging = getMessaging();
const token = await getToken(messaging);

// Envoi au backend
await fetch("/api/fcm/subscribe", {
  method: "POST",
  body: JSON.stringify({
    user_id: userEmail,
    fcm_token: token,
    device: "web",
  }),
});
```

Ce token FCM est :

- ✅ **Stocké dans MongoDB** (collection `fcm_tokens`)
- ✅ **Lié à l'utilisateur** via `user_id` (email)
- ✅ **Persistant entre les sessions**
- ✅ **Valide même après déconnexion**

#### 2. **Stockage Backend**

```go
type FCMToken struct {
    ID        primitive.ObjectID `bson:"_id,omitempty"`
    UserID    string             `bson:"user_id"`     // Email utilisateur
    Token     string             `bson:"token"`       // Token FCM
    Device    string             `bson:"device"`      // Type d'appareil
    CreatedAt time.Time          `bson:"created_at"`
    UpdatedAt time.Time          `bson:"updated_at"`
}
```

Le token est stocké dans la base de données MongoDB et **reste valide indépendamment de l'état de connexion** de l'utilisateur.

#### 3. **Comment ça Marche**

```
┌─────────────────────────────────────────────────────────────┐
│  Cycle de Vie d'un Token FCM                                │
└─────────────────────────────────────────────────────────────┘

1️⃣  PREMIÈRE CONNEXION + ACTIVATION
    ├─ Utilisateur se connecte (email: user@example.com)
    ├─ Clique sur "Activer les notifications"
    ├─ Navigateur génère un Token FCM unique
    ├─ Token envoyé au backend via POST /api/fcm/subscribe
    └─ Token stocké dans MongoDB:
        {
          user_id: "user@example.com",
          token: "eX4mpl3T0k3n...",
          device: "web",
          created_at: "2025-01-01T10:00:00Z"
        }

2️⃣  DÉCONNEXION
    ├─ Utilisateur se déconnecte
    ├─ Session frontend effacée (localStorage, cookies)
    └─ ✅ Token FCM RESTE dans MongoDB (pas supprimé)

3️⃣  RECONNEXION
    ├─ Utilisateur se reconnecte (email: user@example.com)
    └─ ✅ Token FCM toujours présent dans MongoDB

4️⃣  ENVOI DE NOTIFICATION
    ├─ Admin envoie une notification
    ├─ Backend récupère les tokens de "user@example.com"
    ├─ Trouve le token stocké: "eX4mpl3T0k3n..."
    └─ ✅ Notification envoyée avec succès !
```

## 🎯 Cas d'Usage Réels

### ✅ Cas où les Notifications Fonctionnent

| Scénario                            | Résultat                                               |
| ----------------------------------- | ------------------------------------------------------ |
| 🔄 **Déconnexion/Reconnexion**      | ✅ Notifications fonctionnent                          |
| 💻 **Fermer/Rouvrir le navigateur** | ✅ Notifications fonctionnent                          |
| 🌐 **Changer d'onglet**             | ✅ Notifications fonctionnent                          |
| 📱 **Appareil en veille**           | ✅ Notifications fonctionnent                          |
| 🔋 **Redémarrage de l'appareil**    | ✅ Notifications fonctionnent (si navigateur autorisé) |

### ❌ Cas où les Notifications NE Fonctionnent PAS

| Scénario                         | Raison                                 | Solution                                          |
| -------------------------------- | -------------------------------------- | ------------------------------------------------- |
| 🗑️ **Cache navigateur vidé**     | Token FCM supprimé localement          | Réactiver les notifications                       |
| 🚫 **Permissions révoquées**     | Utilisateur a bloqué les notifications | Réaccorder les permissions                        |
| 🔄 **Token expiré/invalide**     | Firebase a invalidé le token           | Backend le détecte et le supprime automatiquement |
| 🌍 **Autre navigateur/appareil** | Token lié au navigateur/appareil       | Activer les notifications sur le nouvel appareil  |

## 🔐 Sécurité et Gestion des Tokens

### Auto-Nettoyage des Tokens Invalides

Le backend nettoie automatiquement les tokens invalides :

```go
// handlers/fcm_handler.go : Ligne 148-154
// Supprimer les tokens invalides après l'envoi
for _, failedToken := range failedTokens {
    if err := h.tokenRepo.Delete(failedToken); err != nil {
        log.Printf("⚠️ Erreur suppression token invalide: %v", err)
    } else {
        log.Printf("🗑️ Token invalide supprimé: %s...", failedToken[:20])
    }
}
```

### Multi-Appareils

Un utilisateur peut avoir **plusieurs tokens FCM** (un par appareil/navigateur) :

```javascript
// Utilisateur connecté sur 3 appareils
MongoDB fcm_tokens:
[
  { user_id: "user@example.com", token: "token_pc_chrome", device: "web" },
  { user_id: "user@example.com", token: "token_mobile_android", device: "android" },
  { user_id: "user@example.com", token: "token_iphone_safari", device: "ios" }
]

// Envoi notification → 3 notifications envoyées (une par appareil)
```

## 📊 Flux Complet

```
┌──────────────────────────────────────────────────────────────┐
│  FRONTEND                                                    │
└──────────────────────────────────────────────────────────────┘
                         │
                         │ 1️⃣ Utilisateur active notifications
                         ▼
            ┌────────────────────────┐
            │  Firebase génère Token │
            └────────────────────────┘
                         │
                         │ 2️⃣ POST /api/fcm/subscribe
                         ▼
┌──────────────────────────────────────────────────────────────┐
│  BACKEND (handlers/fcm_handler.go)                          │
├──────────────────────────────────────────────────────────────┤
│  ✓ Reçoit le token                                          │
│  ✓ Appelle tokenRepo.Upsert(token)                          │
└──────────────────────────────────────────────────────────────┘
                         │
                         │ 3️⃣ Stockage persistant
                         ▼
┌──────────────────────────────────────────────────────────────┐
│  MONGODB (collection fcm_tokens)                            │
├──────────────────────────────────────────────────────────────┤
│  {                                                           │
│    _id: ObjectId("..."),                                     │
│    user_id: "user@example.com",                              │
│    token: "eX4mpl3T0k3n...",                                 │
│    device: "web",                                            │
│    created_at: ISODate("2025-01-01T10:00:00Z"),             │
│    updated_at: ISODate("2025-01-01T10:00:00Z")              │
│  }                                                           │
└──────────────────────────────────────────────────────────────┘
                         │
                         │ ⏰ Plus tard...
                         │ 4️⃣ Admin envoie une notification
                         ▼
┌──────────────────────────────────────────────────────────────┐
│  BACKEND                                                     │
├──────────────────────────────────────────────────────────────┤
│  ✓ Récupère tokens de user@example.com                      │
│  ✓ Trouve le token stocké                                   │
│  ✓ Appelle Firebase Cloud Messaging                         │
└──────────────────────────────────────────────────────────────┘
                         │
                         │ 5️⃣ Notification envoyée
                         ▼
┌──────────────────────────────────────────────────────────────┐
│  APPAREIL UTILISATEUR                                        │
├──────────────────────────────────────────────────────────────┤
│  🔔 Notification affichée !                                  │
│  (même si utilisateur déconnecté)                            │
└──────────────────────────────────────────────────────────────┘
```

## 🧪 Comment Tester

### Test 1 : Déconnexion/Reconnexion

```bash
1. Connectez-vous avec un compte
2. Activez les notifications
3. Vérifiez dans MongoDB que le token est stocké:
   db.fcm_tokens.find({ user_id: "votre@email.com" })
4. Déconnectez-vous
5. Vérifiez que le token est toujours dans MongoDB (il doit l'être)
6. Reconnectez-vous
7. Envoyez une notification test depuis l'admin
8. ✅ La notification doit arriver !
```

### Test 2 : Multi-Appareils

```bash
1. Connectez-vous sur Chrome
2. Activez les notifications
3. Connectez-vous sur Firefox (même compte)
4. Activez les notifications
5. Vérifiez MongoDB:
   db.fcm_tokens.find({ user_id: "votre@email.com" })
   → Devrait montrer 2 tokens
6. Envoyez une notification
7. ✅ Vous devriez recevoir 2 notifications (une par navigateur)
```

## 🔧 Gestion Frontend (Recommandations)

### Au Chargement de l'App

```javascript
// Vérifier si l'utilisateur a déjà activé les notifications
useEffect(() => {
  const checkNotificationStatus = async () => {
    // Vérifier la permission du navigateur
    const permission = Notification.permission;

    if (permission === "granted") {
      // L'utilisateur a déjà accepté les notifications
      // Le token existe déjà dans MongoDB
      console.log("✅ Notifications déjà activées");
    } else if (permission === "denied") {
      // L'utilisateur a bloqué les notifications
      console.log("❌ Notifications bloquées");
    } else {
      // Jamais demandé
      console.log("⚠️ Notifications pas encore demandées");
    }
  };

  if (user) {
    checkNotificationStatus();
  }
}, [user]);
```

### Après Connexion

```javascript
// Pas besoin de redemander le token à chaque connexion !
// Le token est déjà stocké côté backend

// ⚠️ NE PAS FAIRE :
// await activerNotifications(); // À chaque connexion

// ✅ FAIRE :
// Laisser l'utilisateur activer manuellement via un bouton
// Le token persiste automatiquement
```

## 📝 Résumé

| Question                                                           | Réponse                                                            |
| ------------------------------------------------------------------ | ------------------------------------------------------------------ |
| **Les notifications fonctionnent après déconnexion/reconnexion ?** | ✅ **OUI** - Le token reste dans MongoDB                           |
| **Les notifications fonctionnent après fermeture du navigateur ?** | ✅ **OUI** - Le token persiste                                     |
| **Les notifications fonctionnent sur plusieurs appareils ?**       | ✅ **OUI** - Un token par appareil/navigateur                      |
| **Faut-il réactiver les notifications à chaque connexion ?**       | ❌ **NON** - Une seule activation suffit                           |
| **Le token expire-t-il ?**                                         | ⚠️ **Rarement** - Firebase peut l'invalider (géré automatiquement) |

## 🎯 Conclusion

**Votre système de notifications est bien conçu !**

- ✅ Les tokens sont **persistants** et stockés côté serveur
- ✅ Les notifications fonctionnent **même après déconnexion**
- ✅ Le système **nettoie automatiquement** les tokens invalides
- ✅ Support **multi-appareils** intégré
- ✅ Pas besoin de **réactiver à chaque connexion**

L'utilisateur n'a besoin d'activer les notifications **qu'une seule fois** par navigateur/appareil, et elles fonctionneront indéfiniment (sauf si permission révoquée ou cache vidé).
