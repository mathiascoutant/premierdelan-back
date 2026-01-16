package middleware

import (
	"net/http"
	"premier-an-backend/utils"
	"strings"
)

// Guest vérifie que l'utilisateur n'est PAS connecté
// Si un token valide est présent, refuse l'accès
func Guest(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Récupérer le token depuis l'en-tête Authorization
			authHeader := r.Header.Get("Authorization")
			
			// Si pas de header Authorization, continuer (utilisateur non connecté)
			if authHeader == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Vérifier le format "Bearer <token>"
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				// Format invalide, continuer (pas de token valide)
				next.ServeHTTP(w, r)
				return
			}

			tokenString := parts[1]

			// Valider le token (rapide, ne bloque pas)
			_, err := utils.ValidateToken(tokenString, jwtSecret)
			if err == nil {
				// Token valide = utilisateur déjà connecté
				utils.RespondError(w, http.StatusForbidden, "Vous êtes déjà connecté")
				return
			}

			// Token invalide ou expiré, continuer (c'est normal pour une nouvelle connexion)
			next.ServeHTTP(w, r)
		})
	}
}

