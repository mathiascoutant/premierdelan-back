# 🔔 Configuration Frontend pour la Redirection des Notifications

## Problème
Lorsqu'un utilisateur clique sur une notification push, le service worker envoie un message via `postMessage` pour rediriger l'app vers la bonne page.

## Solution Frontend

### 1. Écouter les messages du Service Worker

Ajoutez ce code dans votre fichier principal (ex: `App.tsx`, `main.tsx`, ou `_app.tsx`) :

```typescript
// Écouter les messages du service worker pour les redirections
useEffect(() => {
  if ('serviceWorker' in navigator) {
    navigator.serviceWorker.addEventListener('message', (event) => {
      console.log('📨 Message reçu du service worker:', event.data);
      
      if (event.data && event.data.type === 'NOTIFICATION_CLICK') {
        const { data } = event.data;
        
        console.log('🔔 Clic notification détecté:', data);
        
        // Redirection selon le type
        if (data.type === 'chat_message' && data.conversationId) {
          // Naviguer vers la conversation
          navigate(`/messages?conversation=${data.conversationId}`);
          console.log('💬 Redirection vers conversation:', data.conversationId);
        } else if (data.type === 'chat_invitation') {
          navigate('/messages');
        } else if (data.type === 'new_inscription' && data.event_id) {
          navigate(`/admin/evenements/${data.event_id}`);
        } else if (data.type === 'alert') {
          navigate('/alertes');
        }
      }
    });
  }
}, [navigate]);
```

### 2. Gérer le paramètre URL dans la page Messages

Dans votre composant `/messages` (ex: `Messages.tsx` ou `MessagesPage.tsx`), ajoutez :

```typescript
const searchParams = new URLSearchParams(window.location.search);
const conversationIdFromUrl = searchParams.get('conversation');

useEffect(() => {
  if (conversationIdFromUrl) {
    console.log('🔗 Conversation ID depuis URL:', conversationIdFromUrl);
    // Ouvrir automatiquement cette conversation
    setSelectedConversationId(conversationIdFromUrl);
    // Ou appeler votre fonction pour ouvrir la conversation
    // openConversation(conversationIdFromUrl);
  }
}, [conversationIdFromUrl]);
```

### 3. Alternative : Si vous utilisez React Router

Si vous utilisez `react-router-dom` avec `useSearchParams` :

```typescript
import { useSearchParams } from 'react-router-dom';

const [searchParams] = useSearchParams();
const conversationId = searchParams.get('conversation');

useEffect(() => {
  if (conversationId) {
    console.log('🔗 Ouverture conversation:', conversationId);
    openConversation(conversationId);
  }
}, [conversationId]);
```

## Test

1. **Fermez complètement l'app** (kill depuis le gestionnaire de tâches)
2. **Attendez le redéploiement de Railway** (nouveau service worker)
3. **Rouvrez l'app**
4. **Demandez à quelqu'un de vous envoyer un message**
5. **Cliquez sur la notification**
6. **Vérifiez les logs** dans la console :
   - `📨 Message reçu du service worker`
   - `🔔 Clic notification détecté`
   - `💬 Redirection vers conversation`

## Debugging

Si ça ne marche pas, ouvrez la console et cherchez :
- `👆 Notification cliquée`
- `📦 Type: chat_message`
- `📦 ConversationId: ...`
- `🎯 URL cible complète: ...`
- `🔍 Clients trouvés: 1`
- `🪟 Focus sur client existant`

Si vous ne voyez pas ces logs, le service worker n'est pas à jour.

