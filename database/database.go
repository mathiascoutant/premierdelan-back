package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// DB est l'instance de connexion à la base de données MongoDB
var DB *mongo.Database
var Client *mongo.Client

// Connect établit la connexion à la base de données MongoDB
func Connect(uri, dbName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Options de connexion
	clientOptions := options.Client().ApplyURI(uri)

	// Connexion à MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return fmt.Errorf("erreur lors de la connexion à MongoDB: %w", err)
	}

	// Vérifier la connexion
	if err = client.Ping(ctx, nil); err != nil {
		return fmt.Errorf("erreur lors du ping MongoDB: %w", err)
	}

	Client = client
	DB = client.Database(dbName)

	log.Println("✓ Connexion à MongoDB établie")

	// Créer les index
	if err = createIndexes(); err != nil {
		return fmt.Errorf("erreur lors de la création des index: %w", err)
	}

	return nil
}

// Ping vérifie que la connexion MongoDB est active
func Ping() error {
	if Client == nil {
		return fmt.Errorf("client MongoDB non initialisé")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return Client.Ping(ctx, nil)
}

// Close ferme la connexion à la base de données
func Close() error {
	if Client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return Client.Disconnect(ctx)
	}
	return nil
}

// createIndexes crée les index nécessaires
func createIndexes() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Collection users
	usersCollection := DB.Collection("users")

	// Index unique sur l'email
	emailIndex := mongo.IndexModel{
		Keys:    map[string]interface{}{"email": 1},
		Options: options.Index().SetUnique(true),
	}

	_, err := usersCollection.Indexes().CreateOne(ctx, emailIndex)
	if err != nil {
		return fmt.Errorf("erreur lors de la création de l'index email: %w", err)
	}

	log.Println("✓ Index MongoDB créés")
	return nil
}
