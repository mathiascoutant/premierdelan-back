// Service Worker pour les notifications PWA
const SW_VERSION = 'v2.3.0';
console.log('🔔 Service Worker chargé -', SW_VERSION);

// Force l'activation immédiate du nouveau service worker
self.addEventListener('install', (event) => {
  console.log('📥 Installation du service worker', SW_VERSION);
  self.skipWaiting(); // Force le nouveau SW à s'activer immédiatement
});

self.addEventListener('activate', (event) => {
  console.log('✅ Activation du service worker', SW_VERSION);
  event.waitUntil(
    clients.claim() // Prend le contrôle de tous les clients immédiatement
  );
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
  
  // Récupérer les données de la notification
  const notificationData = event.notification.data || {};
  
  // Détecter le base path (pour GitHub Pages: /premierdelan/)
  // On le détecte depuis l'URL du service worker lui-même
  const swUrl = self.location.pathname;
  const basePath = swUrl.substring(0, swUrl.lastIndexOf('/') + 1);
  
  // Construire l'URL de destination
  let urlPath = '';
  
  if (notificationData.type === 'chat_message' && notificationData.conversationId) {
    urlPath = 'chat?conversation=' + notificationData.conversationId;
  } else if (notificationData.type === 'chat_invitation') {
    urlPath = 'chat';
  } else if (notificationData.type === 'new_inscription' && notificationData.event_id) {
    urlPath = 'admin/evenements/' + notificationData.event_id;
  } else if (notificationData.type === 'alert') {
    urlPath = 'alertes';
  }
  
  // Construire l'URL complète avec le base path
  const fullUrl = self.location.origin + basePath + urlPath;
  
  event.waitUntil(
    clients.matchAll({ 
      type: 'window',
      includeUncontrolled: true 
    }).then(function(clientList) {
      // Chercher un client qui correspond à l'origin
      for (let i = 0; i < clientList.length; i++) {
        const client = clientList[i];
        if (client.url.indexOf(self.location.origin) === 0 && 'focus' in client) {
          // Envoyer le message au client avant de le focus
          client.postMessage({
            type: 'NOTIFICATION_CLICK',
            path: urlPath,
            conversationId: notificationData.conversationId,
            data: notificationData
          });
          return client.focus();
        }
      }
      
      // Aucun client trouvé, ouvrir une nouvelle fenêtre
      if (clients.openWindow) {
        return clients.openWindow(fullUrl);
      }
    })
  );
});

// Gérer la fermeture de la notification
self.addEventListener('notificationclose', function(event) {
  console.log('❌ Notification fermée');
});

