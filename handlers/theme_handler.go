package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"premier-an-backend/constants"
	"premier-an-backend/database"
	"premier-an-backend/middleware"
	"premier-an-backend/models"
	"premier-an-backend/utils"
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
		utils.RespondError(w, http.StatusInternalServerError, constants.ErrServerError)
		return
	}

	utils.RespondJSON(w, http.StatusOK, models.ThemeResponse{
		Success: true,
		Theme:   theme,
	})
}

// SetGlobalTheme définit le thème global du site (admin uniquement)
func (h *ThemeHandler) SetGlobalTheme(w http.ResponseWriter, r *http.Request) {
	// Récupérer les claims depuis le contexte
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		utils.RespondError(w, http.StatusUnauthorized, constants.ErrInvalidToken)
		return
	}

	// Récupérer l'utilisateur par email (claims.UserID est maintenant un email)
	user, err := h.userCollection.FindByEmail(claims.UserID)
	if err != nil || user == nil {
		utils.RespondError(w, http.StatusNotFound, constants.ErrUserNotFound)
		return
	}

	if user.Admin != 1 {
		utils.RespondError(w, http.StatusForbidden, constants.ErrThemeAdminOnly)
		return
	}

	// Parser le body JSON
	var request models.ThemeRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		utils.RespondError(w, http.StatusBadRequest, constants.ErrInvalidJSONBody)
		return
	}

	// Validation du thème
	theme := strings.TrimSpace(request.Theme)
	if theme == "" {
		utils.RespondError(w, http.StatusBadRequest, constants.ErrThemeRequired)
		return
	}

	if theme != "medieval" && theme != "classic" {
		utils.RespondError(w, http.StatusBadRequest, constants.ErrThemeInvalid)
		return
	}

	// Mise à jour du thème global
	err = h.siteSettingRepo.SetGlobalTheme(r.Context(), theme, &user.ID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, constants.ErrServerError)
		return
	}

	utils.RespondJSON(w, http.StatusOK, models.ThemeResponse{
		Success: true,
		Message: "Thème global mis à jour avec succès",
		Theme:   theme,
	})
}
