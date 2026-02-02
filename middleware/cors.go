package middleware

import (
	"log"
	"net/http"
	"premier-an-backend/constants"
	"strings"
)

// CORS gère les en-têtes CORS
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Vérifier si l'origine est autorisée
			allowed := isOriginAllowed(origin, allowedOrigins)

			// Si pas d'origine (requête depuis une app native ou file://), autoriser par défaut
			// Cela permet aux apps mobiles de fonctionner même sans origine
			if origin == "" {
				allowed = true
				w.Header().Set(constants.HeaderAccessControlAllowOrigin, "*")
			} else if allowed {
				w.Header().Set(constants.HeaderAccessControlAllowOrigin, origin)
			} else {
				log.Printf("CORS: origine non autorisée (URI: %s %s)", r.Method, r.URL.Path)
				w.Header().Set(constants.HeaderAccessControlAllowOrigin, origin)
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, ngrok-skip-browser-warning")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Max-Age", "3600")
			w.Header().Set("Access-Control-Expose-Headers", "Content-Type, Authorization")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			// Pour les requêtes non-OPTIONS, continuer seulement si l'origine est autorisée ou si pas d'origine
			if !allowed && origin != "" {
				log.Printf("CORS: origine non autorisée (URI: %s %s)", r.Method, r.URL.Path)
				w.WriteHeader(http.StatusForbidden)
				w.Header().Set(constants.HeaderContentType, constants.HeaderApplicationJSON)
				_, _ = w.Write([]byte(`{"error":"Forbidden","message":"Origine non autorisée"}`))
				return
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
