package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User représente un utilisateur dans le système
type User struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	CodeSoiree string            `json:"code_soiree" bson:"code_soiree,omitempty"`
	Firstname string             `json:"firstname" bson:"firstname"`
	Lastname  string             `json:"lastname" bson:"lastname"`
	Email     string             `json:"email" bson:"email"`
	Phone     string             `json:"phone" bson:"phone"`
	Password  string             `json:"-" bson:"password"` // Le "-" empêche la sérialisation du mot de passe
	Admin     int                `json:"admin" bson:"admin"`   // 0 = utilisateur normal, 1 = admin
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
}

// RegisterRequest représente la requête d'inscription
type RegisterRequest struct {
	CodeSoiree string `json:"code_soiree"`
	Firstname  string `json:"firstname"` // Le frontend envoie "firstname"
	Lastname   string `json:"lastname"`  // Le frontend envoie "lastname"
	Email      string `json:"email" validate:"required,email"`
	Phone      string `json:"phone" validate:"required"` // Le frontend envoie "phone"
	Password   string `json:"password" validate:"required,min=6"`
}

// LoginRequest représente la requête de connexion
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// AuthResponse représente la réponse d'authentification
type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// ErrorResponse représente une réponse d'erreur
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// SuccessResponse représente une réponse de succès générique
type SuccessResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

