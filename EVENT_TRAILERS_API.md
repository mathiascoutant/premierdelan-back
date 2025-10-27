# 🎬 API Gestion des Trailers Vidéo d'Événements

## 📋 Vue d'ensemble

Cette API permet aux administrateurs de gérer les trailers vidéo (bandes d'annonce) des événements via Cloudinary.

### Règles métier

- ✅ **Un seul trailer par événement** : Maximum 1 vidéo par événement
- ✅ **Trailer optionnel** : Un événement peut exister sans trailer
- ✅ **Format vidéo uniquement** : MP4, MOV, AVI, WebM
- ✅ **Taille maximale** : 100 MB
- ✅ **Admin uniquement** : Toutes les opérations sont réservées aux administrateurs

---

## 🔧 Configuration Cloudinary

### Credentials utilisés

```
Cloud Name: dxwhngg8g
API Key: 272996567138936
API Secret: 07tjhLw1jDxG3GGyuvO7wezUZEI
```

### ⚠️ Créer le Upload Preset pour Vidéos

**Important** : Tu dois créer un preset spécifique pour les vidéos :

1. Va sur [Cloudinary Dashboard](https://console.cloudinary.com/)
2. **Settings** → **Upload**
3. **Add upload preset** :
   - **Preset name** : `premierdelan_events` (ou crée `premierdelan_trailers`)
   - **Signing mode** : **Unsigned**
   - **Folder** : `event_trailers`
   - **Resource type** : **Video**
   - **Allowed formats** : `mp4,mov,avi,webm`
   - **Access mode** : **Public**
4. **Save**

---

## 📡 Endpoints

### 1. Ajouter un Trailer (POST)

**Endpoint** : `POST /api/admin/evenements/:eventId/trailer`

**Permissions** : Admin uniquement

**Headers** :
```
Authorization: Bearer <JWT_TOKEN>
Content-Type: multipart/form-data
```

**Body** (FormData) :
```
video: <File> (fichier vidéo)
```

**Validations** :
- L'événement doit exister
- L'événement ne doit **pas déjà avoir** un trailer
- Fichier vidéo requis (mp4, mov, avi, webm)
- Taille maximale : 100 MB

**Réponse succès** (200) :
```json
{
  "success": true,
  "message": "Trailer ajouté avec succès",
  "trailer": {
    "url": "https://res.cloudinary.com/dxwhngg8g/video/upload/v1234567890/event_trailers/abc123/1234567890.mp4",
    "public_id": "event_trailers/abc123/1234567890",
    "duration": 45.5,
    "format": "mp4",
    "size": 15728640,
    "uploaded_at": "2025-10-27T10:30:00Z",
    "thumbnail_url": "https://res.cloudinary.com/dxwhngg8g/video/upload/so_0/v1234567890/event_trailers/abc123/1234567890.jpg"
  }
}
```

**Réponses erreur** :

```json
// 400 - Événement a déjà un trailer
{
  "success": false,
  "message": "Cet événement a déjà un trailer. Veuillez le supprimer avant d'en ajouter un nouveau."
}

// 413 - Fichier trop volumineux
{
  "success": false,
  "message": "Le fichier ne doit pas dépasser 100 MB"
}

// 400 - Format non supporté
{
  "success": false,
  "message": "Format de vidéo non supporté. Formats acceptés : MP4, MOV, AVI, WebM"
}
```

---

### 2. Remplacer un Trailer (PUT)

**Endpoint** : `PUT /api/admin/evenements/:eventId/trailer`

**Permissions** : Admin uniquement

**Headers** :
```
Authorization: Bearer <JWT_TOKEN>
Content-Type: multipart/form-data
```

**Body** (FormData) :
```
video: <File> (nouveau fichier vidéo)
```

**Validations** :
- L'événement doit exister
- L'événement doit **déjà avoir** un trailer
- Fichier vidéo requis
- Taille maximale : 100 MB

**Processus** :
1. Upload de la nouvelle vidéo vers Cloudinary
2. Suppression de l'ancienne vidéo de Cloudinary
3. Mise à jour de la base de données

**Réponse succès** (200) :
```json
{
  "success": true,
  "message": "Trailer remplacé avec succès",
  "trailer": {
    "url": "https://res.cloudinary.com/dxwhngg8g/video/upload/v1234567890/event_trailers/xyz789/1234567890.mp4",
    "public_id": "event_trailers/xyz789/1234567890",
    "duration": 52.3,
    "format": "mp4",
    "size": 18874368,
    "uploaded_at": "2025-10-27T11:00:00Z",
    "thumbnail_url": "https://res.cloudinary.com/dxwhngg8g/video/upload/so_0/v1234567890/event_trailers/xyz789/1234567890.jpg"
  }
}
```

**Réponse erreur** (404) :
```json
{
  "success": false,
  "message": "Cet événement n'a pas de trailer à remplacer."
}
```

---

### 3. Supprimer un Trailer (DELETE)

**Endpoint** : `DELETE /api/admin/evenements/:eventId/trailer`

**Permissions** : Admin uniquement

**Headers** :
```
Authorization: Bearer <JWT_TOKEN>
```

**Validations** :
- L'événement doit exister
- L'événement doit avoir un trailer

**Processus** :
1. Suppression de la vidéo de Cloudinary
2. Suppression du champ `trailer` dans MongoDB

**Réponse succès** (200) :
```json
{
  "success": true,
  "message": "Trailer supprimé avec succès"
}
```

**Réponse erreur** (404) :
```json
{
  "success": false,
  "message": "Cet événement n'a pas de trailer à supprimer."
}
```

---

### 4. Récupérer un Événement avec Trailer

Les endpoints GET existants retournent automatiquement le champ `trailer` :

**Endpoint** : `GET /api/evenements/public` (liste publique)

**Endpoint** : `GET /api/evenements/:eventId` (détails publics)

**Endpoint** : `GET /api/admin/evenements` (liste admin)

**Endpoint** : `GET /api/admin/evenements/:eventId` (détails admin)

**Exemple de réponse** :
```json
{
  "success": true,
  "evenement": {
    "id": "676ea2a1234567890abcdef1",
    "titre": "Soirée du Nouvel An 2026",
    "date": "2026-01-01T00:00:00",
    "description": "...",
    "capacite": 100,
    "inscrits": 45,
    "statut": "ouvert",
    "lieu": "Paris",
    "code_soiree": "NYE2026",
    "trailer": {
      "url": "https://res.cloudinary.com/dxwhngg8g/video/upload/v1234567890/event_trailers/abc123/1234567890.mp4",
      "public_id": "event_trailers/abc123/1234567890",
      "duration": 45.5,
      "format": "mp4",
      "size": 15728640,
      "uploaded_at": "2025-10-27T10:30:00Z",
      "thumbnail_url": "https://res.cloudinary.com/dxwhngg8g/video/upload/so_0/v1234567890/event_trailers/abc123/1234567890.jpg"
    },
    "created_at": "2025-10-20T10:00:00Z",
    "updated_at": "2025-10-27T10:30:00Z"
  }
}
```

**Si pas de trailer** :
```json
{
  "trailer": null
}
```

---

## 🧪 Tests avec cURL

### Upload d'un trailer

```bash
# 1. Se connecter pour obtenir un token admin
TOKEN=$(curl -s -X POST http://localhost:8090/api/connexion \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@gmail.com","password":"password"}' \
  | jq -r '.token')

# 2. Upload du trailer
curl -X POST http://localhost:8090/api/admin/evenements/676ea2a1234567890abcdef1/trailer \
  -H "Authorization: Bearer $TOKEN" \
  -F "video=@/chemin/vers/trailer.mp4"
```

### Remplacement d'un trailer

```bash
curl -X PUT http://localhost:8090/api/admin/evenements/676ea2a1234567890abcdef1/trailer \
  -H "Authorization: Bearer $TOKEN" \
  -F "video=@/chemin/vers/nouveau-trailer.mp4"
```

### Suppression d'un trailer

```bash
curl -X DELETE http://localhost:8090/api/admin/evenements/676ea2a1234567890abcdef1/trailer \
  -H "Authorization: Bearer $TOKEN"
```

---

## 🎨 Exemple Frontend (React/TypeScript)

### Upload de Trailer

```tsx
async function uploadTrailer(eventId: string, videoFile: File) {
  const formData = new FormData();
  formData.append("video", videoFile);

  const token = localStorage.getItem("token");
  
  const response = await fetch(`https://your-backend.com/api/admin/evenements/${eventId}/trailer`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${token}`,
    },
    body: formData,
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.message || "Erreur lors de l'upload");
  }

  return await response.json();
}
```

### Composant d'Upload

```tsx
import { useState } from "react";

interface Props {
  eventId: string;
  existingTrailer?: EventTrailer | null;
  onSuccess: () => void;
}

export function TrailerUpload({ eventId, existingTrailer, onSuccess }: Props) {
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState("");

  const handleUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    // Validation taille (100 MB)
    if (file.size > 100 * 1024 * 1024) {
      setError("Le fichier ne doit pas dépasser 100 MB");
      return;
    }

    // Validation type
    const allowedTypes = ["video/mp4", "video/quicktime", "video/x-msvideo", "video/webm"];
    if (!allowedTypes.includes(file.type)) {
      setError("Format non supporté. Utilisez MP4, MOV, AVI ou WebM");
      return;
    }

    setUploading(true);
    setError("");

    try {
      const method = existingTrailer ? "PUT" : "POST";
      const formData = new FormData();
      formData.append("video", file);

      const token = localStorage.getItem("token");
      const response = await fetch(
        `https://your-backend.com/api/admin/evenements/${eventId}/trailer`,
        {
          method,
          headers: {
            Authorization: `Bearer ${token}`,
          },
          body: formData,
        }
      );

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.message || "Erreur");
      }

      onSuccess();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Erreur inconnue");
    } finally {
      setUploading(false);
    }
  };

  const handleDelete = async () => {
    if (!existingTrailer || !confirm("Supprimer le trailer ?")) return;

    setUploading(true);
    
    try {
      const token = localStorage.getItem("token");
      const response = await fetch(
        `https://your-backend.com/api/admin/evenements/${eventId}/trailer`,
        {
          method: "DELETE",
          headers: {
            Authorization: `Bearer ${token}`,
          },
        }
      );

      if (!response.ok) {
        throw new Error("Erreur lors de la suppression");
      }

      onSuccess();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Erreur inconnue");
    } finally {
      setUploading(false);
    }
  };

  return (
    <div>
      {existingTrailer && (
        <div>
          <video src={existingTrailer.url} controls width="400" />
          <button onClick={handleDelete} disabled={uploading}>
            Supprimer le trailer
          </button>
        </div>
      )}

      <input
        type="file"
        accept="video/mp4,video/quicktime,video/x-msvideo,video/webm"
        onChange={handleUpload}
        disabled={uploading}
      />

      {uploading && <p>Upload en cours...</p>}
      {error && <p style={{ color: "red" }}>{error}</p>}
    </div>
  );
}
```

---

## 📊 Structure MongoDB

### Collection `evenements`

```javascript
{
  "_id": ObjectId("676ea2a1234567890abcdef1"),
  "titre": "Soirée du Nouvel An 2026",
  "date": ISODate("2026-01-01T00:00:00Z"),
  "description": "...",
  // ... autres champs ...
  "trailer": {
    "url": "https://res.cloudinary.com/dxwhngg8g/video/upload/v1234567890/event_trailers/abc123/1234567890.mp4",
    "public_id": "event_trailers/abc123/1234567890",
    "duration": 45.5,
    "format": "mp4",
    "size": 15728640,
    "uploaded_at": ISODate("2025-10-27T10:30:00Z"),
    "thumbnail_url": "https://res.cloudinary.com/dxwhngg8g/video/upload/so_0/v1234567890/event_trailers/abc123/1234567890.jpg"
  }
  // ou trailer: null si pas de trailer
}
```

---

## 🎥 Fonctionnalités Cloudinary

### Génération automatique de miniature

Cloudinary génère automatiquement une image miniature (frame à 0 seconde) :

```
URL vidéo :
https://res.cloudinary.com/dxwhngg8g/video/upload/v1234567890/event_trailers/abc123/1234567890.mp4

URL miniature :
https://res.cloudinary.com/dxwhngg8g/video/upload/so_0/v1234567890/event_trailers/abc123/1234567890.jpg
```

### Streaming adaptatif

Les vidéos sont automatiquement optimisées pour le streaming avec qualité adaptative.

### Organisation des fichiers

```
event_trailers/
  ├── {event_id_1}/
  │   └── {timestamp}.mp4
  ├── {event_id_2}/
  │   └── {timestamp}.mp4
  └── ...
```

---

## 🔒 Sécurité

- ✅ Authentification JWT requise
- ✅ Autorisation admin via middleware
- ✅ Validation du type MIME
- ✅ Validation de la taille (100 MB max)
- ✅ Formats autorisés : MP4, MOV, AVI, WebM uniquement
- ✅ Suppression automatique des anciennes vidéos lors du remplacement

---

## ✅ Checklist d'Implémentation

- [x] Modèle EventTrailer créé
- [x] Champ trailer ajouté au modèle Event
- [x] Handler EventTrailerHandler créé
- [x] Endpoint POST /trailer implémenté
- [x] Endpoint PUT /trailer implémenté
- [x] Endpoint DELETE /trailer implémenté
- [x] Routes admin ajoutées dans main.go
- [x] Validation des fichiers (type, taille)
- [x] Upload vers Cloudinary implémenté
- [x] Suppression Cloudinary implémentée
- [x] Génération de miniatures
- [x] Documentation API complète
- [ ] **Créer preset Cloudinary pour vidéos** ⚠️ À faire
- [ ] Tester les 3 endpoints
- [ ] Ajouter variables Cloudinary sur Render

---

## 🚀 Prochaines Étapes

1. **Créer le preset Cloudinary** (voir section Configuration)
2. **Tester localement** avec les 3 endpoints
3. **Ajouter les variables Cloudinary sur Render** (si pas déjà fait)
4. **Implémenter l'interface admin frontend**
5. **Afficher les trailers dans l'interface publique**

---

**Version** : 1.0  
**Date** : 27 octobre 2025  
**Commit** : a11bcee

