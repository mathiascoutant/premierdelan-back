// Service Worker pour les notifications PWA
const SW_VERSION = 'v2.4.0';
console.log('🔔 Service Worker chargé -', SW_VERSION);

// Force l'activation immédiate du nouveau service worker
self.addEventListener('install', (event) => {
  console.log('📥 Installation du service worker', SW_VERSION);
  self.skipWaiting();
});

self.addEventListener('activate', (event) => {
  console.log('✅ Activation du service worker', SW_VERSION);
  event.waitUntil(clients.claim());
});

// Écouter les notifications push
self.addEventListener('push', function(event) {
  console.log('📩 Notification push reçue');
  
  let payload = {};
  let title = 'Notification';
  let body = 'Nouvelle notification';
  let notificationData = {};
  
  if (event.data) {
    try {
      payload = event.data.json();
      console.log('📦 Payload reçu:', payload);
      
      // FCM envoie les données dans payload.data
      if (payload.data) {
        notificationData = payload.data;
      }
      
      // Le titre et le body peuvent être dans notification ou directement dans payload
      if (payload.notification) {
        title = payload.notification.title || title;
        body = payload.notification.body || body;
      } else {
        title = payload.title || title;
        body = payload.body || body;
      }
    } catch (e) {
      console.error('❌ Erreur parsing payload:', e);
    }
  }
  
  console.log('📨 Notification data:', notificationData);
  
  const options = {
    body: body,
    icon: '/icon-192x192.png',
    badge: '/badge-72x72.png',
    data: notificationData, // Les données FCM sont ici
    vibrate: [200, 100, 200],
    tag: 'notification-' + Date.now(),
    requireInteraction: false
  };

  event.waitUntil(
    self.registration.showNotification(title, options)
  );
});

// Gérer le clic sur la notification
self.addEventListener('notificationclick', function(event) {
  event.notification.close();
  
  const data = event.notification.data || {};
  let url = 'https://mathiascoutant.github.io/premierdelan/';

  if (data.type === 'chat_message' && data.conversationId) {
    url = 'https://mathiascoutant.github.io/premierdelan/chat?conversation=' + data.conversationId;
  } else if (data.type === 'chat_invitation') {
    url = 'https://mathiascoutant.github.io/premierdelan/chat';
  } else if (data.type === 'new_inscription' && data.event_id) {
    url = 'https://mathiascoutant.github.io/premierdelan/admin/evenements/' + data.event_id;
  } else if (data.type === 'alert') {
    url = 'https://mathiascoutant.github.io/premierdelan/alertes';
  }
  
  event.waitUntil(clients.openWindow(url));
});

// Gérer la fermeture de la notification
self.addEventListener('notificationclose', function(event) {
  console.log('❌ Notification fermée');
});

