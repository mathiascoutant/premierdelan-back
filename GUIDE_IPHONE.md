# 📱 Guide pour recevoir des notifications sur iPhone

## ⚠️ Important : Processus en 2 étapes

Pour recevoir des notifications sur votre iPhone, il faut :

1. **S'ABONNER** aux notifications depuis votre PWA (à faire UNE FOIS)
2. **ENVOYER** une notification via l'API

## 🔔 Étape 1 : S'abonner aux notifications (Frontend)

Votre frontend doit implémenter cette fonction et l'appeler quand l'utilisateur se connecte ou clique sur "Activer les notifications" :

```javascript
// Dans votre frontend (React/Vue/Vanilla JS)
async function activerNotificationsPWA(userEmail) {
  try {
    console.log("📱 Activation des notifications...");

    // 1. Vérifier si les notifications sont supportées
    if (!("serviceWorker" in navigator) || !("PushManager" in window)) {
      alert("Les notifications ne sont pas supportées sur cet appareil");
      return;
    }

    // 2. Enregistrer le Service Worker
    const registration = await navigator.serviceWorker.register(
      "/service-worker.js"
    );
    console.log("✅ Service Worker enregistré");

    // 3. Demander la permission
    const permission = await Notification.requestPermission();
    console.log("Permission:", permission);

    if (permission !== "granted") {
      alert(
        "Permission refusée. Activez les notifications dans les paramètres."
      );
      return;
    }

    // 4. Récupérer la clé publique VAPID depuis l'API
    const vapidResponse = await fetch(
      "https://nia-preinstructive-nola.ngrok-free.dev/api/notifications/vapid-public-key"
    );
    const { publicKey } = await vapidResponse.json();
    console.log("✅ Clé VAPID récupérée");

    // 5. S'abonner aux notifications push
    const subscription = await registration.pushManager.subscribe({
      userVisibleOnly: true,
      applicationServerKey: urlBase64ToUint8Array(publicKey),
    });
    console.log("✅ Abonnement push créé");

    // 6. Envoyer l'abonnement au serveur
    const subscribeResponse = await fetch(
      "https://nia-preinstructive-nola.ngrok-free.dev/api/notifications/subscribe",
      {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          user_id: userEmail,
          subscription: subscription.toJSON(),
        }),
      }
    );

    if (!subscribeResponse.ok) {
      throw new Error("Erreur lors de l'abonnement");
    }

    console.log("✅ Abonné aux notifications!");
    alert(
      "🎉 Notifications activées! Vous recevrez maintenant les notifications."
    );
  } catch (error) {
    console.error("❌ Erreur:", error);
    alert("Erreur lors de l'activation des notifications: " + error.message);
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
```

## 📄 Étape 1.5 : Créer le Service Worker

Créez un fichier `public/service-worker.js` dans votre frontend :

```javascript
// public/service-worker.js
console.log("🔔 Service Worker chargé");

// Écouter les notifications push
self.addEventListener("push", function (event) {
  console.log("📩 Notification push reçue");

  const data = event.data ? event.data.json() : {};

  const options = {
    body: data.body || "Nouvelle notification",
    icon: data.icon || "/icon-192x192.png",
    badge: data.badge || "/badge-72x72.png",
    data: data.data || {},
    vibrate: [200, 100, 200],
    tag: "notification-" + Date.now(),
    requireInteraction: false,
  };

  event.waitUntil(
    self.registration.showNotification(data.title || "Notification", options)
  );
});

// Gérer le clic sur la notification
self.addEventListener("notificationclick", function (event) {
  console.log("👆 Notification cliquée");
  event.notification.close();

  event.waitUntil(
    clients
      .matchAll({ type: "window", includeUncontrolled: true })
      .then(function (clientList) {
        for (let i = 0; i < clientList.length; i++) {
          const client = clientList[i];
          if ("focus" in client) {
            return client.focus();
          }
        }
        if (clients.openWindow) {
          return clients.openWindow("/");
        }
      })
  );
});
```

## 🎯 Étape 2 : Appeler la fonction dans votre app

Appelez cette fonction quand vous voulez activer les notifications :

```javascript
// Exemple : Après la connexion
async function handleLogin(email, password) {
  // ... votre code de connexion existant ...

  // Proposer d'activer les notifications
  if (confirm("Voulez-vous activer les notifications?")) {
    await activerNotificationsPWA(email);
  }
}

// Ou avec un bouton
<button onClick={() => activerNotificationsPWA(userEmail)}>
  🔔 Activer les notifications
</button>;
```

## 📱 Sur iPhone

1. Ouvrez Safari sur votre iPhone
2. Allez sur `https://mathiascoutant.github.io`
3. Cliquez sur le bouton "Partager" (icône carré avec flèche)
4. Sélectionnez "Sur l'écran d'accueil"
5. Ouvrez l'app depuis l'écran d'accueil
6. **Cliquez sur votre bouton "Activer les notifications"**
7. Acceptez la permission
8. ✅ Vous êtes maintenant abonné !

## 🧪 Tester l'envoi

Une fois abonné, envoyez une notification depuis votre API :

```bash
# 1. Se connecter
TOKEN=$(curl -s -X POST https://nia-preinstructive-nola.ngrok-free.dev/api/connexion \
  -H "Content-Type: application/json" \
  -d '{"email":"votre@email.com","password":"votre_mdp"}' \
  | jq -r '.token')

# 2. Envoyer la notification
curl -X POST https://nia-preinstructive-nola.ngrok-free.dev/api/notification/test \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "user_id": "votre@email.com",
    "title": "Test iPhone",
    "message": "Ceci est un test depuis l API!"
  }'
```

## ⚠️ Limitations iOS

**Important** : Sur iOS, les notifications PWA ne fonctionnent que si :

- L'app est installée sur l'écran d'accueil (✅ vous l'avez fait)
- L'utilisateur a accepté les permissions
- Le Service Worker est bien enregistré
- iOS 16.4+ (notifications PWA supportées depuis cette version)

## 🔍 Debug

Pour vérifier si vous êtes abonné :

```bash
# Voir tous les abonnements
mongosh premierdelan --eval "db.subscriptions.find().pretty()"

# Compter les abonnements
mongosh premierdelan --eval "db.subscriptions.countDocuments()"
```

## 📊 Vérifier les logs serveur

Quand vous vous abonnez, vous devriez voir dans les logs :

```
✓ Nouvel abonnement créé pour: votre@email.com
```

Quand vous envoyez une notification :

```
✓ Notification envoyée à votre@email.com
📊 Notifications envoyées: 1/1 (échecs: 0)
```

## 🎯 Résumé

1. ❌ **Actuellement** : Vous n'êtes pas encore abonné aux notifications
2. ✅ **Solution** : Ajoutez le code d'abonnement dans votre frontend
3. 📱 **Testez** : Ouvrez votre PWA, cliquez sur "Activer notifications"
4. 🔔 **Recevez** : Envoyez une notification via l'API

Besoin d'aide pour intégrer le code dans votre frontend ? 😊
