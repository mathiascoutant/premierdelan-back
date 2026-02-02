package database

import (
	"testing"
)

func TestPing_clientNil(t *testing.T) {
	// Sauvegarder l'état actuel
	oldClient := Client
	Client = nil
	defer func() { Client = oldClient }()

	err := Ping()
	if err == nil {
		t.Error("Ping() devrait échouer quand Client est nil")
	}
	if err != nil && err.Error() != "client MongoDB non initialisé" {
		t.Errorf("Ping() erreur = %v", err)
	}
}
