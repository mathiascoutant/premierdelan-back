package database

import (
	"context"
	"fmt"
	"premier-an-backend/models"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ChatGroupMessageRepository gère les messages de groupe
type ChatGroupMessageRepository struct {
	collection            *mongo.Collection
	userCollection        *mongo.Collection
	readReceiptCollection *mongo.Collection
}

// NewChatGroupMessageRepository crée une nouvelle instance
func NewChatGroupMessageRepository(db *mongo.Database) *ChatGroupMessageRepository {
	return &ChatGroupMessageRepository{
		collection:            db.Collection("chat_group_messages"),
		userCollection:        db.Collection("users"),
		readReceiptCollection: db.Collection("chat_group_read_receipts"),
	}
}

// Collection retourne la collection MongoDB (pour les requêtes externes)
func (r *ChatGroupMessageRepository) Collection() *mongo.Collection {
	return r.collection
}

// Create crée un nouveau message
func (r *ChatGroupMessageRepository) Create(message *models.ChatGroupMessage) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	message.ID = primitive.NewObjectID()
	message.CreatedAt = time.Now()

	if message.MessageType == "" {
		message.MessageType = "message"
	}

	_, err := r.collection.InsertOne(ctx, message)
	if err != nil {
		return fmt.Errorf("erreur lors de la création du message: %w", err)
	}

	return nil
}

// FindByGroupID récupère les messages d'un groupe avec pagination
func (r *ChatGroupMessageRepository) FindByGroupID(groupID primitive.ObjectID, limit int, before *primitive.ObjectID) ([]models.GroupMessageWithSender, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Construire le filtre
	filter := bson.M{"group_id": groupID}
	if before != nil {
		filter["_id"] = bson.M{"$lt": *before}
	}

	// Pipeline d'agrégation pour joindre les infos de l'expéditeur
	pipeline := []bson.M{
		{"$match": filter},
		{"$sort": bson.M{"created_at": -1}},
		{"$limit": limit},
		// Lookup conditionnnel : ne pas joindre si c'est un message système
		{
			"$lookup": bson.M{
				"from": "users",
				"let":  bson.M{"sender_id": "$sender_id", "msg_type": "$message_type"},
				"pipeline": []bson.M{
					{
						"$match": bson.M{
							"$expr": bson.M{
								"$and": []bson.M{
									{"$eq": []interface{}{"$email", "$$sender_id"}},
									{"$ne": []interface{}{"$$msg_type", "system"}},
								},
							},
						},
					},
				},
				"as": "sender",
			},
		},
		// Inverser l'ordre pour avoir du plus ancien au plus récent
		{"$sort": bson.M{"created_at": 1}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la récupération des messages: %w", err)
	}
	defer cursor.Close(ctx)

	var messages []models.GroupMessageWithSender
	for cursor.Next(ctx) {
		var result struct {
			ID          primitive.ObjectID `bson:"_id"`
			SenderID    string             `bson:"sender_id"`
			Content     string             `bson:"content"`
			MessageType string             `bson:"message_type"`
			CreatedAt   time.Time          `bson:"created_at"`
			Sender      []models.User      `bson:"sender"`
		}
		if err := cursor.Decode(&result); err != nil {
			continue
		}

		msg := models.GroupMessageWithSender{
			ID:          result.ID,
			SenderID:    result.SenderID,
			Content:     result.Content,
			MessageType: result.MessageType,
			CreatedAt:   result.CreatedAt,
		}

		// Ajouter les infos de l'expéditeur si ce n'est pas un message système
		if result.MessageType != "system" && len(result.Sender) > 0 {
			sender := result.Sender[0]
			msg.Sender = &models.UserBasicInfo{
				ID:        sender.Email,
				Firstname: sender.Firstname,
				Lastname:  sender.Lastname,
			}
		}

		messages = append(messages, msg)
	}

	return messages, nil
}

// GetLastMessageByGroup récupère le dernier message d'un groupe
func (r *ChatGroupMessageRepository) GetLastMessageByGroup(groupID primitive.ObjectID) (*models.MessagePreview, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pipeline := []bson.M{
		{"$match": bson.M{"group_id": groupID}},
		{"$sort": bson.M{"created_at": -1}},
		{"$limit": 1},
		{
			"$lookup": bson.M{
				"from": "users",
				"let":  bson.M{"sender_id": "$sender_id", "msg_type": "$message_type"},
				"pipeline": []bson.M{
					{
						"$match": bson.M{
							"$expr": bson.M{
								"$and": []bson.M{
									{"$eq": []interface{}{"$email", "$$sender_id"}},
									{"$ne": []interface{}{"$$msg_type", "system"}},
								},
							},
						},
					},
				},
				"as": "sender",
			},
		},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la récupération du dernier message: %w", err)
	}
	defer cursor.Close(ctx)

	if !cursor.Next(ctx) {
		return nil, nil
	}

	var result struct {
		Content   string        `bson:"content"`
		CreatedAt time.Time     `bson:"created_at"`
		Sender    []models.User `bson:"sender"`
	}
	if err := cursor.Decode(&result); err != nil {
		return nil, err
	}

	preview := &models.MessagePreview{
		Content:   result.Content,
		Timestamp: result.CreatedAt,
		Sender: models.UserBasicInfo{
			ID:        "system",
			Firstname: "Système",
			Lastname:  "",
		},
	}

	if len(result.Sender) > 0 {
		sender := result.Sender[0]
		preview.Sender = models.UserBasicInfo{
			ID:        sender.Email,
			Firstname: sender.Firstname,
			Lastname:  sender.Lastname,
		}
	}

	return preview, nil
}

// MarkAsRead marque les messages comme lus pour un utilisateur
func (r *ChatGroupMessageRepository) MarkAsRead(groupID primitive.ObjectID, userID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Récupérer le dernier message du groupe
	var lastMessage models.ChatGroupMessage
	opts := options.FindOne().SetSort(bson.D{{Key: "created_at", Value: -1}})
	err := r.collection.FindOne(ctx, bson.M{"group_id": groupID}, opts).Decode(&lastMessage)
	if err != nil && err != mongo.ErrNoDocuments {
		return fmt.Errorf("erreur lors de la récupération du dernier message: %w", err)
	}

	// Créer ou mettre à jour le read receipt
	filter := bson.M{
		"group_id": groupID,
		"user_id":  userID,
	}

	update := bson.M{
		"$set": bson.M{
			"last_read_at": time.Now(),
		},
	}

	if err == nil {
		update["$set"].(bson.M)["last_read_message_id"] = lastMessage.ID
	}

	opts2 := options.Update().SetUpsert(true)
	_, err = r.readReceiptCollection.UpdateOne(ctx, filter, update, opts2)
	if err != nil {
		return fmt.Errorf("erreur lors de la mise à jour du read receipt: %w", err)
	}

	return nil
}

// GetUnreadCount compte les messages non lus pour un utilisateur dans un groupe
func (r *ChatGroupMessageRepository) GetUnreadCount(groupID primitive.ObjectID, userID string) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Récupérer le read receipt
	var receipt models.ChatGroupReadReceipt
	err := r.readReceiptCollection.FindOne(ctx, bson.M{
		"group_id": groupID,
		"user_id":  userID,
	}).Decode(&receipt)

	if err == mongo.ErrNoDocuments {
		// Aucun read receipt = tous les messages sont non lus
		count, err := r.collection.CountDocuments(ctx, bson.M{"group_id": groupID})
		return int(count), err
	}

	if err != nil {
		return 0, err
	}

	// Compter les messages plus récents que le dernier lu
	filter := bson.M{"group_id": groupID}
	if receipt.LastReadMessageID != nil {
		filter["_id"] = bson.M{"$gt": *receipt.LastReadMessageID}
	}

	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, err
	}

	return int(count), nil
}

// DeleteByGroupID supprime tous les messages d'un groupe
func (r *ChatGroupMessageRepository) DeleteByGroupID(groupID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := r.collection.DeleteMany(ctx, bson.M{"group_id": groupID})
	if err != nil {
		return fmt.Errorf("erreur lors de la suppression des messages: %w", err)
	}

	return nil
}
