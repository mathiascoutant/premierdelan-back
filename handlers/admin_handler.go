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
	"time"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// AdminHandler gère les requêtes admin
type AdminHandler struct {
	userRepo        *database.UserRepository
	eventRepo       *database.EventRepository
	inscriptionRepo *database.InscriptionRepository
	mediaRepo       *database.MediaRepository
	codeSoireeRepo  *database.CodeSoireeRepository
	fcmService      interface {
		SendToAll(tokens []string, title, body string, data map[string]string) (success int, failed int, failedTokens []string)
	}
	fcmTokenRepo *database.FCMTokenRepository
	wsHub        WebSocketHub
}

// NewAdminHandler crée une nouvelle instance de AdminHandler
func NewAdminHandler(db *mongo.Database, fcmService interface {
	SendToAll(tokens []string, title, body string, data map[string]string) (success int, failed int, failedTokens []string)
}, wsHub WebSocketHub) *AdminHandler {
	return &AdminHandler{
		userRepo:        database.NewUserRepository(db),
		eventRepo:       database.NewEventRepository(db),
		inscriptionRepo: database.NewInscriptionRepository(db),
		mediaRepo:       database.NewMediaRepository(db),
		codeSoireeRepo:  database.NewCodeSoireeRepository(db),
		fcmService:      fcmService,
		fcmTokenRepo:    database.NewFCMTokenRepository(db),
		wsHub:           wsHub,
	}
}

// ========== GESTION DES UTILISATEURS ==========

// GetUsers retourne la liste de tous les utilisateurs
func (h *AdminHandler) GetUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.RespondError(w, http.StatusMethodNotAllowed, "Méthode non autorisée")
		return
	}

	// Utiliser FindAll du repository
	users, err := h.userRepo.FindAll()
	if err != nil {
		log.Printf("Erreur lors de la récupération des utilisateurs: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}

	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"utilisateurs": users,
	})
}

// UpdateUser met à jour un utilisateur
func (h *AdminHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.RespondError(w, http.StatusMethodNotAllowed, "Méthode non autorisée")
		return
	}

	// Récupérer l'ID depuis l'URL
	vars := mux.Vars(r)
	userID, err := primitive.ObjectIDFromHex(vars["id"])
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "ID utilisateur invalide")
		return
	}

	// Décoder la requête
	var req models.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Données invalides")
		return
	}

	// Récupérer l'utilisateur AVANT la mise à jour pour comparer le statut admin
	var oldAdminStatus int
	var adminStatusChanged bool
	if req.Admin != nil {
		oldUser, err := h.userRepo.FindByID(userID)
		if err == nil && oldUser != nil {
			oldAdminStatus = oldUser.Admin
			adminStatusChanged = (oldAdminStatus != *req.Admin)
		}
	}

	// Construire l'update
	update := bson.M{}
	if req.Firstname != "" {
		update["firstname"] = req.Firstname
	}
	if req.Lastname != "" {
		update["lastname"] = req.Lastname
	}
	if req.Email != "" {
		update["email"] = strings.ToLower(strings.TrimSpace(req.Email))
	}
	if req.Phone != "" {
		update["phone"] = req.Phone
	}
	if req.Admin != nil {
		update["admin"] = *req.Admin
	}

	if len(update) == 0 {
		utils.RespondError(w, http.StatusBadRequest, "Aucune donnée à mettre à jour")
		return
	}

	// Mettre à jour l'utilisateur
	if err := h.userRepo.UpdateFields(userID, update); err != nil {
		log.Printf("Erreur lors de la mise à jour de l'utilisateur: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}

	// Récupérer l'utilisateur mis à jour
	updatedUser, err := h.userRepo.FindByID(userID)
	if err != nil || updatedUser == nil {
		log.Printf("Erreur lors de la récupération de l'utilisateur: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}

	// 🔌 Envoyer l'événement WebSocket si les droits admin ont changé
	if adminStatusChanged && h.wsHub != nil && req.Admin != nil {
		payload := map[string]interface{}{
			"type":       "admin_rights_changed",
			"user_id":    userID.Hex(),
			"user_email": updatedUser.Email,
			"admin":      *req.Admin,
		}
		// ⚠️ IMPORTANT : Utiliser l'EMAIL de l'utilisateur, pas l'ObjectID
		// Le WebSocket identifie les utilisateurs par leur email
		h.wsHub.SendToUser(updatedUser.Email, payload)
	}

	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"success":     true,
		"message":     "Utilisateur modifié avec succès",
		"utilisateur": updatedUser,
	})
}

// DeleteUser supprime un utilisateur
func (h *AdminHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.RespondError(w, http.StatusMethodNotAllowed, "Méthode non autorisée")
		return
	}

	// Récupérer l'ID depuis l'URL
	vars := mux.Vars(r)
	userID, err := primitive.ObjectIDFromHex(vars["id"])
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "ID utilisateur invalide")
		return
	}

	// Supprimer l'utilisateur
	if err := h.userRepo.Delete(userID); err != nil {
		log.Printf("Erreur lors de la suppression de l'utilisateur: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}

	log.Printf("✓ Utilisateur supprimé: ID %s", userID.Hex())
	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Utilisateur supprimé",
	})
}

// ========== GESTION DES ÉVÉNEMENTS ==========

// GetEvent retourne les détails d'un événement (admin)
func (h *AdminHandler) GetEvent(w http.ResponseWriter, r *http.Request) {
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

	// Récupérer l'événement
	event, err := h.eventRepo.FindByID(eventID)
	if err != nil || event == nil {
		utils.RespondError(w, http.StatusNotFound, "Événement non trouvé")
		return
	}

	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"evenement": event,
	})
}

// GetEvents retourne la liste de tous les événements
func (h *AdminHandler) GetEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.RespondError(w, http.StatusMethodNotAllowed, "Méthode non autorisée")
		return
	}

	events, err := h.eventRepo.FindAll()
	if err != nil {
		log.Printf("Erreur lors de la récupération des événements: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}

	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"evenements": events,
	})
}

// CreateEvent crée un nouvel événement
func (h *AdminHandler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.RespondError(w, http.StatusMethodNotAllowed, "Méthode non autorisée")
		return
	}

	var req models.CreateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Données invalides")
		return
	}

	// Valider les données
	if req.Titre == "" || req.CodeSoiree == "" {
		utils.RespondError(w, http.StatusBadRequest, "Titre et code_soiree sont requis")
		return
	}

	// Créer l'événement
	event := &models.Event{
		Titre:       req.Titre,
		Date:        req.Date.Time, // Convertir FlexibleTime en time.Time
		Description: req.Description,
		Capacite:    req.Capacite,
		Lieu:        req.Lieu,
		CodeSoiree:  req.CodeSoiree,
		Statut:      req.Statut,
	}

	// Ajouter les dates d'inscription si fournies
	if req.DateOuvertureInscription != nil && !req.DateOuvertureInscription.Time.IsZero() {
		t := req.DateOuvertureInscription.Time
		event.DateOuvertureInscription = &t
	}
	if req.DateFermetureInscription != nil && !req.DateFermetureInscription.Time.IsZero() {
		t := req.DateFermetureInscription.Time
		event.DateFermetureInscription = &t
	}

	if err := h.eventRepo.Create(event); err != nil {
		log.Printf("Erreur lors de la création de l'événement: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur lors de la création de l'événement")
		return
	}

	log.Printf("✓ Événement créé: %s (ID: %s)", event.Titre, event.ID.Hex())
	utils.RespondJSON(w, http.StatusCreated, map[string]interface{}{
		"success":   true,
		"message":   "Événement créé avec succès",
		"evenement": event,
	})
}

// UpdateEvent met à jour un événement
func (h *AdminHandler) UpdateEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.RespondError(w, http.StatusMethodNotAllowed, "Méthode non autorisée")
		return
	}

	// Récupérer l'ID depuis l'URL
	vars := mux.Vars(r)
	eventID, err := primitive.ObjectIDFromHex(vars["id"])
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "ID événement invalide")
		return
	}

	// Décoder la requête
	var req models.UpdateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Données invalides")
		return
	}

	// Construire l'update
	update := bson.M{}
	if req.Titre != "" {
		update["titre"] = req.Titre
	}
	if !req.Date.Time.IsZero() {
		update["date"] = req.Date.Time // Convertir FlexibleTime en time.Time
	}
	if req.Description != "" {
		update["description"] = req.Description
	}
	if req.Capacite > 0 {
		update["capacite"] = req.Capacite
	}
	if req.Lieu != "" {
		update["lieu"] = req.Lieu
	}
	if req.CodeSoiree != "" {
		update["code_soiree"] = req.CodeSoiree
	}
	if req.Statut != "" {
		update["statut"] = req.Statut
	}
	if req.DateOuvertureInscription != nil && !req.DateOuvertureInscription.Time.IsZero() {
		update["date_ouverture_inscription"] = req.DateOuvertureInscription.Time
	}
	if req.DateFermetureInscription != nil && !req.DateFermetureInscription.Time.IsZero() {
		update["date_fermeture_inscription"] = req.DateFermetureInscription.Time
	}

	if len(update) == 0 {
		utils.RespondError(w, http.StatusBadRequest, "Aucune donnée à mettre à jour")
		return
	}

	// Mettre à jour
	if err := h.eventRepo.Update(eventID, update); err != nil {
		log.Printf("Erreur lors de la mise à jour de l'événement: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}

	// Récupérer l'événement mis à jour
	updatedEvent, err := h.eventRepo.FindByID(eventID)
	if err != nil || updatedEvent == nil {
		log.Printf("Erreur lors de la récupération de l'événement: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}

	log.Printf("✓ Événement modifié: %s (ID: %s)", updatedEvent.Titre, eventID.Hex())
	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"success":   true,
		"message":   "Événement modifié",
		"evenement": updatedEvent,
	})
}

// DeleteEvent supprime un événement
func (h *AdminHandler) DeleteEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.RespondError(w, http.StatusMethodNotAllowed, "Méthode non autorisée")
		return
	}

	// Récupérer l'ID depuis l'URL
	vars := mux.Vars(r)
	eventID, err := primitive.ObjectIDFromHex(vars["id"])
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "ID événement invalide")
		return
	}

	// Supprimer l'événement
	if err := h.eventRepo.Delete(eventID); err != nil {
		log.Printf("Erreur lors de la suppression de l'événement: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}

	log.Printf("✓ Événement supprimé: ID %s", eventID.Hex())
	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Événement supprimé",
	})
}

// RecalculateEventCounters recalcule les compteurs d'un événement
func (h *AdminHandler) RecalculateEventCounters(w http.ResponseWriter, r *http.Request) {
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

	// Récupérer l'événement
	event, err := h.eventRepo.FindByID(eventID)
	if err != nil || event == nil {
		utils.RespondError(w, http.StatusNotFound, "Événement non trouvé")
		return
	}

	// Recalculer le nombre total de personnes inscrites
	totalPersonnes, err := h.inscriptionRepo.GetTotalPersonnesByEvent(eventID)
	if err != nil {
		log.Printf("Erreur calcul total personnes: %v", err)
		totalPersonnes = 0
	}

	// Recalculer le nombre de médias
	totalMedias, err := h.mediaRepo.CountByEvent(eventID)
	if err != nil {
		log.Printf("Erreur calcul total médias: %v", err)
		totalMedias = 0
	}

	// Mettre à jour l'événement
	err = h.eventRepo.Update(eventID, map[string]interface{}{
		"inscrits":     totalPersonnes,
		"photos_count": int(totalMedias),
	})

	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Erreur lors du recalcul")
		return
	}

	// Recharger l'événement
	event, _ = h.eventRepo.FindByID(eventID)

	log.Printf("✓ Compteurs recalculés pour %s: %d inscrits, %d médias", event.Titre, totalPersonnes, totalMedias)

	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"message":   "Compteurs recalculés",
		"evenement": event,
	})
}

// ========== STATISTIQUES ==========

// GetStats retourne les statistiques globales
func (h *AdminHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.RespondError(w, http.StatusMethodNotAllowed, "Méthode non autorisée")
		return
	}

	// Utiliser les méthodes du repository
	totalUsers, err := h.userRepo.CountAll()
	if err != nil {
		log.Printf("Erreur comptage users: %v", err)
		totalUsers = 0
	}

	totalAdmins, err := h.userRepo.CountAdmins()
	if err != nil {
		totalAdmins = 0
	}

	totalEvents, err := h.eventRepo.CountAll()
	if err != nil {
		totalEvents = 0
	}

	activeEvents, err := h.eventRepo.CountByStatus("ouvert")
	if err != nil {
		activeEvents = 0
	}

	totalInscrits, err := h.eventRepo.GetTotalInscrits()
	if err != nil {
		totalInscrits = 0
	}

	totalPhotos, err := h.eventRepo.GetTotalPhotos()
	if err != nil {
		totalPhotos = 0
	}

	stats := models.AdminStatsResponse{
		TotalUtilisateurs: int(totalUsers),
		TotalAdmins:       int(totalAdmins),
		TotalEvenements:   int(totalEvents),
		EvenementsActifs:  int(activeEvents),
		TotalInscrits:     totalInscrits,
		TotalPhotos:       totalPhotos,
	}

	utils.RespondJSON(w, http.StatusOK, stats)
}

// ========== NOTIFICATIONS ADMIN ==========

// SendAdminNotification envoie une notification depuis l'espace admin
func (h *AdminHandler) SendAdminNotification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.RespondError(w, http.StatusMethodNotAllowed, "Méthode non autorisée")
		return
	}

	var req struct {
		UserIDs []string          `json:"user_ids"`
		Title   string            `json:"title"`
		Message string            `json:"message"`
		Data    map[string]string `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Données invalides")
		return
	}

	// Récupérer les tokens
	var tokens []string

	if len(req.UserIDs) == 1 && req.UserIDs[0] == "all" {
		// Envoyer à tous
		allTokens, err := h.fcmTokenRepo.FindAll()
		if err != nil {
			log.Printf("Erreur récupération tokens: %v", err)
			utils.RespondError(w, http.StatusInternalServerError, "Erreur serveur")
			return
		}
		for _, t := range allTokens {
			tokens = append(tokens, t.Token)
		}
	} else {
		// Envoyer à des utilisateurs spécifiques
		for _, userID := range req.UserIDs {
			userTokens, err := h.fcmTokenRepo.FindByUserID(userID)
			if err != nil {
				continue
			}
			for _, t := range userTokens {
				tokens = append(tokens, t.Token)
			}
		}
	}

	if len(tokens) == 0 {
		utils.RespondError(w, http.StatusBadRequest, "Aucun token trouvé pour ces utilisateurs")
		return
	}

	// Envoyer les notifications
	title := req.Title
	if title == "" {
		title = "Nouvelle notification"
	}
	message := req.Message
	if message == "" {
		message = "Vous avez reçu une nouvelle notification"
	}

	success, failed, _ := h.fcmService.SendToAll(tokens, title, message, req.Data)

	log.Printf("📊 Admin notification: %d succès, %d échecs", success, failed)
	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Notification envoyée à %d utilisateurs", success),
	})
}

// ========== CODES SOIRÉE ==========

// generateRandomCode génère un code aléatoire
func generateRandomCode(length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
		time.Sleep(1 * time.Nanosecond)
	}
	return string(result)
}

// GenerateCodeSoiree génère un nouveau code de soirée aléatoire
func (h *AdminHandler) GenerateCodeSoiree(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.RespondError(w, http.StatusMethodNotAllowed, "Méthode non autorisée")
		return
	}

	// Générer un code unique (l'admin l'utilisera lors de la création d'un événement)
	code := generateRandomCode(10)

	log.Printf("✓ Code soirée généré: %s", code)

	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"code":       code,
		"created_at": time.Now(),
	})
}

// GetCurrentCodeSoiree retourne le code de soirée actuel
func (h *AdminHandler) GetCurrentCodeSoiree(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.RespondError(w, http.StatusMethodNotAllowed, "Méthode non autorisée")
		return
	}

	// Récupérer le code actuel
	code, err := h.codeSoireeRepo.FindCurrent()
	if err != nil {
		log.Printf("Erreur lors de la récupération du code actuel: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}

	if code == nil {
		utils.RespondError(w, http.StatusNotFound, "Aucun code de soirée actif")
		return
	}

	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"code":         code.Code,
		"created_at":   code.CreatedAt,
		"utilisations": code.Utilisations,
	})
}

// GetAllCodesSoiree retourne tous les codes de soirée (admin uniquement)
func (h *AdminHandler) GetAllCodesSoiree(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.RespondError(w, http.StatusMethodNotAllowed, "Méthode non autorisée")
		return
	}

	// Récupérer tous les codes
	codes, err := h.codeSoireeRepo.FindAll()
	if err != nil {
		log.Printf("Erreur lors de la récupération des codes de soirée: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}

	if codes == nil {
		codes = []models.CodeSoiree{}
	}

	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"codes": codes,
	})
}
