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

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// InscriptionHandler gère les inscriptions aux événements
type InscriptionHandler struct {
	inscriptionRepo *database.InscriptionRepository
	eventRepo       *database.EventRepository
	userRepo        *database.UserRepository
	fcmService      interface {
		SendToAll(tokens []string, title, body string, data map[string]string) (success int, failed int, failedTokens []string)
	}
	fcmTokenRepo *database.FCMTokenRepository
}

// EventWithInscription représente un événement avec les détails de l'inscription de l'utilisateur
type EventWithInscription struct {
	models.Event
	UserInscription *InscriptionDetails `json:"user_inscription,omitempty"`
}

// InscriptionDetails représente les détails de l'inscription dans la réponse
type InscriptionDetails struct {
	ID              string `json:"id"`
	NombrePersonnes int    `json:"nombre_personnes"`
	CreatedAt       string `json:"created_at"`
}

// NewInscriptionHandler crée une nouvelle instance
func NewInscriptionHandler(db *mongo.Database, fcmService interface {
	SendToAll(tokens []string, title, body string, data map[string]string) (success int, failed int, failedTokens []string)
}) *InscriptionHandler {
	return &InscriptionHandler{
		inscriptionRepo: database.NewInscriptionRepository(db),
		eventRepo:       database.NewEventRepository(db),
		userRepo:        database.NewUserRepository(db),
		fcmService:      fcmService,
		fcmTokenRepo:    database.NewFCMTokenRepository(db),
	}
}

// CreateInscription gère la création d'une inscription
func (h *InscriptionHandler) CreateInscription(w http.ResponseWriter, r *http.Request) {
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

	// Décoder la requête
	var req models.CreateInscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Données invalides")
		return
	}

	// Validations
	if req.UserEmail == "" {
		utils.RespondError(w, http.StatusBadRequest, "Email utilisateur requis")
		return
	}

	if req.NombrePersonnes < 1 {
		utils.RespondError(w, http.StatusBadRequest, "Le nombre de personnes doit être au moins 1")
		return
	}

	// Vérifier la cohérence nombre_personnes et accompagnants
	if req.NombrePersonnes-1 != len(req.Accompagnants) {
		utils.RespondError(w, http.StatusBadRequest, "Le nombre d'accompagnants doit être égal à nombre_personnes - 1")
		return
	}

	// Valider les accompagnants
	for _, acc := range req.Accompagnants {
		if acc.Firstname == "" || acc.Lastname == "" {
			utils.RespondError(w, http.StatusBadRequest, "Tous les accompagnants doivent avoir un prénom et un nom")
			return
		}
	}

	// Récupérer l'événement
	event, err := h.eventRepo.FindByID(eventID)
	if err != nil || event == nil {
		utils.RespondError(w, http.StatusNotFound, "Événement non trouvé")
		return
	}

	// Vérifier que l'événement est ouvert
	if event.Statut != "ouvert" {
		utils.RespondJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error":  "Les inscriptions sont fermées pour cet événement",
			"statut": event.Statut,
		})
		return
	}

	// Vérifier que l'utilisateur n'est pas déjà inscrit
	existingInscription, err := h.inscriptionRepo.FindByEventAndUser(eventID, req.UserEmail)
	if err != nil {
		log.Printf("Erreur vérification inscription: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}

	if existingInscription != nil {
		utils.RespondError(w, http.StatusConflict, "Vous êtes déjà inscrit à cet événement")
		return
	}

	// Vérifier les places disponibles
	placesRestantes := event.Capacite - event.Inscrits
	if req.NombrePersonnes > placesRestantes {
		utils.RespondJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error":             "Plus assez de places disponibles",
			"places_restantes":  placesRestantes,
			"demande":           req.NombrePersonnes,
		})
		return
	}

	// Créer l'inscription
	inscription := &models.Inscription{
		EventID:         eventID,
		UserEmail:       req.UserEmail,
		NombrePersonnes: req.NombrePersonnes,
		Accompagnants:   req.Accompagnants,
	}

	if err := h.inscriptionRepo.Create(inscription); err != nil {
		log.Printf("Erreur création inscription: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur lors de la création de l'inscription")
		return
	}

	// Mettre à jour le compteur d'inscrits
	newInscrits := event.Inscrits + req.NombrePersonnes
	if err := h.eventRepo.Update(eventID, map[string]interface{}{
		"inscrits": newInscrits,
	}); err != nil {
		log.Printf("Erreur mise à jour compteur: %v", err)
	}

	// Recharger l'événement pour avoir les données à jour
	event, _ = h.eventRepo.FindByID(eventID)

	log.Printf("✓ Nouvelle inscription: %s à l'événement %s (%d personnes)", req.UserEmail, event.Titre, req.NombrePersonnes)

	// Notifier les admins
	go h.notifyAdminsNewInscription(req.UserEmail, event, req.NombrePersonnes)

	utils.RespondJSON(w, http.StatusCreated, map[string]interface{}{
		"message": "Inscription réussie",
		"inscription": inscription,
		"evenement": map[string]interface{}{
			"id":       event.ID.Hex(),
			"titre":    event.Titre,
			"inscrits": event.Inscrits,
		},
	})
}

// GetInscription récupère l'inscription d'un utilisateur
func (h *InscriptionHandler) GetInscription(w http.ResponseWriter, r *http.Request) {
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

	// Récupérer le user_email depuis les query params
	userEmail := r.URL.Query().Get("user_email")
	if userEmail == "" {
		utils.RespondError(w, http.StatusBadRequest, "Email utilisateur requis")
		return
	}

	// Chercher l'inscription
	inscription, err := h.inscriptionRepo.FindByEventAndUser(eventID, userEmail)
	if err != nil {
		log.Printf("Erreur recherche inscription: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}

	if inscription == nil {
		utils.RespondJSON(w, http.StatusNotFound, map[string]interface{}{
			"error":   "Aucune inscription trouvée",
			"inscrit": false,
		})
		return
	}

	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"inscription": inscription,
	})
}

// UpdateInscription modifie une inscription existante
func (h *InscriptionHandler) UpdateInscription(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
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

	// Décoder la requête
	var req models.UpdateInscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Données invalides")
		return
	}

	// Validations
	if req.UserEmail == "" {
		utils.RespondError(w, http.StatusBadRequest, "Email utilisateur requis")
		return
	}

	if req.NombrePersonnes < 1 {
		utils.RespondError(w, http.StatusBadRequest, "Le nombre de personnes doit être au moins 1")
		return
	}

	// Vérifier la cohérence
	if req.NombrePersonnes-1 != len(req.Accompagnants) {
		utils.RespondError(w, http.StatusBadRequest, "Le nombre d'accompagnants doit être égal à nombre_personnes - 1")
		return
	}

	// Valider les accompagnants
	for _, acc := range req.Accompagnants {
		if acc.Firstname == "" || acc.Lastname == "" {
			utils.RespondError(w, http.StatusBadRequest, "Tous les accompagnants doivent avoir un prénom et un nom")
			return
		}
	}

	// Récupérer l'inscription existante
	inscription, err := h.inscriptionRepo.FindByEventAndUser(eventID, req.UserEmail)
	if err != nil {
		log.Printf("Erreur recherche inscription: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}

	if inscription == nil {
		utils.RespondError(w, http.StatusNotFound, "Aucune inscription trouvée à modifier")
		return
	}

	// Récupérer l'événement
	event, err := h.eventRepo.FindByID(eventID)
	if err != nil || event == nil {
		utils.RespondError(w, http.StatusNotFound, "Événement non trouvé")
		return
	}

	// Vérifier que l'événement est toujours ouvert
	if event.Statut != "ouvert" {
		utils.RespondJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error":  "Les modifications sont fermées pour cet événement",
			"statut": event.Statut,
		})
		return
	}

	// Calculer la différence de personnes
	ancienNombre := inscription.NombrePersonnes
	nouveauNombre := req.NombrePersonnes
	difference := nouveauNombre - ancienNombre

	// Si augmentation, vérifier les places disponibles
	if difference > 0 {
		placesRestantes := event.Capacite - event.Inscrits
		if difference > placesRestantes {
			utils.RespondJSON(w, http.StatusBadRequest, map[string]interface{}{
				"error":                  "Plus assez de places pour cette modification",
				"places_restantes":       placesRestantes,
				"augmentation_demandee": difference,
			})
			return
		}
	}

	// Mettre à jour l'inscription
	inscription.NombrePersonnes = req.NombrePersonnes
	inscription.Accompagnants = req.Accompagnants

	if err := h.inscriptionRepo.Update(inscription); err != nil {
		log.Printf("Erreur mise à jour inscription: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur lors de la modification")
		return
	}

	// Mettre à jour le compteur d'inscrits
	newInscrits := event.Inscrits + difference
	if err := h.eventRepo.Update(eventID, map[string]interface{}{
		"inscrits": newInscrits,
	}); err != nil {
		log.Printf("Erreur mise à jour compteur: %v", err)
	}

	// Recharger l'événement
	event, _ = h.eventRepo.FindByID(eventID)

	log.Printf("✓ Inscription modifiée: %s (diff: %+d personnes)", req.UserEmail, difference)

	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"message":     "Inscription modifiée",
		"inscription": inscription,
		"evenement": map[string]interface{}{
			"id":       event.ID.Hex(),
			"titre":    event.Titre,
			"inscrits": event.Inscrits,
		},
	})
}

// DeleteInscription supprime une inscription
func (h *InscriptionHandler) DeleteInscription(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
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

	// Décoder la requête
	var req models.DesinscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Données invalides")
		return
	}

	if req.UserEmail == "" {
		utils.RespondError(w, http.StatusBadRequest, "Email utilisateur requis")
		return
	}

	// Récupérer l'inscription
	inscription, err := h.inscriptionRepo.FindByEventAndUser(eventID, req.UserEmail)
	if err != nil {
		log.Printf("Erreur recherche inscription: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}

	if inscription == nil {
		utils.RespondError(w, http.StatusNotFound, "Aucune inscription à supprimer")
		return
	}

	nombrePersonnes := inscription.NombrePersonnes

	// Supprimer l'inscription
	if err := h.inscriptionRepo.Delete(eventID, req.UserEmail); err != nil {
		log.Printf("Erreur suppression inscription: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur lors de la désinscription")
		return
	}

	// Mettre à jour le compteur d'inscrits
	event, _ := h.eventRepo.FindByID(eventID)
	if event != nil {
		newInscrits := event.Inscrits - nombrePersonnes
		if newInscrits < 0 {
			newInscrits = 0
		}
		h.eventRepo.Update(eventID, map[string]interface{}{
			"inscrits": newInscrits,
		})

		// Recharger l'événement
		event, _ = h.eventRepo.FindByID(eventID)
	}

	log.Printf("✓ Désinscription: %s (%d personnes libérées)", req.UserEmail, nombrePersonnes)

	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"message":                  "Désinscription réussie",
		"nombre_personnes_liberes": nombrePersonnes,
		"evenement": map[string]interface{}{
			"id":       event.ID.Hex(),
			"titre":    event.Titre,
			"inscrits": event.Inscrits,
		},
	})
}

// GetInscrits retourne la liste des inscrits (admin uniquement)
func (h *InscriptionHandler) GetInscrits(w http.ResponseWriter, r *http.Request) {
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

	// Récupérer toutes les inscriptions
	inscriptions, err := h.inscriptionRepo.FindByEvent(eventID)
	if err != nil {
		log.Printf("Erreur récupération inscriptions: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}

	// Enrichir avec les infos utilisateur
	var inscriptionsWithInfo []models.InscriptionWithUserInfo
	totalPersonnes := 0
	totalAdultes := 0
	totalMineurs := 0

	for _, insc := range inscriptions {
		// Récupérer l'utilisateur
		user, err := h.userRepo.FindByEmail(insc.UserEmail)
		userName := ""
		userPhone := ""
		if err == nil && user != nil {
			userName = fmt.Sprintf("%s %s", user.Firstname, user.Lastname)
			userPhone = user.Phone
		}

		// Compter adultes et mineurs
		adultes := 1 // L'utilisateur principal est toujours adulte
		mineurs := 0
		for _, acc := range insc.Accompagnants {
			if acc.IsAdult {
				adultes++
			} else {
				mineurs++
			}
		}

		totalPersonnes += insc.NombrePersonnes
		totalAdultes += adultes
		totalMineurs += mineurs

		inscriptionsWithInfo = append(inscriptionsWithInfo, models.InscriptionWithUserInfo{
			ID:              insc.ID.Hex(),
			UserEmail:       insc.UserEmail,
			UserName:        userName,
			UserPhone:       userPhone,
			NombrePersonnes: insc.NombrePersonnes,
			Accompagnants:   insc.Accompagnants,
			CreatedAt:       insc.CreatedAt,
			UpdatedAt:       insc.UpdatedAt,
		})
	}

	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"event_id":       event.ID.Hex(),
		"titre":          event.Titre,
		"total_inscrits": len(inscriptions),
		"total_personnes": totalPersonnes,
		"total_adultes":  totalAdultes,
		"total_mineurs":  totalMineurs,
		"inscriptions":   inscriptionsWithInfo,
	})
}

// Helper pour vérifier si l'utilisateur est authentifié
func getUserEmailFromContext(r *http.Request) string {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		return ""
	}
	return claims.Email
}

// DeleteInscriptionAdmin permet à un admin de supprimer n'importe quelle inscription
func (h *InscriptionHandler) DeleteInscriptionAdmin(w http.ResponseWriter, r *http.Request) {
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

	inscriptionID, err := primitive.ObjectIDFromHex(vars["inscription_id"])
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "ID inscription invalide")
		return
	}

	// Récupérer l'inscription
	inscription, err := h.inscriptionRepo.FindByID(inscriptionID)
	if err != nil {
		log.Printf("Erreur recherche inscription: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}

	if inscription == nil {
		utils.RespondError(w, http.StatusNotFound, "Inscription non trouvée")
		return
	}

	// Vérifier que l'inscription appartient bien à cet événement
	if inscription.EventID != eventID {
		utils.RespondError(w, http.StatusBadRequest, "Cette inscription n'appartient pas à cet événement")
		return
	}

	nombrePersonnes := inscription.NombrePersonnes

	// Supprimer l'inscription
	if err := h.inscriptionRepo.DeleteByID(inscriptionID); err != nil {
		log.Printf("Erreur suppression inscription: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur lors de la suppression")
		return
	}

	// Mettre à jour le compteur d'inscrits
	event, _ := h.eventRepo.FindByID(eventID)
	if event != nil {
		newInscrits := event.Inscrits - nombrePersonnes
		if newInscrits < 0 {
			newInscrits = 0
		}
		h.eventRepo.Update(eventID, map[string]interface{}{
			"inscrits": newInscrits,
		})

		// Recharger l'événement
		event, _ = h.eventRepo.FindByID(eventID)
	}

	log.Printf("✓ Inscription supprimée par admin: %s (%d personnes)", inscription.UserEmail, nombrePersonnes)

	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"message":                  "Inscription supprimée avec succès",
		"inscription_id":           inscriptionID.Hex(),
		"nombre_personnes_liberes": nombrePersonnes,
		"evenement": map[string]interface{}{
			"id":       event.ID.Hex(),
			"titre":    event.Titre,
			"inscrits": event.Inscrits,
		},
	})
}

// DeleteAccompagnant supprime un accompagnant spécifique (admin uniquement)
func (h *InscriptionHandler) DeleteAccompagnant(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.RespondError(w, http.StatusMethodNotAllowed, "Méthode non autorisée")
		return
	}

	// Récupérer les paramètres depuis l'URL
	vars := mux.Vars(r)
	eventID, err := primitive.ObjectIDFromHex(vars["event_id"])
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "ID événement invalide")
		return
	}

	inscriptionID, err := primitive.ObjectIDFromHex(vars["inscription_id"])
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "ID inscription invalide")
		return
	}

	indexStr := vars["index"]
	index := 0
	_, err = fmt.Sscanf(indexStr, "%d", &index)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Index invalide")
		return
	}

	// Récupérer l'inscription
	inscription, err := h.inscriptionRepo.FindByID(inscriptionID)
	if err != nil {
		log.Printf("Erreur recherche inscription: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}

	if inscription == nil {
		utils.RespondError(w, http.StatusNotFound, "Inscription non trouvée")
		return
	}

	// Vérifier que l'inscription appartient à cet événement
	if inscription.EventID != eventID {
		utils.RespondError(w, http.StatusBadRequest, "Cette inscription n'appartient pas à cet événement")
		return
	}

	// Vérifier que l'index est valide
	if index < 0 || index >= len(inscription.Accompagnants) {
		utils.RespondError(w, http.StatusBadRequest, "Index d'accompagnant invalide")
		return
	}

	// Récupérer le nom de l'accompagnant avant suppression
	accompagnantName := fmt.Sprintf("%s %s", inscription.Accompagnants[index].Firstname, inscription.Accompagnants[index].Lastname)

	// Retirer l'accompagnant du tableau
	inscription.Accompagnants = append(
		inscription.Accompagnants[:index],
		inscription.Accompagnants[index+1:]...,
	)

	// Décrémenter le nombre de personnes
	inscription.NombrePersonnes--

	// Mettre à jour l'inscription
	if err := h.inscriptionRepo.Update(inscription); err != nil {
		log.Printf("Erreur mise à jour inscription: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur lors de la suppression")
		return
	}

	// Mettre à jour le compteur de l'événement
	event, _ := h.eventRepo.FindByID(eventID)
	if event != nil {
		newInscrits := event.Inscrits - 1
		if newInscrits < 0 {
			newInscrits = 0
		}
		h.eventRepo.Update(eventID, map[string]interface{}{
			"inscrits": newInscrits,
		})

		// Recharger l'événement
		event, _ = h.eventRepo.FindByID(eventID)
	}

	log.Printf("✓ Accompagnant supprimé par admin: %s de l'inscription %s", accompagnantName, inscription.UserEmail)

	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"message":     "Accompagnant supprimé avec succès",
		"inscription": inscription,
		"evenement": map[string]interface{}{
			"id":       event.ID.Hex(),
			"inscrits": event.Inscrits,
		},
	})
}

// notifyAdminsNewInscription envoie une notification aux admins lors d'une nouvelle inscription
func (h *InscriptionHandler) notifyAdminsNewInscription(userEmail string, event *models.Event, nombrePersonnes int) {
	if h.fcmService == nil {
		return
	}

	// Récupérer l'utilisateur qui s'inscrit
	user, err := h.userRepo.FindByEmail(userEmail)
	if err != nil || user == nil {
		log.Printf("⚠️  Impossible de récupérer l'utilisateur %s", userEmail)
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
	title := "🎉 Nouvelle inscription à un événement !"
	message := fmt.Sprintf("%s %s s'est inscrit à %s (%d/%d personnes)", 
		user.Firstname, 
		user.Lastname, 
		event.Titre,
		event.Inscrits,
		event.Capacite,
	)
	
	data := map[string]string{
		"type":             "new_inscription",
		"event_id":         event.ID.Hex(),
		"event_titre":      event.Titre,
		"user_email":       userEmail,
		"user_firstname":   user.Firstname,
		"user_lastname":    user.Lastname,
		"nombre_personnes": fmt.Sprintf("%d", nombrePersonnes),
		"inscrits":         fmt.Sprintf("%d", event.Inscrits),
		"capacite":         fmt.Sprintf("%d", event.Capacite),
	}

	// Envoyer aux admins
	success, failed, _ := h.fcmService.SendToAll(adminTokens, title, message, data)
	log.Printf("📧 Notification inscription envoyée aux admins: %d succès, %d échecs", success, failed)
}

// GetMesEvenements retourne la liste des événements auxquels l'utilisateur est inscrit
func (h *InscriptionHandler) GetMesEvenements(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.RespondError(w, http.StatusMethodNotAllowed, "Méthode non autorisée")
		return
	}

	// Récupérer l'utilisateur depuis le contexte (mis par le middleware Auth)
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Non authentifié")
		return
	}

	userEmail := claims.Email

	// Récupérer toutes les inscriptions de cet utilisateur
	inscriptions, err := h.inscriptionRepo.FindByUser(userEmail)
	if err != nil {
		log.Printf("Erreur lors de la récupération des inscriptions: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}

	// Construire la liste des événements avec les détails d'inscription
	var evenements []EventWithInscription

	for _, inscription := range inscriptions {
		// Récupérer l'événement
		event, err := h.eventRepo.FindByID(inscription.EventID)
		if err != nil {
			log.Printf("Erreur récupération événement %s: %v", inscription.EventID.Hex(), err)
			continue // Passer à l'inscription suivante
		}

		if event == nil {
			continue // Événement supprimé
		}

		// Construire la réponse avec les détails de l'inscription
		eventWithInscription := EventWithInscription{
			Event: *event,
			UserInscription: &InscriptionDetails{
				ID:              inscription.ID.Hex(),
				NombrePersonnes: inscription.NombrePersonnes,
				CreatedAt:       inscription.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			},
		}

		evenements = append(evenements, eventWithInscription)
	}

	// Si aucune inscription, retourner un tableau vide
	if evenements == nil {
		evenements = []EventWithInscription{}
	}

	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"evenements": evenements,
	})
}

