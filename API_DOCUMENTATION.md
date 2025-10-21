# 📚 Documentation API Backend - Premier de l'An

**URL Production** : `https://premierdelan-back.onrender.com`  
**WebSocket** : `wss://premierdelan-back.onrender.com/ws/chat`

---

## 🔐 Authentification

Tous les endpoints protégés nécessitent un header :

```
Authorization: Bearer <JWT_TOKEN>
```

---

## 🎭 Événements

### **GET /api/evenements/public**

Liste publique des événements (pas d'auth requise)

**Response** :

```json
{
  "evenements": [...]
}
```

### **GET /api/evenements/:id**

Détails d'un événement (pas d'auth requise)

**Response** :

```json
{
  "success": true,
  "evenement": {
      "id": "...",
      "titre": "Soirée du Nouvel An",
      "date": "2025-12-31T20:00:00Z",
      "heure": "20h00",
      "lieu": "Chamouillac",
      "description": "...",
      "capacite": 100,
      "inscrits": 45,
      "statut": "ouvert",
      "date_ouverture_inscription": "...",
      "date_fermeture_inscription": "..."
    }
  }
}
```

---

## 📝 Inscriptions aux Événements

### **POST /api/evenements/:id/inscription**

Inscription à un événement (auth requise)

**Body** :

```json
{
  "user_email": "user@example.com",
  "nombre_personnes": 3,
  "accompagnants": [
    {
      "firstname": "Marie",
      "lastname": "DUPONT",
      "is_adult": true
    }
  ]
}
```

**Response** :

```json
{
  "success": true,
  "message": "Inscription confirmée",
  "inscription_id": "..."
}
```

### **GET /api/evenements/:id/inscription**

**Alias** : `GET /api/evenements/:id/inscription/status`

Vérifie si l'utilisateur connecté est inscrit (utilise JWT automatiquement)

**Response** :

```json
{
  "success": true,
  "inscription": {
    "id": "...",
    "event_id": "...",
    "user_email": "...",
    "nombre_personnes": 3,
    "accompagnants": [...],
    "status": "confirmed",
    "registered_at": "..."
  }
}
```

**Ou si non inscrit** :

```json
{
  "success": true,
  "inscription": null
}
```

### **PUT /api/evenements/:id/inscription**

Modifier son inscription (auth requise)

### **DELETE /api/evenements/:id/desinscription**

Se désinscrire (auth requise)

### **GET /api/mes-evenements**

**Alias** : `GET /api/users/me/inscriptions`

Liste des événements auxquels je suis inscrit (auth requise)

**Response** :

```json
{
  "success": true,
  "inscriptions": [
      {
        "id": "...",
        "event_id": "...",
        "user_email": "...",
        "nombre_personnes": 3,
        "accompagnants": [...],
        "status": "confirmed",
        "registered_at": "...",
        "event": {
          "id": "...",
          "titre": "...",
          "date": "...",
          "lieu": "...",
          "description": "..."
        }
      }
    ]
  }
}
```

---

## 📸 Galerie / Médias

### **GET /api/evenements/:id/medias**

Liste des médias d'un événement (pas d'auth requise)

**Response** :

```json
{
  "success": true,
  "photos": [
    {
      "id": "...",
      "url": "https://...",
      "description": "...",
      "uploaded_at": "..."
    }
  ]
}
```

### **POST /api/evenements/:id/medias**

Upload un média (auth requise)

**Body** :

```json
{
  "user_email": "...",
  "type": "image",
  "url": "https://res.cloudinary.com/...",
  "storage_path": "...",
  "filename": "photo.jpg",
  "size": 2048576
}
```

### **DELETE /api/evenements/:id/medias/:mediaId**

Supprimer un média (auth requise)

---

## 💬 Chat Privé (Conversations 1-to-1)

### **GET /api/chat/conversations**

Liste des conversations (auth requise, admin uniquement)

**Response** :

```json
{
  "success": true,
  "data": {
    "conversations": [
      {
        "id": "...",
        "participant": {
          "id": "...",
          "firstname": "...",
          "lastname": "...",
          "email": "...",
          "is_online": true,
          "last_seen": "..."
        },
        "last_message": {
          "content": "...",
          "timestamp": "...",
          "is_read": false
        },
        "status": "accepted",
        "unread_count": 3
      }
    ]
  }
}
```

### **GET /api/chat/conversations/:id/messages**

Messages d'une conversation (auth requise)

**Query params** :

- `limit` : nombre de messages (défaut: 50)

**Response** :

```json
{
  "success": true,
  "data": {
    "messages": [
      {
        "id": "...",
        "conversation_id": "...",
        "sender_id": "...",
        "content": "...",
        "timestamp": "...",
        "delivered_at": "...",
        "read_at": "...",
        "is_read": true
      }
    ]
  }
}
```

### **POST /api/chat/conversations/:id/messages**

Envoyer un message (auth requise)

**Body** :

```json
{
  "content": "Bonjour !"
}
```

### **POST /api/chat/conversations/:id/mark-read**

Marquer les messages comme lus (auth requise)

### **POST /api/chat/invitations**

Inviter un admin au chat (auth requise)

**Body** :

```json
{
  "toUserId": "admin_email@example.com"
}
```

### **GET /api/chat/invitations**

Mes invitations reçues (auth requise)

### **PUT /api/chat/invitations/:id/respond**

Accepter/refuser une invitation (auth requise)

**Body** :

```json
{
  "action": "accept"
}
```

### **GET /api/chat/search**

Rechercher des admins (auth requise)

**Query params** :

- `q` : terme de recherche
- `limit` : nombre de résultats (défaut: 10)

---

## 👥 Groupes de Chat

### **POST /api/chat/groups**

Créer un groupe (auth requise)

**Body** :

```json
{
  "name": "Équipe Dev",
  "member_ids": ["user1@example.com", "user2@example.com"]
}
```

### **GET /api/chat/groups**

Liste de mes groupes (auth requise)

**Response** :

```json
{
  "success": true,
  "data": {
    "groups": [
      {
        "id": "...",
        "name": "...",
        "created_by": {
          "id": "...",
          "firstname": "...",
          "lastname": "...",
          "email": "..."
        },
        "member_count": 5,
        "unread_count": 3,
        "last_message": {
          "content": "...",
          "sender_name": "...",
          "timestamp": "...",
          "created_at": "..."
        },
        "created_at": "..."
      }
    ]
  }
}
```

### **POST /api/chat/groups/:id/invite**

Inviter un membre (auth requise, tous les membres peuvent inviter)

**Body** :

```json
{
  "user_id": "user@example.com"
}
```

**⚠️ Important** : `user_id` doit être un **EMAIL**, pas un ObjectID !

### **GET /api/chat/groups/:id/members**

Liste des membres (auth requise)

**Response** :

```json
{
  "success": true,
  "data": {
    "members": [
      {
        "id": "...",
        "firstname": "...",
        "lastname": "...",
        "email": "...",
        "role": "admin",
        "is_online": true,
        "joined_at": "..."
      }
    ]
  }
}
```

### **POST /api/chat/groups/:id/leave**

Quitter un groupe (auth requise)

### **GET /api/chat/groups/:id/invitations/pending**

**Alias** : `/api/chat/groups/:id/pending-invitations`

Invitations en attente du groupe (auth requise, tous les membres)

### **GET /api/chat/groups/:id/messages**

Messages du groupe (auth requise)

**Query params** :

- `limit` : nombre de messages (défaut: 50)
- `before` : ID pour pagination

**Response** :

```json
{
  "success": true,
  "data": {
    "messages": [
      {
        "id": "...",
        "sender_id": "...",
        "sender": {
          "id": "...",
          "firstname": "...",
          "lastname": "..."
        },
        "content": "...",
        "message_type": "message",
        "timestamp": "...",
        "created_at": "...",
        "delivered_at": "...",
        "read_by": ["email1@...", "email2@..."]
      }
    ]
  }
}
```

### **POST /api/chat/groups/:id/messages**

Envoyer un message dans le groupe (auth requise)

**Body** :

```json
{
  "content": "Bonjour à tous !"
}
```

### **POST /api/chat/groups/:id/mark-read**

Marquer les messages comme lus (auth requise)

### **GET /api/chat/group-invitations/pending**

Mes invitations de groupe reçues (auth requise)

**Response** :

```json
{
  "success": true,
  "data": {
    "invitations": [
      {
        "id": "...",
        "group": {
          "id": "...",
          "name": "...",
          "member_count": 5,
          "created_by": {...}
        },
        "invited_by": {
          "id": "...",
          "firstname": "...",
          "lastname": "..."
        },
        "message": "...",
        "invited_at": "..."
      }
    ]
  }
}
```

### **PUT /api/chat/group-invitations/:id/respond**

Accepter/refuser une invitation de groupe (auth requise)

**Body** :

```json
{
  "action": "accept"
}
```

### **DELETE /api/chat/group-invitations/:id/cancel**

Annuler une invitation envoyée (auth requise)

### **GET /api/chat/users/search**

Rechercher des utilisateurs pour inviter (auth requise)

**Query params** :

- `q` : terme de recherche (min 2 caractères)
- `limit` : nombre de résultats (défaut: 10)

---

## 🔌 WebSocket

### **Connexion**

```
wss://premierdelan-back.onrender.com/ws/chat
```

**Authentification** :

```json
{
  "type": "authenticate",
  "token": "your_jwt_token"
}
```

**Response** :

```json
{
  "type": "authenticated",
  "user_id": "user@example.com"
}
```

### **Événements WebSocket**

#### **Conversations Privées**

**`new_message`** - Nouveau message privé

```json
{
  "type": "new_message",
  "conversation_id": "...",
  "message": {...}
}
```

**`messages_read`** - Messages marqués comme lus

```json
{
  "type": "messages_read",
  "conversation_id": "...",
  "read_by_user_id": "...",
  "read_at": "..."
}
```

**`new_invitation`** - Nouvelle invitation de chat

```json
{
  "type": "new_invitation",
  "invitation": {...}
}
```

**`invitation_accepted`** / **`invitation_rejected`**

**`user_presence`** - Statut en ligne/hors ligne

```json
{
  "type": "user_presence",
  "user_id": "...",
  "is_online": true,
  "last_seen": "..."
}
```

**`user_typing`** - Indicateur de frappe

```json
{
  "type": "user_typing",
  "conversation_id": "...",
  "user_id": "...",
  "username": "Mathias",
  "is_typing": true
}
```

#### **Groupes de Chat**

**`new_group_message`** - Nouveau message groupe

```json
{
  "type": "new_group_message",
  "group_id": "...",
  "message": {
    "id": "...",
    "sender_id": "...",
    "sender": {...},
    "content": "...",
    "timestamp": "...",
    "read_by": [...]
  }
}
```

**`group_invitation`** - Invitation de groupe

```json
{
  "type": "group_invitation",
  "invitation": {...}
}
```

**`group_invitation_accepted`** / **`group_invitation_rejected`**

**`group_member_joined`** - Membre a rejoint le groupe

```json
{
  "type": "group_member_joined",
  "group_id": "...",
  "user": {...},
  "system_message": {...}
}
```

**`group_member_left`** - Membre a quitté le groupe

```json
{
  "type": "group_member_left",
  "group_id": "...",
  "user_id": "...",
  "user_name": "...",
  "message": {...}
}
```

**`group_messages_read`** - Messages marqués comme lus

```json
{
  "type": "group_messages_read",
  "group_id": "...",
  "user_id": "...",
  "read_at": "..."
}
```

**`group_user_typing`** - Indicateur de frappe groupe

```json
{
  "type": "group_user_typing",
  "group_id": "...",
  "user_id": "...",
  "username": "...",
  "is_typing": true
}
```

### **Événements Client → Serveur**

**`join_conversation`** - Rejoindre une conversation

```json
{
  "type": "join_conversation",
  "conversation_id": "..."
}
```

**`leave_conversation`** - Quitter une conversation

```json
{
  "type": "leave_conversation",
  "conversation_id": "..."
}
```

**`typing`** - Indicateur de frappe (conversation)

```json
{
  "type": "typing",
  "conversation_id": "...",
  "is_typing": true
}
```

**`join_group`** - Rejoindre un groupe

```json
{
  "type": "join_group",
  "group_id": "..."
}
```

**`leave_group`** - Quitter un groupe

```json
{
  "type": "leave_group",
  "group_id": "..."
}
```

**`group_typing`** - Indicateur de frappe (groupe)

```json
{
  "type": "group_typing",
  "group_id": "...",
  "is_typing": true
}
```

---

## 📱 Notifications Push (FCM)

Tous les événements WebSocket importants déclenchent aussi une notification FCM si l'utilisateur n'est pas connecté.

**Types de notifications** :

- `chat_message` - Nouveau message privé
- `chat_invitation` - Invitation de chat
- `group_message` - Nouveau message groupe
- `group_invitation` - Invitation de groupe

**Format FCM** :

```json
{
  "notification": {
    "title": "Mathias COUTANT",
    "body": "Salut !"
  },
  "data": {
    "type": "chat_message",
    "conversationId": "...",
    "messageId": "...",
    "senderId": "...",
    "senderName": "..."
  }
}
```

---

## ⚙️ Configuration

### **Variables d'Environnement Render**

| Variable                    | Valeur                                                   | Description                    |
| --------------------------- | -------------------------------------------------------- | ------------------------------ |
| `MONGO_URI`                 | `mongodb+srv://...`                                      | MongoDB Atlas                  |
| `JWT_SECRET`                | `your_secret_key`                                        | Clé secrète JWT                |
| `CORS_ALLOWED_ORIGINS`      | `https://mathiascoutant.github.io,http://localhost:3000` | CORS                           |
| `FIREBASE_CREDENTIALS_JSON` | `{...}`                                                  | Credentials Firebase (JSON)    |
| `PORT`                      | `8090`                                                   | Port serveur (auto par Render) |

---

## 🎯 Points Importants

### **IDs Utilisateur**

⚠️ **CRITIQUE** : Dans le système de groupes, **tous les `user_id` sont des EMAILS**, pas des ObjectID !

**Exemples** :

- `created_by`: `"test3@gmail.com"` ✅
- `invited_by`: `"admin@gmail.com"` ✅
- `sender_id`: `"mathiascoutant@icloud.com"` ✅

### **Badge Messages Non Lus (Groupes)**

**Logique WhatsApp** :

- Si le **dernier message du groupe** est de toi → `unread_count = 0` (pas de badge)
- Sinon, vérifie si tu as marqué comme lu (via `read_receipt`)
- Compte les nouveaux messages après ton dernier lu

### **Format Réponses**

Tous les endpoints utilisent maintenant :

```json
{
  "success": true,
  "message": "...",  // optionnel
  "data": {...}
}
```

Ou en cas d'erreur :

```json
{
  "error": "Forbidden",
  "message": "..."
}
```

---

## 🚀 Déploiement

**Branche** : `main`  
**Auto-deploy** : Chaque push sur `main` déclenche un redéploiement Render (1-2 min)

**Logs en temps réel** :

```
https://dashboard.render.com → premierdelan-back → Logs
```

---

## 📊 Collections MongoDB

- `users` - Utilisateurs
- `events` - Événements
- `inscriptions` - Inscriptions aux événements
- `medias` - Galerie photos/vidéos
- `conversations` - Conversations privées
- `messages` - Messages privés
- `chat_invitations` - Invitations de chat
- `chat_groups` - Groupes de chat
- `chat_group_members` - Membres des groupes
- `chat_group_messages` - Messages de groupe
- `chat_group_invitations` - Invitations de groupe
- `chat_group_read_receipts` - Accusés de lecture groupe
- `fcm_tokens` - Tokens FCM pour notifications
- `site_settings` - Paramètres globaux (thème)

---

**Dernière mise à jour** : 21 octobre 2025  
**Version Backend** : v2.0 (Groupes + Read Receipts + Typing Indicator)
