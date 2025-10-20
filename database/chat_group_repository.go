package database

import (
	"context"
	"fmt"
	"premier-an-backend/models"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
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
			ID:   result.Group.ID,
			Name: result.Group.Name,
			CreatedBy: models.UserBasicInfo{
				ID:        result.Creator.Email,
				Firstname: result.Creator.Firstname,
				Lastname:  result.Creator.Lastname,
			},
			MemberCount: result.MemberCount,
			UnreadCount: 0,   // TODO: Calculer avec read receipts
			LastMessage: nil, // TODO: Récupérer le dernier message
			IsAdmin:     result.Role == "admin",
			JoinedAt:    result.JoinedAt,
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
