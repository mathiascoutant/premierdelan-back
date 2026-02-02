package handlers

import (
	"net/http"

	"premier-an-backend/constants"
	"premier-an-backend/utils"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RequireMethod vérifie que la méthode HTTP est correcte. Retourne false et écrit l'erreur si non.
func RequireMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method != method {
		utils.RespondError(w, http.StatusMethodNotAllowed, constants.ErrMethodNotAllowed)
		return false
	}
	return true
}

// ParseEventID extrait et valide event_id depuis les vars de l'URL.
func ParseEventID(w http.ResponseWriter, r *http.Request) (primitive.ObjectID, bool) {
	vars := mux.Vars(r)
	id, err := primitive.ObjectIDFromHex(vars["event_id"])
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, constants.ErrInvalidEventID)
		return primitive.NilObjectID, false
	}
	return id, true
}

// ParseObjectIDVar extrait et valide un ObjectID depuis les vars (clé configurable, msg d'erreur configurable).
func ParseObjectIDVar(w http.ResponseWriter, vars map[string]string, key, errMsg string) (primitive.ObjectID, bool) {
	id, err := primitive.ObjectIDFromHex(vars[key])
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, errMsg)
		return primitive.NilObjectID, false
	}
	return id, true
}
