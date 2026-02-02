package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"premier-an-backend/constants"
	"premier-an-backend/database"
	"premier-an-backend/middleware"
	"premier-an-backend/models"
	"premier-an-backend/utils"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// EventTrailerHandler gère les trailers vidéo des événements
type EventTrailerHandler struct {
	eventRepo    *database.EventRepository
	cloudName    string
	uploadPreset string
	apiKey       string
	apiSecret    string
}

// NewEventTrailerHandler crée une nouvelle instance
func NewEventTrailerHandler(db *mongo.Database, cloudName, uploadPreset, apiKey, apiSecret string) *EventTrailerHandler {
	return &EventTrailerHandler{
		eventRepo:    database.NewEventRepository(db),
		cloudName:    cloudName,
		uploadPreset: uploadPreset,
		apiKey:       apiKey,
		apiSecret:    apiSecret,
	}
}

// CloudinaryVideoUploadResponse représente la réponse de Cloudinary pour vidéos
type CloudinaryVideoUploadResponse struct {
	PublicID  string  `json:"public_id"`
	SecureURL string  `json:"secure_url"`
	Format    string  `json:"format"`
	Bytes     int64   `json:"bytes"`
	Duration  float64 `json:"duration"`
}

// getEventFromRequest extrait et valide l'événement depuis la requête.
// Retourne (eventObjID, event, true) ou écrit l'erreur et retourne (zero, nil, false).
func (h *EventTrailerHandler) getEventFromRequest(w http.ResponseWriter, r *http.Request) (primitive.ObjectID, *models.Event, bool) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		utils.RespondError(w, http.StatusUnauthorized, constants.ErrNotAuthenticated)
		return primitive.NilObjectID, nil, false
	}
	vars := mux.Vars(r)
	eventID := vars["event_id"]
	if eventID == "" {
		utils.RespondError(w, http.StatusBadRequest, constants.ErrEventIDRequired)
		return primitive.NilObjectID, nil, false
	}
	eventObjID, err := primitive.ObjectIDFromHex(eventID)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, constants.ErrInvalidEventID)
		return primitive.NilObjectID, nil, false
	}
	event, err := h.eventRepo.FindByID(eventObjID)
	if err != nil {
		log.Printf("Événement non trouvé: %v", err)
		utils.RespondError(w, http.StatusNotFound, constants.ErrEventNotFound)
		return primitive.NilObjectID, nil, false
	}
	return eventObjID, event, true
}

// decodeTrailerData décode et valide le body JSON du trailer.
func (h *EventTrailerHandler) decodeTrailerData(w http.ResponseWriter, r *http.Request) (*TrailerDataRequest, bool) {
	var trailerData TrailerDataRequest
	if err := json.NewDecoder(r.Body).Decode(&trailerData); err != nil {
		log.Printf("Erreur décodage JSON trailer: %v", err)
		utils.RespondError(w, http.StatusBadRequest, constants.ErrInvalidJSONBody)
		return nil, false
	}
	if trailerData.URL == "" || trailerData.PublicID == "" {
		utils.RespondError(w, http.StatusBadRequest, "URL et public_id sont requis")
		return nil, false
	}
	return &trailerData, true
}

// TrailerDataRequest représente les données du trailer envoyées par le frontend
type TrailerDataRequest struct {
	URL          string  `json:"url"`
	PublicID     string  `json:"public_id"`
	Duration     float64 `json:"duration"`
	Format       string  `json:"format"`
	Size         int64   `json:"size"`
	ThumbnailURL string  `json:"thumbnail_url"`
}

// UploadTrailer gère l'ajout d'un trailer vidéo (POST)
// Le frontend upload directement vers Cloudinary puis envoie les métadonnées ici
func (h *EventTrailerHandler) UploadTrailer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.RespondError(w, http.StatusMethodNotAllowed, constants.ErrMethodNotAllowed)
		return
	}
	eventObjID, event, ok := h.getEventFromRequest(w, r)
	if !ok {
		return
	}
	if event.Trailer != nil {
		utils.RespondError(w, http.StatusBadRequest, "Cet événement a déjà un trailer. Utilisez PUT pour le remplacer.")
		return
	}
	trailerData, ok := h.decodeTrailerData(w, r)
	if !ok {
		return
	}
	trailer := &models.EventTrailer{
		URL:          trailerData.URL,
		PublicID:     trailerData.PublicID,
		Duration:     trailerData.Duration,
		Format:       trailerData.Format,
		Size:         trailerData.Size,
		UploadedAt:   time.Now(),
		ThumbnailURL: trailerData.ThumbnailURL,
	}
	updateData := bson.M{"trailer": trailer, "updated_at": time.Now()}
	if err := h.eventRepo.Update(eventObjID, updateData); err != nil {
		log.Printf("Erreur mise à jour DB trailer: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, constants.ErrServerError)
		return
	}
	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Trailer ajouté avec succès",
		"trailer": trailer,
	})
}

// ReplaceTrailer gère le remplacement d'un trailer existant (PUT)
// Le frontend upload directement vers Cloudinary puis envoie les métadonnées ici
func (h *EventTrailerHandler) ReplaceTrailer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.RespondError(w, http.StatusMethodNotAllowed, constants.ErrMethodNotAllowed)
		return
	}
	eventObjID, event, ok := h.getEventFromRequest(w, r)
	if !ok {
		return
	}
	if event.Trailer == nil {
		utils.RespondError(w, http.StatusNotFound, "Cet événement n'a pas de trailer à remplacer.")
		return
	}
	oldPublicID := event.Trailer.PublicID
	trailerData, ok := h.decodeTrailerData(w, r)
	if !ok {
		return
	}
	newTrailer := &models.EventTrailer{
		URL:          trailerData.URL,
		PublicID:     trailerData.PublicID,
		Duration:     trailerData.Duration,
		Format:       trailerData.Format,
		Size:         trailerData.Size,
		UploadedAt:   time.Now(),
		ThumbnailURL: trailerData.ThumbnailURL,
	}
	if err := h.deleteVideoFromCloudinary(oldPublicID); err != nil {
		log.Printf("Erreur suppression ancien trailer: %v", err)
	}
	updateData := bson.M{"trailer": newTrailer, "updated_at": time.Now()}
	if err := h.eventRepo.Update(eventObjID, updateData); err != nil {
		log.Printf("Erreur mise à jour DB trailer: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, constants.ErrServerError)
		return
	}
	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Trailer remplacé avec succès",
		"trailer": newTrailer,
	})
}

// DeleteTrailer gère la suppression d'un trailer (DELETE)
func (h *EventTrailerHandler) DeleteTrailer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.RespondError(w, http.StatusMethodNotAllowed, constants.ErrMethodNotAllowed)
		return
	}
	eventObjID, event, ok := h.getEventFromRequest(w, r)
	if !ok {
		return
	}
	if event.Trailer == nil {
		utils.RespondError(w, http.StatusNotFound, "Cet événement n'a pas de trailer à supprimer.")
		return
	}
	publicID := event.Trailer.PublicID
	if err := h.deleteVideoFromCloudinary(publicID); err != nil {
		log.Printf("Erreur suppression Cloudinary: %v", err)
	}
	updateData := bson.M{
		"$unset": bson.M{
			"trailer": "",
		},
		"updated_at": time.Now(),
	}

	if err := h.eventRepo.Update(eventObjID, updateData); err != nil {
		log.Printf("Erreur mise à jour DB trailer: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, constants.ErrServerError)
		return
	}
	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Trailer supprimé avec succès",
	})
}

// deleteVideoFromCloudinary supprime une vidéo de Cloudinary
func (h *EventTrailerHandler) deleteVideoFromCloudinary(publicID string) error {
	// Construire l'URL de suppression
	deleteURL := fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/video/destroy", h.cloudName)

	// Créer les données du formulaire
	data := fmt.Sprintf("public_id=%s&api_key=%s&timestamp=%d", publicID, h.apiKey, time.Now().Unix())

	// Note: Pour une suppression complète, il faudrait signer la requête avec l'API secret
	// Pour l'instant, on utilise une approche simplifiée

	req, err := http.NewRequest("POST", deleteURL, strings.NewReader(data))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.Printf("⚠️  Cloudinary delete error: %s", string(bodyBytes))
		return fmt.Errorf("cloudinary delete returned status %d", resp.StatusCode)
	}

	return nil
}
