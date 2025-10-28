package websocket

import (
	"log"
	"sync"
	"time"
)

// PresenceManager gère la présence des utilisateurs avec timeouts automatiques
type PresenceManager struct {
	// Timeouts actifs par user_id (email)
	userTimeouts map[string]*time.Timer

	// Mutex pour sécuriser les accès concurrents
	mu sync.RWMutex

	// Callback pour mettre à jour la présence en base
	updatePresenceCallback func(userID string, isOnline bool) error

	// Callback pour diffuser les mises à jour de présence
	broadcastPresenceCallback func(userID string, isOnline bool, lastSeen *time.Time)
}

// NewPresenceManager crée un nouveau gestionnaire de présence
func NewPresenceManager(
	updatePresenceCallback func(userID string, isOnline bool) error,
	broadcastPresenceCallback func(userID string, isOnline bool, lastSeen *time.Time),
) *PresenceManager {
	pm := &PresenceManager{
		userTimeouts:              make(map[string]*time.Timer),
		updatePresenceCallback:    updatePresenceCallback,
		broadcastPresenceCallback: broadcastPresenceCallback,
	}

	// Démarrer le nettoyage périodique
	go pm.startCleanupRoutine()

	return pm
}

// UpdateUserPresence met à jour la présence d'un utilisateur
func (pm *PresenceManager) UpdateUserPresence(userID string, isOnline bool) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	log.Printf("👤 Mise à jour présence: %s -> %v", userID, isOnline)

	if isOnline {
		// Annuler le timeout précédent s'il existe
		if timer, exists := pm.userTimeouts[userID]; exists {
			timer.Stop()
			log.Printf("⏰ Timeout précédent annulé pour %s", userID)
		}

		// Programmer un nouveau timeout de 4 minutes
		timer := time.AfterFunc(4*time.Minute, func() {
			pm.handleUserTimeout(userID)
		})
		pm.userTimeouts[userID] = timer

		log.Printf("⏰ Timeout programmé pour %s (4 minutes)", userID)

		// Mettre à jour la base de données
		if pm.updatePresenceCallback != nil {
			if err := pm.updatePresenceCallback(userID, true); err != nil {
				log.Printf("❌ Erreur mise à jour présence en ligne: %v", err)
			}
		}

		// Diffuser la mise à jour
		if pm.broadcastPresenceCallback != nil {
			pm.broadcastPresenceCallback(userID, true, nil)
		}

	} else {
		// Utilisateur se déconnecte manuellement
		if timer, exists := pm.userTimeouts[userID]; exists {
			timer.Stop()
			delete(pm.userTimeouts, userID)
			log.Printf("⏰ Timeout supprimé pour %s (déconnexion manuelle)", userID)
		}

		// Mettre à jour la base de données
		if pm.updatePresenceCallback != nil {
			if err := pm.updatePresenceCallback(userID, false); err != nil {
				log.Printf("❌ Erreur mise à jour présence hors ligne: %v", err)
			}
		}

		// Diffuser la mise à jour avec last_seen
		now := time.Now()
		if pm.broadcastPresenceCallback != nil {
			pm.broadcastPresenceCallback(userID, false, &now)
		}
	}
}

// handleUserTimeout gère le timeout d'inactivité d'un utilisateur
func (pm *PresenceManager) handleUserTimeout(userID string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	log.Printf("⏰ Timeout d'inactivité pour %s (4 minutes)", userID)

	// Supprimer le timeout de la map
	delete(pm.userTimeouts, userID)

	// Mettre à jour la base de données
	if pm.updatePresenceCallback != nil {
		if err := pm.updatePresenceCallback(userID, false); err != nil {
			log.Printf("❌ Erreur mise à jour présence timeout: %v", err)
		}
	}

	// Diffuser la mise à jour avec last_seen
	now := time.Now()
	if pm.broadcastPresenceCallback != nil {
		pm.broadcastPresenceCallback(userID, false, &now)
	}
}

// RemoveUser supprime un utilisateur du gestionnaire de présence
func (pm *PresenceManager) RemoveUser(userID string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if timer, exists := pm.userTimeouts[userID]; exists {
		timer.Stop()
		delete(pm.userTimeouts, userID)
		log.Printf("🗑️  Utilisateur %s supprimé du gestionnaire de présence", userID)
	}
}

// GetActiveUsers retourne la liste des utilisateurs actifs
func (pm *PresenceManager) GetActiveUsers() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	users := make([]string, 0, len(pm.userTimeouts))
	for userID := range pm.userTimeouts {
		users = append(users, userID)
	}
	return users
}

// startCleanupRoutine démarre la routine de nettoyage périodique
func (pm *PresenceManager) startCleanupRoutine() {
	ticker := time.NewTicker(1 * time.Hour) // Nettoyage toutes les heures
	defer ticker.Stop()

	for range ticker.C {
		pm.cleanupOrphanedTimeouts()
	}
}

// cleanupOrphanedTimeouts nettoie les timeouts orphelins
func (pm *PresenceManager) cleanupOrphanedTimeouts() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	cleanedCount := 0

	for userID, timer := range pm.userTimeouts {
		// Vérifier si le timer est encore valide
		// Note: Cette vérification est basique, on pourrait l'améliorer
		// en stockant le timestamp de création du timer
		if timer == nil {
			delete(pm.userTimeouts, userID)
			cleanedCount++
		}
	}

	if cleanedCount > 0 {
		log.Printf("🧹 Nettoyage terminé: %d timeouts orphelins supprimés", cleanedCount)
	}
}

// Shutdown arrête le gestionnaire de présence et marque tous les utilisateurs comme hors ligne
func (pm *PresenceManager) Shutdown() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	log.Printf("🔄 Arrêt du gestionnaire de présence - Marquage des utilisateurs hors ligne...")

	// Arrêter tous les timeouts
	for userID, timer := range pm.userTimeouts {
		timer.Stop()
		log.Printf("⏰ Timeout arrêté pour %s", userID)
	}

	// Marquer tous les utilisateurs actifs comme hors ligne
	now := time.Now()
	for userID := range pm.userTimeouts {
		if pm.updatePresenceCallback != nil {
			if err := pm.updatePresenceCallback(userID, false); err != nil {
				log.Printf("❌ Erreur mise à jour présence shutdown: %v", err)
			}
		}

		if pm.broadcastPresenceCallback != nil {
			pm.broadcastPresenceCallback(userID, false, &now)
		}
	}

	// Vider la map
	pm.userTimeouts = make(map[string]*time.Timer)

	log.Printf("✅ Gestionnaire de présence arrêté")
}
