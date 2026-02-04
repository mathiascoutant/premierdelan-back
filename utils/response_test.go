package utils

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRespondError(t *testing.T) {
	rr := httptest.NewRecorder()
	RespondError(rr, http.StatusBadRequest, "Champ invalide")

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Code = %v, attendu %v", rr.Code, http.StatusBadRequest)
	}
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type = %v", ct)
	}
	if !strings.Contains(rr.Body.String(), "Bad Request") {
		t.Errorf("Body devrait contenir 'Bad Request', got %s", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "Champ invalide") {
		t.Errorf("Body devrait contenir 'Champ invalide', got %s", rr.Body.String())
	}
}

func TestRespondSuccess(t *testing.T) {
	rr := httptest.NewRecorder()
	RespondSuccess(rr, "Succès", map[string]string{"id": "123"})

	if rr.Code != http.StatusOK {
		t.Errorf("Code = %v, attendu 200", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "Succès") {
		t.Errorf("Body devrait contenir 'Succès', got %s", body)
	}
	if !strings.Contains(body, "true") {
		t.Errorf("Body devrait contenir success true, got %s", body)
	}
}
