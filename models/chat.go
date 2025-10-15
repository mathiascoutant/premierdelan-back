package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Participant représente un participant dans une conversation
type Participant struct {
	UserID primitive.ObjectID `json:"user_id" bson:"user_id"`
	Role   string             `json:"role" bson:"role"` // "admin"
	Status string             `json:"status" bson:"status"` // "active", "left"
}

// Conversation représente une conversation entre admins
type Conversation struct {
	ID            primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Participants  []Participant      `json:"participants" bson:"participants"`
	Status        string             `json:"status" bson:"status"` // "pending", "accepted", "rejected", "active"
	CreatedBy     primitive.ObjectID `json:"created_by" bson:"created_by"`
	CreatedAt     time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at" bson:"updated_at"`
	LastMessageAt *time.Time         `json:"last_message_at,omitempty" bson:"last_message_at,omitempty"`
}

// Message représente un message dans une conversation
type Message struct {
	ID             primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	ConversationID primitive.ObjectID `json:"conversation_id" bson:"conversation_id"`
	SenderID       primitive.ObjectID `json:"sender_id" bson:"sender_id"`
	Content        string             `json:"content" bson:"content"`
	Type           string             `json:"type" bson:"type"` // "text", "image", "file"
	IsRead         bool               `json:"is_read" bson:"is_read"`
	ReadBy         []ReadReceipt      `json:"read_by" bson:"read_by"`
	CreatedAt      time.Time          `json:"created_at" bson:"created_at"`
}

// ReadReceipt représente une confirmation de lecture
type ReadReceipt struct {
	UserID  primitive.ObjectID `json:"user_id" bson:"user_id"`
	ReadAt  time.Time          `json:"read_at" bson:"read_at"`
}

// ChatInvitation représente une invitation de chat
type ChatInvitation struct {
	ID          primitive.ObjectID  `json:"id" bson:"_id,omitempty"`
	FromUserID  primitive.ObjectID  `json:"from_user_id" bson:"from_user_id"`
	ToUserID    primitive.ObjectID  `json:"to_user_id" bson:"to_user_id"`
	Status      string              `json:"status" bson:"status"` // "pending", "accepted", "rejected"
	Message     string              `json:"message" bson:"message"`
	CreatedAt   time.Time           `json:"created_at" bson:"created_at"`
	RespondedAt *time.Time          `json:"responded_at,omitempty" bson:"responded_at,omitempty"`
}

// ConversationResponse représente la réponse pour la liste des conversations
type ConversationResponse struct {
	ID           string                 `json:"id"`
	Participant  UserInfo               `json:"participant"`
	LastMessage  *MessageInfo           `json:"last_message,omitempty"`
	Status       string                 `json:"status"`
	UnreadCount  int                    `json:"unread_count"`
}

// UserInfo représente les informations d'un utilisateur
type UserInfo struct {
	ID        string `json:"id"`
	Firstname string `json:"firstname"`
	Lastname  string `json:"lastname"`
	Email     string `json:"email"`
}

// MessageInfo représente les informations d'un message
type MessageInfo struct {
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	IsRead    bool      `json:"is_read"`
}

// MessageRequest représente la requête d'envoi de message
type MessageRequest struct {
	Content string `json:"content" validate:"required"`
}

// InvitationRequest représente la requête d'invitation
type InvitationRequest struct {
	ToUserID string `json:"toUserId" validate:"required"`
	Message  string `json:"message" validate:"required"`
}

// InvitationResponse représente la réponse d'invitation
type InvitationResponse struct {
	Action string `json:"action" validate:"required,oneof=accept reject"`
}

// ChatNotificationRequest représente la requête de notification
type ChatNotificationRequest struct {
	ToUserID string                 `json:"to_user_id" validate:"required"`
	Type     string                 `json:"type" validate:"required,oneof=chat_invitation chat_message"`
	Title    string                 `json:"title" validate:"required"`
	Body     string                 `json:"body" validate:"required"`
	Data     map[string]interface{} `json:"data"`
}

// AdminSearchResponse représente la réponse de recherche d'admins
type AdminSearchResponse struct {
	Admins []UserInfo `json:"admins"`
}

// ChatResponse générique pour les réponses de chat
type ChatResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
}
