package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"premier-an-backend/database"
	"premier-an-backend/middleware"
	"premier-an-backend/models"
	"premier-an-backend/services"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// WebSocketHub interface pour éviter la dépendance circulaire
type WebSocketHub interface {
	SendToUser(userID string, payload interface{})
	SendToConversation(conversationID string, payload interface{}, excludeUserID string)
	IsUserOnline(userID string) bool
}

// ChatHandler gère les requêtes liées au chat admin
type ChatHandler struct {
	chatRepo     *database.ChatRepository
	userRepo     *database.UserRepository
	fcmTokenRepo *database.FCMTokenRepository
	fcmService   *services.FCMService
	wsHub        WebSocketHub
}

// NewChatHandler crée un nouveau handler pour le chat
func NewChatHandler(chatRepo *database.ChatRepository, userRepo *database.UserRepository, fcmTokenRepo *database.FCMTokenRepository, fcmService *services.FCMService, wsHub WebSocketHub) *ChatHandler {
	return &ChatHandler{
		chatRepo:     chatRepo,
		userRepo:     userRepo,
		fcmTokenRepo: fcmTokenRepo,
		fcmService:   fcmService,
		wsHub:        wsHub,
	}
}

// getUserObjectID récupère l'ObjectID d'un utilisateur à partir de son email (UserID du JWT)
func (h *ChatHandler) getUserObjectID(email string) (primitive.ObjectID, error) {
	user, err := h.userRepo.FindByEmail(email)
	if err != nil || user == nil {
		return primitive.NilObjectID, err
	}
	return user.ID, nil
}

// GetConversations récupère les conversations de l'admin connecté
func (h *ChatHandler) GetConversations(w http.ResponseWriter, r *http.Request) {
	// Récupérer les claims depuis le contexte
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		http.Error(w, "Token invalide", http.StatusUnauthorized)
		return
	}

	// Récupérer l'ObjectID de l'utilisateur via son email
	userID, err := h.getUserObjectID(claims.UserID)
	if err != nil {
		log.Printf("Erreur conversion ID: %v", err)
		http.Error(w, "Utilisateur introuvable", http.StatusBadRequest)
		return
	}

	// Vérifier que l'utilisateur est admin
	user, err := h.userRepo.FindByID(userID)
	if err != nil || user.Admin != 1 {
		http.Error(w, "Accès refusé. Admin uniquement", http.StatusForbidden)
		return
	}

	// Récupérer les conversations ET les invitations envoyées
	conversations, err := h.chatRepo.GetConversationsAndInvitations(r.Context(), userID)
	if err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}

	// ✅ Ajouter is_online et last_seen pour chaque participant
	for i := range conversations {
		participantID := conversations[i].Participant.ID

		// Vérifier si le participant est en ligne (via WebSocket)
		if h.wsHub != nil {
			conversations[i].Participant.IsOnline = h.wsHub.IsUserOnline(participantID)
		}

		// Récupérer le last_seen depuis la DB
		if partObjID, err := primitive.ObjectIDFromHex(participantID); err == nil {
			if partUser, err := h.userRepo.FindByID(partObjID); err == nil && partUser != nil {
				conversations[i].Participant.LastSeen = partUser.LastSeen
			}
		}
	}

	response := models.ChatResponse{
		Success: true,
		Data: map[string]interface{}{
			"conversations": conversations,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetMessages récupère les messages d'une conversation
func (h *ChatHandler) GetMessages(w http.ResponseWriter, r *http.Request) {
	// Récupérer les claims depuis le contexte
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		http.Error(w, "Token invalide", http.StatusUnauthorized)
		return
	}

	// Récupérer l'ObjectID de l'utilisateur via son email
	userID, err := h.getUserObjectID(claims.UserID)
	if err != nil {
		log.Printf("Erreur conversion ID: %v", err)
		http.Error(w, "Utilisateur introuvable", http.StatusBadRequest)
		return
	}

	// Vérifier que l'utilisateur est admin
	user, err := h.userRepo.FindByID(userID)
	if err != nil || user.Admin != 1 {
		http.Error(w, "Accès refusé. Admin uniquement", http.StatusForbidden)
		return
	}

	// Récupérer l'ID de la conversation depuis les paramètres d'URL
	vars := mux.Vars(r)
	conversationIDStr := vars["id"]
	if conversationIDStr == "" {
		http.Error(w, "ID de conversation requis", http.StatusBadRequest)
		return
	}

	conversationID, err := primitive.ObjectIDFromHex(conversationIDStr)
	if err != nil {
		http.Error(w, "ID de conversation invalide", http.StatusBadRequest)
		return
	}

	// Vérifier que l'utilisateur fait partie de la conversation
	conversation, err := h.chatRepo.GetConversationByID(r.Context(), conversationID)
	if err != nil {
		http.Error(w, "Conversation non trouvée", http.StatusNotFound)
		return
	}

	// Vérifier que l'utilisateur est participant
	isParticipant := false
	for _, participant := range conversation.Participants {
		if participant.UserID == userID {
			isParticipant = true
			break
		}
	}

	if !isParticipant {
		http.Error(w, "Accès refusé à cette conversation", http.StatusForbidden)
		return
	}

	// Récupérer le paramètre limit
	limit := 50 // Par défaut
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	// Récupérer les messages (et marquer automatiquement comme distribués)
	messages, err := h.chatRepo.GetMessages(r.Context(), conversationID, userID, limit)
	if err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}

	// Enrichir les messages avec les données de l'expéditeur
	for i := range messages {
		sender, err := h.userRepo.FindByID(messages[i].SenderID)
		if err == nil && sender != nil {
			messages[i].Sender = &models.UserInfo{
				ID:              sender.ID.Hex(),
				Firstname:       sender.Firstname,
				Lastname:        sender.Lastname,
				Email:           sender.Email,
				ProfilePicture:  sender.ProfileImageURL,
				ProfileImageURL: sender.ProfileImageURL,
			}
		}
	}

	response := models.ChatResponse{
		Success: true,
		Data: map[string]interface{}{
			"messages": messages,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// MarkConversationAsRead marque tous les messages d'une conversation comme lus
func (h *ChatHandler) MarkConversationAsRead(w http.ResponseWriter, r *http.Request) {
	// Récupérer les claims depuis le contexte
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		http.Error(w, "Token invalide", http.StatusUnauthorized)
		return
	}

	// Récupérer l'ObjectID de l'utilisateur via son email
	userID, err := h.getUserObjectID(claims.UserID)
	if err != nil {
		log.Printf("Erreur conversion ID: %v", err)
		http.Error(w, "Utilisateur introuvable", http.StatusBadRequest)
		return
	}

	// Récupérer l'ID de la conversation depuis les paramètres d'URL
	vars := mux.Vars(r)
	conversationIDStr := vars["id"]
	if conversationIDStr == "" {
		http.Error(w, "ID de conversation requis", http.StatusBadRequest)
		return
	}

	conversationID, err := primitive.ObjectIDFromHex(conversationIDStr)
	if err != nil {
		http.Error(w, "ID de conversation invalide", http.StatusBadRequest)
		return
	}

	// Vérifier que l'utilisateur fait partie de la conversation
	conversation, err := h.chatRepo.GetConversationByID(r.Context(), conversationID)
	if err != nil {
		http.Error(w, "Conversation non trouvée", http.StatusNotFound)
		return
	}

	// Vérifier que l'utilisateur est participant
	isParticipant := false
	for _, participant := range conversation.Participants {
		if participant.UserID == userID {
			isParticipant = true
			break
		}
	}

	if !isParticipant {
		http.Error(w, "Accès refusé à cette conversation", http.StatusForbidden)
		return
	}

	// Marquer les messages comme lus
	markedCount, err := h.chatRepo.MarkConversationAsRead(r.Context(), conversationID, userID)
	if err != nil {
		log.Printf("❌ Erreur marquage lu: %v", err)
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}

	log.Printf("✅ %d messages marqués comme lus dans la conversation %s", markedCount, conversationIDStr)

	// 🔌 Envoyer via WebSocket aux autres participants pour mettre à jour les coches
	if h.wsHub != nil && markedCount > 0 {
		readAt := time.Now()
		payload := map[string]interface{}{
			"type":            "messages_read",
			"conversation_id": conversationIDStr,
			"read_by_user_id": userID.Hex(),
			"read_at":         readAt,
		}

		// Envoyer à tous les autres participants
		for _, participant := range conversation.Participants {
			participantID := participant.UserID.Hex()
			if participantID != userID.Hex() {
				log.Printf("📤 Envoi messages_read WS au participant: %s", participantID)
				h.wsHub.SendToUser(participantID, payload)
			}
		}

		log.Printf("🔌 Événement messages_read envoyé à tous les participants")
	}

	response := models.ChatResponse{
		Success: true,
		Data: map[string]interface{}{
			"marked_read": markedCount,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// SendMessage envoie un message dans une conversation
func (h *ChatHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	// Récupérer les claims depuis le contexte
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		http.Error(w, "Token invalide", http.StatusUnauthorized)
		return
	}

	// Récupérer l'ObjectID de l'utilisateur via son email
	userID, err := h.getUserObjectID(claims.UserID)
	if err != nil {
		log.Printf("Erreur conversion ID: %v", err)
		http.Error(w, "Utilisateur introuvable", http.StatusBadRequest)
		return
	}

	// Vérifier que l'utilisateur est admin
	user, err := h.userRepo.FindByID(userID)
	if err != nil || user.Admin != 1 {
		http.Error(w, "Accès refusé. Admin uniquement", http.StatusForbidden)
		return
	}

	// Récupérer l'ID de la conversation depuis les paramètres d'URL
	vars := mux.Vars(r)
	conversationIDStr := vars["id"]
	if conversationIDStr == "" {
		http.Error(w, "ID de conversation requis", http.StatusBadRequest)
		return
	}

	conversationID, err := primitive.ObjectIDFromHex(conversationIDStr)
	if err != nil {
		http.Error(w, "ID de conversation invalide", http.StatusBadRequest)
		return
	}

	// Vérifier que l'utilisateur fait partie de la conversation
	conversation, err := h.chatRepo.GetConversationByID(r.Context(), conversationID)
	if err != nil {
		http.Error(w, "Conversation non trouvée", http.StatusNotFound)
		return
	}

	// Vérifier que l'utilisateur est participant
	isParticipant := false
	for _, participant := range conversation.Participants {
		if participant.UserID == userID {
			isParticipant = true
			break
		}
	}

	if !isParticipant {
		http.Error(w, "Accès refusé à cette conversation", http.StatusForbidden)
		return
	}

	// Parser le body JSON
	var request models.MessageRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Body JSON invalide", http.StatusBadRequest)
		return
	}

	// Validation
	if strings.TrimSpace(request.Content) == "" {
		http.Error(w, "Contenu du message requis", http.StatusBadRequest)
		return
	}

	// Créer le message
	message := &models.Message{
		ConversationID: conversationID,
		SenderID:       userID,
		Content:        strings.TrimSpace(request.Content),
		Type:           "text",
	}

	// Envoyer le message
	if err := h.chatRepo.SendMessage(r.Context(), message); err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}

	log.Printf("📨 Message créé: ID=%s, ConvID=%s, Content=%s", message.ID.Hex(), conversationIDStr, message.Content)

	// Envoyer une notification aux autres participants (FCM)
	go h.sendMessageNotification(conversation, message, userID)

	// 🔌 Envoyer via WebSocket à TOUS les participants (même ceux qui n'ont pas rejoint la room)
	log.Printf("🔍 wsHub nil? %v", h.wsHub == nil)
	if h.wsHub != nil {
		log.Printf("🔌 Envoi WebSocket à tous les participants de la conversation %s...", conversationIDStr)

		payload := map[string]interface{}{
			"type":            "new_message",
			"conversation_id": conversationIDStr,
			"message": map[string]interface{}{
				"id":              message.ID.Hex(),
				"conversation_id": conversationIDStr,
				"sender_id":       userID.Hex(),
				"content":         message.Content,
				"timestamp":       message.CreatedAt,
				"delivered_at":    message.DeliveredAt,
				"read_at":         message.ReadAt,
			},
		}

		// Envoyer à chaque participant de la conversation
		for _, participant := range conversation.Participants {
			if participant.UserID != userID { // Ne pas renvoyer à l'expéditeur
				// ⚠️  IMPORTANT: Utiliser EMAIL, pas ObjectID !
				// Récupérer l'email du participant depuis la DB
				if participantUser, err := h.userRepo.FindByID(participant.UserID); err == nil && participantUser != nil {
					participantEmail := participantUser.Email
					log.Printf("📤 Envoi WS new_message à %s (email: %s)", participant.UserID.Hex(), participantEmail)
					h.wsHub.SendToUser(participantEmail, payload)
				} else {
					log.Printf("❌ Participant introuvable: %s", participant.UserID.Hex())
				}
			}
		}

		log.Printf("🔌 Message WebSocket envoyé à tous les participants")
	} else {
		log.Printf("⚠️  wsHub est nil - WebSocket non disponible")
	}

	response := models.ChatResponse{
		Success: true,
		Data: map[string]interface{}{
			"message": message,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// SearchAdmins recherche des administrateurs
func (h *ChatHandler) SearchAdmins(w http.ResponseWriter, r *http.Request) {
	// Récupérer les claims depuis le contexte
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		http.Error(w, "Token invalide", http.StatusUnauthorized)
		return
	}

	// Récupérer l'ObjectID de l'utilisateur via son email
	userID, err := h.getUserObjectID(claims.UserID)
	if err != nil {
		log.Printf("Erreur conversion ID: %v", err)
		http.Error(w, "Utilisateur introuvable", http.StatusBadRequest)
		return
	}

	// Vérifier que l'utilisateur est admin
	user, err := h.userRepo.FindByID(userID)
	if err != nil || user.Admin != 1 {
		http.Error(w, "Accès refusé. Admin uniquement", http.StatusForbidden)
		return
	}

	// Récupérer les paramètres de recherche
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Paramètre de recherche 'q' requis", http.StatusBadRequest)
		return
	}

	limit := 10 // Par défaut
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 50 {
			limit = parsedLimit
		}
	}

	// Rechercher les admins
	admins, err := h.chatRepo.SearchAdmins(r.Context(), query, limit)
	if err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}

	response := models.ChatResponse{
		Success: true,
		Data: map[string]interface{}{
			"admins": admins,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// SendInvitation envoie une invitation de chat
func (h *ChatHandler) SendInvitation(w http.ResponseWriter, r *http.Request) {
	// Récupérer les claims depuis le contexte
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		http.Error(w, "Token invalide", http.StatusUnauthorized)
		return
	}

	// Récupérer l'ObjectID de l'utilisateur via son email
	userID, err := h.getUserObjectID(claims.UserID)
	if err != nil {
		log.Printf("Erreur conversion ID: %v", err)
		http.Error(w, "Utilisateur introuvable", http.StatusBadRequest)
		return
	}

	// Vérifier que l'utilisateur est admin
	user, err := h.userRepo.FindByID(userID)
	if err != nil || user.Admin != 1 {
		http.Error(w, "Accès refusé. Admin uniquement", http.StatusForbidden)
		return
	}

	// Parser le body JSON
	var request models.InvitationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Body JSON invalide", http.StatusBadRequest)
		return
	}

	// Validation
	if strings.TrimSpace(request.ToUserID) == "" {
		http.Error(w, "ID utilisateur destinataire requis", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(request.Message) == "" {
		http.Error(w, "Message d'invitation requis", http.StatusBadRequest)
		return
	}

	// Convertir l'ID du destinataire
	toUserID, err := primitive.ObjectIDFromHex(request.ToUserID)
	if err != nil {
		http.Error(w, "ID utilisateur destinataire invalide", http.StatusBadRequest)
		return
	}

	// Vérifier que le destinataire est admin
	toUser, err := h.userRepo.FindByID(toUserID)
	if err != nil || toUser.Admin != 1 {
		http.Error(w, "Le destinataire doit être un administrateur", http.StatusBadRequest)
		return
	}

	// Vérifier qu'on ne s'invite pas soi-même
	if userID == toUserID {
		http.Error(w, "Vous ne pouvez pas vous inviter vous-même", http.StatusBadRequest)
		return
	}

	// Créer l'invitation
	invitation := &models.ChatInvitation{
		FromUserID: userID,
		ToUserID:   toUserID,
		Message:    strings.TrimSpace(request.Message),
	}

	// Envoyer l'invitation
	if err := h.chatRepo.CreateInvitation(r.Context(), invitation); err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}

	// Envoyer une notification (FCM)
	go h.sendInvitationNotification(invitation, user)

	// 🔌 Envoyer via WebSocket au destinataire
	if h.wsHub != nil {
		h.wsHub.SendToUser(
			toUserID.Hex(),
			map[string]interface{}{
				"type": "new_invitation",
				"invitation": map[string]interface{}{
					"id":           invitation.ID.Hex(),
					"from_user_id": userID.Hex(),
					"to_user_id":   toUserID.Hex(),
					"status":       "pending",
					"message":      invitation.Message,
					"created_at":   invitation.CreatedAt,
					"fromUser": map[string]interface{}{
						"id":        user.ID.Hex(),
						"firstname": user.Firstname,
						"lastname":  user.Lastname,
						"email":     user.Email,
					},
				},
			},
		)
		log.Printf("🔌 Invitation WebSocket envoyée à %s", toUserID.Hex())
	}

	response := models.ChatResponse{
		Success: true,
		Data: map[string]interface{}{
			"invitation": invitation,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetInvitations récupère les invitations reçues
func (h *ChatHandler) GetInvitations(w http.ResponseWriter, r *http.Request) {
	// Récupérer les claims depuis le contexte
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		http.Error(w, "Token invalide", http.StatusUnauthorized)
		return
	}

	// Récupérer l'ObjectID de l'utilisateur via son email
	userID, err := h.getUserObjectID(claims.UserID)
	if err != nil {
		log.Printf("Erreur conversion ID: %v", err)
		http.Error(w, "Utilisateur introuvable", http.StatusBadRequest)
		return
	}

	// Vérifier que l'utilisateur est admin
	user, err := h.userRepo.FindByID(userID)
	if err != nil || user.Admin != 1 {
		http.Error(w, "Accès refusé. Admin uniquement", http.StatusForbidden)
		return
	}

	// Récupérer les invitations
	invitations, err := h.chatRepo.GetInvitations(r.Context(), userID)
	if err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}

	response := models.ChatResponse{
		Success: true,
		Data: map[string]interface{}{
			"invitations": invitations,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// RespondToInvitation répond à une invitation
func (h *ChatHandler) RespondToInvitation(w http.ResponseWriter, r *http.Request) {
	// Récupérer les claims depuis le contexte
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		http.Error(w, "Token invalide", http.StatusUnauthorized)
		return
	}

	// Récupérer l'ObjectID de l'utilisateur via son email
	userID, err := h.getUserObjectID(claims.UserID)
	if err != nil {
		log.Printf("Erreur conversion ID: %v", err)
		http.Error(w, "Utilisateur introuvable", http.StatusBadRequest)
		return
	}

	// Vérifier que l'utilisateur est admin
	user, err := h.userRepo.FindByID(userID)
	if err != nil || user.Admin != 1 {
		http.Error(w, "Accès refusé. Admin uniquement", http.StatusForbidden)
		return
	}

	// Récupérer l'ID de l'invitation depuis les paramètres d'URL
	vars := mux.Vars(r)
	invitationIDStr := vars["id"]
	if invitationIDStr == "" {
		http.Error(w, "ID d'invitation requis", http.StatusBadRequest)
		return
	}

	invitationID, err := primitive.ObjectIDFromHex(invitationIDStr)
	if err != nil {
		http.Error(w, "ID d'invitation invalide", http.StatusBadRequest)
		return
	}

	// Parser le body JSON
	var request models.InvitationResponse
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Body JSON invalide", http.StatusBadRequest)
		return
	}

	// Validation
	if request.Action != "accept" && request.Action != "reject" {
		http.Error(w, "Action invalide. Doit être 'accept' ou 'reject'", http.StatusBadRequest)
		return
	}

	// Récupérer l'invitation pour avoir les IDs
	var invitation models.ChatInvitation
	err = h.chatRepo.InvitationCollection.FindOne(r.Context(), bson.M{"_id": invitationID}).Decode(&invitation)
	if err != nil {
		http.Error(w, "Invitation non trouvée", http.StatusNotFound)
		return
	}

	// Répondre à l'invitation
	conversation, err := h.chatRepo.RespondToInvitation(r.Context(), invitationID, request.Action)
	if err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}

	// Si acceptée, envoyer une notification au demandeur
	if request.Action == "accept" && conversation != nil {
		go h.sendAcceptedInvitationNotification(&invitation, user)

		// 🔌 Envoyer via WebSocket à l'expéditeur
		if h.wsHub != nil {
			// Récupérer les infos du participant pour le payload
			h.wsHub.SendToUser(
				invitation.FromUserID.Hex(),
				map[string]interface{}{
					"type":          "invitation_accepted",
					"invitation_id": invitationID.Hex(),
					"conversation": map[string]interface{}{
						"id": conversation.ID.Hex(),
						"participant": map[string]interface{}{
							"id":        user.ID.Hex(),
							"firstname": user.Firstname,
							"lastname":  user.Lastname,
							"email":     user.Email,
						},
						"status":       "accepted",
						"unread_count": 0,
					},
				},
			)
			log.Printf("🔌 invitation_accepted WebSocket envoyée à %s", invitation.FromUserID.Hex())
		}
	} else if request.Action == "reject" {
		// 🔌 Envoyer via WebSocket à l'expéditeur
		if h.wsHub != nil {
			h.wsHub.SendToUser(
				invitation.FromUserID.Hex(),
				map[string]interface{}{
					"type":          "invitation_rejected",
					"invitation_id": invitationID.Hex(),
				},
			)
			log.Printf("🔌 invitation_rejected WebSocket envoyée à %s", invitation.FromUserID.Hex())
		}
	}

	response := models.ChatResponse{
		Success: true,
		Data: map[string]interface{}{
			"conversation": conversation,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// SendChatNotification envoie une notification de chat
func (h *ChatHandler) SendChatNotification(w http.ResponseWriter, r *http.Request) {
	// Récupérer les claims depuis le contexte
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		http.Error(w, "Token invalide", http.StatusUnauthorized)
		return
	}

	// Récupérer l'ObjectID de l'utilisateur via son email
	userID, err := h.getUserObjectID(claims.UserID)
	if err != nil {
		log.Printf("Erreur conversion ID: %v", err)
		http.Error(w, "Utilisateur introuvable", http.StatusBadRequest)
		return
	}

	// Vérifier que l'utilisateur est admin
	user, err := h.userRepo.FindByID(userID)
	if err != nil || user.Admin != 1 {
		http.Error(w, "Accès refusé. Admin uniquement", http.StatusForbidden)
		return
	}

	// Parser le body JSON
	var request models.ChatNotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Body JSON invalide", http.StatusBadRequest)
		return
	}

	// Validation
	if strings.TrimSpace(request.ToUserID) == "" {
		http.Error(w, "ID utilisateur destinataire requis", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(request.Type) == "" {
		http.Error(w, "Type de notification requis", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(request.Title) == "" {
		http.Error(w, "Titre requis", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(request.Body) == "" {
		http.Error(w, "Corps du message requis", http.StatusBadRequest)
		return
	}

	// Convertir l'ID du destinataire
	toUserID, err := primitive.ObjectIDFromHex(request.ToUserID)
	if err != nil {
		http.Error(w, "ID utilisateur destinataire invalide", http.StatusBadRequest)
		return
	}

	// Envoyer la notification
	if h.fcmService != nil {
		// Récupérer l'utilisateur destinataire pour obtenir son email
		toUser, err := h.userRepo.FindByID(toUserID)
		if err != nil {
			http.Error(w, "Utilisateur destinataire non trouvé", http.StatusNotFound)
			return
		}

		// Récupérer les tokens FCM de l'utilisateur (par email)
		fcmTokens, err := h.fcmTokenRepo.FindByUserID(toUser.Email)
		if err == nil && len(fcmTokens) > 0 {
			// Convertir les données en map[string]string pour FCM
			fcmData := make(map[string]string)
			for k, v := range request.Data {
				if str, ok := v.(string); ok {
					fcmData[k] = str
				}
			}

			// Envoyer à tous les tokens de l'utilisateur
			for _, token := range fcmTokens {
				err = h.fcmService.SendToToken(token.Token, request.Title, request.Body, fcmData)
				if err != nil {
					// Log l'erreur mais continue avec les autres tokens
					continue
				}
			}
		}
	}

	response := models.ChatResponse{
		Success: true,
		Message: "Notification envoyée avec succès",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// sendMessageNotification envoie une notification pour un nouveau message
func (h *ChatHandler) sendMessageNotification(conversation *models.Conversation, message *models.Message, senderID primitive.ObjectID) {
	if h.fcmService == nil {
		log.Println("⚠️  FCM Service non disponible pour les notifications de message")
		return
	}

	log.Printf("📨 Envoi de notification pour le message: %s", message.ID.Hex())

	// Trouver les autres participants
	for _, participant := range conversation.Participants {
		if participant.UserID != senderID {
			log.Printf("👤 Participant trouvé: %s", participant.UserID.Hex())

			// Récupérer les informations de l'expéditeur
			sender, err := h.userRepo.FindByID(senderID)
			if err != nil {
				log.Printf("❌ Erreur récupération expéditeur: %v", err)
				continue
			}

			// Format ultra simple : Juste le nom et le message
			title := sender.Firstname + " " + strings.ToUpper(sender.Lastname)
			body := message.Content
			if len(body) > 240 {
				body = body[:240] // Limiter à 240 caractères
			}
			data := map[string]interface{}{
				"type":           "chat_message",
				"conversationId": conversation.ID.Hex(),
				"messageId":      message.ID.Hex(),
				"senderId":       senderID.Hex(),
				"senderName":     sender.Firstname + " " + sender.Lastname,
			}

			// Récupérer l'utilisateur pour obtenir son email
			participantUser, err := h.userRepo.FindByID(participant.UserID)
			if err != nil {
				log.Printf("❌ Erreur récupération participant: %v", err)
				continue
			}

			log.Printf("📧 Email du participant: %s", participantUser.Email)

			// Récupérer les tokens FCM du participant (par email)
			fcmTokens, err := h.fcmTokenRepo.FindByUserID(participantUser.Email)
			if err != nil {
				log.Printf("❌ Erreur récupération tokens FCM: %v", err)
				continue
			}

			log.Printf("🔑 Tokens FCM trouvés: %d", len(fcmTokens))

			if len(fcmTokens) > 0 {
				// Convertir les données en map[string]string pour FCM
				fcmData := make(map[string]string)
				for k, v := range data {
					if str, ok := v.(string); ok {
						fcmData[k] = str
					}
				}

				// 🔍 LOGS CRITIQUES - Vérifier que conversationId est bien présent
				log.Printf("📤 Données FCM à envoyer:")
				log.Printf("   type: %s", fcmData["type"])
				log.Printf("   conversationId: %s", fcmData["conversationId"])
				log.Printf("   messageId: %s", fcmData["messageId"])

				// Envoyer à tous les tokens du participant
				for _, token := range fcmTokens {
					err := h.fcmService.SendToToken(token.Token, title, body, fcmData)
					if err != nil {
						log.Printf("❌ Erreur envoi FCM: %v", err)
					} else {
						log.Printf("✅ Notification envoyée à %s (token: %s...)", participantUser.Email, token.Token[:20])
					}
				}
			} else {
				log.Printf("⚠️  Aucun token FCM trouvé pour %s", participantUser.Email)
			}
		}
	}
}

// sendInvitationNotification envoie une notification pour une invitation
func (h *ChatHandler) sendInvitationNotification(invitation *models.ChatInvitation, fromUser *models.User) {
	if h.fcmService == nil {
		return
	}

	// Format ultra simple : Juste le nom et un message court
	title := fromUser.Firstname + " " + strings.ToUpper(fromUser.Lastname)
	body := "Vous invite à discuter"
	data := map[string]interface{}{
		"type":         "chat_invitation",
		"invitationId": invitation.ID.Hex(),
		"fromUserId":   fromUser.ID.Hex(),
		"fromUserName": fromUser.Firstname + " " + fromUser.Lastname,
	}

	// Récupérer l'utilisateur destinataire pour obtenir son email
	toUser, err := h.userRepo.FindByID(invitation.ToUserID)
	if err != nil {
		return
	}

	// Récupérer les tokens FCM du destinataire (par email)
	fcmTokens, err := h.fcmTokenRepo.FindByUserID(toUser.Email)
	if err == nil && len(fcmTokens) > 0 {
		// Convertir les données en map[string]string pour FCM
		fcmData := make(map[string]string)
		for k, v := range data {
			if str, ok := v.(string); ok {
				fcmData[k] = str
			}
		}
		// Envoyer à tous les tokens du destinataire
		for _, token := range fcmTokens {
			h.fcmService.SendToToken(token.Token, title, body, fcmData)
		}
	}
}

// sendAcceptedInvitationNotification envoie une notification quand une invitation est acceptée
func (h *ChatHandler) sendAcceptedInvitationNotification(invitation *models.ChatInvitation, acceptedByUser *models.User) {
	if h.fcmService == nil {
		return
	}

	// Format ultra simple : Juste le nom et un message court
	title := acceptedByUser.Firstname + " " + strings.ToUpper(acceptedByUser.Lastname)
	body := "A accepté votre invitation"
	data := map[string]interface{}{
		"type":           "chat_invitation_accepted",
		"invitationId":   invitation.ID.Hex(),
		"acceptedBy":     acceptedByUser.ID.Hex(),
		"acceptedByName": acceptedByUser.Firstname + " " + acceptedByUser.Lastname,
	}

	// Récupérer l'utilisateur demandeur pour obtenir son email
	fromUser, err := h.userRepo.FindByID(invitation.FromUserID)
	if err != nil {
		return
	}

	// Récupérer les tokens FCM du demandeur (par email)
	fcmTokens, err := h.fcmTokenRepo.FindByUserID(fromUser.Email)
	if err == nil && len(fcmTokens) > 0 {
		// Convertir les données en map[string]string pour FCM
		fcmData := make(map[string]string)
		for k, v := range data {
			if str, ok := v.(string); ok {
				fcmData[k] = str
			}
		}
		// Envoyer à tous les tokens du demandeur
		for _, token := range fcmTokens {
			h.fcmService.SendToToken(token.Token, title, body, fcmData)
		}
	}
}
