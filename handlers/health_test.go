package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthHandlerHealth(t *testing.T) {
	handler := NewHealthHandler("test")

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rr := httptest.NewRecorder()

	handler.Health(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Health() status = %v, want %v", rr.Code, http.StatusOK)
	}

	ct := rr.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Health() Content-Type = %v, want application/json", ct)
	}

	body := rr.Body.String()
	expectedKeys := []string{"status", "env", "uptime", "go_version"}
	for _, key := range expectedKeys {
		if !strings.Contains(body, key) {
			t.Errorf("Health() body should contain %q, got %s", key, body)
		}
	}
}
