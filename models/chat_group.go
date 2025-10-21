package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ChatGroup représente un groupe de chat
type ChatGroup struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name      string             `json:"name" bson:"name"`
	CreatedBy string             `json:"created_by" bson:"created_by"` // User ID (email)
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time          `json:"updated_at" bson:"updated_at"`
	IsActive  bool               `json:"is_active" bson:"is_active"`
}

// ChatGroupMember représente un membre d'un groupe
type ChatGroupMember struct {
	ID       primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	GroupID  primitive.ObjectID `json:"group_id" bson:"group_id"`
	UserID   string             `json:"user_id" bson:"user_id"` // User ID (email)
	Role     string             `json:"role" bson:"role"`       // "admin" ou "member"
	JoinedAt time.Time          `json:"joined_at" bson:"joined_at"`
}

// ChatGroupInvitation représente une invitation à un groupe
type ChatGroupInvitation struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	GroupID     primitive.ObjectID `json:"group_id" bson:"group_id"`
	InvitedBy   string             `json:"invited_by" bson:"invited_by"`     // User ID qui invite
	InvitedUser string             `json:"invited_user" bson:"invited_user"` // User ID invité
	Message     string             `json:"message,omitempty" bson:"message,omitempty"`
	Status      string             `json:"status" bson:"status"` // "pending", "accepted", "rejected"
	InvitedAt   time.Time          `json:"invited_at" bson:"invited_at"`
	RespondedAt *time.Time         `json:"responded_at,omitempty" bson:"responded_at,omitempty"`
}

// ChatGroupMessage représente un message dans un groupe
type ChatGroupMessage struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	GroupID     primitive.ObjectID `json:"group_id" bson:"group_id"`
	SenderID    string             `json:"sender_id" bson:"sender_id"`
	Content     string             `json:"content" bson:"content"`
	MessageType string             `json:"message_type" bson:"message_type"` // "message" ou "system"
	Timestamp   time.Time          `json:"timestamp" bson:"timestamp"`
	CreatedAt   time.Time          `json:"created_at" bson:"created_at"`
	DeliveredAt *time.Time         `json:"delivered_at,omitempty" bson:"delivered_at,omitempty"`
	ReadBy      []string           `json:"read_by" bson:"read_by"` // Liste d'emails qui ont lu
}

// ChatGroupReadReceipt représente le statut de lecture d'un utilisateur dans un groupe
type ChatGroupReadReceipt struct {
	ID                primitive.ObjectID  `json:"id" bson:"_id,omitempty"`
	GroupID           primitive.ObjectID  `json:"group_id" bson:"group_id"`
	UserID            string              `json:"user_id" bson:"user_id"`
	LastReadMessageID *primitive.ObjectID `json:"last_read_message_id,omitempty" bson:"last_read_message_id,omitempty"`
	LastReadAt        time.Time           `json:"last_read_at" bson:"last_read_at"`
}

// ========================
// Request/Response DTOs
// ========================

// CreateGroupRequest pour créer un groupe
type CreateGroupRequest struct {
	Name      string   `json:"name"`
	MemberIDs []string `json:"member_ids"` // Liste d'IDs utilisateurs à inviter
}

// InviteToGroupRequest pour inviter un utilisateur
type InviteToGroupRequest struct {
	UserID  string `json:"user_id"`
	Message string `json:"message,omitempty"`
}

// RespondToGroupInvitationRequest pour répondre à une invitation
type RespondToGroupInvitationRequest struct {
	Action string `json:"action"` // "accept" ou "reject"
}

// SendGroupMessageRequest pour envoyer un message
type SendGroupMessageRequest struct {
	Content string `json:"content"`
}

// UserBasicInfo informations de base d'un utilisateur
type UserBasicInfo struct {
	ID        string `json:"id"`
	Firstname string `json:"firstname"`
	Lastname  string `json:"lastname"`
	Email     string `json:"email,omitempty"`
}

// MessagePreview aperçu d'un message
type MessagePreview struct {
	Content   string        `json:"content"`
	Sender    UserBasicInfo `json:"sender"`
	Timestamp time.Time     `json:"timestamp"`
}

// GroupInvitationWithDetails invitation avec détails enrichis
type GroupInvitationWithDetails struct {
	ID        primitive.ObjectID `json:"id"`
	Group     GroupBasicInfo     `json:"group"`
	InvitedBy UserBasicInfo      `json:"invited_by"`
	Message   string             `json:"message,omitempty"`
	InvitedAt time.Time          `json:"invited_at"`
}

// GroupBasicInfo informations de base d'un groupe
type GroupBasicInfo struct {
	ID          primitive.ObjectID `json:"id"`
	Name        string             `json:"name"`
	MemberCount int                `json:"member_count"`
	CreatedBy   UserBasicInfo      `json:"created_by"`
}

// GroupWithDetails groupe avec tous les détails pour la liste
type GroupWithDetails struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	CreatedBy    GroupCreatorInfo       `json:"created_by"`
	MemberCount  int                    `json:"member_count"`
	UnreadCount  int                    `json:"unread_count"`
	LastMessage  *GroupLastMessageInfo  `json:"last_message"`
	CreatedAt    time.Time              `json:"created_at"`
}

// GroupCreatorInfo informations du créateur du groupe
type GroupCreatorInfo struct {
	ID        string `json:"id"`
	Firstname string `json:"firstname"`
	Lastname  string `json:"lastname"`
	Email     string `json:"email"`
}

// GroupLastMessageInfo dernier message du groupe
type GroupLastMessageInfo struct {
	Content    string    `json:"content"`
	SenderName string    `json:"sender_name"`
	Timestamp  time.Time `json:"timestamp"`
	CreatedAt  time.Time `json:"created_at"`
}

// GroupMemberWithDetails membre avec détails
type GroupMemberWithDetails struct {
	ID        string    `json:"id"`
	Firstname string    `json:"firstname"`
	Lastname  string    `json:"lastname"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	JoinedAt  time.Time `json:"joined_at"`
	IsOnline  bool      `json:"is_online"`
	LastSeen  time.Time `json:"last_seen,omitempty"`
}

// GroupMessageWithSender message avec informations de l'expéditeur
type GroupMessageWithSender struct {
	ID          primitive.ObjectID `json:"id"`
	SenderID    string             `json:"sender_id"`
	Sender      *UserBasicInfo     `json:"sender,omitempty"`
	Content     string             `json:"content"`
	MessageType string             `json:"message_type"`
	Timestamp   time.Time          `json:"timestamp"`
	CreatedAt   time.Time          `json:"created_at"`
	DeliveredAt *time.Time         `json:"delivered_at,omitempty"`
	ReadBy      []string           `json:"read_by"`
}

// PendingInvitationWithUser invitation en attente avec infos utilisateur
type PendingInvitationWithUser struct {
	ID        primitive.ObjectID `json:"id"`
	User      UserBasicInfo      `json:"user"`
	InvitedBy UserBasicInfo      `json:"invited_by"`
	Status    string             `json:"status"`
	InvitedAt time.Time          `json:"invited_at"`
}
