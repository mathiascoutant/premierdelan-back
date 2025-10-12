package services

import (
	"fmt"
	"log"
	"premier-an-backend/database"
	"premier-an-backend/models"

	"github.com/robfig/cron/v3"
	"go.mongodb.org/mongo-driver/mongo"
)

// NotificationCron gère les notifications automatiques
type NotificationCron struct {
	eventRepo    *database.EventRepository
	fcmTokenRepo *database.FCMTokenRepository
	fcmService   *FCMService
	cron         *cron.Cron
}

// NewNotificationCron crée une nouvelle instance
func NewNotificationCron(db *mongo.Database, fcmService *FCMService) *NotificationCron {
	return &NotificationCron{
		eventRepo:    database.NewEventRepository(db),
		fcmTokenRepo: database.NewFCMTokenRepository(db),
		fcmService:   fcmService,
		cron:         cron.New(),
	}
}

// Start démarre le cron job
func (nc *NotificationCron) Start() {
	// Vérifier toutes les minutes si des événements doivent ouvrir leurs inscriptions
	nc.cron.AddFunc("@every 1m", nc.checkEventOpenings)
	nc.cron.Start()
	log.Println("✓ Cron job notifications démarré (vérification toutes les minutes)")
}

// Stop arrête le cron job
func (nc *NotificationCron) Stop() {
	nc.cron.Stop()
}

// checkEventOpenings vérifie si des événements doivent ouvrir leurs inscriptions
func (nc *NotificationCron) checkEventOpenings() {
	events, err := nc.eventRepo.FindEventsToNotifyOpening()
	if err != nil {
		log.Printf("Erreur recherche événements à notifier: %v", err)
		return
	}

	if len(events) == 0 {
		return // Rien à faire
	}

	log.Printf("🔔 %d événement(s) à notifier pour ouverture d'inscriptions", len(events))

	for _, event := range events {
		nc.sendEventOpeningNotification(event)
		
		// Marquer comme envoyé
		nc.eventRepo.Update(event.ID, map[string]interface{}{
			"notification_sent_opening": true,
		})
	}
}

// sendEventOpeningNotification envoie la notification d'ouverture à tous les utilisateurs
func (nc *NotificationCron) sendEventOpeningNotification(event models.Event) {
	// Récupérer tous les tokens FCM
	allFCMTokens, err := nc.fcmTokenRepo.FindAll()
	if err != nil {
		log.Printf("Erreur récupération tokens FCM: %v", err)
		return
	}

	if len(allFCMTokens) == 0 {
		log.Println("⚠️  Aucun token FCM enregistré")
		return
	}

	// Extraire les tokens
	var tokens []string
	for _, t := range allFCMTokens {
		tokens = append(tokens, t.Token)
	}

	// Préparer la notification
	title := "🎉 Inscriptions ouvertes !"
	message := fmt.Sprintf("Les inscriptions pour '%s' sont maintenant ouvertes !", event.Titre)
	data := map[string]string{
		"action":   "event_opening",
		"url":      "/#evenements",
		"event_id": event.ID.Hex(),
	}

	// Envoyer via FCM
	success, failed, _ := nc.fcmService.SendToAll(tokens, title, message, data)
	log.Printf("📧 Notification ouverture '%s' envoyée: %d succès, %d échecs", event.Titre, success, failed)
}

