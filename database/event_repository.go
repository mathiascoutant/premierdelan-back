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

// EventRepository gère les opérations sur les événements
type EventRepository struct {
	collection *mongo.Collection
}

// NewEventRepository crée une nouvelle instance de EventRepository
func NewEventRepository(db *mongo.Database) *EventRepository {
	return &EventRepository{
		collection: db.Collection("events"),
	}
}

// Create crée un nouvel événement
func (r *EventRepository) Create(event *models.Event) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	event.ID = primitive.NewObjectID()
	event.CreatedAt = time.Now()
	event.UpdatedAt = time.Now()
	event.Inscrits = 0
	event.PhotosCount = 0

	if event.Statut == "" {
		event.Statut = "ouvert"
	}

	_, err := r.collection.InsertOne(ctx, event)
	if err != nil {
		return fmt.Errorf("erreur lors de la création de l'événement: %w", err)
	}

	return nil
}

// FindAll retourne tous les événements
func (r *EventRepository) FindAll() ([]models.Event, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := options.Find().SetSort(bson.D{{Key: "date", Value: -1}})
	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la recherche des événements: %w", err)
	}
	defer cursor.Close(ctx)

	var events []models.Event
	if err = cursor.All(ctx, &events); err != nil {
		return nil, fmt.Errorf("erreur lors du décodage des événements: %w", err)
	}

	return events, nil
}

// FindByID recherche un événement par ID
func (r *EventRepository) FindByID(id primitive.ObjectID) (*models.Event, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var event models.Event
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&event)
	
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("erreur lors de la recherche de l'événement: %w", err)
	}

	return &event, nil
}

// Update met à jour un événement
func (r *EventRepository) Update(id primitive.ObjectID, update bson.M) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update["updated_at"] = time.Now()

	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": update},
	)

	if err != nil {
		return fmt.Errorf("erreur lors de la mise à jour de l'événement: %w", err)
	}

	return nil
}

// Delete supprime un événement
func (r *EventRepository) Delete(id primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("erreur lors de la suppression de l'événement: %w", err)
	}

	return nil
}

// CountAll compte tous les événements
func (r *EventRepository) CountAll() (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	count, err := r.collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return 0, fmt.Errorf("erreur lors du comptage des événements: %w", err)
	}

	return count, nil
}

// CountByStatus compte les événements par statut
func (r *EventRepository) CountByStatus(status string) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	count, err := r.collection.CountDocuments(ctx, bson.M{"statut": status})
	if err != nil {
		return 0, fmt.Errorf("erreur lors du comptage des événements: %w", err)
	}

	return count, nil
}

// GetTotalInscrits calcule le total des inscrits sur tous les événements
func (r *EventRepository) GetTotalInscrits() (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pipeline := []bson.M{
		{"$group": bson.M{
			"_id":   nil,
			"total": bson.M{"$sum": "$inscrits"},
		}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, fmt.Errorf("erreur lors de l'agrégation: %w", err)
	}
	defer cursor.Close(ctx)

	var result []bson.M
	if err = cursor.All(ctx, &result); err != nil {
		return 0, err
	}

	if len(result) == 0 {
		return 0, nil
	}

	total, ok := result[0]["total"].(int32)
	if !ok {
		return 0, nil
	}

	return int(total), nil
}

// FindEventsToNotifyOpening trouve les événements dont l'ouverture vient d'être atteinte
func (r *EventRepository) FindEventsToNotifyOpening() ([]models.Event, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	now := time.Now()
	twoMinutesAgo := now.Add(-2 * time.Minute)

	// Chercher les événements où :
	// - date_ouverture_inscription est entre maintenant et il y a 2 minutes
	// - notification_sent_opening est false ou null
	filter := bson.M{
		"date_ouverture_inscription": bson.M{
			"$lte": now,
			"$gte": twoMinutesAgo,
		},
		"$or": []bson.M{
			{"notification_sent_opening": false},
			{"notification_sent_opening": bson.M{"$exists": false}},
		},
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la recherche des événements: %w", err)
	}
	defer cursor.Close(ctx)

	var events []models.Event
	if err = cursor.All(ctx, &events); err != nil {
		return nil, fmt.Errorf("erreur lors du décodage des événements: %w", err)
	}

	return events, nil
}

// FindByCodeSoiree recherche un événement par code soirée
func (r *EventRepository) FindByCodeSoiree(codeSoiree string) (*models.Event, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var event models.Event
	err := r.collection.FindOne(ctx, bson.M{"code_soiree": codeSoiree}).Decode(&event)

	if err == mongo.ErrNoDocuments {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("erreur lors de la recherche de l'événement: %w", err)
	}

	return &event, nil
}

// GetTotalPhotos calcule le total des photos sur tous les événements
func (r *EventRepository) GetTotalPhotos() (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pipeline := []bson.M{
		{"$group": bson.M{
			"_id":   nil,
			"total": bson.M{"$sum": "$photos_count"},
		}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, fmt.Errorf("erreur lors de l'agrégation: %w", err)
	}
	defer cursor.Close(ctx)

	var result []bson.M
	if err = cursor.All(ctx, &result); err != nil {
		return 0, err
	}

	if len(result) == 0 {
		return 0, nil
	}

	total, ok := result[0]["total"].(int32)
	if !ok {
		return 0, nil
	}

	return int(total), nil
}

