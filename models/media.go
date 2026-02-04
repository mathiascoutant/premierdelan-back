package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Media représente un média (photo ou vidéo) uploadé pour un événement
type Media struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	EventID     primitive.ObjectID `json:"event_id" bson:"event_id"`
	UserEmail   string             `json:"user_email" bson:"user_email"`
	UserName    string             `json:"user_name" bson:"user_name"`
	Type        string             `json:"type" bson:"type"` // "image" ou "video"
	URL         string             `json:"url" bson:"url"`
	StoragePath string             `json:"storage_path" bson:"storage_path"`
	Filename    string             `json:"filename" bson:"filename"`
	Size        int64              `json:"size" bson:"size"`
	UploadedAt  time.Time          `json:"uploaded_at" bson:"uploaded_at"`
}

// CreateMediaRequest représente la requête d'ajout d'un média
type CreateMediaRequest struct {
	UserEmail   string `json:"user_email"`
	Type        string `json:"type"`
	URL         string `json:"url"`
	StoragePath string `json:"storage_path"`
	Filename    string `json:"filename"`
	Size        int64  `json:"size"`
}

// MediasResponse représente la réponse de liste des médias
type MediasResponse struct {
	EventID     string  `json:"event_id"`
	TotalMedias int     `json:"total_medias"`
	TotalImages int     `json:"total_images"`
	TotalVideos int     `json:"total_videos"`
	Medias      []Media `json:"medias"`
}
