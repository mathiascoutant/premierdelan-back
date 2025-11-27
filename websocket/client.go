package websocket

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Temps maximum pour l'écriture d'un message
	writeWait = 10 * time.Second

	// Temps maximum pour la lecture d'un pong
	pongWait = 60 * time.Second

	// Intervalle des pings
	pingPeriod = (pongWait * 9) / 10

	// Taille maximale des messages
	maxMessageSize = 4096
)

// Client représente une connexion WebSocket client
type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan interface{}
	UserID string
}

// readPump pompe les messages de la connexion WebSocket vers le hub
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("❌ Erreur WebSocket: %v", err)
			}
			break
		}

		// Parser le message JSON
		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("❌ Erreur parsing message: %v", err)
			continue
		}

		msgType, ok := msg["type"].(string)
		if !ok {
			continue
		}

		// Traiter les messages
		switch msgType {
		case "join_conversation":
			if convID, ok := msg["conversation_id"].(string); ok {
				c.hub.JoinConversation(c.UserID, convID)
			}

		case "leave_conversation":
			if convID, ok := msg["conversation_id"].(string); ok {
				c.hub.LeaveConversation(c.UserID, convID)
			}

		case "typing":
			// ⌨️ Gérer le typing indicator
			if convID, ok := msg["conversation_id"].(string); ok {
				isTyping, _ := msg["is_typing"].(bool)
				c.hub.HandleTyping(c.UserID, convID, isTyping)
			} else if groupID, ok := msg["group_id"].(string); ok {
				// ⌨️ Gérer le typing indicator pour les groupes
				isTyping, _ := msg["is_typing"].(bool)
				c.hub.HandleGroupTyping(c.UserID, groupID, isTyping)
			} else {
				log.Printf("⚠️  Événement typing sans conversation_id ni group_id")
			}

		case "join_group":
			// 👥 Rejoindre un groupe
			if groupID, ok := msg["group_id"].(string); ok {
				c.hub.JoinGroup(c.UserID, groupID)
				// Confirmer au client
				c.send <- map[string]interface{}{
					"type":     "joined_group",
					"group_id": groupID,
				}
			} else {
				log.Printf("⚠️  Événement join_group sans group_id")
			}

		case "leave_group":
			// 👋 Quitter un groupe
			if groupID, ok := msg["group_id"].(string); ok {
				c.hub.LeaveGroup(c.UserID, groupID)
			}

		case "group_typing":
			// ⌨️ Gérer le typing indicator dans un groupe
			if groupID, ok := msg["group_id"].(string); ok {
				isTyping, _ := msg["is_typing"].(bool)
				c.hub.HandleGroupTyping(c.UserID, groupID, isTyping)
			}

		case "user_presence":
			// 👤 Gérer la présence utilisateur (mise à jour automatique)
			// Le frontend envoie cet événement quand l'utilisateur navigue sur le site
			isOnline, ok := msg["is_online"].(bool)
			if !ok {
				log.Printf("⚠️  Événement user_presence sans is_online")
				continue
			}

			// Extraire last_seen si fourni (format ISO 8601 string)
			var lastSeen *time.Time
			if lastSeenStr, ok := msg["last_seen"].(string); ok && lastSeenStr != "" && lastSeenStr != "null" {
				parsed, err := time.Parse(time.RFC3339, lastSeenStr)
				if err == nil {
					lastSeen = &parsed
				}
			}

			// Mettre à jour la présence via le gestionnaire
			if c.hub.presenceManager != nil {
				c.hub.presenceManager.UpdateUserPresence(c.UserID, isOnline)
				
				// Si l'utilisateur est actif, mettre à jour last_activity en DB
				if isOnline {
					if err := c.hub.updateUserActivityInDB(c.UserID); err != nil {
						log.Printf("⚠️  Erreur mise à jour activité: %v", err)
					}
				} else if lastSeen != nil {
					// Si hors ligne avec last_seen, mettre à jour en DB
					if err := c.hub.updateUserLastSeenInDB(c.UserID, lastSeen); err != nil {
						log.Printf("⚠️  Erreur mise à jour last_seen: %v", err)
					}
				}
				
				log.Printf("✅ Présence mise à jour: %s (online=%v)", c.UserID, isOnline)
			} else {
				log.Printf("⚠️  presenceManager est nil")
			}

		default:
			log.Printf("⚠️  Type de message inconnu: %s", msgType)
		}
	}
}

// writePump pompe les messages du hub vers la connexion WebSocket
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
				// Le hub a fermé le canal
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// Encoder en JSON et envoyer
			if err := c.conn.WriteJSON(message); err != nil {
				log.Printf("❌ Erreur écriture WebSocket: %v", err)
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
