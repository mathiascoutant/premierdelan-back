# Déploiement automatique (CI/CD)

Le backend se déploie automatiquement sur le VPS OVH à chaque **push sur la branche `dev`**.

## Configuration requise

### 1. Secrets GitHub

Dans le dépôt GitHub : **Settings** → **Secrets and variables** → **Actions** → **New repository secret**

| Secret | Description | Exemple |
|--------|-------------|---------|
| `VPS_HOST` | IP ou hostname du VPS OVH | `123.45.67.89` ou `vps.mondomaine.com` |
| `VPS_USER` | Utilisateur SSH | `ubuntu` ou `root` |
| `VPS_SSH_KEY` | Clé privée SSH (contenu complet) | Contenu de `~/.ssh/id_rsa` |
| `VPS_DEPLOY_PATH` | Chemin du projet sur le VPS | `/home/ubuntu/premier-an/back` |

### 2. Préparer le VPS

Sur le serveur OVH :

```bash
# Cloner le repo (si pas déjà fait)
git clone https://github.com/VOTRE_ORG/premier-an.git
cd premier-an/back

# Ou si le back est dans un repo séparé
git clone https://github.com/VOTRE_ORG/back.git
cd back
```

### 3. Clé SSH pour GitHub Actions

La clé utilisée par GitHub Actions doit être autorisée sur le VPS :

```bash
# Sur votre machine locale : générer une clé dédiée (optionnel)
ssh-keygen -t ed25519 -C "github-actions-deploy" -f deploy_key -N ""

# Copier la clé publique sur le VPS
ssh-copy-id -i deploy_key.pub ubuntu@VOTRE_VPS_IP

# Le contenu de deploy_key (sans .pub) va dans VPS_SSH_KEY
cat deploy_key
```

### 4. Service systemd

Le script `deploy.sh` utilise `systemctl` pour gérer le service. L'utilisateur SSH doit avoir les droits sudo pour :

```bash
sudo systemctl start backend
sudo systemctl stop backend
```

Configurer le sudoers si nécessaire :

```bash
# Éditer /etc/sudoers (avec visudo)
ubuntu ALL=(ALL) NOPASSWD: /bin/systemctl start backend
ubuntu ALL=(ALL) NOPASSWD: /bin/systemctl stop backend
ubuntu ALL=(ALL) NOPASSWD: /bin/systemctl status backend
```

### 5. Fichier unit systemd

Exemple `/etc/systemd/system/backend.service` :

```ini
[Unit]
Description=Premier de l'An - Backend API
After=network.target

[Service]
Type=simple
User=ubuntu
WorkingDirectory=/home/ubuntu/premier-an/back
ExecStart=/home/ubuntu/premier-an/back/backend
Restart=on-failure
RestartSec=5
EnvironmentFile=/home/ubuntu/premier-an/back/.env

[Install]
WantedBy=multi-user.target
```

## Flux de déploiement

1. **Push sur `dev`** → GitHub Actions se déclenche
2. Connexion SSH au VPS
3. `cd` vers le répertoire du projet
4. Exécution de `./deploy.sh dev`
5. Le script : arrête le service → `git pull origin dev` → compile → redémarre

## Déploiement manuel

Sur le VPS :

```bash
cd /chemin/vers/back
./deploy.sh dev   # pour la branche dev
./deploy.sh main  # pour la branche main (production)
```

## SonarQube (analyse qualité + couverture)

Un workflow dédié (`sonarqube.yml`) s'exécute à chaque push sur dev.

**Secrets à ajouter :**
- `SONAR_TOKEN` : Token depuis SonarQube > Mon compte > Sécurité
- `SONAR_HOST_URL` : `https://sonarcloud.io` (ou l'URL de ton instance)

La couverture de tests est générée via `go test -coverprofile=coverage.out`.

---

## Vérifier le statut

- **GitHub** : onglet **Actions** du dépôt
- **VPS** : `journalctl -u backend -f` pour les logs
