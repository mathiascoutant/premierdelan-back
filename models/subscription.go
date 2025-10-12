package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// PushSubscription représente un abonnement aux notifications push
type PushSubscription struct {
	ID       primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID   string             `json:"user_id" bson:"user_id"` // Email de l'utilisateur
	Endpoint string             `json:"endpoint" bson:"endpoint"`
	Keys     PushKeys           `json:"keys" bson:"keys"`
	Created  time.Time          `json:"created_at" bson:"created_at"`
}

// PushKeys contient les clés de chiffrement pour les notifications
type PushKeys struct {
	P256dh string `json:"p256dh" bson:"p256dh"`
	Auth   string `json:"auth" bson:"auth"`
}

// SubscribeRequest représente la requête d'abonnement aux notifications
type SubscribeRequest struct {
	UserID       string   `json:"user_id"`
	Subscription struct {
		Endpoint string   `json:"endpoint"`
		Keys     PushKeys `json:"keys"`
	} `json:"subscription"`
}

// NotificationRequest représente la requête pour envoyer une notification
type NotificationRequest struct {
	UserID  string `json:"user_id"`
	Title   string `json:"title,omitempty"`
	Message string `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// NotificationPayload représente le contenu d'une notification
type NotificationPayload struct {
	Title   string      `json:"title"`
	Body    string      `json:"body"`
	Icon    string      `json:"icon,omitempty"`
	Badge   string      `json:"badge,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Actions []Action    `json:"actions,omitempty"`
}

// Action représente une action dans une notification
type Action struct {
	Action string `json:"action"`
	Title  string `json:"title"`
}

