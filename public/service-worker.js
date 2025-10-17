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
  console.log('👆 Notification cliquée');
  
  event.notification.close();
  
  // Récupérer les données de la notification
  const notificationData = event.notification.data || {};
  console.log('📦 Données notification:', notificationData);
  
  // Déterminer l'URL de destination selon le type de notification
  let targetUrl = '/';
  
  if (notificationData.type === 'chat_message' && notificationData.conversationId) {
    // Rediriger vers la conversation spécifique
    targetUrl = `/messages?conversation=${notificationData.conversationId}`;
  } else if (notificationData.type === 'chat_invitation') {
    // Rediriger vers la page des messages (invitations)
    targetUrl = '/messages';
  } else if (notificationData.type === 'new_inscription' && notificationData.event_id) {
    // Rediriger vers l'événement
    targetUrl = `/admin/evenements/${notificationData.event_id}`;
  } else if (notificationData.type === 'alert') {
    // Rediriger vers les alertes
    targetUrl = '/alertes';
  }
  
  console.log('🎯 URL cible:', targetUrl);
  
  // Ouvrir ou focus sur votre site
  event.waitUntil(
    clients.matchAll({ type: 'window', includeUncontrolled: true })
      .then(function(clientList) {
        // Si une fenêtre est déjà ouverte, la naviguer vers l'URL cible
        for (let i = 0; i < clientList.length; i++) {
          const client = clientList[i];
          if ('focus' in client) {
            return client.focus().then(() => {
              if ('navigate' in client) {
                return client.navigate(targetUrl);
              }
            });
          }
        }
        // Sinon ouvrir une nouvelle fenêtre avec l'URL cible
        if (clients.openWindow) {
          return clients.openWindow(targetUrl);
        }
      })
  );
});

// Gérer la fermeture de la notification
self.addEventListener('notificationclose', function(event) {
  console.log('❌ Notification fermée');
});

