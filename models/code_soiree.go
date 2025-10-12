package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CodeSoiree représente un code de soirée
type CodeSoiree struct {
	ID           primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Code         string             `json:"code" bson:"code"`
	Utilisations int                `json:"utilisations" bson:"utilisations"`
	CreatedAt    time.Time          `json:"created_at" bson:"created_at"`
	Active       bool               `json:"active" bson:"active"`
}

