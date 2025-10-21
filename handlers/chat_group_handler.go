package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"premier-an-backend/database"
	"premier-an-backend/middleware"
	"premier-an-backend/models"
	"premier-an-backend/services"
	"premier-an-backend/utils"
	"premier-an-backend/websocket"
	"strconv"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// ChatGroupHandler gère les requêtes de groupes de chat
type ChatGroupHandler struct {
	groupRepo      *database.ChatGroupRepository
	invitationRepo *database.ChatGroupInvitationRepository
	messageRepo    *database.ChatGroupMessageRepository
	userRepo       *database.UserRepository
	fcmTokenRepo   *database.FCMTokenRepository
	fcmService     *services.FCMService
	wsHub          *websocket.Hub
}

// NewChatGroupHandler crée une nouvelle instance
func NewChatGroupHandler(
	db *mongo.Database,
	fcmService *services.FCMService,
	wsHub *websocket.Hub,
) *ChatGroupHandler {
	return &ChatGroupHandler{
		groupRepo:      database.NewChatGroupRepository(db),
		invitationRepo: database.NewChatGroupInvitationRepository(db),
		messageRepo:    database.NewChatGroupMessageRepository(db),
		userRepo:       database.NewUserRepository(db),
		fcmTokenRepo:   database.NewFCMTokenRepository(db),
		fcmService:     fcmService,
		wsHub:          wsHub,
	}
}

// CreateGroup crée un nouveau groupe
func (h *ChatGroupHandler) CreateGroup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.RespondError(w, http.StatusMethodNotAllowed, "Méthode non autorisée")
		return
	}

	// Récupérer l'utilisateur authentifié
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Non authentifié")
		return
	}

	var req models.CreateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Données invalides")
		return
	}

	// Valider
	if req.Name == "" {
		utils.RespondError(w, http.StatusBadRequest, "Le nom du groupe est requis")
		return
	}

	// Créer le groupe
	group := &models.ChatGroup{
		Name:      req.Name,
		CreatedBy: claims.UserID,
	}

	if err := h.groupRepo.Create(group); err != nil {
		log.Printf("Erreur création groupe: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur lors de la création du groupe")
		return
	}

	// Ajouter le créateur comme admin
	member := &models.ChatGroupMember{
		GroupID: group.ID,
		UserID:  claims.UserID,
		Role:    "admin",
	}

	if err := h.groupRepo.AddMember(member); err != nil {
		log.Printf("Erreur ajout admin: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur lors de l'ajout du créateur")
		return
	}

	// Créer les invitations pour les membres
	for _, memberID := range req.MemberIDs {
		// Vérifier que l'utilisateur existe
		user, err := h.userRepo.FindByEmail(memberID)
		if err != nil || user == nil {
			log.Printf("Utilisateur non trouvé: %s", memberID)
			continue
		}

		// Créer l'invitation
		invitation := &models.ChatGroupInvitation{
			GroupID:     group.ID,
			InvitedBy:   claims.UserID,
			InvitedUser: memberID,
		}

		if err := h.invitationRepo.Create(invitation); err != nil {
			log.Printf("Erreur création invitation: %v", err)
			continue
		}

		// Envoyer notification WebSocket
		h.sendGroupInvitationNotification(group, invitation, user)

		// Envoyer notification FCM
		h.sendGroupInvitationFCM(group, user)
	}

	// Compter les membres
	memberCount, _ := h.groupRepo.GetMemberCount(group.ID)

	log.Printf("✓ Groupe créé: %s par %s", group.Name, claims.UserID)
	utils.RespondSuccess(w, "Groupe créé avec succès", map[string]interface{}{
		"id":           group.ID.Hex(),
		"name":         group.Name,
		"created_by":   group.CreatedBy,
		"created_at":   group.CreatedAt,
		"member_count": memberCount,
	})
}

// GetGroups récupère tous les groupes de l'utilisateur
func (h *ChatGroupHandler) GetGroups(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.RespondError(w, http.StatusMethodNotAllowed, "Méthode non autorisée")
		return
	}

	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Non authentifié")
		return
	}

	// Utiliser la nouvelle méthode qui retourne tout enrichi
	groups, err := h.groupRepo.GetUserGroups(claims.Email, h.messageRepo.Collection())
	if err != nil {
		log.Printf("Erreur récupération groupes: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}

	utils.RespondSuccess(w, "Groupes récupérés", map[string]interface{}{
		"groups": groups,
	})
}

// InviteToGroup invite un utilisateur dans un groupe
func (h *ChatGroupHandler) InviteToGroup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.RespondError(w, http.StatusMethodNotAllowed, "Méthode non autorisée")
		return
	}

	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Non authentifié")
		return
	}

	// Récupérer l'ID du groupe
	vars := mux.Vars(r)
	groupID, err := primitive.ObjectIDFromHex(vars["group_id"])
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "ID de groupe invalide")
		return
	}

	var req models.InviteToGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Données invalides")
		return
	}

	// Vérifier que l'utilisateur est admin du groupe
	isAdmin, err := h.groupRepo.IsAdmin(groupID, claims.UserID)
	if err != nil || !isAdmin {
		utils.RespondError(w, http.StatusForbidden, "Seuls les admins peuvent inviter")
		return
	}

	// Vérifier que le groupe existe
	group, err := h.groupRepo.FindByID(groupID)
	if err != nil || group == nil {
		utils.RespondError(w, http.StatusNotFound, "Groupe non trouvé")
		return
	}

	// Vérifier que l'utilisateur existe
	user, err := h.userRepo.FindByEmail(req.UserID)
	if err != nil || user == nil {
		utils.RespondError(w, http.StatusNotFound, "Utilisateur non trouvé")
		return
	}

	// Vérifier que l'utilisateur n'est pas déjà membre
	isMember, _ := h.groupRepo.IsMember(groupID, req.UserID)
	if isMember {
		utils.RespondError(w, http.StatusConflict, "Cet utilisateur est déjà membre")
		return
	}

	// Vérifier qu'il n'y a pas déjà une invitation en attente
	hasPending, _ := h.invitationRepo.HasPendingInvitation(groupID, req.UserID)
	if hasPending {
		utils.RespondError(w, http.StatusConflict, "Une invitation est déjà en attente")
		return
	}

	// Créer l'invitation
	invitation := &models.ChatGroupInvitation{
		GroupID:     groupID,
		InvitedBy:   claims.UserID,
		InvitedUser: req.UserID,
		Message:     req.Message,
	}

	if err := h.invitationRepo.Create(invitation); err != nil {
		log.Printf("Erreur création invitation: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}

	// Envoyer notifications
	h.sendGroupInvitationNotification(group, invitation, user)
	h.sendGroupInvitationFCM(group, user)

	log.Printf("✓ Invitation envoyée: %s -> %s (groupe: %s)", claims.UserID, req.UserID, group.Name)
	utils.RespondSuccess(w, "Invitation envoyée", map[string]interface{}{
		"id":           invitation.ID.Hex(),
		"group_id":     groupID.Hex(),
		"invited_user": req.UserID,
		"status":       invitation.Status,
		"invited_at":   invitation.InvitedAt,
	})
}

// GetPendingInvitations récupère les invitations en attente de l'utilisateur
func (h *ChatGroupHandler) GetPendingInvitations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.RespondError(w, http.StatusMethodNotAllowed, "Méthode non autorisée")
		return
	}

	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Non authentifié")
		return
	}

	// ✅ Utiliser l'email au lieu de l'ID (invited_user est un email dans la DB)
	invitations, err := h.invitationRepo.FindPendingByUser(claims.Email)
	if err != nil {
		log.Printf("Erreur récupération invitations: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}

	utils.RespondSuccess(w, "Invitations récupérées", map[string]interface{}{
		"invitations": invitations,
	})
}

// RespondToInvitation répond à une invitation (accepter/refuser)
func (h *ChatGroupHandler) RespondToInvitation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.RespondError(w, http.StatusMethodNotAllowed, "Méthode non autorisée")
		return
	}

	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Non authentifié")
		return
	}

	// Récupérer l'ID de l'invitation
	vars := mux.Vars(r)
	invitationID, err := primitive.ObjectIDFromHex(vars["invitation_id"])
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "ID d'invitation invalide")
		return
	}

	var req models.RespondToGroupInvitationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Données invalides")
		return
	}

	// Valider l'action
	if req.Action != "accept" && req.Action != "reject" {
		utils.RespondError(w, http.StatusBadRequest, "Action invalide (accept ou reject)")
		return
	}

	// Récupérer l'invitation
	invitation, err := h.invitationRepo.FindByID(invitationID)
	if err != nil || invitation == nil {
		utils.RespondError(w, http.StatusNotFound, "Invitation non trouvée")
		return
	}

	// Vérifier que c'est bien l'utilisateur invité
	if invitation.InvitedUser != claims.UserID {
		utils.RespondError(w, http.StatusForbidden, "Cette invitation ne vous est pas destinée")
		return
	}

	// Vérifier que l'invitation est en attente
	if invitation.Status != "pending" {
		utils.RespondError(w, http.StatusConflict, "Cette invitation a déjà été traitée")
		return
	}

	// Récupérer le groupe
	group, err := h.groupRepo.FindByID(invitation.GroupID)
	if err != nil || group == nil {
		utils.RespondError(w, http.StatusNotFound, "Groupe non trouvé")
		return
	}

	// Traiter selon l'action
	if req.Action == "accept" {
		h.acceptInvitation(w, invitation, group, claims.UserID)
	} else {
		h.rejectInvitation(w, invitation, group, claims.UserID)
	}
}

// acceptInvitation accepte une invitation
func (h *ChatGroupHandler) acceptInvitation(w http.ResponseWriter, invitation *models.ChatGroupInvitation, group *models.ChatGroup, userID string) {
	// Mettre à jour l'invitation
	if err := h.invitationRepo.UpdateStatus(invitation.ID, "accepted"); err != nil {
		log.Printf("Erreur mise à jour invitation: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}

	// Ajouter l'utilisateur comme membre
	member := &models.ChatGroupMember{
		GroupID: invitation.GroupID,
		UserID:  userID,
		Role:    "member",
	}

	if err := h.groupRepo.AddMember(member); err != nil {
		log.Printf("Erreur ajout membre: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}

	// Récupérer les infos de l'utilisateur
	user, _ := h.userRepo.FindByEmail(userID)

	// Créer un message système
	systemMessage := &models.ChatGroupMessage{
		GroupID:     invitation.GroupID,
		SenderID:    "system",
		Content:     fmt.Sprintf("%s %s a rejoint le groupe", user.Firstname, user.Lastname),
		MessageType: "system",
	}

	if err := h.messageRepo.Create(systemMessage); err != nil {
		log.Printf("Erreur création message système: %v", err)
	}

	// Notifier tous les membres du groupe via WebSocket
	h.broadcastMemberJoined(group, user, systemMessage)

	// Notifier l'admin qui a invité
	h.notifyInvitationAccepted(invitation, user, group)

	log.Printf("✓ Invitation acceptée: %s a rejoint %s", userID, group.Name)
	utils.RespondSuccess(w, "Invitation acceptée", map[string]interface{}{
		"group": map[string]interface{}{
			"id":   group.ID.Hex(),
			"name": group.Name,
		},
	})
}

// rejectInvitation refuse une invitation
func (h *ChatGroupHandler) rejectInvitation(w http.ResponseWriter, invitation *models.ChatGroupInvitation, group *models.ChatGroup, userID string) {
	// Mettre à jour l'invitation
	if err := h.invitationRepo.UpdateStatus(invitation.ID, "rejected"); err != nil {
		log.Printf("Erreur mise à jour invitation: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}

	// Récupérer les infos de l'utilisateur
	user, _ := h.userRepo.FindByEmail(userID)

	// Notifier UNIQUEMENT l'admin qui a invité (silencieux)
	h.notifyInvitationRejected(invitation, user, group)

	log.Printf("✓ Invitation refusée: %s a refusé %s", userID, group.Name)
	utils.RespondSuccess(w, "Invitation refusée", nil)
}

// GetGroupMembers récupère les membres d'un groupe
func (h *ChatGroupHandler) GetGroupMembers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.RespondError(w, http.StatusMethodNotAllowed, "Méthode non autorisée")
		return
	}

	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Non authentifié")
		return
	}

	vars := mux.Vars(r)
	groupID, err := primitive.ObjectIDFromHex(vars["group_id"])
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "ID de groupe invalide")
		return
	}

	// Vérifier que l'utilisateur est membre
	isMember, err := h.groupRepo.IsMember(groupID, claims.UserID)
	if err != nil || !isMember {
		utils.RespondError(w, http.StatusForbidden, "Vous n'êtes pas membre de ce groupe")
		return
	}

	// Récupérer les membres
	members, err := h.groupRepo.GetMembers(groupID)
	if err != nil {
		log.Printf("Erreur récupération membres: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}

	utils.RespondSuccess(w, "Membres récupérés", map[string]interface{}{
		"members": members,
	})
}

// GetGroupPendingInvitations récupère les invitations en attente d'un groupe (admin seulement)
func (h *ChatGroupHandler) GetGroupPendingInvitations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.RespondError(w, http.StatusMethodNotAllowed, "Méthode non autorisée")
		return
	}

	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Non authentifié")
		return
	}

	vars := mux.Vars(r)
	groupID, err := primitive.ObjectIDFromHex(vars["group_id"])
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "ID de groupe invalide")
		return
	}

	// Vérifier que l'utilisateur est admin
	isAdmin, err := h.groupRepo.IsAdmin(groupID, claims.UserID)
	if err != nil || !isAdmin {
		utils.RespondError(w, http.StatusForbidden, "Seuls les admins peuvent voir les invitations")
		return
	}

	// Récupérer les invitations
	invitations, err := h.invitationRepo.FindPendingByGroup(groupID)
	if err != nil {
		log.Printf("Erreur récupération invitations: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}

	utils.RespondSuccess(w, "Invitations récupérées", map[string]interface{}{
		"invitations": invitations,
	})
}

// CancelInvitation annule une invitation (admin seulement)
func (h *ChatGroupHandler) CancelInvitation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.RespondError(w, http.StatusMethodNotAllowed, "Méthode non autorisée")
		return
	}

	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Non authentifié")
		return
	}

	vars := mux.Vars(r)
	invitationID, err := primitive.ObjectIDFromHex(vars["invitation_id"])
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "ID d'invitation invalide")
		return
	}

	// Récupérer l'invitation
	invitation, err := h.invitationRepo.FindByID(invitationID)
	if err != nil || invitation == nil {
		utils.RespondError(w, http.StatusNotFound, "Invitation non trouvée")
		return
	}

	// Vérifier que l'utilisateur est admin du groupe
	isAdmin, err := h.groupRepo.IsAdmin(invitation.GroupID, claims.UserID)
	if err != nil || !isAdmin {
		utils.RespondError(w, http.StatusForbidden, "Seuls les admins peuvent annuler des invitations")
		return
	}

	// Supprimer l'invitation
	if err := h.invitationRepo.Delete(invitationID); err != nil {
		log.Printf("Erreur suppression invitation: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}

	log.Printf("✓ Invitation annulée: %s", invitationID.Hex())
	utils.RespondSuccess(w, "Invitation annulée", nil)
}

// SendMessage envoie un message dans un groupe
func (h *ChatGroupHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.RespondError(w, http.StatusMethodNotAllowed, "Méthode non autorisée")
		return
	}

	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Non authentifié")
		return
	}

	vars := mux.Vars(r)
	groupID, err := primitive.ObjectIDFromHex(vars["group_id"])
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "ID de groupe invalide")
		return
	}

	var req models.SendGroupMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Données invalides")
		return
	}

	// Valider
	if req.Content == "" {
		utils.RespondError(w, http.StatusBadRequest, "Le contenu est requis")
		return
	}

	// Vérifier que l'utilisateur est membre
	isMember, err := h.groupRepo.IsMember(groupID, claims.UserID)
	if err != nil || !isMember {
		utils.RespondError(w, http.StatusForbidden, "Vous n'êtes pas membre de ce groupe")
		return
	}

	// Créer le message
	message := &models.ChatGroupMessage{
		GroupID:     groupID,
		SenderID:    claims.UserID,
		Content:     req.Content,
		MessageType: "message",
	}

	if err := h.messageRepo.Create(message); err != nil {
		log.Printf("Erreur création message: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}

	// Récupérer les infos de l'expéditeur
	sender, _ := h.userRepo.FindByEmail(claims.UserID)

	messageWithSender := models.GroupMessageWithSender{
		ID:          message.ID,
		SenderID:    message.SenderID,
		Content:     message.Content,
		MessageType: message.MessageType,
		CreatedAt:   message.CreatedAt,
	}

	if sender != nil {
		messageWithSender.Sender = &models.UserBasicInfo{
			ID:        sender.Email,
			Firstname: sender.Firstname,
			Lastname:  sender.Lastname,
		}
	}

	// Diffuser via WebSocket
	h.broadcastGroupMessage(groupID, &messageWithSender)

	// Envoyer FCM aux membres non connectés
	group, _ := h.groupRepo.FindByID(groupID)
	if group != nil {
		h.sendGroupMessageFCM(group, sender, message)
	}

	log.Printf("✓ Message envoyé dans le groupe %s par %s", groupID.Hex(), claims.UserID)
	utils.RespondSuccess(w, "Message envoyé", messageWithSender)
}

// GetMessages récupère les messages d'un groupe
func (h *ChatGroupHandler) GetMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.RespondError(w, http.StatusMethodNotAllowed, "Méthode non autorisée")
		return
	}

	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Non authentifié")
		return
	}

	vars := mux.Vars(r)
	groupID, err := primitive.ObjectIDFromHex(vars["group_id"])
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "ID de groupe invalide")
		return
	}

	// Vérifier que l'utilisateur est membre
	isMember, err := h.groupRepo.IsMember(groupID, claims.UserID)
	if err != nil || !isMember {
		utils.RespondError(w, http.StatusForbidden, "Vous n'êtes pas membre de ce groupe")
		return
	}

	// Paramètres de pagination
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	// Before (pour pagination)
	var before *primitive.ObjectID
	beforeStr := r.URL.Query().Get("before")
	if beforeStr != "" {
		if beforeID, err := primitive.ObjectIDFromHex(beforeStr); err == nil {
			before = &beforeID
		}
	}

	// Récupérer les messages
	messages, err := h.messageRepo.FindByGroupID(groupID, limit, before)
	if err != nil {
		log.Printf("Erreur récupération messages: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}

	utils.RespondSuccess(w, "Messages récupérés", map[string]interface{}{
		"messages": messages,
	})
}

// MarkAsRead marque les messages d'un groupe comme lus
func (h *ChatGroupHandler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.RespondError(w, http.StatusMethodNotAllowed, "Méthode non autorisée")
		return
	}

	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Non authentifié")
		return
	}

	vars := mux.Vars(r)
	groupID, err := primitive.ObjectIDFromHex(vars["group_id"])
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "ID de groupe invalide")
		return
	}

	// Vérifier que l'utilisateur est membre
	isMember, err := h.groupRepo.IsMember(groupID, claims.UserID)
	if err != nil || !isMember {
		utils.RespondError(w, http.StatusForbidden, "Vous n'êtes pas membre de ce groupe")
		return
	}

	// Marquer comme lu
	if err := h.messageRepo.MarkAsRead(groupID, claims.UserID); err != nil {
		log.Printf("Erreur marquage comme lu: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}

	// Notifier les autres membres via WebSocket
	h.broadcastMessagesRead(groupID, claims.UserID)

	log.Printf("✓ Messages marqués comme lus dans le groupe %s par %s", groupID.Hex(), claims.UserID)
	utils.RespondSuccess(w, "Messages marqués comme lus", nil)
}

// SearchUsers recherche des utilisateurs (pour inviter)
func (h *ChatGroupHandler) SearchUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.RespondError(w, http.StatusMethodNotAllowed, "Méthode non autorisée")
		return
	}

	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Non authentifié")
		return
	}

	query := r.URL.Query().Get("q")
	if len(query) < 2 {
		utils.RespondError(w, http.StatusBadRequest, "La recherche doit contenir au moins 2 caractères")
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 10
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	// Rechercher les utilisateurs
	users, err := h.userRepo.SearchUsers(query, limit, claims.UserID)
	if err != nil {
		log.Printf("Erreur recherche utilisateurs: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, "Erreur serveur")
		return
	}

	// Formater les résultats
	results := make([]map[string]interface{}, 0)
	for _, user := range users {
		results = append(results, map[string]interface{}{
			"id":        user.Email,
			"firstname": user.Firstname,
			"lastname":  user.Lastname,
			"email":     user.Email,
			"admin":     user.Admin,
		})
	}

	utils.RespondSuccess(w, "Utilisateurs trouvés", map[string]interface{}{
		"users": results,
	})
}
