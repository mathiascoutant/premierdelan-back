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
	mediaRepo *database.MediaRepository
	eventRepo *database.EventRepository
	userRepo  *database.UserRepository
}

// NewMediaHandler crée une nouvelle instance
func NewMediaHandler(db *mongo.Database) *MediaHandler {
	return &MediaHandler{
		mediaRepo: database.NewMediaRepository(db),
		eventRepo: database.NewEventRepository(db),
		userRepo:  database.NewUserRepository(db),
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
	h.eventRepo.Update(eventID, map[string]interface{}{
		"photos_count": int(totalMedias),
	})

	log.Printf("✓ Média ajouté: %s (%s) par %s", req.Filename, req.Type, req.UserEmail)

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
	h.eventRepo.Update(eventID, map[string]interface{}{
		"photos_count": int(totalMedias),
	})

	log.Printf("✓ Média supprimé: %s par %s", media.Filename, claims.Email)

	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"message":  "Média supprimé avec succès",
		"media_id": mediaID.Hex(),
	})
}

