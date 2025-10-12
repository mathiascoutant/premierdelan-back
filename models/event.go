package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Event représente un événement dans le système
type Event struct {
	ID                        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Titre                     string             `json:"titre" bson:"titre"`
	Date                      FlexibleTime       `json:"date" bson:"date"`
	Description               string             `json:"description" bson:"description"`
	Capacite                  int                `json:"capacite" bson:"capacite"`
	Inscrits                  int                `json:"inscrits" bson:"inscrits"`
	PhotosCount               int                `json:"photos_count" bson:"photos_count"`
	Statut                    string             `json:"statut" bson:"statut"` // "ouvert", "complet", "annule", "termine", "prochainement"
	Lieu                      string             `json:"lieu" bson:"lieu"`
	CodeSoiree                string             `json:"code_soiree" bson:"code_soiree"`
	DateOuvertureInscription  *FlexibleTime      `json:"date_ouverture_inscription,omitempty" bson:"date_ouverture_inscription,omitempty"`
	DateFermetureInscription  *FlexibleTime      `json:"date_fermeture_inscription,omitempty" bson:"date_fermeture_inscription,omitempty"`
	NotificationSentOpening   bool               `json:"notification_sent_opening" bson:"notification_sent_opening"`
	CreatedAt                 time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt                 time.Time          `json:"updated_at" bson:"updated_at"`
}

// CreateEventRequest représente la requête de création d'événement
type CreateEventRequest struct {
	Titre                    string        `json:"titre"`
	Date                     FlexibleTime  `json:"date"`
	Description              string        `json:"description"`
	Capacite                 int           `json:"capacite"`
	Lieu                     string        `json:"lieu"`
	CodeSoiree               string        `json:"code_soiree"`
	Statut                   string        `json:"statut"`
	DateOuvertureInscription *FlexibleTime `json:"date_ouverture_inscription,omitempty"`
	DateFermetureInscription *FlexibleTime `json:"date_fermeture_inscription,omitempty"`
}

// UpdateEventRequest représente la requête de modification d'événement
type UpdateEventRequest struct {
	Titre                    string        `json:"titre,omitempty"`
	Date                     FlexibleTime  `json:"date,omitempty"`
	Description              string        `json:"description,omitempty"`
	Capacite                 int           `json:"capacite,omitempty"`
	Lieu                     string        `json:"lieu,omitempty"`
	CodeSoiree               string        `json:"code_soiree,omitempty"`
	Statut                   string        `json:"statut,omitempty"`
	DateOuvertureInscription *FlexibleTime `json:"date_ouverture_inscription,omitempty"`
	DateFermetureInscription *FlexibleTime `json:"date_fermeture_inscription,omitempty"`
}

// UpdateUserRequest représente la requête de modification d'utilisateur
type UpdateUserRequest struct {
	Firstname string `json:"firstname,omitempty"`
	Lastname  string `json:"lastname,omitempty"`
	Email     string `json:"email,omitempty"`
	Phone     string `json:"phone,omitempty"`
	Admin     *int   `json:"admin,omitempty"` // Pointeur pour distinguer 0 de non-fourni
}

// AdminStatsResponse représente les statistiques admin
type AdminStatsResponse struct {
	TotalUtilisateurs  int `json:"total_utilisateurs"`
	TotalAdmins        int `json:"total_admins"`
	TotalEvenements    int `json:"total_evenements"`
	EvenementsActifs   int `json:"evenements_actifs"`
	TotalInscrits      int `json:"total_inscrits"`
	TotalPhotos        int `json:"total_photos"`
}

