package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CriticalAlert représente une alerte critique envoyée par le frontend
type CriticalAlert struct {
	ID               primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	AdminEmail       string             `json:"admin_email" bson:"admin_email"`
	ErrorType        string             `json:"error_type" bson:"error_type"` // "SERVER_ERROR", "NETWORK_ERROR", "CONNECTION_ERROR"
	ErrorMessage     string             `json:"error_message" bson:"error_message"`
	EndpointFailed   string             `json:"endpoint_failed" bson:"endpoint_failed"`
	Timestamp        time.Time          `json:"timestamp" bson:"timestamp"`
	UserAgent        string             `json:"user_agent" bson:"user_agent"`
	NotificationSent bool               `json:"notification_sent" bson:"notification_sent"`
	CreatedAt        time.Time          `json:"created_at" bson:"created_at"`
}

// CriticalAlertRequest représente la requête d'alerte critique
type CriticalAlertRequest struct {
	AdminEmail     string `json:"admin_email"`
	ErrorType      string `json:"error_type"`
	ErrorMessage   string `json:"error_message"`
	EndpointFailed string `json:"endpoint_failed"`
	Timestamp      string `json:"timestamp"`
	UserAgent      string `json:"user_agent"`
}
