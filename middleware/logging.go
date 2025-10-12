package middleware

import (
	"log"
	"net/http"
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

// Logging enregistre les requêtes HTTP
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Créer un wrapper pour capturer le code de statut
		rw := newResponseWriter(w)

		// Traiter la requête
		next.ServeHTTP(rw, r)

		// Enregistrer les informations
		duration := time.Since(start)
		log.Printf(
			"%s %s %d %s",
			r.Method,
			r.RequestURI,
			rw.statusCode,
			duration,
		)
	})
}

