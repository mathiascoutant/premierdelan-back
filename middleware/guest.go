package middleware

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"premier-an-backend/utils"
	"strings"
	"time"
)

// Guest vérifie que l'utilisateur n'est PAS connecté
// Si un token valide est présent, refuse l'accès
func Guest(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Log pour debug (seulement pour les routes de connexion)
			if r.URL.Path == "/api/connexion" || r.URL.Path == "/api/auth/login" {
				timestamp := time.Now().Format("2006/01/02 15:04:05")
				authHeader := r.Header.Get("Authorization")
				fmt.Fprintf(os.Stderr, "%s 🔍 [GUEST] Vérification pour %s - Auth: '%s'\n", timestamp, r.URL.Path, authHeader)
			}
			
			// Récupérer le token depuis l'en-tête Authorization
			authHeader := r.Header.Get("Authorization")
			
			// Si pas de header Authorization, continuer (utilisateur non connecté)
			if authHeader == "" {
				if r.URL.Path == "/api/connexion" || r.URL.Path == "/api/auth/login" {
					timestamp := time.Now().Format("2006/01/02 15:04:05")
					fmt.Fprintf(os.Stderr, "%s 🔓 [GUEST] Pas de token - autorisation de continuer vers %s\n", timestamp, r.URL.Path)
				}
				log.Printf("🔓 [GUEST] Pas de token - autorisation de continuer vers %s", r.URL.Path)
				next.ServeHTTP(w, r)
				return
			}

			// Vérifier le format "Bearer <token>"
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				// Format invalide, continuer (pas de token valide)
				if r.URL.Path == "/api/connexion" || r.URL.Path == "/api/auth/login" {
					timestamp := time.Now().Format("2006/01/02 15:04:05")
					fmt.Fprintf(os.Stderr, "%s ⚠️  [GUEST] Format de token invalide - autorisation de continuer vers %s\n", timestamp, r.URL.Path)
				}
				log.Printf("⚠️  [GUEST] Format de token invalide - autorisation de continuer vers %s", r.URL.Path)
				next.ServeHTTP(w, r)
				return
			}

			tokenString := parts[1]

			// Valider le token (rapide, ne bloque pas)
			_, err := utils.ValidateToken(tokenString, jwtSecret)
			if err == nil {
				// Token valide = utilisateur déjà connecté
				if r.URL.Path == "/api/connexion" || r.URL.Path == "/api/auth/login" {
					timestamp := time.Now().Format("2006/01/02 15:04:05")
					fmt.Fprintf(os.Stderr, "%s 🚫 [GUEST] Token valide détecté - refus d'accès à %s (utilisateur déjà connecté)\n", timestamp, r.URL.Path)
				}
				log.Printf("🚫 [GUEST] Token valide détecté - refus d'accès à %s (utilisateur déjà connecté)", r.URL.Path)
				utils.RespondError(w, http.StatusForbidden, "Vous êtes déjà connecté")
				return
			}

			// Token invalide ou expiré, continuer (c'est normal pour une nouvelle connexion)
			if r.URL.Path == "/api/connexion" || r.URL.Path == "/api/auth/login" {
				timestamp := time.Now().Format("2006/01/02 15:04:05")
				fmt.Fprintf(os.Stderr, "%s 🔓 [GUEST] Token invalide/expiré - autorisation de continuer vers %s (erreur: %v)\n", timestamp, r.URL.Path, err)
			}
			log.Printf("🔓 [GUEST] Token invalide/expiré - autorisation de continuer vers %s (erreur: %v)", r.URL.Path, err)
			next.ServeHTTP(w, r)
		})
	}
}

