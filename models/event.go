package models

import (
	"encoding/json"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// EventTrailer représente une vidéo trailer d'événement
type EventTrailer struct {
	URL          string    `json:"url" bson:"url"`                               // URL de la vidéo sur Cloudinary
	PublicID     string    `json:"public_id" bson:"public_id"`                   // ID public Cloudinary (pour suppression)
	Duration     float64   `json:"duration,omitempty" bson:"duration,omitempty"` // Durée en secondes
	Format       string    `json:"format" bson:"format"`                         // Format vidéo (mp4, webm, etc.)
	Size         int64     `json:"size" bson:"size"`                             // Taille du fichier en bytes
	UploadedAt   time.Time `json:"uploaded_at" bson:"uploaded_at"`               // Date d'upload
	ThumbnailURL string    `json:"thumbnail_url" bson:"thumbnail_url"`           // URL de la miniature
}

// Event représente un événement dans le système
type Event struct {
	ID                       primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Titre                    string             `json:"titre" bson:"titre"`
	Date                     time.Time          `json:"date" bson:"date"` // Retour à time.Time standard
	Description              string             `json:"description" bson:"description"`
	Capacite                 int                `json:"capacite" bson:"capacite"`
	Inscrits                 int                `json:"inscrits" bson:"inscrits"`
	PhotosCount              int                `json:"photos_count" bson:"photos_count"`
	Statut                   string             `json:"statut" bson:"statut"` // "ouvert", "complet", "annule", "termine", "prochainement"
	Lieu                     string             `json:"lieu" bson:"lieu"`
	Adresse                  string             `json:"adresse" bson:"adresse"`
	CodePostal               string             `json:"code_postal" bson:"code_postal"`
	Ville                    string             `json:"ville" bson:"ville"`
	Pays                     string             `json:"pays" bson:"pays"`
	CodeSoiree               string             `json:"code_soiree" bson:"code_soiree"`
	DateOuvertureInscription *time.Time         `json:"date_ouverture_inscription,omitempty" bson:"date_ouverture_inscription,omitempty"` // Retour à *time.Time
	DateFermetureInscription *time.Time         `json:"date_fermeture_inscription,omitempty" bson:"date_fermeture_inscription,omitempty"` // Retour à *time.Time
	NotificationSentOpening  bool               `json:"notification_sent_opening" bson:"notification_sent_opening"`
	Trailer                  *EventTrailer      `json:"trailer,omitempty" bson:"trailer,omitempty"` // Vidéo trailer (optionnel)
	CreatedAt                time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt                time.Time          `json:"updated_at" bson:"updated_at"`
}

// CreateEventRequest représente la requête de création d'événement
type CreateEventRequest struct {
	Titre                    string        `json:"titre"`
	Date                     FlexibleTime  `json:"date"`
	Description              string        `json:"description"`
	Capacite                 int           `json:"capacite"`
	Lieu                     string        `json:"lieu"`
	Adresse                  string        `json:"adresse"`
	CodePostal               string        `json:"code_postal"`
	Ville                    string        `json:"ville"`
	Pays                     string        `json:"pays"`
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
	Adresse                  string        `json:"adresse,omitempty"`
	CodePostal               string        `json:"code_postal,omitempty"`
	Ville                    string        `json:"ville,omitempty"`
	Pays                     string        `json:"pays,omitempty"`
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
	TotalUtilisateurs int `json:"total_utilisateurs"`
	TotalAdmins       int `json:"total_admins"`
	TotalEvenements   int `json:"total_evenements"`
	EvenementsActifs  int `json:"evenements_actifs"`
	TotalInscrits     int `json:"total_inscrits"`
	TotalPhotos       int `json:"total_photos"`
}

// MarshalJSON implémente un marshaller JSON personnalisé pour Event
// pour formater les dates en heure française (Europe/Paris)
func (e Event) MarshalJSON() ([]byte, error) {
	// Charger la timezone de Paris
	paris, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		paris = time.FixedZone("CET", 2*3600)
	}

	// Créer un type alias pour éviter la récursion infinie
	type Alias Event

	// Convertir les dates en heure française
	dateStr := ""
	if !e.Date.IsZero() {
		dateStr = e.Date.In(paris).Format("2006-01-02T15:04:05")
	}

	dateOuvertureStr := (*string)(nil)
	if e.DateOuvertureInscription != nil && !e.DateOuvertureInscription.IsZero() {
		s := e.DateOuvertureInscription.In(paris).Format("2006-01-02T15:04:05")
		dateOuvertureStr = &s
	}

	dateFermetureStr := (*string)(nil)
	if e.DateFermetureInscription != nil && !e.DateFermetureInscription.IsZero() {
		s := e.DateFermetureInscription.In(paris).Format("2006-01-02T15:04:05")
		dateFermetureStr = &s
	}

	// Créer une structure temporaire avec les dates formatées
	return json.Marshal(&struct {
		*Alias
		Date                     string  `json:"date"`
		DateOuvertureInscription *string `json:"date_ouverture_inscription,omitempty"`
		DateFermetureInscription *string `json:"date_fermeture_inscription,omitempty"`
	}{
		Alias:                    (*Alias)(&e),
		Date:                     dateStr,
		DateOuvertureInscription: dateOuvertureStr,
		DateFermetureInscription: dateFermetureStr,
	})
}
