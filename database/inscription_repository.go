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

// InscriptionRepository gère les opérations sur les inscriptions
type InscriptionRepository struct {
	collection *mongo.Collection
}

// NewInscriptionRepository crée une nouvelle instance de InscriptionRepository
func NewInscriptionRepository(db *mongo.Database) *InscriptionRepository {
	return &InscriptionRepository{
		collection: db.Collection("inscriptions"),
	}
}

// Create crée une nouvelle inscription
func (r *InscriptionRepository) Create(inscription *models.Inscription) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	inscription.ID = primitive.NewObjectID()
	inscription.CreatedAt = time.Now()
	inscription.UpdatedAt = time.Now()

	if inscription.Accompagnants == nil {
		inscription.Accompagnants = []models.Accompagnant{}
	}

	_, err := r.collection.InsertOne(ctx, inscription)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("vous êtes déjà inscrit à cet événement")
		}
		return fmt.Errorf("erreur lors de la création de l'inscription: %w", err)
	}

	return nil
}

// FindByEventAndUser recherche une inscription par événement et utilisateur
func (r *InscriptionRepository) FindByEventAndUser(eventID primitive.ObjectID, userEmail string) (*models.Inscription, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var inscription models.Inscription
	err := r.collection.FindOne(ctx, bson.M{
		"event_id":   eventID,
		"user_email": userEmail,
	}).Decode(&inscription)

	if err == mongo.ErrNoDocuments {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("erreur lors de la recherche de l'inscription: %w", err)
	}

	return &inscription, nil
}

// FindByEvent retourne toutes les inscriptions pour un événement
func (r *InscriptionRepository) FindByEvent(eventID primitive.ObjectID) ([]models.Inscription, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := r.collection.Find(ctx, bson.M{"event_id": eventID})
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la recherche des inscriptions: %w", err)
	}
	defer cursor.Close(ctx)

	var inscriptions []models.Inscription
	if err = cursor.All(ctx, &inscriptions); err != nil {
		return nil, fmt.Errorf("erreur lors du décodage des inscriptions: %w", err)
	}

	return inscriptions, nil
}

// Update met à jour une inscription
func (r *InscriptionRepository) Update(inscription *models.Inscription) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	inscription.UpdatedAt = time.Now()

	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": inscription.ID},
		bson.M{"$set": bson.M{
			"nombre_personnes": inscription.NombrePersonnes,
			"accompagnants":    inscription.Accompagnants,
			"updated_at":       inscription.UpdatedAt,
		}},
	)

	if err != nil {
		return fmt.Errorf("erreur lors de la mise à jour de l'inscription: %w", err)
	}

	return nil
}

// Delete supprime une inscription par event_id et user_email
func (r *InscriptionRepository) Delete(eventID primitive.ObjectID, userEmail string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.collection.DeleteOne(ctx, bson.M{
		"event_id":   eventID,
		"user_email": userEmail,
	})
	if err != nil {
		return fmt.Errorf("erreur lors de la suppression de l'inscription: %w", err)
	}

	return nil
}

// DeleteByID supprime une inscription par son ID
func (r *InscriptionRepository) DeleteByID(id primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("erreur lors de la suppression de l'inscription: %w", err)
	}

	return nil
}

// FindByID recherche une inscription par ID
func (r *InscriptionRepository) FindByID(id primitive.ObjectID) (*models.Inscription, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var inscription models.Inscription
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&inscription)

	if err == mongo.ErrNoDocuments {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("erreur lors de la recherche de l'inscription: %w", err)
	}

	return &inscription, nil
}

// CountByEvent compte le nombre d'inscriptions pour un événement
func (r *InscriptionRepository) CountByEvent(eventID primitive.ObjectID) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	count, err := r.collection.CountDocuments(ctx, bson.M{"event_id": eventID})
	if err != nil {
		return 0, fmt.Errorf("erreur lors du comptage des inscriptions: %w", err)
	}

	return count, nil
}

// GetTotalPersonnesByEvent calcule le nombre total de personnes inscrites à un événement
func (r *InscriptionRepository) GetTotalPersonnesByEvent(eventID primitive.ObjectID) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pipeline := []bson.M{
		{"$match": bson.M{"event_id": eventID}},
		{"$group": bson.M{
			"_id":   nil,
			"total": bson.M{"$sum": "$nombre_personnes"},
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

// FindByUser retourne toutes les inscriptions d'un utilisateur
func (r *InscriptionRepository) FindByUser(userEmail string) ([]models.Inscription, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := r.collection.Find(ctx, bson.M{"user_email": userEmail})
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la recherche des inscriptions de l'utilisateur: %w", err)
	}
	defer cursor.Close(ctx)

	var inscriptions []models.Inscription
	if err = cursor.All(ctx, &inscriptions); err != nil {
		return nil, fmt.Errorf("erreur lors du décodage des inscriptions: %w", err)
	}

	return inscriptions, nil
}

// FindByEventID retourne toutes les inscriptions d'un événement
func (r *InscriptionRepository) FindByEventID(eventID primitive.ObjectID) ([]models.Inscription, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := r.collection.Find(ctx, bson.M{"event_id": eventID})
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la recherche des inscriptions de l'événement: %w", err)
	}
	defer cursor.Close(ctx)

	var inscriptions []models.Inscription
	if err = cursor.All(ctx, &inscriptions); err != nil {
		return nil, fmt.Errorf("erreur lors du décodage des inscriptions: %w", err)
	}

	return inscriptions, nil
}

