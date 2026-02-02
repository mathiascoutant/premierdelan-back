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
	// Vérifier la méthode HTTP
	if r.Method != http.MethodPost {
		utils.RespondError(w, http.StatusMethodNotAllowed, constants.ErrMethodNotAllowed)
		return
	}

	// Vérifier que l'utilisateur est authentifié (l'autorisation admin est gérée par le middleware)
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Non authentifié")
		return
	}

	// Récupérer l'ID de l'événement
	vars := mux.Vars(r)
	eventID := vars["event_id"]

	if eventID == "" {
		utils.RespondError(w, http.StatusBadRequest, "ID d'événement requis")
		return
	}

	// Convertir en ObjectID
	eventObjID, err := primitive.ObjectIDFromHex(eventID)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "ID d'événement invalide")
		return
	}

	// Récupérer l'événement
	event, err := h.eventRepo.FindByID(eventObjID)
	if err != nil {
		log.Printf("❌ Événement non trouvé: %v", err)
		utils.RespondError(w, http.StatusNotFound, "Événement non trouvé")
		return
	}

	// Vérifier que l'événement n'a pas déjà un trailer
	if event.Trailer != nil {
		utils.RespondError(w, http.StatusBadRequest, "Cet événement a déjà un trailer. Utilisez PUT pour le remplacer.")
		return
	}

	// Décoder les données JSON du trailer
	var trailerData TrailerDataRequest
	if err := json.NewDecoder(r.Body).Decode(&trailerData); err != nil {
		log.Printf("❌ Erreur décodage JSON: %v", err)
		utils.RespondError(w, http.StatusBadRequest, "Données JSON invalides")
		return
	}

	// Validation des champs requis
	if trailerData.URL == "" || trailerData.PublicID == "" {
		utils.RespondError(w, http.StatusBadRequest, "URL et public_id sont requis")
		return
	}

	log.Printf("📤 Ajout trailer pour événement %s (format: %s, taille: %d bytes)", eventID, trailerData.Format, trailerData.Size)

	// Créer l'objet EventTrailer
	trailer := &models.EventTrailer{
		URL:          trailerData.URL,
		PublicID:     trailerData.PublicID,
		Duration:     trailerData.Duration,
		Format:       trailerData.Format,
		Size:         trailerData.Size,
		UploadedAt:   time.Now(),
		ThumbnailURL: trailerData.ThumbnailURL,
	}

	log.Printf("✅ Métadonnées trailer reçues: %s", trailer.URL)

	// Mettre à jour l'événement dans la base de données
	updateData := bson.M{
		"trailer":    trailer,
		"updated_at": time.Now(),
	}

	if err := h.eventRepo.Update(eventObjID, updateData); err != nil {
		log.Printf("❌ Erreur mise à jour DB: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur lors de la mise à jour de l'événement")
		return
	}

	log.Printf("✅ Trailer ajouté à l'événement %s", eventID)

	// Réponse
	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Trailer ajouté avec succès",
		"trailer": trailer,
	})
}

// ReplaceTrailer gère le remplacement d'un trailer existant (PUT)
// Le frontend upload directement vers Cloudinary puis envoie les métadonnées ici
func (h *EventTrailerHandler) ReplaceTrailer(w http.ResponseWriter, r *http.Request) {
	// Vérifier la méthode HTTP
	if r.Method != http.MethodPut {
		utils.RespondError(w, http.StatusMethodNotAllowed, constants.ErrMethodNotAllowed)
		return
	}

	// Vérifier que l'utilisateur est authentifié (l'autorisation admin est gérée par le middleware)
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Non authentifié")
		return
	}

	// Récupérer l'ID de l'événement
	vars := mux.Vars(r)
	eventID := vars["event_id"]

	if eventID == "" {
		utils.RespondError(w, http.StatusBadRequest, "ID d'événement requis")
		return
	}

	// Convertir en ObjectID
	eventObjID, err := primitive.ObjectIDFromHex(eventID)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "ID d'événement invalide")
		return
	}

	// Récupérer l'événement
	event, err := h.eventRepo.FindByID(eventObjID)
	if err != nil {
		log.Printf("❌ Événement non trouvé: %v", err)
		utils.RespondError(w, http.StatusNotFound, "Événement non trouvé")
		return
	}

	// Vérifier que l'événement a bien un trailer à remplacer
	if event.Trailer == nil {
		utils.RespondError(w, http.StatusNotFound, "Cet événement n'a pas de trailer à remplacer.")
		return
	}

	// Sauvegarder l'ancien public_id pour suppression
	oldPublicID := event.Trailer.PublicID

	// Décoder les données JSON du nouveau trailer
	var trailerData TrailerDataRequest
	if err := json.NewDecoder(r.Body).Decode(&trailerData); err != nil {
		log.Printf("❌ Erreur décodage JSON: %v", err)
		utils.RespondError(w, http.StatusBadRequest, "Données JSON invalides")
		return
	}

	// Validation des champs requis
	if trailerData.URL == "" || trailerData.PublicID == "" {
		utils.RespondError(w, http.StatusBadRequest, "URL et public_id sont requis")
		return
	}

	log.Printf("🔄 Remplacement trailer pour événement %s (format: %s, taille: %d bytes)", eventID, trailerData.Format, trailerData.Size)

	// Créer l'objet EventTrailer
	newTrailer := &models.EventTrailer{
		URL:          trailerData.URL,
		PublicID:     trailerData.PublicID,
		Duration:     trailerData.Duration,
		Format:       trailerData.Format,
		Size:         trailerData.Size,
		UploadedAt:   time.Now(),
		ThumbnailURL: trailerData.ThumbnailURL,
	}

	log.Printf("✅ Nouveau trailer reçu: %s", newTrailer.URL)

	// Supprimer l'ancienne vidéo de Cloudinary
	if err := h.deleteVideoFromCloudinary(oldPublicID); err != nil {
		log.Printf("⚠️  Erreur suppression ancien trailer: %v (continuons quand même)", err)
	} else {
		log.Printf("✅ Ancien trailer supprimé de Cloudinary")
	}

	// Mettre à jour l'événement dans la base de données
	updateData := bson.M{
		"trailer":    newTrailer,
		"updated_at": time.Now(),
	}

	if err := h.eventRepo.Update(eventObjID, updateData); err != nil {
		log.Printf("❌ Erreur mise à jour DB: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur lors de la mise à jour de l'événement")
		return
	}

	log.Printf("✅ Trailer remplacé pour l'événement %s", eventID)

	// Réponse
	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Trailer remplacé avec succès",
		"trailer": newTrailer,
	})
}

// DeleteTrailer gère la suppression d'un trailer (DELETE)
func (h *EventTrailerHandler) DeleteTrailer(w http.ResponseWriter, r *http.Request) {
	// Vérifier la méthode HTTP
	if r.Method != http.MethodDelete {
		utils.RespondError(w, http.StatusMethodNotAllowed, constants.ErrMethodNotAllowed)
		return
	}

	// Vérifier que l'utilisateur est authentifié (l'autorisation admin est gérée par le middleware)
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Non authentifié")
		return
	}

	// Récupérer l'ID de l'événement
	vars := mux.Vars(r)
	eventID := vars["event_id"]

	if eventID == "" {
		utils.RespondError(w, http.StatusBadRequest, "ID d'événement requis")
		return
	}

	// Convertir en ObjectID
	eventObjID, err := primitive.ObjectIDFromHex(eventID)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "ID d'événement invalide")
		return
	}

	// Récupérer l'événement
	event, err := h.eventRepo.FindByID(eventObjID)
	if err != nil {
		log.Printf("❌ Événement non trouvé: %v", err)
		utils.RespondError(w, http.StatusNotFound, "Événement non trouvé")
		return
	}

	// Vérifier que l'événement a bien un trailer à supprimer
	if event.Trailer == nil {
		utils.RespondError(w, http.StatusNotFound, "Cet événement n'a pas de trailer à supprimer.")
		return
	}

	publicID := event.Trailer.PublicID
	log.Printf("🗑️  Suppression trailer pour événement %s (public_id: %s)", eventID, publicID)

	// Supprimer la vidéo de Cloudinary
	if err := h.deleteVideoFromCloudinary(publicID); err != nil {
		log.Printf("⚠️  Erreur suppression Cloudinary: %v (continuons quand même)", err)
	} else {
		log.Printf("✅ Trailer supprimé de Cloudinary")
	}

	// Mettre à jour l'événement dans la base de données (supprimer le champ trailer)
	updateData := bson.M{
		"$unset": bson.M{
			"trailer": "",
		},
		"updated_at": time.Now(),
	}

	if err := h.eventRepo.Update(eventObjID, updateData); err != nil {
		log.Printf("❌ Erreur mise à jour DB: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur lors de la mise à jour de l'événement")
		return
	}

	log.Printf("✅ Trailer supprimé de l'événement %s", eventID)

	// Réponse
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
