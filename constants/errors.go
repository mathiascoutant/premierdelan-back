package constants

// Messages d'erreur HTTP courants
const (
	ErrMethodNotAllowed   = "Méthode non autorisée"
	ErrServerError        = "Erreur serveur"
	ErrInvalidData        = "Données invalides"
	ErrNotAuthenticated   = "Non authentifié"
	ErrInvalidToken       = "Token invalide"
	ErrInvalidEventID     = "ID événement invalide"
	ErrInvalidGroupID     = "ID de groupe invalide"
	ErrGroupNotFound      = "Groupe non trouvé"
	ErrNotGroupMember     = "Vous n'êtes pas membre de ce groupe"
	ErrUserNotFound       = "Utilisateur introuvable"
	ErrAdminOnly          = "Accès refusé. Admin uniquement"
	ErrConvIDRequired     = "ID de conversation requis"
	ErrInvalidConvID      = "ID de conversation invalide"
	ErrConvNotFound       = "Conversation non trouvée"
	ErrConvAccessDenied   = "Accès refusé à cette conversation"
	ErrInvalidJSONBody    = "Body JSON invalide"
	ErrIDConversion       = "Erreur conversion ID: %v"
	ErrDecodeInscriptions = "erreur lors du décodage des inscriptions: %w"
)

// En-têtes HTTP
const (
	HeaderContentType     = "Content-Type"
	HeaderApplicationJSON = "application/json"
)
