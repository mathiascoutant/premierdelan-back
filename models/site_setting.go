package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SiteSetting représente un paramètre global du site
type SiteSetting struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Key       string             `json:"key" bson:"key"`
	Value     string             `json:"value" bson:"value"`
	UpdatedAt time.Time          `json:"updated_at" bson:"updated_at"`
	UpdatedBy *primitive.ObjectID `json:"updated_by,omitempty" bson:"updated_by,omitempty"`
}

// ThemeRequest représente la requête de modification du thème
type ThemeRequest struct {
	Theme string `json:"theme" validate:"required,oneof=medieval classic"`
}

// ThemeResponse représente la réponse du thème
type ThemeResponse struct {
	Success bool   `json:"success"`
	Theme   string `json:"theme"`
	Message string `json:"message,omitempty"`
}
