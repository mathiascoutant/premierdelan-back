# 📸 Configuration Cloudinary pour Photo de Profil

Ce guide t'aide à configurer Cloudinary pour permettre aux utilisateurs d'uploader leur photo de profil.

---

## 🎯 Étapes Complètes

### 1️⃣ Créer un Compte Cloudinary

1. Va sur https://cloudinary.com/users/register_free
2. Inscris-toi avec ton email
3. Confirme ton email
4. Tu arrives sur le **Dashboard**

✅ **Avantages** : 25 GB gratuit, pas de carte bancaire requise

---

### 2️⃣ Récupérer tes Identifiants

Dans le **Dashboard Cloudinary**, tu verras :

```
Product Environment Credentials
────────────────────────────────
Cloud Name:  dxxxxxxxxx
API Key:     123456789012345
API Secret:  xxxxxxxxxxxxxxxxxxx
```

📋 **Note ces informations** (tu en auras besoin pour le backend)

---

### 3️⃣ Créer un Upload Preset

1. Dashboard → **Settings** (⚙️) → **Upload**
2. Scroll vers le bas : **Upload presets**
3. Clique **Add upload preset**
4. Configure :

| Paramètre | Valeur |
|-----------|--------|
| **Preset name** | `premierdelan_profiles` |
| **Signing mode** | `Unsigned` |
| **Folder** | `profiles` |
| **Access mode** | `Public` |
| **Allowed formats** | `jpg,png,gif,webp` |

5. **Save**

💡 **Pourquoi Unsigned ?** Permet au backend de faire des uploads directs sans signature supplémentaire.

---

### 4️⃣ Configurer le Backend

#### Option A : Fichier `.env` local (développement)

Crée ou modifie `.env` :

```bash
# Cloudinary
CLOUDINARY_CLOUD_NAME=dxxxxxxxxx
CLOUDINARY_UPLOAD_PRESET=premierdelan_profiles
CLOUDINARY_API_KEY=123456789012345
CLOUDINARY_API_SECRET=xxxxxxxxxxxxxxxxxxx
```

#### Option B : Variables d'environnement Render (production)

Dans ton projet Render :
1. Dashboard → **Environment**
2. Ajoute ces variables :

```
CLOUDINARY_CLOUD_NAME=dxxxxxxxxx
CLOUDINARY_UPLOAD_PRESET=premierdelan_profiles
CLOUDINARY_API_KEY=123456789012345
CLOUDINARY_API_SECRET=xxxxxxxxxxxxxxxxxxx
```

3. **Save Changes** → Render va redéployer automatiquement

---

### 5️⃣ Tester l'Upload

#### Test 1 : Via le script de test

```bash
cd /Users/mathias/Desktop/    /premier\ de\ l\'an/site/back
chmod +x test-profile-upload.sh
./test-profile-upload.sh
```

Le script te demandera :
- Email utilisateur
- Mot de passe
- Chemin vers une image (ex: `~/Desktop/photo.jpg`)

#### Test 2 : Via cURL manuel

```bash
# 1. Se connecter pour obtenir un token
TOKEN=$(curl -s -X POST http://localhost:8090/api/connexion \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password"}' \
  | jq -r '.token')

# 2. Upload de l'image
curl -X POST http://localhost:8090/api/user/profile/image \
  -H "Authorization: Bearer $TOKEN" \
  -F "profileImage=@/chemin/vers/photo.jpg"
```

---

### 6️⃣ Intégration Frontend

Voici un exemple React/Next.js complet :

```tsx
"use client";

import { useState } from "react";

export function ProfileImageUpload() {
  const [uploading, setUploading] = useState(false);
  const [imageUrl, setImageUrl] = useState("");
  const [error, setError] = useState("");

  const handleUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    // Validation taille (5 MB)
    if (file.size > 5 * 1024 * 1024) {
      setError("Le fichier ne doit pas dépasser 5 MB");
      return;
    }

    // Validation type
    const allowedTypes = ["image/jpeg", "image/jpg", "image/png", "image/webp", "image/gif"];
    if (!allowedTypes.includes(file.type)) {
      setError("Format non supporté. Utilisez JPEG, PNG, WebP ou GIF");
      return;
    }

    setUploading(true);
    setError("");

    try {
      const formData = new FormData();
      formData.append("profileImage", file);

      const token = localStorage.getItem("token");
      const response = await fetch("https://your-backend.com/api/user/profile/image", {
        method: "POST",
        headers: {
          Authorization: `Bearer ${token}`,
        },
        body: formData,
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || "Erreur lors de l'upload");
      }

      const data = await response.json();
      setImageUrl(data.imageUrl);
      
      // Mettre à jour le profil dans le state global
      // dispatch(updateProfileImage(data.imageUrl));
      
    } catch (err) {
      setError(err instanceof Error ? err.message : "Erreur inconnue");
    } finally {
      setUploading(false);
    }
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-4">
        {imageUrl && (
          <img
            src={imageUrl}
            alt="Photo de profil"
            className="w-20 h-20 rounded-full object-cover"
          />
        )}
        
        <label className="cursor-pointer bg-blue-500 text-white px-4 py-2 rounded hover:bg-blue-600">
          {uploading ? "Upload en cours..." : "Choisir une photo"}
          <input
            type="file"
            accept="image/jpeg,image/jpg,image/png,image/webp,image/gif"
            onChange={handleUpload}
            disabled={uploading}
            className="hidden"
          />
        </label>
      </div>

      {error && (
        <p className="text-red-500 text-sm">{error}</p>
      )}

      {uploading && (
        <div className="flex items-center gap-2">
          <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-blue-500"></div>
          <span className="text-sm text-gray-600">Upload en cours...</span>
        </div>
      )}
    </div>
  );
}
```

---

## 🔍 Vérification

### ✅ Checklist de Configuration

- [ ] Compte Cloudinary créé
- [ ] Upload preset `premierdelan_profiles` créé
- [ ] Variables d'environnement configurées (`.env` ou Render)
- [ ] Backend redémarré
- [ ] Test d'upload réussi

### 📊 Vérifier dans Cloudinary

1. Dashboard → **Media Library**
2. Tu devrais voir un dossier `profiles/`
3. Dedans, des sous-dossiers par utilisateur (ex: `user_example_com`)
4. Les images sont automatiquement transformées en 400x400

---

## 🎨 Transformations Cloudinary

L'image est automatiquement :
- ✅ Redimensionnée à **400x400 px**
- ✅ Optimisée en **qualité automatique**
- ✅ Convertie au **format optimal** (WebP si supporté)

Tu peux aussi transformer les URLs manuellement :

```
# URL d'origine
https://res.cloudinary.com/dxxx/image/upload/v123/profiles/user/photo.jpg

# Vignette 100x100
https://res.cloudinary.com/dxxx/image/upload/w_100,h_100,c_fill/v123/profiles/user/photo.jpg

# Qualité réduite (économiser de la bande passante)
https://res.cloudinary.com/dxxx/image/upload/q_50/v123/profiles/user/photo.jpg

# Format WebP
https://res.cloudinary.com/dxxx/image/upload/f_webp/v123/profiles/user/photo.jpg
```

---

## 🐛 Dépannage

### Erreur : "Token d'authentification invalide"

➡️ Vérifie que tu es bien connecté et que le token est valide.

### Erreur : "Cloudinary returned status 400"

➡️ Vérifie que :
- Le `CLOUDINARY_CLOUD_NAME` est correct
- L'upload preset `premierdelan_profiles` existe
- Le preset est bien en mode **Unsigned**

### Erreur : "Format de fichier non supporté"

➡️ Seuls JPEG, PNG, WebP et GIF sont acceptés.

### Erreur : "Le fichier ne doit pas dépasser 5 MB"

➡️ Réduis la taille de l'image avant l'upload.

---

## 📚 Documentation Complète

- **API** : Voir `CLOUDINARY_UPLOAD_API.md`
- **Test** : Utilise `test-profile-upload.sh`
- **Cloudinary Docs** : https://cloudinary.com/documentation/image_upload_api_reference

---

## 🎉 C'est Tout !

Ton système d'upload de photo de profil est maintenant **opérationnel** ! 🚀

**Prochaines étapes possibles** :
1. Ajouter la suppression de l'ancienne image lors d'un nouvel upload
2. Ajouter un endpoint pour supprimer la photo de profil
3. Ajouter un crop/resize côté frontend avant l'upload
4. Afficher la photo de profil dans le chat et les listes d'utilisateurs

💡 **Astuce** : Cloudinary offre 25 GB gratuit, largement suffisant pour des centaines de milliers de photos de profil !

