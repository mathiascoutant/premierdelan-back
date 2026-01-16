package middleware

import (
	"log"
	"net/http"
	"strings"
)

// CORS gère les en-têtes CORS
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			
			// Log pour debug (seulement pour les routes de connexion)
			if r.URL.Path == "/api/connexion" || r.URL.Path == "/api/auth/login" {
				log.Printf("🌐 [CORS] Requête %s %s - Origin: '%s'", r.Method, r.URL.Path, origin)
			}
			
			// Vérifier si l'origine est autorisée
			allowed := isOriginAllowed(origin, allowedOrigins)

			// Si pas d'origine (requête depuis une app native ou file://), autoriser par défaut
			// Cela permet aux apps mobiles de fonctionner même sans origine
			if origin == "" {
				allowed = true
				// Pour les requêtes sans origine, on peut définir * ou ne rien définir
				// Mais on définit quand même les en-têtes pour permettre les requêtes
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else if allowed {
				// Si l'origine est autorisée, l'utiliser explicitement
				w.Header().Set("Access-Control-Allow-Origin", origin)
			} else {
				// Si l'origine n'est pas autorisée, on la refuse mais on log pour le débogage
				log.Printf("⚠️  CORS: Origine non autorisée: %s (URI: %s %s)", origin, r.Method, r.URL.Path)
				// On définit quand même les en-têtes pour permettre au client de voir l'erreur
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}

			// Toujours définir les en-têtes CORS nécessaires
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, ngrok-skip-browser-warning")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Max-Age", "3600")
			w.Header().Set("Access-Control-Expose-Headers", "Content-Type, Authorization")

			// Gérer les requêtes OPTIONS (preflight)
			if r.Method == http.MethodOptions {
				if allowed || origin == "" {
					w.WriteHeader(http.StatusNoContent)
				} else {
					// Même si l'origine n'est pas autorisée, on répond avec 200 pour éviter les erreurs CORS
					// Le vrai problème sera géré par le frontend
					w.WriteHeader(http.StatusNoContent)
				}
				return
			}

			// Pour les requêtes non-OPTIONS, continuer seulement si l'origine est autorisée ou si pas d'origine
			if !allowed && origin != "" {
				log.Printf("❌ [CORS] Blocage requête - Origine: %s, URI: %s %s", origin, r.Method, r.URL.Path)
				w.WriteHeader(http.StatusForbidden)
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"error":"Forbidden","message":"Origine non autorisée"}`))
				return
			}

			// Log pour debug (seulement pour les routes de connexion)
			if (r.URL.Path == "/api/connexion" || r.URL.Path == "/api/auth/login") && r.Method != http.MethodOptions {
				log.Printf("✅ [CORS] Autorisation accordée - Origin: '%s', Allowed: %v", origin, allowed)
			}

			next.ServeHTTP(w, r)
		})
	}
}

// isOriginAllowed vérifie si une origine est dans la liste des origines autorisées
func isOriginAllowed(origin string, allowedOrigins []string) bool {
	if origin == "" {
		return false // Géré séparément dans le middleware
	}
	
	for _, allowedOrigin := range allowedOrigins {
		// Support pour wildcard
		if allowedOrigin == "*" {
			return true
		}
		// Comparaison exacte
		if allowedOrigin == origin {
			return true
		}
		// Support pour les sous-domaines (ex: *.example.com)
		if strings.HasPrefix(allowedOrigin, "*.") {
			domain := strings.TrimPrefix(allowedOrigin, "*.")
			if strings.HasSuffix(origin, "."+domain) || origin == domain {
				return true
			}
		}
	}
	return false
}

