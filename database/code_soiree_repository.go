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

// CodeSoireeRepository gère les opérations sur les codes de soirée
// Les codes sont stockés dans la collection "events" (champ code_soiree)
type CodeSoireeRepository struct {
	eventCollection *mongo.Collection
	userCollection  *mongo.Collection
}

// NewCodeSoireeRepository crée une nouvelle instance
func NewCodeSoireeRepository(db *mongo.Database) *CodeSoireeRepository {
	return &CodeSoireeRepository{
		eventCollection: db.Collection("events"),
		userCollection:  db.Collection("users"),
	}
}

// IsCodeValid vérifie si un code existe dans un événement (pas annulé)
func (r *CodeSoireeRepository) IsCodeValid(code string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Chercher un événement avec ce code_soiree qui n'est pas annulé
	filter := bson.M{
		"code_soiree": code,
		"statut":      bson.M{"$ne": "annule"}, // Pas annulé
	}

	count, err := r.eventCollection.CountDocuments(ctx, filter)

	if err != nil {
		return false, fmt.Errorf("erreur lors de la vérification du code: %w", err)
	}

	// Debug log
	fmt.Printf("🔍 IsCodeValid('%s'): count=%d, valid=%v\n", code, count, count > 0)

	return count > 0, nil
}

// IncrementUsage ne fait rien (compteur géré par nb d'utilisateurs dans DB)
func (r *CodeSoireeRepository) IncrementUsage(code string) error {
	// Pas besoin d'incrémenter - le compteur est le nombre d'utilisateurs avec ce code dans la DB
	return nil
}

// FindAll retourne tous les codes de soirée depuis les événements
func (r *CodeSoireeRepository) FindAll() ([]models.CodeSoiree, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Récupérer tous les code_soiree uniques depuis events
	pipeline := []bson.M{
		{"$match": bson.M{"code_soiree": bson.M{"$ne": ""}}}, // Ignorer les événements sans code
		{"$group": bson.M{
			"_id":        "$code_soiree",
			"created_at": bson.M{"$first": "$created_at"},
			"statut":     bson.M{"$first": "$statut"},
		}},
		{"$sort": bson.M{"created_at": -1}},
	}

	cursor, err := r.eventCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la recherche des codes: %w", err)
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err = cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("erreur lors du décodage: %w", err)
	}

	// Construire la liste des codes avec le compteur d'utilisations
	var codes []models.CodeSoiree
	for _, result := range results {
		code := result["_id"].(string)
		createdAt := result["created_at"].(primitive.DateTime).Time()
		statut := result["statut"].(string)

		// Compter le nombre d'utilisateurs avec ce code
		utilisations, _ := r.userCollection.CountDocuments(ctx, bson.M{"code_soiree": code})

		codes = append(codes, models.CodeSoiree{
			Code:         code,
			Utilisations: int(utilisations),
			CreatedAt:    createdAt,
			Active:       statut != "annule", // Actif si l'événement n'est pas annulé
		})
	}

	return codes, nil
}

// FindCurrent retourne le code de soirée le plus récent
func (r *CodeSoireeRepository) FindCurrent() (*models.CodeSoiree, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Récupérer l'événement le plus récent avec un code_soiree
	pipeline := []bson.M{
		{"$match": bson.M{
			"code_soiree": bson.M{"$ne": ""},
			"statut":      bson.M{"$ne": "annule"},
		}},
		{"$sort": bson.M{"created_at": -1}},
		{"$limit": 1},
	}

	cursor, err := r.eventCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la recherche du code actuel: %w", err)
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err = cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("erreur lors du décodage: %w", err)
	}

	if len(results) == 0 {
		return nil, nil // Aucun code trouvé
	}

	result := results[0]
	code := result["code_soiree"].(string)
	createdAt := result["created_at"].(primitive.DateTime).Time()

	// Compter les utilisations
	utilisations, _ := r.userCollection.CountDocuments(ctx, bson.M{"code_soiree": code})

	return &models.CodeSoiree{
		Code:         code,
		Utilisations: int(utilisations),
		CreatedAt:    createdAt,
		Active:       true,
	}, nil
}

// Create crée un nouveau code en créant un nouvel événement (non utilisé - codes créés avec events)
func (r *CodeSoireeRepository) Create(code *models.CodeSoiree) error {
	// Les codes sont créés automatiquement quand on crée un événement
	// Cette méthode n'est plus utilisée
	return nil
}
