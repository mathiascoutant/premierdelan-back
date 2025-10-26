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
	"premier-an-backend/utils"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

// CloudinaryHandler gère les uploads vers Cloudinary
type CloudinaryHandler struct {
	userRepo      *database.UserRepository
	cloudName     string
	uploadPreset  string
	apiKey        string
	apiSecret     string
}

// NewCloudinaryHandler crée une nouvelle instance
func NewCloudinaryHandler(db *mongo.Database, cloudName, uploadPreset, apiKey, apiSecret string) *CloudinaryHandler {
	return &CloudinaryHandler{
		userRepo:     database.NewUserRepository(db),
		cloudName:    cloudName,
		uploadPreset: uploadPreset,
		apiKey:       apiKey,
		apiSecret:    apiSecret,
	}
}

// CloudinaryUploadResponse représente la réponse de Cloudinary
type CloudinaryUploadResponse struct {
	PublicID  string `json:"public_id"`
	SecureURL string `json:"secure_url"`
	Format    string `json:"format"`
	Bytes     int    `json:"bytes"`
}

// UploadProfileImage gère l'upload de la photo de profil
func (h *CloudinaryHandler) UploadProfileImage(w http.ResponseWriter, r *http.Request) {
	// Vérifier la méthode HTTP
	if r.Method != http.MethodPost {
		utils.RespondError(w, http.StatusMethodNotAllowed, "Méthode non autorisée")
		return
	}

	// Récupérer l'utilisateur depuis le contexte (mis par le middleware Auth)
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Token d'authentification invalide")
		return
	}

	userEmail := claims.Email

	// Parser le formulaire multipart (limite 10 MB)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		log.Printf("Erreur parsing form: %v", err)
		utils.RespondError(w, http.StatusBadRequest, "Erreur lors du parsing du formulaire")
		return
	}

	// Récupérer le fichier
	file, header, err := r.FormFile("profileImage")
	if err != nil {
		log.Printf("Erreur récupération fichier: %v", err)
		utils.RespondError(w, http.StatusBadRequest, "Aucun fichier fourni")
		return
	}
	defer file.Close()

	// Validation de la taille (5 MB max)
	if header.Size > 5*1024*1024 {
		utils.RespondError(w, http.StatusRequestEntityTooLarge, "Le fichier ne doit pas dépasser 5 MB")
		return
	}

	// Validation du type MIME
	contentType := header.Header.Get("Content-Type")
	allowedTypes := []string{"image/jpeg", "image/jpg", "image/png", "image/webp", "image/gif"}
	isValidType := false
	for _, t := range allowedTypes {
		if contentType == t {
			isValidType = true
			break
		}
	}

	if !isValidType {
		utils.RespondError(w, http.StatusBadRequest, "Format de fichier non supporté. Formats acceptés : JPEG, PNG, WebP, GIF")
		return
	}

	log.Printf("📤 Upload photo de profil pour %s (%s, %d bytes)", userEmail, contentType, header.Size)

	// Récupérer l'utilisateur
	user, err := h.userRepo.FindByEmail(userEmail)
	if err != nil || user == nil {
		log.Printf("Erreur récupération utilisateur: %v", err)
		utils.RespondError(w, http.StatusNotFound, "Utilisateur non trouvé")
		return
	}

	// Upload vers Cloudinary
	cloudinaryURL, err := h.uploadToCloudinary(file, userEmail, header.Filename)
	if err != nil {
		log.Printf("Erreur upload Cloudinary: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur lors de l'upload de l'image")
		return
	}

	log.Printf("✅ Upload Cloudinary réussi: %s", cloudinaryURL)

	// Mettre à jour la base de données
	updateData := map[string]interface{}{
		"profile_image_url": cloudinaryURL,
	}

	if err := h.userRepo.UpdateByEmail(userEmail, updateData); err != nil {
		log.Printf("Erreur mise à jour DB: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur lors de la mise à jour du profil")
		return
	}

	// Récupérer l'utilisateur mis à jour
	updatedUser, err := h.userRepo.FindByEmail(userEmail)
	if err != nil || updatedUser == nil {
		log.Printf("Erreur récupération utilisateur mis à jour: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}

	log.Printf("✅ Photo de profil mise à jour: %s", userEmail)

	// Réponse
	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"success":  true,
		"message":  "Photo de profil mise à jour avec succès",
		"imageUrl": cloudinaryURL,
		"user": map[string]interface{}{
			"id":              updatedUser.ID.Hex(),
			"firstname":       updatedUser.Firstname,
			"lastname":        updatedUser.Lastname,
			"email":           updatedUser.Email,
			"phone":           updatedUser.Phone,
			"profileImageUrl": updatedUser.ProfileImageURL,
			"admin":           updatedUser.Admin,
			"code_soiree":     updatedUser.CodeSoiree,
		},
	})
}

// uploadToCloudinary envoie le fichier vers Cloudinary
func (h *CloudinaryHandler) uploadToCloudinary(file multipart.File, userEmail, filename string) (string, error) {
	// Construire l'URL d'upload Cloudinary
	uploadURL := fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/image/upload", h.cloudName)

	// Créer un buffer pour le formulaire multipart
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Ajouter le fichier
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return "", err
	}

	if _, err := io.Copy(part, file); err != nil {
		return "", err
	}

	// Ajouter l'upload preset
	if err := writer.WriteField("upload_preset", h.uploadPreset); err != nil {
		return "", err
	}

	// Ajouter le dossier (organiser par utilisateur)
	folder := fmt.Sprintf("profiles/%s", strings.Replace(userEmail, "@", "_", -1))
	if err := writer.WriteField("folder", folder); err != nil {
		return "", err
	}

	// Ajouter un timestamp pour éviter les doublons
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	if err := writer.WriteField("public_id", timestamp); err != nil {
		return "", err
	}

	// Transformation automatique: resize à 400x400, format auto, qualité auto
	if err := writer.WriteField("transformation", "c_fill,w_400,h_400,q_auto,f_auto"); err != nil {
		return "", err
	}

	writer.Close()

	// Créer la requête HTTP
	req, err := http.NewRequest("POST", uploadURL, body)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Envoyer la requête
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Lire la réponse
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.Printf("❌ Cloudinary error: %s", string(bodyBytes))
		return "", fmt.Errorf("cloudinary returned status %d", resp.StatusCode)
	}

	var cloudinaryResp CloudinaryUploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&cloudinaryResp); err != nil {
		return "", err
	}

	return cloudinaryResp.SecureURL, nil
}

