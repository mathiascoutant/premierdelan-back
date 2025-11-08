package middleware

import (
	"log"
	"net/http"
	"premier-an-backend/database"
	"premier-an-backend/utils"

	"go.mongodb.org/mongo-driver/mongo"
)

// RequireAdmin vérifie que l'utilisateur est un administrateur
func RequireAdmin(db *mongo.Database) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Récupérer les claims depuis le contexte (mis par le middleware Auth)
			claims := GetUserFromContext(r.Context())
			if claims == nil {
				utils.RespondError(w, http.StatusUnauthorized, "Non authentifié")
				return
			}

			// Récupérer l'utilisateur complet depuis la DB par email
			userRepo := database.NewUserRepository(db)
			user, err := userRepo.FindByEmail(claims.UserID)
			if err != nil || user == nil {
				log.Printf("Utilisateur non trouvé: %v", err)
				utils.RespondError(w, http.StatusUnauthorized, "Utilisateur non trouvé")
				return
			}

			// Vérifier si l'utilisateur est admin
			if user.Admin != 1 {
				log.Printf("⚠️  Accès admin refusé pour: %s (admin=%d)", user.Email, user.Admin)
				utils.RespondError(w, http.StatusForbidden, "Accès refusé - Admin uniquement")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
