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
	CloudinaryCloudName     string
	CloudinaryUploadPreset  string
	CloudinaryVideoPreset   string
	CloudinaryPreviewPreset string
	CloudinaryAPIKey        string
	CloudinaryAPISecret     string
	SlackWebhookURL         string
}

// Load charge la configuration depuis les variables d'environnement
func Load() (*Config, error) {
	// Charger le fichier .env s'il existe
	_ = godotenv.Load()

	config := &Config{
		Port:                    getEnv("PORT", "8090"),
		Host:                    getEnv("HOST", "0.0.0.0"), // 0.0.0.0 pour serveur cloud
		MongoURI:                getEnv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDB:                 getEnv("MONGO_DB", "premier_an_db"),
		JWTSecret:               getEnv("JWT_SECRET", ""),
		Environment:             getEnv("ENVIRONMENT", "development"),
		VAPIDPublicKey:          getEnv("VAPID_PUBLIC_KEY", ""),
		VAPIDPrivateKey:         getEnv("VAPID_PRIVATE_KEY", ""),
		VAPIDSubject:            getEnv("VAPID_SUBJECT", "mailto:contact@example.com"),
		FirebaseCredentialsFile: getEnv("FIREBASE_CREDENTIALS_FILE", "firebase-service-account.json"),
		FCMVAPIDKey:             getEnv("FCM_VAPID_KEY", ""),
		CloudinaryCloudName:     getEnv("CLOUDINARY_CLOUD_NAME", ""),
		CloudinaryUploadPreset:  getEnv("CLOUDINARY_UPLOAD_PRESET", "premierdelan_profiles"),
		CloudinaryVideoPreset:   getEnv("CLOUDINARY_VIDEO_PRESET", "premierdelan_trailers"),
		CloudinaryPreviewPreset: getEnv("CLOUDINARY_PREVIEW_PRESET", "premierdelan_gallery_preview"),
		CloudinaryAPIKey:        getEnv("CLOUDINARY_API_KEY", ""),
		CloudinaryAPISecret:     getEnv("CLOUDINARY_API_SECRET", ""),
		SlackWebhookURL:         getEnv("SLACK_WEBHOOK_URL", ""),
	}

	// Parser les origines CORS
	origins := getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000")
	originsList := strings.Split(origins, ",")
	// Nettoyer les espaces autour de chaque origine
	config.CORSOrigins = make([]string, 0, len(originsList))
	for _, origin := range originsList {
		trimmed := strings.TrimSpace(origin)
		if trimmed != "" {
			config.CORSOrigins = append(config.CORSOrigins, trimmed)
		}
	}

	// Valider les configurations critiques
	if config.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET est requis")
	}

	// Les clés VAPID sont optionnelles (on utilise FCM maintenant)
	// Pas besoin de les valider en production

	return config, nil
}

// getEnv récupère une variable d'environnement avec une valeur par défaut
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
