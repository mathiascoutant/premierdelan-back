# 🔥 Configuration Firebase Cloud Messaging (FCM)

## Étape 1 : Créer un projet Firebase (5 min)

### 1.1 Aller sur Firebase Console

👉 https://console.firebase.google.com/

### 1.2 Cliquer sur "Ajouter un projet"

- Nom du projet : `premier-de-lan` (ou le nom de votre choix)
- Accepter les conditions
- **Désactiver Google Analytics** (pas nécessaire pour les notifications)
- Cliquer sur "Créer le projet"

### 1.3 Ajouter une application Web

1. Dans la console Firebase, cliquer sur l'icône **Web** (`</>`)
2. Surnom de l'app : `Premier de l'an PWA`
3. ✅ Cocher "Configurer aussi Firebase Hosting" (optionnel)
4. Cliquer sur "Enregistrer l'application"

### 1.4 Récupérer la configuration

Firebase vous donnera un code comme celui-ci :

```javascript
const firebaseConfig = {
  apiKey: "AIzaSyC...",
  authDomain: "premier-de-lan.firebaseapp.com",
  projectId: "premier-de-lan",
  storageBucket: "premier-de-lan.appspot.com",
  messagingSenderId: "123456789",
  appId: "1:123456789:web:abc123",
};
```

**📝 GARDEZ CES VALEURS**, vous en aurez besoin !

## Étape 2 : Activer Cloud Messaging

### 2.1 Dans la console Firebase

1. Menu latéral → **Engagement** → **Cloud Messaging**
2. Cliquer sur "Commencer"
3. Si demandé, activer l'API Cloud Messaging

### 2.2 Générer une clé de serveur (Server Key)

1. Aller dans **Paramètres du projet** (icône engrenage en haut à gauche)
2. Onglet **Cloud Messaging**
3. Copier la **Clé du serveur** (Server key) - ressemble à : `AAAA...`
4. **📝 GARDEZ CETTE CLÉ**, c'est votre `FCM_SERVER_KEY` pour le backend !

### 2.3 Générer un certificat Web Push (VAPID)

1. Toujours dans **Paramètres → Cloud Messaging**
2. Section **Certificats de clés Web Push**
3. Cliquer sur "Générer une paire de clés"
4. Copier la **Clé publique** - ressemble à : `BH3z...`
5. **📝 GARDEZ CETTE CLÉ**, c'est votre `VAPID_KEY` pour le frontend !

## Étape 3 : Mettre à jour votre fichier .env

Ouvrez `/Users/mathias/Desktop/    /premier de l'an/site/back/.env` et ajoutez :

```env
# Firebase Cloud Messaging
FCM_SERVER_KEY=AAAA_votre_server_key_ici
FCM_PROJECT_ID=premier-de-lan
```

## Étape 4 : Frontend - Configuration Firebase

### 4.1 Créer `public/firebase-config.js`

```javascript
// Configuration Firebase (PUBLIQUE - pas de secret ici)
const firebaseConfig = {
  apiKey: "AIzaSyC...", // ⚠️ Remplacez par VOS valeurs
  authDomain: "premier-de-lan.firebaseapp.com",
  projectId: "premier-de-lan",
  storageBucket: "premier-de-lan.appspot.com",
  messagingSenderId: "123456789",
  appId: "1:123456789:web:abc123",
};

// VAPID Key pour les notifications Web
const vapidKey = "BH3z..."; // ⚠️ Remplacez par VOTRE clé VAPID
```

### 4.2 Créer `public/firebase-messaging-sw.js` (Service Worker)

```javascript
// Service Worker pour Firebase Cloud Messaging
importScripts(
  "https://www.gstatic.com/firebasejs/10.7.1/firebase-app-compat.js"
);
importScripts(
  "https://www.gstatic.com/firebasejs/10.7.1/firebase-messaging-compat.js"
);
importScripts("/firebase-config.js");

// Initialiser Firebase
firebase.initializeApp(firebaseConfig);

// Initialiser Messaging
const messaging = firebase.messaging();

// Gérer les notifications en arrière-plan
messaging.onBackgroundMessage((payload) => {
  console.log("📩 Notification reçue en arrière-plan:", payload);

  const notificationTitle =
    payload.notification.title || "Nouvelle notification";
  const notificationOptions = {
    body: payload.notification.body || "",
    icon: payload.notification.icon || "/icon-192x192.png",
    badge: "/badge-72x72.png",
    data: payload.data || {},
    vibrate: [200, 100, 200],
  };

  self.registration.showNotification(notificationTitle, notificationOptions);
});
```

### 4.3 Code Frontend pour s'abonner

```javascript
// Importer Firebase (dans votre HTML ou via npm)
import { initializeApp } from "firebase/app";
import { getMessaging, getToken, onMessage } from "firebase/messaging";

// Initialiser Firebase
const app = initializeApp(firebaseConfig);
const messaging = getMessaging(app);

// Fonction pour s'abonner aux notifications
async function subscribeToNotifications(userEmail) {
  try {
    // Demander la permission
    const permission = await Notification.requestPermission();
    if (permission !== "granted") {
      console.log("Permission refusée");
      return;
    }

    // Récupérer le token FCM
    const token = await getToken(messaging, { vapidKey: vapidKey });
    console.log("✅ Token FCM:", token);

    // Envoyer le token au backend
    await fetch(
      "https://nia-preinstructive-nola.ngrok-free.dev/api/fcm/subscribe",
      {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "ngrok-skip-browser-warning": "true",
        },
        body: JSON.stringify({
          user_id: userEmail,
          fcm_token: token,
        }),
      }
    );

    console.log("✅ Abonné aux notifications FCM!");
    alert("🎉 Notifications activées!");
  } catch (error) {
    console.error("❌ Erreur:", error);
    alert("Erreur: " + error.message);
  }
}

// Gérer les notifications quand l'app est au premier plan
onMessage(messaging, (payload) => {
  console.log("📩 Notification reçue au premier plan:", payload);

  // Afficher une notification même si l'app est ouverte
  new Notification(payload.notification.title, {
    body: payload.notification.body,
    icon: payload.notification.icon || "/icon-192x192.png",
  });
});
```

## 📋 Checklist de configuration

- [ ] Projet Firebase créé
- [ ] Application Web ajoutée
- [ ] Cloud Messaging activé
- [ ] Server Key copiée (pour le backend)
- [ ] VAPID Key copiée (pour le frontend)
- [ ] Fichier `.env` mis à jour
- [ ] `firebase-config.js` créé
- [ ] `firebase-messaging-sw.js` créé
- [ ] Code d'abonnement ajouté au frontend

## 🚀 Prochaines étapes

Une fois cette configuration terminée, je vais :

1. Modifier le backend Go pour utiliser FCM
2. Créer les nouveaux endpoints `/api/fcm/subscribe` et `/api/fcm/send`
3. Tester l'envoi de notifications

## 💡 Astuce

Les clés Firebase sont **publiques** (apiKey, projectId, etc.) et peuvent être dans votre code frontend.
Seule la **Server Key** doit rester **secrète** côté backend dans le fichier `.env`.

## ❓ Besoin d'aide ?

Si vous avez des questions pendant la configuration, n'hésitez pas !
