package utils

import (
	"encoding/json"
	"net/http"
	"premier-an-backend/models"
)

// RespondJSON envoie une réponse JSON
func RespondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	// S'assurer que les en-têtes ne sont pas déjà écrits
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "application/json")
	}
	
	// Écrire le code de statut
	if statusCode > 0 {
		w.WriteHeader(statusCode)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	
	// Encoder et envoyer les données
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			// Si l'encodage échoue, essayer d'envoyer une erreur simple
			// Mais seulement si les en-têtes n'ont pas encore été écrits
			if statusCode == http.StatusOK {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error":"Internal Server Error","message":"Erreur lors de l'encodage JSON"}`))
			}
		}
	}
}

// RespondError envoie une réponse d'erreur JSON
func RespondError(w http.ResponseWriter, statusCode int, message string) {
	RespondJSON(w, statusCode, models.ErrorResponse{
		Error:   http.StatusText(statusCode),
		Message: message,
	})
}

// RespondSuccess envoie une réponse de succès JSON
func RespondSuccess(w http.ResponseWriter, message string, data interface{}) {
	RespondJSON(w, http.StatusOK, models.SuccessResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

