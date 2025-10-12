package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config contient toutes les configurations de l'application
type Config struct {
	Port                    string
	Host                    string
	MongoURI                string
	MongoDB                 string
	JWTSecret               string
	Environment             string
	CORSOrigins             []string
	VAPIDPublicKey          string
	VAPIDPrivateKey         string
	VAPIDSubject            string
	FirebaseCredentialsFile string
	FCMVAPIDKey             string
}

// Load charge la configuration depuis les variables d'environnement
func Load() (*Config, error) {
	// Charger le fichier .env s'il existe
	_ = godotenv.Load()

	config := &Config{
		Port:                    getEnv("PORT", "8090"),
		Host:                    getEnv("HOST", "0.0.0.0"), // 0.0.0.0 pour Railway/Cloud
		MongoURI:                getEnv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDB:                 getEnv("MONGO_DB", "premier_an_db"),
		JWTSecret:               getEnv("JWT_SECRET", ""),
		Environment:             getEnv("ENVIRONMENT", "development"),
		VAPIDPublicKey:          getEnv("VAPID_PUBLIC_KEY", ""),
		VAPIDPrivateKey:         getEnv("VAPID_PRIVATE_KEY", ""),
		VAPIDSubject:            getEnv("VAPID_SUBJECT", "mailto:contact@example.com"),
		FirebaseCredentialsFile: getEnv("FIREBASE_CREDENTIALS_FILE", "firebase-service-account.json"),
		FCMVAPIDKey:             getEnv("FCM_VAPID_KEY", ""),
	}

	// Parser les origines CORS
	origins := getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000")
	config.CORSOrigins = strings.Split(origins, ",")

	// Valider les configurations critiques
	if config.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET est requis")
	}

	// Les clés VAPID sont optionnelles en développement
	if config.Environment == "production" {
		if config.VAPIDPublicKey == "" || config.VAPIDPrivateKey == "" {
			return nil, fmt.Errorf("VAPID_PUBLIC_KEY et VAPID_PRIVATE_KEY sont requis en production")
		}
	}

	return config, nil
}

// getEnv récupère une variable d'environnement avec une valeur par défaut
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
