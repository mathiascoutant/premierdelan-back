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

// UserRepository gère les opérations sur les utilisateurs
type UserRepository struct {
	collection *mongo.Collection
}

// NewUserRepository crée une nouvelle instance de UserRepository
func NewUserRepository(db *mongo.Database) *UserRepository {
	return &UserRepository{
		collection: db.Collection("users"),
	}
}

// Create crée un nouvel utilisateur
func (r *UserRepository) Create(user *models.User) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	user.CreatedAt = time.Now()
	user.ID = primitive.NewObjectID()

	_, err := r.collection.InsertOne(ctx, user)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("cet email est déjà utilisé")
		}
		return fmt.Errorf("erreur lors de la création de l'utilisateur: %w", err)
	}

	return nil
}

// FindByEmail recherche un utilisateur par email
func (r *UserRepository) FindByEmail(email string) (*models.User, error) {
	// Augmenter le timeout à 10 secondes pour éviter les blocages
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var user models.User
	err := r.collection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("erreur lors de la recherche de l'utilisateur: %w", err)
	}

	return &user, nil
}

// FindByID recherche un utilisateur par ID
func (r *UserRepository) FindByID(id primitive.ObjectID) (*models.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user models.User
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&user)
	
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("erreur lors de la recherche de l'utilisateur: %w", err)
	}

	return &user, nil
}

// EmailExists vérifie si un email existe déjà
func (r *UserRepository) EmailExists(email string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	count, err := r.collection.CountDocuments(ctx, bson.M{"email": email})
	if err != nil {
		return false, fmt.Errorf("erreur lors de la vérification de l'email: %w", err)
	}

	return count > 0, nil
}

// Update met à jour un utilisateur
func (r *UserRepository) Update(user *models.User) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"_id": user.ID}
	update := bson.M{
		"$set": bson.M{
			"firstname": user.Firstname,
			"lastname":  user.Lastname,
			"email":     user.Email,
			"phone":     user.Phone,
			"password":  user.Password,
		},
	}

	_, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("erreur lors de la mise à jour de l'utilisateur: %w", err)
	}

	return nil
}

// Delete supprime un utilisateur
func (r *UserRepository) Delete(id primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("erreur lors de la suppression de l'utilisateur: %w", err)
	}

	return nil
}

// FindAdmins retourne tous les utilisateurs admins
func (r *UserRepository) FindAdmins() ([]models.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var admins []models.User
	cursor, err := r.collection.Find(ctx, bson.M{"admin": 1})
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la recherche des admins: %w", err)
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &admins); err != nil {
		return nil, fmt.Errorf("erreur lors du décodage des admins: %w", err)
	}

	return admins, nil
}

// UpdateLastSeen met à jour le timestamp de dernière activité d'un utilisateur
func (r *UserRepository) UpdateLastSeen(userID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now()
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": userID},
		bson.M{"$set": bson.M{"last_seen": now}},
	)
	
	if err != nil {
		return fmt.Errorf("erreur mise à jour last_seen: %w", err)
	}

	return nil
}

// UpdateByEmail met à jour un utilisateur par email
func (r *UserRepository) UpdateByEmail(email string, updateData map[string]interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := r.collection.UpdateOne(
		ctx,
		bson.M{"email": email},
		bson.M{"$set": updateData},
	)

	if err != nil {
		return fmt.Errorf("erreur mise à jour utilisateur: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("utilisateur non trouvé")
	}

	return nil
}
