package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"premier-an-backend/utils"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Permettre toutes les origines pour développement
		return true
	},
}

// Handler gère les connexions WebSocket
type Handler struct {
	hub       *Hub
	jwtSecret string
}

// NewHandler crée un nouveau handler WebSocket
func NewHandler(hub *Hub, jwtSecret string) *Handler {
	return &Handler{
		hub:       hub,
		jwtSecret: jwtSecret,
	}
}

// ServeWS gère les requêtes WebSocket
func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) {

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("❌ Erreur upgrade WebSocket: %v", err)
		http.Error(w, "WebSocket upgrade failed", http.StatusBadRequest)
		return
	}

	// Créer le client (pas encore authentifié)
	client := &Client{
		hub:    h.hub,
		conn:   conn,
		send:   make(chan interface{}, 256),
		UserID: "", // Sera défini après authentification
	}

	// Attendre le message d'authentification
	go func() {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("❌ Erreur lecture auth: %v", err)
			conn.Close()
			return
		}

		// Parser le message d'authentification
		var authMsg map[string]interface{}
		if err := json.Unmarshal(message, &authMsg); err != nil {
			log.Printf("❌ Erreur parsing auth: %v", err)
			conn.Close()
			return
		}

		// Vérifier le type
		if authMsg["type"] != "authenticate" {
			log.Println("❌ Premier message doit être 'authenticate'")
			_ = conn.WriteJSON(map[string]interface{}{
				"type":    "error",
				"message": "Authentification requise",
			})
			conn.Close()
			return
		}

		// Extraire et valider le token
		token, ok := authMsg["token"].(string)
		if !ok || token == "" {
			log.Println("❌ Token manquant")
			_ = conn.WriteJSON(map[string]interface{}{
				"type":    "error",
				"message": "Token requis",
			})
			conn.Close()
			return
		}

		// Valider le JWT token
		claims, err := utils.ValidateToken(token, h.jwtSecret)
		if err != nil {
			log.Printf("❌ Token invalide: %v", err)
			_ = conn.WriteJSON(map[string]interface{}{
				"type":    "error",
				"message": "Token invalide ou expiré",
			})
			conn.Close()
			return
		}

		// Authentification réussie
		client.UserID = claims.UserID

		// Envoyer la confirmation
		_ = conn.WriteJSON(map[string]interface{}{
			"type":    "authenticated",
			"user_id": client.UserID,
		})

		// Enregistrer le client dans le hub
		h.hub.register <- client

		// Démarrer les pumps
		go client.writePump()
		go client.readPump()
	}()
}
