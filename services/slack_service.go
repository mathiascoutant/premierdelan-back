package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// SlackService gère l'envoi de notifications Slack
type SlackService struct {
	webhookURL string
	client     *http.Client
}

// SlackMessage représente un message Slack
type SlackMessage struct {
	Text        string       `json:"text,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

// Attachment représente une pièce jointe Slack
type Attachment struct {
	Color     string  `json:"color,omitempty"`
	Title     string  `json:"title,omitempty"`
	Text      string  `json:"text,omitempty"`
	Fields    []Field `json:"fields,omitempty"`
	Timestamp int64   `json:"ts,omitempty"`
	Footer    string  `json:"footer,omitempty"`
}

// Field représente un champ dans une pièce jointe Slack
type Field struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// NewSlackService crée une nouvelle instance de SlackService
func NewSlackService(webhookURL string) *SlackService {
	if webhookURL == "" {
		log.Println("⚠️  Slack webhook URL non configuré - notifications Slack désactivées")
		return &SlackService{
			webhookURL: "",
			client: &http.Client{
				Timeout: 5 * time.Second,
			},
		}
	}

	return &SlackService{
		webhookURL: webhookURL,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// SendErrorNotification envoie une notification d'erreur sur Slack
func (s *SlackService) SendErrorNotification(errorType, method, path, statusCode, message, origin, userAgent string) error {
	if s.webhookURL == "" {
		return nil // Service désactivé
	}

	// Déterminer la couleur selon le type d'erreur
	color := "danger" // Rouge par défaut
	if statusCode == "403" {
		color = "warning" // Orange pour les erreurs CORS/Forbidden
	}

	// Créer le message Slack
	slackMsg := SlackMessage{
		Attachments: []Attachment{
			{
				Color:     color,
				Title:     fmt.Sprintf("🚨 Erreur serveur: %s", errorType),
				Text:      message,
				Timestamp: time.Now().Unix(),
				Footer:    "Premier de l'An - Backend",
				Fields: []Field{
					{
						Title: "Méthode",
						Value: method,
						Short: true,
					},
					{
						Title: "Status Code",
						Value: statusCode,
						Short: true,
					},
					{
						Title: "Chemin",
						Value: path,
						Short: false,
					},
				},
			},
		},
	}

	// Ajouter l'origine si disponible
	if origin != "" {
		slackMsg.Attachments[0].Fields = append(slackMsg.Attachments[0].Fields, Field{
			Title: "Origin",
			Value: origin,
			Short: true,
		})
	}

	// Ajouter le User-Agent si disponible
	if userAgent != "" {
		slackMsg.Attachments[0].Fields = append(slackMsg.Attachments[0].Fields, Field{
			Title: "User-Agent",
			Value: userAgent,
			Short: false,
		})
	}

	// Convertir en JSON
	jsonData, err := json.Marshal(slackMsg)
	if err != nil {
		return fmt.Errorf("erreur lors de la sérialisation du message Slack: %w", err)
	}

	// Envoyer la requête
	req, err := http.NewRequest("POST", s.webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("erreur lors de la création de la requête: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("erreur lors de l'envoi à Slack: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Slack a retourné un code d'erreur: %d", resp.StatusCode)
	}

	log.Printf("✓ Notification Slack envoyée pour l'erreur: %s %s", method, path)
	return nil
}

// SendCriticalError envoie une notification pour une erreur critique
func (s *SlackService) SendCriticalError(method, path, statusCode, errorMessage, origin, userAgent string) {
	if err := s.SendErrorNotification(
		"Erreur Critique",
		method,
		path,
		statusCode,
		errorMessage,
		origin,
		userAgent,
	); err != nil {
		log.Printf("❌ Erreur lors de l'envoi de la notification Slack: %v", err)
	}
}

// SendCORSError envoie une notification pour une erreur CORS
func (s *SlackService) SendCORSError(method, path, origin, userAgent string) {
	if err := s.SendErrorNotification(
		"Erreur CORS",
		method,
		path,
		"403",
		fmt.Sprintf("Origine non autorisée: %s", origin),
		origin,
		userAgent,
	); err != nil {
		log.Printf("❌ Erreur lors de l'envoi de la notification Slack: %v", err)
	}
}
