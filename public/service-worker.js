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
  console.log('📦 Type:', notificationData.type);
  console.log('📦 ConversationId:', notificationData.conversationId);
  
  // Obtenir l'URL de base (origin)
  const baseUrl = self.location.origin;
  
  // Déterminer l'URL de destination selon le type de notification
  let targetUrl = baseUrl + '/';
  
  if (notificationData.type === 'chat_message' && notificationData.conversationId) {
    // Rediriger vers la conversation spécifique
    targetUrl = baseUrl + `/messages?conversation=${notificationData.conversationId}`;
    console.log('💬 Chat message détecté');
  } else if (notificationData.type === 'chat_invitation') {
    // Rediriger vers la page des messages (invitations)
    targetUrl = baseUrl + '/messages';
    console.log('✉️ Chat invitation détecté');
  } else if (notificationData.type === 'new_inscription' && notificationData.event_id) {
    // Rediriger vers l'événement
    targetUrl = baseUrl + `/admin/evenements/${notificationData.event_id}`;
    console.log('📝 Nouvelle inscription détectée');
  } else if (notificationData.type === 'alert') {
    // Rediriger vers les alertes
    targetUrl = baseUrl + '/alertes';
    console.log('🚨 Alerte détectée');
  }
  
  console.log('🎯 URL cible complète:', targetUrl);
  
  // Ouvrir ou focus sur votre site
  event.waitUntil(
    clients.matchAll({ 
      type: 'window',
      includeUncontrolled: true 
    }).then(function(clientList) {
      console.log('🔍 Clients trouvés:', clientList.length);
      
      // Si une fenêtre est déjà ouverte
      if (clientList.length > 0) {
        const client = clientList[0];
        console.log('🪟 Focus sur client existant');
        
        // Essayer de naviguer avec postMessage (plus fiable sur Safari)
        client.postMessage({
          type: 'NOTIFICATION_CLICK',
          url: targetUrl,
          data: notificationData
        });
        
        return client.focus();
      }
      
      // Sinon ouvrir une nouvelle fenêtre avec l'URL cible
      console.log('🆕 Ouverture nouvelle fenêtre');
      if (clients.openWindow) {
        return clients.openWindow(targetUrl);
      }
    }).catch(function(error) {
      console.error('❌ Erreur lors du clic notification:', error);
    })
  );
});

// Gérer la fermeture de la notification
self.addEventListener('notificationclose', function(event) {
  console.log('❌ Notification fermée');
});

