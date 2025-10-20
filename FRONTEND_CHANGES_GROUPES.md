# 🔄 Changements Frontend - Système de Groupes de Chat

## ⚠️ **CHANGEMENT IMPORTANT : JWT UserID**

### 🔧 **Ce qui a changé dans le backend**

Avant, le JWT stockait l'**ObjectID MongoDB** comme `user_id` :
```json
{
  "user_id": "68f488fdf5a3cc50432df034",
  "email": "mathias@example.com"
}
```

**Maintenant**, le JWT stocke l'**email** comme `user_id` :
```json
{
  "user_id": "mathias@example.com",
  "email": "mathias@example.com"
}
```

### 📝 **Action requise côté frontend**

#### ✅ **Les utilisateurs doivent se reconnecter**

Après le déploiement du backend, **tous les utilisateurs doivent se reconnecter** pour obtenir un nouveau token JWT avec le bon format.

#### 🔍 **Si vous stockez l'user_id localement**

Si votre frontend stocke `user_id` quelque part (localStorage, state, etc.), assurez-vous que c'est cohérent :

**Avant :**
```typescript
const userId = claims.user_id; // "68f488fdf5a3cc50432df034"
```

**Maintenant :**
```typescript
const userId = claims.user_id; // "mathias@example.com"
```

---

## 📋 **API Groupes de Chat - Endpoints disponibles**

### 1️⃣ **Créer un groupe** (Admin uniquement)

```typescript
POST /api/admin/chat/groups
Headers: {
  Authorization: "Bearer YOUR_TOKEN"
}
Body: {
  name: "Équipe Marketing",
  member_ids: ["user1@example.com", "user2@example.com"]
}

Response: {
  success: true,
  message: "Groupe créé avec succès",
  data: {
    id: "group-uuid",
    name: "Équipe Marketing",
    created_by: "admin@example.com",
    created_at: "2025-01-20T10:30:00Z",
    member_count: 3
  }
}
```

---

### 2️⃣ **Récupérer mes groupes**

```typescript
GET /api/admin/chat/groups
// ou pour les non-admins :
GET /api/chat/groups

Response: {
  success: true,
  message: "Groupes récupérés",
  data: {
    groups: [
      {
        id: "group-uuid",
        name: "Équipe Marketing",
        created_by: {
          id: "admin@example.com",
          firstname: "Mathias",
          lastname: "Coutant"
        },
        member_count: 5,
        unread_count: 3,
        last_message: {
          content: "Bonjour tout le monde",
          sender: {
            firstname: "John",
            lastname: "Doe"
          },
          timestamp: "2025-01-20T10:45:00Z"
        },
        is_admin: true,
        joined_at: "2025-01-20T10:30:00Z"
      }
    ]
  }
}
```

---

### 3️⃣ **Récupérer mes invitations en attente**

```typescript
GET /api/admin/chat/group-invitations/pending
// ou pour les non-admins :
GET /api/chat/group-invitations/pending

Response: {
  success: true,
  message: "Invitations récupérées",
  data: {
    invitations: [
      {
        id: "invitation-uuid",
        group: {
          id: "group-uuid",
          name: "Équipe Marketing",
          member_count: 5,
          created_by: {
            firstname: "Mathias",
            lastname: "Coutant"
          }
        },
        invited_by: {
          firstname: "John",
          lastname: "Doe"
        },
        message: "Rejoins-nous !",
        invited_at: "2025-01-20T10:35:00Z"
      }
    ]
  }
}
```

**⚠️ Note :** Si `invitations` est `null`, cela signifie qu'il n'y a aucune invitation en attente.

---

### 4️⃣ **Accepter ou refuser une invitation**

```typescript
PUT /api/admin/chat/group-invitations/{invitation_id}/respond
// ou pour les non-admins :
PUT /api/chat/group-invitations/{invitation_id}/respond

Body: {
  action: "accept"  // ou "reject"
}

Response (si accept): {
  success: true,
  message: "Invitation acceptée",
  data: {
    group: {
      id: "group-uuid",
      name: "Équipe Marketing"
    }
  }
}
```

---

### 5️⃣ **Inviter un utilisateur** (Admin du groupe uniquement)

```typescript
POST /api/admin/chat/groups/{group_id}/invite

Body: {
  user_id: "user@example.com",
  message: "Rejoins-nous !"  // optionnel
}

Response: {
  success: true,
  message: "Invitation envoyée",
  data: {
    id: "invitation-uuid",
    group_id: "group-uuid",
    invited_user: "user@example.com",
    status: "pending",
    invited_at: "2025-01-20T10:35:00Z"
  }
}
```

---

### 6️⃣ **Rechercher des utilisateurs** (pour inviter)

```typescript
GET /api/admin/chat/users/search?q=mathias&limit=10

Response: {
  success: true,
  message: "Utilisateurs trouvés",
  data: {
    users: [
      {
        id: "mathias@example.com",
        firstname: "Mathias",
        lastname: "Coutant",
        email: "mathias@example.com",
        admin: 1  // 1 = admin, 0 = user normal
      }
    ]
  }
}
```

---

### 7️⃣ **Récupérer les membres d'un groupe**

```typescript
GET /api/admin/chat/groups/{group_id}/members
// ou pour les non-admins :
GET /api/chat/groups/{group_id}/members

Response: {
  success: true,
  message: "Membres récupérés",
  data: {
    members: [
      {
        id: "user@example.com",
        firstname: "Mathias",
        lastname: "Coutant",
        email: "mathias@example.com",
        role: "admin",
        joined_at: "2025-01-20T10:30:00Z",
        is_online: true,
        last_seen: "2025-01-20T11:00:00Z"
      }
    ]
  }
}
```

---

### 8️⃣ **Envoyer un message dans un groupe**

```typescript
POST /api/admin/chat/groups/{group_id}/messages
// ou pour les non-admins :
POST /api/chat/groups/{group_id}/messages

Body: {
  content: "Bonjour tout le monde !"
}

Response: {
  success: true,
  message: "Message envoyé",
  data: {
    id: "message-uuid",
    group_id: "group-uuid",
    sender_id: "user@example.com",
    sender: {
      firstname: "Mathias",
      lastname: "Coutant"
    },
    content: "Bonjour tout le monde !",
    message_type: "message",
    created_at: "2025-01-20T11:00:00Z"
  }
}
```

---

### 9️⃣ **Récupérer les messages d'un groupe**

```typescript
GET /api/admin/chat/groups/{group_id}/messages?limit=50&before={message_id}
// ou pour les non-admins :
GET /api/chat/groups/{group_id}/messages?limit=50&before={message_id}

Response: {
  success: true,
  message: "Messages récupérés",
  data: {
    messages: [
      {
        id: "message-uuid",
        sender_id: "user@example.com",
        sender: {
          firstname: "Mathias",
          lastname: "Coutant"
        },
        content: "Bonjour tout le monde !",
        message_type: "message",  // ou "system"
        created_at: "2025-01-20T11:00:00Z"
      }
    ]
  }
}
```

---

### 🔟 **Marquer les messages comme lus**

```typescript
POST /api/admin/chat/groups/{group_id}/mark-read
// ou pour les non-admins :
POST /api/chat/groups/{group_id}/mark-read

Response: {
  success: true,
  message: "Messages marqués comme lus"
}
```

---

## 🔌 **WebSocket - Événements de Groupes**

### **Rejoindre un groupe**

```typescript
ws.send(JSON.stringify({
  type: 'join_group',
  group_id: 'group-uuid'
}));

// Réponse du serveur :
{
  type: 'joined_group',
  group_id: 'group-uuid'
}
```

---

### **Quitter un groupe**

```typescript
ws.send(JSON.stringify({
  type: 'leave_group',
  group_id: 'group-uuid'
}));
```

---

### **Typing indicator (groupe)**

```typescript
ws.send(JSON.stringify({
  type: 'group_typing',
  group_id: 'group-uuid',
  is_typing: true
}));

// Le serveur diffuse aux autres membres :
{
  type: 'group_user_typing',
  group_id: 'group-uuid',
  user_id: 'user@example.com',
  username: 'Mathias Coutant',
  is_typing: true
}
```

---

### **📨 Événements reçus du serveur**

#### **Nouvelle invitation**
```typescript
{
  type: 'group_invitation',
  invitation: {
    id: 'invitation-uuid',
    group: {
      id: 'group-uuid',
      name: 'Équipe Marketing',
      created_by: {
        firstname: 'Mathias',
        lastname: 'Coutant'
      }
    },
    invited_by: {
      firstname: 'John',
      lastname: 'Doe'
    },
    message: 'Rejoins-nous !',
    invited_at: '2025-01-20T10:35:00Z'
  }
}
```

#### **Invitation acceptée** (reçu par l'admin qui a invité)
```typescript
{
  type: 'group_invitation_accepted',
  group_id: 'group-uuid',
  user: {
    id: 'user@example.com',
    firstname: 'Jane',
    lastname: 'Smith'
  },
  accepted_at: '2025-01-20T10:40:00Z'
}
```

#### **Membre a rejoint** (reçu par tous les membres)
```typescript
{
  type: 'group_member_joined',
  group_id: 'group-uuid',
  user: {
    id: 'user@example.com',
    firstname: 'Jane',
    lastname: 'Smith',
    email: 'jane@example.com'
  },
  system_message: {
    id: 'message-uuid',
    content: 'Jane Smith a rejoint le groupe',
    message_type: 'system',
    created_at: '2025-01-20T10:40:00Z'
  }
}
```

#### **Nouveau message**
```typescript
{
  type: 'new_group_message',
  group_id: 'group-uuid',
  message: {
    id: 'message-uuid',
    sender_id: 'user@example.com',
    sender: {
      firstname: 'Mathias',
      lastname: 'Coutant'
    },
    content: 'Bonjour !',
    message_type: 'message',
    created_at: '2025-01-20T11:00:00Z'
  }
}
```

---

## ✅ **Checklist Frontend**

### **Étape 1 : Forcer la reconnexion**
- [ ] Afficher un message demandant aux utilisateurs de se reconnecter
- [ ] Invalider les anciens tokens JWT
- [ ] Rediriger vers la page de login

### **Étape 2 : Implémenter l'UI des groupes**
- [ ] Page liste des groupes
- [ ] Page de création de groupe (admin)
- [ ] Modal d'invitation
- [ ] Badge notifications pour invitations en attente
- [ ] Liste des invitations en attente

### **Étape 3 : Implémenter le chat de groupe**
- [ ] Page de discussion de groupe
- [ ] Affichage des membres
- [ ] Typing indicators
- [ ] Messages système (X a rejoint, etc.)
- [ ] Badge unread count

### **Étape 4 : WebSocket**
- [ ] Écouter `group_invitation`
- [ ] Écouter `group_member_joined`
- [ ] Écouter `new_group_message`
- [ ] Écouter `group_user_typing`
- [ ] Émettre `join_group` au montage
- [ ] Émettre `leave_group` au démontage
- [ ] Émettre `group_typing` pendant la saisie

---

## 🎨 **Exemple d'intégration TypeScript/React**

### **Hook pour les groupes**

```typescript
import { useState, useEffect } from 'react';
import { apiRequest } from './api';

export const useGroups = () => {
  const [groups, setGroups] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchGroups = async () => {
      try {
        const response = await apiRequest('/api/chat/groups', 'GET');
        setGroups(response.data.groups || []);
      } catch (error) {
        console.error('Erreur chargement groupes:', error);
      } finally {
        setLoading(false);
      }
    };

    fetchGroups();
  }, []);

  return { groups, loading };
};
```

### **Hook pour les invitations**

```typescript
export const usePendingInvitations = () => {
  const [invitations, setInvitations] = useState([]);

  useEffect(() => {
    const fetchInvitations = async () => {
      const response = await apiRequest(
        '/api/chat/group-invitations/pending',
        'GET'
      );
      setInvitations(response.data.invitations || []);
    };

    fetchInvitations();
  }, []);

  const acceptInvitation = async (invitationId: string) => {
    await apiRequest(
      `/api/chat/group-invitations/${invitationId}/respond`,
      'PUT',
      { action: 'accept' }
    );
    // Rafraîchir la liste
    const response = await apiRequest(
      '/api/chat/group-invitations/pending',
      'GET'
    );
    setInvitations(response.data.invitations || []);
  };

  return { invitations, acceptInvitation };
};
```

---

## 📱 **Notifications FCM**

Les notifications push sont automatiquement envoyées pour :
- ✅ Nouvelle invitation de groupe
- ✅ Nouveau message dans un groupe (si offline)
- ✅ Membre a rejoint le groupe

**Format des données dans la notification :**
```typescript
{
  type: 'group_invitation' | 'group_message' | 'group_member_joined',
  group_id: 'group-uuid',
  group_name: 'Équipe Marketing',
  // ... autres données selon le type
}
```

---

## 🔥 **Points importants**

1. **Les utilisateurs doivent se reconnecter après le déploiement**
2. **user_id est maintenant un email, pas un ObjectID**
3. **Les invitations peuvent retourner `null` si aucune invitation**
4. **Les messages système ont `message_type: "system"`**
5. **Seuls les admins du groupe peuvent inviter**
6. **Tous les membres peuvent envoyer des messages**

---

**Date:** 2025-10-20  
**Version:** 1.1  
**Backend:** Ready ✅  
**À déployer:** Oui (en cours sur Render)

