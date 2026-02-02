package services

import (
	"testing"
)

// TestDisabledFCMService vérifie que NewDisabledFCMService fonctionne
func TestDisabledFCMService(t *testing.T) {
	svc := NewDisabledFCMService()
	if svc == nil {
		t.Fatal("NewDisabledFCMService() ne doit pas retourner nil")
	}
	// SendToAll sur un service désactivé ne doit pas paniquer
	success, failed, _ := svc.SendToAll([]string{}, "t", "b", nil)
	if success != 0 || failed != 0 {
		t.Errorf("SendToAll sur service désactivé: success=%d, failed=%d", success, failed)
	}
}
