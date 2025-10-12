# 🚀 Déploiement sur Railway.app

## ✅ Étape 1 : Claim ton Projet

Dans Railway, clique sur **"Claim Project"** pour rendre le projet permanent.

---

## 📦 Étape 2 : Ajouter MongoDB

1. Clique **"+ New"** dans ton projet
2. Sélectionne **"Database"**
3. Choisis **"Add MongoDB"**
4. Attends 2-3 minutes que Railway crée la base

✅ Railway génère automatiquement `MONGO_URL`

---

## 🔧 Étape 3 : Ajouter ton Backend

### Option A : Depuis GitHub (RECOMMANDÉ)

1. Push ton code sur GitHub :

```bash
cd "/Users/mathias/Desktop/    /premier de l'an/site/back"
git init
git add .
git commit -m "Initial commit"
git branch -M main
git remote add origin https://github.com/TON_USERNAME/premier-de-lan-backend.git
git push -u origin main
```

2. Dans Railway :
   - Clique **"+ New"**
   - Choisis **"GitHub Repo"**
   - Sélectionne ton repo

### Option B : Railway CLI

```bash
# 1. Installer Railway CLI
npm i -g @railway/cli

# 2. Login
railway login

# 3. Link au projet
railway link

# 4. Déployer
railway up
```

---

## ⚙️ Étape 4 : Variables d'Environnement

Dans ton service Backend sur Railway, va dans **"Variables"** et ajoute :

### Variables Obligatoires

```bash
# Port (Railway utilise PORT automatiquement)
PORT=8090

# MongoDB (connecté automatiquement)
MONGO_URI=${{MongoDB.MONGO_URL}}
MONGO_DB=premierdelan

# JWT
JWT_SECRET=votre_secret_jwt_super_securise_a_changer_en_production

# Environment
ENVIRONMENT=production

# CORS (ajoute ton URL Railway une fois générée)
CORS_ALLOWED_ORIGINS=https://mathiascoutant.github.io,https://TON-APP.railway.app

# Firebase Cloud Messaging - VAPID Key
FCM_VAPID_KEY=BKtsyuWpu2lZY64MGiqwnBglbWFUBd9oMQWnmH9F3Y6DJ8gBSmXo0ASIwCZXxyK1XvXu_CxKwAd3cVSw-sNQ70o

# Firebase Service Account (copie le contenu de firebase-service-account.json)
FIREBASE_CREDENTIALS_JSON={"type":"service_account","project_id":"premier-de-lan","private_key_id":"...","private_key":"...","client_email":"...","client_id":"...","auth_uri":"...","token_uri":"...","auth_provider_x509_cert_url":"...","client_x509_cert_url":"...","universe_domain":"googleapis.com"}
```

### Comment obtenir FIREBASE_CREDENTIALS_JSON

```bash
# Sur ton Mac, copie le contenu du fichier Firebase
cat firebase-service-account.json | tr -d '\n'
```

Puis colle le résultat dans Railway (tout sur une ligne).

---

## 🔄 Étape 5 : Modifier le Code pour Railway

Le code doit lire Firebase depuis la variable d'environnement au lieu du fichier.

Je prépare les modifications...

---

## 📊 Étape 6 : Obtenir ton URL

Une fois déployé :

1. Va dans **"Settings"** de ton service Backend
2. Copie l'URL publique : `https://loving-playfulness-production.up.railway.app`
3. **Remplace ngrok** par cette URL dans ton frontend

---

## ✅ Avantages

- ✅ URL permanente (ne change jamais)
- ✅ Toujours en ligne (pas de "sleep")
- ✅ Gratuit sans carte bancaire
- ✅ SSL automatique (HTTPS)
- ✅ Logs en temps réel
- ✅ Redémarrage automatique en cas d'erreur

---

**Tu veux que je modifie le code pour qu'il lise Firebase depuis la variable d'environnement ?** 🚀
