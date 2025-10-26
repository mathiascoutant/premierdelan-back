# 📸 Exemples Frontend pour Upload Photo de Profil

## ⚠️ Erreur Courante : Content-Type

**ERREUR** : `request Content-Type isn't multipart/form-data`

**CAUSE** : Le frontend définit explicitement le header `Content-Type`, ce qui empêche le navigateur de définir automatiquement le boundary nécessaire pour `multipart/form-data`.

---

## ✅ Solution : NE PAS définir le Content-Type

Lors de l'envoi de `FormData`, **NE DÉFINIS PAS** le header `Content-Type`. Le navigateur le fera automatiquement avec le bon boundary.

---

## 🎯 Exemples Corrects

### ✅ Fetch (Vanilla JavaScript)

```javascript
// ✅ BON : Pas de Content-Type défini
async function uploadProfileImage(file) {
  const formData = new FormData();
  formData.append("profileImage", file);

  const response = await fetch(
    "https://your-backend.com/api/user/profile/image",
    {
      method: "POST",
      headers: {
        Authorization: `Bearer ${localStorage.getItem("token")}`,
        // ❌ NE PAS AJOUTER: "Content-Type": "multipart/form-data"
      },
      body: formData,
    }
  );

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error);
  }

  return await response.json();
}
```

---

### ✅ Axios

```javascript
import axios from "axios";

// ✅ BON : Axios gère automatiquement le Content-Type
async function uploadProfileImage(file) {
  const formData = new FormData();
  formData.append("profileImage", file);

  const response = await axios.post(
    "https://your-backend.com/api/user/profile/image",
    formData,
    {
      headers: {
        Authorization: `Bearer ${localStorage.getItem("token")}`,
        // ❌ NE PAS AJOUTER: "Content-Type": "multipart/form-data"
      },
    }
  );

  return response.data;
}
```

---

### ✅ React + TypeScript

```tsx
import { useState } from "react";

export function ProfileImageUpload() {
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState("");
  const [imageUrl, setImageUrl] = useState("");

  const handleUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    // Validation
    if (file.size > 5 * 1024 * 1024) {
      setError("Le fichier ne doit pas dépasser 5 MB");
      return;
    }

    const allowedTypes = [
      "image/jpeg",
      "image/jpg",
      "image/png",
      "image/webp",
      "image/gif",
    ];
    if (!allowedTypes.includes(file.type)) {
      setError("Format non supporté");
      return;
    }

    setUploading(true);
    setError("");

    try {
      const formData = new FormData();
      formData.append("profileImage", file);

      const token = localStorage.getItem("token");

      // ✅ BON : Pas de Content-Type défini
      const response = await fetch(
        "https://your-backend.com/api/user/profile/image",
        {
          method: "POST",
          headers: {
            Authorization: `Bearer ${token}`,
          },
          body: formData,
        }
      );

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || "Erreur upload");
      }

      const data = await response.json();
      setImageUrl(data.imageUrl);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Erreur inconnue");
    } finally {
      setUploading(false);
    }
  };

  return (
    <div>
      <input
        type="file"
        accept="image/jpeg,image/jpg,image/png,image/webp,image/gif"
        onChange={handleUpload}
        disabled={uploading}
      />
      {error && <p style={{ color: "red" }}>{error}</p>}
      {uploading && <p>Upload en cours...</p>}
      {imageUrl && <img src={imageUrl} alt="Profil" width="200" />}
    </div>
  );
}
```

---

### ✅ Next.js App Router

```tsx
"use client";

import { useState } from "react";

export function ProfileImageUpload() {
  const [uploading, setUploading] = useState(false);

  const handleUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    setUploading(true);

    try {
      const formData = new FormData();
      formData.append("profileImage", file);

      // ✅ BON : Utiliser fetch sans Content-Type
      const response = await fetch("/api/user/profile/image", {
        method: "POST",
        headers: {
          Authorization: `Bearer ${localStorage.getItem("token")}`,
        },
        body: formData,
      });

      const data = await response.json();

      if (!response.ok) {
        throw new Error(data.error);
      }

      console.log("Upload réussi:", data.imageUrl);
    } catch (error) {
      console.error("Erreur upload:", error);
    } finally {
      setUploading(false);
    }
  };

  return (
    <label>
      <input type="file" accept="image/*" onChange={handleUpload} />
      {uploading && "Upload en cours..."}
    </label>
  );
}
```

---

## ❌ Exemples INCORRECTS (à éviter)

### ❌ Définir Content-Type manuellement

```javascript
// ❌ MAUVAIS : Content-Type défini manuellement
const response = await fetch(url, {
  method: "POST",
  headers: {
    Authorization: `Bearer ${token}`,
    "Content-Type": "multipart/form-data", // ❌ ERREUR !
  },
  body: formData,
});
```

**Pourquoi c'est mal ?**  
Le navigateur a besoin d'ajouter un `boundary` au `Content-Type` pour séparer les différentes parties du formulaire :

```
Content-Type: multipart/form-data; boundary=----WebKitFormBoundary7MA4YWxkTrZu0gW
```

Si tu définis manuellement `Content-Type`, le boundary n'est pas ajouté et le serveur ne peut pas parser le formulaire.

---

### ❌ Envoyer JSON au lieu de FormData

```javascript
// ❌ MAUVAIS : Envoyer le fichier en JSON
const response = await fetch(url, {
  method: "POST",
  headers: {
    "Content-Type": "application/json",
    Authorization: `Bearer ${token}`,
  },
  body: JSON.stringify({ profileImage: file }), // ❌ ERREUR !
});
```

**Pourquoi c'est mal ?**  
Un fichier ne peut pas être sérialisé en JSON. Tu dois utiliser `FormData`.

---

### ❌ Oublier d'append le fichier

```javascript
// ❌ MAUVAIS : FormData vide
const formData = new FormData();
// Oubli d'ajouter le fichier !

const response = await fetch(url, {
  method: "POST",
  body: formData,
});
```

**Correction** : Ajoute le fichier avec `formData.append("profileImage", file)`.

---

## 🔍 Debugging

### Vérifier le Content-Type dans le navigateur

1. Ouvre les **DevTools** (F12)
2. Va dans **Network**
3. Déclenche l'upload
4. Clique sur la requête `POST /api/user/profile/image`
5. Regarde les **Headers** :

**✅ Bon** :

```
Content-Type: multipart/form-data; boundary=----WebKitFormBoundary7MA4YWxkTrZu0gW
```

**❌ Mauvais** :

```
Content-Type: multipart/form-data
```

(Pas de boundary !)

---

### Vérifier le FormData

```javascript
// Afficher le contenu du FormData
const formData = new FormData();
formData.append("profileImage", file);

for (let [key, value] of formData.entries()) {
  console.log(key, value);
}
// Devrait afficher: profileImage File { name: "photo.jpg", ... }
```

---

## 📚 Résumé

| ✅ À FAIRE                                   | ❌ À ÉVITER                         |
| -------------------------------------------- | ----------------------------------- |
| Utiliser `FormData`                          | Envoyer du JSON                     |
| Laisser le navigateur définir `Content-Type` | Définir `Content-Type` manuellement |
| `formData.append("profileImage", file)`      | Oublier d'append le fichier         |
| Vérifier la taille du fichier (< 5 MB)       | Envoyer des fichiers trop gros      |
| Vérifier le type MIME                        | Envoyer des formats non supportés   |

---

## 🎉 C'est Prêt !

Si tu suis ces exemples, l'upload fonctionnera parfaitement ! 🚀

**Prochaines étapes** :

1. Copie un des exemples ci-dessus
2. Adapte l'URL du backend
3. Teste l'upload
4. Affiche l'image uploadée dans ton UI

💡 **Astuce** : Si tu utilises un framework avec un client HTTP custom (Axios, SWR, React Query), assure-toi qu'il ne définit pas automatiquement le `Content-Type` pour les `FormData`.
