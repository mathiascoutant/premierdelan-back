package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"premier-an-backend/constants"
	"premier-an-backend/database"
	"premier-an-backend/middleware"
	"premier-an-backend/models"
	"premier-an-backend/services"
	"premier-an-backend/utils"
	"premier-an-backend/websocket"
	"strconv"
	"strings"

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
		utils.RespondError(w, http.StatusMethodNotAllowed, constants.ErrMethodNotAllowed)
		return
	}

	// Récupérer l'utilisateur authentifié
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		utils.RespondError(w, http.StatusUnauthorized, constants.ErrNotAuthenticated)
		return
	}

	var req models.CreateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, constants.ErrInvalidData)
		return
	}

	// Valider
	if req.Name == "" {
		utils.RespondError(w, http.StatusBadRequest, "Le nom du groupe est requis")
		return
	}

	// Créer le groupe (created_by en DB est un email)
	group := &models.ChatGroup{
		Name:      req.Name,
		CreatedBy: claims.Email,
	}

	if err := h.groupRepo.Create(group); err != nil {
		log.Printf("Erreur création groupe: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, constants.ErrCreateGroup)
		return
	}

	// ✅ IMPORTANT : Ajouter le créateur comme admin (user_id en DB est un email)
	// Vérifier d'abord si le créateur n'est pas déjà membre (au cas où)
	isAlreadyMember, _ := h.groupRepo.IsMember(group.ID, claims.Email)
	if !isAlreadyMember {
		member := &models.ChatGroupMember{
			GroupID: group.ID,
			UserID:  claims.Email,
			Role:    "admin",
		}

		if err := h.groupRepo.AddMember(member); err != nil {
			log.Printf("❌ Erreur ajout créateur comme membre: %v", err)
			utils.RespondError(w, http.StatusInternalServerError, constants.ErrAddCreator)
			return
		}

		// Vérifier que l'ajout s'est bien passé
		isMember, err := h.groupRepo.IsMember(group.ID, claims.Email)
		if err != nil || !isMember {
			log.Printf("❌ Le créateur n'a pas été ajouté comme membre (vérification échouée)")
			utils.RespondError(w, http.StatusInternalServerError, constants.ErrCheckCreator)
			return
		}
		log.Println("Créateur ajouté comme membre admin du groupe")
	} else {
		log.Printf("⚠️ Le créateur est déjà membre du groupe (cas improbable)")
	}

	// Créer les invitations pour les membres
	for _, memberID := range req.MemberIDs {
		// Vérifier que l'utilisateur existe
		user, err := h.userRepo.FindByEmail(memberID)
		if err != nil || user == nil {
			log.Println("Utilisateur non trouvé")
			continue
		}

		// Créer l'invitation (invited_by en DB est un email)
		invitation := &models.ChatGroupInvitation{
			GroupID:     group.ID,
			InvitedBy:   claims.Email,
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

	// 🔌 Faire rejoindre automatiquement le créateur à la room WebSocket du groupe
	if h.wsHub != nil {
		h.wsHub.JoinGroup(claims.Email, group.ID.Hex())
	}

	// 🔌 Envoyer un événement WebSocket au créateur avec le format GroupWithDetails
	if h.wsHub != nil {
		// Récupérer les infos du créateur
		creator, err := h.userRepo.FindByEmail(claims.Email)
		if err == nil && creator != nil {
			// Construire un GroupWithDetails complet comme dans GetUserGroups
			groupDetails := map[string]interface{}{
				"id":           group.ID.Hex(),
				"name":         group.Name,
				"member_count": memberCount,
				"unread_count": 0,
				"created_at":   group.CreatedAt,
				"created_by": map[string]interface{}{
					"id":        creator.Email,
					"firstname": creator.Firstname,
					"lastname":  creator.Lastname,
					"email":     creator.Email,
				},
				"last_message": nil, // Pas de message au moment de la création
			}

			payload := map[string]interface{}{
				"type":  "group_created",
				"group": groupDetails,
			}
			h.wsHub.SendToUser(claims.Email, payload)
		}
	}

	log.Println("Groupe créé")
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
		utils.RespondError(w, http.StatusMethodNotAllowed, constants.ErrMethodNotAllowed)
		return
	}

	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		utils.RespondError(w, http.StatusUnauthorized, constants.ErrNotAuthenticated)
		return
	}

	// Utiliser la nouvelle méthode qui retourne tout enrichi
	groups, err := h.groupRepo.GetUserGroups(claims.Email, h.messageRepo.Collection())
	if err != nil {
		log.Printf("Erreur récupération groupes: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, constants.ErrServerError)
		return
	}

	utils.RespondSuccess(w, "Groupes récupérés", map[string]interface{}{
		"groups": groups,
	})
}

// InviteToGroup invite un utilisateur dans un groupe
func (h *ChatGroupHandler) InviteToGroup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.RespondError(w, http.StatusMethodNotAllowed, constants.ErrMethodNotAllowed)
		return
	}

	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		utils.RespondError(w, http.StatusUnauthorized, constants.ErrNotAuthenticated)
		return
	}

	// Récupérer l'ID du groupe
	vars := mux.Vars(r)
	groupID, err := primitive.ObjectIDFromHex(vars["group_id"])
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, constants.ErrInvalidGroupID)
		return
	}

	var req models.InviteToGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, constants.ErrInvalidData)
		return
	}

	// Vérifier que l'utilisateur est membre du groupe (tous les membres peuvent inviter)
	isMember, err := h.groupRepo.IsMember(groupID, claims.Email)
	if err != nil || !isMember {
		utils.RespondError(w, http.StatusForbidden, constants.ErrInviteMembersOnly)
		return
	}

	// Vérifier que le groupe existe
	group, err := h.groupRepo.FindByID(groupID)
	if err != nil || group == nil {
		utils.RespondError(w, http.StatusNotFound, constants.ErrGroupNotFound)
		return
	}

	// Vérifier que l'utilisateur existe
	user, err := h.userRepo.FindByEmail(req.UserID)
	if err != nil || user == nil {
		utils.RespondError(w, http.StatusNotFound, "Utilisateur non trouvé")
		return
	}

	// Vérifier que l'utilisateur n'est pas déjà membre
	isAlreadyMember, _ := h.groupRepo.IsMember(groupID, req.UserID)
	if isAlreadyMember {
		utils.RespondError(w, http.StatusConflict, constants.ErrUserAlreadyMember)
		return
	}

	// Vérifier qu'il n'y a pas déjà une invitation en attente
	hasPending, _ := h.invitationRepo.HasPendingInvitation(groupID, req.UserID)
	if hasPending {
		utils.RespondError(w, http.StatusConflict, constants.ErrInvitationPending)
		return
	}

	// Créer l'invitation (invited_by en DB est un email)
	invitation := &models.ChatGroupInvitation{
		GroupID:     groupID,
		InvitedBy:   claims.Email,
		InvitedUser: req.UserID,
		Message:     req.Message,
	}

	if err := h.invitationRepo.Create(invitation); err != nil {
		log.Printf("Erreur création invitation: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, constants.ErrServerError)
		return
	}

	// Envoyer notifications
	h.sendGroupInvitationNotification(group, invitation, user)
	h.sendGroupInvitationFCM(group, user)

	log.Printf("Invitation envoyée (groupe: %s)", group.Name)
	utils.RespondSuccess(w, "Invitation envoyée", map[string]interface{}{
		"invitation_id": invitation.ID.Hex(),
	})
}

// GetPendingInvitations récupère les invitations en attente de l'utilisateur
func (h *ChatGroupHandler) GetPendingInvitations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.RespondError(w, http.StatusMethodNotAllowed, constants.ErrMethodNotAllowed)
		return
	}

	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		utils.RespondError(w, http.StatusUnauthorized, constants.ErrNotAuthenticated)
		return
	}

	// ✅ Utiliser l'email au lieu de l'ID (invited_user est un email dans la DB)
	invitations, err := h.invitationRepo.FindPendingByUser(claims.Email)
	if err != nil {
		log.Printf("Erreur récupération invitations: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, constants.ErrServerError)
		return
	}

	utils.RespondSuccess(w, "Invitations récupérées", map[string]interface{}{
		"invitations": invitations,
	})
}

// RespondToInvitation répond à une invitation (accepter/refuser)
func (h *ChatGroupHandler) RespondToInvitation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.RespondError(w, http.StatusMethodNotAllowed, constants.ErrMethodNotAllowed)
		return
	}

	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		utils.RespondError(w, http.StatusUnauthorized, constants.ErrNotAuthenticated)
		return
	}

	// Récupérer l'ID de l'invitation
	vars := mux.Vars(r)
	invitationID, err := primitive.ObjectIDFromHex(vars["invitation_id"])
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, constants.ErrInvitationInvalidID)
		return
	}

	var req models.RespondToGroupInvitationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, constants.ErrInvalidData)
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
		utils.RespondError(w, http.StatusNotFound, constants.ErrInvitationNotFound)
		return
	}

	// Vérifier que c'est bien l'utilisateur invité (invited_user en DB est un email)
	if invitation.InvitedUser != claims.Email {
		utils.RespondError(w, http.StatusForbidden, constants.ErrInvitationNotForYou)
		return
	}

	// Vérifier que l'invitation est en attente
	if invitation.Status != "pending" {
		utils.RespondError(w, http.StatusConflict, constants.ErrInvitationAlreadyDone)
		return
	}

	// Récupérer le groupe
	group, err := h.groupRepo.FindByID(invitation.GroupID)
	if err != nil || group == nil {
		utils.RespondError(w, http.StatusNotFound, constants.ErrGroupNotFound)
		return
	}

	// Traiter selon l'action
	if req.Action == "accept" {
		h.acceptInvitation(w, invitation, group, claims.Email)
	} else {
		h.rejectInvitation(w, invitation, group, claims.Email)
	}
}

// acceptInvitation accepte une invitation
func (h *ChatGroupHandler) acceptInvitation(w http.ResponseWriter, invitation *models.ChatGroupInvitation, group *models.ChatGroup, userID string) {
	// Mettre à jour l'invitation
	if err := h.invitationRepo.UpdateStatus(invitation.ID, "accepted"); err != nil {
		log.Printf("Erreur mise à jour invitation: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, constants.ErrServerError)
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
		utils.RespondError(w, http.StatusInternalServerError, constants.ErrServerError)
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

	log.Printf("Invitation acceptée (groupe: %s)", group.Name)
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
		utils.RespondError(w, http.StatusInternalServerError, constants.ErrServerError)
		return
	}

	// Récupérer les infos de l'utilisateur
	user, _ := h.userRepo.FindByEmail(userID)

	// Notifier UNIQUEMENT l'admin qui a invité (silencieux)
	h.notifyInvitationRejected(invitation, user, group)

	log.Printf("Invitation refusée (groupe: %s)", group.Name)
	utils.RespondSuccess(w, "Invitation refusée", nil)
}

// GetGroupMembers récupère les membres d'un groupe
func (h *ChatGroupHandler) GetGroupMembers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.RespondError(w, http.StatusMethodNotAllowed, constants.ErrMethodNotAllowed)
		return
	}

	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		utils.RespondError(w, http.StatusUnauthorized, constants.ErrNotAuthenticated)
		return
	}

	vars := mux.Vars(r)
	groupID, err := primitive.ObjectIDFromHex(vars["group_id"])
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, constants.ErrInvalidGroupID)
		return
	}

	// Vérifier que l'utilisateur est membre (user_id en DB est un email)
	isMember, err := h.groupRepo.IsMember(groupID, claims.Email)
	if err != nil || !isMember {
		utils.RespondError(w, http.StatusForbidden, constants.ErrNotGroupMember)
		return
	}

	// Récupérer les membres
	members, err := h.groupRepo.GetMembers(groupID)
	if err != nil {
		log.Printf("Erreur récupération membres: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, constants.ErrServerError)
		return
	}

	// ✅ Ajouter is_online pour chaque membre
	for i := range members {
		if h.wsHub != nil {
			members[i].IsOnline = h.wsHub.IsUserOnline(members[i].Email)
		}
	}

	utils.RespondSuccess(w, "Membres récupérés", map[string]interface{}{
		"members": members,
	})
}

// LeaveGroup permet à un utilisateur de quitter un groupe
func (h *ChatGroupHandler) LeaveGroup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.RespondError(w, http.StatusMethodNotAllowed, constants.ErrMethodNotAllowed)
		return
	}

	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		utils.RespondError(w, http.StatusUnauthorized, constants.ErrNotAuthenticated)
		return
	}

	vars := mux.Vars(r)
	groupID, err := primitive.ObjectIDFromHex(vars["group_id"])
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, constants.ErrInvalidGroupID)
		return
	}

	// Vérifier que l'utilisateur est membre (user_id en DB est un email)
	isMember, err := h.groupRepo.IsMember(groupID, claims.Email)
	if err != nil || !isMember {
		utils.RespondError(w, http.StatusForbidden, constants.ErrNotGroupMember)
		return
	}

	// Récupérer le groupe
	group, err := h.groupRepo.FindByID(groupID)
	if err != nil || group == nil {
		utils.RespondError(w, http.StatusNotFound, constants.ErrGroupNotFound)
		return
	}

	// Récupérer l'utilisateur
	user, _ := h.userRepo.FindByEmail(claims.Email)
	userName := "Un membre"
	if user != nil {
		userName = user.Firstname + " " + user.Lastname
	}

	// Retirer l'utilisateur du groupe
	if err := h.groupRepo.RemoveMember(groupID, claims.Email); err != nil {
		log.Printf("Erreur suppression membre: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, constants.ErrServerError)
		return
	}

	// Créer un message système
	systemMessage := &models.ChatGroupMessage{
		GroupID:     groupID,
		SenderID:    "system",
		Content:     fmt.Sprintf("%s a quitté le groupe", userName),
		MessageType: "system",
	}

	if err := h.messageRepo.Create(systemMessage); err != nil {
		log.Printf("Erreur création message système: %v", err)
	}

	// Notifier les autres membres via WebSocket
	members, _ := h.groupRepo.GetMembers(groupID)
	payload := map[string]interface{}{
		"type":      "group_member_left",
		"group_id":  groupID.Hex(),
		"user_id":   claims.Email,
		"user_name": userName,
		"message": map[string]interface{}{
			"id":           systemMessage.ID.Hex(),
			"sender_id":    "system",
			"content":      systemMessage.Content,
			"message_type": "system",
			"created_at":   systemMessage.CreatedAt,
		},
	}

	// Envoyer à tous les membres (JSON direct)
	// ✅ member.ID est maintenant l'email (corrigé dans GetMembers)
	for _, member := range members {
		log.Println("Envoi WS group_member_left")
		h.wsHub.SendToUser(member.ID, payload) // ✅ Utiliser ID (qui est l'email)
	}

	log.Printf("Utilisateur a quitté le groupe %s", group.Name)
	utils.RespondSuccess(w, "Vous avez quitté le groupe", nil)
}

// GetGroupPendingInvitations récupère les invitations en attente d'un groupe (admin seulement)
func (h *ChatGroupHandler) GetGroupPendingInvitations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.RespondError(w, http.StatusMethodNotAllowed, constants.ErrMethodNotAllowed)
		return
	}

	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		utils.RespondError(w, http.StatusUnauthorized, constants.ErrNotAuthenticated)
		return
	}

	vars := mux.Vars(r)
	groupID, err := primitive.ObjectIDFromHex(vars["group_id"])
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, constants.ErrInvalidGroupID)
		return
	}

	// Vérifier que l'utilisateur est membre (tous les membres peuvent voir les invitations)
	isMember, err := h.groupRepo.IsMember(groupID, claims.Email)
	if err != nil || !isMember {
		utils.RespondError(w, http.StatusForbidden, constants.ErrInvitationsMembersOnly)
		return
	}

	// Récupérer les invitations
	invitations, err := h.invitationRepo.FindPendingByGroup(groupID)
	if err != nil {
		log.Printf("Erreur récupération invitations: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, constants.ErrServerError)
		return
	}

	utils.RespondSuccess(w, "Invitations récupérées", map[string]interface{}{
		"invitations": invitations,
	})
}

// CancelInvitation annule une invitation (admin seulement)
func (h *ChatGroupHandler) CancelInvitation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.RespondError(w, http.StatusMethodNotAllowed, constants.ErrMethodNotAllowed)
		return
	}

	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		utils.RespondError(w, http.StatusUnauthorized, constants.ErrNotAuthenticated)
		return
	}

	vars := mux.Vars(r)
	invitationID, err := primitive.ObjectIDFromHex(vars["invitation_id"])
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, constants.ErrInvitationInvalidID)
		return
	}

	// Récupérer l'invitation
	invitation, err := h.invitationRepo.FindByID(invitationID)
	if err != nil || invitation == nil {
		utils.RespondError(w, http.StatusNotFound, constants.ErrInvitationNotFound)
		return
	}

	// Vérifier que l'utilisateur est admin du groupe
	isAdmin, err := h.groupRepo.IsAdmin(invitation.GroupID, claims.UserID)
	if err != nil || !isAdmin {
		utils.RespondError(w, http.StatusForbidden, constants.ErrInvitationsAdminCancel)
		return
	}

	// Supprimer l'invitation
	if err := h.invitationRepo.Delete(invitationID); err != nil {
		log.Printf("Erreur suppression invitation: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, constants.ErrServerError)
		return
	}

	log.Println("Invitation annulée")
	utils.RespondSuccess(w, "Invitation annulée", nil)
}

// SendMessage envoie un message dans un groupe
func (h *ChatGroupHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.RespondError(w, http.StatusMethodNotAllowed, constants.ErrMethodNotAllowed)
		return
	}

	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		utils.RespondError(w, http.StatusUnauthorized, constants.ErrNotAuthenticated)
		return
	}

	vars := mux.Vars(r)
	groupID, err := primitive.ObjectIDFromHex(vars["group_id"])
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, constants.ErrInvalidGroupID)
		return
	}

	var req models.SendGroupMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, constants.ErrInvalidData)
		return
	}

	// Valider
	if req.Content == "" {
		utils.RespondError(w, http.StatusBadRequest, constants.ErrMessageContentRequired)
		return
	}

	// Vérifier que l'utilisateur est membre (user_id en DB est un email)
	isMember, err := h.groupRepo.IsMember(groupID, claims.Email)
	if err != nil || !isMember {
		utils.RespondError(w, http.StatusForbidden, constants.ErrNotGroupMember)
		return
	}

	// Normaliser l'email pour la cohérence
	normalizedEmail := strings.ToLower(strings.TrimSpace(claims.Email))

	// Créer le message
	message := &models.ChatGroupMessage{
		GroupID:     groupID,
		SenderID:    normalizedEmail, // Utiliser l'email normalisé
		Content:     req.Content,
		MessageType: "message",
	}

	if err := h.messageRepo.Create(message); err != nil {
		log.Printf("Erreur création message: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, constants.ErrServerError)
		return
	}

	// Récupérer les infos de l'expéditeur (utiliser l'email normalisé)
	sender, err := h.userRepo.FindByEmail(normalizedEmail)
	if err != nil {
		log.Printf("Erreur récupération expéditeur: %v", err)
	}
	if sender == nil {
		log.Println("Expéditeur non trouvé en base de données")
	} else {
		log.Println("Expéditeur trouvé")
	}

	messageWithSender := models.GroupMessageWithSender{
		ID:          message.ID,
		SenderID:    message.SenderID,
		Content:     message.Content,
		MessageType: message.MessageType,
		Timestamp:   message.Timestamp,
		CreatedAt:   message.CreatedAt,
		DeliveredAt: message.DeliveredAt,
		ReadBy:      message.ReadBy,
	}

	if sender != nil {
		messageWithSender.Sender = &models.UserBasicInfo{
			ID:              sender.Email,
			Firstname:       sender.Firstname,
			Lastname:        sender.Lastname,
			Email:           sender.Email,
			ProfilePicture:  sender.ProfileImageURL,
			ProfileImageURL: sender.ProfileImageURL,
		}
		log.Println("Infos expéditeur ajoutées au message")
	} else {
		log.Println("Aucune info expéditeur disponible")
	}

	// Diffuser via WebSocket
	h.broadcastGroupMessage(groupID, &messageWithSender)

	// Envoyer FCM aux membres non connectés
	group, _ := h.groupRepo.FindByID(groupID)
	if group != nil {
		h.sendGroupMessageFCM(group, sender, message)
	}

	log.Printf("Message envoyé dans le groupe %s", groupID.Hex())
	utils.RespondSuccess(w, "Message envoyé", messageWithSender)
}

// GetMessages récupère les messages d'un groupe
func (h *ChatGroupHandler) GetMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.RespondError(w, http.StatusMethodNotAllowed, constants.ErrMethodNotAllowed)
		return
	}

	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		utils.RespondError(w, http.StatusUnauthorized, constants.ErrNotAuthenticated)
		return
	}

	vars := mux.Vars(r)
	groupID, err := primitive.ObjectIDFromHex(vars["group_id"])
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, constants.ErrInvalidGroupID)
		return
	}

	// Vérifier que l'utilisateur est membre (user_id en DB est un email)
	isMember, err := h.groupRepo.IsMember(groupID, claims.Email)
	if err != nil {
		log.Printf("❌ Erreur IsMember: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, constants.ErrServerError)
		return
	}
	if !isMember {
		log.Printf("User n'est pas membre du groupe %s", groupID.Hex())
		utils.RespondError(w, http.StatusForbidden, constants.ErrNotGroupMember)
		return
	}

	log.Printf("✅ User %s est membre du groupe %s", claims.Email, groupID.Hex())

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

	log.Printf("📨 Récupération messages: limit=%d, before=%v", limit, before)

	// Récupérer les messages
	messages, err := h.messageRepo.FindByGroupID(groupID, limit, before)
	if err != nil {
		log.Printf("❌ Erreur récupération messages: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, constants.ErrServerError)
		return
	}

	log.Printf("✅ Messages récupérés: %d", len(messages))

	// Enrichir les messages avec les données de l'expéditeur
	var enrichedMessages []models.GroupMessageWithSender
	for _, msg := range messages {
		enrichedMsg := models.GroupMessageWithSender{
			ID:          msg.ID,
			SenderID:    msg.SenderID,
			Content:     msg.Content,
			MessageType: msg.MessageType,
			Timestamp:   msg.Timestamp,
			CreatedAt:   msg.CreatedAt,
			DeliveredAt: msg.DeliveredAt,
			ReadBy:      msg.ReadBy,
		}

		// Récupérer les infos de l'expéditeur
		sender, err := h.userRepo.FindByEmail(msg.SenderID)
		if err == nil && sender != nil {
			enrichedMsg.Sender = &models.UserBasicInfo{
				ID:              sender.Email,
				Firstname:       sender.Firstname,
				Lastname:        sender.Lastname,
				Email:           sender.Email,
				ProfilePicture:  sender.ProfileImageURL,
				ProfileImageURL: sender.ProfileImageURL,
			}
		}

		enrichedMessages = append(enrichedMessages, enrichedMsg)
	}

	utils.RespondSuccess(w, "Messages récupérés", map[string]interface{}{
		"messages": enrichedMessages,
	})
}

// MarkAsRead marque les messages d'un groupe comme lus
func (h *ChatGroupHandler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.RespondError(w, http.StatusMethodNotAllowed, constants.ErrMethodNotAllowed)
		return
	}

	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		utils.RespondError(w, http.StatusUnauthorized, constants.ErrNotAuthenticated)
		return
	}

	vars := mux.Vars(r)
	groupID, err := primitive.ObjectIDFromHex(vars["group_id"])
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, constants.ErrInvalidGroupID)
		return
	}

	// Vérifier que l'utilisateur est membre (user_id en DB est un email)
	isMember, err := h.groupRepo.IsMember(groupID, claims.Email)
	if err != nil || !isMember {
		utils.RespondError(w, http.StatusForbidden, constants.ErrNotGroupMember)
		return
	}

	// Marquer comme lu
	if err := h.messageRepo.MarkAsRead(groupID, claims.Email); err != nil {
		log.Printf("Erreur marquage comme lu: %v", err)
		utils.RespondError(w, http.StatusInternalServerError, constants.ErrServerError)
		return
	}

	// Notifier les autres membres via WebSocket
	h.broadcastMessagesRead(groupID, claims.Email)

	log.Printf("Messages marqués comme lus dans le groupe %s", groupID.Hex())
	utils.RespondSuccess(w, "Messages marqués comme lus", nil)
}

// SearchUsers recherche des utilisateurs (pour inviter)
func (h *ChatGroupHandler) SearchUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.RespondError(w, http.StatusMethodNotAllowed, constants.ErrMethodNotAllowed)
		return
	}

	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		utils.RespondError(w, http.StatusUnauthorized, constants.ErrNotAuthenticated)
		return
	}

	query := r.URL.Query().Get("q")
	if len(query) < 2 {
		utils.RespondError(w, http.StatusBadRequest, constants.ErrSearchMinChars)
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
		utils.RespondError(w, http.StatusInternalServerError, constants.ErrServerError)
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
