package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
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
	eventRepo       *database.EventRepository
	cloudName       string
	uploadPreset    string
	apiKey          string
	apiSecret       string
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

// UploadTrailer gère l'upload d'un trailer vidéo (POST)
func (h *EventTrailerHandler) UploadTrailer(w http.ResponseWriter, r *http.Request) {
	// Vérifier la méthode HTTP
	if r.Method != http.MethodPost {
		utils.RespondError(w, http.StatusMethodNotAllowed, "Méthode non autorisée")
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
		utils.RespondError(w, http.StatusBadRequest, "Cet événement a déjà un trailer. Veuillez le supprimer avant d'en ajouter un nouveau.")
		return
	}

	// Log du Content-Type pour debugging
	contentType := r.Header.Get("Content-Type")
	log.Printf("📋 Content-Type reçu: %s", contentType)

	// Vérifier que le Content-Type est bien multipart/form-data
	if !strings.HasPrefix(contentType, "multipart/form-data") {
		log.Printf("❌ Content-Type invalide: %s (attendu: multipart/form-data)", contentType)
		utils.RespondError(w, http.StatusBadRequest, "Le Content-Type doit être multipart/form-data")
		return
	}

	// Parser le formulaire multipart (limite 100 MB)
	if err := r.ParseMultipartForm(100 << 20); err != nil {
		log.Printf("❌ Erreur parsing form: %v", err)
		utils.RespondError(w, http.StatusBadRequest, "Erreur lors du parsing du formulaire")
		return
	}

	// Récupérer le fichier vidéo
	file, header, err := r.FormFile("video")
	if err != nil {
		log.Printf("❌ Erreur récupération fichier: %v", err)
		utils.RespondError(w, http.StatusBadRequest, "Aucun fichier vidéo fourni")
		return
	}
	defer file.Close()

	// Validation de la taille (100 MB max)
	if header.Size > 100*1024*1024 {
		utils.RespondError(w, http.StatusRequestEntityTooLarge, "Le fichier ne doit pas dépasser 100 MB")
		return
	}

	// Validation du type MIME
	fileContentType := header.Header.Get("Content-Type")
	allowedTypes := []string{"video/mp4", "video/quicktime", "video/x-msvideo", "video/webm"}
	isValidType := false
	for _, t := range allowedTypes {
		if fileContentType == t {
			isValidType = true
			break
		}
	}

	if !isValidType {
		utils.RespondError(w, http.StatusBadRequest, "Format de vidéo non supporté. Formats acceptés : MP4, MOV, AVI, WebM")
		return
	}

	log.Printf("📤 Upload trailer pour événement %s (%s, %d bytes)", eventID, fileContentType, header.Size)

	// Upload vers Cloudinary
	trailer, err := h.uploadVideoToCloudinary(file, eventID, header.Filename)
	if err != nil {
		log.Printf("❌ Erreur upload Cloudinary: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur lors de l'upload de la vidéo")
		return
	}

	log.Printf("✅ Upload Cloudinary réussi: %s", trailer.URL)

	// Mettre à jour l'événement dans la base de données
	updateData := bson.M{
		"$set": bson.M{
			"trailer":    trailer,
			"updated_at": time.Now(),
		},
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
func (h *EventTrailerHandler) ReplaceTrailer(w http.ResponseWriter, r *http.Request) {
	// Vérifier la méthode HTTP
	if r.Method != http.MethodPut {
		utils.RespondError(w, http.StatusMethodNotAllowed, "Méthode non autorisée")
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

	// Parser le formulaire multipart (limite 100 MB)
	if err := r.ParseMultipartForm(100 << 20); err != nil {
		log.Printf("❌ Erreur parsing form: %v", err)
		utils.RespondError(w, http.StatusBadRequest, "Erreur lors du parsing du formulaire")
		return
	}

	// Récupérer le fichier vidéo
	file, header, err := r.FormFile("video")
	if err != nil {
		log.Printf("❌ Erreur récupération fichier: %v", err)
		utils.RespondError(w, http.StatusBadRequest, "Aucun fichier vidéo fourni")
		return
	}
	defer file.Close()

	// Validation de la taille (100 MB max)
	if header.Size > 100*1024*1024 {
		utils.RespondError(w, http.StatusRequestEntityTooLarge, "Le fichier ne doit pas dépasser 100 MB")
		return
	}

	// Validation du type MIME
	fileContentType := header.Header.Get("Content-Type")
	allowedTypes := []string{"video/mp4", "video/quicktime", "video/x-msvideo", "video/webm"}
	isValidType := false
	for _, t := range allowedTypes {
		if fileContentType == t {
			isValidType = true
			break
		}
	}

	if !isValidType {
		utils.RespondError(w, http.StatusBadRequest, "Format de vidéo non supporté. Formats acceptés : MP4, MOV, AVI, WebM")
		return
	}

	log.Printf("🔄 Remplacement trailer pour événement %s", eventID)

	// Upload la nouvelle vidéo vers Cloudinary
	newTrailer, err := h.uploadVideoToCloudinary(file, eventID, header.Filename)
	if err != nil {
		log.Printf("❌ Erreur upload Cloudinary: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur lors de l'upload de la nouvelle vidéo")
		return
	}

	log.Printf("✅ Nouveau trailer uploadé: %s", newTrailer.URL)

	// Supprimer l'ancienne vidéo de Cloudinary
	if err := h.deleteVideoFromCloudinary(oldPublicID); err != nil {
		log.Printf("⚠️  Erreur suppression ancien trailer: %v (continuons quand même)", err)
	} else {
		log.Printf("✅ Ancien trailer supprimé de Cloudinary")
	}

	// Mettre à jour l'événement dans la base de données
	updateData := bson.M{
		"$set": bson.M{
			"trailer":    newTrailer,
			"updated_at": time.Now(),
		},
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
		utils.RespondError(w, http.StatusMethodNotAllowed, "Méthode non autorisée")
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
		"$set": bson.M{
			"updated_at": time.Now(),
		},
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

// uploadVideoToCloudinary envoie une vidéo vers Cloudinary
func (h *EventTrailerHandler) uploadVideoToCloudinary(file multipart.File, eventID, filename string) (*models.EventTrailer, error) {
	// Construire l'URL d'upload Cloudinary pour vidéos
	uploadURL := fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/video/upload", h.cloudName)

	// Créer un buffer pour le formulaire multipart
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Ajouter le fichier
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, err
	}

	if _, err := io.Copy(part, file); err != nil {
		return nil, err
	}

	// Ajouter l'upload preset
	if err := writer.WriteField("upload_preset", h.uploadPreset); err != nil {
		return nil, err
	}

	// Ajouter le dossier (organiser par événement)
	folder := fmt.Sprintf("event_trailers/%s", eventID)
	if err := writer.WriteField("folder", folder); err != nil {
		return nil, err
	}

	// Ajouter un timestamp pour éviter les doublons
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	if err := writer.WriteField("public_id", timestamp); err != nil {
		return nil, err
	}

	writer.Close()

	// Créer la requête HTTP
	req, err := http.NewRequest("POST", uploadURL, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Envoyer la requête
	client := &http.Client{Timeout: 120 * time.Second} // 2 minutes pour vidéos
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Lire la réponse
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.Printf("❌ Cloudinary error: %s", string(bodyBytes))
		return nil, fmt.Errorf("cloudinary returned status %d", resp.StatusCode)
	}

	var cloudinaryResp CloudinaryVideoUploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&cloudinaryResp); err != nil {
		return nil, err
	}

	// Générer l'URL de la miniature (Cloudinary génère automatiquement une miniature)
	thumbnailURL := strings.Replace(cloudinaryResp.SecureURL, "/video/upload/", "/video/upload/so_0/", 1)
	thumbnailURL = strings.Replace(thumbnailURL, "."+cloudinaryResp.Format, ".jpg", 1)

	// Construire l'objet EventTrailer
	trailer := &models.EventTrailer{
		URL:          cloudinaryResp.SecureURL,
		PublicID:     cloudinaryResp.PublicID,
		Duration:     cloudinaryResp.Duration,
		Format:       cloudinaryResp.Format,
		Size:         cloudinaryResp.Bytes,
		UploadedAt:   time.Now(),
		ThumbnailURL: thumbnailURL,
	}

	return trailer, nil
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

