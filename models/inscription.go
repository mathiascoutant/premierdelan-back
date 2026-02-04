package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Accompagnant représente une personne accompagnant l'utilisateur principal
type Accompagnant struct {
	Firstname string `json:"firstname" bson:"firstname"`
	Lastname  string `json:"lastname" bson:"lastname"`
	IsAdult   bool   `json:"is_adult" bson:"is_adult"`
}

// Inscription représente l'inscription d'un utilisateur à un événement
type Inscription struct {
	ID              primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	EventID         primitive.ObjectID `json:"event_id" bson:"event_id"`
	UserEmail       string             `json:"user_email" bson:"user_email"`
	NombrePersonnes int                `json:"nombre_personnes" bson:"nombre_personnes"`
	Accompagnants   []Accompagnant     `json:"accompagnants" bson:"accompagnants"`
	CreatedAt       time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at" bson:"updated_at"`
}

// CreateInscriptionRequest représente la requête de création d'inscription
type CreateInscriptionRequest struct {
	UserEmail       string         `json:"user_email"`
	NombrePersonnes int            `json:"nombre_personnes"`
	Accompagnants   []Accompagnant `json:"accompagnants"`
}

// UpdateInscriptionRequest représente la requête de modification d'inscription
type UpdateInscriptionRequest struct {
	UserEmail       string         `json:"user_email"`
	NombrePersonnes int            `json:"nombre_personnes"`
	Accompagnants   []Accompagnant `json:"accompagnants"`
}

// DesinscriptionRequest représente la requête de désinscription
type DesinscriptionRequest struct {
	UserEmail string `json:"user_email"`
}

// InscriptionWithUserInfo contient les infos complètes pour l'admin
type InscriptionWithUserInfo struct {
	ID              string         `json:"id"`
	UserEmail       string         `json:"user_email"`
	UserName        string         `json:"user_name"`
	UserPhone       string         `json:"user_phone"`
	NombrePersonnes int            `json:"nombre_personnes"`
	Accompagnants   []Accompagnant `json:"accompagnants"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}
