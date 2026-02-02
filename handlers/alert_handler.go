package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"premier-an-backend/constants"
	"premier-an-backend/database"
	"premier-an-backend/models"
	"premier-an-backend/utils"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

// AlertHandler gère les alertes critiques
type AlertHandler struct {
	alertRepo    *database.AlertRepository
	userRepo     *database.UserRepository
	fcmTokenRepo *database.FCMTokenRepository
	fcmService   interface {
		SendToAll(tokens []string, title, body string, data map[string]string) (success int, failed int, failedTokens []string)
	}
}

// NewAlertHandler crée une nouvelle instance
func NewAlertHandler(db *mongo.Database, fcmService interface {
	SendToAll(tokens []string, title, body string, data map[string]string) (success int, failed int, failedTokens []string)
}) *AlertHandler {
	return &AlertHandler{
		alertRepo:    database.NewAlertRepository(db),
		userRepo:     database.NewUserRepository(db),
		fcmTokenRepo: database.NewFCMTokenRepository(db),
		fcmService:   fcmService,
	}
}

// SendCriticalAlert reçoit et traite une alerte critique du frontend
func (h *AlertHandler) SendCriticalAlert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.RespondError(w, http.StatusMethodNotAllowed, constants.ErrMethodNotAllowed)
		return
	}

	// Décoder la requête
	var req models.CriticalAlertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, constants.ErrInvalidData)
		return
	}

	// Validation des champs requis
	if req.AdminEmail == "" || req.ErrorType == "" || req.ErrorMessage == "" || req.EndpointFailed == "" {
		utils.RespondError(w, http.StatusBadRequest, constants.ErrAlertFieldsInvalid)
		return
	}

	// Limiter la longueur des messages
	if len(req.ErrorMessage) > 500 {
		req.ErrorMessage = req.ErrorMessage[:500]
	}

	// Rate limiting : Maximum 5 alertes par minute pour cet admin
	count, err := h.alertRepo.CountRecentAlerts(req.AdminEmail, 1)
	if err != nil {
		log.Printf("Erreur vérification rate limit: %v", err)
	}

	if count >= 5 {
		utils.RespondJSON(w, http.StatusTooManyRequests, map[string]interface{}{
			"error":       true,
			"message":     "Trop d'alertes envoyées. Veuillez patienter.",
			"retry_after": 60,
		})
		return
	}

	// Vérifier que l'admin existe
	admin, err := h.userRepo.FindByEmail(req.AdminEmail)
	if err != nil || admin == nil {
		log.Println("Admin non trouvé pour alerte")
		utils.RespondError(w, http.StatusNotFound, constants.ErrAdminNotFound)
		return
	}

	// Vérifier que l'utilisateur est bien admin
	if admin.Admin != 1 {
		utils.RespondError(w, http.StatusForbidden, constants.ErrNotAdmin)
		return
	}

	// Récupérer les tokens FCM de l'admin
	fcmTokens, err := h.fcmTokenRepo.FindByUserID(admin.ID.Hex())
	if err != nil {
		log.Printf("Erreur récupération tokens FCM admin: %v", err)
	}

	if len(fcmTokens) == 0 {
		// Pas de token FCM, mais on enregistre quand même l'alerte
		log.Println("Admin sans token FCM pour alerte")
	}

	// Parser le timestamp
	timestamp, err := time.Parse(time.RFC3339, req.Timestamp)
	if err != nil {
		timestamp = time.Now() // Fallback
	}

	// Créer l'alerte en DB
	alert := &models.CriticalAlert{
		AdminEmail:       req.AdminEmail,
		ErrorType:        req.ErrorType,
		ErrorMessage:     req.ErrorMessage,
		EndpointFailed:   req.EndpointFailed,
		Timestamp:        timestamp,
		UserAgent:        req.UserAgent,
		NotificationSent: false,
	}

	if err := h.alertRepo.Create(alert); err != nil {
		log.Printf("Erreur création alerte: %v", err)
	}

	// Si pas de tokens FCM, retourner quand même un succès
	if len(fcmTokens) == 0 {
		utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
			"success":           true,
			"message":           "Alerte enregistrée mais admin sans token FCM",
			"notification_sent": false,
		})
		return
	}

	// Extraire les tokens
	var tokens []string
	for _, t := range fcmTokens {
		tokens = append(tokens, t.Token)
	}

	// Construire la notification
	title := "🚨 Alerte Critique - Site"
	body := fmt.Sprintf("%s: %s", getErrorTypeLabel(req.ErrorType), req.ErrorMessage)

	data := map[string]string{
		"type":         "critical_error",
		"error_type":   req.ErrorType,
		"endpoint":     req.EndpointFailed,
		"timestamp":    req.Timestamp,
		"user_agent":   req.UserAgent,
		"click_action": "https://mathiascoutant.github.io/premierdelan/maintenance",
	}

	// Envoyer la notification
	success, failed, _ := h.fcmService.SendToAll(tokens, title, body, data)

	// Mettre à jour l'alerte
	if success > 0 {
		alert.NotificationSent = true
	}

	log.Printf("Alerte critique envoyée: %d succès, %d échecs", success, failed)

	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"success":           true,
		"message":           "Alerte envoyée à l'administrateur",
		"notification_sent": success > 0,
		"tokens_sent":       len(tokens),
		"success_count":     success,
	})
}

// getErrorTypeLabel retourne le label français pour un type d'erreur
func getErrorTypeLabel(errorType string) string {
	switch errorType {
	case "SERVER_ERROR":
		return "Erreur Serveur"
	case "NETWORK_ERROR":
		return "Erreur Réseau"
	case "CONNECTION_ERROR":
		return "Erreur Connexion"
	default:
		return "Erreur Inconnue"
	}
}
