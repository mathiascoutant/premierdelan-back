package chatws

import (
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Temps maximum pour l'écriture d'un message
	writeWait = 10 * time.Second

	// Temps entre les pings
	pingPeriod = 54 * time.Second

	// Temps maximum pour lire le pong
	pongWait = 60 * time.Second

	// Taille maximale du message
	maxMessageSize = 512 * 1024 // 512 KB
)

// Client représente une connexion WebSocket
type Client struct {
	// Hub de gestion
	hub *Hub

	// Connexion WebSocket
	conn *websocket.Conn

	// Canal pour envoyer des messages
	send chan *Message

	// ID de l'utilisateur (après authentification)
	UserID string

	// Conversations auxquelles le client participe
	conversations map[string]bool
}

// ClientMessage représente un message reçu du client
type ClientMessage struct {
	Type           string                 `json:"type"`
	Token          string                 `json:"token,omitempty"`
	ConversationID string                 `json:"conversation_id,omitempty"`
	Message        map[string]interface{} `json:"message,omitempty"`
	SenderID       string                 `json:"sender_id,omitempty"`
}

// NewClient crée un nouveau client WebSocket
func NewClient(hub *Hub, conn *websocket.Conn) *Client {
	return &Client{
		hub:           hub,
		conn:          conn,
		send:          make(chan *Message, 256),
		conversations: make(map[string]bool),
	}
}

// readPump gère la lecture des messages depuis le WebSocket
func (c *Client) readPump(jwtSecret string, validateToken func(string, string) (string, error)) {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		var msg ClientMessage
		err := c.conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Erreur WebSocket: %v", err)
			}
			break
		}

		// Gérer les différents types de messages
		switch msg.Type {
		case "authenticate":
			c.handleAuthenticate(msg.Token, jwtSecret, validateToken)

		case "join_conversation":
			if c.UserID != "" && msg.ConversationID != "" {
				c.hub.JoinConversation(c, msg.ConversationID)
				c.conversations[msg.ConversationID] = true
			}

		case "leave_conversation":
			if c.UserID != "" && msg.ConversationID != "" {
				c.hub.LeaveConversation(c, msg.ConversationID)
				delete(c.conversations, msg.ConversationID)
			}

		case "send_message":
			if c.UserID != "" && msg.ConversationID != "" {
				// Diffuser le message aux autres participants
				c.hub.NotifyConversation(msg.ConversationID, "new_message", map[string]interface{}{
					"conversation_id": msg.ConversationID,
					"message":         msg.Message,
				})
			}
		}
	}
}

// writePump gère l'écriture des messages vers le WebSocket
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// Envoyer le message JSON
			if err := c.conn.WriteJSON(message); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleAuthenticate gère l'authentification du client
func (c *Client) handleAuthenticate(token string, jwtSecret string, validateToken func(string, string) (string, error)) {
	userID, err := validateToken(token, jwtSecret)
	if err != nil {
		// Envoyer une erreur
		c.send <- &Message{
			Type: "error",
			Data: map[string]interface{}{
				"message": "Authentification échouée",
			},
		}
		return
	}

	// Authentification réussie
	c.UserID = userID
	c.hub.register <- c

	// Envoyer une confirmation
	c.send <- &Message{
		Type: "authenticated",
		Data: map[string]interface{}{
			"success": true,
			"user_id": userID,
		},
	}

	log.Printf("🔑 Client WebSocket authentifié: %s", userID)
}

// SendMessage envoie un message à ce client
func (c *Client) SendMessage(message *Message) {
	select {
	case c.send <- message:
	default:
		// Canal plein, fermer le client
		close(c.send)
	}
}

