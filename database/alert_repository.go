package database

import (
	"context"
	"fmt"
	"premier-an-backend/models"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// AlertRepository gère les opérations sur les alertes critiques
type AlertRepository struct {
	collection *mongo.Collection
}

// NewAlertRepository crée une nouvelle instance
func NewAlertRepository(db *mongo.Database) *AlertRepository {
	return &AlertRepository{
		collection: db.Collection("admin_alerts"),
	}
}

// Create crée une nouvelle alerte
func (r *AlertRepository) Create(alert *models.CriticalAlert) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	alert.ID = primitive.NewObjectID()
	alert.CreatedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, alert)
	if err != nil {
		return fmt.Errorf("erreur lors de la création de l'alerte: %w", err)
	}

	return nil
}

// CountRecentByIP compte le nombre d'alertes récentes depuis une IP (pour rate limiting)
// Note: On stocke l'IP dans user_agent pour simplifier
func (r *AlertRepository) CountRecentAlerts(adminEmail string, sinceMinutes int) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	since := time.Now().Add(-time.Duration(sinceMinutes) * time.Minute)

	count, err := r.collection.CountDocuments(ctx, map[string]interface{}{
		"admin_email": adminEmail,
		"created_at":  map[string]interface{}{"$gte": since},
	})

	if err != nil {
		return 0, fmt.Errorf("erreur lors du comptage des alertes: %w", err)
	}

	return count, nil
}
