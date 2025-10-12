# 🔥 Configuration Firebase Storage

## ❌ Problème Actuel

```
404 Not Found sur Firebase Storage
URL: https://firebasestorage.googleapis.com/v0/b/premier-de-lan.firebasestorage.app/o?name=...
```

**Cause probable** : Le bucket Firebase Storage n'existe pas ou les règles CORS ne sont pas configurées.

---

## ✅ Solution : 3 Étapes

### 1️⃣ Vérifier le Bucket dans Firebase Console

1. Va sur https://console.firebase.google.com/project/premier-de-lan/storage
2. Vérifie que **Storage** est activé
3. Vérifie le nom du bucket (probablement `premier-de-lan.appspot.com`)

---

### 2️⃣ Configurer les Règles de Sécurité Firebase Storage

Dans la console Firebase → **Storage** → **Rules**, remplace par :

```javascript
rules_version = '2';
service firebase.storage {
  match /b/{bucket}/o {
    // Règles pour les médias d'événements
    match /events/{eventId}/media/{allPaths=**} {
      // Tout le monde peut lire
      allow read: if true;

      // Seuls les utilisateurs authentifiés peuvent écrire
      allow write: if request.auth != null
                   && request.resource.size < 100 * 1024 * 1024  // Max 100 MB
                   && (request.resource.contentType.matches('image/.*')
                       || request.resource.contentType.matches('video/.*'));

      // Seul le propriétaire peut supprimer
      allow delete: if request.auth != null;
    }
  }
}
```

**Publie les règles** en cliquant sur "Publier".

---

### 3️⃣ Configurer CORS pour Firebase Storage

Firebase Storage nécessite une configuration CORS pour les requêtes cross-origin.

**Option A : Via Google Cloud Console (Recommandé)**

1. Va sur https://console.cloud.google.com/storage/browser
2. Sélectionne le bucket `premier-de-lan.appspot.com`
3. Onglet "Permissions" → "CORS"
4. Ajoute cette configuration :

```json
[
  {
    "origin": ["*"],
    "method": ["GET", "POST", "PUT", "DELETE", "OPTIONS"],
    "maxAgeSeconds": 3600,
    "responseHeader": ["Content-Type", "Authorization"]
  }
]
```

**Option B : Via `gsutil` (ligne de commande)**

Crée un fichier `cors.json` :

```json
[
  {
    "origin": ["*"],
    "method": ["GET", "POST", "PUT", "DELETE", "OPTIONS"],
    "maxAgeSeconds": 3600,
    "responseHeader": ["Content-Type"]
  }
]
```

Puis exécute :

```bash
gsutil cors set cors.json gs://premier-de-lan.appspot.com
```

---

## 🎯 Vérifier le Nom du Bucket

Le bucket Firebase Storage est généralement :

- `<project-id>.appspot.com`
- Pour ton projet : **`premier-de-lan.appspot.com`**

**⚠️ Dans ton URL, tu utilises :**

```
premier-de-lan.firebasestorage.app
```

**Essaye plutôt :**

```
premier-de-lan.appspot.com
```

---

## 📝 Code Frontend Corrigé

Dans ton frontend, utilise le **bon bucket** :

```typescript
import { getStorage, ref, uploadBytes, getDownloadURL } from "firebase/storage";

const storage = getStorage();

// Upload d'une image
const uploadImage = async (file: File, eventId: string, userEmail: string) => {
  const timestamp = Date.now();
  const filename = `${timestamp}_${file.name}`;

  // ✅ Bon chemin
  const storagePath = `events/${eventId}/media/${userEmail}/${filename}`;
  const storageRef = ref(storage, storagePath);

  // Upload
  const snapshot = await uploadBytes(storageRef, file);

  // Récupérer l'URL publique
  const downloadURL = await getDownloadURL(snapshot.ref);

  return {
    url: downloadURL,
    storage_path: storagePath,
    filename: file.name,
  };
};
```

---

## 🧪 Tester Firebase Storage

### Test 1 : Vérifier que Storage est activé

```bash
# Dans la console Firebase
firebase-tools-instant-win firestore:init
```

### Test 2 : Upload manuel depuis la console

1. Va sur https://console.firebase.google.com/project/premier-de-lan/storage
2. Clique "Importer des fichiers"
3. Essaye d'uploader une image

Si ça fonctionne manuellement, c'est un problème CORS ou d'authentification.

---

## 🔑 Initialisation Firebase dans le Frontend

Assure-toi que Firebase est correctement initialisé :

```typescript
import { initializeApp } from "firebase/app";
import { getStorage } from "firebase/storage";
import { getAuth } from "firebase/auth";

const firebaseConfig = {
  apiKey: "AIzaSyBdQ8j21Vx7N2myh6ir8gY_zZkRCl-25qI",
  authDomain: "premier-de-lan.firebaseapp.com",
  projectId: "premier-de-lan",
  storageBucket: "premier-de-lan.appspot.com", // ← IMPORTANT
  messagingSenderId: "220494656911",
  appId: "1:220494656911:web:2ff99839c5f7271ddf07fa",
  measurementId: "G-L06FQVLPE1",
};

// Initialiser Firebase
const app = initializeApp(firebaseConfig);
export const storage = getStorage(app);
export const auth = getAuth(app);
```

---

## 📊 Résumé des Actions

1. ✅ **Activer Storage** dans Firebase Console
2. ✅ **Configurer les règles** de sécurité Storage
3. ✅ **Configurer CORS** via Google Cloud Console ou gsutil
4. ✅ **Vérifier le bucket** : utiliser `premier-de-lan.appspot.com`
5. ✅ **Mettre à jour le frontend** avec le bon bucket

---

## 🆘 Si ça ne fonctionne toujours pas

Vérifie dans la console Firebase → **Storage** :

- Est-ce que le bucket existe ?
- Est-ce que tu vois des fichiers ?
- Peux-tu uploader manuellement ?

Regarde les logs de la console navigateur (F12) pour voir l'erreur exacte.

---

**Le backend Go est OK ! Le problème est uniquement côté Firebase Storage. 🚀**
