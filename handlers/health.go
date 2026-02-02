package handlers

import (
	"net/http"
	"premier-an-backend/database"
	"premier-an-backend/utils"
	"runtime"
	"time"
)

var startTime = time.Now()

// HealthHandler gère les endpoints de santé
type HealthHandler struct {
	environment string
}

// NewHealthHandler crée un nouveau HealthHandler
func NewHealthHandler(environment string) *HealthHandler {
	return &HealthHandler{environment: environment}
}

// Health retourne l'état de santé du serveur avec métriques
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(startTime).String()

	// Vérifier la connexion MongoDB
	dbStatus := "ok"
	if err := database.Ping(); err != nil {
		dbStatus = "error"
	}

	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "ok",
		"message":   "Le serveur fonctionne correctement",
		"env":       h.environment,
		"database":  "MongoDB",
		"db_status": dbStatus,
		"uptime":    uptime,
		"go_version": runtime.Version(),
	})
}
