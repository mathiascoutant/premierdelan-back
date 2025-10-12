package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"premier-an-backend/database"
	"premier-an-backend/models"
	"premier-an-backend/utils"

	webpush "github.com/SherClockHolmes/webpush-go"
	"go.mongodb.org/mongo-driver/mongo"
)

// NotificationHandler gère les requêtes de notifications push
type NotificationHandler struct {
	subscriptionRepo *database.SubscriptionRepository
	vapidPublicKey   string
	vapidPrivateKey  string
	vapidSubject     string
}

// NewNotificationHandler crée une nouvelle instance de NotificationHandler
func NewNotificationHandler(db *mongo.Database, vapidPublicKey, vapidPrivateKey, vapidSubject string) *NotificationHandler {
	return &NotificationHandler{
		subscriptionRepo: database.NewSubscriptionRepository(db),
		vapidPublicKey:   vapidPublicKey,
		vapidPrivateKey:  vapidPrivateKey,
		vapidSubject:     vapidSubject,
	}
}

// Subscribe permet à un utilisateur de s'abonner aux notifications
func (h *NotificationHandler) Subscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.RespondError(w, http.StatusMethodNotAllowed, "Méthode non autorisée")
		return
	}

	var req models.SubscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Données invalides")
		return
	}

	// Vérifier si l'abonnement existe déjà
	existing, err := h.subscriptionRepo.FindByEndpoint(req.Subscription.Endpoint)
	if err != nil {
		log.Printf("Erreur lors de la vérification de l'abonnement: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}

	if existing != nil {
		utils.RespondSuccess(w, "Abonnement déjà existant", nil)
		return
	}

	// Créer l'abonnement
	subscription := &models.PushSubscription{
		UserID:   req.UserID,
		Endpoint: req.Subscription.Endpoint,
		Keys:     req.Subscription.Keys,
	}

	if err := h.subscriptionRepo.Create(subscription); err != nil {
		log.Printf("Erreur lors de la création de l'abonnement: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur lors de la création de l'abonnement")
		return
	}

	log.Printf("✓ Nouvel abonnement créé pour: %s", req.UserID)
	utils.RespondSuccess(w, "Abonnement créé avec succès", subscription)
}

// Unsubscribe permet à un utilisateur de se désabonner
func (h *NotificationHandler) Unsubscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.RespondError(w, http.StatusMethodNotAllowed, "Méthode non autorisée")
		return
	}

	var req struct {
		Endpoint string `json:"endpoint"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Données invalides")
		return
	}

	if err := h.subscriptionRepo.Delete(req.Endpoint); err != nil {
		log.Printf("Erreur lors de la suppression de l'abonnement: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}

	log.Printf("✓ Abonnement supprimé: %s", req.Endpoint)
	utils.RespondSuccess(w, "Désabonnement réussi", nil)
}

// SendTestNotification envoie une notification de test à tous les abonnés
func (h *NotificationHandler) SendTestNotification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.RespondError(w, http.StatusMethodNotAllowed, "Méthode non autorisée")
		return
	}

	var req models.NotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Données invalides")
		return
	}

	// Récupérer tous les abonnements
	subscriptions, err := h.subscriptionRepo.FindAll()
	if err != nil {
		log.Printf("Erreur lors de la récupération des abonnements: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}

	if len(subscriptions) == 0 {
		utils.RespondSuccess(w, "Aucun abonné trouvé", map[string]interface{}{
			"sent": 0,
			"total": 0,
		})
		return
	}

	// Créer la notification
	title := req.Title
	if title == "" {
		title = "Nouvelle notification"
	}
	
	message := req.Message
	if message == "" {
		message = "Vous avez reçu une nouvelle notification"
	}

	payload := models.NotificationPayload{
		Title: title,
		Body:  message,
		Icon:  "/icon-192x192.png",
		Badge: "/badge-72x72.png",
		Data:  req.Data,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Erreur lors de la création du payload: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}

	// Envoyer les notifications
	sent := 0
	failed := 0

	for _, sub := range subscriptions {
		s := &webpush.Subscription{
			Endpoint: sub.Endpoint,
			Keys: webpush.Keys{
				P256dh: sub.Keys.P256dh,
				Auth:   sub.Keys.Auth,
			},
		}

		resp, err := webpush.SendNotification(payloadBytes, s, &webpush.Options{
			Subscriber:      h.vapidSubject,
			VAPIDPublicKey:  h.vapidPublicKey,
			VAPIDPrivateKey: h.vapidPrivateKey,
			TTL:             86400, // 24 heures en secondes
			Urgency:         webpush.UrgencyHigh,
		})

		if err != nil {
			log.Printf("❌ Erreur lors de l'envoi de la notification à %s: %v", sub.UserID, err)
			failed++
			
			// Si l'endpoint n'est plus valide (410 Gone), supprimer l'abonnement
			if resp != nil && resp.StatusCode == 410 {
				log.Printf("🗑️  Suppression de l'abonnement invalide: %s", sub.Endpoint)
				_ = h.subscriptionRepo.Delete(sub.Endpoint)
			}
			continue
		}

		if resp.StatusCode == 201 || resp.StatusCode == 200 {
			log.Printf("✓ Notification envoyée à %s", sub.UserID)
			sent++
		} else {
			// Lire le corps de la réponse pour voir l'erreur exacte
			bodyBytes := make([]byte, 0)
			if resp != nil && resp.Body != nil {
				bodyBytes, _ = io.ReadAll(resp.Body)
			}
			log.Printf("⚠️  Réponse inattendue pour %s: %d - Body: %s", sub.UserID, resp.StatusCode, string(bodyBytes))
			log.Printf("🔍 Endpoint: %s", sub.Endpoint)
			log.Printf("🔍 VAPID Subject: %s", h.vapidSubject)
			log.Printf("🔍 VAPID Public Key: %s", h.vapidPublicKey[:50]+"...")
			failed++
		}

		if resp != nil {
			resp.Body.Close()
		}
	}

	log.Printf("📊 Notifications envoyées: %d/%d (échecs: %d)", sent, len(subscriptions), failed)

	utils.RespondSuccess(w, "Notifications envoyées", map[string]interface{}{
		"sent":   sent,
		"failed": failed,
		"total":  len(subscriptions),
	})
}

// GetVAPIDPublicKey retourne la clé publique VAPID
func (h *NotificationHandler) GetVAPIDPublicKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.RespondError(w, http.StatusMethodNotAllowed, "Méthode non autorisée")
		return
	}

	utils.RespondJSON(w, http.StatusOK, map[string]string{
		"publicKey": h.vapidPublicKey,
	})
}

