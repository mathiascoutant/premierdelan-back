package websocket

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"premier-an-backend/database"
	"premier-an-backend/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// UserRepository interface pour éviter la dépendance circulaire
type UserRepository interface {
	UpdateLastSeen(userID primitive.ObjectID) error
	FindByID(userID primitive.ObjectID) (*models.User, error)
	FindByEmail(email string) (*models.User, error)
}

// ChatRepository interface pour récupérer les conversations d'un utilisateur
type ChatRepository interface {
	GetConversations(ctx context.Context, userID primitive.ObjectID) ([]models.ConversationResponse, error)
}

// Hub gère les connexions WebSocket actives
type Hub struct {
	// Connexions actives par user_id
	connections map[string]*Client

	// Rooms de conversations (conversation_id -> [user_id])
	rooms map[string]map[string]bool

	// Group rooms (group_id -> [user_id])
	groupRooms map[string]map[string]bool

	// Mutex pour sécuriser les accès concurrents
	mu sync.RWMutex

	// Canal pour enregistrer les clients
	register chan *Client

	// Canal pour désenregistrer les clients
	unregister chan *Client

	// Canal pour diffuser les messages
	broadcast chan *Message

	// Repositories pour la gestion de la présence
	userRepo *database.UserRepository
	chatRepo *database.ChatRepository

	// Gestionnaire de présence avec timeouts automatiques
	presenceManager *PresenceManager
}

// Message représente un message WebSocket à diffuser
type Message struct {
	Type           string
	ConversationID string
	UserIDs        []string // Si vide, envoyer à toute la conversation
	ExcludeUserID  string   // Ne pas envoyer à cet utilisateur
	Payload        interface{}
}

// NewHub crée un nouveau hub WebSocket
func NewHub(userRepo *database.UserRepository, chatRepo *database.ChatRepository) *Hub {
	hub := &Hub{
		connections: make(map[string]*Client),
		rooms:       make(map[string]map[string]bool),
		groupRooms:  make(map[string]map[string]bool),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		broadcast:   make(chan *Message, 256),
		userRepo:    userRepo,
		chatRepo:    chatRepo,
	}

	// Initialiser le gestionnaire de présence
	hub.presenceManager = NewPresenceManager(
		hub.updateUserPresenceInDB,
		hub.broadcastPresenceUpdate,
		hub.getCurrentUserStatus,
	)

	return hub
}

// Run démarre la boucle principale du hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.connections[client.UserID] = client
			h.mu.Unlock()

			// 🔌 Auto-joindre toutes les conversations de l'utilisateur
			go h.autoJoinUserConversations(client.UserID)

			// 🔌 Auto-joindre tous les groupes de l'utilisateur
			go h.autoJoinUserGroups(client.UserID)

			// 🔌 Mettre à jour la présence avec timeout automatique
			if h.presenceManager != nil {
				h.presenceManager.UpdateUserPresence(client.UserID, true)
			}

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.connections[client.UserID]; ok {
				delete(h.connections, client.UserID)
				close(client.send)

				// Retirer de toutes les rooms
				for roomID, members := range h.rooms {
					delete(members, client.UserID)
					if len(members) == 0 {
						delete(h.rooms, roomID)
					}
				}

				// Retirer de toutes les group rooms
				for groupID, members := range h.groupRooms {
					delete(members, client.UserID)
					if len(members) == 0 {
						delete(h.groupRooms, groupID)
					}
				}
			}
			h.mu.Unlock()

			// 🔌 Mettre à jour la présence (marquer comme hors ligne immédiatement)
			if h.presenceManager != nil {
				h.presenceManager.UpdateUserPresence(client.UserID, false)
				h.presenceManager.RemoveUser(client.UserID)
			}

		case message := <-h.broadcast:
			h.mu.RLock()

			// Si UserIDs spécifié, envoyer uniquement à ces utilisateurs
			if len(message.UserIDs) > 0 {
				for _, userID := range message.UserIDs {
					if userID == message.ExcludeUserID {
						continue
					}
					if client, ok := h.connections[userID]; ok {
						select {
						case client.send <- message.Payload:
							// Message envoyé avec succès
						default:
							log.Printf("❌ Canal plein pour %s", userID)
							close(client.send)
							delete(h.connections, userID)
						}
					} else {
						// Utilisateur non connecté - c'est normal s'il n'est pas sur une page avec WebSocket
						// Ne pas logger comme erreur pour éviter de polluer les logs
						_ = userID // Utilisé implicitement dans le commentaire ci-dessus
					}
				}
			} else if message.ConversationID != "" {
				// Sinon, envoyer à tous les membres de la conversation
				if members, ok := h.rooms[message.ConversationID]; ok {
					for userID := range members {
						if userID == message.ExcludeUserID {
							continue
						}
						if client, ok := h.connections[userID]; ok {
							select {
							case client.send <- message.Payload:
								// Message envoyé avec succès
							default:
								log.Printf("❌ Canal plein pour %s", userID)
								close(client.send)
								delete(h.connections, userID)
							}
						} else {
							// Utilisateur dans la room mais pas connecté - c'est normal
							// Ne pas logger comme erreur pour éviter de polluer les logs
							_ = userID // Utilisé implicitement dans le commentaire ci-dessus
						}
					}
				}
			}

			h.mu.RUnlock()
		}
	}
}

// JoinConversation ajoute un utilisateur à une room de conversation
func (h *Hub) JoinConversation(userID, conversationID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.rooms[conversationID] == nil {
		h.rooms[conversationID] = make(map[string]bool)
	}
	h.rooms[conversationID][userID] = true
}

// LeaveConversation retire un utilisateur d'une room de conversation
func (h *Hub) LeaveConversation(userID, conversationID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if members, ok := h.rooms[conversationID]; ok {
		delete(members, userID)
		if len(members) == 0 {
			delete(h.rooms, conversationID)
		}
	}
}

// SendToUser envoie un message à un utilisateur spécifique
func (h *Hub) SendToUser(userID string, payload interface{}) {
	h.broadcast <- &Message{
		UserIDs: []string{userID},
		Payload: payload,
	}
}

// SendToConversation envoie un message à tous les membres d'une conversation
func (h *Hub) SendToConversation(conversationID string, payload interface{}, excludeUserID string) {
	h.broadcast <- &Message{
		ConversationID: conversationID,
		ExcludeUserID:  excludeUserID,
		Payload:        payload,
	}
}

// IsUserOnline vérifie si un utilisateur est actuellement connecté
func (h *Hub) IsUserOnline(userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	_, online := h.connections[userID]
	return online
}

// notifyUserPresence envoie un événement de présence à tous les contacts d'un utilisateur
func (h *Hub) notifyUserPresence(userID string, isOnline bool) {
	if h.chatRepo == nil {
		return
	}

	// Récupérer l'utilisateur par email (userID est maintenant un email)
	user, err := h.userRepo.FindByEmail(userID)
	if err != nil || user == nil {
		log.Printf("❌ Utilisateur invalide pour présence: %s", userID)
		return
	}
	userObjID := user.ID

	// Récupérer toutes les conversations de cet utilisateur
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conversations, err := h.chatRepo.GetConversations(ctx, userObjID)
	if err != nil {
		log.Printf("❌ Erreur récupération conversations pour présence: %v", err)
		return
	}

	// Récupérer last_seen depuis la DB
	lastSeenStr := time.Now().Format(time.RFC3339)
	if !isOnline && user.LastSeen != nil {
		lastSeenStr = user.LastSeen.Format(time.RFC3339)
	}

	// Payload de présence
	payload := map[string]interface{}{
		"type":      "user_presence",
		"user_id":   userID,
		"is_online": isOnline,
		"last_seen": lastSeenStr, // ✅ Format ISO 8601 string
	}

	// Envoyer à tous les autres participants (éviter doublons)
	// ⚠️  IMPORTANT: Utiliser EMAIL, pas ObjectID !
	sentTo := make(map[string]bool)
	for _, conv := range conversations {
		otherUserEmail := conv.Participant.Email // ✅ Utiliser Email au lieu de ID (ObjectID)
		if otherUserEmail != userID && !sentTo[otherUserEmail] {
			h.SendToUser(otherUserEmail, payload)
			sentTo[otherUserEmail] = true
		}
	}

}

// autoJoinUserConversations ajoute automatiquement l'utilisateur à toutes ses conversations
func (h *Hub) autoJoinUserConversations(userID string) {
	if h.chatRepo == nil {
		return
	}

	// Récupérer l'utilisateur par email
	user, err := h.userRepo.FindByEmail(userID)
	if err != nil || user == nil {
		log.Printf("❌ Utilisateur invalide pour auto-join: %s", userID)
		return
	}
	userObjID := user.ID

	// Récupérer toutes les conversations de cet utilisateur
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conversations, err := h.chatRepo.GetConversations(ctx, userObjID)
	if err != nil {
		log.Printf("❌ Erreur récupération conversations pour auto-join: %v", err)
		return
	}

	// Joindre chaque conversation
	for _, conv := range conversations {
		if conv.ID != "" {
			h.JoinConversation(userID, conv.ID)
		}
	}
}

// autoJoinUserGroups ajoute automatiquement l'utilisateur à tous ses groupes
func (h *Hub) autoJoinUserGroups(userID string) {
	// Récupérer l'utilisateur par email
	user, err := h.userRepo.FindByEmail(userID)
	if err != nil || user == nil {
		log.Printf("❌ Utilisateur invalide pour auto-join groupes: %s", userID)
		return
	}

	// Récupérer tous les groupes de cet utilisateur
	// Note: On aurait besoin d'accès au groupRepo, mais pour l'instant on fait confiance
	// TODO: Implémenter la récupération des groupes depuis la DB
	// Pour l'instant, on laisse les utilisateurs rejoindre manuellement via join_group
}

// HandleTyping gère l'événement "typing" et l'envoie aux autres participants
func (h *Hub) HandleTyping(userID, conversationID string, isTyping bool) {
	// Récupérer le prénom de l'utilisateur
	username := "Quelqu'un"
	if h.userRepo != nil {
		if user, err := h.userRepo.FindByEmail(userID); err == nil && user != nil {
			username = user.Firstname
		}
	}

	// Payload à envoyer aux autres participants
	payload := map[string]interface{}{
		"type":            "user_typing",
		"conversation_id": conversationID,
		"user_id":         userID,
		"username":        username,
		"is_typing":       isTyping,
	}

	// Envoyer via SendToConversation (qui envoie à tous SAUF l'expéditeur)
	h.SendToConversation(conversationID, payload, userID)
}

// ====================================
// Méthodes pour les groupes de chat
// ====================================

// JoinGroup ajoute un utilisateur à une room de groupe
func (h *Hub) JoinGroup(userID, groupID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.groupRooms[groupID] == nil {
		h.groupRooms[groupID] = make(map[string]bool)
	}
	h.groupRooms[groupID][userID] = true
}

// LeaveGroup retire un utilisateur d'une room de groupe
func (h *Hub) LeaveGroup(userID, groupID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if members, ok := h.groupRooms[groupID]; ok {
		delete(members, userID)
		if len(members) == 0 {
			delete(h.groupRooms, groupID)
		}
	}
}

// BroadcastToGroup envoie un message à tous les membres d'un groupe (y compris l'expéditeur)
func (h *Hub) BroadcastToGroup(groupID string, payload interface{}) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if members, ok := h.groupRooms[groupID]; ok {
		for userID := range members {
			if client, ok := h.connections[userID]; ok {
				select {
				case client.send <- payload:
					// Message envoyé avec succès
				default:
					log.Printf("❌ Canal plein pour %s", userID)
				}
			} else {
				// Utilisateur dans le groupe mais pas connecté - c'est normal
				// Ne pas logger comme erreur pour éviter de polluer les logs
			}
		}
	} else {
	}
}

// BroadcastToUser envoie un message à un utilisateur spécifique (alias pour SendToUser)
func (h *Hub) BroadcastToUser(userID string, payload []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if client, ok := h.connections[userID]; ok {
		select {
		case client.send <- payload:
			// Message envoyé avec succès
		default:
			log.Printf("❌ Canal plein pour l'utilisateur %s", userID)
		}
	} else {
		// Utilisateur non connecté - c'est normal
		// Ne pas logger comme erreur pour éviter de polluer les logs
	}
}

// HandleGroupTyping gère l'événement "typing" dans un groupe
func (h *Hub) HandleGroupTyping(userID, groupID string, isTyping bool) {
	// Convertir groupID string en ObjectID pour validation
	_, err := primitive.ObjectIDFromHex(groupID)
	if err != nil {
		log.Printf("❌ GroupID invalide: %s", groupID)
		return
	}

	// Vérifier que l'utilisateur est membre du groupe
	// Note: On aurait besoin d'accès au groupRepo, mais pour l'instant on fait confiance
	// TODO: Ajouter validation d'appartenance au groupe si nécessaire

	// Récupérer le prénom de l'utilisateur
	username := "Quelqu'un"
	if h.userRepo != nil {
		if user, err := h.userRepo.FindByEmail(userID); err == nil && user != nil {
			username = user.Firstname + " " + user.Lastname
		}
	}

	// Payload à envoyer aux autres participants
	payload := map[string]interface{}{
		"type":      "user_typing",
		"group_id":  groupID,
		"user_id":   userID,
		"username":  username,
		"is_typing": isTyping,
	}

	// Envoyer via BroadcastToGroup (qui envoie maintenant à tout le monde, y compris l'expéditeur)
	h.BroadcastToGroup(groupID, payload)
}

// ====================================
// Méthodes pour le gestionnaire de présence
// ====================================

// getCurrentUserStatus récupère le statut actuel d'un utilisateur depuis la base de données
func (h *Hub) getCurrentUserStatus(userID string) (bool, error) {
	if h.userRepo == nil {
		return false, fmt.Errorf("userRepo nil")
	}

	// Récupérer l'utilisateur par email pour vérifier qu'il existe
	user, err := h.userRepo.FindByEmail(userID)
	if err != nil || user == nil {
		return false, fmt.Errorf("utilisateur non trouvé: %s", userID)
	}

	// Récupérer is_online depuis la DB avec une requête directe
	// On utilise database.DB directement car userRepo n'expose pas la collection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var result struct {
		IsOnline bool `bson:"is_online"`
	}
	err = database.DB.Collection("users").FindOne(ctx, bson.M{"email": userID}).Decode(&result)
	if err != nil {
		// Si le champ n'existe pas ou erreur, considérer comme false (hors ligne par défaut)
		return false, nil
	}

	return result.IsOnline, nil
}

// updateUserPresenceInDB met à jour la présence d'un utilisateur en base de données
func (h *Hub) updateUserPresenceInDB(userID string, isOnline bool) error {
	if h.userRepo == nil {
		return nil
	}

	// Récupérer l'utilisateur par email
	user, err := h.userRepo.FindByEmail(userID)
	if err != nil || user == nil {
		log.Printf("❌ Utilisateur non trouvé pour mise à jour présence: %s", userID)
		return err
	}

	// Mettre à jour la présence
	updateData := map[string]interface{}{
		"is_online": isOnline,
	}

	if isOnline {
		// Si en ligne, mettre à jour last_activity
		updateData["last_activity"] = time.Now()
	} else {
		// Si hors ligne, mettre à jour last_seen
		updateData["last_seen"] = time.Now()
	}

	// Utiliser UpdateByEmail si disponible, sinon UpdateByID
	if err := h.userRepo.UpdateByEmail(userID, updateData); err != nil {
		log.Printf("❌ Erreur mise à jour présence en DB: %v", err)
		return err
	}

	return nil
}

// updateUserActivityInDB met à jour la dernière activité d'un utilisateur en base de données
func (h *Hub) updateUserActivityInDB(userID string) error {
	if h.userRepo == nil {
		return nil
	}

	// Mettre à jour last_activity (timestamp de dernière activité)
	updateData := map[string]interface{}{
		"last_activity": time.Now(),
		"is_online":     true, // S'assurer que is_online est à true
	}

	if err := h.userRepo.UpdateByEmail(userID, updateData); err != nil {
		log.Printf("❌ Erreur mise à jour activité en DB: %v", err)
		return err
	}

	return nil
}

// updateUserLastSeenInDB met à jour le last_seen d'un utilisateur en base de données
func (h *Hub) updateUserLastSeenInDB(userID string, lastSeen *time.Time) error {
	if h.userRepo == nil {
		return nil
	}

	if lastSeen == nil {
		return nil
	}

	// Mettre à jour last_seen
	updateData := map[string]interface{}{
		"last_seen": *lastSeen,
		"is_online": false, // S'assurer que is_online est à false
	}

	if err := h.userRepo.UpdateByEmail(userID, updateData); err != nil {
		log.Printf("❌ Erreur mise à jour last_seen en DB: %v", err)
		return err
	}

	return nil
}

// broadcastPresenceUpdate diffuse une mise à jour de présence à tous les contacts
func (h *Hub) broadcastPresenceUpdate(userID string, isOnline bool, lastSeen *time.Time) {
	if h.chatRepo == nil {
		return
	}

	// Récupérer l'utilisateur par email
	user, err := h.userRepo.FindByEmail(userID)
	if err != nil || user == nil {
		log.Printf("❌ Utilisateur invalide pour diffusion présence: %s", userID)
		return
	}
	userObjID := user.ID

	// Récupérer toutes les conversations de cet utilisateur
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conversations, err := h.chatRepo.GetConversations(ctx, userObjID)
	if err != nil {
		log.Printf("❌ Erreur récupération conversations pour diffusion présence: %v", err)
		return
	}

	// Préparer le payload de présence
	payload := map[string]interface{}{
		"type":      "user_presence",
		"user_id":   userID,
		"is_online": isOnline,
	}

	// Ajouter last_seen (format ISO 8601 string ou null)
	if isOnline {
		// Si en ligne, last_seen est null
		payload["last_seen"] = nil
	} else if lastSeen != nil {
		// Si hors ligne avec last_seen fourni, l'inclure
		payload["last_seen"] = lastSeen.Format(time.RFC3339)
	} else {
		// Si hors ligne sans last_seen, utiliser celui de l'utilisateur en DB
		if user.LastSeen != nil {
			payload["last_seen"] = user.LastSeen.Format(time.RFC3339)
		} else {
			payload["last_seen"] = nil
		}
	}

	// Envoyer à tous les autres participants (éviter doublons)
	sentTo := make(map[string]bool)
	for _, conv := range conversations {
		otherUserEmail := conv.Participant.Email
		if otherUserEmail != userID && !sentTo[otherUserEmail] {
			h.SendToUser(otherUserEmail, payload)
			sentTo[otherUserEmail] = true
		}
	}
}

// Shutdown arrête le hub et marque tous les utilisateurs comme hors ligne
func (h *Hub) Shutdown() {
	// Arrêter le gestionnaire de présence
	if h.presenceManager != nil {
		h.presenceManager.Shutdown()
	}

	// Fermer toutes les connexions
	h.mu.Lock()
	for _, client := range h.connections {
		close(client.send)
		client.conn.Close()
	}
	h.connections = make(map[string]*Client)
	h.mu.Unlock()
}
