# Frontend – WebSocket group chat checklist

Ce qui a été modifié côté backend :

- Tous les événements WebSocket de groupe (`new_group_message`, `group_user_typing`, `group_member_joined`, `group_messages_read`, `group_created`, `joined_group`) sont désormais envoyés **à tous les membres**, y compris le créateur/admin.
- L'événement de typing pour les groupes a désormais le type explicite `group_user_typing`.

Pour que l'interface affiche exactement la même chose pour tout le monde :

1. **Connexion WebSocket**
   - Utiliser le WebSocket global (`wss://premierdelan-back…/ws`).
   - Après l’authentification côté front, envoyer `join_group` pour chaque groupe chargé (ID Mongo en hex).

2. **Gestion des événements**
   - `group_created` : ajouter le groupe à la liste si l’utilisateur actuel est concerné (créateur ou invité).
   - `joined_group` : confirmation locale que la room WS est rejointe.
   - `new_group_message` :
     ```json
     {
       "type": "new_group_message",
       "group_id": "...",
       "message": {
         "id": "...",
         "sender_id": "email",
         "sender": {
           "id": "email",
           "firstname": "Prénom",
           "lastname": "Nom",
           "email": "email",
           "profile_picture": "URL nullable"
         },
         "content": "Message",
         "message_type": "message|system",
         "timestamp": "...",
         "read_by": ["email", ...]
       }
     }
     ```
     → Ajouter immédiatement le message dans la conversation ouverte, sans refetch.
   - `group_user_typing` :
     ```json
     {
       "type": "group_user_typing",
       "group_id": "...",
       "user_id": "email",
       "username": "Prénom Nom",
       "is_typing": true|false
     }
     ```
     → Afficher le statut “X est en train d’écrire…” pour tous les membres (y compris l’expéditeur).
   - `group_member_joined` / `group_member_left` : rafraîchir la liste des membres et les messages système.
   - `group_messages_read` : marquer les messages comme lus dans l’UI si `user_id` correspond à la personne affichée.

3. **Envoi côté frontend**
   - Pour signaler la frappe :
     ```ts
     ws.send(JSON.stringify({
       type: 'typing',
       group_id: currentGroupId,
       is_typing: true|false
     }));
     ```
   - Pour envoyer un message : continuer à utiliser l’endpoint REST `POST /api/admin/chat/groups/{id}/messages` qui déclenche automatiquement le WS.

4. **Gestion offline / rechargement**
   - À chaque reconnexion WS, renvoyer `join_group` pour tous les groupes affichés.
   - Rafraîchir la liste via `GET /api/chat/groups` au mount de la page, puis se fier aux WS pour les mises à jour temps réel.

En appliquant ces points, le front offrira exactement la même expérience au créateur et aux autres membres (messages instantanés + typing synchronisé).

