package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"premier-an-backend/database"
	"premier-an-backend/middleware"
	"premier-an-backend/models"
)

// ThemeHandler gère les requêtes liées au thème global du site
type ThemeHandler struct {
	siteSettingRepo *database.SiteSettingRepository
	userCollection  *database.UserRepository
}

// NewThemeHandler crée un nouveau handler pour les thèmes
func NewThemeHandler(siteSettingRepo *database.SiteSettingRepository, userCollection *database.UserRepository) *ThemeHandler {
	return &ThemeHandler{
		siteSettingRepo: siteSettingRepo,
		userCollection:  userCollection,
	}
}

// GetGlobalTheme récupère le thème global du site (endpoint public)
func (h *ThemeHandler) GetGlobalTheme(w http.ResponseWriter, r *http.Request) {
	theme, err := h.siteSettingRepo.GetGlobalTheme(r.Context())
	if err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}

	response := models.ThemeResponse{
		Success: true,
		Theme:   theme,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// SetGlobalTheme définit le thème global du site (admin uniquement)
func (h *ThemeHandler) SetGlobalTheme(w http.ResponseWriter, r *http.Request) {
	// Récupérer les claims depuis le contexte
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		http.Error(w, "Token invalide", http.StatusUnauthorized)
		return
	}

	// Récupérer l'utilisateur par email (claims.UserID est maintenant un email)
	user, err := h.userCollection.FindByEmail(claims.UserID)
	if err != nil || user == nil {
		http.Error(w, "Utilisateur non trouvé", http.StatusNotFound)
		return
	}

	if user.Admin != 1 {
		http.Error(w, "Accès refusé. Seuls les administrateurs peuvent modifier le thème global", http.StatusForbidden)
		return
	}

	// Parser le body JSON
	var request models.ThemeRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Body JSON invalide", http.StatusBadRequest)
		return
	}

	// Validation du thème
	theme := strings.TrimSpace(request.Theme)
	if theme == "" {
		http.Error(w, "Thème requis", http.StatusBadRequest)
		return
	}

	if theme != "medieval" && theme != "classic" {
		http.Error(w, "Thème invalide. Doit être \"medieval\" ou \"classic\"", http.StatusBadRequest)
		return
	}

	// Mise à jour du thème global
	err = h.siteSettingRepo.SetGlobalTheme(r.Context(), theme, &user.ID)
	if err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}

	// Réponse JSON
	response := models.ThemeResponse{
		Success: true,
		Message: "Thème global mis à jour avec succès",
		Theme:   theme,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}
