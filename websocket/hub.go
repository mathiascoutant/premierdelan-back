package chatws

import (
	"log"
	"sync"
)

// Hub maintient l'ensemble des clients actifs et diffuse les messages
type Hub struct {
	// Clients enregistrés par user_id
	clients map[string]*Client

	// Clients par conversation_id
	conversations map[string]map[*Client]bool

	// Canal pour enregistrer les clients
	register chan *Client

	// Canal pour désenregistrer les clients
	unregister chan *Client

	// Canal pour diffuser les messages
	broadcast chan *Message

	// Mutex pour la sécurité concurrentielle
	mu sync.RWMutex
}

// Message représente un message WebSocket
type Message struct {
	Type           string                 `json:"type"`
	ConversationID string                 `json:"conversation_id,omitempty"`
	Data           map[string]interface{} `json:"data,omitempty"`
	TargetUserID   string                 `json:"-"` // ID de l'utilisateur cible
	TargetClients  []*Client              `json:"-"` // Clients cibles
}

// NewHub crée un nouveau hub WebSocket
func NewHub() *Hub {
	return &Hub{
		clients:       make(map[string]*Client),
		conversations: make(map[string]map[*Client]bool),
		register:      make(chan *Client),
		unregister:    make(chan *Client),
		broadcast:     make(chan *Message, 256),
	}
}

// Run démarre le hub et gère les événements
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if client.UserID != "" {
				h.clients[client.UserID] = client
				log.Printf("✅ Client WebSocket enregistré: %s", client.UserID)
			}
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if client.UserID != "" {
				if _, ok := h.clients[client.UserID]; ok {
					delete(h.clients, client.UserID)
					close(client.send)
					log.Printf("❌ Client WebSocket désenregistré: %s", client.UserID)
				}
			}
			// Retirer de toutes les conversations
			for _, clients := range h.conversations {
				delete(clients, client)
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			
			// Si message pour un utilisateur spécifique
			if message.TargetUserID != "" {
				if client, ok := h.clients[message.TargetUserID]; ok {
					select {
					case client.send <- message:
					default:
						close(client.send)
						delete(h.clients, client.UserID)
					}
				}
			}
			
			// Si message pour une conversation
			if message.ConversationID != "" {
				if clients, ok := h.conversations[message.ConversationID]; ok {
					for client := range clients {
						select {
						case client.send <- message:
						default:
							close(client.send)
							delete(clients, client)
						}
					}
				}
			}
			
			// Si clients cibles spécifiés
			for _, client := range message.TargetClients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					h.mu.Lock()
					delete(h.clients, client.UserID)
					h.mu.Unlock()
				}
			}
			
			h.mu.RUnlock()
		}
	}
}

// JoinConversation ajoute un client à une conversation
func (h *Hub) JoinConversation(client *Client, conversationID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.conversations[conversationID] == nil {
		h.conversations[conversationID] = make(map[*Client]bool)
	}
	h.conversations[conversationID][client] = true
	log.Printf("👤 Client %s a rejoint la conversation %s", client.UserID, conversationID)
}

// LeaveConversation retire un client d'une conversation
func (h *Hub) LeaveConversation(client *Client, conversationID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.conversations[conversationID]; ok {
		delete(clients, client)
		log.Printf("👋 Client %s a quitté la conversation %s", client.UserID, conversationID)
	}
}

// NotifyUser envoie un message à un utilisateur spécifique
func (h *Hub) NotifyUser(userID string, eventType string, data map[string]interface{}) {
	h.broadcast <- &Message{
		Type:         eventType,
		Data:         data,
		TargetUserID: userID,
	}
}

// NotifyConversation envoie un message à tous les participants d'une conversation
func (h *Hub) NotifyConversation(conversationID string, eventType string, data map[string]interface{}) {
	h.broadcast <- &Message{
		Type:           eventType,
		ConversationID: conversationID,
		Data:           data,
	}
}

