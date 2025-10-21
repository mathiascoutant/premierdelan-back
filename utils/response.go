package utils

import (
	"encoding/json"
	"net/http"
	"premier-an-backend/models"
)

// RespondJSON envoie une réponse JSON
func RespondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			http.Error(w, "Erreur lors de l'encodage JSON", http.StatusInternalServerError)
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

