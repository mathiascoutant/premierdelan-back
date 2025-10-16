package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"premier-an-backend/database"
	"premier-an-backend/services"
)

// TestNotifHandler - Handler ultra simple pour tester les notifications
type TestNotifHandler struct {
	fcmTokenRepo *database.FCMTokenRepository
	fcmService   *services.FCMService
}

// NewTestNotifHandler crée le handler
func NewTestNotifHandler(fcmTokenRepo *database.FCMTokenRepository, fcmService *services.FCMService) *TestNotifHandler {
	return &TestNotifHandler{
		fcmTokenRepo: fcmTokenRepo,
		fcmService:   fcmService,
	}
}

// SendSimpleTest - Version ULTRA SIMPLE pour tester
func (h *TestNotifHandler) SendSimpleTest(w http.ResponseWriter, r *http.Request) {
	log.Println("🧪 ========== TEST NOTIFICATION ULTRA SIMPLE ==========")
	
	var req struct {
		Email string `json:"email"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("❌ Erreur décodage: %v", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	
	if req.Email == "" {
		req.Email = "mathiascoutant@icloud.com" // Default pour test
	}
	
	log.Printf("📧 Email: %s", req.Email)
	
	// Récupérer le token
	fcmTokens, err := h.fcmTokenRepo.FindByUserID(req.Email)
	if err != nil {
		log.Printf("❌ Erreur DB: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	
	if len(fcmTokens) == 0 {
		log.Printf("⚠️  Aucun token pour: %s", req.Email)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "No token found",
		})
		return
	}
	
	// Extraire le token string
	tokenString := fcmTokens[0].Token
	log.Printf("📱 Token trouvé: %s...", tokenString[:30])
	
	// Message ULTRA SIMPLE
	title := "TEST"
	message := "Si tu vois ça, ça marche !"
	
	log.Printf("📤 Envoi notification...")
	log.Printf("   Title: %s", title)
	log.Printf("   Message: %s", message)
	log.Printf("   Token: %s...", tokenString[:30])
	
	// Envoyer
	err = h.fcmService.SendToToken(tokenString, title, message, nil)
	if err != nil {
		log.Printf("❌ ERREUR ENVOI: %v", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}
	
	log.Printf("✅ NOTIFICATION ENVOYÉE AVEC SUCCÈS !")
	log.Println("🧪 ================================================")
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Notification sent",
		"token":   tokenString[:30] + "...",
	})
}

