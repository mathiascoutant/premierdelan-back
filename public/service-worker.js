// Service Worker pour les notifications PWA
console.log('🔔 Service Worker chargé');

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
  
  // ⚠️ Sur iOS avec FCMOptions.Link, ne PAS afficher de notification ici
  // iOS affiche automatiquement la notification avec l'URL
  // Afficher uniquement si type n'est pas chat_message (pour éviter doublons)
  if (notificationData.type === 'chat_message') {
    console.log('🍎 Notification chat - iOS gère automatiquement via FCMOptions.Link');
    return; // Ne rien faire, iOS s'en occupe
  }
  
  const options = {
    body: body,
    icon: '/icon-192x192.png',
    badge: '/badge-72x72.png',
    data: notificationData,
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
  console.log('👆 Notification cliquée');
  
  event.notification.close();
  
  // Ouvrir ou focus sur votre site
  event.waitUntil(
    clients.matchAll({ type: 'window', includeUncontrolled: true })
      .then(function(clientList) {
        // Si une fenêtre est déjà ouverte, la focus
        for (let i = 0; i < clientList.length; i++) {
          const client = clientList[i];
          if (client.url === '/' && 'focus' in client) {
            return client.focus();
          }
        }
        // Sinon ouvrir une nouvelle fenêtre
        if (clients.openWindow) {
          return clients.openWindow('/');
        }
      })
  );
});

// Gérer la fermeture de la notification
self.addEventListener('notificationclose', function(event) {
  console.log('❌ Notification fermée');
});

