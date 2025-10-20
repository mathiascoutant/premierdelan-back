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

// ChatGroupInvitationRepository gère les invitations de groupe
type ChatGroupInvitationRepository struct {
	collection      *mongo.Collection
	groupCollection *mongo.Collection
	userCollection  *mongo.Collection
}

// NewChatGroupInvitationRepository crée une nouvelle instance
func NewChatGroupInvitationRepository(db *mongo.Database) *ChatGroupInvitationRepository {
	return &ChatGroupInvitationRepository{
		collection:      db.Collection("chat_group_invitations"),
		groupCollection: db.Collection("chat_groups"),
		userCollection:  db.Collection("users"),
	}
}

// Create crée une nouvelle invitation
func (r *ChatGroupInvitationRepository) Create(invitation *models.ChatGroupInvitation) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	invitation.ID = primitive.NewObjectID()
	invitation.InvitedAt = time.Now()
	invitation.Status = "pending"

	_, err := r.collection.InsertOne(ctx, invitation)
	if err != nil {
		return fmt.Errorf("erreur lors de la création de l'invitation: %w", err)
	}

	return nil
}

// FindByID trouve une invitation par ID
func (r *ChatGroupInvitationRepository) FindByID(invitationID primitive.ObjectID) (*models.ChatGroupInvitation, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var invitation models.ChatGroupInvitation
	err := r.collection.FindOne(ctx, bson.M{"_id": invitationID}).Decode(&invitation)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la recherche de l'invitation: %w", err)
	}

	return &invitation, nil
}

// FindPendingByUser trouve toutes les invitations en attente pour un utilisateur
func (r *ChatGroupInvitationRepository) FindPendingByUser(userID string) ([]models.GroupInvitationWithDetails, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Pipeline d'agrégation pour enrichir les invitations
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"invited_user": userID,
				"status":       "pending",
			},
		},
		// Joindre avec les groupes
		{
			"$lookup": bson.M{
				"from":         "chat_groups",
				"localField":   "group_id",
				"foreignField": "_id",
				"as":           "group",
			},
		},
		{"$unwind": "$group"},
		// Joindre avec le créateur du groupe
		{
			"$lookup": bson.M{
				"from":         "users",
				"localField":   "group.created_by",
				"foreignField": "email",
				"as":           "group_creator",
			},
		},
		{"$unwind": "$group_creator"},
		// Joindre avec l'utilisateur qui a invité
		{
			"$lookup": bson.M{
				"from":         "users",
				"localField":   "invited_by",
				"foreignField": "email",
				"as":           "inviter",
			},
		},
		{"$unwind": "$inviter"},
		// Compter les membres
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
		{"$sort": bson.M{"invited_at": -1}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la récupération des invitations: %w", err)
	}
	defer cursor.Close(ctx)

	var invitations []models.GroupInvitationWithDetails
	for cursor.Next(ctx) {
		var result struct {
			ID           primitive.ObjectID `bson:"_id"`
			GroupID      primitive.ObjectID `bson:"group_id"`
			Message      string             `bson:"message"`
			InvitedAt    time.Time          `bson:"invited_at"`
			Group        models.ChatGroup   `bson:"group"`
			GroupCreator models.User        `bson:"group_creator"`
			Inviter      models.User        `bson:"inviter"`
			MemberCount  int                `bson:"member_count"`
		}
		if err := cursor.Decode(&result); err != nil {
			continue
		}

		invitations = append(invitations, models.GroupInvitationWithDetails{
			ID: result.ID,
			Group: models.GroupBasicInfo{
				ID:          result.GroupID,
				Name:        result.Group.Name,
				MemberCount: result.MemberCount,
				CreatedBy: models.UserBasicInfo{
					ID:        result.GroupCreator.Email,
					Firstname: result.GroupCreator.Firstname,
					Lastname:  result.GroupCreator.Lastname,
				},
			},
			InvitedBy: models.UserBasicInfo{
				ID:        result.Inviter.Email,
				Firstname: result.Inviter.Firstname,
				Lastname:  result.Inviter.Lastname,
			},
			Message:   result.Message,
			InvitedAt: result.InvitedAt,
		})
	}

	return invitations, nil
}

// FindPendingByGroup trouve toutes les invitations en attente pour un groupe
func (r *ChatGroupInvitationRepository) FindPendingByGroup(groupID primitive.ObjectID) ([]models.PendingInvitationWithUser, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Pipeline d'agrégation
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"group_id": groupID,
				"status":   "pending",
			},
		},
		// Joindre avec l'utilisateur invité
		{
			"$lookup": bson.M{
				"from":         "users",
				"localField":   "invited_user",
				"foreignField": "email",
				"as":           "user",
			},
		},
		{"$unwind": "$user"},
		// Joindre avec l'utilisateur qui a invité
		{
			"$lookup": bson.M{
				"from":         "users",
				"localField":   "invited_by",
				"foreignField": "email",
				"as":           "inviter",
			},
		},
		{"$unwind": "$inviter"},
		{"$sort": bson.M{"invited_at": -1}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la récupération des invitations: %w", err)
	}
	defer cursor.Close(ctx)

	var invitations []models.PendingInvitationWithUser
	for cursor.Next(ctx) {
		var result struct {
			ID        primitive.ObjectID `bson:"_id"`
			Status    string             `bson:"status"`
			InvitedAt time.Time          `bson:"invited_at"`
			User      models.User        `bson:"user"`
			Inviter   models.User        `bson:"inviter"`
		}
		if err := cursor.Decode(&result); err != nil {
			continue
		}

		invitations = append(invitations, models.PendingInvitationWithUser{
			ID: result.ID,
			User: models.UserBasicInfo{
				ID:        result.User.Email,
				Firstname: result.User.Firstname,
				Lastname:  result.User.Lastname,
				Email:     result.User.Email,
			},
			InvitedBy: models.UserBasicInfo{
				ID:        result.Inviter.Email,
				Firstname: result.Inviter.Firstname,
				Lastname:  result.Inviter.Lastname,
			},
			Status:    result.Status,
			InvitedAt: result.InvitedAt,
		})
	}

	return invitations, nil
}

// HasPendingInvitation vérifie si un utilisateur a déjà une invitation en attente
func (r *ChatGroupInvitationRepository) HasPendingInvitation(groupID primitive.ObjectID, userID string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	count, err := r.collection.CountDocuments(ctx, bson.M{
		"group_id":     groupID,
		"invited_user": userID,
		"status":       "pending",
	})
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// UpdateStatus met à jour le statut d'une invitation
func (r *ChatGroupInvitationRepository) UpdateStatus(invitationID primitive.ObjectID, status string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now()
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": invitationID},
		bson.M{"$set": bson.M{
			"status":       status,
			"responded_at": now,
		}},
	)
	if err != nil {
		return fmt.Errorf("erreur lors de la mise à jour de l'invitation: %w", err)
	}

	return nil
}

// Delete supprime une invitation
func (r *ChatGroupInvitationRepository) Delete(invitationID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": invitationID})
	if err != nil {
		return fmt.Errorf("erreur lors de la suppression de l'invitation: %w", err)
	}

	return nil
}
