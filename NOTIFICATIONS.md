# 🔔 Guide des Notifications PWA

## Configuration

Les notifications PWA sont maintenant configurées ! Les clés VAPID ont été générées et ajoutées au fichier `.env`.

## Endpoints disponibles

### 1. Récupérer la clé publique VAPID (Public)

```
GET /api/notifications/vapid-public-key
```

**Réponse :**

```json
{
  "publicKey": "BA4u1XlrejmjKKKAYuO7JKAzqFc2h3I2RHyiRT2Uet-tIXi0_0NCdFDGDSsKJKNGDBu5ekCzu6mzl1965y0KhAA"
}
```

### 2. S'abonner aux notifications (Public)

```
POST /api/notifications/subscribe
Content-Type: application/json
```

**Body :**

```json
{
  "user_id": "user@email.com",
  "subscription": {
    "endpoint": "https://fcm.googleapis.com/fcm/send/...",
    "keys": {
      "p256dh": "clé_publique_du_navigateur",
      "auth": "clé_auth_du_navigateur"
    }
  }
}
```

### 3. Se désabonner (Public)

```
POST /api/notifications/unsubscribe
Content-Type: application/json
```

**Body :**

```json
{
  "endpoint": "https://fcm.googleapis.com/fcm/send/..."
}
```

### 4. Envoyer une notification test (Protégé) ⭐

```
POST /api/notification/test
Authorization: Bearer <votre_token>
Content-Type: application/json
```

**Body :**

```json
{
  "user_id": "email@utilisateur.com",
  "title": "Titre de la notification",
  "message": "Message de la notification",
  "data": {
    "custom": "data"
  }
}
```

**Réponse :**

```json
{
  "message": "Notifications envoyées",
  "data": {
    "sent": 5,
    "failed": 0,
    "total": 5
  }
}
```

## Code Frontend pour s'abonner

### 1. Enregistrer le Service Worker

Créez un fichier `service-worker.js` à la racine de votre frontend :

```javascript
// service-worker.js
self.addEventListener("push", function (event) {
  const data = event.data ? event.data.json() : {};

  const options = {
    body: data.body || "Nouvelle notification",
    icon: data.icon || "/icon-192x192.png",
    badge: data.badge || "/badge-72x72.png",
    data: data.data || {},
    actions: data.actions || [],
  };

  event.waitUntil(
    self.registration.showNotification(data.title || "Notification", options)
  );
});

self.addEventListener("notificationclick", function (event) {
  event.notification.close();

  // Ouvrir ou focus sur une page
  event.waitUntil(clients.openWindow("/"));
});
```

### 2. S'abonner aux notifications (Frontend)

```javascript
// Dans votre application React/Vue/Vanilla JS

async function subscribeToNotifications(userEmail) {
  try {
    // 1. Enregistrer le service worker
    const registration = await navigator.serviceWorker.register(
      "/service-worker.js"
    );
    console.log("Service Worker enregistré");

    // 2. Demander la permission
    const permission = await Notification.requestPermission();
    if (permission !== "granted") {
      console.log("Permission refusée");
      return;
    }

    // 3. Récupérer la clé publique VAPID
    const response = await fetch(
      "http://localhost:8090/api/notifications/vapid-public-key"
    );
    const { publicKey } = await response.json();

    // 4. Créer l'abonnement
    const subscription = await registration.pushManager.subscribe({
      userVisibleOnly: true,
      applicationServerKey: urlBase64ToUint8Array(publicKey),
    });

    // 5. Envoyer l'abonnement au serveur
    await fetch("http://localhost:8090/api/notifications/subscribe", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        user_id: userEmail,
        subscription: subscription.toJSON(),
      }),
    });

    console.log("✅ Abonné aux notifications!");
  } catch (error) {
    console.error("❌ Erreur:", error);
  }
}

// Fonction utilitaire pour convertir la clé VAPID
function urlBase64ToUint8Array(base64String) {
  const padding = "=".repeat((4 - (base64String.length % 4)) % 4);
  const base64 = (base64String + padding)
    .replace(/\-/g, "+")
    .replace(/_/g, "/");

  const rawData = window.atob(base64);
  const outputArray = new Uint8Array(rawData.length);

  for (let i = 0; i < rawData.length; ++i) {
    outputArray[i] = rawData.charCodeAt(i);
  }
  return outputArray;
}

// Utilisation
subscribeToNotifications("user@email.com");
```

## Test avec curl

### 1. Connexion et récupération du token

```bash
TOKEN=$(curl -s -X POST http://localhost:8090/api/connexion \
  -H "Content-Type: application/json" \
  -d '{"email":"test@email.com","password":"password123"}' \
  | jq -r '.token')

echo "Token: $TOKEN"
```

### 2. Envoyer une notification à tous les abonnés

```bash
curl -X POST http://localhost:8090/api/notification/test \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "user_id": "test@email.com",
    "title": "Test de notification",
    "message": "Ceci est un test!",
    "data": {
      "url": "/dashboard"
    }
  }'
```

## Test avec Ngrok

Si vous utilisez ngrok :

```bash
# Remplacez l'URL locale par l'URL ngrok dans votre frontend
const API_URL = 'https://nia-preinstructive-nola.ngrok-free.dev';

// Puis utilisez API_URL dans vos fetch
fetch(`${API_URL}/api/notifications/subscribe`, ...)
```

## Fonctionnalités

✅ **Abonnement aux notifications** - Les utilisateurs peuvent s'abonner depuis n'importe quel appareil  
✅ **Notifications push** - Envoi de notifications même quand le navigateur est fermé  
✅ **Gestion des abonnements** - Stockage dans MongoDB  
✅ **Endpoint protégé** - Seuls les utilisateurs authentifiés peuvent envoyer des notifications  
✅ **Broadcast** - Envoie à tous les abonnés en une fois  
✅ **Nettoyage automatique** - Supprime les abonnements invalides (410 Gone)

## Sécurité

- 🔒 Les clés VAPID sont stockées dans `.env` (jamais dans le code)
- 🔐 L'endpoint d'envoi nécessite un token JWT valide
- ✅ Les clés sont générées avec ECDSA P-256
- 📧 Le VAPID Subject contient votre email

## Logs

Le serveur log toutes les actions :

- ✓ Nouvel abonnement créé
- ✓ Notification envoyée
- ❌ Échec d'envoi
- 🗑️ Suppression d'abonnement invalide
- 📊 Statistiques d'envoi

## Régénérer les clés VAPID

Si vous devez régénérer les clés :

```bash
go run cmd/generate-vapid/main.go
```

Puis copiez les nouvelles clés dans votre fichier `.env`.

⚠️ **Attention** : Régénérer les clés invalidera tous les abonnements existants. Les utilisateurs devront se réabonner.
