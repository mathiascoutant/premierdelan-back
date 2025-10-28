package websocket

import (
	"context"
	"log"
	"sync"
	"time"

	"premier-an-backend/database"
	"premier-an-backend/models"

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
			log.Printf("🔌 Client connecté: %s (total: %d)", client.UserID, len(h.connections))

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
			log.Printf("👋 Client déconnecté: %s (total: %d)", client.UserID, len(h.connections))

			// 🔌 Mettre à jour la présence (marquer comme hors ligne immédiatement)
			if h.presenceManager != nil {
				h.presenceManager.UpdateUserPresence(client.UserID, false)
				h.presenceManager.RemoveUser(client.UserID)
			}

		case message := <-h.broadcast:
			h.mu.RLock()

			log.Printf("📡 Broadcast: ConvID=%s, UserIDs=%v, Exclude=%s", message.ConversationID, message.UserIDs, message.ExcludeUserID)

			// Si UserIDs spécifié, envoyer uniquement à ces utilisateurs
			if len(message.UserIDs) > 0 {
				log.Printf("📤 Envoi à utilisateurs spécifiques: %v", message.UserIDs)
				for _, userID := range message.UserIDs {
					if userID == message.ExcludeUserID {
						log.Printf("⏭️  Skip user %s (exclu)", userID)
						continue
					}
					if client, ok := h.connections[userID]; ok {
						select {
						case client.send <- message.Payload:
							log.Printf("✅ Message envoyé à %s", userID)
						default:
							log.Printf("❌ Canal plein pour %s", userID)
							close(client.send)
							delete(h.connections, userID)
						}
					} else {
						log.Printf("⚠️  User %s non connecté", userID)
					}
				}
			} else if message.ConversationID != "" {
				// Sinon, envoyer à tous les membres de la conversation
				if members, ok := h.rooms[message.ConversationID]; ok {
					log.Printf("📤 Conversation %s a %d membres dans la room", message.ConversationID, len(members))
					for userID := range members {
						if userID == message.ExcludeUserID {
							log.Printf("⏭️  Skip user %s (expéditeur)", userID)
							continue
						}
						if client, ok := h.connections[userID]; ok {
							select {
							case client.send <- message.Payload:
								log.Printf("✅ Message WS envoyé à %s", userID)
							default:
								log.Printf("❌ Canal plein pour %s", userID)
								close(client.send)
								delete(h.connections, userID)
							}
						} else {
							log.Printf("⚠️  User %s dans la room mais pas connecté WS", userID)
						}
					}
				} else {
					log.Printf("⚠️  Conversation %s n'a aucun membre dans les rooms", message.ConversationID)
					log.Printf("🔍 Rooms actuelles: %v", h.rooms)
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
	log.Printf("✅ User %s a rejoint la conversation %s", userID, conversationID)
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
		log.Printf("👋 User %s a quitté la conversation %s", userID, conversationID)
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
		log.Printf("⚠️  chatRepo nil - présence non notifiée")
		return
	}

	log.Printf("👁️  Notification présence pour %s (online=%v)", userID, isOnline)

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

	log.Printf("📦 Payload user_presence: %+v", payload)

	// Envoyer à tous les autres participants (éviter doublons)
	// ⚠️  IMPORTANT: Utiliser EMAIL, pas ObjectID !
	sentTo := make(map[string]bool)
	for _, conv := range conversations {
		otherUserEmail := conv.Participant.Email // ✅ Utiliser Email au lieu de ID (ObjectID)
		if otherUserEmail != userID && !sentTo[otherUserEmail] {
			h.SendToUser(otherUserEmail, payload)
			sentTo[otherUserEmail] = true
			log.Printf("📤 Présence envoyée à %s", otherUserEmail)
		}
	}

	log.Printf("✅ Présence notifiée à %d contacts", len(sentTo))
}

// autoJoinUserConversations ajoute automatiquement l'utilisateur à toutes ses conversations
func (h *Hub) autoJoinUserConversations(userID string) {
	if h.chatRepo == nil {
		log.Printf("⚠️  chatRepo nil - auto-join impossible")
		return
	}

	log.Printf("🔄 Auto-join conversations pour %s", userID)

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
	joinedCount := 0
	for _, conv := range conversations {
		if conv.ID != "" {
			h.JoinConversation(userID, conv.ID)
			joinedCount++
		}
	}

	log.Printf("✅ Auto-join terminé: %d conversations rejointes", joinedCount)
}

// autoJoinUserGroups ajoute automatiquement l'utilisateur à tous ses groupes
func (h *Hub) autoJoinUserGroups(userID string) {
	log.Printf("🔄 Auto-join groupes pour %s", userID)

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
	
	log.Printf("✅ Auto-join groupes terminé pour %s", userID)
}

// HandleTyping gère l'événement "typing" et l'envoie aux autres participants
func (h *Hub) HandleTyping(userID, conversationID string, isTyping bool) {
	log.Printf("⌨️  Typing: user=%s, conv=%s, typing=%v", userID, conversationID, isTyping)

	// Récupérer le prénom de l'utilisateur
	username := "Quelqu'un"
	if h.userRepo != nil {
		if user, err := h.userRepo.FindByEmail(userID); err == nil && user != nil {
			username = user.Firstname
			log.Printf("✅ Username récupéré: %s", username)
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

	log.Printf("✅ Typing indicator envoyé pour conversation %s", conversationID)
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
	log.Printf("✅ User %s a rejoint le groupe %s", userID, groupID)
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
		log.Printf("👋 User %s a quitté le groupe %s", userID, groupID)
	}
}

// BroadcastToGroup envoie un message à tous les membres d'un groupe
func (h *Hub) BroadcastToGroup(groupID string, payload interface{}, excludeUserID string) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	log.Printf("📡 Broadcast groupe: GroupID=%s, Exclude=%s", groupID, excludeUserID)
	log.Printf("🔍 Group rooms disponibles: %+v", h.groupRooms)

	if members, ok := h.groupRooms[groupID]; ok {
		log.Printf("📤 Groupe %s a %d membres dans la room: %v", groupID, len(members), getKeys(members))
		sentCount := 0
		for userID := range members {
			if userID == excludeUserID {
				log.Printf("⏭️  Skip user %s (exclu)", userID)
				continue
			}
			if client, ok := h.connections[userID]; ok {
				select {
				case client.send <- payload:
					log.Printf("✅ Message groupe envoyé à %s", userID)
					sentCount++
				default:
					log.Printf("❌ Canal plein pour %s", userID)
				}
			} else {
				log.Printf("⚠️  User %s dans le groupe mais pas connecté WS", userID)
			}
		}
		log.Printf("📊 Broadcast groupe terminé: %d messages envoyés", sentCount)
	} else {
		log.Printf("⚠️  Groupe %s n'a aucun membre dans les rooms", groupID)
		log.Printf("🔍 Group rooms disponibles: %v", h.groupRooms)
		log.Printf("💡 Suggestion: L'utilisateur doit d'abord rejoindre le groupe via 'join_group'")
	}
}

// getKeys retourne les clés d'une map pour le debug
func getKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// BroadcastToUser envoie un message à un utilisateur spécifique (alias pour SendToUser)
func (h *Hub) BroadcastToUser(userID string, payload []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if client, ok := h.connections[userID]; ok {
		select {
		case client.send <- payload:
			log.Printf("✅ Message envoyé à l'utilisateur %s", userID)
		default:
			log.Printf("❌ Canal plein pour l'utilisateur %s", userID)
		}
	} else {
		log.Printf("⚠️  Utilisateur %s non connecté", userID)
	}
}

// HandleGroupTyping gère l'événement "typing" dans un groupe
func (h *Hub) HandleGroupTyping(userID, groupID string, isTyping bool) {
	log.Printf("⌨️  Group Typing: user=%s, group=%s, typing=%v", userID, groupID, isTyping)

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
			log.Printf("✅ Username récupéré: %s", username)
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

	// Envoyer via BroadcastToGroup (qui envoie à tous SAUF l'expéditeur)
	h.BroadcastToGroup(groupID, payload, userID)

	log.Printf("✅ Group typing indicator envoyé pour groupe %s", groupID)
}

// ====================================
// Méthodes pour le gestionnaire de présence
// ====================================

// updateUserPresenceInDB met à jour la présence d'un utilisateur en base de données
func (h *Hub) updateUserPresenceInDB(userID string, isOnline bool) error {
	if h.userRepo == nil {
		log.Printf("⚠️  userRepo nil - présence non mise à jour en DB")
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

	if !isOnline {
		// Si hors ligne, mettre à jour last_seen
		updateData["last_seen"] = time.Now()
	}

	// Utiliser UpdateByEmail si disponible, sinon UpdateByID
	if err := h.userRepo.UpdateByEmail(userID, updateData); err != nil {
		log.Printf("❌ Erreur mise à jour présence en DB: %v", err)
		return err
	}

	log.Printf("✅ Présence mise à jour en DB: %s -> %v", userID, isOnline)
	return nil
}

// broadcastPresenceUpdate diffuse une mise à jour de présence à tous les contacts
func (h *Hub) broadcastPresenceUpdate(userID string, isOnline bool, lastSeen *time.Time) {
	if h.chatRepo == nil {
		log.Printf("⚠️  chatRepo nil - présence non diffusée")
		return
	}

	log.Printf("👁️  Diffusion présence pour %s (online=%v)", userID, isOnline)

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

	// Ajouter last_seen si hors ligne
	if !isOnline && lastSeen != nil {
		payload["last_seen"] = lastSeen.Format(time.RFC3339)
	}

	log.Printf("📦 Payload user_presence: %+v", payload)

	// Envoyer à tous les autres participants (éviter doublons)
	sentTo := make(map[string]bool)
	for _, conv := range conversations {
		otherUserEmail := conv.Participant.Email
		if otherUserEmail != userID && !sentTo[otherUserEmail] {
			h.SendToUser(otherUserEmail, payload)
			sentTo[otherUserEmail] = true
			log.Printf("📤 Présence diffusée à %s", otherUserEmail)
		}
	}

	log.Printf("✅ Présence diffusée à %d contacts", len(sentTo))
}

// Shutdown arrête le hub et marque tous les utilisateurs comme hors ligne
func (h *Hub) Shutdown() {
	log.Printf("🔄 Arrêt du hub WebSocket...")

	// Arrêter le gestionnaire de présence
	if h.presenceManager != nil {
		h.presenceManager.Shutdown()
	}

	// Fermer toutes les connexions
	h.mu.Lock()
	for userID, client := range h.connections {
		close(client.send)
		client.conn.Close()
		log.Printf("🔌 Connexion fermée pour %s", userID)
	}
	h.connections = make(map[string]*Client)
	h.mu.Unlock()

	log.Printf("✅ Hub WebSocket arrêté")
}
