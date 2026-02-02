package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"premier-an-backend/constants"
	"premier-an-backend/database"
	"premier-an-backend/middleware"
	"premier-an-backend/models"
	"premier-an-backend/services"
	"premier-an-backend/utils"

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

// requireAuth vérifie l'auth. Retourne (userID, true) ou écrit l'erreur et (zero, false).
func (h *ChatHandler) requireAuth(w http.ResponseWriter, r *http.Request) (primitive.ObjectID, bool) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		utils.RespondError(w, http.StatusUnauthorized, constants.ErrInvalidToken)
		return primitive.NilObjectID, false
	}
	userID, err := h.getUserObjectID(claims.UserID)
	if err != nil {
		log.Printf(constants.ErrIDConversion, err)
		utils.RespondError(w, http.StatusBadRequest, constants.ErrUserNotFound)
		return primitive.NilObjectID, false
	}
	return userID, true
}

// requireAdmin vérifie l'auth et le rôle admin. Retourne (userID, true) ou écrit l'erreur et (zero, false).
func (h *ChatHandler) requireAdmin(w http.ResponseWriter, r *http.Request) (primitive.ObjectID, bool) {
	userID, ok := h.requireAuth(w, r)
	if !ok {
		return primitive.NilObjectID, false
	}
	user, err := h.userRepo.FindByID(userID)
	if err != nil || user.Admin != 1 {
		utils.RespondError(w, http.StatusForbidden, constants.ErrAdminOnly)
		return primitive.NilObjectID, false
	}
	return userID, true
}

// GetConversations récupère les conversations de l'admin connecté
func (h *ChatHandler) GetConversations(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireAdmin(w, r)
	if !ok {
		return
	}
	// Récupérer les conversations ET les invitations envoyées
	conversations, err := h.chatRepo.GetConversationsAndInvitations(r.Context(), userID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, constants.ErrServerError)
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

	utils.RespondJSON(w, http.StatusOK, models.ChatResponse{
		Success: true,
		Data: map[string]interface{}{
			"conversations": conversations,
		},
	})
}

// GetMessages récupère les messages d'une conversation
func (h *ChatHandler) GetMessages(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireAdmin(w, r)
	if !ok {
		return
	}
	// Récupérer l'ID de la conversation depuis les paramètres d'URL
	vars := mux.Vars(r)
	conversationIDStr := vars["id"]
	if conversationIDStr == "" {
		utils.RespondError(w, http.StatusBadRequest, constants.ErrConvIDRequired)
		return
	}

	conversationID, err := primitive.ObjectIDFromHex(conversationIDStr)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, constants.ErrInvalidConvID)
		return
	}

	// Vérifier que l'utilisateur fait partie de la conversation
	conversation, err := h.chatRepo.GetConversationByID(r.Context(), conversationID)
	if err != nil {
		utils.RespondError(w, http.StatusNotFound, constants.ErrConvNotFound)
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
		utils.RespondError(w, http.StatusForbidden, constants.ErrConvAccessDenied)
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
		utils.RespondError(w, http.StatusInternalServerError, constants.ErrServerError)
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

	utils.RespondJSON(w, http.StatusOK, models.ChatResponse{
		Success: true,
		Data: map[string]interface{}{
			"messages": messages,
		},
	})
}

// MarkConversationAsRead marque tous les messages d'une conversation comme lus
func (h *ChatHandler) MarkConversationAsRead(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireAuth(w, r)
	if !ok {
		return
	}
	// Récupérer l'ID de la conversation depuis les paramètres d'URL
	vars := mux.Vars(r)
	conversationIDStr := vars["id"]
	if conversationIDStr == "" {
		utils.RespondError(w, http.StatusBadRequest, constants.ErrConvIDRequired)
		return
	}

	conversationID, err := primitive.ObjectIDFromHex(conversationIDStr)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, constants.ErrInvalidConvID)
		return
	}

	// Vérifier que l'utilisateur fait partie de la conversation
	conversation, err := h.chatRepo.GetConversationByID(r.Context(), conversationID)
	if err != nil {
		utils.RespondError(w, http.StatusNotFound, constants.ErrConvNotFound)
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
		utils.RespondError(w, http.StatusForbidden, constants.ErrConvAccessDenied)
		return
	}

	// ⚠️ IMPORTANT : Récupérer les expéditeurs AVANT de marquer comme lus
	// Car après le marquage, les messages sont déjà marqués comme lus
	var senderIDs []primitive.ObjectID
	if h.wsHub != nil {
		senderIDs, err = h.chatRepo.GetSendersOfUnreadMessages(r.Context(), conversationID, userID)
		if err != nil {
			log.Printf("Erreur récupération expéditeurs: %v", err)
		}
	}

	// Marquer les messages comme lus
	markedCount, err := h.chatRepo.MarkConversationAsRead(r.Context(), conversationID, userID)
	if err != nil {
		log.Printf("Erreur marquage lu: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, constants.ErrServerError)
		return
	}

	// 🔌 Envoyer via WebSocket UNIQUEMENT aux expéditeurs des messages qui viennent d'être lus
	if h.wsHub != nil && markedCount > 0 && len(senderIDs) > 0 {
		readAt := time.Now()
		payload := map[string]interface{}{
			"type":            "messages_read",
			"conversation_id": conversationIDStr,
			"read_at":         readAt.Format(time.RFC3339),
		}

		// ⚠️ CRITIQUE : Envoyer uniquement aux expéditeurs des messages (pas à tous les participants)
		for _, senderID := range senderIDs {
			// Convertir ObjectID en email pour SendToUser
			senderUser, err := h.userRepo.FindByID(senderID)
			if err != nil || senderUser == nil {
				continue
			}

			// ⚠️ IMPORTANT : Utiliser l'EMAIL de l'expéditeur, pas l'ObjectID
			// Le WebSocket identifie les utilisateurs par leur email
			h.wsHub.SendToUser(senderUser.Email, payload)
		}
	}

	utils.RespondJSON(w, http.StatusOK, models.ChatResponse{
		Success: true,
		Data: map[string]interface{}{
			"marked_read": markedCount,
		},
	})
}

// SendMessage envoie un message dans une conversation
func (h *ChatHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireAdmin(w, r)
	if !ok {
		return
	}
	// Récupérer l'ID de la conversation depuis les paramètres d'URL
	vars := mux.Vars(r)
	conversationIDStr := vars["id"]
	if conversationIDStr == "" {
		utils.RespondError(w, http.StatusBadRequest, constants.ErrConvIDRequired)
		return
	}

	conversationID, err := primitive.ObjectIDFromHex(conversationIDStr)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, constants.ErrInvalidConvID)
		return
	}

	// Vérifier que l'utilisateur fait partie de la conversation
	conversation, err := h.chatRepo.GetConversationByID(r.Context(), conversationID)
	if err != nil {
		utils.RespondError(w, http.StatusNotFound, constants.ErrConvNotFound)
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
		utils.RespondError(w, http.StatusForbidden, constants.ErrConvAccessDenied)
		return
	}

	// Parser le body JSON
	var request models.MessageRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		utils.RespondError(w, http.StatusBadRequest, constants.ErrInvalidJSONBody)
		return
	}

	// Validation
	if strings.TrimSpace(request.Content) == "" {
		utils.RespondError(w, http.StatusBadRequest, constants.ErrMessageContentRequired)
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
		utils.RespondError(w, http.StatusInternalServerError, constants.ErrServerError)
		return
	}

	// Envoyer une notification aux autres participants (FCM)
	go h.sendMessageNotification(conversation, message, userID)

	// 🔌 Envoyer via WebSocket à TOUS les participants (même ceux qui n'ont pas rejoint la room)
	if h.wsHub != nil {

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
					h.wsHub.SendToUser(participantEmail, payload)
				}
			}
		}

	}

	utils.RespondJSON(w, http.StatusOK, models.ChatResponse{
		Success: true,
		Data: map[string]interface{}{
			"message": message,
		},
	})
}

// SearchAdmins recherche des administrateurs
func (h *ChatHandler) SearchAdmins(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireAdmin(w, r); !ok {
		return
	}
	// Récupérer les paramètres de recherche
	query := r.URL.Query().Get("q")
	if query == "" {
		utils.RespondError(w, http.StatusBadRequest, constants.ErrSearchParamRequired)
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
		utils.RespondError(w, http.StatusInternalServerError, constants.ErrServerError)
		return
	}

	utils.RespondJSON(w, http.StatusOK, models.ChatResponse{
		Success: true,
		Data: map[string]interface{}{
			"admins": admins,
		},
	})
}

// SendInvitation envoie une invitation de chat
func (h *ChatHandler) SendInvitation(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireAdmin(w, r)
	if !ok {
		return
	}
	user, _ := h.userRepo.FindByID(userID)

	// Parser le body JSON
	var request models.InvitationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		utils.RespondError(w, http.StatusBadRequest, constants.ErrInvalidJSONBody)
		return
	}

	// Validation
	if strings.TrimSpace(request.ToUserID) == "" {
		utils.RespondError(w, http.StatusBadRequest, constants.ErrRecipientIDRequired)
		return
	}

	if strings.TrimSpace(request.Message) == "" {
		utils.RespondError(w, http.StatusBadRequest, constants.ErrInvitationMessageRequired)
		return
	}

	// Convertir l'ID du destinataire
	toUserID, err := primitive.ObjectIDFromHex(request.ToUserID)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, constants.ErrRecipientIDInvalid)
		return
	}

	// Vérifier que le destinataire est admin
	toUser, err := h.userRepo.FindByID(toUserID)
	if err != nil || toUser.Admin != 1 {
		utils.RespondError(w, http.StatusBadRequest, constants.ErrRecipientMustBeAdmin)
		return
	}

	// Vérifier qu'on ne s'invite pas soi-même
	if userID == toUserID {
		utils.RespondError(w, http.StatusBadRequest, constants.ErrCannotInviteSelf)
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
		utils.RespondError(w, http.StatusInternalServerError, constants.ErrServerError)
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
	}

	utils.RespondJSON(w, http.StatusOK, models.ChatResponse{
		Success: true,
		Data: map[string]interface{}{
			"invitation": invitation,
		},
	})
}

// GetInvitations récupère les invitations reçues
func (h *ChatHandler) GetInvitations(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireAdmin(w, r)
	if !ok {
		return
	}

	// Récupérer les invitations
	invitations, err := h.chatRepo.GetInvitations(r.Context(), userID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, constants.ErrServerError)
		return
	}

	utils.RespondJSON(w, http.StatusOK, models.ChatResponse{
		Success: true,
		Data: map[string]interface{}{
			"invitations": invitations,
		},
	})
}

// RespondToInvitation répond à une invitation
func (h *ChatHandler) RespondToInvitation(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireAdmin(w, r)
	if !ok {
		return
	}
	user, _ := h.userRepo.FindByID(userID)

	// Récupérer l'ID de l'invitation depuis les paramètres d'URL
	vars := mux.Vars(r)
	invitationIDStr := vars["id"]
	if invitationIDStr == "" {
		utils.RespondError(w, http.StatusBadRequest, constants.ErrInvitationIDRequired)
		return
	}

	invitationID, err := primitive.ObjectIDFromHex(invitationIDStr)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, constants.ErrInvitationInvalidID)
		return
	}

	// Parser le body JSON
	var request models.InvitationResponse
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		utils.RespondError(w, http.StatusBadRequest, constants.ErrInvalidJSONBody)
		return
	}

	// Validation
	if request.Action != "accept" && request.Action != "reject" {
		utils.RespondError(w, http.StatusBadRequest, constants.ErrActionAcceptReject)
		return
	}

	// Récupérer l'invitation pour avoir les IDs
	var invitation models.ChatInvitation
	err = h.chatRepo.InvitationCollection.FindOne(r.Context(), bson.M{"_id": invitationID}).Decode(&invitation)
	if err != nil {
		utils.RespondError(w, http.StatusNotFound, constants.ErrInvitationNotFound)
		return
	}

	// Répondre à l'invitation
	conversation, err := h.chatRepo.RespondToInvitation(r.Context(), invitationID, request.Action)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, constants.ErrServerError)
		return
	}

	// Si acceptée, envoyer une notification au demandeur
	if request.Action == "accept" && conversation != nil {
		go h.sendAcceptedInvitationNotification(&invitation, user)

		// 🔌 Envoyer via WebSocket à l'expéditeur
		if h.wsHub != nil {
			// Récupérer l'utilisateur qui a créé l'invitation (expéditeur)
			fromUser, err := h.userRepo.FindByID(invitation.FromUserID)
			if err == nil && fromUser != nil {
				// ⚠️ IMPORTANT : Utiliser l'EMAIL de l'expéditeur, pas l'ObjectID
				// Le WebSocket identifie les utilisateurs par leur email
				h.wsHub.SendToUser(
					fromUser.Email,
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
			}
		}
	} else if request.Action == "reject" {
		// 🔌 Envoyer via WebSocket à l'expéditeur
		if h.wsHub != nil {
			// Récupérer l'utilisateur qui a créé l'invitation (expéditeur)
			fromUser, err := h.userRepo.FindByID(invitation.FromUserID)
			if err == nil && fromUser != nil {
				// ⚠️ IMPORTANT : Utiliser l'EMAIL de l'expéditeur, pas l'ObjectID
				// Le WebSocket identifie les utilisateurs par leur email
				h.wsHub.SendToUser(
					fromUser.Email,
					map[string]interface{}{
						"type":          "invitation_rejected",
						"invitation_id": invitationID.Hex(),
					},
				)
			}
		}
	}

	utils.RespondJSON(w, http.StatusOK, models.ChatResponse{
		Success: true,
		Data: map[string]interface{}{
			"conversation": conversation,
		},
	})
}

// SendChatNotification envoie une notification de chat
func (h *ChatHandler) SendChatNotification(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireAdmin(w, r); !ok {
		return
	}

	// Parser le body JSON
	var request models.ChatNotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		utils.RespondError(w, http.StatusBadRequest, constants.ErrInvalidJSONBody)
		return
	}

	// Validation
	if strings.TrimSpace(request.ToUserID) == "" {
		utils.RespondError(w, http.StatusBadRequest, constants.ErrRecipientIDRequired)
		return
	}

	if strings.TrimSpace(request.Type) == "" {
		utils.RespondError(w, http.StatusBadRequest, constants.ErrNotificationTypeRequired)
		return
	}

	if strings.TrimSpace(request.Title) == "" {
		utils.RespondError(w, http.StatusBadRequest, constants.ErrTitleRequired)
		return
	}

	if strings.TrimSpace(request.Body) == "" {
		utils.RespondError(w, http.StatusBadRequest, constants.ErrBodyRequired)
		return
	}

	// Convertir l'ID du destinataire
	toUserID, err := primitive.ObjectIDFromHex(request.ToUserID)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, constants.ErrRecipientIDInvalid)
		return
	}

	// Envoyer la notification
	if h.fcmService != nil {
		// Récupérer l'utilisateur destinataire pour obtenir son email
		toUser, err := h.userRepo.FindByID(toUserID)
		if err != nil {
			utils.RespondError(w, http.StatusNotFound, constants.ErrRecipientNotFound)
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

	utils.RespondJSON(w, http.StatusOK, models.ChatResponse{
		Success: true,
		Message: "Notification envoyée avec succès",
	})
}

// sendMessageNotification envoie une notification pour un nouveau message
func (h *ChatHandler) sendMessageNotification(conversation *models.Conversation, message *models.Message, senderID primitive.ObjectID) {
	if h.fcmService == nil {
		return
	}

	// Trouver les autres participants
	for _, participant := range conversation.Participants {
		if participant.UserID != senderID {

			// Récupérer les informations de l'expéditeur
			sender, err := h.userRepo.FindByID(senderID)
			if err != nil {
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
				continue
			}

			// Récupérer les tokens FCM du participant (par email)
			fcmTokens, err := h.fcmTokenRepo.FindByUserID(participantUser.Email)
			if err != nil {
				continue
			}

			if len(fcmTokens) > 0 {
				// Convertir les données en map[string]string pour FCM
				fcmData := make(map[string]string)
				for k, v := range data {
					if str, ok := v.(string); ok {
						fcmData[k] = str
					}
				}

				// 🔍 LOGS CRITIQUES - Vérifier que conversationId est bien présent

				// Envoyer à tous les tokens du participant
				for _, token := range fcmTokens {
					_ = h.fcmService.SendToToken(token.Token, title, body, fcmData)
				}
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
			_ = h.fcmService.SendToToken(token.Token, title, body, fcmData)
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
			_ = h.fcmService.SendToToken(token.Token, title, body, fcmData)
		}
	}
}
