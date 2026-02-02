package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"premier-an-backend/database"
	"premier-an-backend/middleware"
	"premier-an-backend/models"
	"premier-an-backend/utils"
	"strings"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// MediaHandler gère les médias des événements
type MediaHandler struct {
	mediaRepo       *database.MediaRepository
	eventRepo       *database.EventRepository
	userRepo        *database.UserRepository
	inscriptionRepo *database.InscriptionRepository
	fcmTokenRepo    *database.FCMTokenRepository
	fcmService      interface {
		SendToAll(tokens []string, title, body string, data map[string]string) (success int, failed int, failedTokens []string)
	}
	cloudName     string
	previewPreset string
}

// NewMediaHandler crée une nouvelle instance
func NewMediaHandler(
	db *mongo.Database,
	fcmService interface {
		SendToAll(tokens []string, title, body string, data map[string]string) (success int, failed int, failedTokens []string)
	},
	cloudName, previewPreset string,
) *MediaHandler {
	return &MediaHandler{
		mediaRepo:       database.NewMediaRepository(db),
		eventRepo:       database.NewEventRepository(db),
		userRepo:        database.NewUserRepository(db),
		inscriptionRepo: database.NewInscriptionRepository(db),
		fcmTokenRepo:    database.NewFCMTokenRepository(db),
		fcmService:      fcmService,
		cloudName:       cloudName,
		previewPreset:   previewPreset,
	}
}

// GetMedias retourne tous les médias d'un événement (PUBLIC)
func (h *MediaHandler) GetMedias(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.RespondError(w, http.StatusMethodNotAllowed, "Méthode non autorisée")
		return
	}

	// Récupérer l'event_id depuis l'URL
	vars := mux.Vars(r)
	eventID, err := primitive.ObjectIDFromHex(vars["event_id"])
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "ID événement invalide")
		return
	}

	// Vérifier que l'événement existe
	event, err := h.eventRepo.FindByID(eventID)
	if err != nil || event == nil {
		utils.RespondError(w, http.StatusNotFound, "Événement non trouvé")
		return
	}

	// Récupérer tous les médias
	medias, err := h.mediaRepo.FindByEvent(eventID)
	if err != nil {
		log.Printf("Erreur récupération médias: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}

	// Si aucun média, retourner un tableau vide
	if medias == nil {
		medias = []models.Media{}
	}

	// Compter les images et vidéos
	totalImages := 0
	totalVideos := 0
	for _, media := range medias {
		if media.Type == "image" {
			totalImages++
		} else if media.Type == "video" {
			totalVideos++
		}
	}

	// Réponse conforme à la spécification (pas de wrapper "data")
	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"photos":  medias,
	})
}

// CreateMedia enregistre un média après upload Firebase
func (h *MediaHandler) CreateMedia(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.RespondError(w, http.StatusMethodNotAllowed, "Méthode non autorisée")
		return
	}

	// Récupérer l'event_id depuis l'URL
	vars := mux.Vars(r)
	eventID, err := primitive.ObjectIDFromHex(vars["event_id"])
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "ID événement invalide")
		return
	}

	// Vérifier que l'événement existe
	event, err := h.eventRepo.FindByID(eventID)
	if err != nil || event == nil {
		utils.RespondError(w, http.StatusNotFound, "Événement non trouvé")
		return
	}

	// Décoder la requête
	var req models.CreateMediaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Données invalides")
		return
	}

	// Validations
	if req.UserEmail == "" {
		utils.RespondError(w, http.StatusBadRequest, "Email utilisateur requis")
		return
	}

	if req.Type != "image" && req.Type != "video" {
		utils.RespondError(w, http.StatusBadRequest, "Type de média invalide. Utilisez 'image' ou 'video'.")
		return
	}

	if req.URL == "" {
		utils.RespondError(w, http.StatusBadRequest, "URL du média requise")
		return
	}

	// Vérifier que l'URL est valide (Firebase ou Cloudinary)
	validURL := strings.HasPrefix(req.URL, "https://firebasestorage.googleapis.com") ||
		strings.HasPrefix(req.URL, "https://res.cloudinary.com") ||
		strings.Contains(req.URL, "cloudinary.com")

	if !validURL {
		utils.RespondError(w, http.StatusBadRequest, "URL de média invalide")
		return
	}

	if req.Filename == "" {
		utils.RespondError(w, http.StatusBadRequest, "Nom de fichier requis")
		return
	}

	// Récupérer l'utilisateur pour obtenir son nom
	user, err := h.userRepo.FindByEmail(req.UserEmail)
	userName := ""
	if err == nil && user != nil {
		userName = fmt.Sprintf("%s %s", user.Firstname, user.Lastname)
	}

	// Créer le média
	media := &models.Media{
		EventID:     eventID,
		UserEmail:   req.UserEmail,
		UserName:    userName,
		Type:        req.Type,
		URL:         req.URL,
		StoragePath: req.StoragePath,
		Filename:    req.Filename,
		Size:        req.Size,
	}

	if err := h.mediaRepo.Create(media); err != nil {
		log.Printf("Erreur création média: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur lors de l'ajout du média")
		return
	}

	// Mettre à jour le compteur photos_count
	totalMedias, _ := h.mediaRepo.CountByEvent(eventID)
	_ = h.eventRepo.Update(eventID, map[string]interface{}{
		"photos_count": int(totalMedias),
	})

	log.Printf("✓ Média ajouté: %s (%s) par %s", req.Filename, req.Type, req.UserEmail)

	// NOUVEAU: Envoyer notification de galerie
	go h.sendGalleryNotification(eventID, req.UserEmail, userName, req.URL)

	utils.RespondJSON(w, http.StatusCreated, map[string]interface{}{
		"message": "Média ajouté avec succès",
		"media":   media,
	})
}

// DeleteMedia supprime un média
func (h *MediaHandler) DeleteMedia(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.RespondError(w, http.StatusMethodNotAllowed, "Méthode non autorisée")
		return
	}

	// Récupérer les IDs depuis l'URL
	vars := mux.Vars(r)
	eventID, err := primitive.ObjectIDFromHex(vars["event_id"])
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "ID événement invalide")
		return
	}

	mediaID, err := primitive.ObjectIDFromHex(vars["media_id"])
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "ID média invalide")
		return
	}

	// Récupérer le média
	media, err := h.mediaRepo.FindByID(mediaID)
	if err != nil {
		log.Printf("Erreur recherche média: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}

	if media == nil {
		utils.RespondError(w, http.StatusNotFound, "Média non trouvé")
		return
	}

	// Vérifier que le média appartient bien à cet événement
	if media.EventID != eventID {
		utils.RespondError(w, http.StatusBadRequest, "Ce média n'appartient pas à cet événement")
		return
	}

	// Vérifier que l'utilisateur authentifié est le propriétaire
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Non autorisé")
		return
	}

	if media.UserEmail != claims.Email {
		utils.RespondError(w, http.StatusForbidden, "Vous ne pouvez supprimer que vos propres médias")
		return
	}

	// Supprimer le média
	if err := h.mediaRepo.Delete(mediaID); err != nil {
		log.Printf("Erreur suppression média: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur lors de la suppression")
		return
	}

	// Mettre à jour le compteur photos_count
	totalMedias, _ := h.mediaRepo.CountByEvent(eventID)
	_ = h.eventRepo.Update(eventID, map[string]interface{}{
		"photos_count": int(totalMedias),
	})

	log.Printf("✓ Média supprimé: %s par %s", media.Filename, claims.Email)

	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"message":  "Média supprimé avec succès",
		"media_id": mediaID.Hex(),
	})
}

// sendGalleryNotification envoie une notification de galerie
func (h *MediaHandler) sendGalleryNotification(eventID primitive.ObjectID, userEmail, userName, mediaURL string) {
	// Récupérer l'événement
	event, err := h.eventRepo.FindByID(eventID)
	if err != nil || event == nil {
		log.Printf("❌ Erreur récupération événement pour notification: %v", err)
		return
	}

	// Récupérer les participants de l'événement (exclure l'utilisateur qui a ajouté)
	participants, err := h.getEventParticipants(eventID, userEmail)
	if err != nil {
		log.Printf("❌ Erreur récupération participants: %v", err)
		return
	}

	if len(participants) == 0 {
		log.Printf("ℹ️  Aucun participant trouvé pour l'événement %s", eventID.Hex())
		return
	}

	// Générer l'URL de preview avec flou
	previewURL := h.generatePreviewURL(mediaURL)
	log.Printf("🖼️  URL preview générée: %s", previewURL)

	// Construire le message de notification
	title := "Nouveau contenu ajouté"
	body := fmt.Sprintf("%s a ajouté une photo dans la galerie %s", userName, event.Titre)

	// Préparer les données de la notification
	notificationData := map[string]string{
		"type":        "gallery_update",
		"event_id":    eventID.Hex(),
		"user_name":   userName,
		"media_count": "1",
		"event_title": event.Titre,
		"action_url":  fmt.Sprintf("/galerie-event/%s", eventID.Hex()),
	}

	// Envoyer les notifications
	successCount, failedCount, failedTokens := h.fcmService.SendToAll(participants, title, body, notificationData)

	// Nettoyer les tokens invalides
	if len(failedTokens) > 0 {
		go h.cleanupInvalidTokens(failedTokens)
	}

	log.Printf("📱 Notification galerie envoyée: %s - %s - %d succès, %d échecs", userName, event.Titre, successCount, failedCount)
}

// getEventParticipants récupère les participants d'un événement (exclut l'utilisateur qui a ajouté)
func (h *MediaHandler) getEventParticipants(eventID primitive.ObjectID, excludeUserEmail string) ([]string, error) {
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
func (h *MediaHandler) generatePreviewURL(originalURL string) string {
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

// cleanupInvalidTokens nettoie les tokens FCM invalides
func (h *MediaHandler) cleanupInvalidTokens(failedTokens []string) {
	for _, token := range failedTokens {
		log.Printf("🧹 Nettoyage token invalide: %s", token)
		// Ici on pourrait supprimer le token de la base de données
		// Pour l'instant on log juste
	}
}
