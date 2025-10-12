# 📸 Configuration Cloudinary pour la Galerie

## 🎯 Pourquoi Cloudinary ?

- ✅ **25 GB gratuit** (vs 5 GB Firebase)
- ✅ **Pas de carte bancaire** requise
- ✅ **Optimisation automatique** des images
- ✅ **Transformation d'images** (resize, crop, etc.)
- ✅ **Vidéos supportées**

---

## 🚀 Étape 1 : Créer un Compte Cloudinary

1. Va sur https://cloudinary.com/users/register_free
2. Inscris-toi avec ton email (`mathias.coutant.pxcom@gmail.com`)
3. Confirme ton email
4. Tu arrives sur le Dashboard

---

## 🔑 Étape 2 : Récupérer tes Identifiants

Dans le **Dashboard Cloudinary**, note ces infos :

```
Cloud Name: dxxxxxxxx
API Key: 123456789012345
API Secret: xxxxxxxxxxxxxxxxxxxxx
```

**⚠️ Important** : Garde l'API Secret **secret** ! Ne le mets jamais dans le code frontend.

---

## 📋 Étape 3 : Créer un Upload Preset

1. Dashboard → **Settings** (icône ⚙️) → **Upload**
2. Scroll vers **Upload presets**
3. Clique **Add upload preset**
4. Configure :
   - **Preset name** : `premierdelan_events`
   - **Signing mode** : `Unsigned` (important pour upload frontend)
   - **Folder** : `events` (optionnel)
   - **Access mode** : `Public`
   - **Allowed formats** : `jpg,png,gif,webp,mp4,mov,avi,webm`
5. **Save**

---

## 💻 Étape 4 : Code Frontend (React/Next.js)

### Installation

```bash
npm install cloudinary-react
# ou simplement utiliser fetch
```

### Configuration

Crée un fichier `lib/cloudinary.ts` :

```typescript
// lib/cloudinary.ts
export const CLOUDINARY_CONFIG = {
  cloudName: "TON_CLOUD_NAME", // ← À remplacer
  uploadPreset: "premierdelan_events",
  apiKey: "TON_API_KEY", // Optionnel pour l'upload (pas besoin si unsigned preset)
};

export const CLOUDINARY_UPLOAD_URL = `https://api.cloudinary.com/v1_1/${CLOUDINARY_CONFIG.cloudName}/auto/upload`;
```

### Fonction d'Upload

```typescript
// lib/uploadMedia.ts
import { CLOUDINARY_CONFIG, CLOUDINARY_UPLOAD_URL } from "./cloudinary";

interface UploadResult {
  url: string;
  storage_path: string;
  filename: string;
  size: number;
  type: "image" | "video";
}

export async function uploadToCloudinary(
  file: File,
  eventId: string,
  userEmail: string
): Promise<UploadResult> {
  const formData = new FormData();
  formData.append("file", file);
  formData.append("upload_preset", CLOUDINARY_CONFIG.uploadPreset);

  // Définir le dossier de destination
  const folder = `events/${eventId}/media/${userEmail.replace("@", "_")}`;
  formData.append("folder", folder);

  // Ajouter des métadonnées
  formData.append("context", `user_email=${userEmail}|event_id=${eventId}`);

  // Upload vers Cloudinary
  const response = await fetch(CLOUDINARY_UPLOAD_URL, {
    method: "POST",
    body: formData,
  });

  if (!response.ok) {
    throw new Error("Erreur lors de l'upload vers Cloudinary");
  }

  const data = await response.json();

  // Déterminer le type (image ou video)
  const type = data.resource_type === "video" ? "video" : "image";

  return {
    url: data.secure_url, // URL publique HTTPS
    storage_path: data.public_id, // Chemin pour suppression
    filename: file.name,
    size: data.bytes,
    type: type,
  };
}
```

### Fonction de Suppression

```typescript
// lib/deleteMedia.ts
import { CLOUDINARY_CONFIG } from "./cloudinary";

export async function deleteFromCloudinary(publicId: string): Promise<void> {
  // ⚠️ La suppression depuis le frontend nécessite une API route côté backend
  // ou un endpoint backend dédié qui utilise l'API Secret

  // Option A : Appeler ton backend Go
  const response = await fetch(`/api/cloudinary/delete`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ public_id: publicId }),
  });

  if (!response.ok) {
    throw new Error("Erreur lors de la suppression");
  }
}
```

---

## 🎨 Exemple Complet : Upload depuis le Frontend

```typescript
// components/MediaUpload.tsx
import { uploadToCloudinary } from "@/lib/uploadMedia";

export default function MediaUpload({ eventId, userEmail }: Props) {
  const handleFileUpload = async (
    event: React.ChangeEvent<HTMLInputElement>
  ) => {
    const file = event.target.files?.[0];
    if (!file) return;

    try {
      // 1. Upload vers Cloudinary
      console.log("📤 Upload vers Cloudinary...");
      const cloudinaryResult = await uploadToCloudinary(
        file,
        eventId,
        userEmail
      );

      console.log("✅ Upload Cloudinary réussi:", cloudinaryResult.url);

      // 2. Enregistrer les métadonnées dans ton backend Go
      const response = await fetch(
        `https://nia-preinstructive-nola.ngrok-free.dev/api/evenements/${eventId}/medias`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${token}`,
            "ngrok-skip-browser-warning": "true",
          },
          body: JSON.stringify({
            user_email: userEmail,
            type: cloudinaryResult.type,
            url: cloudinaryResult.url,
            storage_path: cloudinaryResult.storage_path,
            filename: cloudinaryResult.filename,
            size: cloudinaryResult.size,
          }),
        }
      );

      if (!response.ok) {
        throw new Error("Erreur lors de l'enregistrement");
      }

      const data = await response.json();
      console.log("✅ Média enregistré:", data);

      // 3. Rafraîchir la galerie
      refreshGallery();
    } catch (error) {
      console.error("❌ Erreur:", error);
      alert("Erreur lors de l'upload");
    }
  };

  return (
    <input type="file" accept="image/*,video/*" onChange={handleFileUpload} />
  );
}
```

---

## 🗑️ Suppression avec Cloudinary

Pour la suppression, tu as **2 options** :

### Option A : Backend Go supprime directement (Recommandé)

Ajoute un endpoint dans ton backend Go :

```go
// handlers/cloudinary_handler.go
func (h *CloudinaryHandler) DeleteFromCloudinary(w http.ResponseWriter, r *http.Request) {
    // Utiliser l'API Cloudinary Admin
    // Nécessite CLOUDINARY_API_SECRET dans .env

    // DELETE depuis Cloudinary
    // Puis DELETE les métadonnées de MongoDB
}
```

### Option B : Frontend supprime, Backend nettoie

1. Frontend supprime via API Cloudinary (requiert signature)
2. Frontend appelle ton backend pour supprimer les métadonnées

---

## 🔐 Variables d'Environnement

### Frontend `.env.local`

```bash
NEXT_PUBLIC_CLOUDINARY_CLOUD_NAME=ton_cloud_name
NEXT_PUBLIC_CLOUDINARY_UPLOAD_PRESET=premierdelan_events
```

### Backend `.env` (pour suppression)

```bash
CLOUDINARY_CLOUD_NAME=ton_cloud_name
CLOUDINARY_API_KEY=123456789012345
CLOUDINARY_API_SECRET=xxxxxxxxxxxxxxxxxxxxx
```

---

## 🧪 Test Manuel

### Test 1 : Upload via Postman/cURL

```bash
curl -X POST \
  https://api.cloudinary.com/v1_1/TON_CLOUD_NAME/image/upload \
  -F "file=@/path/to/photo.jpg" \
  -F "upload_preset=premierdelan_events" \
  -F "folder=events/test123/media/test@example.com"
```

Si ça fonctionne, tu recevras :

```json
{
  "public_id": "events/test123/media/test@example.com/abc123",
  "secure_url": "https://res.cloudinary.com/TON_CLOUD_NAME/image/upload/v1234567890/events/test123/media/test@example.com/abc123.jpg",
  "bytes": 2456789,
  "format": "jpg",
  "resource_type": "image"
}
```

---

## ✅ Workflow Complet

```
1. 📤 Frontend : Upload fichier vers Cloudinary
   └─> Cloudinary retourne URL publique

2. 💾 Frontend : POST /api/evenements/{id}/medias
   └─> Backend Go enregistre métadonnées dans MongoDB

3. 📋 Frontend : Rafraîchit la galerie
   └─> GET /api/evenements/{id}/medias

4. 🗑️ Frontend : Suppression
   └─> Backend Go supprime métadonnées + appelle API Cloudinary
```

---

## 🎉 Avantages Cloudinary

- ✅ **25 GB gratuit** (5x plus que Firebase)
- ✅ **Pas de carte bancaire**
- ✅ **Transformations d'images** gratuites :
  - Resize : `https://res.cloudinary.com/.../w_300,h_300/photo.jpg`
  - Crop : `https://res.cloudinary.com/.../c_fill,w_500,h_500/photo.jpg`
  - Qualité : `https://res.cloudinary.com/.../q_auto/photo.jpg`
- ✅ **CDN mondial** inclus
- ✅ **Vidéos** supportées nativement

---

## 📝 Actions Immédiates

1. Crée un compte sur https://cloudinary.com/users/register_free
2. Note ton **Cloud Name** et crée un **Upload Preset**
3. Mets à jour ton frontend avec le code ci-dessus
4. **Le backend Go est déjà prêt !** ✅

---

**Cloudinary + Backend Go = Solution parfaite ! 🚀**
