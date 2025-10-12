// Service Worker pour les notifications PWA
console.log('🔔 Service Worker chargé');

// Écouter les notifications push
self.addEventListener('push', function(event) {
  console.log('📩 Notification push reçue');
  
  const data = event.data ? event.data.json() : {};
  
  const options = {
    body: data.body || 'Nouvelle notification',
    icon: data.icon || '/icon-192x192.png',
    badge: data.badge || '/badge-72x72.png',
    data: data.data || {},
    vibrate: [200, 100, 200],
    tag: 'notification-' + Date.now(),
    requireInteraction: false,
    actions: data.actions || []
  };

  event.waitUntil(
    self.registration.showNotification(data.title || 'Notification', options)
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

