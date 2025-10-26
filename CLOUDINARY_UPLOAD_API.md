# 📸 API Upload Photo de Profil avec Cloudinary

## 🎯 Endpoint

```
POST /api/user/profile/image
```

**Protection** : Authentification requise (JWT Bearer Token)

---

## 📋 Configuration Cloudinary Requise

### Variables d'environnement Backend

```bash
CLOUDINARY_CLOUD_NAME=votre_cloud_name
CLOUDINARY_UPLOAD_PRESET=premierdelan_profiles
CLOUDINARY_API_KEY=votre_api_key
CLOUDINARY_API_SECRET=votre_api_secret
```

### Configuration du Upload Preset dans Cloudinary

1. Va sur [Cloudinary Dashboard](https://console.cloudinary.com/)
2. **Settings** → **Upload**
3. **Add upload preset** :
   - **Preset name** : `premierdelan_profiles`
   - **Signing mode** : `Unsigned` (pour upload direct backend)
   - **Folder** : `profiles` (optionnel)
   - **Access mode** : `Public`
   - **Allowed formats** : `jpg,jpeg,png,webp,gif`
   - **Transformation** : Automatique (resize 400x400)

---

## 📤 Requête

### Headers

```
Authorization: Bearer YOUR_JWT_TOKEN
Content-Type: multipart/form-data
```

### Body (multipart/form-data)

| Champ | Type | Description | Requis |
|-------|------|-------------|--------|
| `profileImage` | File | Image de profil (JPEG, PNG, WebP, GIF) | ✅ Oui |

### Contraintes

- **Taille maximale** : 5 MB
- **Formats acceptés** : JPEG, PNG, WebP, GIF
- **Transformation automatique** : Resize 400x400, qualité auto, format auto

---

## ✅ Réponse en cas de succès

**Status** : `200 OK`

```json
{
  "success": true,
  "message": "Photo de profil mise à jour avec succès",
  "imageUrl": "https://res.cloudinary.com/your_cloud/image/upload/v1234567890/profiles/user_example_com/1234567890.jpg",
  "user": {
    "id": "676ea2a1234567890abcdef1",
    "firstname": "Mathias",
    "lastname": "Coutant",
    "email": "user@example.com",
    "phone": "+33612345678",
    "profileImageUrl": "https://res.cloudinary.com/your_cloud/image/upload/v1234567890/profiles/user_example_com/1234567890.jpg",
    "admin": 0,
    "code_soiree": "CODE123"
  }
}
```

---

## ❌ Réponses d'erreur

### Aucun fichier fourni

**Status** : `400 Bad Request`

```json
{
  "error": "Aucun fichier fourni"
}
```

### Fichier trop volumineux

**Status** : `413 Request Entity Too Large`

```json
{
  "error": "Le fichier ne doit pas dépasser 5 MB"
}
```

### Format non supporté

**Status** : `400 Bad Request`

```json
{
  "error": "Format de fichier non supporté. Formats acceptés : JPEG, PNG, WebP, GIF"
}
```

### Non authentifié

**Status** : `401 Unauthorized`

```json
{
  "error": "Token d'authentification invalide"
}
```

### Erreur serveur

**Status** : `500 Internal Server Error`

```json
{
  "error": "Erreur lors de l'upload de l'image"
}
```

---

## 🧪 Test avec cURL

```bash
# Remplace YOUR_JWT_TOKEN par un vrai token obtenu depuis /api/connexion
curl -X POST \
  http://localhost:8090/api/user/profile/image \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -F "profileImage=@/chemin/vers/photo.jpg"
```

---

## 🌐 Exemple Frontend (React/Next.js)

### Upload avec `fetch`

```typescript
async function uploadProfileImage(file: File) {
  const formData = new FormData();
  formData.append("profileImage", file);

  const response = await fetch("https://your-backend.com/api/user/profile/image", {
    method: "POST",
    headers: {
      Authorization: `Bearer ${localStorage.getItem("token")}`,
    },
    body: formData,
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error || "Erreur lors de l'upload");
  }

  const data = await response.json();
  console.log("Upload réussi:", data.imageUrl);
  return data;
}
```

### Composant d'Upload

```tsx
import { useState } from "react";

export function ProfileImageUpload() {
  const [uploading, setUploading] = useState(false);
  const [imageUrl, setImageUrl] = useState("");

  const handleFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    // Validation taille (5 MB)
    if (file.size > 5 * 1024 * 1024) {
      alert("Le fichier ne doit pas dépasser 5 MB");
      return;
    }

    // Validation type
    const allowedTypes = ["image/jpeg", "image/jpg", "image/png", "image/webp", "image/gif"];
    if (!allowedTypes.includes(file.type)) {
      alert("Format non supporté. Utilisez JPEG, PNG, WebP ou GIF");
      return;
    }

    setUploading(true);

    try {
      const data = await uploadProfileImage(file);
      setImageUrl(data.imageUrl);
      alert("Photo de profil mise à jour avec succès !");
    } catch (error) {
      console.error(error);
      alert("Erreur lors de l'upload");
    } finally {
      setUploading(false);
    }
  };

  return (
    <div>
      <input
        type="file"
        accept="image/jpeg,image/jpg,image/png,image/webp,image/gif"
        onChange={handleFileChange}
        disabled={uploading}
      />
      {uploading && <p>Upload en cours...</p>}
      {imageUrl && <img src={imageUrl} alt="Photo de profil" width="200" />}
    </div>
  );
}
```

---

## 🔄 Workflow Complet

```
1. 👤 Frontend : Utilisateur sélectionne une image

2. 📤 Frontend : Upload vers /api/user/profile/image (avec JWT)

3. 🔍 Backend : Valide le fichier (type, taille)

4. ☁️  Backend : Upload vers Cloudinary
   └─> Transformation automatique (resize 400x400)
   └─> Retourne URL publique

5. 💾 Backend : Met à jour MongoDB (profile_image_url)

6. ✅ Frontend : Reçoit l'URL et met à jour l'UI
```

---

## 🗑️ Suppression d'image (à implémenter)

Pour supprimer l'ancienne image lors d'un nouvel upload, tu peux :

1. **Option A** : Garder l'historique (ne rien supprimer)
2. **Option B** : Supprimer l'ancienne via l'API Cloudinary avant le nouvel upload

### Exemple de suppression (à ajouter dans le handler)

```go
// Avant l'upload, supprimer l'ancienne image si elle existe
if user.ProfileImageURL != "" {
    // Extraire le public_id depuis l'URL
    // Ex: https://res.cloudinary.com/.../profiles/user_example_com/1234567890.jpg
    // public_id = profiles/user_example_com/1234567890
    
    // Appeler l'API Cloudinary Admin pour supprimer
    // DELETE https://api.cloudinary.com/v1_1/{cloud_name}/resources/image/upload
}
```

---

## 📊 Avantages Cloudinary

✅ **25 GB gratuit** (vs 5 GB Firebase)  
✅ **Transformation automatique** (resize, crop, qualité)  
✅ **CDN mondial** inclus  
✅ **Optimisation automatique** (format WebP auto, compression)  
✅ **URLs transformables** :
   - `?w_200,h_200` → Vignette 200x200
   - `?q_auto` → Qualité automatique
   - `?f_auto` → Format automatique (WebP si supporté)

---

## 🎉 C'est Prêt !

L'endpoint est **prêt à l'emploi** ! Configure simplement tes variables d'environnement Cloudinary et tu peux commencer à uploader des photos de profil.

**Prochaines étapes** :
1. Créer un compte Cloudinary
2. Configurer l'upload preset
3. Ajouter les variables d'environnement
4. Redémarrer le backend
5. Tester avec cURL ou depuis le frontend

🚀 **Bon upload !**

