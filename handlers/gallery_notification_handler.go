package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"premier-an-backend/database"
	"premier-an-backend/middleware"
	"premier-an-backend/utils"
	"strings"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// GalleryNotificationHandler gère les notifications de galerie
type GalleryNotificationHandler struct {
	eventRepo       *database.EventRepository
	userRepo        *database.UserRepository
	inscriptionRepo *database.InscriptionRepository
	fcmTokenRepo    *database.FCMTokenRepository
	fcmService      interface {
		SendToAll(tokens []string, title, body string, data map[string]string) (success int, failed int, failedTokens []string)
	}
	cloudName       string
	previewPreset   string
}

// NewGalleryNotificationHandler crée une nouvelle instance
func NewGalleryNotificationHandler(
	db *mongo.Database,
	fcmService interface {
		SendToAll(tokens []string, title, body string, data map[string]string) (success int, failed int, failedTokens []string)
	},
	cloudName, previewPreset string,
) *GalleryNotificationHandler {
	return &GalleryNotificationHandler{
		eventRepo:       database.NewEventRepository(db),
		userRepo:        database.NewUserRepository(db),
		inscriptionRepo: database.NewInscriptionRepository(db),
		fcmTokenRepo:    database.NewFCMTokenRepository(db),
		fcmService:      fcmService,
		cloudName:       cloudName,
		previewPreset:   previewPreset,
	}
}

// GalleryNotificationRequest représente la requête de notification
type GalleryNotificationRequest struct {
	UserEmail       string `json:"user_email"`
	UserName        string `json:"user_name"`
	MediaCount      int    `json:"media_count"`
	MediaPreviewURL string `json:"media_preview_url"`
	EventTitle      string `json:"event_title"`
}

// SendGalleryNotification gère l'envoi de notifications de galerie
func (h *GalleryNotificationHandler) SendGalleryNotification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.RespondError(w, http.StatusMethodNotAllowed, "Méthode non autorisée")
		return
	}

	// Authentification requise
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Non authentifié")
		return
	}

	vars := mux.Vars(r)
	eventID := vars["eventId"]
	if eventID == "" {
		utils.RespondError(w, http.StatusBadRequest, "ID d'événement requis")
		return
	}

	eventObjID, err := primitive.ObjectIDFromHex(eventID)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "ID d'événement invalide")
		return
	}

	// Décoder la requête
	var req GalleryNotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("❌ Erreur décodage JSON: %v", err)
		utils.RespondError(w, http.StatusBadRequest, "Données JSON invalides")
		return
	}

	// Validation des données
	if req.UserEmail == "" || req.MediaCount <= 0 {
		utils.RespondError(w, http.StatusBadRequest, "user_email et media_count sont requis")
		return
	}

	log.Printf("📱 Envoi notification galerie: %s - %d médias - %s", req.UserName, req.MediaCount, req.EventTitle)

	// 1. Récupérer l'événement
	event, err := h.eventRepo.FindByID(eventObjID)
	if err != nil || event == nil {
		log.Printf("❌ Événement non trouvé: %v", err)
		utils.RespondError(w, http.StatusNotFound, "Événement non trouvé")
		return
	}

	// 2. Récupérer les participants de l'événement (exclure l'utilisateur qui a ajouté)
	participants, err := h.getEventParticipants(eventObjID, req.UserEmail)
	if err != nil {
		log.Printf("❌ Erreur récupération participants: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur lors de la récupération des participants")
		return
	}

	if len(participants) == 0 {
		log.Printf("ℹ️  Aucun participant trouvé pour l'événement %s", eventID)
		utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
			"success":            true,
			"notifications_sent": 0,
			"message":            "Aucun participant à notifier",
		})
		return
	}

	// 3. Générer l'URL de preview avec flou
	previewURL := h.generatePreviewURL(req.MediaPreviewURL)
	log.Printf("🖼️  URL preview générée: %s", previewURL)

	// 4. Construire le message de notification
	title := "Nouveau contenu ajouté"
	body := h.buildNotificationMessage(req.UserName, req.MediaCount, event.Titre)

	// 5. Préparer les données de la notification
	notificationData := map[string]string{
		"type":        "gallery_update",
		"event_id":    eventID,
		"user_name":   req.UserName,
		"media_count": fmt.Sprintf("%d", req.MediaCount),
		"event_title": event.Titre,
		"action_url":  fmt.Sprintf("/galerie-event/%s", eventID),
	}

	// 6. Envoyer les notifications
	successCount, failedCount, failedTokens := h.fcmService.SendToAll(participants, title, body, notificationData)

	// 7. Nettoyer les tokens invalides
	if len(failedTokens) > 0 {
		go h.cleanupInvalidTokens(failedTokens)
	}

	log.Printf("📊 Notifications envoyées: %d succès, %d échecs", successCount, failedCount)

	// 8. Réponse
	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"success":            true,
		"notifications_sent": successCount,
		"failed_count":       failedCount,
		"message":            "Notifications envoyées avec succès",
		"preview_url":        previewURL,
	})
}

// getEventParticipants récupère les participants d'un événement (exclut l'utilisateur qui a ajouté)
func (h *GalleryNotificationHandler) getEventParticipants(eventID primitive.ObjectID, excludeUserEmail string) ([]string, error) {
	// Récupérer les inscriptions de l'événement
	inscriptions, err := h.inscriptionRepo.FindByEventID(eventID)
	if err != nil {
		return nil, err
	}

	var participants []string
	for _, inscription := range inscriptions {
		// Exclure l'utilisateur qui a ajouté les médias
		if inscription.UserEmail == excludeUserEmail {
			continue
		}

		// Récupérer les tokens FCM de l'utilisateur depuis la collection fcm_tokens
		tokens, err := h.fcmTokenRepo.FindByUserID(inscription.UserEmail)
		if err != nil {
			log.Printf("⚠️  Erreur récupération tokens FCM pour %s: %v", inscription.UserEmail, err)
			continue
		}

		// Ajouter tous les tokens valides de cet utilisateur
		for _, token := range tokens {
			if token.Token != "" {
				participants = append(participants, token.Token)
			}
		}
	}

	log.Printf("📱 Participants trouvés: %d tokens pour l'événement %s", len(participants), eventID.Hex())
	return participants, nil
}

// generatePreviewURL génère une URL de preview avec flou
func (h *GalleryNotificationHandler) generatePreviewURL(originalURL string) string {
	if originalURL == "" {
		return ""
	}

	// Ajouter la transformation de flou à l'URL Cloudinary
	// Format: https://res.cloudinary.com/cloud_name/image/upload/TRANSFORMATION/v123/public_id.jpg
	// On ajoute: w_400,h_400,c_fill,q_auto,f_auto,blur_100

	// Vérifier si c'est une URL Cloudinary
	if !strings.Contains(originalURL, "res.cloudinary.com") {
		return originalURL
	}

	// Extraire la partie après /upload/
	parts := strings.Split(originalURL, "/upload/")
	if len(parts) < 2 {
		return originalURL
	}

	// Construire la nouvelle URL avec transformation
	transformation := "w_400,h_400,c_fill,q_auto,f_auto,blur_100"
	newURL := parts[0] + "/upload/" + transformation + "/" + parts[1]

	return newURL
}

// buildNotificationMessage construit le message de notification
func (h *GalleryNotificationHandler) buildNotificationMessage(userName string, mediaCount int, eventTitle string) string {
	if mediaCount == 1 {
		return fmt.Sprintf("%s a ajouté une photo dans la galerie %s", userName, eventTitle)
	}
	return fmt.Sprintf("%s a ajouté %d médias dans la galerie %s", userName, mediaCount, eventTitle)
}

// cleanupInvalidTokens nettoie les tokens FCM invalides
func (h *GalleryNotificationHandler) cleanupInvalidTokens(failedTokens []string) {
	for _, token := range failedTokens {
		log.Printf("🧹 Nettoyage token invalide: %s", token)
		// Ici on pourrait supprimer le token de la base de données
		// Pour l'instant on log juste
	}
}

// TestGalleryNotification endpoint de test pour les notifications
func (h *GalleryNotificationHandler) TestGalleryNotification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.RespondError(w, http.StatusMethodNotAllowed, "Méthode non autorisée")
		return
	}

	// Authentification requise
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Non authentifié")
		return
	}

	vars := mux.Vars(r)
	_ = vars["eventId"] // Variable non utilisée dans le test

	// Données de test
	testData := GalleryNotificationRequest{
		UserEmail:       claims.Email,
		UserName:        "Test User",
		MediaCount:      3,
		MediaPreviewURL: "https://res.cloudinary.com/dxwhngg8g/image/upload/v123/test.jpg",
		EventTitle:      "Test Event",
	}

	// Simuler l'envoi
	previewURL := h.generatePreviewURL(testData.MediaPreviewURL)
	body := h.buildNotificationMessage(testData.UserName, testData.MediaCount, testData.EventTitle)

	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"success":      true,
		"message":      "Test de notification galerie",
		"preview_url":  previewURL,
		"notification": map[string]string{
			"title": "Nouveau contenu ajouté",
			"body":  body,
		},
		"test_data": testData,
	})
}
