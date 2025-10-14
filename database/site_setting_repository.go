package database

import (
	"context"
	"time"

	"premier-an-backend/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// SiteSettingRepository gère les opérations sur les paramètres du site
type SiteSettingRepository struct {
	collection *mongo.Collection
}

// NewSiteSettingRepository crée un nouveau repository pour les paramètres du site
func NewSiteSettingRepository(db *mongo.Database) *SiteSettingRepository {
	return &SiteSettingRepository{
		collection: db.Collection("site_settings"),
	}
}

// GetGlobalTheme récupère le thème global du site
func (r *SiteSettingRepository) GetGlobalTheme(ctx context.Context) (string, error) {
	var setting models.SiteSetting
	
	err := r.collection.FindOne(ctx, bson.M{"key": "global_theme"}).Decode(&setting)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Créer le thème par défaut si n'existe pas
			defaultTheme := models.SiteSetting{
				Key:       "global_theme",
				Value:     "medieval",
				UpdatedAt: time.Now(),
				UpdatedBy: nil,
			}
			
			_, err = r.collection.InsertOne(ctx, defaultTheme)
			if err != nil {
				return "", err
			}
			
			return "medieval", nil
		}
		return "", err
	}
	
	return setting.Value, nil
}

// SetGlobalTheme définit le thème global du site
func (r *SiteSettingRepository) SetGlobalTheme(ctx context.Context, theme string, updatedBy *primitive.ObjectID) error {
	filter := bson.M{"key": "global_theme"}
	update := bson.M{
		"$set": bson.M{
			"value":      theme,
			"updated_at": time.Now(),
			"updated_by": updatedBy,
		},
	}
	
	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

// GetAllSettings récupère tous les paramètres du site (pour admin)
func (r *SiteSettingRepository) GetAllSettings(ctx context.Context) ([]models.SiteSetting, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	
	var settings []models.SiteSetting
	if err = cursor.All(ctx, &settings); err != nil {
		return nil, err
	}
	
	return settings, nil
}
