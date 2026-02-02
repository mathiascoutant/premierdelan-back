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

// MediaRepository gère les opérations sur les médias
type MediaRepository struct {
	collection *mongo.Collection
}

// NewMediaRepository crée une nouvelle instance de MediaRepository
func NewMediaRepository(db *mongo.Database) *MediaRepository {
	return &MediaRepository{
		collection: db.Collection("medias"),
	}
}

// Create crée un nouveau média
func (r *MediaRepository) Create(media *models.Media) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	media.ID = primitive.NewObjectID()
	media.UploadedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, media)
	if err != nil {
		return fmt.Errorf("erreur lors de la création du média: %w", err)
	}

	return nil
}

// FindByEvent retourne tous les médias d'un événement
func (r *MediaRepository) FindByEvent(eventID primitive.ObjectID) ([]models.Media, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Trier par date de création décroissante (plus récent en premier)
	opts := options.Find().SetSort(bson.D{{Key: "uploaded_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, bson.M{"event_id": eventID}, opts)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la recherche des médias: %w", err)
	}
	defer cursor.Close(ctx)

	var medias []models.Media
	if err = cursor.All(ctx, &medias); err != nil {
		return nil, fmt.Errorf("erreur lors du décodage des médias: %w", err)
	}

	return medias, nil
}

// FindByID recherche un média par ID
func (r *MediaRepository) FindByID(id primitive.ObjectID) (*models.Media, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var media models.Media
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&media)

	if err == mongo.ErrNoDocuments {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("erreur lors de la recherche du média: %w", err)
	}

	return &media, nil
}

// Delete supprime un média
func (r *MediaRepository) Delete(id primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("erreur lors de la suppression du média: %w", err)
	}

	return nil
}

// CountByEvent compte le nombre de médias pour un événement
func (r *MediaRepository) CountByEvent(eventID primitive.ObjectID) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	count, err := r.collection.CountDocuments(ctx, bson.M{"event_id": eventID})
	if err != nil {
		return 0, fmt.Errorf("erreur lors du comptage des médias: %w", err)
	}

	return count, nil
}

// CountByEventAndType compte les médias par type
func (r *MediaRepository) CountByEventAndType(eventID primitive.ObjectID, mediaType string) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	count, err := r.collection.CountDocuments(ctx, bson.M{
		"event_id": eventID,
		"type":     mediaType,
	})
	if err != nil {
		return 0, fmt.Errorf("erreur lors du comptage des médias par type: %w", err)
	}

	return count, nil
}
