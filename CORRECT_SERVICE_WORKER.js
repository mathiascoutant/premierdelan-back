// ========================================
// Service Worker Firebase - SANS "from ..."
// À copier dans: public/firebase-messaging-sw.js
// ========================================

importScripts('https://www.gstatic.com/firebasejs/10.7.1/firebase-app-compat.js');
importScripts('https://www.gstatic.com/firebasejs/10.7.1/firebase-messaging-compat.js');

// Configuration Firebase
firebase.initializeApp({
  apiKey: "AIzaSyBdQ8j21Vx7N2myh6ir8gY_zZkRCl-25qI",
  authDomain: "premier-de-lan.firebaseapp.com",
  projectId: "premier-de-lan",
  storageBucket: "premier-de-lan.firebasestorage.app",
  messagingSenderId: "220494656911",
  appId: "1:220494656911:web:2ff99839c5f7271ddf07fa"
});

const messaging = firebase.messaging();

// ⭐ IMPORTANT: Gérer les DATA MESSAGES (pas les notification messages)
// Cela évite le "from ..." sur iOS
self.addEventListener('push', function(event) {
  console.log('📩 Push reçu:', event);
  
  try {
    const payload = event.data.json();
    console.log('Payload complet:', payload);
    
    // Le backend envoie UNIQUEMENT des data messages
    // Donc les infos sont dans payload.data (pas payload.notification)
    const data = payload.data || {};
    
    const title = data.title || 'Notification';
    const message = data.message || '';
    
    console.log('Titre:', title);
    console.log('Message:', message);
    
    const notificationOptions = {
      body: message,
      icon: '/icon-192x192.png',
      badge: '/badge-72x72.png',
      vibrate: [200, 100, 200],
      tag: 'premier-de-lan-notif',
      requireInteraction: false,
      data: data
    };
    
    event.waitUntil(
      self.registration.showNotification(title, notificationOptions)
    );
  } catch (error) {
    console.error('❌ Erreur dans le Service Worker:', error);
  }
});

// Gérer le clic sur la notification
self.addEventListener('notificationclick', function(event) {
  console.log('👆 Notification cliquée');
  event.notification.close();
  
  const url = event.notification.data?.url || '/';
  
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
          return clients.openWindow(url);
        }
      })
  );
});

console.log('✅ Service Worker Firebase chargé - Mode DATA MESSAGE (sans from)');
