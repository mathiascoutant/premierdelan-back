package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"premier-an-backend/constants"
	"premier-an-backend/database"
	"premier-an-backend/models"
	"premier-an-backend/services"
	"premier-an-backend/utils"

	"go.mongodb.org/mongo-driver/mongo"
)

// FCMHandler gère les requêtes de notifications FCM
type FCMHandler struct {
	fcmService *services.FCMService
	tokenRepo  *database.FCMTokenRepository
}

// NewFCMHandler crée une nouvelle instance de FCMHandler
func NewFCMHandler(db *mongo.Database, fcmService *services.FCMService) *FCMHandler {
	return &FCMHandler{
		fcmService: fcmService,
		tokenRepo:  database.NewFCMTokenRepository(db),
	}
}

// Subscribe enregistre un token FCM pour un utilisateur
func (h *FCMHandler) Subscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.RespondError(w, http.StatusMethodNotAllowed, constants.ErrMethodNotAllowed)
		return
	}

	var req models.FCMSubscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Données invalides")
		return
	}

	// Valider les données
	if req.UserID == "" || req.FCMToken == "" {
		utils.RespondError(w, http.StatusBadRequest, "user_id et fcm_token sont requis")
		return
	}

	// Créer ou mettre à jour le token
	token := &models.FCMToken{
		UserID:    req.UserID,
		Token:     req.FCMToken,
		Device:    req.Device,
		UserAgent: req.UserAgent,
	}

	if err := h.tokenRepo.Upsert(token); err != nil {
		log.Printf("Erreur lors de l'enregistrement du token FCM: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, constants.ErrServerError)
		return
	}

	log.Println("Token FCM enregistré")
	utils.RespondSuccess(w, "Abonnement FCM réussi", token)
}

// Unsubscribe supprime un token FCM
func (h *FCMHandler) Unsubscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.RespondError(w, http.StatusMethodNotAllowed, constants.ErrMethodNotAllowed)
		return
	}

	var req struct {
		FCMToken string `json:"fcm_token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Données invalides")
		return
	}

	if req.FCMToken == "" {
		utils.RespondError(w, http.StatusBadRequest, "fcm_token est requis")
		return
	}

	if err := h.tokenRepo.Delete(req.FCMToken); err != nil {
		log.Printf("Erreur lors de la suppression du token: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, constants.ErrServerError)
		return
	}

	log.Println("Token FCM supprimé")
	utils.RespondSuccess(w, "Désabonnement réussi", nil)
}

// SendNotification envoie une notification à tous les abonnés
func (h *FCMHandler) SendNotification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.RespondError(w, http.StatusMethodNotAllowed, constants.ErrMethodNotAllowed)
		return
	}

	var req models.FCMNotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Données invalides")
		return
	}

	// Récupérer tous les tokens
	allTokens, err := h.tokenRepo.FindAll()
	if err != nil {
		log.Printf("Erreur lors de la récupération des tokens: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, constants.ErrServerError)
		return
	}

	if len(allTokens) == 0 {
		utils.RespondSuccess(w, "Aucun abonné trouvé", models.FCMNotificationResponse{
			Success: 0,
			Failed:  0,
			Total:   0,
		})
		return
	}

	// Extraire les tokens
	tokens := make([]string, len(allTokens))
	for i, t := range allTokens {
		tokens[i] = t.Token
	}

	// Préparer le message
	title := req.Title
	if title == "" {
		title = "Nouvelle notification"
	}

	message := req.Message
	if message == "" {
		message = "Vous avez reçu une nouvelle notification"
	}

	// Envoyer les notifications
	success, failed, failedTokens := h.fcmService.SendToAll(tokens, title, message, req.Data)

	// Supprimer les tokens invalides
	for _, failedToken := range failedTokens {
		if err := h.tokenRepo.Delete(failedToken); err != nil {
			log.Printf("⚠️  Erreur lors de la suppression du token invalide: %v", err)
		} else {
			log.Println("Token invalide supprimé")
		}
	}

	response := models.FCMNotificationResponse{
		Success:      success,
		Failed:       failed,
		Total:        len(allTokens),
		FailedTokens: failedTokens,
	}

	log.Printf("📊 Notifications FCM envoyées: %d succès, %d échecs sur %d total", success, failed, len(allTokens))
	utils.RespondSuccess(w, "Notifications envoyées", response)
}

// SendToUser envoie une notification à un utilisateur spécifique
func (h *FCMHandler) SendToUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.RespondError(w, http.StatusMethodNotAllowed, constants.ErrMethodNotAllowed)
		return
	}

	var req models.FCMNotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Données invalides")
		return
	}

	if req.UserID == "" {
		utils.RespondError(w, http.StatusBadRequest, "user_id est requis")
		return
	}

	// Récupérer les tokens de l'utilisateur
	userTokens, err := h.tokenRepo.FindByUserID(req.UserID)
	if err != nil {
		log.Printf("Erreur lors de la récupération des tokens: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, constants.ErrServerError)
		return
	}

	if len(userTokens) == 0 {
		utils.RespondError(w, http.StatusNotFound, "Aucun token trouvé pour cet utilisateur")
		return
	}

	// Extraire les tokens
	tokens := make([]string, len(userTokens))
	for i, t := range userTokens {
		tokens[i] = t.Token
	}

	// Préparer le message
	title := req.Title
	if title == "" {
		title = "Nouvelle notification"
	}

	message := req.Message
	if message == "" {
		message = "Vous avez reçu une nouvelle notification"
	}

	// Envoyer les notifications
	success, failed, failedTokens := h.fcmService.SendToAll(tokens, title, message, req.Data)

	// Supprimer les tokens invalides
	for _, failedToken := range failedTokens {
		_ = h.tokenRepo.Delete(failedToken)
	}

	response := models.FCMNotificationResponse{
		Success:      success,
		Failed:       failed,
		Total:        len(userTokens),
		FailedTokens: failedTokens,
	}

	log.Printf("Notifications envoyées: %d succès, %d échecs", success, failed)
	utils.RespondSuccess(w, "Notifications envoyées", response)
}
