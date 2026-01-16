# 🖥️ Configuration Serveur IP - Backend Premier de l'An

## 📍 Informations Serveur

**URL Backend** : `https://57.131.29.179/api/`  
**WebSocket** : `wss://57.131.29.179/ws/chat`  
**Type** : Serveur dédié avec IP fixe

## 🔧 Variables d'Environnement Requises

Assurez-vous que ces variables sont configurées sur le serveur :

```bash
# Configuration serveur
PORT=8090
HOST=0.0.0.0

# Base de données MongoDB
MONGO_URI=mongodb://localhost:27017
MONGO_DB=premier_an_db

# JWT Secret (OBLIGATOIRE - doit être sécurisé)
JWT_SECRET=votre_secret_jwt_super_securise

# Environnement
ENVIRONMENT=production

# CORS - Autoriser votre frontend
CORS_ALLOWED_ORIGINS=https://votre-frontend.com,https://57.131.29.179

# Cloudinary (pour upload d'images et vidéos)
CLOUDINARY_CLOUD_NAME=votre_cloud_name
CLOUDINARY_UPLOAD_PRESET=premierdelan_profiles
CLOUDINARY_VIDEO_PRESET=premierdelan_trailers
CLOUDINARY_PREVIEW_PRESET=premierdelan_gallery_preview
CLOUDINARY_API_KEY=votre_api_key
CLOUDINARY_API_SECRET=votre_api_secret

# Firebase (pour notifications push - OPTIONNEL)
FIREBASE_CREDENTIALS_BASE64=votre_credentials_base64_encoded
# OU
FIREBASE_CREDENTIALS_FILE=/path/to/firebase-service-account.json
FCM_VAPID_KEY=votre_fcm_vapid_key
```

## 🚀 Déploiement sur le Serveur

### Option 1 : Build et Exécution Directe

```bash
# 1. Se connecter au serveur
ssh user@57.131.29.179

# 2. Aller dans le dossier du projet
cd /path/to/premier-an-backend

# 3. Pull les dernières modifications
git pull origin main

# 4. Compiler le projet
go build -o premier-an-backend

# 5. Redémarrer le service
sudo systemctl restart premier-an-backend
# OU
./premier-an-backend
```

### Option 2 : Avec systemd Service

Créer un fichier `/etc/systemd/system/premier-an-backend.service` :

```ini
[Unit]
Description=Premier de l'An Backend API
After=network.target mongodb.service

[Service]
Type=simple
User=www-data
WorkingDirectory=/path/to/premier-an-backend
ExecStart=/path/to/premier-an-backend/premier-an-backend
Restart=always
RestartSec=10
StandardOutput=append:/var/log/premier-an-backend/output.log
StandardError=append:/var/log/premier-an-backend/error.log

# Variables d'environnement
Environment="PORT=8090"
Environment="HOST=0.0.0.0"
Environment="MONGO_URI=mongodb://localhost:27017"
Environment="MONGO_DB=premier_an_db"
Environment="JWT_SECRET=votre_secret"
Environment="ENVIRONMENT=production"
Environment="CORS_ALLOWED_ORIGINS=https://votre-frontend.com"

[Install]
WantedBy=multi-user.target
```

Activer et démarrer :

```bash
sudo systemctl daemon-reload
sudo systemctl enable premier-an-backend
sudo systemctl start premier-an-backend
sudo systemctl status premier-an-backend
```

### Option 3 : Avec Docker

```bash
# Build l'image
docker build -t premier-an-backend .

# Lancer le conteneur
docker run -d \
  --name premier-an-backend \
  -p 8090:8090 \
  -e PORT=8090 \
  -e HOST=0.0.0.0 \
  -e MONGO_URI=mongodb://host.docker.internal:27017 \
  -e MONGO_DB=premier_an_db \
  -e JWT_SECRET=votre_secret \
  -e ENVIRONMENT=production \
  -e CORS_ALLOWED_ORIGINS=https://votre-frontend.com \
  --restart unless-stopped \
  premier-an-backend
```

## 🔐 Configuration Nginx (Reverse Proxy)

Si vous utilisez Nginx comme reverse proxy :

```nginx
server {
    listen 443 ssl http2;
    server_name 57.131.29.179;

    ssl_certificate /etc/ssl/certs/server.crt;
    ssl_certificate_key /etc/ssl/private/server.key;

    # Backend API
    location /api/ {
        proxy_pass http://localhost:8090/api/;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # WebSocket
    location /ws/ {
        proxy_pass http://localhost:8090/ws/;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 86400;
    }
}
```

## 📝 Configuration Frontend

Mettre à jour l'URL de l'API dans votre frontend :

### Next.js / React

```javascript
// .env.production
NEXT_PUBLIC_API_URL=https://57.131.29.179/api
NEXT_PUBLIC_WS_URL=wss://57.131.29.179/ws/chat

// OU dans votre config
const API_BASE_URL = "https://57.131.29.179/api";
const WS_URL = "wss://57.131.29.179/ws/chat";
```

### Fichier de config

```javascript
// config/api.js
export const API_CONFIG = {
  baseURL: process.env.NEXT_PUBLIC_API_URL || "https://57.131.29.179/api",
  wsURL: process.env.NEXT_PUBLIC_WS_URL || "wss://57.131.29.179/ws/chat",
  timeout: 30000,
};
```

## 🔍 Vérification et Tests

### Test de Santé

```bash
curl https://57.131.29.179/api/health
```

Réponse attendue :

```json
{
  "status": "healthy",
  "timestamp": "2025-01-01T12:00:00Z",
  "version": "1.0.0"
}
```

### Test des Événements Publics

```bash
curl https://57.131.29.179/api/evenements/public
```

### Test WebSocket

```bash
# Installer wscat si nécessaire
npm install -g wscat

# Test connexion WebSocket
wscat -c wss://57.131.29.179/ws/chat
```

## 📊 Monitoring et Logs

### Voir les logs en temps réel

```bash
# Si systemd
sudo journalctl -u premier-an-backend -f

# Si logs fichier
tail -f /var/log/premier-an-backend/output.log

# Si Docker
docker logs -f premier-an-backend
```

### Vérifier l'état du service

```bash
# Si systemd
sudo systemctl status premier-an-backend

# Si Docker
docker ps | grep premier-an-backend
```

## 🔄 Mise à Jour du Backend

```bash
# 1. Se connecter au serveur
ssh user@57.131.29.179

# 2. Aller dans le dossier
cd /path/to/premier-an-backend

# 3. Pull les changements
git pull origin main

# 4. Recompiler
go build -o premier-an-backend

# 5. Redémarrer
sudo systemctl restart premier-an-backend
# OU
docker restart premier-an-backend
```

## 🛡️ Sécurité

### Points Importants

1. **JWT_SECRET** : Doit être une chaîne aléatoire longue et sécurisée
2. **HTTPS** : Toujours utiliser HTTPS en production
3. **Firewall** : Ouvrir uniquement les ports nécessaires (443, 8090)
4. **MongoDB** : Ne pas exposer directement, utiliser localhost
5. **CORS** : Limiter aux origines autorisées (pas de wildcard `*` en production)

### Générer un JWT Secret sécurisé

```bash
openssl rand -base64 64
```

## 📞 Support

En cas de problème :

1. Vérifier les logs : `sudo journalctl -u premier-an-backend -n 100`
2. Vérifier que MongoDB est actif : `sudo systemctl status mongodb`
3. Vérifier les variables d'environnement
4. Tester la connexion à l'API : `curl https://57.131.29.179/api/health`

## 🎯 Checklist de Déploiement

- [ ] MongoDB installé et accessible
- [ ] Variables d'environnement configurées
- [ ] JWT_SECRET généré et sécurisé
- [ ] CORS configuré avec les bonnes origines
- [ ] Certificat SSL configuré (pour HTTPS)
- [ ] Nginx/reverse proxy configuré
- [ ] Service systemd créé et activé
- [ ] Logs configurés et accessibles
- [ ] Firewall configuré (ports 443, 8090)
- [ ] Tests API réussis
- [ ] Tests WebSocket réussis
- [ ] Frontend mis à jour avec la nouvelle URL
