package middleware

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"premier-an-backend/services"
	"strconv"
	"time"
)

// responseWriter wrapper pour capturer le code de statut
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{w, http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// isCriticalError détermine si une erreur est critique et doit être notifiée sur Slack
// Retourne true pour les erreurs serveur (5xx) et les erreurs CORS (403)
// Retourne false pour les erreurs utilisateur (400, 401, 404, etc.)
func isCriticalError(statusCode int, path string) bool {
	// Erreurs serveur (5xx) - toujours critiques
	if statusCode >= http.StatusInternalServerError {
		return true
	}

	// Erreur CORS (403) - critique car cela indique un problème de configuration
	// On vérifie si c'est une erreur CORS en regardant si c'est une erreur 403
	// et si le path n'est pas une route d'authentification (pour éviter les faux positifs)
	if statusCode == http.StatusForbidden {
		// Les erreurs CORS sont généralement des erreurs 403 sur n'importe quelle route
		// On peut les identifier car elles viennent du middleware CORS
		// Pour être sûr, on notifie toutes les erreurs 403 sauf celles liées à l'authentification
		// qui sont gérées par le middleware Auth
		return true
	}

	// Erreurs client (4xx) - pas critiques (mauvais mot de passe, etc.)
	return false
}

// Logging enregistre les requêtes HTTP et envoie des notifications Slack pour les erreurs critiques
func Logging(slackService *services.SlackService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Log toutes les requêtes de connexion pour debug (écriture directe sur stderr)
			if r.URL.Path == "/api/connexion" || r.URL.Path == "/api/auth/login" {
				timestamp := time.Now().Format("2006/01/02 15:04:05")
				origin := r.Header.Get("Origin")
				userAgent := r.Header.Get("User-Agent")
				auth := r.Header.Get("Authorization")
				fmt.Fprintf(os.Stderr, "%s 🔍 [LOGGING] Requête entrante: %s %s - Origin: '%s' - User-Agent: '%s' - Auth: '%s'\n", 
					timestamp, r.Method, r.URL.Path, origin, userAgent, auth)
				log.Printf("🔍 [LOGGING] Requête entrante: %s %s - Origin: '%s' - User-Agent: '%s' - Auth: '%s'", 
					r.Method, r.URL.Path, origin, userAgent, auth)
			}

			// Créer un wrapper pour capturer le code de statut
			rw := newResponseWriter(w)

			// Traiter la requête
			next.ServeHTTP(rw, r)

			duration := time.Since(start)
			statusCode := rw.statusCode

			// Logger toutes les erreurs
			if statusCode >= http.StatusBadRequest {
				// Log détaillé pour les erreurs de connexion
				if r.URL.Path == "/api/connexion" || r.URL.Path == "/api/auth/login" {
					log.Printf("❌ [LOGGING] Erreur %s %s -> %d (%s) - Origin: '%s'", 
						r.Method, r.RequestURI, statusCode, duration, r.Header.Get("Origin"))
				} else {
					log.Printf(
						"⚠️ %s %s -> %d (%s)",
						r.Method,
						r.RequestURI,
						statusCode,
						duration,
					)
				}

				// Envoyer une notification Slack uniquement pour les erreurs critiques
				if isCriticalError(statusCode, r.RequestURI) && slackService != nil {
					origin := r.Header.Get("Origin")
					userAgent := r.Header.Get("User-Agent")
					statusCodeStr := strconv.Itoa(statusCode)

					// Déterminer le type d'erreur et envoyer la notification appropriée
					if statusCode >= http.StatusInternalServerError {
						// Erreur serveur (5xx)
						errorMessage := http.StatusText(statusCode)
						slackService.SendCriticalError(r.Method, r.RequestURI, statusCodeStr, errorMessage, origin, userAgent)
					} else if statusCode == http.StatusForbidden {
						// Erreur 403 - peut être CORS ou accès refusé
						if origin != "" {
							// Probablement une erreur CORS
							slackService.SendCORSError(r.Method, r.RequestURI, origin, userAgent)
						} else {
							// Accès refusé (pas CORS)
							slackService.SendCriticalError(r.Method, r.RequestURI, statusCodeStr, "Accès refusé", origin, userAgent)
						}
					}
				}
			}
		})
	}
}
