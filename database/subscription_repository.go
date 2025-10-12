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

// SubscriptionRepository gère les opérations sur les abonnements push
type SubscriptionRepository struct {
	collection *mongo.Collection
}

// NewSubscriptionRepository crée une nouvelle instance de SubscriptionRepository
func NewSubscriptionRepository(db *mongo.Database) *SubscriptionRepository {
	return &SubscriptionRepository{
		collection: db.Collection("subscriptions"),
	}
}

// Create crée un nouvel abonnement
func (r *SubscriptionRepository) Create(subscription *models.PushSubscription) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	subscription.ID = primitive.NewObjectID()
	subscription.Created = time.Now()

	_, err := r.collection.InsertOne(ctx, subscription)
	if err != nil {
		return fmt.Errorf("erreur lors de la création de l'abonnement: %w", err)
	}

	return nil
}

// FindByUserID recherche tous les abonnements d'un utilisateur
func (r *SubscriptionRepository) FindByUserID(userID string) ([]models.PushSubscription, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var subscriptions []models.PushSubscription
	cursor, err := r.collection.Find(ctx, bson.M{"user_id": userID})
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la recherche des abonnements: %w", err)
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &subscriptions); err != nil {
		return nil, fmt.Errorf("erreur lors du décodage des abonnements: %w", err)
	}

	return subscriptions, nil
}

// FindAll retourne tous les abonnements
func (r *SubscriptionRepository) FindAll() ([]models.PushSubscription, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var subscriptions []models.PushSubscription
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la recherche des abonnements: %w", err)
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &subscriptions); err != nil {
		return nil, fmt.Errorf("erreur lors du décodage des abonnements: %w", err)
	}

	return subscriptions, nil
}

// FindByEndpoint recherche un abonnement par endpoint
func (r *SubscriptionRepository) FindByEndpoint(endpoint string) (*models.PushSubscription, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var subscription models.PushSubscription
	err := r.collection.FindOne(ctx, bson.M{"endpoint": endpoint}).Decode(&subscription)
	
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("erreur lors de la recherche de l'abonnement: %w", err)
	}

	return &subscription, nil
}

// Delete supprime un abonnement par endpoint
func (r *SubscriptionRepository) Delete(endpoint string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.collection.DeleteOne(ctx, bson.M{"endpoint": endpoint})
	if err != nil {
		return fmt.Errorf("erreur lors de la suppression de l'abonnement: %w", err)
	}

	return nil
}

// DeleteByUserID supprime tous les abonnements d'un utilisateur
func (r *SubscriptionRepository) DeleteByUserID(userID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.collection.DeleteMany(ctx, bson.M{"user_id": userID})
	if err != nil {
		return fmt.Errorf("erreur lors de la suppression des abonnements: %w", err)
	}

	return nil
}

