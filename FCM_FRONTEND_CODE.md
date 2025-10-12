# 🔥 Code Frontend pour Firebase Cloud Messaging

## 📋 Configuration Firebase

Voici les valeurs à utiliser dans votre frontend :

```javascript
const firebaseConfig = {
  apiKey: "AIzaSyBdQ8j21Vx7N2myh6ir8gY_zZkRCl-25qI",
  authDomain: "premier-de-lan.firebaseapp.com",
  projectId: "premier-de-lan",
  storageBucket: "premier-de-lan.firebasestorage.app",
  messagingSenderId: "220494656911",
  appId: "1:220494656911:web:2ff99839c5f7271ddf07fa",
  measurementId: "G-L06FQVLPE1"
};

const vapidKey = "BKtsyuWpu2lZY64MGiqwnBglbWFUBd9oMQWnmH9F3Y6DJ8gBSmXo0ASIwCZXxyK1XvXu_CxKwAd3cVSw-sNQ70o";
```

## 📄 Fichier 1 : `public/firebase-messaging-sw.js`

Créez ce fichier dans votre dossier `public/` :

```javascript
// Service Worker pour Firebase Cloud Messaging
importScripts('https://www.gstatic.com/firebasejs/10.7.1/firebase-app-compat.js');
importScripts('https://www.gstatic.com/firebasejs/10.7.1/firebase-messaging-compat.js');

// Configuration Firebase
const firebaseConfig = {
  apiKey: "AIzaSyBdQ8j21Vx7N2myh6ir8gY_zZkRCl-25qI",
  authDomain: "premier-de-lan.firebaseapp.com",
  projectId: "premier-de-lan",
  storageBucket: "premier-de-lan.firebasestorage.app",
  messagingSenderId: "220494656911",
  appId: "1:220494656911:web:2ff99839c5f7271ddf07fa"
};

// Initialiser Firebase
firebase.initializeApp(firebaseConfig);

// Initialiser Messaging
const messaging = firebase.messaging();

// Gérer les notifications en arrière-plan
messaging.onBackgroundMessage((payload) => {
  console.log('📩 Notification reçue en arrière-plan:', payload);
  
  const notificationTitle = payload.notification?.title || 'Nouvelle notification';
  const notificationOptions = {
    body: payload.notification?.body || '',
    icon: '/icon-192x192.png',
    badge: '/badge-72x72.png',
    data: payload.data || {},
    vibrate: [200, 100, 200]
  };

  self.registration.showNotification(notificationTitle, notificationOptions);
});

// Gérer le clic sur la notification
self.addEventListener('notificationclick', function(event) {
  console.log('👆 Notification cliquée');
  event.notification.close();
  
  event.waitUntil(
    clients.matchAll({ type: 'window', includeUncontrolled: true })
      .then(function(clientList) {
        for (let i = 0; i < clientList.length; i++) {
          const client = clientList[i];
          if ('focus' in client) {
            return client.focus();
          }
        }
        if (clients.openWindow) {
          return clients.openWindow('/');
        }
      })
  );
});
```

## 📄 Fichier 2 : Code Frontend (React/Vue/Vanilla JS)

### Installation des dépendances (si vous utilisez npm)

```bash
npm install firebase
```

### Code JavaScript

```javascript
import { initializeApp } from 'firebase/app';
import { getMessaging, getToken, onMessage } from 'firebase/messaging';

// Configuration Firebase
const firebaseConfig = {
  apiKey: "AIzaSyBdQ8j21Vx7N2myh6ir8gY_zZkRCl-25qI",
  authDomain: "premier-de-lan.firebaseapp.com",
  projectId: "premier-de-lan",
  storageBucket: "premier-de-lan.firebasestorage.app",
  messagingSenderId: "220494656911",
  appId: "1:220494656911:web:2ff99839c5f7271ddf07fa"
};

const vapidKey = "BKtsyuWpu2lZY64MGiqwnBglbWFUBd9oMQWnmH9F3Y6DJ8gBSmXo0ASIwCZXxyK1XvXu_CxKwAd3cVSw-sNQ70o";

// Initialiser Firebase
const app = initializeApp(firebaseConfig);
const messaging = getMessaging(app);

// Fonction pour activer les notifications
async function activerNotificationsFCM(userEmail) {
  try {
    console.log('📱 Activation des notifications FCM...');
    
    // 1. Vérifier le support
    if (!('serviceWorker' in navigator)) {
      alert('Les Service Workers ne sont pas supportés');
      return;
    }

    // 2. Enregistrer le Service Worker
    const registration = await navigator.serviceWorker.register('/firebase-messaging-sw.js');
    console.log('✅ Service Worker enregistré');

    // 3. Demander la permission
    const permission = await Notification.requestPermission();
    console.log('Permission:', permission);
    
    if (permission !== 'granted') {
      alert('Permission refusée');
      return;
    }

    // 4. Obtenir le token FCM
    const fcmToken = await getToken(messaging, { 
      vapidKey: vapidKey,
      serviceWorkerRegistration: registration
    });
    
    console.log('✅ Token FCM obtenu:', fcmToken);

    // 5. Envoyer le token au backend
    const response = await fetch('https://nia-preinstructive-nola.ngrok-free.dev/api/fcm/subscribe', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'ngrok-skip-browser-warning': 'true'
      },
      body: JSON.stringify({
        user_id: userEmail,
        fcm_token: fcmToken,
        device: 'Web',
        user_agent: navigator.userAgent
      })
    });

    if (!response.ok) {
      throw new Error('Erreur lors de l\'abonnement');
    }

    const data = await response.json();
    console.log('✅ Abonné aux notifications FCM!', data);
    alert('🎉 Notifications activées! Vous recevrez les notifications.');

  } catch (error) {
    console.error('❌ Erreur:', error);
    alert('Erreur: ' + error.message);
  }
}

// Gérer les notifications quand l'app est au premier plan
onMessage(messaging, (payload) => {
  console.log('📩 Notification reçue au premier plan:', payload);
  
  // Afficher une notification même si l'app est ouverte
  if (Notification.permission === 'granted') {
    new Notification(payload.notification.title, {
      body: payload.notification.body,
      icon: '/icon-192x192.png',
      badge: '/badge-72x72.png'
    });
  }
});

// Export de la fonction
export { activerNotificationsFCM };
```

## 🎨 Intégration dans votre HTML/React

### HTML Simple (Vanilla JS)

```html
<!DOCTYPE html>
<html>
<head>
  <title>Premier de l'an</title>
</head>
<body>
  <h1>Mon App</h1>
  
  <!-- Bouton pour activer les notifications -->
  <button id="btn-notif" onclick="activerNotifications()">
    🔔 Activer les notifications
  </button>

  <!-- Charger Firebase depuis CDN -->
  <script type="module">
    import { initializeApp } from 'https://www.gstatic.com/firebasejs/10.7.1/firebase-app.js';
    import { getMessaging, getToken, onMessage } from 'https://www.gstatic.com/firebasejs/10.7.1/firebase-messaging.js';

    const firebaseConfig = {
      apiKey: "AIzaSyBdQ8j21Vx7N2myh6ir8gY_zZkRCl-25qI",
      authDomain: "premier-de-lan.firebaseapp.com",
      projectId: "premier-de-lan",
      storageBucket: "premier-de-lan.firebasestorage.app",
      messagingSenderId: "220494656911",
      appId: "1:220494656911:web:2ff99839c5f7271ddf07fa"
    };

    const vapidKey = "BKtsyuWpu2lZY64MGiqwnBglbWFUBd9oMQWnmH9F3Y6DJ8gBSmXo0ASIwCZXxyK1XvXu_CxKwAd3cVSw-sNQ70o";

    const app = initializeApp(firebaseConfig);
    const messaging = getMessaging(app);

    window.activerNotifications = async function() {
      try {
        const registration = await navigator.serviceWorker.register('/firebase-messaging-sw.js');
        const permission = await Notification.requestPermission();
        
        if (permission !== 'granted') {
          alert('Permission refusée');
          return;
        }

        const fcmToken = await getToken(messaging, { 
          vapidKey: vapidKey,
          serviceWorkerRegistration: registration
        });

        const userEmail = 'mathiascoutant@icloud.com'; // Récupérez l'email de l'utilisateur connecté
        
        await fetch('https://nia-preinstructive-nola.ngrok-free.dev/api/fcm/subscribe', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'ngrok-skip-browser-warning': 'true'
          },
          body: JSON.stringify({
            user_id: userEmail,
            fcm_token: fcmToken
          })
        });

        alert('🎉 Notifications activées!');
      } catch (error) {
        console.error(error);
        alert('Erreur: ' + error.message);
      }
    };

    // Gérer les notifications au premier plan
    onMessage(messaging, (payload) => {
      console.log('Notification reçue:', payload);
      new Notification(payload.notification.title, {
        body: payload.notification.body,
        icon: '/icon-192x192.png'
      });
    });
  </script>
</body>
</html>
```

## 🧪 Test de l'API backend

### 1. Tester que la clé VAPID est accessible
```bash
curl https://nia-preinstructive-nola.ngrok-free.dev/api/fcm/vapid-key
```

### 2. Envoyer une notification (après s'être abonné)
```bash
# Se connecter
TOKEN=$(curl -s -X POST https://nia-preinstructive-nola.ngrok-free.dev/api/connexion \
  -H "Content-Type: application/json" \
  -H "ngrok-skip-browser-warning: true" \
  -d '{"email":"mathiascoutant@icloud.com","password":"test1234"}' \
  | python3 -c "import sys, json; print(json.load(sys.stdin)['token'])")

# Envoyer notification
curl -X POST https://nia-preinstructive-nola.ngrok-free.dev/api/fcm/send \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -H "ngrok-skip-browser-warning: true" \
  -d '{
    "user_id": "mathiascoutant@icloud.com",
    "title": "🔥 Test FCM",
    "message": "Notification depuis Firebase!",
    "data": {
      "action": "test"
    }
  }'
```

## 🎯 Format de la requête backend (ce que votre frontend envoie déjà)

```
POST /api/notification/test
```

est maintenant remplacé par :

```
POST /api/fcm/send
```

Même format, meilleure compatibilité ! 🚀

## ✅ Résumé

- ✅ Backend configuré avec Firebase
- ✅ Routes FCM créées
- ✅ Serveur démarré et opérationnel
- 📝 Code frontend documenté

**Maintenant, ajoutez le code du fichier `public/firebase-messaging-sw.js` dans votre frontend GitHub Pages et testez !** 🎉

