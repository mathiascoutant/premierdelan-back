package chatws

import (
	"log"
	"net/http"
	"premier-an-backend/utils"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Autoriser toutes les origines (à sécuriser en production)
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

// ServeWS gère les connexions WebSocket
func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) {
	// Upgrader la connexion HTTP en WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Erreur lors de l'upgrade WebSocket: %v", err)
		return
	}

	// Créer un nouveau client
	client := NewClient(h.hub, conn)

	// Fonction de validation du token
	validateToken := func(token string, secret string) (string, error) {
		claims, err := utils.ValidateToken(token, secret)
		if err != nil {
			return "", err
		}
		return claims.UserID, nil
	}

	// Démarrer les pompes de lecture et d'écriture
	go client.writePump()
	go client.readPump(h.jwtSecret, validateToken)

	log.Printf("🔌 Nouvelle connexion WebSocket établie")
}

// GetHub retourne le hub WebSocket
func (h *Handler) GetHub() *Hub {
	return h.hub
}

