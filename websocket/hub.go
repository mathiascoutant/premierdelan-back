package websocket

import (
	"log"
	"sync"
)

// Hub gère les connexions WebSocket actives
type Hub struct {
	// Connexions actives par user_id
	connections map[string]*Client

	// Rooms de conversations (conversation_id -> [user_id])
	rooms map[string]map[string]bool

	// Mutex pour sécuriser les accès concurrents
	mu sync.RWMutex

	// Canal pour enregistrer les clients
	register chan *Client

	// Canal pour désenregistrer les clients
	unregister chan *Client

	// Canal pour diffuser les messages
	broadcast chan *Message
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
func NewHub() *Hub {
	return &Hub{
		connections: make(map[string]*Client),
		rooms:       make(map[string]map[string]bool),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		broadcast:   make(chan *Message, 256),
	}
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
			}
			h.mu.Unlock()
			log.Printf("👋 Client déconnecté: %s (total: %d)", client.UserID, len(h.connections))

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

