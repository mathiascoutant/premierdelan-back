package services

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

// FCMService gère l'envoi des notifications via Firebase Cloud Messaging
type FCMService struct {
	client *messaging.Client
}

// NewFCMService crée une nouvelle instance de FCMService
func NewFCMService(credentialsFile string) (*FCMService, error) {
	ctx := context.Background()

	var app *firebase.App
	var err error

	// Vérifier si FIREBASE_CREDENTIALS_JSON existe (pour Railway/Cloud)
	credentialsJSON := os.Getenv("FIREBASE_CREDENTIALS_JSON")
	
	if credentialsJSON != "" {
		// Lire depuis la variable d'environnement
		log.Println("📦 Utilisation des credentials Firebase depuis FIREBASE_CREDENTIALS_JSON")
		opt := option.WithCredentialsJSON([]byte(credentialsJSON))
		app, err = firebase.NewApp(ctx, nil, opt)
	} else {
		// Lire depuis le fichier (développement local)
		log.Printf("📦 Utilisation des credentials Firebase depuis le fichier: %s", credentialsFile)
		opt := option.WithCredentialsFile(credentialsFile)
		app, err = firebase.NewApp(ctx, nil, opt)
	}

	if err != nil {
		return nil, fmt.Errorf("erreur lors de l'initialisation de Firebase: %w", err)
	}

	// Créer le client messaging
	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la création du client FCM: %w", err)
	}

	log.Println("✓ Firebase Cloud Messaging initialisé")

	return &FCMService{
		client: client,
	}, nil
}

// SendToToken envoie une notification à un token spécifique
func (s *FCMService) SendToToken(token string, title, body string, data map[string]string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Préparer UNIQUEMENT des data messages (pas de Notification pour éviter "from ...")
	if data == nil {
		data = make(map[string]string)
	}
	data["title"] = title
	data["message"] = body
	
	message := &messaging.Message{
		Token: token,
		Data:  data, // UNIQUEMENT data, pas de Notification
		Webpush: &messaging.WebpushConfig{
			Headers: map[string]string{
				"Urgency": "high",
			},
		},
	}

	response, err := s.client.Send(ctx, message)
	if err != nil {
		return fmt.Errorf("erreur lors de l'envoi de la notification: %w", err)
	}

	log.Printf("✓ Message envoyé avec succès: %s", response)
	return nil
}

// SendToMultipleTokens envoie une notification à plusieurs tokens
func (s *FCMService) SendToMultipleTokens(tokens []string, title, body string, data map[string]string) (success int, failed int, failedTokens []string, err error) {
	if len(tokens) == 0 {
		return 0, 0, nil, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Préparer UNIQUEMENT des data messages (pas de Notification pour éviter "from ...")
	if data == nil {
		data = make(map[string]string)
	}
	data["title"] = title
	data["message"] = body
	
	message := &messaging.MulticastMessage{
		Data: data, // UNIQUEMENT data, pas de Notification
		Webpush: &messaging.WebpushConfig{
			Headers: map[string]string{
				"Urgency": "high",
			},
		},
		Tokens: tokens,
	}

	response, err := s.client.SendEachForMulticast(ctx, message)
	if err != nil {
		return 0, 0, nil, fmt.Errorf("erreur lors de l'envoi multicast: %w", err)
	}

	// Collecter les tokens qui ont échoué
	failedTokens = make([]string, 0)
	for idx, resp := range response.Responses {
		if !resp.Success {
			failedTokens = append(failedTokens, tokens[idx])
			log.Printf("❌ Échec pour le token %s: %v", tokens[idx][:20]+"...", resp.Error)
		}
	}

	success = response.SuccessCount
	failed = response.FailureCount

	log.Printf("📊 Envoi multicast: %d succès, %d échecs sur %d total", success, failed, len(tokens))

	return success, failed, failedTokens, nil
}

// SendToAll envoie une notification à tous les tokens fournis
func (s *FCMService) SendToAll(tokens []string, title, body string, data map[string]string) (success int, failed int, failedTokens []string) {
	// FCM a une limite de 500 tokens par requête
	const batchSize = 500

	totalSuccess := 0
	totalFailed := 0
	allFailedTokens := make([]string, 0)

	for i := 0; i < len(tokens); i += batchSize {
		end := i + batchSize
		if end > len(tokens) {
			end = len(tokens)
		}

		batch := tokens[i:end]
		s, f, ft, err := s.SendToMultipleTokens(batch, title, body, data)
		
		if err != nil {
			log.Printf("❌ Erreur pour le batch %d: %v", i/batchSize+1, err)
			totalFailed += len(batch)
			continue
		}

		totalSuccess += s
		totalFailed += f
		allFailedTokens = append(allFailedTokens, ft...)
	}

	return totalSuccess, totalFailed, allFailedTokens
}
