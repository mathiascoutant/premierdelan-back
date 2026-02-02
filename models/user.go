package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User représente un utilisateur dans le système
type User struct {
	ID              primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	CodeSoiree      string             `json:"code_soiree" bson:"code_soiree,omitempty"`
	Firstname       string             `json:"firstname" bson:"firstname"`
	Lastname        string             `json:"lastname" bson:"lastname"`
	Email           string             `json:"email" bson:"email"`
	Phone           string             `json:"phone" bson:"phone"`
	Password        string             `json:"-" bson:"password"`                                            // Le "-" empêche la sérialisation du mot de passe
	ProfileImageURL string             `json:"profileImageUrl,omitempty" bson:"profile_image_url,omitempty"` // URL de la photo de profil
	FCMToken        string             `json:"fcm_token,omitempty" bson:"fcm_token,omitempty"`               // Token FCM pour les notifications
	Admin           int                `json:"admin" bson:"admin"`                                           // 0 = utilisateur normal, 1 = admin
	LastSeen        *time.Time         `json:"last_seen,omitempty" bson:"last_seen,omitempty"`               // Dernière activité WebSocket
	CreatedAt       time.Time          `json:"created_at" bson:"created_at"`
}

// RegisterRequest représente la requête d'inscription
type RegisterRequest struct {
	CodeSoiree string `json:"codesoiree"` // Frontend envoie "codesoiree" (sans underscore)
	Firstname  string `json:"prenom"`     // Frontend envoie "prenom" (français)
	Lastname   string `json:"nom"`        // Frontend envoie "nom" (français)
	Email      string `json:"email" validate:"required,email"`
	Phone      string `json:"telephone" validate:"required"` // Frontend envoie "telephone" (français)
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
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}
