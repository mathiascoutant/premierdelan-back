# 🔐 Gestion des Administrateurs

## Champ Admin

Tous les utilisateurs ont maintenant un champ `admin` :
- `admin: 0` → Utilisateur normal
- `admin: 1` → Administrateur

## 📝 Commandes MongoDB

### Passer un utilisateur en admin
```bash
mongosh premierdelan --eval 'db.users.updateOne({email: "user@email.com"}, {$set: {admin: 1}})'
```

### Retirer les droits admin
```bash
mongosh premierdelan --eval 'db.users.updateOne({email: "user@email.com"}, {$set: {admin: 0}})'
```

### Voir tous les admins
```bash
mongosh premierdelan --eval 'db.users.find({admin: 1}, {firstname: 1, lastname: 1, email: 1, admin: 1}).pretty()'
```

### Voir tous les utilisateurs avec leur statut
```bash
mongosh premierdelan --eval 'db.users.find({}, {firstname: 1, lastname: 1, email: 1, admin: 1}).pretty()'
```

## 🔧 Utilisation dans le code

### Vérifier si un utilisateur est admin

Après la connexion, le champ `admin` est inclus dans la réponse :

```json
{
  "token": "...",
  "user": {
    "id": "...",
    "firstname": "Mathias",
    "lastname": "COUTANT",
    "email": "mathias@gmail.com",
    "admin": 1,  // ⭐ 1 = admin, 0 = utilisateur normal
    "created_at": "..."
  }
}
```

### Côté Frontend

```javascript
// Stocker les infos utilisateur après connexion
const userData = response.user;

if (userData.admin === 1) {
  console.log('✅ Utilisateur est ADMIN');
  // Afficher les fonctionnalités admin
  document.getElementById('admin-panel').style.display = 'block';
} else {
  console.log('👤 Utilisateur normal');
  // Masquer les fonctionnalités admin
  document.getElementById('admin-panel').style.display = 'none';
}
```

## 🎯 Exemples d'utilisation

### Protéger les routes admin côté backend

Vous pouvez créer un middleware pour vérifier si l'utilisateur est admin :

```go
// middleware/admin.go
func RequireAdmin(userRepo *database.UserRepository) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Récupérer l'utilisateur depuis le contexte (déjà authentifié)
            claims := GetUserFromContext(r.Context())
            if claims == nil {
                utils.RespondError(w, http.StatusUnauthorized, "Non authentifié")
                return
            }
            
            // Récupérer l'utilisateur complet depuis la DB
            userID, _ := primitive.ObjectIDFromHex(claims.UserID)
            user, err := userRepo.FindByID(userID)
            if err != nil || user == nil {
                utils.RespondError(w, http.StatusUnauthorized, "Utilisateur non trouvé")
                return
            }
            
            // Vérifier si admin
            if user.Admin != 1 {
                utils.RespondError(w, http.StatusForbidden, "Accès réservé aux administrateurs")
                return
            }
            
            next.ServeHTTP(w, r)
        })
    }
}
```

### Route admin protégée

```go
// Dans main.go
adminRouter := protected.PathPrefix("/admin").Subrouter()
adminRouter.Use(middleware.RequireAdmin(userRepo))

adminRouter.HandleFunc("/users", adminHandler.ListUsers).Methods("GET")
adminRouter.HandleFunc("/stats", adminHandler.GetStats).Methods("GET")
```

## 📊 Statistiques

```bash
# Compter les admins
mongosh premierdelan --eval 'db.users.countDocuments({admin: 1})'

# Compter les utilisateurs normaux
mongosh premierdelan --eval 'db.users.countDocuments({admin: 0})'
```

## ⚠️ Sécurité

- ✅ Le champ `admin` est stocké dans MongoDB
- ✅ Il est retourné lors de la connexion
- ✅ Le frontend peut l'utiliser pour afficher/masquer des fonctionnalités
- ⚠️ Mais TOUJOURS vérifier côté backend avant d'autoriser des actions sensibles
- 🔐 Ne jamais faire confiance uniquement au frontend pour la sécurité

## 🚀 Commandes rapides

```bash
# Promouvoir mathiascoutant@icloud.com en admin
mongosh premierdelan --eval 'db.users.updateOne({email: "mathiascoutant@icloud.com"}, {$set: {admin: 1}})'

# Rétrograder en utilisateur normal
mongosh premierdelan --eval 'db.users.updateOne({email: "mathiascoutant@icloud.com"}, {$set: {admin: 0}})'
```

