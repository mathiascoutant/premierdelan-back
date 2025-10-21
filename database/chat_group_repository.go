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

// ChatGroupRepository gère les opérations sur les groupes de chat
type ChatGroupRepository struct {
	collection        *mongo.Collection
	membersCollection *mongo.Collection
	userCollection    *mongo.Collection
}

// NewChatGroupRepository crée une nouvelle instance
func NewChatGroupRepository(db *mongo.Database) *ChatGroupRepository {
	return &ChatGroupRepository{
		collection:        db.Collection("chat_groups"),
		membersCollection: db.Collection("chat_group_members"),
		userCollection:    db.Collection("users"),
	}
}

// Create crée un nouveau groupe
func (r *ChatGroupRepository) Create(group *models.ChatGroup) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	group.ID = primitive.NewObjectID()
	group.CreatedAt = time.Now()
	group.UpdatedAt = time.Now()
	group.IsActive = true

	_, err := r.collection.InsertOne(ctx, group)
	if err != nil {
		return fmt.Errorf("erreur lors de la création du groupe: %w", err)
	}

	return nil
}

// FindByID trouve un groupe par ID
func (r *ChatGroupRepository) FindByID(groupID primitive.ObjectID) (*models.ChatGroup, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var group models.ChatGroup
	err := r.collection.FindOne(ctx, bson.M{"_id": groupID, "is_active": true}).Decode(&group)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la recherche du groupe: %w", err)
	}

	return &group, nil
}

// FindGroupsByUserID trouve tous les groupes d'un utilisateur
func (r *ChatGroupRepository) FindGroupsByUserID(userID string) ([]models.ChatGroup, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Trouver les IDs de groupes dont l'utilisateur est membre
	memberCursor, err := r.membersCollection.Find(ctx, bson.M{"user_id": userID})
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la recherche des membres: %w", err)
	}
	defer memberCursor.Close(ctx)

	var members []models.ChatGroupMember
	if err = memberCursor.All(ctx, &members); err != nil {
		return nil, err
	}

	if len(members) == 0 {
		return []models.ChatGroup{}, nil
	}

	// Extraire les IDs des groupes
	groupIDs := make([]primitive.ObjectID, len(members))
	for i, member := range members {
		groupIDs[i] = member.GroupID
	}

	// Trouver tous les groupes
	groupCursor, err := r.collection.Find(ctx, bson.M{
		"_id":       bson.M{"$in": groupIDs},
		"is_active": true,
	})
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la recherche des groupes: %w", err)
	}
	defer groupCursor.Close(ctx)

	var groups []models.ChatGroup
	if err = groupCursor.All(ctx, &groups); err != nil {
		return nil, err
	}

	return groups, nil
}

// AddMember ajoute un membre à un groupe
func (r *ChatGroupRepository) AddMember(member *models.ChatGroupMember) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	member.ID = primitive.NewObjectID()
	member.JoinedAt = time.Now()

	_, err := r.membersCollection.InsertOne(ctx, member)
	if err != nil {
		return fmt.Errorf("erreur lors de l'ajout du membre: %w", err)
	}

	return nil
}

// IsMember vérifie si un utilisateur est membre d'un groupe
func (r *ChatGroupRepository) IsMember(groupID primitive.ObjectID, userID string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	count, err := r.membersCollection.CountDocuments(ctx, bson.M{
		"group_id": groupID,
		"user_id":  userID,
	})
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// IsAdmin vérifie si un utilisateur est admin d'un groupe
func (r *ChatGroupRepository) IsAdmin(groupID primitive.ObjectID, userID string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	count, err := r.membersCollection.CountDocuments(ctx, bson.M{
		"group_id": groupID,
		"user_id":  userID,
		"role":     "admin",
	})
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// GetMembers récupère tous les membres d'un groupe avec leurs infos
func (r *ChatGroupRepository) GetMembers(groupID primitive.ObjectID) ([]models.GroupMemberWithDetails, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Pipeline d'agrégation pour joindre les infos utilisateur
	pipeline := []bson.M{
		{"$match": bson.M{"group_id": groupID}},
		{
			"$lookup": bson.M{
				"from":         "users",
				"localField":   "user_id",
				"foreignField": "email",
				"as":           "user_info",
			},
		},
		{"$unwind": "$user_info"},
		{
			"$project": bson.M{
				"user_id":   1,
				"role":      1,
				"joined_at": 1,
				"firstname": "$user_info.firstname",
				"lastname":  "$user_info.lastname",
				"email":     "$user_info.email",
			},
		},
		{"$sort": bson.M{"joined_at": 1}},
	}

	cursor, err := r.membersCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la récupération des membres: %w", err)
	}
	defer cursor.Close(ctx)

	var members []models.GroupMemberWithDetails
	for cursor.Next(ctx) {
		var result struct {
			UserID    string    `bson:"user_id"`
			Role      string    `bson:"role"`
			JoinedAt  time.Time `bson:"joined_at"`
			Firstname string    `bson:"firstname"`
			Lastname  string    `bson:"lastname"`
			Email     string    `bson:"email"`
		}
		if err := cursor.Decode(&result); err != nil {
			continue
		}

		members = append(members, models.GroupMemberWithDetails{
			ID:        result.UserID,
			Firstname: result.Firstname,
			Lastname:  result.Lastname,
			Email:     result.Email,
			Role:      result.Role,
			JoinedAt:  result.JoinedAt,
			IsOnline:  false, // TODO: Intégrer avec le système de présence WebSocket
		})
	}

	return members, nil
}

// GetMemberCount compte le nombre de membres d'un groupe
func (r *ChatGroupRepository) GetMemberCount(groupID primitive.ObjectID) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	count, err := r.membersCollection.CountDocuments(ctx, bson.M{"group_id": groupID})
	if err != nil {
		return 0, err
	}

	return int(count), nil
}

// GetUserRole récupère le rôle d'un utilisateur dans un groupe
func (r *ChatGroupRepository) GetUserRole(groupID primitive.ObjectID, userID string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var member models.ChatGroupMember
	err := r.membersCollection.FindOne(ctx, bson.M{
		"group_id": groupID,
		"user_id":  userID,
	}).Decode(&member)

	if err == mongo.ErrNoDocuments {
		return "", nil
	}
	if err != nil {
		return "", err
	}

	return member.Role, nil
}

// RemoveMember supprime un membre d'un groupe
func (r *ChatGroupRepository) RemoveMember(groupID primitive.ObjectID, userID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.membersCollection.DeleteOne(ctx, bson.M{
		"group_id": groupID,
		"user_id":  userID,
	})
	if err != nil {
		return fmt.Errorf("erreur lors de la suppression du membre: %w", err)
	}

	return nil
}

// GetGroupsWithDetails récupère les groupes avec tous les détails pour un utilisateur
func (r *ChatGroupRepository) GetGroupsWithDetails(userID string) ([]models.GroupWithDetails, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Pipeline complexe pour récupérer tous les détails
	pipeline := []bson.M{
		// 1. Filtrer les membres de l'utilisateur
		{"$match": bson.M{"user_id": userID}},
		// 2. Joindre avec les groupes
		{
			"$lookup": bson.M{
				"from":         "chat_groups",
				"localField":   "group_id",
				"foreignField": "_id",
				"as":           "group",
			},
		},
		{"$unwind": "$group"},
		// 3. Filtrer les groupes actifs
		{"$match": bson.M{"group.is_active": true}},
		// 4. Joindre avec les infos du créateur
		{
			"$lookup": bson.M{
				"from":         "users",
				"localField":   "group.created_by",
				"foreignField": "email",
				"as":           "creator",
			},
		},
		{"$unwind": "$creator"},
		// 5. Compter les membres
		{
			"$lookup": bson.M{
				"from":         "chat_group_members",
				"localField":   "group_id",
				"foreignField": "group_id",
				"as":           "members",
			},
		},
		{
			"$addFields": bson.M{
				"member_count": bson.M{"$size": "$members"},
			},
		},
		// 6. Trier par date de mise à jour
		{"$sort": bson.M{"group.updated_at": -1}},
	}

	cursor, err := r.membersCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la récupération des groupes: %w", err)
	}
	defer cursor.Close(ctx)

	var groups []models.GroupWithDetails
	for cursor.Next(ctx) {
		var result struct {
			GroupID     primitive.ObjectID `bson:"group_id"`
			Role        string             `bson:"role"`
			JoinedAt    time.Time          `bson:"joined_at"`
			Group       models.ChatGroup   `bson:"group"`
			Creator     models.User        `bson:"creator"`
			MemberCount int                `bson:"member_count"`
		}
		if err := cursor.Decode(&result); err != nil {
			continue
		}

		groups = append(groups, models.GroupWithDetails{
			ID:   result.Group.ID.Hex(),
			Name: result.Group.Name,
			CreatedBy: models.GroupCreatorInfo{
				ID:        result.Creator.Email,
				Firstname: result.Creator.Firstname,
				Lastname:  result.Creator.Lastname,
				Email:     result.Creator.Email,
			},
			MemberCount: result.MemberCount,
			UnreadCount: 0,
			CreatedAt:   result.Group.CreatedAt,
		})
	}

	return groups, nil
}

// Delete supprime (désactive) un groupe
func (r *ChatGroupRepository) Delete(groupID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": groupID},
		bson.M{"$set": bson.M{
			"is_active":  false,
			"updated_at": time.Now(),
		}},
	)
	if err != nil {
		return fmt.Errorf("erreur lors de la suppression du groupe: %w", err)
	}

	return nil
}

// Update met à jour un groupe
func (r *ChatGroupRepository) Update(groupID primitive.ObjectID, updates bson.M) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	updates["updated_at"] = time.Now()

	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": groupID},
		bson.M{"$set": updates},
	)
	if err != nil {
		return fmt.Errorf("erreur lors de la mise à jour du groupe: %w", err)
	}

	return nil
}

// GetUserGroups récupère tous les groupes d'un utilisateur avec détails
func (r *ChatGroupRepository) GetUserGroups(userEmail string, messagesCollection *mongo.Collection) ([]models.GroupWithDetails, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	log := func(msg string, args ...interface{}) {
		fmt.Printf("[GetUserGroups] "+msg+"\n", args...)
	}

	log("Recherche groupes pour: %s", userEmail)

	// 1. Trouver les groupes où l'utilisateur est membre
	memberCursor, err := r.membersCollection.Find(ctx, bson.M{
		"user_id": userEmail,
	})
	if err != nil {
		return nil, fmt.Errorf("erreur récupération membres: %w", err)
	}
	defer memberCursor.Close(ctx)

	var groupIDs []primitive.ObjectID
	for memberCursor.Next(ctx) {
		var member models.ChatGroupMember
		if err := memberCursor.Decode(&member); err != nil {
			log("Erreur décodage membre: %v", err)
			continue
		}
		log("Membre trouvé: GroupID=%s, Role=%s", member.GroupID.Hex(), member.Role)
		groupIDs = append(groupIDs, member.GroupID)
	}

	log("Total groupes trouvés: %d", len(groupIDs))

	if len(groupIDs) == 0 {
		log("Aucun groupe trouvé pour cet utilisateur")
		return []models.GroupWithDetails{}, nil
	}

	// 2. Pipeline d'agrégation pour récupérer tous les détails
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"_id":       bson.M{"$in": groupIDs},
				"is_active": true,
			},
		},
		// Joindre avec le créateur
		{
			"$lookup": bson.M{
				"from":         "users",
				"localField":   "created_by",
				"foreignField": "email",
				"as":           "creator",
			},
		},
		{"$unwind": bson.M{"path": "$creator", "preserveNullAndEmptyArrays": true}},
		// Compter les membres
		{
			"$lookup": bson.M{
				"from":         "chat_group_members",
				"localField":   "_id",
				"foreignField": "group_id",
				"as":           "members",
			},
		},
		{
			"$addFields": bson.M{
				"member_count": bson.M{"$size": "$members"},
			},
		},
		{"$sort": bson.M{"created_at": -1}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("erreur agrégation groupes: %w", err)
	}
	defer cursor.Close(ctx)

	var groups []models.GroupWithDetails
	for cursor.Next(ctx) {
		var result struct {
			ID          primitive.ObjectID `bson:"_id"`
			Name        string             `bson:"name"`
			CreatedBy   string             `bson:"created_by"`
			CreatedAt   time.Time          `bson:"created_at"`
			MemberCount int                `bson:"member_count"`
			Creator     *models.User       `bson:"creator"`
		}
		if err := cursor.Decode(&result); err != nil {
			continue
		}

		group := models.GroupWithDetails{
			ID:          result.ID.Hex(),
			Name:        result.Name,
			MemberCount: result.MemberCount,
			UnreadCount: 0,
			CreatedAt:   result.CreatedAt,
		}

		// Infos du créateur
		if result.Creator != nil {
			group.CreatedBy = models.GroupCreatorInfo{
				ID:        result.Creator.Email,
				Firstname: result.Creator.Firstname,
				Lastname:  result.Creator.Lastname,
				Email:     result.Creator.Email,
			}
		} else {
			group.CreatedBy = models.GroupCreatorInfo{
				ID:        result.CreatedBy,
				Firstname: "Inconnu",
				Lastname:  "",
				Email:     result.CreatedBy,
			}
		}

		// Récupérer le dernier message
		lastMessage, lastMessageSenderID, lastMessageID, err := r.getLastGroupMessage(ctx, result.ID, messagesCollection)
		if err == nil && lastMessage != nil {
			group.LastMessage = lastMessage
			
			// ✅ Si le dernier message est de l'utilisateur → pas de badge
			if lastMessageSenderID == userEmail {
				group.UnreadCount = 0
			} else {
				// Vérifier si l'utilisateur a lu les messages (via read receipt)
				unreadCount, err := r.countUnreadMessagesWithReceipt(ctx, result.ID, userEmail, lastMessageID, messagesCollection)
				if err == nil {
					group.UnreadCount = unreadCount
				}
			}
		} else {
			// Pas de dernier message → pas de badge
			group.UnreadCount = 0
		}

		groups = append(groups, group)
	}

	// Trier par dernier message (les plus récents en premier)
	// Les groupes sans message restent triés par created_at
	return groups, nil
}

// getLastGroupMessage récupère le dernier message d'un groupe (retourne message, sender_id, message_id)
func (r *ChatGroupRepository) getLastGroupMessage(ctx context.Context, groupID primitive.ObjectID, messagesCollection *mongo.Collection) (*models.GroupLastMessageInfo, string, primitive.ObjectID, error) {
	var result struct {
		ID        primitive.ObjectID `bson:"_id"`
		Content   string             `bson:"content"`
		SenderID  string             `bson:"sender_id"`
		Timestamp time.Time          `bson:"timestamp"`
		CreatedAt time.Time          `bson:"created_at"`
	}

	err := messagesCollection.FindOne(
		ctx,
		bson.M{"group_id": groupID},
		options.FindOne().SetSort(bson.M{"created_at": -1}),
	).Decode(&result)

	if err == mongo.ErrNoDocuments {
		return nil, "", primitive.NilObjectID, nil
	}
	if err != nil {
		return nil, "", primitive.NilObjectID, err
	}

	// Récupérer le nom de l'expéditeur
	var sender models.User
	err = r.userCollection.FindOne(ctx, bson.M{"email": result.SenderID}).Decode(&sender)
	senderName := "Inconnu"
	if err == nil {
		senderName = sender.Firstname + " " + sender.Lastname
	}

	return &models.GroupLastMessageInfo{
		Content:    result.Content,
		SenderName: senderName,
		Timestamp:  result.Timestamp,
		CreatedAt:  result.CreatedAt,
	}, result.SenderID, result.ID, nil
}

// countUnreadMessagesWithReceipt compte les messages non lus en vérifiant le read receipt
func (r *ChatGroupRepository) countUnreadMessagesWithReceipt(ctx context.Context, groupID primitive.ObjectID, userEmail string, lastMessageID primitive.ObjectID, messagesCollection *mongo.Collection) (int, error) {
	// Récupérer le read receipt de l'utilisateur dans ce groupe
	readReceiptCollection := messagesCollection.Database().Collection("chat_group_read_receipts")
	var receipt models.ChatGroupReadReceipt
	err := readReceiptCollection.FindOne(ctx, bson.M{
		"group_id": groupID,
		"user_id":  userEmail,
	}).Decode(&receipt)

	// Si pas de read receipt, compter tous les messages des autres
	if err == mongo.ErrNoDocuments {
		count, _ := messagesCollection.CountDocuments(ctx, bson.M{
			"group_id":  groupID,
			"sender_id": bson.M{"$ne": userEmail},
		})
		return int(count), nil
	}

	// Si l'utilisateur a lu jusqu'au dernier message → 0
	if receipt.LastReadMessageID != nil && *receipt.LastReadMessageID == lastMessageID {
		return 0, nil
	}

	// Compter les messages après le dernier lu
	var filter bson.M
	if receipt.LastReadMessageID != nil {
		filter = bson.M{
			"group_id":  groupID,
			"sender_id": bson.M{"$ne": userEmail},
			"_id":       bson.M{"$gt": *receipt.LastReadMessageID},
		}
	} else {
		filter = bson.M{
			"group_id":  groupID,
			"sender_id": bson.M{"$ne": userEmail},
		}
	}

	count, err := messagesCollection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, err
	}
	return int(count), nil
}
