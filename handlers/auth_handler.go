package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"premier-an-backend/constants"
	"premier-an-backend/database"
	"premier-an-backend/middleware"
	"premier-an-backend/models"
	"premier-an-backend/utils"
	"strings"

	"go.mongodb.org/mongo-driver/mongo"
)

// AuthHandler gère les requêtes d'authentification
type AuthHandler struct {
	userRepo       *database.UserRepository
	eventRepo      *database.EventRepository
	codeSoireeRepo *database.CodeSoireeRepository
	jwtSecret      string
	fcmService     interface {
		SendToAll(tokens []string, title, body string, data map[string]string) (success int, failed int, failedTokens []string)
	}
	fcmTokenRepo *database.FCMTokenRepository
}

// NewAuthHandler crée une nouvelle instance de AuthHandler
func NewAuthHandler(db *mongo.Database, jwtSecret string, fcmService interface {
	SendToAll(tokens []string, title, body string, data map[string]string) (success int, failed int, failedTokens []string)
}) *AuthHandler {
	return &AuthHandler{
		userRepo:       database.NewUserRepository(db),
		eventRepo:      database.NewEventRepository(db),
		codeSoireeRepo: database.NewCodeSoireeRepository(db),
		jwtSecret:      jwtSecret,
		fcmService:     fcmService,
		fcmTokenRepo:   database.NewFCMTokenRepository(db),
	}
}

// Register gère l'inscription d'un nouvel utilisateur
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	// Vérifier la méthode HTTP
	if !RequireMethod(w, r, http.MethodPost) {
		return
	}

	// Décoder la requête
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Erreur parsing JSON inscription: %v", err)
		utils.RespondError(w, http.StatusBadRequest, constants.ErrInvalidData)
		return
	}

	// Logger les données reçues pour débogage
	log.Println("Inscription reçue")

	// Valider les données
	if err := h.validateRegisterRequest(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Vérifier que le code soirée existe et est actif
	log.Println("Vérification du code soirée")
	codeValid, err := h.codeSoireeRepo.IsCodeValid(req.CodeSoiree)
	if err != nil {
		log.Printf("Erreur lors de la vérification du code soirée: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, constants.ErrServerError)
		return
	}
	if !codeValid {
		log.Println("Code soirée invalide ou inactif")
		utils.RespondError(w, http.StatusBadRequest, constants.ErrCodeInvalidOrInactive)
		return
	}

	log.Println("Code soirée valide")

	// Vérifier si l'email existe déjà
	exists, err := h.userRepo.EmailExists(req.Email)
	if err != nil {
		log.Printf("Erreur lors de la vérification de l'email: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, constants.ErrServerError)
		return
	}
	if exists {
		utils.RespondError(w, http.StatusConflict, constants.ErrEmailAlreadyUsed)
		return
	}

	// Hacher le mot de passe avec bcrypt
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		log.Printf("Erreur lors du hachage du mot de passe: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, constants.ErrServerError)
		return
	}

	// Créer l'utilisateur
	user := &models.User{
		CodeSoiree: req.CodeSoiree,
		Firstname:  req.Firstname,
		Lastname:   req.Lastname,
		Email:      strings.ToLower(strings.TrimSpace(req.Email)),
		Phone:      req.Phone,
		Password:   hashedPassword,
		Admin:      0, // Par défaut, les nouveaux utilisateurs ne sont pas admin
	}

	if err := h.userRepo.Create(user); err != nil {
		log.Printf("Erreur lors de la création de l'utilisateur: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, constants.ErrCreateAccount)
		return
	}

	// Incrémenter le compteur d'utilisations du code soirée
	if err := h.codeSoireeRepo.IncrementUsage(req.CodeSoiree); err != nil {
		log.Printf("Erreur lors de l'incrémentation du code soirée: %v", err)
		// Ne pas bloquer l'inscription si l'incrémentation échoue
	}

	// Générer le token JWT (utiliser l'email comme UserID pour cohérence)
	token, err := utils.GenerateToken(user.Email, user.Email, h.jwtSecret)
	if err != nil {
		log.Printf("Erreur lors de la génération du token: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, constants.ErrServerError)
		return
	}

	// Répondre avec le token et les informations de l'utilisateur
	response := models.AuthResponse{
		Token: token,
		User:  *user,
	}

	log.Println("Nouvel utilisateur inscrit")

	utils.RespondJSON(w, http.StatusCreated, response)
}

// notifyAdminsNewUser envoie une notification aux admins lors d'une nouvelle inscription
func (h *AuthHandler) notifyAdminsNewUser(user *models.User) {
	if h.fcmService == nil {
		return
	}

	// Récupérer tous les admins
	admins, err := h.userRepo.FindAdmins()
	if err != nil {
		log.Printf("⚠️  Erreur récupération admins: %v", err)
		return
	}

	if len(admins) == 0 {
		log.Println("⚠️  Aucun admin à notifier")
		return
	}

	// Récupérer les tokens FCM des admins
	var adminTokens []string
	for _, admin := range admins {
		tokens, err := h.fcmTokenRepo.FindByUserID(admin.Email)
		if err != nil {
			continue
		}
		for _, t := range tokens {
			adminTokens = append(adminTokens, t.Token)
		}
	}

	if len(adminTokens) == 0 {
		log.Println("⚠️  Aucun token FCM pour les admins")
		return
	}

	// Préparer la notification
	title := "🎉 Nouvelle inscription !"
	message := fmt.Sprintf("%s %s vient de s'inscrire", user.Firstname, user.Lastname)
	data := map[string]string{
		"type":      "new_user",
		"user_id":   user.ID.Hex(),
		"email":     user.Email,
		"firstname": user.Firstname,
		"lastname":  user.Lastname,
	}

	// Envoyer aux admins
	success, failed, _ := h.fcmService.SendToAll(adminTokens, title, message, data)
	log.Printf("📧 Notification nouvelle inscription envoyée aux admins: %d succès, %d échecs", success, failed)
}

// Login gère la connexion d'un utilisateur
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	// Vérifier la méthode HTTP
	if !RequireMethod(w, r, http.MethodPost) {
		return
	}

	// Décoder la requête
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, constants.ErrInvalidData)
		return
	}

	// Valider les données
	if err := h.validateLoginRequest(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Rechercher l'utilisateur par email
	email := strings.ToLower(strings.TrimSpace(req.Email))
	user, err := h.userRepo.FindByEmail(email)
	if err != nil {
		log.Printf("Erreur lors de la recherche de l'utilisateur: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, constants.ErrServerError)
		return
	}

	if user == nil {
		utils.RespondError(w, http.StatusUnauthorized, constants.ErrLoginInvalid)
		return
	}

	// Vérifier le mot de passe
	if !utils.CheckPassword(user.Password, req.Password) {
		utils.RespondError(w, http.StatusUnauthorized, constants.ErrLoginInvalid)
		return
	}

	// Générer le token JWT
	token, err := utils.GenerateToken(user.Email, user.Email, h.jwtSecret)
	if err != nil {
		log.Printf("Erreur lors de la génération du token: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, constants.ErrServerError)
		return
	}

	// Répondre avec le token et les informations de l'utilisateur
	response := models.AuthResponse{
		Token: token,
		User:  *user,
	}

	log.Println("Utilisateur connecté")
	utils.RespondJSON(w, http.StatusOK, response)
}

// validateRegisterRequest valide les données d'inscription
func (h *AuthHandler) validateRegisterRequest(req *models.RegisterRequest) error {
	if req.Firstname != "" {
		if err := utils.ValidateRequired("firstname", req.Firstname); err != nil {
			return err
		}
	}
	if req.Lastname != "" {
		if err := utils.ValidateRequired("lastname", req.Lastname); err != nil {
			return err
		}
	}
	if err := utils.ValidateEmail(req.Email); err != nil {
		return err
	}
	if req.Phone != "" {
		if err := utils.ValidatePhone(req.Phone); err != nil {
			return err
		}
	}
	if err := utils.ValidatePassword(req.Password); err != nil {
		return err
	}
	return nil
}

// validateLoginRequest valide les données de connexion
func (h *AuthHandler) validateLoginRequest(req *models.LoginRequest) error {
	if err := utils.ValidateEmail(req.Email); err != nil {
		return err
	}
	if err := utils.ValidatePassword(req.Password); err != nil {
		return err
	}
	return nil
}

// UpdateProfile met à jour le profil utilisateur
func (h *AuthHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	// Vérifier la méthode HTTP
	if r.Method != http.MethodPut && r.Method != http.MethodPatch {
		utils.RespondError(w, http.StatusMethodNotAllowed, constants.ErrMethodNotAllowed)
		return
	}

	// Récupérer l'utilisateur depuis le contexte (mis par le middleware Auth)
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		utils.RespondError(w, http.StatusUnauthorized, constants.ErrAuthTokenInvalid)
		return
	}

	userEmail := claims.Email

	// Décoder la requête
	var req struct {
		Firstname       string `json:"firstname"`
		Lastname        string `json:"lastname"`
		Email           string `json:"email"`
		Phone           string `json:"phone"`
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword"`
		ConfirmPassword string `json:"confirmPassword"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, constants.ErrInvalidData)
		return
	}

	// Validation des champs obligatoires
	if req.Firstname == "" || req.Lastname == "" || req.Email == "" {
		utils.RespondError(w, http.StatusBadRequest, constants.ErrFirstLastEmailRequired)
		return
	}

	// Nettoyer et normaliser l'email
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	// Validation du format de l'email
	if err := utils.ValidateEmail(req.Email); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Format d'email invalide")
		return
	}

	// Vérifier que l'email n'est pas déjà utilisé par un autre utilisateur
	if req.Email != userEmail {
		exists, err := h.userRepo.EmailExists(req.Email)
		if err != nil {
			log.Printf("Erreur vérification email: %v", err)
			utils.RespondError(w, http.StatusInternalServerError, constants.ErrServerError)
			return
		}
		if exists {
			utils.RespondError(w, http.StatusBadRequest, constants.ErrEmailAlreadyUsed)
			return
		}
	}

	// Récupérer l'utilisateur actuel
	user, err := h.userRepo.FindByEmail(userEmail)
	if err != nil || user == nil {
		log.Printf("Erreur récupération utilisateur: %v", err)
		utils.RespondError(w, http.StatusNotFound, "Utilisateur non trouvé")
		return
	}

	// Préparer les données de mise à jour
	updateData := map[string]interface{}{
		"firstname": req.Firstname,
		"lastname":  req.Lastname,
		"email":     req.Email,
		"phone":     req.Phone,
	}

	// Gestion du changement de mot de passe
	hasPasswordFields := req.CurrentPassword != "" || req.NewPassword != "" || req.ConfirmPassword != ""

	if hasPasswordFields {
		// Validation : tous les champs de mot de passe requis
		if req.CurrentPassword == "" || req.NewPassword == "" || req.ConfirmPassword == "" {
			utils.RespondError(w, http.StatusBadRequest, constants.ErrPasswordFieldsRequired)
			return
		}

		// Validation : les nouveaux mots de passe correspondent
		if req.NewPassword != req.ConfirmPassword {
			utils.RespondError(w, http.StatusBadRequest, constants.ErrPasswordMismatch)
			return
		}

		// Validation : longueur minimale du nouveau mot de passe
		if len(req.NewPassword) < 8 {
			utils.RespondError(w, http.StatusBadRequest, constants.ErrPasswordMinLength)
			return
		}

		// Vérifier le mot de passe actuel
		if !utils.CheckPassword(req.CurrentPassword, user.Password) {
			utils.RespondError(w, http.StatusBadRequest, constants.ErrPasswordCurrentWrong)
			return
		}

		// Hasher le nouveau mot de passe
		hashedPassword, err := utils.HashPassword(req.NewPassword)
		if err != nil {
			log.Printf("Erreur hachage mot de passe: %v", err)
			utils.RespondError(w, http.StatusInternalServerError, constants.ErrServerError)
			return
		}

		updateData["password"] = hashedPassword
		log.Println("Changement de mot de passe effectué")
	}

	// Mettre à jour l'utilisateur
	if err := h.userRepo.UpdateByEmail(userEmail, updateData); err != nil {
		log.Printf("Erreur mise à jour utilisateur: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, constants.ErrUpdateProfile)
		return
	}

	// Récupérer l'utilisateur mis à jour
	updatedUser, err := h.userRepo.FindByEmail(req.Email)
	if err != nil || updatedUser == nil {
		log.Printf("Erreur récupération utilisateur mis à jour: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, constants.ErrServerError)
		return
	}

	log.Println("Profil mis à jour")

	// Réponse
	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Profil mis à jour avec succès",
		"user": map[string]interface{}{
			"_id":             updatedUser.ID.Hex(),
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
