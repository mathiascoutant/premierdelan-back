package handlers

import (
	"log"
	"net/http"
	"premier-an-backend/constants"
	"premier-an-backend/database"
	"premier-an-backend/models"
	"premier-an-backend/utils"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// EventHandler gère les requêtes publiques pour les événements
type EventHandler struct {
	eventRepo *database.EventRepository
}

// NewEventHandler crée une nouvelle instance de EventHandler
func NewEventHandler(db *mongo.Database) *EventHandler {
	return &EventHandler{
		eventRepo: database.NewEventRepository(db),
	}
}

// GetPublicEvents retourne la liste publique des événements
func (h *EventHandler) GetPublicEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.RespondError(w, http.StatusMethodNotAllowed, constants.ErrMethodNotAllowed)
		return
	}

	// Récupérer tous les événements
	events, err := h.eventRepo.FindAll()
	if err != nil {
		log.Printf("Erreur lors de la récupération des événements publics: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, constants.ErrServerError)
		return
	}

	// Si aucun événement, retourner un tableau vide
	if events == nil {
		events = []models.Event{}
	}

	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"evenements": events,
	})
}

// GetPublicEvent retourne les détails d'un événement spécifique (PUBLIC)
func (h *EventHandler) GetPublicEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.RespondError(w, http.StatusMethodNotAllowed, constants.ErrMethodNotAllowed)
		return
	}

	// Récupérer l'event_id depuis l'URL
	vars := mux.Vars(r)
	eventID, err := primitive.ObjectIDFromHex(vars["event_id"])
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, constants.ErrInvalidEventID)
		return
	}

	// Récupérer l'événement
	event, err := h.eventRepo.FindByID(eventID)
	if err != nil {
		log.Printf("Erreur lors de la récupération de l'événement: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, constants.ErrServerError)
		return
	}

	if event == nil {
		utils.RespondError(w, http.StatusNotFound, constants.ErrEventNotFound)
		return
	}

	// Réponse conforme à la spécification (pas de wrapper "data")
	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"success":   true,
		"evenement": event,
	})
}
