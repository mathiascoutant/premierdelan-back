package middleware

import (
	"context"
	"net/http"
	"premier-an-backend/utils"
	"strings"
)

type contextKey string

const UserContextKey contextKey = "user"

// Auth vérifie le token JWT
func Auth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Récupérer le token depuis l'en-tête Authorization
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				utils.RespondError(w, http.StatusUnauthorized, "Token d'authentification manquant")
				return
			}

			// Vérifier le format "Bearer <token>"
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				utils.RespondError(w, http.StatusUnauthorized, "Format du token invalide")
				return
			}

			tokenString := parts[1]

			// Valider le token
			claims, err := utils.ValidateToken(tokenString, jwtSecret)
			if err != nil {
				utils.RespondError(w, http.StatusUnauthorized, "Token invalide ou expiré")
				return
			}

			// Ajouter les informations de l'utilisateur au contexte
			ctx := context.WithValue(r.Context(), UserContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserFromContext récupère les informations de l'utilisateur depuis le contexte
func GetUserFromContext(ctx context.Context) *utils.Claims {
	claims, ok := ctx.Value(UserContextKey).(*utils.Claims)
	if !ok {
		return nil
	}
	return claims
}
