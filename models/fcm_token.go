package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// FCMToken représente un token FCM pour les notifications
type FCMToken struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID    string             `json:"user_id" bson:"user_id"` // Email de l'utilisateur
	Token     string             `json:"token" bson:"token"`
	Device    string             `json:"device,omitempty" bson:"device,omitempty"` // Type d'appareil (iOS, Android, Web)
	UserAgent string             `json:"user_agent,omitempty" bson:"user_agent,omitempty"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time          `json:"updated_at" bson:"updated_at"`
}

// FCMSubscribeRequest représente la requête d'abonnement FCM
type FCMSubscribeRequest struct {
	UserID    string `json:"user_id"`
	FCMToken  string `json:"fcm_token"`
	Device    string `json:"device,omitempty"`
	UserAgent string `json:"user_agent,omitempty"`
}

// FCMNotificationRequest représente la requête pour envoyer une notification FCM
type FCMNotificationRequest struct {
	UserID  string            `json:"user_id"`
	Title   string            `json:"title,omitempty"`
	Message string            `json:"message,omitempty"`
	Data    map[string]string `json:"data,omitempty"`
}

// FCMNotificationResponse représente la réponse d'envoi FCM
type FCMNotificationResponse struct {
	Success      int      `json:"success"`
	Failed       int      `json:"failed"`
	Total        int      `json:"total"`
	FailedTokens []string `json:"failed_tokens,omitempty"`
}
