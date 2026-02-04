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

// FCMTokenRepository gère les opérations sur les tokens FCM
type FCMTokenRepository struct {
	collection *mongo.Collection
}

// NewFCMTokenRepository crée une nouvelle instance de FCMTokenRepository
func NewFCMTokenRepository(db *mongo.Database) *FCMTokenRepository {
	return &FCMTokenRepository{
		collection: db.Collection("fcm_tokens"),
	}
}

// Create ou update un token FCM
func (r *FCMTokenRepository) Upsert(token *models.FCMToken) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Vérifier si le token existe déjà
	existing, err := r.FindByToken(token.Token)
	if err != nil {
		return err
	}

	if existing != nil {
		// Update
		token.ID = existing.ID
		token.CreatedAt = existing.CreatedAt
		token.UpdatedAt = time.Now()

		filter := bson.M{"_id": existing.ID}
		update := bson.M{"$set": token}

		_, err := r.collection.UpdateOne(ctx, filter, update)
		return err
	}

	// Create
	token.ID = primitive.NewObjectID()
	token.CreatedAt = time.Now()
	token.UpdatedAt = time.Now()

	_, err = r.collection.InsertOne(ctx, token)
	if err != nil {
		return fmt.Errorf("erreur lors de la création du token FCM: %w", err)
	}

	return nil
}

// FindByUserID recherche tous les tokens d'un utilisateur
func (r *FCMTokenRepository) FindByUserID(userID string) ([]models.FCMToken, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var tokens []models.FCMToken
	cursor, err := r.collection.Find(ctx, bson.M{"user_id": userID})
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la recherche des tokens: %w", err)
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &tokens); err != nil {
		return nil, fmt.Errorf("erreur lors du décodage des tokens: %w", err)
	}

	return tokens, nil
}

// FindAll retourne tous les tokens FCM
func (r *FCMTokenRepository) FindAll() ([]models.FCMToken, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var tokens []models.FCMToken
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la recherche des tokens: %w", err)
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &tokens); err != nil {
		return nil, fmt.Errorf("erreur lors du décodage des tokens: %w", err)
	}

	return tokens, nil
}

// FindByToken recherche un token spécifique
func (r *FCMTokenRepository) FindByToken(token string) (*models.FCMToken, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var fcmToken models.FCMToken
	err := r.collection.FindOne(ctx, bson.M{"token": token}).Decode(&fcmToken)

	if err == mongo.ErrNoDocuments {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("erreur lors de la recherche du token: %w", err)
	}

	return &fcmToken, nil
}

// Delete supprime un token
func (r *FCMTokenRepository) Delete(token string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.collection.DeleteOne(ctx, bson.M{"token": token})
	if err != nil {
		return fmt.Errorf("erreur lors de la suppression du token: %w", err)
	}

	return nil
}

// DeleteByUserID supprime tous les tokens d'un utilisateur
func (r *FCMTokenRepository) DeleteByUserID(userID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.collection.DeleteMany(ctx, bson.M{"user_id": userID})
	if err != nil {
		return fmt.Errorf("erreur lors de la suppression des tokens: %w", err)
	}

	return nil
}
