package handlers

import (
	"fmt"
	"log"
	"premier-an-backend/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// sendGroupInvitationNotification envoie une notification WebSocket d'invitation
func (h *ChatGroupHandler) sendGroupInvitationNotification(group *models.ChatGroup, invitation *models.ChatGroupInvitation, invitedUser *models.User) {
	// Récupérer les infos de l'inviteur
	inviter, err := h.userRepo.FindByEmail(invitation.InvitedBy)
	if err != nil || inviter == nil {
		log.Printf("Erreur récupération inviteur: %v", err)
		return
	}

	// Récupérer le créateur du groupe
	creator, err := h.userRepo.FindByEmail(group.CreatedBy)
	if err != nil || creator == nil {
		log.Printf("Erreur récupération créateur: %v", err)
		return
	}

	// Compter les membres
	memberCount, _ := h.groupRepo.GetMemberCount(group.ID)

	// Préparer le payload
	payload := map[string]interface{}{
		"type": "group_invitation",
		"invitation": map[string]interface{}{
			"id": invitation.ID.Hex(),
			"group": map[string]interface{}{
				"id":   group.ID.Hex(),
				"name": group.Name,
				"created_by": map[string]interface{}{
					"firstname": creator.Firstname,
					"lastname":  creator.Lastname,
				},
				"member_count": memberCount,
			},
			"invited_by": map[string]interface{}{
				"firstname": inviter.Firstname,
				"lastname":  inviter.Lastname,
			},
			"message":    invitation.Message,
			"invited_at": invitation.InvitedAt,
		},
	}

	// Envoyer via WebSocket (JSON direct)
	h.wsHub.SendToUser(invitation.InvitedUser, payload)

	log.Printf("📨 Notification WebSocket envoyée: group_invitation à %s", invitation.InvitedUser)
}

// sendGroupInvitationFCM envoie une notification FCM d'invitation
func (h *ChatGroupHandler) sendGroupInvitationFCM(group *models.ChatGroup, invitedUser *models.User) {
	// Récupérer les tokens FCM de l'utilisateur invité
	tokens, err := h.fcmTokenRepo.FindByUserID(invitedUser.Email)
	if err != nil || len(tokens) == 0 {
		return
	}

	// Récupérer les infos du créateur
	creator, _ := h.userRepo.FindByEmail(group.CreatedBy)
	if creator == nil {
		return
	}

	title := "📨 Nouvelle invitation de groupe"
	message := fmt.Sprintf("%s %s vous invite à rejoindre \"%s\"", creator.Firstname, creator.Lastname, group.Name)

	data := map[string]string{
		"type":       "group_invitation",
		"group_id":   group.ID.Hex(),
		"group_name": group.Name,
	}

	// Envoyer à tous les tokens de l'utilisateur
	for _, token := range tokens {
		if err := h.fcmService.SendToToken(token.Token, title, message, data); err != nil {
			log.Printf("❌ Erreur envoi FCM: %v", err)
		}
	}

	log.Printf("📱 Notification FCM envoyée: group_invitation à %s", invitedUser.Email)
}

// notifyInvitationAccepted notifie l'admin que l'invitation a été acceptée
func (h *ChatGroupHandler) notifyInvitationAccepted(invitation *models.ChatGroupInvitation, user *models.User, group *models.ChatGroup) {
	payload := map[string]interface{}{
		"type":     "group_invitation_accepted",
		"group_id": group.ID.Hex(),
		"user": map[string]interface{}{
			"id":        user.Email,
			"firstname": user.Firstname,
			"lastname":  user.Lastname,
		},
		"accepted_at": invitation.RespondedAt,
	}

	h.wsHub.SendToUser(invitation.InvitedBy, payload)

	log.Printf("📨 Notification: invitation acceptée par %s (groupe: %s)", user.Email, group.Name)
}

// notifyInvitationRejected notifie l'admin que l'invitation a été refusée
func (h *ChatGroupHandler) notifyInvitationRejected(invitation *models.ChatGroupInvitation, user *models.User, group *models.ChatGroup) {
	payload := map[string]interface{}{
		"type":     "group_invitation_rejected",
		"group_id": group.ID.Hex(),
		"user": map[string]interface{}{
			"id":        user.Email,
			"firstname": user.Firstname,
			"lastname":  user.Lastname,
		},
		"rejected_at": invitation.RespondedAt,
	}

	h.wsHub.SendToUser(invitation.InvitedBy, payload)

	log.Printf("📨 Notification: invitation refusée par %s (groupe: %s)", user.Email, group.Name)
}

// broadcastMemberJoined diffuse à tous les membres qu'un nouveau membre a rejoint
func (h *ChatGroupHandler) broadcastMemberJoined(group *models.ChatGroup, user *models.User, systemMessage *models.ChatGroupMessage) {
	// Récupérer tous les membres du groupe
	members, err := h.groupRepo.GetMembers(group.ID)
	if err != nil {
		log.Printf("Erreur récupération membres: %v", err)
		return
	}

	payload := map[string]interface{}{
		"type":     "group_member_joined",
		"group_id": group.ID.Hex(),
		"user": map[string]interface{}{
			"id":        user.Email,
			"firstname": user.Firstname,
			"lastname":  user.Lastname,
			"email":     user.Email,
		},
		"system_message": map[string]interface{}{
			"id":           systemMessage.ID.Hex(),
			"content":      systemMessage.Content,
			"message_type": systemMessage.MessageType,
			"created_at":   systemMessage.CreatedAt,
		},
	}

	// Envoyer à tous les membres (JSON direct)
	// ✅ member.ID est maintenant l'email (corrigé dans GetMembers)
	for _, member := range members {
		log.Printf("📤 Envoi WS member_joined à %s", member.ID)
		h.wsHub.SendToUser(member.ID, payload) // ✅ Utiliser ID (qui est l'email)
	}

	log.Printf("📨 Notification diffusée: member_joined dans groupe %s", group.Name)
}

// broadcastGroupMessage diffuse un nouveau message à tous les membres connectés du groupe
func (h *ChatGroupHandler) broadcastGroupMessage(groupID primitive.ObjectID, message *models.GroupMessageWithSender) {
	// Récupérer tous les membres du groupe
	members, err := h.groupRepo.GetMembers(groupID)
	if err != nil {
		log.Printf("Erreur récupération membres: %v", err)
		return
	}

	payload := map[string]interface{}{
		"type":     "new_group_message",
		"group_id": groupID.Hex(),
		"message":  message,
	}

	// Envoyer à tous les membres sauf l'expéditeur (JSON direct)
	// ✅ member.ID est maintenant l'email (corrigé dans GetMembers)
	for _, member := range members {
		if member.ID != message.SenderID { // ✅ Comparer avec ID (qui est maintenant l'email)
			log.Printf("📤 Envoi WS new_group_message à %s", member.ID)
			h.wsHub.SendToUser(member.ID, payload) // ✅ Utiliser ID (qui est l'email)
		}
	}

	log.Printf("📨 Message diffusé dans le groupe %s", groupID.Hex())
}

// sendGroupMessageFCM envoie une notification FCM pour un nouveau message
func (h *ChatGroupHandler) sendGroupMessageFCM(group *models.ChatGroup, sender *models.User, message *models.ChatGroupMessage) {
	// Récupérer tous les membres du groupe
	members, err := h.groupRepo.GetMembers(group.ID)
	if err != nil {
		return
	}

	// Collecter les tokens de tous les membres (sauf l'expéditeur)
	var tokens []string
	for _, member := range members {
		if member.ID == sender.Email {
			continue
		}

		memberTokens, err := h.fcmTokenRepo.FindByUserID(member.ID)
		if err != nil || len(memberTokens) == 0 {
			continue
		}

		for _, token := range memberTokens {
			tokens = append(tokens, token.Token)
		}
	}

	if len(tokens) == 0 {
		return
	}

	// Préparer la notification
	title := fmt.Sprintf("👥 %s", group.Name)
	body := fmt.Sprintf("%s: %s", sender.Firstname, message.Content)

	// Limiter le corps du message
	if len(body) > 100 {
		body = body[:97] + "..."
	}

	data := map[string]string{
		"type":        "group_message",
		"group_id":    group.ID.Hex(),
		"message_id":  message.ID.Hex(),
		"sender_name": fmt.Sprintf("%s %s", sender.Firstname, sender.Lastname),
	}

	// Envoyer via FCM
	success, failed, _ := h.fcmService.SendToAll(tokens, title, body, data)
	log.Printf("📱 FCM group message: %d succès, %d échecs", success, failed)
}

// broadcastMessagesRead notifie que des messages ont été lus
func (h *ChatGroupHandler) broadcastMessagesRead(groupID primitive.ObjectID, userID string) {
	// Récupérer tous les membres du groupe
	members, err := h.groupRepo.GetMembers(groupID)
	if err != nil {
		return
	}

	payload := map[string]interface{}{
		"type":     "group_messages_read",
		"group_id": groupID.Hex(),
		"user_id":  userID,
		"read_at":  models.FlexibleTime{},
	}

	// Envoyer à tous les membres sauf celui qui a lu (JSON direct)
	// ✅ member.ID est maintenant l'email (corrigé dans GetMembers)
	for _, member := range members {
		if member.ID != userID { // ✅ Comparer avec ID (qui est maintenant l'email)
			log.Printf("📤 Envoi WS messages_read à %s", member.ID)
			h.wsHub.SendToUser(member.ID, payload) // ✅ Utiliser ID (qui est l'email)
		}
	}
}
