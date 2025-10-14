package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"premier-an-backend/database"
	"premier-an-backend/models"
	"premier-an-backend/utils"
	"strings"

	"go.mongodb.org/mongo-driver/mongo"
)

// AuthHandler gère les requêtes d'authentification
type AuthHandler struct {
	userRepo         *database.UserRepository
	eventRepo        *database.EventRepository
	codeSoireeRepo   *database.CodeSoireeRepository
	jwtSecret        string
	fcmService       interface {
		SendToAll(tokens []string, title, body string, data map[string]string) (success int, failed int, failedTokens []string)
	}
	fcmTokenRepo     *database.FCMTokenRepository
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
	if r.Method != http.MethodPost {
		utils.RespondError(w, http.StatusMethodNotAllowed, "Méthode non autorisée")
		return
	}

	// Décoder la requête
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Données invalides")
		return
	}

	// Valider les données
	if err := h.validateRegisterRequest(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Vérifier que le code soirée existe et est actif
	codeValid, err := h.codeSoireeRepo.IsCodeValid(req.CodeSoiree)
	if err != nil {
		log.Printf("Erreur lors de la vérification du code soirée: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}
	if !codeValid {
		utils.RespondError(w, http.StatusBadRequest, "Code de soirée invalide ou inactif")
		return
	}

	// Vérifier si l'email existe déjà
	exists, err := h.userRepo.EmailExists(req.Email)
	if err != nil {
		log.Printf("Erreur lors de la vérification de l'email: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}
	if exists {
		utils.RespondError(w, http.StatusConflict, "Cet email est déjà utilisé")
		return
	}

	// Hacher le mot de passe avec bcrypt
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		log.Printf("Erreur lors du hachage du mot de passe: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur serveur")
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
		utils.RespondError(w, http.StatusInternalServerError, "Erreur lors de la création du compte")
		return
	}

	// Incrémenter le compteur d'utilisations du code soirée
	if err := h.codeSoireeRepo.IncrementUsage(req.CodeSoiree); err != nil {
		log.Printf("Erreur lors de l'incrémentation du code soirée: %v", err)
		// Ne pas bloquer l'inscription si l'incrémentation échoue
	}

	// Générer le token JWT
	token, err := utils.GenerateToken(user.ID.Hex(), user.Email, h.jwtSecret)
	if err != nil {
		log.Printf("Erreur lors de la génération du token: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}

	// Répondre avec le token et les informations de l'utilisateur
	response := map[string]interface{}{
		"success": true,
		"token":   token,
		"user":    *user,
	}

	log.Printf("✓ Nouvel utilisateur inscrit: %s (ID: %s)", user.Email, user.ID.Hex())
	
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
	if r.Method != http.MethodPost {
		utils.RespondError(w, http.StatusMethodNotAllowed, "Méthode non autorisée")
		return
	}

	// Décoder la requête
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Données invalides")
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
		utils.RespondError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}

	if user == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Email ou mot de passe incorrect")
		return
	}

	// Vérifier le mot de passe
	if !utils.CheckPassword(user.Password, req.Password) {
		utils.RespondError(w, http.StatusUnauthorized, "Email ou mot de passe incorrect")
		return
	}

	// Générer le token JWT
	token, err := utils.GenerateToken(user.ID.Hex(), user.Email, h.jwtSecret)
	if err != nil {
		log.Printf("Erreur lors de la génération du token: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}

	// Répondre avec le token et les informations de l'utilisateur
	response := map[string]interface{}{
		"success": true,
		"token":   token,
		"user":    *user,
	}

	log.Printf("✓ Utilisateur connecté: %s (ID: %s)", user.Email, user.ID.Hex())
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
