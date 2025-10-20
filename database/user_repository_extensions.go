package database

import (
	"context"
	"fmt"
	"premier-an-backend/models"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// FindAll retourne tous les utilisateurs
func (r *UserRepository) FindAll() ([]models.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la recherche des utilisateurs: %w", err)
	}
	defer cursor.Close(ctx)

	var users []models.User
	if err = cursor.All(ctx, &users); err != nil {
		return nil, fmt.Errorf("erreur lors du décodage des utilisateurs: %w", err)
	}

	return users, nil
}

// UpdateFields met à jour des champs spécifiques d'un utilisateur
func (r *UserRepository) UpdateFields(id primitive.ObjectID, fields bson.M) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": fields},
	)

	if err != nil {
		return fmt.Errorf("erreur lors de la mise à jour: %w", err)
	}

	return nil
}

// CountAll compte tous les utilisateurs
func (r *UserRepository) CountAll() (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	count, err := r.collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return 0, fmt.Errorf("erreur lors du comptage: %w", err)
	}

	return count, nil
}

// CountAdmins compte les administrateurs
func (r *UserRepository) CountAdmins() (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	count, err := r.collection.CountDocuments(ctx, bson.M{"admin": 1})
	if err != nil {
		return 0, fmt.Errorf("erreur lors du comptage des admins: %w", err)
	}

	return count, nil
}

// SearchUsers recherche des utilisateurs par nom, prénom ou email
func (r *UserRepository) SearchUsers(query string, limit int, excludeUserID string) ([]models.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Créer un filtre de recherche avec regex (insensible à la casse)
	filter := bson.M{
		"$and": []bson.M{
			{
				"$or": []bson.M{
					{"firstname": bson.M{"$regex": query, "$options": "i"}},
					{"lastname": bson.M{"$regex": query, "$options": "i"}},
					{"email": bson.M{"$regex": query, "$options": "i"}},
				},
			},
			// Exclure l'utilisateur courant
			{"email": bson.M{"$ne": excludeUserID}},
		},
	}

	opts := options.Find().
		SetLimit(int64(limit)).
		SetSort(bson.D{{Key: "firstname", Value: 1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la recherche: %w", err)
	}
	defer cursor.Close(ctx)

	var users []models.User
	if err = cursor.All(ctx, &users); err != nil {
		return nil, fmt.Errorf("erreur lors du décodage: %w", err)
	}

	return users, nil
}
