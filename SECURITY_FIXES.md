# 🔒 Corrections de Sécurité - 16/01/2026

## 📋 Résumé

Mise à jour des dépendances pour corriger les vulnérabilités détectées par `govulncheck`.

---

## ✅ Corrections Appliquées

### 1. github.com/golang-jwt/jwt/v5
- **Avant** : v5.2.0
- **Après** : v5.2.2
- **Vulnérabilité corrigée** : GO-2025-3553 (Excessive memory allocation during header parsing)

### 2. github.com/golang-jwt/jwt/v4
- **Avant** : v4.5.0
- **Après** : v4.5.2
- **Vulnérabilité corrigée** : GO-2025-3553 (Excessive memory allocation during header parsing)

### 3. golang.org/x/net
- **Avant** : v0.18.0
- **Après** : v0.23.0
- **Vulnérabilité corrigée** : GO-2024-2687 (HTTP/2 CONTINUATION flood)

### 4. golang.org/x/crypto
- **Mise à jour automatique** : v0.18.0 → v0.21.0 (dépendance transitive)

### 5. golang.org/x/sys
- **Mise à jour automatique** : v0.16.0 → v0.18.0 (dépendance transitive)

---

## ⚠️ Vulnérabilités Restantes (13)

**13 vulnérabilités** dans la bibliothèque standard Go nécessitent une mise à jour de Go lui-même :

- **Version Go actuelle** : 1.24.2
- **Version Go requise** : 1.24.11 minimum (pour corriger toutes les vulnérabilités)

**Vulnérabilités concernées** :
- GO-2025-4175, GO-2025-4155 (crypto/x509) - Fixed in 1.24.11
- GO-2025-4013, GO-2025-4012, GO-2025-4011, GO-2025-4010, GO-2025-4009, GO-2025-4008 (divers) - Fixed in 1.24.8
- GO-2025-4007 (crypto/x509) - Fixed in 1.24.9
- GO-2025-3956 (os/exec) - Fixed in 1.24.6
- GO-2025-3751, GO-2025-3750, GO-2025-3749 (divers) - Fixed in 1.24.4

**Action requise** : Mettre à jour Go sur le serveur vers 1.24.11 ou supérieur.

```bash
# Sur le serveur
# 1. Télécharger la dernière version de Go 1.24.x
# 2. Installer
# 3. Vérifier
go version  # Doit afficher go1.24.11 ou supérieur
# 4. Recompiler et redéployer
cd ~/projects/premierdelan-back
make build
sudo systemctl restart backend
```

---

## 📝 Changements dans le Code

- ✅ `go.mod` : Mises à jour des versions des dépendances
- ✅ `go.sum` : Nouveaux checksums de sécurité
- ✅ Aucun changement de code source nécessaire (corrections dans les dépendances uniquement)

---

## ✅ Validation

Après déploiement, vérifier avec :
```bash
make deps-vuln
```

Les 2 vulnérabilités dans les dépendances JWT et net devraient être corrigées.
Les 13 vulnérabilités dans la bibliothèque standard Go seront corrigées après mise à jour de Go.

---

## 📅 Historique

| Date | Action | Packages | Statut |
|------|--------|----------|--------|
| 2026-01-16 | Mise à jour dépendances | jwt/v5, jwt/v4, golang.org/x/net | ✅ Appliqué |
| 2026-01-16 | Mise à jour Go requise | Go 1.24.2 → 1.24.11+ | ⏳ À faire |

---

**Dernière mise à jour** : 2026-01-16
