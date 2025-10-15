package database

import (
	"context"
	"time"

	"premier-an-backend/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ChatRepository gère les opérations sur les conversations et messages
type ChatRepository struct {
	conversationCollection *mongo.Collection
	messageCollection      *mongo.Collection
	InvitationCollection   *mongo.Collection // Exporté pour accès depuis le handler
	userCollection         *mongo.Collection
}

// NewChatRepository crée un nouveau repository pour le chat
func NewChatRepository(db *mongo.Database) *ChatRepository {
	return &ChatRepository{
		conversationCollection: db.Collection("conversations"),
		messageCollection:      db.Collection("messages"),
		InvitationCollection:   db.Collection("chat_invitations"),
		userCollection:         db.Collection("users"),
	}
}

// GetConversationsAndInvitations récupère les conversations ET les invitations envoyées d'un utilisateur
func (r *ChatRepository) GetConversationsAndInvitations(ctx context.Context, userID primitive.ObjectID) ([]models.ConversationResponse, error) {
	var allConversations []models.ConversationResponse
	
	// 1. Récupérer les conversations acceptées
	conversations, err := r.GetConversations(ctx, userID)
	if err != nil {
		return nil, err
	}
	allConversations = append(allConversations, conversations...)
	
	// 2. Récupérer les invitations envoyées (status: pending)
	sentInvitations, err := r.getSentInvitations(ctx, userID)
	if err == nil {
		allConversations = append(allConversations, sentInvitations...)
	}
	
	return allConversations, nil
}

// GetConversations récupère les conversations acceptées d'un utilisateur
func (r *ChatRepository) GetConversations(ctx context.Context, userID primitive.ObjectID) ([]models.ConversationResponse, error) {
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"participants.user_id": userID,
				"status": bson.M{"$in": []string{"accepted", "active"}},
			},
		},
		{
			"$lookup": bson.M{
				"from":         "messages",
				"localField":   "_id",
				"foreignField": "conversation_id",
				"as":           "messages",
			},
		},
		{
			"$lookup": bson.M{
				"from":         "users",
				"let":          bson.M{"participants": "$participants"},
				"pipeline": []bson.M{
					{
						"$match": bson.M{
							"$expr": bson.M{
								"$and": []bson.M{
									{"$in": []interface{}{"$_id", "$$participants.user_id"}},
									{"$ne": []interface{}{"$_id", userID}},
								},
							},
						},
					},
				},
				"as": "other_participants",
			},
		},
		{
			"$addFields": bson.M{
				"last_message": bson.M{
					"$arrayElemAt": []interface{}{"$messages", -1},
				},
				"unread_count": bson.M{
					"$size": bson.M{
						"$filter": bson.M{
							"input": "$messages",
							"cond": bson.M{
								"$and": []bson.M{
									{"$ne": []interface{}{"$$this.sender_id", userID}},
									{"$not": bson.M{"$in": []interface{}{userID, "$$this.read_by.user_id"}}},
								},
							},
						},
					},
				},
			},
		},
		{
			"$sort": bson.M{"last_message_at": -1},
		},
	}

	cursor, err := r.conversationCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var conversations []models.ConversationResponse
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}

		// Extraire les informations de la conversation
		conversationID := doc["_id"].(primitive.ObjectID).Hex()
		status := doc["status"].(string)

		// Extraire le participant (autre utilisateur)
		var participant models.UserInfo
		if otherParticipants, ok := doc["other_participants"].(primitive.A); ok && len(otherParticipants) > 0 {
			if otherUser, ok := otherParticipants[0].(bson.M); ok {
				participant.ID = otherUser["_id"].(primitive.ObjectID).Hex()
				participant.Firstname = otherUser["firstname"].(string)
				participant.Lastname = otherUser["lastname"].(string)
				participant.Email = otherUser["email"].(string)
			}
		}

		// Extraire le dernier message
		var lastMessage *models.MessageInfo
		if lastMsg, ok := doc["last_message"].(bson.M); ok {
			// Convertir primitive.DateTime en time.Time
			var timestamp time.Time
			if createdAt, ok := lastMsg["created_at"].(primitive.DateTime); ok {
				timestamp = createdAt.Time()
			}
			
			lastMessage = &models.MessageInfo{
				Content:   lastMsg["content"].(string),
				Timestamp: timestamp,
				IsRead:    lastMsg["is_read"].(bool),
			}
		}

		// Extraire le nombre de messages non lus
		unreadCount := 0
		if count, ok := doc["unread_count"].(int32); ok {
			unreadCount = int(count)
		}

		conversations = append(conversations, models.ConversationResponse{
			ID:           conversationID,
			Participant:  participant,
			LastMessage:  lastMessage,
			Status:       status,
			UnreadCount:  unreadCount,
		})
	}

	return conversations, nil
}

// GetMessages récupère les messages d'une conversation
func (r *ChatRepository) GetMessages(ctx context.Context, conversationID primitive.ObjectID, limit int) ([]models.Message, error) {
	filter := bson.M{"conversation_id": conversationID}
	opts := options.Find().SetSort(bson.D{{"created_at", -1}}).SetLimit(int64(limit))

	cursor, err := r.messageCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var messages []models.Message
	if err = cursor.All(ctx, &messages); err != nil {
		return nil, err
	}

	// Inverser l'ordre pour avoir les plus anciens en premier
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// SendMessage envoie un message dans une conversation
func (r *ChatRepository) SendMessage(ctx context.Context, message *models.Message) error {
	message.CreatedAt = time.Now()
	message.IsRead = false
	message.ReadBy = []models.ReadReceipt{}

	_, err := r.messageCollection.InsertOne(ctx, message)
	if err != nil {
		return err
	}

	// Mettre à jour la conversation avec la date du dernier message
	now := time.Now()
	_, err = r.conversationCollection.UpdateOne(
		ctx,
		bson.M{"_id": message.ConversationID},
		bson.M{
			"$set": bson.M{
				"last_message_at": now,
				"updated_at":      now,
			},
		},
	)

	return err
}

// SearchAdmins recherche des administrateurs
func (r *ChatRepository) SearchAdmins(ctx context.Context, query string, limit int) ([]models.UserInfo, error) {
	filter := bson.M{
		"admin": 1,
		"$or": []bson.M{
			{"firstname": bson.M{"$regex": query, "$options": "i"}},
			{"lastname": bson.M{"$regex": query, "$options": "i"}},
			{"email": bson.M{"$regex": query, "$options": "i"}},
		},
	}

	opts := options.Find().SetLimit(int64(limit))
	cursor, err := r.userCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var admins []models.UserInfo
	for cursor.Next(ctx) {
		var user bson.M
		if err := cursor.Decode(&user); err != nil {
			continue
		}

		admins = append(admins, models.UserInfo{
			ID:        user["_id"].(primitive.ObjectID).Hex(),
			Firstname: user["firstname"].(string),
			Lastname:  user["lastname"].(string),
			Email:     user["email"].(string),
		})
	}

	return admins, nil
}

// CreateInvitation crée une invitation de chat
func (r *ChatRepository) CreateInvitation(ctx context.Context, invitation *models.ChatInvitation) error {
	invitation.CreatedAt = time.Now()
	invitation.Status = "pending"
	
	result, err := r.InvitationCollection.InsertOne(ctx, invitation)
	if err != nil {
		return err
	}
	
	// Définir l'ID généré par MongoDB
	if insertedID, ok := result.InsertedID.(primitive.ObjectID); ok {
		invitation.ID = insertedID
	}
	
	return nil
}

// GetInvitations récupère les invitations reçues par un utilisateur
func (r *ChatRepository) GetInvitations(ctx context.Context, userID primitive.ObjectID) ([]bson.M, error) {
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"to_user_id": userID,
				"status":     "pending",
			},
		},
		{
			"$lookup": bson.M{
				"from":         "users",
				"localField":   "from_user_id",
				"foreignField": "_id",
				"as":           "from_user",
			},
		},
		{
			"$unwind": "$from_user",
		},
		{
			"$sort": bson.M{"created_at": -1},
		},
	}

	cursor, err := r.InvitationCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var invitations []bson.M
	if err = cursor.All(ctx, &invitations); err != nil {
		return nil, err
	}

	return invitations, nil
}

// getSentInvitations récupère les invitations envoyées par un utilisateur
func (r *ChatRepository) getSentInvitations(ctx context.Context, userID primitive.ObjectID) ([]models.ConversationResponse, error) {
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"from_user_id": userID,
				"status":       "pending",
			},
		},
		{
			"$lookup": bson.M{
				"from":         "users",
				"localField":   "to_user_id",
				"foreignField": "_id",
				"as":           "to_user",
			},
		},
		{
			"$unwind": "$to_user",
		},
		{
			"$sort": bson.M{"created_at": -1},
		},
	}

	cursor, err := r.InvitationCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var invitations []models.ConversationResponse
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}

		// Extraire les informations de l'invitation
		invitationID := doc["_id"].(primitive.ObjectID).Hex()

		// Extraire le participant (destinataire)
		var participant models.UserInfo
		if toUser, ok := doc["to_user"].(bson.M); ok {
			participant.ID = toUser["_id"].(primitive.ObjectID).Hex()
			participant.Firstname = toUser["firstname"].(string)
			participant.Lastname = toUser["lastname"].(string)
			participant.Email = toUser["email"].(string)
		}

		invitations = append(invitations, models.ConversationResponse{
			ID:          invitationID,
			Participant: participant,
			Status:      "pending",
			UnreadCount: 0,
		})
	}

	return invitations, nil
}

// RespondToInvitation répond à une invitation
func (r *ChatRepository) RespondToInvitation(ctx context.Context, invitationID primitive.ObjectID, action string) (*models.Conversation, error) {
	now := time.Now()
	
	// Convertir "accept" en "accepted" et "reject" en "rejected"
	status := action + "ed"
	
	update := bson.M{
		"$set": bson.M{
			"status":       status,
			"responded_at": now,
		},
	}

	_, err := r.InvitationCollection.UpdateOne(
		ctx,
		bson.M{"_id": invitationID},
		update,
	)
	if err != nil {
		return nil, err
	}

	// Si acceptée, créer la conversation
	if action == "accept" {
		// Récupérer l'invitation pour créer la conversation
		var invitation models.ChatInvitation
		err = r.InvitationCollection.FindOne(ctx, bson.M{"_id": invitationID}).Decode(&invitation)
		if err != nil {
			return nil, err
		}

		conversation := &models.Conversation{
			Participants: []models.Participant{
				{UserID: invitation.FromUserID, Role: "admin", Status: "active"},
				{UserID: invitation.ToUserID, Role: "admin", Status: "active"},
			},
			Status:    "accepted",
			CreatedBy: invitation.FromUserID,
			CreatedAt: now,
			UpdatedAt: now,
		}

		_, err = r.conversationCollection.InsertOne(ctx, conversation)
		if err != nil {
			return nil, err
		}

		return conversation, nil
	}

	return nil, nil
}

// GetConversationByID récupère une conversation par son ID
func (r *ChatRepository) GetConversationByID(ctx context.Context, conversationID primitive.ObjectID) (*models.Conversation, error) {
	var conversation models.Conversation
	err := r.conversationCollection.FindOne(ctx, bson.M{"_id": conversationID}).Decode(&conversation)
	if err != nil {
		return nil, err
	}
	return &conversation, nil
}

// MarkMessageAsRead marque un message comme lu
func (r *ChatRepository) MarkMessageAsRead(ctx context.Context, messageID primitive.ObjectID, userID primitive.ObjectID) error {
	update := bson.M{
		"$addToSet": bson.M{
			"read_by": models.ReadReceipt{
				UserID: userID,
				ReadAt: time.Now(),
			},
		},
		"$set": bson.M{
			"is_read": true,
		},
	}

	_, err := r.messageCollection.UpdateOne(
		ctx,
		bson.M{"_id": messageID},
		update,
	)
	return err
}
